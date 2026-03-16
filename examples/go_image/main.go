package main

import (
	"image"
	"image/draw"
	_ "image/png"
	"log"
	"net/http"

	"github.com/cshum/vipsgen/vips"
)

// goImageToVips converts any image.Image to a *vips.Image via NewImageFromMemory.
// It normalizes to NRGBA (4-band, 8-bit) before handing off to libvips.
// The sRGB interpretation is explicitly restored after NewImageFromMemory, which
// always assigns VIPS_INTERPRETATION_MULTIBAND
func goImageToVips(src image.Image) (*vips.Image, error) {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Fast path: already *image.NRGBA with zero origin and contiguous stride
	if n, ok := src.(*image.NRGBA); ok && bounds.Min.X == 0 && bounds.Min.Y == 0 && n.Stride == w*4 {
		return newVipsFromRawRGBA(n.Pix, w, h)
	}

	// Slow path: normalize any image.Image to NRGBA via draw.Draw
	nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(nrgba, nrgba.Bounds(), src, bounds.Min, draw.Src)
	return newVipsFromRawRGBA(nrgba.Pix, w, h)
}

// newVipsFromRawRGBA wraps NewImageFromMemory and restores the sRGB interpretation.
// NewImageFromMemory assigns VIPS_INTERPRETATION_MULTIBAND by default; this
// metadata-only Copy corrects that without converting any pixel data.
func newVipsFromRawRGBA(pix []byte, w, h int) (*vips.Image, error) {
	img, err := vips.NewImageFromMemory(pix, w, h, 4)
	if err != nil {
		return nil, err
	}
	copied, err := img.Copy(&vips.CopyOptions{Interpretation: vips.InterpretationSrgb})
	if err != nil {
		img.Close()
		return nil, err
	}
	img.Close()
	return copied, nil
}

// vipsToGoImage converts a *vips.Image back to *image.NRGBA via WriteToMemory.
// The caller is responsible for ensuring the image is in a compatible format
// (e.g. sRGB, uchar bands) before calling this.
func vipsToGoImage(img *vips.Image) (*image.NRGBA, error) {
	buf, err := img.WriteToMemory()
	if err != nil {
		return nil, err
	}
	nrgba := &image.NRGBA{
		Pix:    buf,
		Stride: img.Width() * img.Bands(),
		Rect:   image.Rect(0, 0, img.Width(), img.Height()),
	}
	return nrgba, nil
}

func main() {
	// Fetch a PNG from the network as a Go image.Image
	resp, err := http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png")
	if err != nil {
		log.Fatalf("Failed to fetch image: %v", err)
	}
	defer resp.Body.Close()

	src, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Fatalf("Failed to decode image: %v", err)
	}

	// Import: Go image.Image → vips.Image
	vipsImg, err := goImageToVips(src)
	if err != nil {
		log.Fatalf("Failed to import image: %v", err)
	}
	defer vipsImg.Close()
	log.Printf("Imported image: %dx%d, bands=%d", vipsImg.Width(), vipsImg.Height(), vipsImg.Bands())

	// Process with libvips
	if err = vipsImg.Resize(0.5, nil); err != nil {
		log.Fatalf("Failed to resize: %v", err)
	}
	if err = vipsImg.Flatten(&vips.FlattenOptions{
		Background: []float64{255, 255, 0}, // yellow background
	}); err != nil {
		log.Fatalf("Failed to flatten: %v", err)
	}
	log.Printf("Processed image: %dx%d", vipsImg.Width(), vipsImg.Height())

	// Export: vips.Image → Go image.NRGBA via WriteToMemory
	goImg, err := vipsToGoImage(vipsImg)
	if err != nil {
		log.Fatalf("Failed to export image: %v", err)
	}
	log.Printf("Exported to Go image: %dx%d, pix len=%d",
		goImg.Bounds().Dx(), goImg.Bounds().Dy(), len(goImg.Pix))

	// Also save to disk via libvips
	if err = vipsImg.Jpegsave("gopher.jpg", nil); err != nil {
		log.Fatalf("Failed to save: %v", err)
	}
	log.Println("Successfully saved gopher.jpg")
}
