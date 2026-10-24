package main

import (
	"log"

	"github.com/cshum/vipsgen/v2/examples/testdata"
	"github.com/cshum/vipsgen/v2/vips"
)

func main() {
	// N: -1 loads every frame, so the multi-page crop below has pages to work on.
	image, err := vips.NewImageFromFile(testdata.Path("dancing-banana.gif"), &vips.LoadOptions{N: -1})
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer image.Close()
	log.Printf("Loaded image: %s %dx%d\n", image.Format(), image.Width(), image.Height())
	// crop with animation support
	if err = image.ExtractAreaMultiPage(30, 40, 50, 70); err != nil {
		log.Fatalf("Failed to crop image: %v", err)
	}
	// Flatten image with cyan background
	if err = image.Flatten(&vips.FlattenOptions{Background: []float64{0, 255, 255}}); err != nil {
		log.Fatalf("Failed to flatten image: %v", err)
	}
	log.Printf("Processed image: %dx%d\n", image.Width(), image.Height())
	err = image.Gifsave("dancing-banana-cropped.gif", nil)
	if err != nil {
		log.Fatalf("Failed to save image: %v", err)
	}
	log.Println("Successfully saved processed images")
}
