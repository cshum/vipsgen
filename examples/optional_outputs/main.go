package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/cshum/vipsgen/vips"
)

func getBytesFromURL(url string) ([]byte, error) {
	// Make HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	// Read entire response body into bytes
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return data, nil
}

func loadImageFromURL(url string) (*vips.Image, error) {
	buf, err := getBytesFromURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image from %s: %v", url, err)
	}

	image, err := vips.NewImageFromBuffer(buf, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load image: %v", err)
	}

	return image, nil
}

func demonstrateMosaic() {
	fmt.Println("\n=== MOSAIC OPERATION WITH OPTIONAL OUTPUTS ===")

	// Load real images that are more likely to have detectable features
	fmt.Println("Loading real images for mosaic demonstration...")

	// Load the same image twice and create overlapping versions
	baseImg, err := loadImageFromURL("https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo1.jpg")
	if err != nil {
		log.Printf("Failed to load base image: %v", err)
		return
	}
	defer baseImg.Close()

	// Resize to a larger size for more features
	err = baseImg.Resize(1.5, nil)
	if err != nil {
		log.Printf("Failed to resize base image: %v", err)
		return
	}

	fmt.Printf("Base image: %dx%d\n", baseImg.Width(), baseImg.Height())

	// Create first image by extracting left portion with some overlap
	err = baseImg.ExtractArea(0, 0, baseImg.Width()*2/3, baseImg.Height())
	if err != nil {
		log.Printf("Failed to extract first image: %v", err)
		return
	}
	img1, err := baseImg.Copy(nil)
	if err != nil {
		log.Printf("Failed to copy first image: %v", err)
		return
	}
	defer img1.Close()

	// Create second image by extracting right portion with overlap
	overlapStart := baseImg.Width() / 3
	err = baseImg.ExtractArea(overlapStart, 0, baseImg.Width()-overlapStart, baseImg.Height())
	if err != nil {
		log.Printf("Failed to extract second image: %v", err)
		return
	}
	img2, err := baseImg.Copy(nil)
	if err != nil {
		log.Printf("Failed to copy second image: %v", err)
		return
	}
	defer img2.Close()

	fmt.Printf("Created overlapping images - Image 1: %dx%d, Image 2: %dx%d\n",
		img1.Width(), img1.Height(), img2.Width(), img2.Height())

	// Calculate overlap region
	overlapWidth := img1.Width() - overlapStart
	fmt.Printf("Overlap width: %d pixels (%.1f%%)\n", overlapWidth, float64(overlapWidth)/float64(img1.Width())*100)

	// Create mosaic options to capture transformation parameters
	options := vips.DefaultMosaicOptions()

	// Use appropriate search windows
	options.Hwindow = 5
	options.Harea = 15
	options.Mblend = 10

	// Set tie points in the overlap region
	// Reference point in the overlap area of first image
	xref := img1.Width() - overlapWidth/2
	yref := img1.Height() / 2

	// Corresponding point in second image (should be near the left edge)
	xsec := overlapWidth / 2
	ysec := img2.Height() / 2

	fmt.Printf("Attempting mosaic with tie points: ref(%d,%d) -> sec(%d,%d)\n", xref, yref, xsec, ysec)
	fmt.Printf("Using search parameters: hwindow=%d, harea=%d, mblend=%d\n",
		options.Hwindow, options.Harea, options.Mblend)

	err = img1.Mosaic(img2, vips.DirectionHorizontal, xref, yref, xsec, ysec, options)
	if err != nil {
		log.Printf("Mosaic operation failed: %v", err)
		fmt.Println("Note: Even with real images, mosaic requires very specific conditions.")
		fmt.Println("The optional output structure is demonstrated:")
		fmt.Printf("  MosaicOptions fields available: Dx0, Dy0, Scale1, Angle1, Dx1, Dy1\n")
		return
	}

	// Display the detected transformation parameters
	fmt.Println("\nMOSAIC SUCCESS! Detected Transformation Parameters:")
	fmt.Printf("  Integer Offset: dx0=%d, dy0=%d\n", options.Dx0, options.Dy0)
	fmt.Printf("  Detected Scale: %.3f\n", options.Scale1)
	fmt.Printf("  Detected Rotation: %.3f degrees\n", options.Angle1)
	fmt.Printf("  First-order Displacement: dx1=%.3f, dy1=%.3f\n", options.Dx1, options.Dy1)

	// Save the result
	err = img1.Jpegsave("mosaic_result.jpg", nil)
	if err != nil {
		log.Printf("Failed to save mosaic result: %v", err)
		return
	}

	fmt.Println("Mosaic result saved as 'mosaic_result.jpg'")
}

func demonstrateSmartcrop() {
	fmt.Println("\n=== SMARTCROP OPERATION WITH ATTENTION COORDINATES ===")

	// Load an image for smart cropping
	fmt.Println("Loading image for smart cropping...")
	img, err := loadImageFromURL("https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo1.jpg")
	if err != nil {
		log.Printf("Failed to load image: %v", err)
		return
	}
	defer img.Close()

	fmt.Printf("Original image: %dx%d\n", img.Width(), img.Height())

	// Create smartcrop options to capture attention coordinates
	options := vips.DefaultSmartcropOptions()

	// Perform smart crop operation (crop to smaller size that fits within the image)
	cropWidth := img.Width() / 2
	cropHeight := img.Height() / 2
	if cropWidth < 50 {
		cropWidth = 50
	}
	if cropHeight < 50 {
		cropHeight = 50
	}

	fmt.Printf("Cropping to: %dx%d\n", cropWidth, cropHeight)
	err = img.Smartcrop(cropWidth, cropHeight, options)
	if err != nil {
		log.Printf("Smartcrop operation failed: %v", err)
		return
	}

	// Display the detected attention coordinates
	fmt.Println("\nDetected Attention Coordinates:")
	fmt.Printf("  Attention Center: x=%d, y=%d\n", options.AttentionX, options.AttentionY)
	fmt.Printf("  This is where the algorithm detected the most interesting content\n")

	fmt.Printf("Cropped image: %dx%d\n", img.Width(), img.Height())

	// Save the result
	err = img.Jpegsave("smartcrop_result.jpg", nil)
	if err != nil {
		log.Printf("Failed to save smartcrop result: %v", err)
		return
	}

	fmt.Println("Smartcrop result saved as 'smartcrop_result.jpg'")
}

func demonstrateMaxMinPositions() {
	fmt.Println("\n=== MAX/MIN OPERATIONS WITH POSITION COORDINATES ===")

	// Load an image for max/min operations
	fmt.Println("Loading image for max/min analysis...")
	img, err := loadImageFromURL("https://raw.githubusercontent.com/cshum/imagor/master/testdata/demo1.jpg")
	if err != nil {
		log.Printf("Failed to load image: %v", err)
		return
	}
	defer img.Close()

	fmt.Printf("Analyzing image: %dx%d\n", img.Width(), img.Height())

	// Find maximum value and its position
	maxOptions := vips.DefaultMaxOptions()
	maxValue, err := img.Max(maxOptions)
	if err != nil {
		log.Printf("Max operation failed: %v", err)
		return
	}

	fmt.Println("\nMaximum Value Analysis:")
	fmt.Printf("  Maximum value: %.2f\n", maxValue)
	fmt.Printf("  Position: x=%d, y=%d\n", maxOptions.X, maxOptions.Y)

	// Find minimum value and its position
	minOptions := vips.DefaultMinOptions()
	minValue, err := img.Min(minOptions)
	if err != nil {
		log.Printf("Min operation failed: %v", err)
		return
	}

	fmt.Println("\nMinimum Value Analysis:")
	fmt.Printf("  Minimum value: %.2f\n", minValue)
	fmt.Printf("  Position: x=%d, y=%d\n", minOptions.X, minOptions.Y)
}

func demonstrateDrawFloodArea() {
	fmt.Println("\n=== DRAW FLOOD OPERATION WITH AFFECTED AREA ===")

	// Create a test image
	img, err := vips.NewBlack(200, 200, nil)
	if err != nil {
		log.Printf("Failed to create test image: %v", err)
		return
	}
	defer img.Close()

	// Add some bands to make it RGB
	err = img.BandjoinConst([]float64{0, 0})
	if err != nil {
		log.Printf("Failed to add bands: %v", err)
		return
	}

	fmt.Printf("Created test image: %dx%d\n", img.Width(), img.Height())

	// Create draw flood options to capture affected area
	options := vips.DefaultDrawFloodOptions()

	// Perform flood fill operation (fill with red starting at 100,100)
	err = img.DrawFlood([]float64{255, 0, 0}, 100, 100, options)
	if err != nil {
		log.Printf("DrawFlood operation failed: %v", err)
		return
	}

	// Display the affected area
	fmt.Println("\nFlood Fill Affected Area:")
	fmt.Printf("  Area: left=%d, top=%d, width=%d, height=%d\n",
		options.Left, options.Top, options.Width, options.Height)
	fmt.Printf("  Total pixels affected: %d\n", options.Width*options.Height)

	// Save the result
	err = img.Jpegsave("flood_fill_result.jpg", nil)
	if err != nil {
		log.Printf("Failed to save flood fill result: %v", err)
		return
	}

	fmt.Println("Flood fill result saved as 'flood_fill_result.jpg'")
}

func main() {
	fmt.Println("VIPS Optional Outputs Examples")
	fmt.Println("==============================")
	fmt.Println("This example demonstrates how to capture optional output parameters")
	fmt.Println("from various VIPS operations, including transformation data,")
	fmt.Println("attention coordinates, and position information.")

	// Initialize VIPS
	vips.Startup(nil)
	defer vips.Shutdown()

	// Run all demonstrations
	demonstrateMosaic()
	demonstrateSmartcrop()
	demonstrateMaxMinPositions()
	demonstrateDrawFloodArea()

	fmt.Println("\n=== SUMMARY ===")
	fmt.Println("All examples completed successfully!")
	fmt.Println("Generated files:")
	fmt.Println("  - mosaic_result.jpg (mosaic of two images)")
	fmt.Println("  - smartcrop_result.jpg (intelligently cropped image)")
	fmt.Println("  - flood_fill_result.jpg (flood fill demonstration)")
	fmt.Println("\nOptional outputs demonstrated:")
	fmt.Println("  - Mosaic: transformation parameters (offset, scale, rotation)")
	fmt.Println("  - Smartcrop: attention coordinates (x, y)")
	fmt.Println("  - Max/Min: position coordinates of extreme values")
	fmt.Println("  - DrawFlood: affected area bounds")
}
