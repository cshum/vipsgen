// Applies an operation to every frame of an animated image.
//
// vipsgen binds libvips, so an operation without a MultiPage helper runs on the
// stacked image the frames live in. Pixelating is such an operation: on the stack the
// block grid runs across frame boundaries and the page structure is lost, so the frames
// have to be taken apart first.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/cshum/vipsgen/v2/vips"
)

const (
	url     = "https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif"
	outPath = "dancing-banana-pixelated.gif"
	block   = 10
)

// pixelate averages blocks of pixels and scales each block back up.
func pixelate(img *vips.Image, block int) error {
	if block <= 1 {
		return nil
	}
	if err := img.Shrink(float64(block), float64(block), nil); err != nil {
		return err
	}
	return img.Zoom(block, block)
}

// transformFrames applies op to each page of a stacked image and returns the restacked
// result, with a single-frame image passed through as a copy.
func transformFrames(img *vips.Image, op func(*vips.Image) error) (*vips.Image, error) {
	pageHeight := img.PageHeight()
	if pageHeight <= 0 || img.Height() <= pageHeight {
		frame, err := img.Copy(nil)
		if err != nil {
			return nil, err
		}
		if err := op(frame); err != nil {
			frame.Close()
			return nil, err
		}
		return frame, nil
	}

	var frames []*vips.Image
	defer func() {
		for _, f := range frames {
			f.Close()
		}
	}()
	for top := 0; top+pageHeight <= img.Height(); top += pageHeight {
		frame, err := img.Copy(nil)
		if err != nil {
			return nil, err
		}
		// ExtractArea works in place, so each frame is cut from its own copy.
		if err := frame.ExtractArea(0, top, img.Width(), pageHeight); err != nil {
			frame.Close()
			return nil, err
		}
		if err := op(frame); err != nil {
			frame.Close()
			return nil, err
		}
		frames = append(frames, frame)
	}

	out, err := vips.NewArrayjoin(frames, &vips.ArrayjoinOptions{Across: 1})
	if err != nil {
		return nil, err
	}
	// The frames may have changed size, so the page height comes from the result.
	if err := out.SetPageHeight(out.Height() / len(frames)); err != nil {
		out.Close()
		return nil, err
	}
	// Restacking builds a new image, so the frame timings have to be carried over.
	if delays, err := img.PageDelay(); err == nil {
		if err := out.SetArrayInt("delay", delays); err != nil {
			out.Close()
			return nil, err
		}
	}
	return out, nil
}

func main() {
	data, err := fetch(url)
	if err != nil {
		log.Fatalf("Failed to fetch image: %v", err)
	}

	// N: -1 loads every frame, which is what a frame-by-frame transform needs.
	img, err := vips.NewImageFromBuffer(data, &vips.LoadOptions{N: -1})
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer img.Close()
	log.Printf("Loaded %d frames of %d rows", img.Height()/img.PageHeight(), img.PageHeight())

	// The same call on the stacked image would pixelate across the frame boundary and
	// leave one page where there were several.
	result, err := transformFrames(img, func(frame *vips.Image) error {
		return pixelate(frame, block)
	})
	if err != nil {
		log.Fatalf("Failed to transform frames: %v", err)
	}
	defer result.Close()

	if err := result.Gifsave(outPath, nil); err != nil {
		log.Fatalf("Failed to save image: %v", err)
	}
	log.Printf("Saved %s: %d frames of %d rows", outPath,
		result.Height()/result.PageHeight(), result.PageHeight())
}

func fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
