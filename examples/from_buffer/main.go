package main

import (
	"github.com/cshum/vipsgen/v2/examples/testdata"
	"github.com/cshum/vipsgen/v2/vips"
	_ "image/png"
	"log"
	"os"
)

func main() {
	buf, err := os.ReadFile(testdata.Path("gopher.png"))
	if err != nil {
		log.Fatalf("Failed to read image: %v", err)
	}

	// Create vips.Image from Go image
	image, err := vips.NewImageFromBuffer(buf, nil)
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}
	defer image.Close()
	log.Printf("Loaded image: %s %dx%d\n", image.Format(), image.Width(), image.Height())
	err = image.Resize(0.5, nil)
	if err != nil {
		log.Fatalf("Failed to resize image: %v", err)
	}
	err = image.Flatten(&vips.FlattenOptions{
		Background: []float64{255, 255, 0}, // yellow background
	})
	if err != nil {
		log.Fatalf("Failed to flatten image: %v", err)
	}
	log.Printf("Processed image: %dx%d\n", image.Width(), image.Height())
	err = image.Jpegsave("gopher.jpg", nil)
	if err != nil {
		log.Fatalf("Failed to save image: %v", err)
	}
	log.Println("Successfully saved processed images")
}
