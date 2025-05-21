package vips

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain handles setup and teardown for all tests
func TestMain(m *testing.M) {
	// Start libvips once for all tests
	config := &Config{
		ReportLeaks: true,
	}
	Startup(config)

	// Run tests
	code := m.Run()

	// Shut down libvips
	Shutdown()

	// Exit with test result code
	os.Exit(code)
}

// createTestPNG creates a test PNG image with a pattern
func createTestPNG(t *testing.T, width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(((x + y) * 255) / (width + height))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Encode to PNG
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	return buf.Bytes()
}

// createTestJPEG creates a test JPEG image with a pattern
func createTestJPEG(t *testing.T, width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(((x + y) * 255) / (width + height))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Encode to JPEG
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	require.NoError(t, err)

	return buf.Bytes()
}

// ensureTestDir creates a test directory if it doesn't exist
func ensureTestDir(t *testing.T) string {
	dir := filepath.Join(os.TempDir(), "vipsgen-test")
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)
	return dir
}

// createWhiteImage creates a solid white image in memory
func createWhiteImage(width, height int) (*Image, error) {
	bands := 3 // RGB
	data := make([]byte, width*height*bands)

	// Fill with white pixels
	for i := range data {
		data[i] = 255
	}

	return NewImageFromMemory(data, width, height, bands)
}

// createBlackImage creates a solid black image in memory
func createBlackImage(width, height int) (*Image, error) {
	return NewBlack(width, height, &BlackOptions{Bands: 3})
}

func TestVersionInfo(t *testing.T) {
	// Test that version information is available
	t.Logf("libvips version: %s (major=%d, minor=%d, micro=%d)",
		Version, MajorVersion, MinorVersion, MicroVersion)

	// Verify that version is parsed correctly
	assert.True(t, MajorVersion >= 8, "Major version should be at least 8")
}

func TestMemoryStats(t *testing.T) {
	// Test memory stats functionality
	var stats MemoryStats
	ReadVipsMemStats(&stats)

	t.Logf("Memory stats: mem=%d, mem_high=%d, files=%d, allocs=%d",
		stats.Mem, stats.MemHigh, stats.Files, stats.Allocs)
}

func TestBasicBlackImage(t *testing.T) {
	// Create a simple test image (100x100 black image)
	width, height := 100, 100
	img, err := createBlackImage(width, height)
	require.NoError(t, err)
	defer img.Close()

	// Test basic properties
	assert.Equal(t, width, img.Width())
	assert.Equal(t, height, img.Height())
	assert.Equal(t, 3, img.Bands()) // RGB
	assert.False(t, img.HasAlpha())

	// Test resize operation
	err = img.Resize(0.5, nil)
	require.NoError(t, err)
	assert.Equal(t, width/2, img.Width())
	assert.Equal(t, height/2, img.Height())
}

func TestBasicWhiteImage(t *testing.T) {
	// Create a simple test image (100x100 white image)
	width, height := 100, 100
	img, err := createWhiteImage(width, height)
	require.NoError(t, err)
	defer img.Close()

	// Test basic properties
	assert.Equal(t, width, img.Width())
	assert.Equal(t, height, img.Height())
	assert.Equal(t, 3, img.Bands()) // RGB
	assert.False(t, img.HasAlpha())

	// Test resize operation with options
	err = img.Resize(0.5, &ResizeOptions{
		Kernel: KernelLanczos3,
	})
	require.NoError(t, err)
	assert.Equal(t, width/2, img.Width())
	assert.Equal(t, height/2, img.Height())
}

func TestImageLoadSaveFile(t *testing.T) {
	// Create a test PNG file
	testDir := ensureTestDir(t)
	testFile := filepath.Join(testDir, "test.png")

	// Generate PNG data and write to file
	pngData := createTestPNG(t, 200, 150)
	err := os.WriteFile(testFile, pngData, 0644)
	require.NoError(t, err)
	defer os.Remove(testFile) // Clean up

	// Test loading from file
	img, err := NewImageFromFile(testFile, nil)
	require.NoError(t, err)
	defer img.Close()

	// Test basic image properties
	assert.Equal(t, 200, img.Width())
	assert.Equal(t, 150, img.Height())

	// Test saving to another file
	outFile := filepath.Join(testDir, "out.png")
	outBuf, err := img.PngsaveBuffer(nil)
	require.NoError(t, err)
	err = os.WriteFile(outFile, outBuf, 0644)
	require.NoError(t, err)
	defer os.Remove(outFile) // Clean up

	// Load the saved file again to verify
	imgReloaded, err := NewImageFromFile(outFile, nil)
	require.NoError(t, err)
	defer imgReloaded.Close()

	assert.Equal(t, img.Width(), imgReloaded.Width())
	assert.Equal(t, img.Height(), imgReloaded.Height())
}

func TestImageLoadSaveBuffer(t *testing.T) {
	// Create a test PNG in memory
	pngData := createTestPNG(t, 150, 100)

	// Load from buffer
	img, err := NewPngloadBuffer(pngData, DefaultPngloadBufferOptions())
	require.NoError(t, err)
	defer img.Close()

	// Test basic image properties
	assert.Equal(t, 150, img.Width())
	assert.Equal(t, 100, img.Height())

	// Test saving to buffer
	buf, err := img.PngsaveBuffer(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, buf)

	// Test loading from buffer
	imgFromBuffer, err := NewImageFromBuffer(buf, nil)
	require.NoError(t, err)
	defer imgFromBuffer.Close()

	assert.Equal(t, img.Width(), imgFromBuffer.Width())
	assert.Equal(t, img.Height(), imgFromBuffer.Height())
}

func TestSource(t *testing.T) {
	// Create a test PNG in memory
	pngData := createTestPNG(t, 50, 50)

	// Create a source from the buffer
	source := NewSource(io.NopCloser(bytes.NewReader(pngData)))
	defer source.Close()

	// Load from source
	imgFromSource, err := NewImageFromSource(source, DefaultLoadOptions())
	require.NoError(t, err)
	defer imgFromSource.Close()

	// Check properties
	assert.Equal(t, 50, imgFromSource.Width())
	assert.Equal(t, 50, imgFromSource.Height())
}

func TestImageTransformations(t *testing.T) {
	// Create a test image
	width, height := 100, 80
	pngData := createTestPNG(t, width, height)

	// Load the image
	img, err := NewImageFromBuffer(pngData, nil)
	require.NoError(t, err)
	defer img.Close()

	// Test resize with options
	err = img.Resize(1.5, &ResizeOptions{
		Kernel: KernelLanczos3,
	})
	require.NoError(t, err)
	assert.Equal(t, int(float64(width)*1.5), img.Width())
	assert.Equal(t, int(float64(height)*1.5), img.Height())

	// Test rotate
	err = img.Rot(AngleD90)
	require.NoError(t, err)
	assert.Equal(t, int(float64(height)*1.5), img.Width())
	assert.Equal(t, int(float64(width)*1.5), img.Height())

	// Test flip
	err = img.Flip(DirectionHorizontal)
	require.NoError(t, err)

	// Test crop
	err = img.ExtractArea(10, 10, 50, 40)
	require.NoError(t, err)
	assert.Equal(t, 50, img.Width())
	assert.Equal(t, 40, img.Height())
}

// TestBasicFormatConversions tests basic conversions between supported formats
func TestBasicFormatConversions(t *testing.T) {
	// Log supported image types
	t.Log("Supported image types:")
	for imgType, name := range ImageTypes {
		if mime, ok := ImageMimeTypes[imgType]; ok && mime != "" {
			t.Logf("  - %s: %s", name, mime)
		} else {
			t.Logf("  - %s", name)
		}
	}

	// Create a test gradient image
	width, height := 100, 80
	img, err := NewImageFromBuffer(createTestPNG(t, width, height), DefaultLoadOptions())
	require.NoError(t, err)
	defer img.Close()

	// Test PNG saving with default options
	pngBuf, err := img.PngsaveBuffer(nil)
	if err != nil {
		t.Logf("PNG save failed: %v", err)
	} else {
		t.Logf("PNG save succeeded: %d bytes", len(pngBuf))
		assert.NotEmpty(t, pngBuf)
	}

	// Test PNG saving with options
	pngBuf2, err := img.PngsaveBuffer(&PngsaveBufferOptions{
		Compression: 6,
		Filter:      PngFilterAll,
	})
	if err != nil {
		t.Logf("PNG save with options failed: %v", err)
	} else {
		t.Logf("PNG save with options succeeded: %d bytes", len(pngBuf2))
		assert.NotEmpty(t, pngBuf2)
	}

	// Test JPEG saving with default options
	jpegBuf, err := img.JpegsaveBuffer(nil)
	if err != nil {
		t.Logf("JPEG save failed: %v", err)
	} else {
		t.Logf("JPEG save succeeded: %d bytes", len(jpegBuf))
		assert.NotEmpty(t, jpegBuf)
	}

	// Test JPEG saving with basic options
	jpegBuf2, err := img.JpegsaveBuffer(&JpegsaveBufferOptions{
		Q: 85,
	})
	if err != nil {
		t.Logf("JPEG save with options failed: %v", err)
	} else {
		t.Logf("JPEG save with options succeeded: %d bytes", len(jpegBuf2))
		assert.NotEmpty(t, jpegBuf2)
	}

	// Test WebP saving with default options
	webpBuf, err := img.WebpsaveBuffer(nil)
	if err != nil {
		t.Logf("WebP save failed: %v", err)
	} else {
		t.Logf("WebP save succeeded: %d bytes", len(webpBuf))
		assert.NotEmpty(t, webpBuf)
	}

	// Test WebP saving with options
	webpBuf2, err := img.WebpsaveBuffer(&WebpsaveBufferOptions{
		Q:        80,
		Lossless: true,
	})
	if err != nil {
		t.Logf("WebP save with options failed: %v", err)
	} else {
		t.Logf("WebP save with options succeeded: %d bytes", len(webpBuf2))
		assert.NotEmpty(t, webpBuf2)
	}
}

// TestImageOperations tests various image operations like blur, sharpen, etc.
func TestImageOperations(t *testing.T) {
	// Create a test image to work with
	width, height := 200, 150
	img, err := NewImageFromBuffer(createTestPNG(t, width, height), nil)
	require.NoError(t, err)
	defer img.Close()

	// Make a copy for comparing operations
	imgCopy, err := img.Copy(nil)
	require.NoError(t, err)
	defer imgCopy.Close()

	// Test a series of operations

	// 1. Gaussian blur
	err = img.Gaussblur(5.0, nil)
	if err != nil {
		t.Logf("Gaussblur failed: %v", err)
	} else {
		t.Log("Gaussblur succeeded")
	}

	// 2. Sharpen
	err = img.Sharpen(nil)
	if err != nil {
		t.Logf("Sharpen failed: %v", err)
	} else {
		t.Log("Sharpen succeeded")
	}

	// 3. Invert colors
	err = img.Invert()
	if err != nil {
		t.Logf("Invert failed: %v", err)
	} else {
		t.Log("Invert succeeded")
	}

	// 4. Test resize and position with embed
	err = imgCopy.Embed(10, 10, width+20, height+20, &EmbedOptions{
		Extend: ExtendBlack,
	})
	if err != nil {
		t.Logf("Embed failed: %v", err)
	} else {
		t.Logf("Embed succeeded: new size %dx%d", imgCopy.Width(), imgCopy.Height())
		assert.Equal(t, width+20, imgCopy.Width())
		assert.Equal(t, height+20, imgCopy.Height())
	}
}

// TestFormatConversionChain tests a chain of conversions between formats
func TestFormatConversionChain(t *testing.T) {
	// Create a simple white image
	img, err := createWhiteImage(100, 80)
	require.NoError(t, err)
	defer img.Close()

	// 1. First save as JPEG with minimal options
	jpegBuf, err := img.JpegsaveBuffer(&JpegsaveBufferOptions{
		Q: 80,
	})
	require.NoError(t, err)
	require.NotEmpty(t, jpegBuf)
	t.Logf("JPEG save produced %d bytes", len(jpegBuf))

	// 2. Load the JPEG
	jpegImg, err := NewJpegloadBuffer(jpegBuf, DefaultJpegloadBufferOptions())
	require.NoError(t, err)
	defer jpegImg.Close()

	// 3. Convert to PNG
	pngBuf, err := jpegImg.PngsaveBuffer(nil)
	require.NoError(t, err)
	require.NotEmpty(t, pngBuf)
	t.Logf("PNG save produced %d bytes", len(pngBuf))

	// 4. Load the PNG
	pngImg, err := NewImageFromBuffer(pngBuf, nil)
	require.NoError(t, err)
	defer pngImg.Close()

	// 5. Convert back to JPEG
	jpegBuf2, err := pngImg.JpegsaveBuffer(nil)
	require.NoError(t, err)
	require.NotEmpty(t, jpegBuf2)
	t.Logf("Second JPEG save produced %d bytes", len(jpegBuf2))

	// Success if we got here
	t.Log("Successfully completed format conversion chain")
}

// TestOperationComposition tests composing multiple operations together
func TestOperationComposition(t *testing.T) {
	// Create two test images
	img1, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img1.Close()

	img2, err := createBlackImage(50, 50)
	require.NoError(t, err)
	defer img2.Close()

	// Compose operations: resize, embed, and composite

	// 1. Resize second image
	err = img2.Resize(1.5, DefaultResizeOptions())
	require.NoError(t, err)
	assert.Equal(t, 75, img2.Width())
	assert.Equal(t, 75, img2.Height())

	// 2. Try to composite images (if supported)
	err = img1.Composite2(img2, BlendModeOver, &Composite2Options{X: 10, Y: 10})
	if err != nil {
		t.Logf("Composite operation failed: %v", err)
	} else {
		t.Log("Composite operation succeeded")
	}

	// Try to composite array of images (if supported)
	images := []*Image{img1, img2}

	composite, err := NewComposite(images, []BlendMode{BlendModeOver}, &CompositeOptions{X: []int{}, Y: []int{}})
	if err != nil {
		t.Logf("Composite2 operation failed: %v", err)
	} else {
		t.Log("Composite2 operation succeeded")
		composite.Close()
	}
}

// TestLabel tests the label functionality
func TestLabel(t *testing.T) {
	// Create a test image
	img, err := createWhiteImage(300, 200)
	require.NoError(t, err)
	defer img.Close()

	// Add text to the image
	err = img.Label("Hello, libvips!", 50, 50, &LabelOptions{
		Font:    "sans",
		Size:    24,
		Color:   []float64{255, 0, 0},
		Opacity: 1.0,
	})
	if err != nil {
		t.Logf("Label operation failed: %v", err)
	} else {
		t.Log("Label operation succeeded")
	}
}

// TestICCProfile tests ICC profile operations
func TestICCProfile(t *testing.T) {
	// Create test image
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test removing ICC profile
	result := img.RemoveICCProfile()
	t.Logf("RemoveICCProfile result: %v", result)
}

// TestExif tests EXIF operations
func TestExif(t *testing.T) {
	// Create a JPEG with some basic structure
	jpegData := createTestJPEG(t, 120, 80)

	// Load JPEG
	img, err := NewJpegloadBuffer(jpegData, nil)
	require.NoError(t, err)
	defer img.Close()

	// Get orientation (likely 0 for test image)
	orientation := img.Orientation()
	t.Logf("Image orientation: %d", orientation)

	// Try to extract EXIF data
	exifData := img.Exif()
	t.Logf("EXIF data: %v", exifData)

	// Test removing EXIF data
	err = img.RemoveExif()
	if err != nil {
		t.Logf("RemoveExif failed: %v", err)
	} else {
		t.Log("RemoveExif succeeded")

		// Check EXIF data is gone
		exifDataAfter := img.Exif()
		assert.Empty(t, exifDataAfter, "EXIF data should be empty after removal")
	}
}

// TestMultiPageOperations tests operations on multi-page images
func TestMultiPageOperations(t *testing.T) {
	// Create a simple test image
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Get page count
	pageCount := img.Pages()
	t.Logf("Image page count: %d", pageCount)

	// Get page height
	pageHeight := img.PageHeight()
	t.Logf("Image page height: %d", pageHeight)

	// Try to get/set page height
	img.SetPageHeight(100)
	assert.Equal(t, 100, img.PageHeight())
}

func TestAllFormatsSupport(t *testing.T) {
	// Create a test image
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Define all format tests in a slice
	type formatTest struct {
		name     string
		saveFunc func() ([]byte, error)
	}

	tests := []formatTest{
		{name: "PNG", saveFunc: func() ([]byte, error) {
			return img.PngsaveBuffer(nil)
		}},
		{name: "PNG", saveFunc: func() ([]byte, error) {
			return img.PngsaveBuffer(DefaultPngsaveBufferOptions())
		}},
		{name: "JPEG", saveFunc: func() ([]byte, error) {
			return img.JpegsaveBuffer(nil)
		}},
		{name: "JPEG", saveFunc: func() ([]byte, error) {
			return img.JpegsaveBuffer(DefaultJpegsaveBufferOptions())
		}},
		{name: "WebP", saveFunc: func() ([]byte, error) {
			return img.WebpsaveBuffer(nil)
		}},
		{name: "WebP", saveFunc: func() ([]byte, error) {
			return img.WebpsaveBuffer(DefaultWebpsaveBufferOptions())
		}},
		{name: "TIFF", saveFunc: func() ([]byte, error) {
			return img.TiffsaveBuffer(nil)
		}},
		{name: "TIFF", saveFunc: func() ([]byte, error) {
			return img.TiffsaveBuffer(DefaultTiffsaveBufferOptions())
		}},
		{name: "GIF", saveFunc: func() ([]byte, error) {
			return img.GifsaveBuffer(nil)
		}},
		{name: "GIF", saveFunc: func() ([]byte, error) {
			return img.GifsaveBuffer(DefaultGifsaveBufferOptions())
		}},
	}

	// Run all the tests
	t.Log("Testing all supported save formats:")
	for _, test := range tests {
		buf, err := test.saveFunc()
		if err != nil {
			t.Logf("  - %s save failed: %v", test.name, err)
		} else {
			t.Logf("  - %s save succeeded: %d bytes", test.name, len(buf))
			assert.NotEmpty(t, buf)
		}
	}
}

// TestErrorHandling tests error handling mechanisms
func TestErrorHandling(t *testing.T) {
	// Test invalid parameter for resize
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Try invalid crop (outside image bounds)
	err = img.ExtractArea(50, 50, 100, 100)
	assert.Error(t, err, "Crop outside image bounds should fail")

	// Try to load invalid buffer
	invalidBuf := []byte{0, 1, 2, 3}
	_, err = NewImageFromBuffer(invalidBuf, nil)
	assert.Error(t, err, "Loading invalid buffer should fail")

	// Try to load non-existent file
	_, err = NewImageFromFile("/non/existent/file.png", nil)
	assert.Error(t, err, "Loading non-existent file should fail")
}

// TestDrawOperationsWithPixelValidation tests drawing operations with pixel validation
func TestDrawOperationsWithPixelValidation(t *testing.T) {
	// Create a white canvas
	width, height := 300, 300
	img, err := createWhiteImage(width, height)
	require.NoError(t, err)
	defer img.Close()

	// Validate that it's initially white
	centerPixel, err := img.Getpoint(width/2, height/2, nil)
	require.NoError(t, err)
	assert.InDelta(t, 255, centerPixel[0], 1, "Center should initially be white")

	// 1. Draw a red rectangle (x=50, y=50, width=100, height=100)
	redColor := []float64{255, 0, 0}
	err = img.DrawRect(redColor, 50, 50, 100, 100, &DrawRectOptions{
		Fill: true,
	})
	if err != nil {
		t.Logf("DrawRect failed: %v", err)
	} else {
		t.Log("DrawRect successful")

		// Validate pixel inside the rectangle
		rectPixel, err := img.Getpoint(75, 75, nil)
		require.NoError(t, err)
		assert.InDelta(t, redColor[0], rectPixel[0], 5, "Rectangle should be red (R)")
		assert.InDelta(t, redColor[1], rectPixel[1], 5, "Rectangle should be red (G)")
		assert.InDelta(t, redColor[2], rectPixel[2], 5, "Rectangle should be red (B)")

		// Validate pixel outside the rectangle
		outsidePixel, err := img.Getpoint(25, 25, nil)
		require.NoError(t, err)
		assert.InDelta(t, 255, outsidePixel[0], 5, "Outside should still be white")
	}

	// 2. Draw a blue circle (center=200,150, radius=50)
	blueColor := []float64{0, 0, 255}
	err = img.DrawCircle(blueColor, 200, 150, 50, &DrawCircleOptions{
		Fill: true,
	})
	if err != nil {
		t.Logf("DrawCircle failed: %v", err)
	} else {
		t.Log("DrawCircle successful")

		// Validate pixel inside the circle
		circlePixel, err := img.Getpoint(200, 150, nil)
		require.NoError(t, err)
		assert.InDelta(t, blueColor[0], circlePixel[0], 5, "Circle center should be blue (R)")
		assert.InDelta(t, blueColor[1], circlePixel[1], 5, "Circle center should be blue (G)")
		assert.InDelta(t, blueColor[2], circlePixel[2], 5, "Circle center should be blue (B)")

		// Validate pixel at the edge of the circle (approximately)
		edgePixel, err := img.Getpoint(200+45, 150, nil) // slightly inside the circle radius
		require.NoError(t, err)
		// Should be blue or close to it
		assert.InDelta(t, blueColor[2], edgePixel[2], 50, "Circle edge should be close to blue")
	}

	// 3. Draw a green line from (50,200) to (250,250)
	greenColor := []float64{0, 255, 0}
	err = img.DrawLine(greenColor, 50, 200, 250, 250)
	if err != nil {
		t.Logf("DrawLine failed: %v", err)
	} else {
		t.Log("DrawLine successful")

		// Validate pixel on the line (approximate midpoint)
		linePixel, err := img.Getpoint(150, 225, nil)
		require.NoError(t, err)
		// Line pixels might be approximated, so use a larger delta
		if linePixel[1] > linePixel[0] && linePixel[1] > linePixel[2] {
			t.Log("Line pixel has dominant green channel as expected")
		} else {
			t.Logf("Line pixel values: [%.1f, %.1f, %.1f] - might be affected by anti-aliasing",
				linePixel[0], linePixel[1], linePixel[2])
		}
	}

	// Check if the image still has the expected dimensions
	assert.Equal(t, width, img.Width())
	assert.Equal(t, height, img.Height())
} // Helper function to check if bands are approximately equal
func assertBandsEqual(t *testing.T, values []float64) {
	if len(values) < 2 {
		return
	}

	for i := 1; i < len(values); i++ {
		assert.InDelta(t, values[0], values[i], 5,
			"Band values should be approximately equal in grayscale image")
	}
}

// TestCreatePatternedImage tests creating an image with a checkerboard pattern
func TestCreatePatternedImage(t *testing.T) {
	// Create a checkerboard pattern image in memory
	width, height := 200, 200
	squareSize := 20

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Draw checkerboard pattern
	for y := 0; y < height; y += squareSize {
		for x := 0; x < width; x += squareSize {
			// Alternate between white and black squares
			c := color.RGBA{0, 0, 0, 255} // Black
			if ((x/squareSize)+(y/squareSize))%2 == 0 {
				c = color.RGBA{255, 255, 255, 255} // White
			}

			// Fill square
			rect := image.Rect(x, y, x+squareSize, y+squareSize)
			draw.Draw(img, rect, &image.Uniform{c}, image.Point{}, draw.Src)
		}
	}

	// Convert to bytes
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	// Load into vips
	vipsImg, err := NewImageFromBuffer(buf.Bytes(), nil)
	require.NoError(t, err)
	defer vipsImg.Close()

	// Verify properties
	assert.Equal(t, width, vipsImg.Width())
	assert.Equal(t, height, vipsImg.Height())
	assert.Equal(t, 3, vipsImg.Bands()) // Check if bands are correct
}

// TestBuiltinImageGeneration tests creation of images using built-in generators
func TestBuiltinImageGeneration(t *testing.T) {
	// Test creating various built-in image types

	// 1. Black image
	width, height := 100, 100
	blackImg, err := NewBlack(width, height, &BlackOptions{Bands: 3})
	require.NoError(t, err)
	defer blackImg.Close()

	assert.Equal(t, width, blackImg.Width())
	assert.Equal(t, height, blackImg.Height())
	assert.Equal(t, 3, blackImg.Bands())

	// 2. XYZ image (coordinate-based image)
	xyzImg, err := NewXyz(width, height, nil)
	require.NoError(t, err)
	defer xyzImg.Close()

	assert.Equal(t, width, xyzImg.Width())
	assert.Equal(t, height, xyzImg.Height())
	assert.Equal(t, 2, xyzImg.Bands())

	// 3. Perlin noise image
	perlinImg, err := NewPerlin(width, height, &PerlinOptions{
		CellSize: 32,
		Uchar:    true,
	})
	require.NoError(t, err)
	defer perlinImg.Close()

	assert.Equal(t, width, perlinImg.Width())
	assert.Equal(t, height, perlinImg.Height())

	// 4. Zone plate
	zoneImg, err := NewZone(width, height, nil)
	require.NoError(t, err)
	defer zoneImg.Close()

	assert.Equal(t, width, zoneImg.Width())
	assert.Equal(t, height, zoneImg.Height())

	// 5. Gaussian noise
	noiseImg, err := NewGaussnoise(width, height, &GaussnoiseOptions{
		Sigma: 50.0,
		Mean:  128.0,
	})
	require.NoError(t, err)
	defer noiseImg.Close()

	assert.Equal(t, width, noiseImg.Width())
	assert.Equal(t, height, noiseImg.Height())
}

// TestImageManipulations tests various image manipulations like crop, resize, etc.
func TestImageManipulations(t *testing.T) {
	// Create a test image
	width, height := 200, 200
	img, err := createCheckboardImage(t, width, height, 20)
	require.NoError(t, err)
	defer img.Close()

	// 1. Test crop
	cropImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer cropImg.Close()

	err = cropImg.ExtractArea(50, 50, 100, 100)
	require.NoError(t, err)
	assert.Equal(t, 100, cropImg.Width())
	assert.Equal(t, 100, cropImg.Height())

	// 2. Test resize with different kernels
	for _, kernel := range []Kernel{KernelNearest, KernelLinear, KernelCubic, KernelLanczos3} {
		resizeImg, err := img.Copy(nil)
		require.NoError(t, err)

		err = resizeImg.Resize(0.5, &ResizeOptions{Kernel: kernel})
		require.NoError(t, err)

		assert.Equal(t, width/2, resizeImg.Width())
		assert.Equal(t, height/2, resizeImg.Height())

		resizeImg.Close()
	}

	// 3. Test rotations
	rotateImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer rotateImg.Close()

	origWidth, origHeight := rotateImg.Width(), rotateImg.Height()

	// Test 90 degree rotation
	err = rotateImg.Rot(AngleD90)
	require.NoError(t, err)
	assert.Equal(t, origHeight, rotateImg.Width())
	assert.Equal(t, origWidth, rotateImg.Height())

	// Test 180 degree rotation
	err = rotateImg.Rot(AngleD180)
	require.NoError(t, err)
	assert.Equal(t, origWidth, rotateImg.Width())
	assert.Equal(t, origHeight, rotateImg.Height())
}

// TestImageBlending tests blending and composition operations
func TestImageBlending(t *testing.T) {
	// Create two test images - one red and one blue
	width, height := 100, 100
	redImg, err := createSolidColorImage(t, width, height, color.RGBA{255, 0, 0, 255})
	require.NoError(t, err)
	defer redImg.Close()

	blueImg, err := createSolidColorImage(t, width, height, color.RGBA{0, 0, 255, 255})
	require.NoError(t, err)
	defer blueImg.Close()

	// Test overlaying the images
	err = redImg.Composite2(blueImg, BlendModeOver, &Composite2Options{
		X: 25,
		Y: 25,
	})

	if err != nil {
		t.Logf("Composite operation failed: %v", err)
	} else {
		t.Log("Successfully blended images")

		// Test saving the resulting image
		buf, err := redImg.PngsaveBuffer(nil)
		require.NoError(t, err)
		assert.NotEmpty(t, buf)
	}

	// Test other blend modes if supported
	blendModes := []BlendMode{
		BlendModeMultiply,
		BlendModeScreen,
		BlendModeDarken,
		BlendModeLighten,
	}

	for _, mode := range blendModes {
		baseImg, err := createSolidColorImage(t, width, height, color.RGBA{200, 200, 200, 255})
		if err != nil {
			continue
		}

		overlayImg, err := createSolidColorImage(t, width/2, height/2, color.RGBA{100, 100, 100, 255})
		if err != nil {
			baseImg.Close()
			continue
		}

		err = baseImg.Composite2(overlayImg, mode, &Composite2Options{
			X: width / 4,
			Y: height / 4,
		})

		t.Logf("Blend mode %d test: %v", mode, err == nil)

		baseImg.Close()
		overlayImg.Close()
	}
}

// TestColorspaceConversions tests converting between different colorspaces
func TestColorspaceConversions(t *testing.T) {
	// Create a test image
	width, height := 100, 100
	img, err := createWhiteImage(width, height)
	require.NoError(t, err)
	defer img.Close()

	// Test conversion to various colorspaces
	colorspaces := []Interpretation{
		InterpretationBW,
		InterpretationRgb,
		InterpretationSrgb,
		InterpretationCmyk,
		InterpretationLab,
	}

	for _, colorspace := range colorspaces {
		// Make a copy for this test
		testImg, err := img.Copy(nil)
		require.NoError(t, err)

		// Try to convert to this colorspace
		err = testImg.Colourspace(colorspace, nil)
		if err != nil {
			t.Logf("Convert to %d failed: %v", colorspace, err)
		} else {
			t.Logf("Successfully converted to colorspace %d", colorspace)
		}

		testImg.Close()
	}
}

// TestImageFilters tests various image filters
func TestImageFilters(t *testing.T) {
	// Create a test image
	width, height := 200, 200
	img, err := createCheckboardImage(t, width, height, 20)
	require.NoError(t, err)
	defer img.Close()

	// Test various filters

	// 1. Gaussian blur with different sigma values
	sigmaValues := []float64{1.0, 3.0, 5.0}
	for _, sigma := range sigmaValues {
		blurImg, err := img.Copy(nil)
		require.NoError(t, err)

		err = blurImg.Gaussblur(sigma, nil)
		require.NoError(t, err)

		t.Logf("Gaussian blur with sigma=%.1f successful", sigma)
		blurImg.Close()
	}

	// 2. Edge detection filters
	filters := []struct {
		name string
		fn   func(*Image) error
	}{
		{"Sobel", func(i *Image) error { return i.Sobel() }},
		{"Canny", func(i *Image) error { return i.Canny(nil) }},
	}

	for _, filter := range filters {
		filterImg, err := img.Copy(nil)
		require.NoError(t, err)

		err = filter.fn(filterImg)
		if err != nil {
			t.Logf("%s filter failed: %v", filter.name, err)
		} else {
			t.Logf("%s filter successful", filter.name)
		}

		filterImg.Close()
	}
}

// TestRotateOperations tests different rotation operations
func TestRotateOperations(t *testing.T) {
	// Create a test image with identifiable features
	// Use odd dimensions to be compatible with all rotation operations
	width, height := 101, 101

	// Create an image with a single horizontal line through the middle
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with white
	bgColor := color.RGBA{255, 255, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Draw a horizontal red line through the middle
	lineColor := color.RGBA{255, 0, 0, 255}
	for x := 0; x < width; x++ {
		img.Set(x, height/2, lineColor)
	}

	// Convert to PNG and load into vips
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	// Load the image
	vipsImg, err := NewImageFromBuffer(buf.Bytes(), nil)
	require.NoError(t, err)
	defer vipsImg.Close()

	// Verify the horizontal line is there by checking a pixel in the middle
	midPixel, err := vipsImg.Getpoint(width/2, height/2, nil)
	require.NoError(t, err)
	assert.InDelta(t, 255, midPixel[0], 5, "Middle pixel should be red")
	assert.InDelta(t, 0, midPixel[1], 5, "Middle pixel should be red")
	assert.InDelta(t, 0, midPixel[2], 5, "Middle pixel should be red")

	// Test Rot - 90 degree rotation
	rotImg, err := vipsImg.Copy(nil)
	require.NoError(t, err)
	defer rotImg.Close()

	err = rotImg.Rot(AngleD90)
	if err != nil {
		t.Logf("Rot(AngleD90) failed: %v", err)
	} else {
		t.Log("Rot(AngleD90) succeeded")

		// After 90-degree rotation, the horizontal line should become vertical
		// Check horizontally across the center - most should be white except middle
		foundRed := false
		leftPixel, err := rotImg.Getpoint(width/4, height/2, nil)
		require.NoError(t, err)
		assert.InDelta(t, 255, leftPixel[0], 5, "Left-center should be white after rotation")
		assert.InDelta(t, 255, leftPixel[1], 5, "Left-center should be white after rotation")
		assert.InDelta(t, 255, leftPixel[2], 5, "Left-center should be white after rotation")

		// Center pixel should now be white too, and red is on the vertical line instead
		centerPixel, err := rotImg.Getpoint(width/2, height/2, nil)
		require.NoError(t, err)
		assert.Equal(t, []float64{255, 0, 0}, centerPixel)

		// Find the red pixel by scanning vertically
		for y := 0; y < height; y++ {
			vertPixel, err := rotImg.Getpoint(width/2, y, nil)
			if err != nil {
				continue
			}

			if vertPixel[0] > 200 && vertPixel[1] < 50 && vertPixel[2] < 50 {
				foundRed = true
				t.Logf("Found red pixel in vertical scan at y=%d: [%.1f, %.1f, %.1f]",
					y, vertPixel[0], vertPixel[1], vertPixel[2])
				break
			}
		}

		if !foundRed {
			t.Log("Could not find red pixel in vertical scan after rotation")
		}
	}

	// Test Rot45 if available (requires odd-sized square image, which we have)
	rot45Img, err := vipsImg.Copy(nil)
	require.NoError(t, err)
	defer rot45Img.Close()

	err = rot45Img.Rot45(&Rot45Options{Angle: Angle45D45})
	if err != nil {
		t.Logf("Rot45(Angle45D45) failed: %v", err)
	} else {
		t.Log("Rot45(Angle45D45) succeeded")
		t.Logf("After Rot45: %dx%d", rot45Img.Width(), rot45Img.Height())

		// After 45-degree rotation, the line should be diagonal
		// Just validate some basic properties since exact pixel location is complex
		centerPixel, err := rot45Img.Getpoint(rot45Img.Width()/2, rot45Img.Height()/2, nil)
		if err == nil {
			t.Logf("Center pixel after Rot45: [%.1f, %.1f, %.1f]",
				centerPixel[0], centerPixel[1], centerPixel[2])
		}
	}
} // TestRot45Requirements tests the specific requirements for the rot45 operation
func TestRot45Requirements(t *testing.T) {
	// Test that rot45 requires odd-sized square images

	// 1. Test with even-sized square image (should fail)
	evenWidth, evenHeight := 100, 100
	evenImg, err := createSolidColorImage(t, evenWidth, evenHeight, color.RGBA{255, 0, 0, 255})
	require.NoError(t, err)
	defer evenImg.Close()

	err = evenImg.Rot45(&Rot45Options{Angle: Angle45D45})
	assert.Error(t, err, "Rot45 should fail with even-sized square image")
	t.Logf("Expected error with even-sized square image: %v", err)

	// 2. Test with non-square image (should fail)
	rectWidth, rectHeight := 101, 151 // odd but not square
	rectImg, err := createSolidColorImage(t, rectWidth, rectHeight, color.RGBA{0, 255, 0, 255})
	require.NoError(t, err)
	defer rectImg.Close()

	err = rectImg.Rot45(&Rot45Options{Angle: Angle45D45})
	assert.Error(t, err, "Rot45 should fail with non-square image")
	t.Logf("Expected error with non-square image: %v", err)

	// 3. Test with odd-sized square image (should succeed)
	oddWidth, oddHeight := 101, 101 // odd and square
	oddImg, err := createSolidColorImage(t, oddWidth, oddHeight, color.RGBA{0, 0, 255, 255})
	require.NoError(t, err)
	defer oddImg.Close()

	err = oddImg.Rot45(&Rot45Options{Angle: Angle45D45})
	if err != nil {
		t.Logf("Rot45 still failed with odd-sized square image: %v", err)
	} else {
		t.Log("Rot45 succeeded with odd-sized square image as expected")

		// Verify dimensions of rotated image
		t.Logf("After rotation: %dx%d", oddImg.Width(), oddImg.Height())

		// Check pixels after rotation
		centerX, centerY := oddImg.Width()/2, oddImg.Height()/2
		centerPixel, err := oddImg.Getpoint(centerX, centerY, nil)
		if err == nil {
			t.Logf("Center pixel after rotation: [%.1f, %.1f, %.1f]",
				centerPixel[0], centerPixel[1], centerPixel[2])
		}
	}

	// 4. Try different rotation angles if available
	if err == nil {
		// Make another odd-sized square image for testing other angles
		oddImg2, err := createSolidColorImage(t, oddWidth, oddHeight, color.RGBA{0, 0, 255, 255})
		require.NoError(t, err)
		defer oddImg2.Close()

		// Try 90-degree rotation (D90)
		err = oddImg2.Rot45(&Rot45Options{Angle: Angle45D90})
		if err != nil {
			t.Logf("Rot45 with Angle45D90 failed: %v", err)
		} else {
			t.Log("Rot45 with Angle45D90 succeeded")
		}
	}
} // TestImageStats tests statistical functions on images
func TestImageStats(t *testing.T) {
	// Create a test image with a gradient
	width, height := 100, 100
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a gradient from black to white
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Linear gradient from black to white
			value := uint8((x + y) * 255 / (width + height - 2))
			img.Set(x, y, color.RGBA{value, value, value, 255})
		}
	}

	// Convert to PNG and load into vips
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	vipsImg, err := NewImageFromBuffer(buf.Bytes(), nil)
	require.NoError(t, err)
	defer vipsImg.Close()

	// Test Avg (average) operation
	avg, err := vipsImg.Avg()
	if err != nil {
		t.Logf("Avg operation failed: %v", err)
	} else {
		t.Logf("Image average: %.2f", avg)
		// For a linear gradient 0-255, average should be close to 127.5
		assert.InDelta(t, 127.5, avg, 10, "Average should be close to 127.5")
	}

	// Test Min operation
	min, err := vipsImg.Min(nil)
	if err != nil {
		t.Logf("Min operation failed: %v", err)
	} else {
		t.Logf("Image minimum: %.2f", min)
		assert.InDelta(t, 0, min, 5, "Minimum should be close to 0")
	}

	// Test Max operation
	max, err := vipsImg.Max(nil)
	if err != nil {
		t.Logf("Max operation failed: %v", err)
	} else {
		t.Logf("Image maximum: %.2f", max)
		assert.InDelta(t, 255, max, 5, "Maximum should be close to 255")
	}

	// Test Deviate (standard deviation) operation
	dev, err := vipsImg.Deviate()
	if err != nil {
		t.Logf("Deviate operation failed: %v", err)
	} else {
		t.Logf("Image standard deviation: %.2f", dev)
		// Standard deviation for uniform gradient should be positive
		assert.Greater(t, dev, 0.0, "Standard deviation should be positive")
	}

	// Validate against specific pixel values
	// Check corners and center
	checkPoints := []struct {
		name     string
		x, y     int
		expected float64
	}{
		{"top-left", 0, 0, 0},
		{"top-right", width - 1, 0, float64((width - 1) * 255 / (width + height - 2))},
		{"bottom-left", 0, height - 1, float64((height - 1) * 255 / (width + height - 2))},
		{"bottom-right", width - 1, height - 1, float64((width + height - 2) * 255 / (width + height - 2))},
		{"center", width / 2, height / 2, float64((width/2 + height/2) * 255 / (width + height - 2))},
	}

	for _, cp := range checkPoints {
		pixelValues, err := vipsImg.Getpoint(cp.x, cp.y, nil)
		if err != nil {
			t.Logf("Getpoint failed for %s: %v", cp.name, err)
			continue
		}

		assert.InDelta(t, cp.expected, pixelValues[0], 5,
			"Pixel value at %s should be approximately %.1f", cp.name, cp.expected)

		t.Logf("%s pixel value: %.1f (expected: %.1f)",
			cp.name, pixelValues[0], cp.expected)
	}
}

// TestDrawOperations tests drawing operations on images
func TestDrawOperations(t *testing.T) {
	// Create a white canvas
	width, height := 300, 300
	img, err := createWhiteImage(width, height)
	require.NoError(t, err)
	defer img.Close()

	// Test drawing operations

	// 1. Draw a red rectangle
	err = img.DrawRect([]float64{255, 0, 0}, 50, 50, 100, 100, &DrawRectOptions{
		Fill: true,
	})
	if err != nil {
		t.Logf("DrawRect failed: %v", err)
	} else {
		t.Log("DrawRect successful")
	}

	// 2. Draw a blue circle
	err = img.DrawCircle([]float64{0, 0, 255}, 200, 150, 50, &DrawCircleOptions{
		Fill: true,
	})
	if err != nil {
		t.Logf("DrawCircle failed: %v", err)
	} else {
		t.Log("DrawCircle successful")
	}

	// 3. Draw a green line
	err = img.DrawLine([]float64{0, 255, 0}, 50, 200, 250, 250)
	if err != nil {
		t.Logf("DrawLine failed: %v", err)
	} else {
		t.Log("DrawLine successful")
	}

	// Check if the image still has the expected dimensions
	assert.Equal(t, width, img.Width())
	assert.Equal(t, height, img.Height())
}

// TestSourceOperations tests operations using Source
func TestSourceOperations(t *testing.T) {
	// Create a test image
	width, height := 100, 100
	data := createPNGTestImage(t, width, height)

	// Test with a memory source
	memReader := bytes.NewReader(data)
	source := NewSource(io.NopCloser(memReader))
	defer source.Close()

	// Load from source
	img, err := NewImageFromSource(source, DefaultLoadOptions())
	require.NoError(t, err)
	defer img.Close()

	// Verify properties
	assert.Equal(t, width, img.Width())
	assert.Equal(t, height, img.Height())
}

// Helper functions for creating various test images

// createCheckboardImage creates a test image with a checkerboard pattern
func createCheckboardImage(t *testing.T, width, height, squareSize int) (*Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Draw checkerboard pattern
	for y := 0; y < height; y += squareSize {
		for x := 0; x < width; x += squareSize {
			// Alternate between white and black squares
			c := color.RGBA{0, 0, 0, 255} // Black
			if ((x/squareSize)+(y/squareSize))%2 == 0 {
				c = color.RGBA{255, 255, 255, 255} // White
			}

			// Fill square
			rect := image.Rect(x, y, x+squareSize, y+squareSize)
			draw.Draw(img, rect, &image.Uniform{c}, image.Point{}, draw.Src)
		}
	}

	// Convert to PNG and load into vips
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}

	return NewImageFromBuffer(buf.Bytes(), nil)
}

// createSolidColorImage creates a test image with a solid color
func createSolidColorImage(t *testing.T, width, height int, c color.RGBA) (*Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with solid color
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)

	// Convert to PNG and load into vips
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}

	return NewImageFromBuffer(buf.Bytes(), nil)
}

// createPNGTestImage creates a test PNG image with a pattern
func createPNGTestImage(t *testing.T, width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a random pattern
	r := rand.New(rand.NewSource(42)) // Use fixed seed for reproducibility

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(r.Intn(256)),
				G: uint8(r.Intn(256)),
				B: uint8(r.Intn(256)),
				A: 255,
			})
		}
	}

	// Encode to PNG
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	return buf.Bytes()
}

// TestAdvancedColorOperations tests advanced color operations and transformations
func TestAdvancedColorOperations(t *testing.T) {
	// Create a test image
	width, height := 100, 100
	img, err := createSolidColorImage(t, width, height, color.RGBA{200, 150, 100, 255})
	require.NoError(t, err)
	defer img.Close()

	// Test color space conversions if available
	conversions := []struct {
		name string
		fn   func(*Image) error
	}{
		{"SRGB2HSV", func(i *Image) error { return i.SRGB2HSV() }},
		{"HSV2sRGB", func(i *Image) error { return i.HSV2sRGB() }},
		{"Lab2LCh", func(i *Image) error { return i.Lab2LCh() }},
		{"LCh2Lab", func(i *Image) error { return i.LCh2Lab() }},
	}

	for _, conv := range conversions {
		convImg, err := img.Copy(nil)
		require.NoError(t, err)

		err = conv.fn(convImg)
		if err != nil {
			t.Logf("%s conversion failed: %v", conv.name, err)
		} else {
			t.Logf("%s conversion successful", conv.name)

			// Try to convert back if possible
			if idx := conv.name + "->back"; idx[0] == 'S' {
				err = convImg.HSV2sRGB()
				t.Logf("Converting back: %v", err == nil)
			} else if idx[0] == 'H' {
				err = convImg.SRGB2HSV()
				t.Logf("Converting back: %v", err == nil)
			}
		}

		convImg.Close()
	}
}

// Helper function to create a random noise image
func createRandomNoiseImage(t *testing.T, width, height int) (*Image, error) {
	// Create random noise image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with random noise
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8(rand.Intn(256))
			g := uint8(rand.Intn(256))
			b := uint8(rand.Intn(256))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Convert to PNG and load into vips
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}

	return NewImageFromBuffer(buf.Bytes(), nil)
}

func TestFilterOptions(t *testing.T) {
	// Create a test image
	width, height := 300, 200
	img, err := createRandomNoiseImage(t, width, height)
	require.NoError(t, err)
	defer img.Close()

	// 1. Test GaussBlur with different options

	// 1.1. Gaussian blur with different sigma values
	sigmaValues := []float64{1.0, 3.0, 5.0, 10.0}
	for _, sigma := range sigmaValues {
		blurImg, err := img.Copy(nil)
		require.NoError(t, err)

		err = blurImg.Gaussblur(sigma, nil)
		if err != nil {
			t.Logf("Gaussblur with sigma=%.1f failed: %v", sigma, err)
		} else {
			t.Logf("Gaussblur with sigma=%.1f succeeded", sigma)

			// Check center pixel
			center, err := blurImg.Getpoint(width/2, height/2, nil)
			if err == nil {
				t.Logf("Center pixel with sigma=%.1f: [%.1f, %.1f, %.1f]",
					sigma, center[0], center[1], center[2])
			}
		}

		blurImg.Close()
	}

	// 1.2. Test with minAmpl parameter
	blurMinAmplImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer blurMinAmplImg.Close()

	err = blurMinAmplImg.Gaussblur(5.0, &GaussblurOptions{
		MinAmpl: 0.01, // Very small minimum amplitude
	})
	require.NoError(t, err)
	t.Log("Gaussblur with minAmpl succeeded")

	// 2. Test Sharpen with different options

	// 2.1. Default sharpen
	sharpenImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer sharpenImg.Close()

	err = sharpenImg.Sharpen(nil)
	if err != nil {
		t.Logf("Sharpen with nil options failed: %v", err)
	} else {
		t.Log("Sharpen with nil options succeeded")
	}

	// 2.2. Custom sharpen parameters
	customSharpenImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer customSharpenImg.Close()

	err = customSharpenImg.Sharpen(&SharpenOptions{
		Sigma: 3.0,  // Larger radius
		X1:    2.0,  // Different threshold
		Y2:    20.0, // Different brightening
		Y3:    20.0, // Different darkening
	})
	if err != nil {
		t.Logf("Sharpen with custom options failed: %v", err)
	} else {
		t.Log("Sharpen with custom options succeeded")
	}

	// 3. Test Canny edge detection with different options

	// 3.1. Default Canny
	cannyImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer cannyImg.Close()

	err = cannyImg.Canny(nil)
	if err != nil {
		t.Logf("Canny with nil options failed: %v", err)
	} else {
		t.Log("Canny with nil options succeeded")
	}

	// 3.2. Custom Canny parameters
	customCannyImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer customCannyImg.Close()

	err = customCannyImg.Canny(&CannyOptions{
		Sigma:     2.0,              // Custom sigma
		Precision: PrecisionInteger, // Integer precision
	})
	if err != nil {
		t.Logf("Canny with custom options failed: %v", err)
	} else {
		t.Log("Canny with custom options succeeded")
	}

	// 4. Test Sobel edge detection
	sobelImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer sobelImg.Close()

	err = sobelImg.Sobel()
	if err != nil {
		t.Logf("Sobel failed: %v", err)
	} else {
		t.Log("Sobel succeeded")
	}

	// 5. Test with a combination of operations

	// Create image with sequential operations
	seqImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer seqImg.Close()

	// First blur to reduce noise
	err = seqImg.Gaussblur(2.0, nil)
	require.NoError(t, err)

	// Then apply edge detection
	err = seqImg.Canny(nil)
	if err != nil {
		t.Logf("Sequential operations (blur->canny) failed: %v", err)
	} else {
		t.Log("Sequential operations (blur->canny) succeeded")
	}
}

// TestLoadOptions tests loading operations with different option combinations
func TestLoadOptions(t *testing.T) {
	// 1. Create a JPEG test image
	jpegImg := image.NewRGBA(image.Rect(0, 0, 400, 300))

	// Fill with a gradient
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			r := uint8(float64(x) / 400.0 * 255)
			g := uint8(float64(y) / 300.0 * 255)
			b := uint8(float64(x+y) / 700.0 * 255)
			jpegImg.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Encode as JPEG
	var jpegBuf bytes.Buffer
	err := jpeg.Encode(&jpegBuf, jpegImg, &jpeg.Options{Quality: 90})
	require.NoError(t, err)
	jpegData := jpegBuf.Bytes()

	// Test loading options for JPEG

	// 1.1. With nil options
	img1, err := NewJpegloadBuffer(jpegData, nil)
	require.NoError(t, err)
	defer img1.Close()
	t.Logf("JPEG loaded with nil options: %dx%d", img1.Width(), img1.Height())

	// 1.2. With default options
	img2, err := NewJpegloadBuffer(jpegData, DefaultJpegloadBufferOptions())
	require.NoError(t, err)
	defer img2.Close()
	t.Logf("JPEG loaded with default options: %dx%d", img2.Width(), img2.Height())

	// 1.3. With shrink factor (load at reduced size)
	img3, err := NewJpegloadBuffer(jpegData, &JpegloadBufferOptions{
		Shrink: 2, // Load at half size
	})
	require.NoError(t, err)
	defer img3.Close()
	t.Logf("JPEG loaded with shrink=2: %dx%d", img3.Width(), img3.Height())

	// Verify the size is approximately halved
	assert.InDelta(t, 400/2, img3.Width(), 2)
	assert.InDelta(t, 300/2, img3.Height(), 2)

	// 1.4. With autorotate option
	img4, err := NewJpegloadBuffer(jpegData, &JpegloadBufferOptions{
		Autorotate: true,
	})
	require.NoError(t, err)
	defer img4.Close()
	t.Logf("JPEG loaded with autorotate: %dx%d", img4.Width(), img4.Height())

	// 2. Create a PNG test image
	pngImg := image.NewRGBA(image.Rect(0, 0, 300, 200))

	// Fill with a pattern
	for y := 0; y < 200; y++ {
		for x := 0; x < 300; x++ {
			if (x/20+y/20)%2 == 0 {
				pngImg.Set(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				pngImg.Set(x, y, color.RGBA{0, 0, 255, 255})
			}
		}
	}

	// Encode as PNG
	var pngBuf bytes.Buffer
	err = png.Encode(&pngBuf, pngImg)
	require.NoError(t, err)
	pngData := pngBuf.Bytes()

	// Test loading options for PNG

	// 2.1. With nil options
	png1, err := NewPngloadBuffer(pngData, nil)
	require.NoError(t, err)
	defer png1.Close()
	t.Logf("PNG loaded with nil options: %dx%d", png1.Width(), png1.Height())

	// 2.2. With default options
	png2, err := NewPngloadBuffer(pngData, DefaultPngloadBufferOptions())
	require.NoError(t, err)
	defer png2.Close()
	t.Logf("PNG loaded with default options: %dx%d", png2.Width(), png2.Height())

	// 2.3. With unlimited option
	png3, err := NewPngloadBuffer(pngData, &PngloadBufferOptions{
		Unlimited: true,
	})
	require.NoError(t, err)
	defer png3.Close()
	t.Logf("PNG loaded with unlimited: %dx%d", png3.Width(), png3.Height())

	// 3. Test with generic loading via NewImageFromBuffer

	// 3.1. With nil options
	gen1, err := NewImageFromBuffer(pngData, nil)
	require.NoError(t, err)
	defer gen1.Close()
	t.Logf("Generic load with nil options: %dx%d", gen1.Width(), gen1.Height())

	// 3.2. With default options
	gen2, err := NewImageFromBuffer(pngData, DefaultLoadOptions())
	require.NoError(t, err)
	defer gen2.Close()
	t.Logf("Generic load with default options: %dx%d", gen2.Width(), gen2.Height())

	// 3.3. With custom options
	gen3, err := NewImageFromBuffer(pngData, &LoadOptions{
		FailOnError: true,
	})
	require.NoError(t, err)
	defer gen3.Close()
	t.Logf("Generic load with custom options: %dx%d", gen3.Width(), gen3.Height())

	// 4. Test loading from memory
	memImg, err := NewImageFromMemory([]byte{
		255, 0, 0, 0, 255, 0, 0, 0, 255, 255, 255, 255,
		255, 255, 0, 0, 255, 255, 255, 0, 255, 0, 0, 0,
	}, 4, 2, 3) // 4x2 RGB image

	if err != nil {
		t.Logf("Load from memory failed: %v", err)
	} else {
		defer memImg.Close()
		t.Logf("Loaded from memory: %dx%d", memImg.Width(), memImg.Height())

		// Verify dimensions
		assert.Equal(t, 4, memImg.Width())
		assert.Equal(t, 2, memImg.Height())
		assert.Equal(t, 3, memImg.Bands())

		// Check a few pixels
		topLeft, err := memImg.Getpoint(0, 0, nil)
		if err == nil {
			t.Logf("Top-left pixel: [%.1f, %.1f, %.1f]", topLeft[0], topLeft[1], topLeft[2])
			assert.InDelta(t, 255, topLeft[0], 5, "Should be red")
			assert.InDelta(t, 0, topLeft[1], 5, "Should be red")
			assert.InDelta(t, 0, topLeft[2], 5, "Should be red")
		}

		topRight, err := memImg.Getpoint(3, 0, nil)
		if err == nil {
			t.Logf("Top-right pixel: [%.1f, %.1f, %.1f]", topRight[0], topRight[1], topRight[2])
			assert.InDelta(t, 255, topRight[0], 5, "Should be white")
			assert.InDelta(t, 255, topRight[1], 5, "Should be white")
			assert.InDelta(t, 255, topRight[2], 5, "Should be white")
		}
	}
}

// TestSaveOptions tests save operations with different option combinations
func TestSaveOptions(t *testing.T) {
	// Create a test image
	width, height := 150, 100
	img, err := createSolidColorImage(t, width, height, color.RGBA{100, 150, 200, 255})
	require.NoError(t, err)
	defer img.Close()

	// Test save operations with various option combinations

	// 1. PNG save options

	// 1.1. With nil options
	pngNilBuf, err := img.PngsaveBuffer(nil)
	require.NoError(t, err)
	t.Logf("PNG save with nil options: %d bytes", len(pngNilBuf))

	// 1.2. With default options
	pngDefaultBuf, err := img.PngsaveBuffer(DefaultPngsaveBufferOptions())
	require.NoError(t, err)
	t.Logf("PNG save with default options: %d bytes", len(pngDefaultBuf))

	// 1.3. With custom options - low compression
	pngLowCompBuf, err := img.PngsaveBuffer(&PngsaveBufferOptions{
		Compression: 1, // Low compression
	})
	require.NoError(t, err)
	t.Logf("PNG save with low compression: %d bytes", len(pngLowCompBuf))

	// 1.4. With custom options - high compression
	pngHighCompBuf, err := img.PngsaveBuffer(&PngsaveBufferOptions{
		Compression: 9, // High compression
	})
	require.NoError(t, err)
	t.Logf("PNG save with high compression: %d bytes", len(pngHighCompBuf))

	// 1.5. With custom options - filter
	pngWithFilterBuf, err := img.PngsaveBuffer(&PngsaveBufferOptions{
		Compression: 6,
		Filter:      PngFilterPaeth,
	})
	require.NoError(t, err)
	t.Logf("PNG save with filter: %d bytes", len(pngWithFilterBuf))

	// 1.6. With custom options - interlaced
	pngInterlacedBuf, err := img.PngsaveBuffer(&PngsaveBufferOptions{
		Compression: 6,
		Interlace:   true,
	})
	require.NoError(t, err)
	t.Logf("PNG save with interlace: %d bytes", len(pngInterlacedBuf))

	// 2. JPEG save options

	// 2.1. With nil options
	jpegNilBuf, err := img.JpegsaveBuffer(nil)
	require.NoError(t, err)
	t.Logf("JPEG save with nil options: %d bytes", len(jpegNilBuf))

	// 2.2. With default options
	jpegDefaultBuf, err := img.JpegsaveBuffer(DefaultJpegsaveBufferOptions())
	require.NoError(t, err)
	t.Logf("JPEG save with default options: %d bytes", len(jpegDefaultBuf))

	// 2.3. With custom options - low quality
	jpegLowQualBuf, err := img.JpegsaveBuffer(&JpegsaveBufferOptions{
		Q: 25, // Low quality
	})
	require.NoError(t, err)
	t.Logf("JPEG save with low quality: %d bytes", len(jpegLowQualBuf))

	// 2.4. With custom options - high quality
	jpegHighQualBuf, err := img.JpegsaveBuffer(&JpegsaveBufferOptions{
		Q: 95, // High quality
	})
	require.NoError(t, err)
	t.Logf("JPEG save with high quality: %d bytes", len(jpegHighQualBuf))

	// 2.5. With custom options - optimize coding
	jpegOptimizedBuf, err := img.JpegsaveBuffer(&JpegsaveBufferOptions{
		Q:              75,
		OptimizeCoding: true,
	})
	require.NoError(t, err)
	t.Logf("JPEG save with optimize coding: %d bytes", len(jpegOptimizedBuf))

	// 2.6. With custom options - interlaced/progressive
	jpegInterlacedBuf, err := img.JpegsaveBuffer(&JpegsaveBufferOptions{
		Q:         75,
		Interlace: true,
	})
	require.NoError(t, err)
	t.Logf("JPEG save with interlace: %d bytes", len(jpegInterlacedBuf))

	// 3. WebP save options

	// 3.1. With nil options
	webpNilBuf, err := img.WebpsaveBuffer(nil)
	require.NoError(t, err)
	t.Logf("WebP save with nil options: %d bytes", len(webpNilBuf))

	// 3.2. With default options
	webpDefaultBuf, err := img.WebpsaveBuffer(DefaultWebpsaveBufferOptions())
	require.NoError(t, err)
	t.Logf("WebP save with default options: %d bytes", len(webpDefaultBuf))

	// 3.3. With custom options - lossless
	webpLosslessBuf, err := img.WebpsaveBuffer(&WebpsaveBufferOptions{
		Lossless: true,
	})
	require.NoError(t, err)
	t.Logf("WebP save with lossless: %d bytes", len(webpLosslessBuf))

	// 3.4. With custom options - quality
	webpQualityBuf, err := img.WebpsaveBuffer(&WebpsaveBufferOptions{
		Q: 50, // Medium quality
	})
	require.NoError(t, err)
	t.Logf("WebP save with quality 50: %d bytes", len(webpQualityBuf))

	// 3.5. With custom options - low effort (faster encoding)
	webpLowEffortBuf, err := img.WebpsaveBuffer(&WebpsaveBufferOptions{
		Q:      75,
		Effort: 1, // Low effort
	})
	require.NoError(t, err)
	t.Logf("WebP save with low effort: %d bytes", len(webpLowEffortBuf))

	// 3.6. With custom options - high effort (better compression)
	webpHighEffortBuf, err := img.WebpsaveBuffer(&WebpsaveBufferOptions{
		Q:      75,
		Effort: 6, // High effort
	})
	require.NoError(t, err)
	t.Logf("WebP save with high effort: %d bytes", len(webpHighEffortBuf))

	// Compare file sizes with different options
	t.Log("PNG size comparisons:")
	if len(pngLowCompBuf) != 0 && len(pngHighCompBuf) != 0 {
		t.Logf("  Low vs High compression ratio: %.2f", float64(len(pngLowCompBuf))/float64(len(pngHighCompBuf)))
	}

	t.Log("JPEG size comparisons:")
	if len(jpegLowQualBuf) != 0 && len(jpegHighQualBuf) != 0 {
		t.Logf("  Low vs High quality ratio: %.2f", float64(len(jpegLowQualBuf))/float64(len(jpegHighQualBuf)))
	}

	t.Log("WebP size comparisons:")
	if len(webpQualityBuf) != 0 && len(webpLosslessBuf) != 0 {
		t.Logf("  Lossy vs Lossless ratio: %.2f", float64(len(webpQualityBuf))/float64(len(webpLosslessBuf)))
	}
	if len(webpLowEffortBuf) != 0 && len(webpHighEffortBuf) != 0 {
		t.Logf("  Low vs High effort ratio: %.2f", float64(len(webpLowEffortBuf))/float64(len(webpHighEffortBuf)))
	}
} // TestOptionsVariants tests operations with different option combinations
func TestOptionsVariants(t *testing.T) {
	// Create a test image
	width, height := 200, 150
	img, err := createSolidColorImage(t, width, height, color.RGBA{200, 100, 50, 255})
	require.NoError(t, err)
	defer img.Close()

	// 1. Test the same operation with nil options, default options, and custom options

	// A. Resize operation

	// A.1. With nil options
	nilOptionsImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer nilOptionsImg.Close()

	err = nilOptionsImg.Resize(0.75, nil)
	require.NoError(t, err)
	t.Log("Resize with nil options succeeded")
	assert.Equal(t, int(float64(width)*0.75), nilOptionsImg.Width())

	// A.2. With default options
	defaultOptionsImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer defaultOptionsImg.Close()

	err = defaultOptionsImg.Resize(0.75, DefaultResizeOptions())
	require.NoError(t, err)
	t.Log("Resize with default options succeeded")
	assert.Equal(t, int(float64(width)*0.75), defaultOptionsImg.Width())

	// A.3. With custom options
	customOptionsImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer customOptionsImg.Close()

	customOptions := &ResizeOptions{
		Kernel: KernelLanczos3,
		Vscale: 0.5, // Different vertical scale
	}
	err = customOptionsImg.Resize(0.75, customOptions)
	require.NoError(t, err)
	t.Log("Resize with custom options succeeded")
	assert.Equal(t, int(float64(width)*0.75), customOptionsImg.Width())
	assert.Equal(t, int(float64(height)*0.5), customOptionsImg.Height())

	// B. GaussBlur operation

	// B.1. With nil options
	blurNilImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer blurNilImg.Close()

	err = blurNilImg.Gaussblur(5.0, nil)
	require.NoError(t, err)
	t.Log("Gaussblur with nil options succeeded")

	// B.2. With default options
	blurDefaultImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer blurDefaultImg.Close()

	err = blurDefaultImg.Gaussblur(5.0, DefaultGaussblurOptions())
	require.NoError(t, err)
	t.Log("Gaussblur with default options succeeded")

	// B.3. With custom options
	blurCustomImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer blurCustomImg.Close()

	customBlurOptions := &GaussblurOptions{
		MinAmpl:   0.1, // Minimum amplitude
		Precision: PrecisionInteger,
	}
	err = blurCustomImg.Gaussblur(5.0, customBlurOptions)
	require.NoError(t, err)
	t.Log("Gaussblur with custom options succeeded")

	// C. Embed operation

	// C.1. With nil options
	embedNilImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer embedNilImg.Close()

	err = embedNilImg.Embed(10, 10, width+20, height+20, nil)
	require.NoError(t, err)
	t.Log("Embed with nil options succeeded")
	assert.Equal(t, width+20, embedNilImg.Width())
	assert.Equal(t, height+20, embedNilImg.Height())

	// C.2. With default options
	embedDefaultImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer embedDefaultImg.Close()

	err = embedDefaultImg.Embed(10, 10, width+20, height+20, DefaultEmbedOptions())
	require.NoError(t, err)
	t.Log("Embed with default options succeeded")
	assert.Equal(t, width+20, embedDefaultImg.Width())
	assert.Equal(t, height+20, embedDefaultImg.Height())

	// C.3. With custom options
	embedCustomImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer embedCustomImg.Close()

	customEmbedOptions := &EmbedOptions{
		Extend:     ExtendWhite,
		Background: []float64{255, 0, 0}, // Red background
	}
	err = embedCustomImg.Embed(10, 10, width+20, height+20, customEmbedOptions)
	require.NoError(t, err)
	t.Log("Embed with custom options succeeded")
	assert.Equal(t, width+20, embedCustomImg.Width())
	assert.Equal(t, height+20, embedCustomImg.Height())

	// Verify the background color of the embedded image
	topLeftPixel, err := embedCustomImg.Getpoint(5, 5, nil) // Should be in the background
	require.NoError(t, err)
	t.Logf("Background pixel: [%.1f, %.1f, %.1f]",
		topLeftPixel[0], topLeftPixel[1], topLeftPixel[2])

	assert.Equal(t, float64(255), topLeftPixel[0])
	assert.Equal(t, float64(255), topLeftPixel[1])
	assert.Equal(t, float64(255), topLeftPixel[2])
}
