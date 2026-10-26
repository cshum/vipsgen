// Applies an operation to every frame of an animated image.
//
// vipsgen binds libvips, so an operation without a MultiPage helper runs on the
// stacked image the frames live in. Pixelating is such an operation: on the stack the
// block grid runs across frame boundaries and the page structure is lost, so the frames
// have to be taken apart first.
package main

import (
	"log"

	"github.com/cshum/vipsgen/v2/examples/testdata"
	"github.com/cshum/vipsgen/v2/vips"
)

const (
	outPath = "dancing-banana-pixelated.gif"
	block   = 10
)

func main() {
	// N: -1 loads every frame, which is what a frame-by-frame transform needs.
	img, err := vips.NewGifload(testdata.Path("dancing-banana.gif"), &vips.GifloadOptions{N: -1})
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer img.Close()

	pageHeight := img.PageHeight()
	pages := img.Height() / pageHeight
	log.Printf("Loaded %d frames of %d rows", pages, pageHeight)

	// The frames are cut from a copy: the image itself is rebuilt from them below.
	src, err := img.Copy(nil)
	if err != nil {
		log.Fatalf("Failed to copy image: %v", err)
	}
	defer src.Close()

	// The first frame becomes the image, and each remaining frame is joined onto it.
	if err := img.ExtractArea(0, 0, src.Width(), pageHeight); err != nil {
		log.Fatalf("Failed to extract frame 0: %v", err)
	}
	if err := img.Shrink(block, block, nil); err != nil {
		log.Fatalf("Failed to shrink frame 0: %v", err)
	}
	if err := img.Zoom(block, block); err != nil {
		log.Fatalf("Failed to zoom frame 0: %v", err)
	}

	for page := 1; page < pages; page++ {
		frame, err := src.Copy(nil)
		if err != nil {
			log.Fatalf("Failed to copy frame %d: %v", page, err)
		}
		if err := frame.ExtractArea(0, page*pageHeight, src.Width(), pageHeight); err != nil {
			frame.Close()
			log.Fatalf("Failed to extract frame %d: %v", page, err)
		}
		if err := frame.Shrink(block, block, nil); err != nil {
			frame.Close()
			log.Fatalf("Failed to shrink frame %d: %v", page, err)
		}
		if err := frame.Zoom(block, block); err != nil {
			frame.Close()
			log.Fatalf("Failed to zoom frame %d: %v", page, err)
		}
		// Join appends the frame below the ones already in the image, so the order and
		// the page delays the image already carries are kept.
		if err := img.Join(frame, vips.DirectionVertical, nil); err != nil {
			frame.Close()
			log.Fatalf("Failed to join frame %d: %v", page, err)
		}
		frame.Close()
	}
	if err := img.SetPageHeight(img.Height() / pages); err != nil {
		log.Fatalf("Failed to set page height: %v", err)
	}
	if err := img.SetPages(pages); err != nil {
		log.Fatalf("Failed to set pages: %v", err)
	}

	if err := img.Gifsave(outPath, nil); err != nil {
		log.Fatalf("Failed to save image: %v", err)
	}
	log.Printf("Saved %s: %d frames of %d rows", outPath, img.Height()/img.PageHeight(), img.PageHeight())
}
