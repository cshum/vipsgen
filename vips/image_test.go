package vips

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	Startup(&Config{
		ReportLeaks: true,
	})

	// Get initial memory stats
	var initialStats MemoryStats
	ReadVipsMemStats(&initialStats)

	// Run the tests
	code := m.Run()

	// Get final memory stats
	var finalStats MemoryStats
	ReadVipsMemStats(&finalStats)

	// Check for memory leaks
	memLeaked := finalStats.Mem > initialStats.Mem
	filesLeaked := finalStats.Files > initialStats.Files
	allocsLeaked := finalStats.Allocs > initialStats.Allocs

	if memLeaked || filesLeaked || allocsLeaked {
		fmt.Printf("MEMORY LEAK DETECTED!\n")
		fmt.Printf("Initial stats - Mem: %d, Files: %d, Allocs: %d\n",
			initialStats.Mem, initialStats.Files, initialStats.Allocs)
		fmt.Printf("Final stats   - Mem: %d, Files: %d, Allocs: %d\n",
			finalStats.Mem, finalStats.Files, finalStats.Allocs)
		fmt.Printf("Differences   - Mem: %+d, Files: %+d, Allocs: %+d\n",
			finalStats.Mem-initialStats.Mem,
			finalStats.Files-initialStats.Files,
			finalStats.Allocs-initialStats.Allocs)

		Shutdown()
		os.Exit(1) // Exit with error code
	}

	fmt.Printf("No memory leaks detected.\n")
	fmt.Printf("Final stats - Mem: %d, Files: %d, Allocs: %d\n",
		finalStats.Mem, finalStats.Files, finalStats.Allocs)

	Shutdown()
	os.Exit(code) // Exit with the test result code
}

// createTestPngBuffer creates a test PNG image with a pattern
func createTestPngBuffer(t *testing.T, width, height int) []byte {
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

// createTestJpegBuffer creates a test JPEG image with a pattern
func createTestJpegBuffer(t *testing.T, width, height int) []byte {
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

func createTestGradientImage(t *testing.T, width, height int) (*Image, error) {
	// Create a gradient image for testing return values
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create gradient from black to white
			value := uint8(float64(x+y) / float64(width+height-2) * 255)
			img.Set(x, y, color.RGBA{value, value, value, 255})
		}
	}

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	return NewImageFromBuffer(buf.Bytes(), nil)
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
func TestImageType_MimeType(t *testing.T) {
	tests := []struct {
		imageType    ImageType
		expectedMime string
		expectedOk   bool
	}{
		{
			imageType:    ImageTypeJpeg,
			expectedMime: "image/jpeg",
			expectedOk:   true,
		},
		{
			imageType:    ImageTypeAvif,
			expectedMime: "image/avif",
			expectedOk:   true,
		},
		{
			imageType:    ImageTypeVips,
			expectedMime: "image/vnd.libvips",
			expectedOk:   true,
		},
		{
			imageType:    ImageTypeCsv,
			expectedMime: "text/csv",
			expectedOk:   true,
		},
		{
			imageType:    ImageTypeUnknown,
			expectedMime: "",
			expectedOk:   false,
		},
		{
			imageType:    ImageTypeMagick,
			expectedMime: "",
			expectedOk:   false,
		},
		{
			imageType:    ImageType("invalid"),
			expectedMime: "",
			expectedOk:   false,
		},
		{
			imageType:    ImageType(""),
			expectedMime: "",
			expectedOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.imageType), func(t *testing.T) {
			mime, ok := tt.imageType.MimeType()
			assert.Equal(t, tt.expectedOk, ok, "ImageType.MimeType() ok = %v, expected %v", ok, tt.expectedOk)
			assert.Equal(t, tt.expectedMime, mime, "ImageType.MimeType() mime = %q, expected %q", mime, tt.expectedMime)
			if tt.expectedOk {
				t.Logf("✓ %s -> %s", tt.imageType, mime)
			} else {
				t.Logf("✗ %s -> (not found)", tt.imageType)
			}
		})
	}
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
	pngData := createTestPngBuffer(t, 200, 150)
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
	pngData := createTestPngBuffer(t, 150, 100)

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
	pngData := createTestPngBuffer(t, 50, 50)

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
	pngData := createTestPngBuffer(t, width, height)

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
	// Create a test gradient image
	width, height := 100, 80
	img, err := NewImageFromBuffer(createTestPngBuffer(t, width, height), DefaultLoadOptions())
	require.NoError(t, err)
	defer img.Close()

	// Test PNG saving with default options
	pngBuf, err := img.PngsaveBuffer(nil)
	require.NoError(t, err)
	t.Logf("PNG save succeeded: %d bytes", len(pngBuf))
	assert.NotEmpty(t, pngBuf)

	// Test PNG saving with options
	pngBuf2, err := img.PngsaveBuffer(&PngsaveBufferOptions{
		Compression: 6,
		Filter:      PngFilterAll,
	})
	require.NoError(t, err)
	t.Logf("PNG save with options succeeded: %d bytes", len(pngBuf2))
	assert.NotEmpty(t, pngBuf2)

	// Test JPEG saving with default options
	jpegBuf, err := img.JpegsaveBuffer(nil)
	require.NoError(t, err)
	t.Logf("JPEG save succeeded: %d bytes", len(jpegBuf))
	assert.NotEmpty(t, jpegBuf)

	// Test JPEG saving with basic options
	jpegBuf2, err := img.JpegsaveBuffer(&JpegsaveBufferOptions{
		Q: 85,
	})
	require.NoError(t, err)
	t.Logf("JPEG save with options succeeded: %d bytes", len(jpegBuf2))
	assert.NotEmpty(t, jpegBuf2)

	// Test WebP saving with default options
	webpBuf, err := img.WebpsaveBuffer(nil)
	require.NoError(t, err)
	t.Logf("WebP save succeeded: %d bytes", len(webpBuf))
	assert.NotEmpty(t, webpBuf)

	// Test WebP saving with options
	webpBuf2, err := img.WebpsaveBuffer(&WebpsaveBufferOptions{
		Q:        80,
		Lossless: true,
	})
	require.NoError(t, err)
	t.Logf("WebP save with options succeeded: %d bytes", len(webpBuf2))
	assert.NotEmpty(t, webpBuf2)
}

// TestImageOperations tests various image operations like blur, sharpen, etc.
func TestImageOperations(t *testing.T) {
	// Create a test image to work with
	width, height := 200, 150
	img, err := NewImageFromBuffer(createTestPngBuffer(t, width, height), nil)
	require.NoError(t, err)
	defer img.Close()

	// Make a copy for comparing operations
	imgCopy, err := img.Copy(nil)
	require.NoError(t, err)
	defer imgCopy.Close()

	// Test a series of operations

	// 1. Gaussian blur
	err = img.Gaussblur(5.0, nil)
	require.NoError(t, err)

	// 2. Sharpen
	err = img.Sharpen(nil)
	require.NoError(t, err)

	// 3. Invert colors
	err = img.Invert()
	require.NoError(t, err)

	// 4. Test resize and position with embed
	err = imgCopy.Embed(10, 10, width+20, height+20, &EmbedOptions{
		Extend: ExtendBlack,
	})
	require.NoError(t, err)
	t.Logf("Embed succeeded: new size %dx%d", imgCopy.Width(), imgCopy.Height())
	assert.Equal(t, width+20, imgCopy.Width())
	assert.Equal(t, height+20, imgCopy.Height())
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
	require.NoError(t, err)

	img3, err := createBlackImage(100, 100)
	require.NoError(t, err)
	defer img3.Close()

	// Try to composite array of images (if supported)
	images := []*Image{img1, img2, img3}

	composite, err := NewComposite(images, []BlendMode{BlendModeOver, BlendModeAdd}, &CompositeOptions{X: []int{10, 20}, Y: []int{20, 10}})
	require.NoError(t, err)
	defer composite.Close()
}

// TestLabel tests the label functionality
func TestLabel(t *testing.T) {
	// Create a test image
	img, err := NewPngloadBuffer(createTestPngBuffer(t, 100, 100), nil)
	require.NoError(t, err)
	defer img.Close()
	t.Logf("Bands %d", img.Bands())
	err = img.Addalpha()
	require.NoError(t, err)

	// Add text to the image
	err = img.Label("Hello, libvips!", 50, 50, &LabelOptions{
		Font:    "sans",
		Size:    24,
		Color:   []float64{255, 0, 0},
		Opacity: 1.0,
	})
	require.NoError(t, err)
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
	jpegData := createTestJpegBuffer(t, 120, 80)

	// Load JPEG
	img, err := NewJpegloadBuffer(jpegData, nil)
	require.NoError(t, err)
	defer img.Close()

	err = img.SetOrientation(2)
	require.NoError(t, err)
	orientation := img.Orientation()
	assert.Equal(t, 2, orientation)

	// Try to extract EXIF data
	exifData := img.Exif()
	t.Logf("EXIF data: %v", exifData)

	// Test removing EXIF data
	err = img.RemoveExif()
	require.NoError(t, err)

	// Check EXIF data is gone
	exifDataAfter := img.Exif()
	assert.Empty(t, exifDataAfter, "EXIF data should be empty after removal")
}

// TestMultiPageOperations tests operations on multi-page images
func TestMultiPageOperations(t *testing.T) {
	// Create a simple test image
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Get page count
	pageCount := img.Pages()
	assert.Equal(t, 1, pageCount, "Image should have 1 page")

	// Get page height
	pageHeight := img.PageHeight()
	assert.Equal(t, 100, pageHeight, "Image should have 100 page height")

	// Try to get/set page height
	err = img.SetPageHeight(50)
	require.NoError(t, err)
	err = img.SetPages(2)
	require.NoError(t, err)
	assert.Equal(t, 50, img.PageHeight())
	assert.Equal(t, 2, img.Pages())
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
		require.NoError(t, err)
		t.Logf("  - %s save succeeded: %d bytes", test.name, len(buf))
		assert.NotEmpty(t, buf)
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
	require.NotEmpty(t, centerPixel, "Getpoint should return pixel values")
	assert.InDelta(t, 255, centerPixel[0], 1, "Center should initially be white")

	// 1. Draw a red rectangle (x=50, y=50, width=100, height=100)
	redColor := []float64{255, 0, 0}
	err = img.DrawRect(redColor, 50, 50, 100, 100, &DrawRectOptions{
		Fill: true,
	})
	require.NoError(t, err)
	t.Log("DrawRect successful")

	// Validate pixel inside the rectangle
	rectPixel, err := img.Getpoint(75, 75, nil)
	require.NoError(t, err)
	require.NotEmpty(t, rectPixel, "Getpoint should return pixel values")
	assert.InDelta(t, redColor[0], rectPixel[0], 5, "Rectangle should be red (R)")
	assert.InDelta(t, redColor[1], rectPixel[1], 5, "Rectangle should be red (G)")
	assert.InDelta(t, redColor[2], rectPixel[2], 5, "Rectangle should be red (B)")

	// Validate pixel outside the rectangle
	outsidePixel, err := img.Getpoint(25, 25, nil)
	require.NoError(t, err)
	require.NotEmpty(t, outsidePixel, "Getpoint should return pixel values")
	assert.InDelta(t, 255, outsidePixel[0], 5, "Outside should still be white")

	// 2. Draw a blue circle (center=200,150, radius=50)
	blueColor := []float64{0, 0, 255}
	err = img.DrawCircle(blueColor, 200, 150, 50, &DrawCircleOptions{
		Fill: true,
	})
	require.NoError(t, err)
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

	// 3. Draw a green line from (50,200) to (250,250)
	greenColor := []float64{0, 255, 0}
	err = img.DrawLine(greenColor, 50, 200, 250, 250)
	require.NoError(t, err)
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

	// Check if the image still has the expected dimensions
	assert.Equal(t, width, img.Width())
	assert.Equal(t, height, img.Height())
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
	require.NoError(t, err)
	t.Log("Successfully blended images")

	// Test saving the resulting image
	buf, err := redImg.PngsaveBuffer(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, buf)

	// Test other blend modes if supported
	blendModes := []BlendMode{
		BlendModeMultiply,
		BlendModeScreen,
		BlendModeDarken,
		BlendModeLighten,
	}

	for _, mode := range blendModes {
		baseImg, err := createSolidColorImage(t, width, height, color.RGBA{200, 200, 200, 255})
		require.NoError(t, err)

		overlayImg, err := createSolidColorImage(t, width/2, height/2, color.RGBA{100, 100, 100, 255})
		require.NoError(t, err)

		err = baseImg.Composite2(overlayImg, mode, &Composite2Options{
			X: width / 4,
			Y: height / 4,
		})
		require.NoError(t, err)

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
		require.NoError(t, err)

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
		require.NoError(t, err)
		t.Logf("%s filter successful", filter.name)

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
	require.NotEmpty(t, midPixel, "Getpoint should return pixel values")
	assert.InDelta(t, 255, midPixel[0], 5, "Middle pixel should be red")
	assert.InDelta(t, 0, midPixel[1], 5, "Middle pixel should be red")
	assert.InDelta(t, 0, midPixel[2], 5, "Middle pixel should be red")

	// Test Rot - 90 degree rotation
	rotImg, err := vipsImg.Copy(nil)
	require.NoError(t, err)
	defer rotImg.Close()

	err = rotImg.Rot(AngleD90)
	require.NoError(t, err)
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

	assert.True(t, foundRed, "Could not find red pixel in vertical scan after rotation")

	// Test Rot45 if available (requires odd-sized square image, which we have)
	rot45Img, err := vipsImg.Copy(nil)
	require.NoError(t, err)
	defer rot45Img.Close()

	err = rot45Img.Rot45(&Rot45Options{Angle: Angle45D45})
	require.NoError(t, err)
	t.Log("Rot45(Angle45D45) succeeded")
	t.Logf("After Rot45: %dx%d", rot45Img.Width(), rot45Img.Height())

	// After 45-degree rotation, the line should be diagonal
	// Just validate some basic properties since exact pixel location is complex
	centerPixel, err = rot45Img.Getpoint(rot45Img.Width()/2, rot45Img.Height()/2, nil)
	require.NoError(t, err)
	t.Logf("Center pixel after Rot45: [%.1f, %.1f, %.1f]",
		centerPixel[0], centerPixel[1], centerPixel[2])
}

// TestRot45Requirements tests the specific requirements for the rot45 operation
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
	require.NoError(t, err)
	t.Log("Rot45 succeeded with odd-sized square image as expected")

	// Verify dimensions of rotated image
	t.Logf("After rotation: %dx%d", oddImg.Width(), oddImg.Height())

	// Check pixels after rotation
	centerX, centerY := oddImg.Width()/2, oddImg.Height()/2
	centerPixel, err := oddImg.Getpoint(centerX, centerY, nil)
	require.NoError(t, err)
	require.NotEmpty(t, centerPixel, "Getpoint should return pixel values")
	t.Logf("Center pixel after rotation: [%.1f, %.1f, %.1f]",
		centerPixel[0], centerPixel[1], centerPixel[2])

	// 4. Try different rotation angles if available
	// Make another odd-sized square image for testing other angles
	oddImg2, err := createSolidColorImage(t, oddWidth, oddHeight, color.RGBA{0, 0, 255, 255})
	require.NoError(t, err)
	defer oddImg2.Close()

	// Try 90-degree rotation (D90)
	err = oddImg2.Rot45(&Rot45Options{Angle: Angle45D90})
	require.NoError(t, err, "Rot45 with Angle45D90 succeeded")
}

// TestImageStats tests statistical functions on images
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
	require.NoError(t, err)
	t.Logf("Image average: %.2f", avg)
	// For a linear gradient 0-255, average should be close to 127.5
	assert.InDelta(t, 127.5, avg, 10, "Average should be close to 127.5")

	// Test Min operation
	minVal, err := vipsImg.Min(nil)
	require.NoError(t, err)
	t.Logf("Image minimum: %.2f", minVal)
	assert.InDelta(t, 0, minVal, 5, "Minimum should be close to 0")

	// Test Max operation
	maxVal, err := vipsImg.Max(nil)
	require.NoError(t, err)
	t.Logf("Image maximum: %.2f", maxVal)
	assert.InDelta(t, 255, maxVal, 5, "Maximum should be close to 255")

	// Test Deviate (standard deviation) operation
	dev, err := vipsImg.Deviate()
	require.NoError(t, err)
	t.Logf("Image standard deviation: %.2f", dev)
	// Standard deviation for uniform gradient should be positive
	assert.Greater(t, dev, 0.0, "Standard deviation should be positive")

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
		require.NoError(t, err)
		require.NotEmpty(t, pixelValues, "Getpoint should return pixel values")

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

	// 1. Draw a red rectangle (50,50 to 150,150)
	err = img.DrawRect([]float64{255, 0, 0}, 50, 50, 100, 100, &DrawRectOptions{
		Fill: true,
	})
	require.NoError(t, err)
	t.Log("DrawRect successful")

	// Verify red rectangle color
	redPixel, err := img.Getpoint(100, 100, nil) // Center of the red rectangle
	require.NoError(t, err)
	require.NotEmpty(t, redPixel, "Getpoint should return pixel values")
	assert.InDelta(t, 255.0, redPixel[0], 1.0, "Red channel should be ~255")
	assert.InDelta(t, 0.0, redPixel[1], 1.0, "Green channel should be ~0")
	assert.InDelta(t, 0.0, redPixel[2], 1.0, "Blue channel should be ~0")
	t.Logf("Red rectangle pixel at (100,100): R=%.1f, G=%.1f, B=%.1f", redPixel[0], redPixel[1], redPixel[2])

	// Verify white background is still white
	whitePixel, err := img.Getpoint(25, 25, nil) // Outside the red rectangle
	require.NoError(t, err)
	assert.InDelta(t, 255.0, whitePixel[0], 1.0, "Background red channel should be ~255")
	assert.InDelta(t, 255.0, whitePixel[1], 1.0, "Background green channel should be ~255")
	assert.InDelta(t, 255.0, whitePixel[2], 1.0, "Background blue channel should be ~255")

	// 2. Draw a blue circle (center at 200,150, radius 50)
	err = img.DrawCircle([]float64{0, 0, 255}, 200, 150, 50, &DrawCircleOptions{
		Fill: true,
	})
	require.NoError(t, err)
	t.Log("DrawCircle successful")

	// Verify blue circle color at center
	bluePixel, err := img.Getpoint(200, 150, nil) // Center of the blue circle
	require.NoError(t, err)
	assert.InDelta(t, 0.0, bluePixel[0], 1.0, "Red channel should be ~0")
	assert.InDelta(t, 0.0, bluePixel[1], 1.0, "Green channel should be ~0")
	assert.InDelta(t, 255.0, bluePixel[2], 1.0, "Blue channel should be ~255")
	t.Logf("Blue circle pixel at (200,150): R=%.1f, G=%.1f, B=%.1f", bluePixel[0], bluePixel[1], bluePixel[2])

	// Verify a point slightly inside the circle edge
	circleEdgePixel, err := img.Getpoint(225, 150, nil) // 25 pixels right of center (within radius 50)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, circleEdgePixel[0], 1.0, "Circle edge red channel should be ~0")
	assert.InDelta(t, 0.0, circleEdgePixel[1], 1.0, "Circle edge green channel should be ~0")
	assert.InDelta(t, 255.0, circleEdgePixel[2], 1.0, "Circle edge blue channel should be ~255")

	// 3. Draw a green line from (50,200) to (250,250)
	err = img.DrawLine([]float64{0, 255, 0}, 50, 200, 250, 250)
	require.NoError(t, err)
	t.Log("DrawLine successful")

	// Verify green line color at a point along the line
	// The line goes from (50,200) to (250,250), so midpoint is approximately (150,225)
	greenPixel, err := img.Getpoint(150, 225, nil)
	require.NoError(t, err)
	assert.InDelta(t, 0.0, greenPixel[0], 1.0, "Line red channel should be ~0")
	assert.InDelta(t, 255.0, greenPixel[1], 1.0, "Line green channel should be ~255")
	assert.InDelta(t, 0.0, greenPixel[2], 1.0, "Line blue channel should be ~0")
	t.Logf("Green line pixel at (150,225): R=%.1f, G=%.1f, B=%.1f", greenPixel[0], greenPixel[1], greenPixel[2])

}

// TestSourceOperations tests operations using Source
func TestSourceOperations(t *testing.T) {
	// Create a test image
	width, height := 100, 100
	data := createTestPngBuffer(t, width, height)

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
	require.NoError(t, err)

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
	require.NoError(t, err)

	return NewImageFromBuffer(buf.Bytes(), nil)
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
		require.NoError(t, err)
		t.Logf("%s conversion successful", conv.name)

		// Try to convert back if possible
		if idx := conv.name + "->back"; idx[0] == 'S' {
			require.NoError(t, convImg.HSV2sRGB())
			t.Logf("Converting back: %v", err == nil)
		} else if idx[0] == 'H' {
			require.NoError(t, convImg.SRGB2HSV())
			t.Logf("Converting back: %v", err == nil)
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
	require.NoError(t, err)

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
		require.NoError(t, err)
		t.Logf("Gaussblur with sigma=%.1f succeeded", sigma)

		// Check center pixel
		center, err := blurImg.Getpoint(width/2, height/2, nil)
		require.NoError(t, err)
		t.Logf("Center pixel with sigma=%.1f: [%.1f, %.1f, %.1f]",
			sigma, center[0], center[1], center[2])

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
	require.NoError(t, err)
	t.Log("Sharpen with nil options succeeded")

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
	require.NoError(t, err)
	t.Log("Sharpen with custom options succeeded")

	// 3. Test Canny edge detection with different options

	// 3.1. Default Canny
	cannyImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer cannyImg.Close()

	err = cannyImg.Canny(nil)
	require.NoError(t, err)
	t.Log("Canny with nil options succeeded")

	// 3.2. Custom Canny parameters
	customCannyImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer customCannyImg.Close()

	err = customCannyImg.Canny(&CannyOptions{
		Sigma:     2.0,              // Custom sigma
		Precision: PrecisionInteger, // Integer precision
	})
	require.NoError(t, err)
	t.Log("Canny with custom options succeeded")

	// 4. Test Sobel edge detection
	sobelImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer sobelImg.Close()

	err = sobelImg.Sobel()
	require.NoError(t, err)
	t.Log("Sobel succeeded")

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
	require.NoError(t, err)
	t.Log("Sequential operations (blur->canny) succeeded")
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
	require.NoError(t, err)
	defer memImg.Close()
	t.Logf("Loaded from memory: %dx%d", memImg.Width(), memImg.Height())

	// Verify dimensions
	assert.Equal(t, 4, memImg.Width())
	assert.Equal(t, 2, memImg.Height())
	assert.Equal(t, 3, memImg.Bands())

	// Check a few pixels
	topLeft, err := memImg.Getpoint(0, 0, nil)
	require.NoError(t, err)
	t.Logf("Top-left pixel: [%.1f, %.1f, %.1f]", topLeft[0], topLeft[1], topLeft[2])
	assert.InDelta(t, 255, topLeft[0], 5, "Should be red")
	assert.InDelta(t, 0, topLeft[1], 5, "Should be red")
	assert.InDelta(t, 0, topLeft[2], 5, "Should be red")

	topRight, err := memImg.Getpoint(3, 0, nil)
	require.NoError(t, err)
	t.Logf("Top-right pixel: [%.1f, %.1f, %.1f]", topRight[0], topRight[1], topRight[2])
	assert.InDelta(t, 255, topRight[0], 5, "Should be white")
	assert.InDelta(t, 255, topRight[1], 5, "Should be white")
	assert.InDelta(t, 255, topRight[2], 5, "Should be white")
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

func TestImage_HasAlpha(t *testing.T) {
	// Test PNG without alpha
	pngData := createTestPngBuffer(t, 100, 100)
	img, err := NewImageFromBuffer(pngData, nil)
	require.NoError(t, err)
	defer img.Close()

	assert.False(t, img.HasAlpha(), "PNG RGB should not have alpha")

	// Test adding alpha channel
	err = img.Addalpha()
	require.NoError(t, err)

	assert.True(t, img.HasAlpha(), "Image should have alpha after adding alpha channel")
}

func TestImage_HasICCProfile(t *testing.T) {
	// Create test image
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Initially should not have ICC profile
	assert.False(t, img.HasICCProfile(), "New image should not have ICC profile")

	// Test getting non-existent profile
	profile, hasProfile := img.GetICCProfile()
	assert.False(t, hasProfile, "Should not have ICC profile")
	assert.Nil(t, profile, "Profile data should be nil")
}

func TestImage_Orientation(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Initially should have no orientation (returns 0)
	orientation := img.Orientation()
	assert.Equal(t, 0, orientation, "New image should have no orientation set")

	// Test setting orientation
	err = img.SetOrientation(6) // 90 degrees clockwise
	require.NoError(t, err)

	orientation = img.Orientation()
	assert.Equal(t, 6, orientation, "Orientation should be set to 6")

	// Test removing orientation
	err = img.RemoveOrientation()
	require.NoError(t, err)

	orientation = img.Orientation()
	assert.Equal(t, 0, orientation, "Orientation should be removed")
}

func TestImage_Pages(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Single image should have 1 page
	pages := img.Pages()
	assert.Equal(t, 1, pages, "Single image should have 1 page")

	// Test setting pages
	err = img.SetPages(5)
	require.NoError(t, err)

	pages = img.Pages()
	assert.Equal(t, 5, pages, "Should have 5 pages after setting")
}

func TestImage_PageHeight(t *testing.T) {
	img, err := createWhiteImage(100, 200)
	require.NoError(t, err)
	defer img.Close()

	// Page height should initially be the full height
	pageHeight := img.PageHeight()
	assert.Equal(t, 200, pageHeight, "Page height should be full image height")

	// Test setting page height
	err = img.SetPageHeight(50)
	require.NoError(t, err)

	pageHeight = img.PageHeight()
	assert.Equal(t, 50, pageHeight, "Page height should be set to 50")
}

func TestImage_GenericMetadata(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test string metadata
	assert.False(t, img.HasField("test-string"))
	img.SetString("test-string", "hello world")
	assert.NoError(t, err)
	value, err := img.GetString("test-string")
	assert.Equal(t, "hello world", value, "String metadata should match")
	assert.True(t, img.HasField("test-string"))

	// Test integer metadata
	assert.False(t, img.HasField("test-int"))
	img.SetInt("test-int", 42)
	assert.NoError(t, err)
	intValue, err := img.GetInt("test-int")
	assert.Equal(t, 42, intValue, "Integer metadata should match")
	assert.True(t, img.HasField("test-int"))

	// Test double metadata
	assert.False(t, img.HasField("test-double"))
	img.SetDouble("test-double", 3.14159)
	assert.NoError(t, err)
	doubleValue, err := img.GetDouble("test-double")
	assert.InDelta(t, 3.14159, doubleValue, 0.00001, "Double metadata should match")
	assert.True(t, img.HasField("test-double"))
}

func TestImage_GetFields(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Set some metadata
	img.SetString("custom-field", "test")
	img.SetInt("custom-number", 123)

	// Get all fields
	fields := img.GetFields()
	assert.NotEmpty(t, fields, "Should have some fields")

	// Check that our custom fields are included
	hasCustomField := false
	hasCustomNumber := false
	for _, field := range fields {
		if field == "custom-field" {
			hasCustomField = true
		}
		if field == "custom-number" {
			hasCustomNumber = true
		}
	}
	assert.True(t, hasCustomField, "Should contain custom-field")
	assert.True(t, hasCustomNumber, "Should contain custom-number")
}

func TestImage_GetAsString(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Set different types of metadata
	img.SetInt("test-int", 42)
	img.SetDouble("test-double", 3.14)
	img.SetString("test-string", "hello")

	// Test getting as string
	intAsString, err := img.GetAsString("test-int")
	assert.NoError(t, err)
	assert.Equal(t, "42", intAsString, "Integer should convert to string")

	doubleAsString, err := img.GetAsString("test-double")
	assert.NoError(t, err)
	assert.Contains(t, doubleAsString, "3.14", "Double should convert to string")

	stringAsString, err := img.GetAsString("test-string")
	assert.NoError(t, err)
	assert.Equal(t, "hello", stringAsString, "String should remain string")
}

func TestImage_ExifData(t *testing.T) {
	// Create image with some EXIF-like metadata
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Set some EXIF-like fields
	img.SetString("exif-Make", "Test Camera")
	img.SetString("exif-Model", "Test Model")
	img.SetInt("exif-Orientation", 1)
	img.SetString("non-exif-field", "should not appear")

	// Get EXIF data
	exifData := img.Exif()

	// Check that only EXIF fields are returned
	assert.Contains(t, exifData, "exif-Make", "Should contain EXIF Make field")
	assert.Contains(t, exifData, "exif-Model", "Should contain EXIF Model field")
	assert.NotContains(t, exifData, "non-exif-field", "Should not contain non-EXIF field")

	// Check values
	assert.Equal(t, "Test Camera", exifData["exif-Make"], "EXIF Make should match")
	assert.Equal(t, "Test Model", exifData["exif-Model"], "EXIF Model should match")
}

func TestImage_IsColorSpaceSupported(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// RGB should be supported
	assert.True(t, img.IsColorSpaceSupported(), "RGB colorspace should be supported")
}

func TestImage_RemoveICCProfile(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test removing non-existent profile (should not error)
	err = img.RemoveICCProfile()
	require.NoError(t, err, "Removing non-existent ICC profile should not error")

	// Verify still no profile
	assert.False(t, img.HasICCProfile(), "Should still not have ICC profile")
}

func TestImage_MetadataWithRealImage(t *testing.T) {
	// Test with actual PNG data
	pngData := createTestPngBuffer(t, 50, 50)
	img, err := NewImageFromBuffer(pngData, nil)
	require.NoError(t, err)
	defer img.Close()

	// Test basic properties
	assert.Equal(t, 50, img.Width(), "Width should be 50")
	assert.Equal(t, 50, img.Height(), "Height should be 50")
	assert.Equal(t, 3, img.Bands(), "PNG RGB should have 3 bands")

	// Test setting and getting orientation
	err = img.SetOrientation(8) // 270 degrees
	require.NoError(t, err)

	orientation := img.Orientation()
	assert.Equal(t, 8, orientation, "Orientation should be 8")

	// Test fields
	fields := img.GetFields()
	assert.NotEmpty(t, fields, "Should have fields")

	// Orientation should be in fields
	hasOrientation := false
	for _, field := range fields {
		if field == "orientation" {
			hasOrientation = true
			break
		}
	}
	assert.True(t, hasOrientation, "Should have orientation field")
}

func TestImage_ErrorHandling(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test getting non-existent metadata (should return zero values, not error)
	nonExistentInt, err := img.GetInt("non-existent-field")
	assert.Error(t, err)
	assert.Equal(t, 0, nonExistentInt, "Non-existent int field should return 0")

	nonExistentString, err := img.GetString("non-existent-field")
	assert.Error(t, err)
	assert.Equal(t, "", nonExistentString, "Non-existent string field should return empty string")

	nonExistentDouble, err := img.GetDouble("non-existent-field")
	assert.Error(t, err)
	assert.Equal(t, 0.0, nonExistentDouble, "Non-existent double field should return 0.0")

	nonExistentBlob, err := img.GetBlob("non-existent-field")
	assert.Error(t, err)
	assert.Empty(t, nonExistentBlob, "Non-existent blob field should return empty or nil")

	// Test getting non-existent arrays (should return nil/error, not crash)
	nonExistentIntArray, err := img.PageDelay()
	assert.Error(t, err)
	assert.Empty(t, nonExistentIntArray, "Non-existent array field should return empty or nil")

	nonExistentDoubleArray, err := img.Background()
	assert.Error(t, err)
	assert.Empty(t, nonExistentDoubleArray, "Non-existent array field should return empty or nil")
}

func TestHasOperation(t *testing.T) {
	// Test operations that should always exist in any libvips installation
	assert.True(t, HasOperation("copy"), "copy operation should always exist")
	assert.True(t, HasOperation("resize"), "resize operation should always exist")
	assert.True(t, HasOperation("embed"), "embed operation should always exist")
	assert.True(t, HasOperation("extract_area"), "extract_area operation should always exist")

	// Test common format operations that should exist in most installations
	assert.True(t, HasOperation("jpegload"), "jpegload should exist in most installations")
	assert.True(t, HasOperation("jpegsave"), "jpegsave should exist in most installations")
	assert.True(t, HasOperation("pngload"), "pngload should exist in most installations")
	assert.True(t, HasOperation("pngsave"), "pngsave should exist in most installations")

	// Test newer format that might not be available
	jxlExists := HasOperation("jxlload")
	avifExists := HasOperation("avifload")
	heifExists := HasOperation("heifload")

	// These are just informational - log what's available
	t.Logf("JPEG XL support: %v", jxlExists)
	t.Logf("AVIF support: %v", avifExists)
	t.Logf("HEIF support: %v", heifExists)

	// Test operations that definitely should not exist
	assert.False(t, HasOperation("nonexistent_operation"), "nonexistent operation should return false")
	assert.False(t, HasOperation("fake_operation_xyz"), "fake operation should return false")
	assert.False(t, HasOperation(""), "empty string should return false")

	// Test with invalid characters that might cause issues
	assert.False(t, HasOperation("invalid-operation-name!"), "operation with invalid chars should return false")
	assert.False(t, HasOperation("operation with spaces"), "operation with spaces should return false")
}

func TestParameterBinding(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test different parameter types are correctly passed through

	// 1. Integer parameters
	err = img.Resize(2.0, &ResizeOptions{
		Kernel: KernelLanczos3, // enum -> int conversion
	})
	require.NoError(t, err)
	assert.Equal(t, 200, img.Width())

	// 2. Float parameters
	blurImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer blurImg.Close()

	err = blurImg.Gaussblur(2.5, &GaussblurOptions{
		MinAmpl: 0.1, // float64 parameter
	})
	require.NoError(t, err)

	// 3. Boolean parameters
	embedImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer embedImg.Close()

	err = embedImg.Embed(10, 10, 220, 220, &EmbedOptions{
		Extend: ExtendBackground,
	})
	require.NoError(t, err)

	// 4. String parameters
	textImg, err := NewText("Hello", &TextOptions{
		Font: "sans 20", // string parameter
		Dpi:  72,
	})
	require.NoError(t, err)
	defer textImg.Close()
	assert.Greater(t, textImg.Width(), 0)

	// 5. Array parameters
	linearImg, err := img.Copy(nil)
	require.NoError(t, err)
	defer linearImg.Close()

	err = linearImg.Linear([]float64{1.2, 1.1, 1.0}, []float64{10, 5, 0}, nil)
	require.NoError(t, err)
}

func TestArrayParameterBinding(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// 1. Float64 arrays
	err = img.Linear([]float64{1.5, 1.2, 1.0}, []float64{10, 20, 30}, nil)
	require.NoError(t, err)

	// 3. Single element arrays
	err = img.Linear([]float64{1.1}, []float64{5}, nil)
	require.NoError(t, err)
}

func TestEnumParameterBinding(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test enum parameters are correctly converted to C values

	// Test different extend modes
	extendModes := []Extend{
		ExtendBlack,
		ExtendCopy,
		ExtendRepeat,
		ExtendMirror,
		ExtendWhite,
		ExtendBackground,
	}

	for _, mode := range extendModes {
		testImg, err := img.Copy(nil)
		require.NoError(t, err)

		err = testImg.Embed(5, 5, 110, 110, &EmbedOptions{
			Extend: mode,
		})
		require.NoError(t, err, "Extend mode %v should work", mode)
		testImg.Close()
	}

	// Test different blend modes
	baseImg, err := createWhiteImage(50, 50)
	require.NoError(t, err)
	defer baseImg.Close()

	overlayImg, err := createSolidColorImage(t, 30, 30, color.RGBA{255, 0, 0, 255})
	require.NoError(t, err)
	defer overlayImg.Close()

	blendModes := []BlendMode{
		BlendModeOver,
		BlendModeIn,
		BlendModeOut,
		BlendModeAtop,
		BlendModeXor,
		BlendModeMultiply,
		BlendModeScreen,
		BlendModeOverlay,
		BlendModeDarken,
		BlendModeLighten,
	}

	for _, mode := range blendModes {
		testImg, err := baseImg.Copy(nil)
		require.NoError(t, err)

		err = testImg.Composite2(overlayImg, mode, &Composite2Options{X: 10, Y: 10})
		require.NoError(t, err, "Blend mode %v should work", mode)
		testImg.Close()
	}
}

func TestSingleReturnValues(t *testing.T) {
	img, err := createTestGradientImage(t, 100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test operations that return single values

	// 1. Float return
	avgValue, err := img.Avg()
	require.NoError(t, err)
	assert.IsType(t, float64(0), avgValue)
	assert.Greater(t, avgValue, 0.0)
	assert.Less(t, avgValue, 255.0)

	// 2. Float return with bounds checking
	minValue, err := img.Min(nil)
	require.NoError(t, err)
	assert.IsType(t, float64(0), minValue)
	assert.GreaterOrEqual(t, minValue, 0.0)

	maxValue, err := img.Max(nil)
	require.NoError(t, err)
	assert.IsType(t, float64(0), maxValue)
	assert.LessOrEqual(t, maxValue, 255.0)

	// 3. Standard deviation
	devValue, err := img.Deviate()
	require.NoError(t, err)
	assert.IsType(t, float64(0), devValue)
	assert.GreaterOrEqual(t, devValue, 0.0)
}

func TestArrayReturnValues(t *testing.T) {
	img, err := createTestGradientImage(t, 100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test operations that return arrays

	// 1. Vector return (array + length)
	vector, err := img.Getpoint(50, 50, nil)
	require.NoError(t, err)
	assert.IsType(t, []float64{}, vector)
	assert.Len(t, vector, img.Bands())

	// Verify vector values are reasonable
	for i, val := range vector {
		assert.GreaterOrEqual(t, val, 0.0, "Band %d should be >= 0", i)
		assert.LessOrEqual(t, val, 255.0, "Band %d should be <= 255", i)
	}
}

func TestImageReturnValues(t *testing.T) {
	// Test operations that return new images

	// 1. Image creation functions
	blackImg, err := NewBlack(100, 100, &BlackOptions{Bands: 3})
	require.NoError(t, err)
	defer blackImg.Close()

	assert.IsType(t, &Image{}, blackImg)
	assert.Equal(t, 100, blackImg.Width())
	assert.Equal(t, 100, blackImg.Height())
	assert.Equal(t, 3, blackImg.Bands())

	// 2. Image transformation functions
	resizedImg, err := blackImg.Copy(nil)
	require.NoError(t, err)
	defer resizedImg.Close()

	err = resizedImg.Resize(0.5, nil)
	require.NoError(t, err)
	assert.Equal(t, 50, resizedImg.Width())
	assert.Equal(t, 50, resizedImg.Height())

	// 3. Multiple image returns
	img1, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img1.Close()

	img2, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img2.Close()
}

func TestErrorPropagation(t *testing.T) {
	// 1. Invalid file loading
	_, err := NewImageFromFile("/nonexistent/file.png", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")

	// 2. Invalid buffer loading
	invalidBuf := []byte{0x00, 0x01, 0x02, 0x03}
	_, err = NewImageFromBuffer(invalidBuf, nil)
	assert.Error(t, err)

	// 3. Invalid operation parameters
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Try to extract area outside bounds
	err = img.ExtractArea(50, 50, 200, 200) // extends beyond image
	assert.Error(t, err)

	// Try to resize with invalid scale
	err = img.Resize(-1.0, nil) // negative scale
	assert.Error(t, err)
}

func TestNilParameterHandling(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test that nil options are handled correctly

	// 1. Operations with nil options should use defaults
	err = img.Resize(0.5, nil)
	require.NoError(t, err)

	err = img.Gaussblur(2.0, nil)
	require.NoError(t, err)

	err = img.Embed(10, 10, 120, 120, nil)
	require.NoError(t, err)

	// 2. Buffer operations with nil options
	buf, err := img.PngsaveBuffer(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, buf)

	buf, err = img.JpegsaveBuffer(nil)
	require.NoError(t, err)
	assert.NotEmpty(t, buf)
}

func TestImageLifecycle(t *testing.T) {
	// Test proper image lifecycle management

	// 1. Create and immediately close
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)

	err = img.Resize(0.5, nil)
	require.NoError(t, err)

	// 2. Multiple close calls should be safe
	img.Close()
	img.Close()
	img.Close()
}

func TestImageCopySemantics(t *testing.T) {
	// Test that image copies work correctly

	original, err := createTestGradientImage(t, 100, 100)
	require.NoError(t, err)
	defer original.Close()

	// 1. Copy should create independent image
	copied, err := original.Copy(nil)
	require.NoError(t, err)
	defer copied.Close()

	assert.Equal(t, original.Width(), copied.Width())
	assert.Equal(t, original.Height(), copied.Height())
	assert.Equal(t, original.Bands(), copied.Bands())

	// 2. Modifying copy should not affect original
	originalWidth := original.Width()
	err = copied.Resize(0.5, nil)
	require.NoError(t, err)

	assert.Equal(t, originalWidth, original.Width()) // Original unchanged
	assert.Equal(t, originalWidth/2, copied.Width()) // Copy changed
}

func TestSourceLifecycle(t *testing.T) {
	pngData := createTestPngBuffer(t, 100, 100)

	// 1. Create source and close immediately
	reader := bytes.NewReader(pngData)
	source := NewSource(io.NopCloser(reader))
	source.Close()

	// 2. Multiple close calls should be safe
	source.Close()
	source.Close()

	// 3. Test source with successful image loading
	reader2 := bytes.NewReader(pngData)
	source2 := NewSource(io.NopCloser(reader2))
	defer source2.Close()

	img, err := NewImageFromSource(source2, nil)
	require.NoError(t, err)
	defer img.Close()

	assert.Equal(t, 100, img.Width())
	assert.Equal(t, 100, img.Height())
}

func TestDefaultOptions(t *testing.T) {
	// Test that default option functions work correctly

	// 1. Resize options
	resizeOpts := DefaultResizeOptions()
	assert.NotNil(t, resizeOpts)
	assert.IsType(t, &ResizeOptions{}, resizeOpts)

	// 2. Gaussblur options
	blurOpts := DefaultGaussblurOptions()
	assert.NotNil(t, blurOpts)
	assert.IsType(t, &GaussblurOptions{}, blurOpts)

	// 3. PNG save options
	pngOpts := DefaultPngsaveBufferOptions()
	assert.NotNil(t, pngOpts)
	assert.IsType(t, &PngsaveBufferOptions{}, pngOpts)

	// 4. JPEG save options
	jpegOpts := DefaultJpegsaveBufferOptions()
	assert.NotNil(t, jpegOpts)
	assert.IsType(t, &JpegsaveBufferOptions{}, jpegOpts)

	// 5. Load options
	loadOpts := DefaultLoadOptions()
	assert.NotNil(t, loadOpts)
	assert.IsType(t, &LoadOptions{}, loadOpts)
}

func TestOptionsFieldTypes(t *testing.T) {
	// Test that options structures have correct field types

	// Use reflection to verify field types match expectations
	resizeOpts := &ResizeOptions{}
	resizeType := reflect.TypeOf(resizeOpts).Elem()

	// Find Kernel field
	kernelField, found := resizeType.FieldByName("Kernel")
	require.True(t, found, "ResizeOptions should have Kernel field")
	assert.Equal(t, "Kernel", kernelField.Type.Name())

	// Find Gap field
	gapField, found := resizeType.FieldByName("Gap")
	require.True(t, found, "ResizeOptions should have Gap field")
	assert.Equal(t, "float64", gapField.Type.Name())

	// Test PNG options
	pngOpts := &PngsaveBufferOptions{}
	pngType := reflect.TypeOf(pngOpts).Elem()

	compressionField, found := pngType.FieldByName("Compression")
	require.True(t, found, "PngsaveBufferOptions should have Compression field")
	assert.Equal(t, "int", compressionField.Type.Name())
}

func TestEnumTypeSafety(t *testing.T) {
	// Test that enum types are properly typed

	// 1. Kernel enum
	var kernel Kernel = KernelLanczos3
	assert.IsType(t, Kernel(0), kernel)

	// 2. Extend enum
	var extend Extend = ExtendBlack
	assert.IsType(t, Extend(0), extend)

	// 3. BlendMode enum
	var blend BlendMode = BlendModeOver
	assert.IsType(t, BlendMode(0), blend)

	// 4. Angle enum
	var angle Angle = AngleD90
	assert.IsType(t, Angle(0), angle)

	// Test that enum values are distinct
	assert.NotEqual(t, KernelNearest, KernelLinear)
	assert.NotEqual(t, ExtendBlack, ExtendWhite)
	assert.NotEqual(t, BlendModeOver, BlendModeMultiply)
}

func TestImageTypeConstants(t *testing.T) {
	// Test ImageType constants and methods

	// 1. Type constants exist and are distinct
	assert.NotEqual(t, ImageTypeUnknown, ImageTypeJpeg)
	assert.NotEqual(t, ImageTypeJpeg, ImageTypePng)
	assert.NotEqual(t, ImageTypePng, ImageTypeWebp)

	// 2. MIME type method works
	jpegMime, ok := ImageTypeJpeg.MimeType()
	assert.True(t, ok)
	assert.Equal(t, "image/jpeg", jpegMime)

	pngMime, ok := ImageTypePng.MimeType()
	assert.True(t, ok)
	assert.Equal(t, "image/png", pngMime)

	unknownMime, ok := ImageTypeUnknown.MimeType()
	assert.False(t, ok)
	assert.Empty(t, unknownMime)
}

func TestCPointerHandling(t *testing.T) {
	// Test that C pointers are properly managed

	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)

	// Get the underlying C pointer (if accessible)
	// This tests that the binding properly manages C resources
	assert.NotNil(t, img)

	// Perform operations that would use the C pointer
	err = img.Resize(0.5, nil)
	require.NoError(t, err)

	// Close should properly free C resources
	img.Close()
}

func TestMemoryAlignment(t *testing.T) {
	// Test operations that involve memory layout

	// Create image from raw memory
	width, height, bands := 10, 10, 3
	data := make([]byte, width*height*bands)

	// Fill with pattern
	for i := range data {
		data[i] = byte(i % 256)
	}

	img, err := NewImageFromMemory(data, width, height, bands)
	require.NoError(t, err)
	defer img.Close()

	assert.Equal(t, width, img.Width())
	assert.Equal(t, height, img.Height())
	assert.Equal(t, bands, img.Bands())

	// Verify pixel values match what we set
	pixel, err := img.Getpoint(0, 0, nil)
	require.NoError(t, err)
	assert.Len(t, pixel, bands)
}

func TestGCIntegration(t *testing.T) {
	// Test that Go GC doesn't interfere with C memory management
	var images []*Image

	// Create multiple images
	for i := 0; i < 10; i++ {
		img, err := createWhiteImage(100, 100)
		require.NoError(t, err)
		images = append(images, img)
	}

	// Force GC
	runtime.GC()
	runtime.GC()

	// Images should still be valid
	for i, img := range images {
		assert.Equal(t, 100, img.Width(), "Image %d should still be valid after GC", i)
		assert.Equal(t, 100, img.Height(), "Image %d should still be valid after GC", i)
	}
	// Clean up
	for _, img := range images {
		img.Close()
	}
}

func TestKeepAliveSemantics(t *testing.T) {
	// Test that runtime.KeepAlive works correctly for buffer operations

	pngData := createTestPngBuffer(t, 100, 100)

	// Load from buffer - this should keep the buffer alive during operation
	img, err := NewImageFromBuffer(pngData, nil)
	require.NoError(t, err)
	defer img.Close()

	// Clear our reference to the buffer
	pngData = nil

	// Force GC
	runtime.GC()
	runtime.GC()

	// Image should still be valid
	assert.Equal(t, 100, img.Width())
	assert.Equal(t, 100, img.Height())

	// Should be able to perform operations
	err = img.Resize(0.5, nil)
	require.NoError(t, err)
}

func TestRequiredVsOptionalParameters(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test that operations work both with and without optional parameters

	// 1. Required parameters only (should call basic C function)
	err = img.Resize(0.5, nil)
	require.NoError(t, err)
	assert.Equal(t, 50, img.Width())

	// 2. With optional parameters (should call _with_options C function)
	img2, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img2.Close()

	err = img2.Resize(0.5, &ResizeOptions{
		Kernel: KernelLanczos3,
		Gap:    2.0,
	})
	require.NoError(t, err)
	assert.Equal(t, 50, img2.Width())

	// 3. Default options should behave same as nil
	img3, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img3.Close()

	err = img3.Resize(0.5, DefaultResizeOptions())
	require.NoError(t, err)
	assert.Equal(t, 50, img3.Width())
}

func TestOptionalParameterDefaults(t *testing.T) {
	// Test that optional parameters use correct default values when not specified

	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test with partial options (some fields set, others default)
	err = img.Gaussblur(2.0, &GaussblurOptions{
		MinAmpl: 0.1, // Set this field
		// Precision field should use default
	})
	require.NoError(t, err)

	// Test PNG save with partial options
	buf, err := img.PngsaveBuffer(&PngsaveBufferOptions{
		Compression: 6, // Set this field
		// Other fields should use defaults
	})
	require.NoError(t, err)
	assert.NotEmpty(t, buf)
}

func TestStringParameterHandling(t *testing.T) {
	// Test that string parameters are properly converted to C strings and cleaned up

	textImg, err := NewText("Hello World", &TextOptions{
		Font: "sans 20",
	})
	require.NoError(t, err)
	defer textImg.Close()
	assert.Greater(t, textImg.Width(), 0)
	assert.Greater(t, textImg.Height(), 0)

	textImg3, err := NewText("Test\nLine2\tTab", &TextOptions{
		Font: "sans 12",
	})
	require.NoError(t, err)
	defer textImg3.Close()
	assert.Greater(t, textImg3.Width(), 0)
	assert.Greater(t, textImg3.Height(), 0)

	textImg4, err := NewText("Hello 世界 🌍", &TextOptions{
		Font: "sans 16",
	})
	require.NoError(t, err)
	defer textImg4.Close()
	assert.Greater(t, textImg4.Width(), 0)
	assert.Greater(t, textImg4.Height(), 0)
}

func TestStringMetadataBinding(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test string metadata setting and getting

	// 1. ASCII strings
	img.SetString("test-ascii", "hello world")
	value, err := img.GetString("test-ascii")
	require.NoError(t, err)
	assert.Equal(t, "hello world", value)

	// 2. Empty strings
	img.SetString("test-empty", "")
	value, err = img.GetString("test-empty")
	require.NoError(t, err)
	assert.Equal(t, "", value)

	// 3. Strings with special characters
	testStr := "line1\nline2\ttab\"quote'apostrophe"
	img.SetString("test-special", testStr)
	value, err = img.GetString("test-special")
	require.NoError(t, err)
	assert.Equal(t, testStr, value)

	// 4. Unicode strings
	unicodeStr := "Hello 世界 🌍 Привет мир"
	img.SetString("test-unicode", unicodeStr)
	value, err = img.GetString("test-unicode")
	require.NoError(t, err)
	assert.Equal(t, unicodeStr, value)
}

func TestBufferParameterBinding(t *testing.T) {
	pngData := createTestPngBuffer(t, 50, 50)
	img, err := NewPngloadBuffer(pngData, nil)
	require.NoError(t, err)
	defer img.Close()

	assert.Equal(t, 50, img.Width())
	assert.Equal(t, 50, img.Height())

	// 2. JPEG buffer loading
	jpegData := createTestJpegBuffer(t, 60, 40)
	img2, err := NewJpegloadBuffer(jpegData, nil)
	require.NoError(t, err)
	defer img2.Close()

	assert.Equal(t, 60, img2.Width())
	assert.Equal(t, 40, img2.Height())

	// 3. Empty buffer (should fail gracefully)
	_, err = NewPngloadBuffer([]byte{}, nil)
	assert.Error(t, err)

	// 4. Invalid buffer (should fail gracefully)
	_, err = NewPngloadBuffer([]byte{1, 2, 3, 4}, nil)
	assert.Error(t, err)
}

func TestMemoryBufferBinding(t *testing.T) {
	// Test NewImageFromMemory with various buffer configurations
	// 1. Valid RGB buffer
	width, height, bands := 10, 8, 3
	data := make([]byte, width*height*bands)
	for i := range data {
		data[i] = byte(i % 256)
	}

	img, err := NewImageFromMemory(data, width, height, bands)
	require.NoError(t, err)
	defer img.Close()

	assert.Equal(t, width, img.Width())
	assert.Equal(t, height, img.Height())
	assert.Equal(t, bands, img.Bands())

	// 2. Single band (grayscale)
	grayData := make([]byte, width*height*1)
	for i := range grayData {
		grayData[i] = 128
	}

	grayImg, err := NewImageFromMemory(grayData, width, height, 1)
	require.NoError(t, err)
	defer grayImg.Close()

	assert.Equal(t, 1, grayImg.Bands())

	// 3. RGBA buffer
	rgbaData := make([]byte, width*height*4)
	for i := 0; i < len(rgbaData); i += 4 {
		rgbaData[i] = 255   // R
		rgbaData[i+1] = 0   // G
		rgbaData[i+2] = 0   // B
		rgbaData[i+3] = 255 // A
	}

	rgbaImg, err := NewImageFromMemory(rgbaData, width, height, 4)
	require.NoError(t, err)
	defer rgbaImg.Close()

	assert.Equal(t, 4, rgbaImg.Bands())
}

func TestBufferReturnBinding(t *testing.T) {
	img, err := createWhiteImage(50, 50)
	require.NoError(t, err)
	defer img.Close()

	// Test that buffer returns properly manage memory

	// 1. Multiple buffer saves should work
	for i := 0; i < 5; i++ {
		buf, err := img.PngsaveBuffer(nil)
		require.NoError(t, err)
		assert.NotEmpty(t, buf)
		assert.Greater(t, len(buf), 100)

		// Verify buffer contents are valid
		assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, buf[:4]) // PNG signature
	}

	// 2. Different formats should return different buffers
	pngBuf, err := img.PngsaveBuffer(nil)
	require.NoError(t, err)

	jpegBuf, err := img.JpegsaveBuffer(nil)
	require.NoError(t, err)

	webpBuf, err := img.WebpsaveBuffer(nil)
	require.NoError(t, err)

	// Buffers should be different
	assert.NotEqual(t, pngBuf[:4], jpegBuf[:4])
	assert.NotEqual(t, pngBuf[:4], webpBuf[:4])
	assert.NotEqual(t, jpegBuf[:4], webpBuf[:4])
}

func TestInterpolationBinding(t *testing.T) {
	// Test Interpolate object parameter binding

	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test different interpolation types
	interpolations := []InterpolateType{
		InterpolateNearest,
		InterpolateBilinear,
		InterpolateBicubic,
		InterpolateLbb,
		InterpolateNohalo,
		InterpolateVsqbs,
	}

	for _, interpType := range interpolations {
		testImg, err := img.Copy(nil)
		require.NoError(t, err)

		// Create interpolation object
		interp := NewInterpolate(interpType)
		assert.NotNil(t, interp)

		// Test with operations that accept interpolation
		err = testImg.Resize(0.5, &ResizeOptions{
			Kernel: KernelLanczos3,
		})
		require.NoError(t, err, "Interpolation %v should work", interpType)

		// Clean up
		interp.Close()
		testImg.Close()
	}
}

func TestInterpolationLifecycle(t *testing.T) {
	// Test interpolation object lifecycle

	// 1. Create and close immediately
	interp := NewInterpolate(InterpolateBilinear)
	assert.NotNil(t, interp)
	interp.Close()

	// 2. Multiple close calls should be safe
	interp.Close()
	interp.Close()

	// 3. Invalid interpolation name should fallback
	invalidInterp := NewInterpolate(InterpolateType("invalid"))
	assert.NotNil(t, invalidInterp) // Should fallback to default
	invalidInterp.Close()
}

func TestSourceParameterBinding(t *testing.T) {
	pngData := createTestPngBuffer(t, 100, 100)

	// Test different source configurations

	// 1. Basic reader source
	reader1 := bytes.NewReader(pngData)
	source1 := NewSource(io.NopCloser(reader1))
	defer source1.Close()

	img1, err := NewImageFromSource(source1, nil)
	require.NoError(t, err)
	defer img1.Close()

	// 2. ReadSeeker source (should enable seeking)
	reader2 := bytes.NewReader(pngData)
	source2 := NewSource(io.NopCloser(reader2))
	defer source2.Close()

	img2, err := NewImageFromSource(source2, nil)
	require.NoError(t, err)
	defer img2.Close()

	// Both should work the same
	assert.Equal(t, img1.Width(), img2.Width())
	assert.Equal(t, img1.Height(), img2.Height())
}

func TestSourceWithOptions(t *testing.T) {
	jpegData := createTestJpegBuffer(t, 200, 150)

	reader := bytes.NewReader(jpegData)
	source := NewSource(io.NopCloser(reader))
	defer source.Close()

	// Test loading from source with options
	img, err := NewImageFromSource(source, &LoadOptions{
		Shrink:      2,
		Autorotate:  true,
		FailOnError: true,
	})
	require.NoError(t, err)
	defer img.Close()

	// Should be shrunk
	assert.InDelta(t, 100, img.Width(), 10) // Allow some variance
	assert.InDelta(t, 75, img.Height(), 10)
}

func TestGeneratedFunctionSignatures(t *testing.T) {
	// Test that generated functions have expected signatures

	// Use reflection to verify function signatures
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	imgType := reflect.TypeOf(img)

	// 1. Check that Resize method exists with correct signature
	resizeMethod, found := imgType.MethodByName("Resize")
	require.True(t, found, "Resize method should exist")

	// Should have signature: func(*Image, float64, *ResizeOptions) error
	assert.Equal(t, 3, resizeMethod.Type.NumIn())  // receiver + 2 params
	assert.Equal(t, 1, resizeMethod.Type.NumOut()) // error return

	// 2. Check save buffer methods
	pngSaveMethod, found := imgType.MethodByName("PngsaveBuffer")
	require.True(t, found, "PngsaveBuffer method should exist")

	// Should return ([]byte, error)
	assert.Equal(t, 2, pngSaveMethod.Type.NumOut())
	assert.Equal(t, "[]uint8", pngSaveMethod.Type.Out(0).String()) // []byte
	assert.True(t, pngSaveMethod.Type.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()))
}

func TestOptionStructureGeneration(t *testing.T) {
	// Test that option structures have expected fields

	// 1. ResizeOptions
	opts := &ResizeOptions{}
	optsType := reflect.TypeOf(opts).Elem()

	expectedFields := []string{"Kernel", "Gap", "Vscale"}
	for _, fieldName := range expectedFields {
		field, found := optsType.FieldByName(fieldName)
		assert.True(t, found, "ResizeOptions should have %s field", fieldName)
		assert.True(t, field.IsExported(), "Field %s should be exported", fieldName)
	}

	// 2. PngsaveBufferOptions
	pngOpts := &PngsaveBufferOptions{}
	pngOptsType := reflect.TypeOf(pngOpts).Elem()

	pngFields := []string{"Compression", "Interlace", "Filter"}
	for _, fieldName := range pngFields {
		field, found := pngOptsType.FieldByName(fieldName)
		assert.True(t, found, "PngsaveBufferOptions should have %s field", fieldName)
		assert.True(t, field.IsExported(), "Field %s should be exported", fieldName)
	}
}

func TestEnumConstantGeneration(t *testing.T) {
	// Test that enum constants are properly generated

	// 1. Kernel enum values should be distinct
	kernels := []Kernel{
		KernelNearest,
		KernelLinear,
		KernelCubic,
		KernelMitchell,
		KernelLanczos2,
		KernelLanczos3,
	}

	// All should be different values
	for i := 0; i < len(kernels); i++ {
		for j := i + 1; j < len(kernels); j++ {
			assert.NotEqual(t, kernels[i], kernels[j],
				"Kernel values %d and %d should be different", i, j)
		}
	}

	// 2. BlendMode enum values
	blendModes := []BlendMode{
		BlendModeOver,
		BlendModeIn,
		BlendModeOut,
		BlendModeAtop,
		BlendModeXor,
		BlendModeMultiply,
		BlendModeScreen,
	}

	for i := 0; i < len(blendModes); i++ {
		for j := i + 1; j < len(blendModes); j++ {
			assert.NotEqual(t, blendModes[i], blendModes[j],
				"BlendMode values %d and %d should be different", i, j)
		}
	}
}

func TestFormatSpecificOptions(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test format-specific save options work correctly

	// 1. PNG-specific options
	pngBuf, err := img.PngsaveBuffer(&PngsaveBufferOptions{
		Compression: 9,
		Interlace:   true,
		Filter:      PngFilterPaeth,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, pngBuf)

	// 2. JPEG-specific options
	jpegBuf, err := img.JpegsaveBuffer(&JpegsaveBufferOptions{
		Q:              95,
		OptimizeCoding: true,
		Interlace:      true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, jpegBuf)

	// 3. WebP-specific options
	webpBuf, err := img.WebpsaveBuffer(&WebpsaveBufferOptions{
		Q:        80,
		Lossless: false,
		Effort:   6,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, webpBuf)

	// 1. JPEG with shrink-on-load
	jpegData := createTestJpegBuffer(t, 400, 300)
	jpegImg, err := NewJpegloadBuffer(jpegData, &JpegloadBufferOptions{
		Shrink:     2,
		Autorotate: true,
	})
	require.NoError(t, err)
	defer jpegImg.Close()

	// Should be approximately half size
	assert.InDelta(t, 200, jpegImg.Width(), 20)
	assert.InDelta(t, 150, jpegImg.Height(), 20)

	// 2. PNG with specific options
	pngData := createTestPngBuffer(t, 200, 200)
	pngImg, err := NewPngloadBuffer(pngData, &PngloadBufferOptions{
		Unlimited: true,
	})
	require.NoError(t, err)
	defer pngImg.Close()

	assert.Equal(t, 200, pngImg.Width())
	assert.Equal(t, 200, pngImg.Height())
}

// writeCloser wraps an io.Writer to make it an io.WriteCloser
type writeCloser struct {
	io.Writer
}

func (w *writeCloser) Close() error {
	// If the underlying writer implements io.Closer, close it
	if closer, ok := w.Writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func TestTarget(t *testing.T) {
	// Create a test image
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Create a buffer target
	var buf bytes.Buffer
	target := NewTarget(&writeCloser{&buf})
	defer target.Close()

	// Save to target using WebpsaveTarget
	err = img.WebpsaveTarget(target, &WebpsaveTargetOptions{
		Q:      85,
		Effort: 4,
	})
	require.NoError(t, err)

	// Verify data was written
	assert.Greater(t, buf.Len(), 0, "Target should have received data")

	// Verify it's valid WebP data (check signature)
	data := buf.Bytes()
	assert.GreaterOrEqual(t, len(data), 12, "WebP data should be at least 12 bytes")
	assert.Equal(t, "RIFF", string(data[0:4]), "Should start with RIFF")
	assert.Equal(t, "WEBP", string(data[8:12]), "Should contain WEBP signature")
}

func TestTargetLifecycle(t *testing.T) {
	// Test target lifecycle management

	// 1. Create target and close immediately
	var buf bytes.Buffer
	target := NewTarget(&writeCloser{&buf})
	target.Close()

	// 2. Multiple close calls should be safe
	target.Close()
	target.Close()

	// 3. Test with file target
	testDir := ensureTestDir(t)
	testFile := filepath.Join(testDir, "target_test.webp")

	file, err := os.Create(testFile)
	require.NoError(t, err)
	defer os.Remove(testFile)

	fileTarget := NewTarget(file) // file implements io.WriteCloser
	defer fileTarget.Close()

	// Create and save test image
	img, err := createWhiteImage(50, 50)
	require.NoError(t, err)
	defer img.Close()

	err = img.WebpsaveTarget(fileTarget, nil)
	require.NoError(t, err)

	// Verify file was created and has content
	fileInfo, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0), "Target file should have content")
}

func TestTargetPNGSaveOptions(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test key PNG save options with targets
	testCases := []struct {
		name    string
		options *PngsaveTargetOptions
	}{
		{"nil options", nil},
		{"default options", DefaultPngsaveTargetOptions()},
		{"high compression", &PngsaveTargetOptions{Compression: 9}},
		{"low compression", &PngsaveTargetOptions{Compression: 1}},
		{"interlaced", &PngsaveTargetOptions{Compression: 6, Interlace: true}},
		{"with filter", &PngsaveTargetOptions{Compression: 6, Filter: PngFilterPaeth}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			target := NewTarget(&writeCloser{&buf})
			defer target.Close()

			err := img.PngsaveTarget(target, tc.options)
			require.NoError(t, err, "PngsaveTarget should succeed with %s", tc.name)

			data := buf.Bytes()
			assert.Greater(t, len(data), 50, "Should produce PNG data with %s", tc.name)
			assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, data[0:4], "Should be valid PNG with %s", tc.name)

			t.Logf("%s: produced %d bytes", tc.name, len(data))
		})
	}
}

func TestTargetFormatSelection(t *testing.T) {
	// Test choosing the right format based on use case
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test format selection for different scenarios
	testCases := []struct {
		name     string
		saveFunc func(*Target) error
		checkSig func([]byte) bool
	}{
		{
			name: "WebP for modern web",
			saveFunc: func(target *Target) error {
				return img.WebpsaveTarget(target, &WebpsaveTargetOptions{Q: 80})
			},
			checkSig: func(data []byte) bool {
				return len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP"
			},
		},
		{
			name: "JPEG for photos",
			saveFunc: func(target *Target) error {
				return img.JpegsaveTarget(target, &JpegsaveTargetOptions{Q: 80})
			},
			checkSig: func(data []byte) bool {
				return len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8
			},
		},
		{
			name: "PNG for lossless",
			saveFunc: func(target *Target) error {
				return img.PngsaveTarget(target, &PngsaveTargetOptions{Compression: 6})
			},
			checkSig: func(data []byte) bool {
				return len(data) >= 4 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			target := NewTarget(&writeCloser{&buf})
			defer target.Close()

			err := tc.saveFunc(target)
			require.NoError(t, err, "Save should succeed for %s", tc.name)

			data := buf.Bytes()
			assert.True(t, tc.checkSig(data), "Should have correct format signature for %s", tc.name)
			assert.Greater(t, len(data), 50, "Should produce substantial data for %s", tc.name)

			t.Logf("%s: %d bytes", tc.name, len(data))
		})
	}
}

func TestTargetWithMultipleFormats(t *testing.T) {
	// Test that Target can be used with different save operations
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Test WebP save to target
	var webpBuf bytes.Buffer
	webpTarget := NewTarget(&writeCloser{&webpBuf})
	defer webpTarget.Close()

	err = img.WebpsaveTarget(webpTarget, nil)
	require.NoError(t, err)
	assert.Greater(t, webpBuf.Len(), 0, "WebP target save should work")

	// Test JPEG save to target
	var jpegBuf bytes.Buffer
	jpegTarget := NewTarget(&writeCloser{&jpegBuf})
	defer jpegTarget.Close()

	err = img.JpegsaveTarget(jpegTarget, &JpegsaveTargetOptions{
		Q:              80,
		OptimizeCoding: true,
	})
	require.NoError(t, err)
	assert.Greater(t, jpegBuf.Len(), 0, "JPEG target save should work")

	// Test PNG save to target
	var pngBuf bytes.Buffer
	pngTarget := NewTarget(&writeCloser{&pngBuf})
	defer pngTarget.Close()

	err = img.PngsaveTarget(pngTarget, &PngsaveTargetOptions{
		Compression: 6,
		Filter:      PngFilterAll,
	})
	require.NoError(t, err)
	assert.Greater(t, pngBuf.Len(), 0, "PNG target save should work")

	// Verify different formats produce different signatures
	webpData := webpBuf.Bytes()
	jpegData := jpegBuf.Bytes()
	pngData := pngBuf.Bytes()

	assert.Equal(t, "RIFF", string(webpData[0:4]), "WebP should have RIFF signature")
	assert.Equal(t, []byte{0xFF, 0xD8}, jpegData[0:2], "JPEG should have JPEG signature")
	assert.Equal(t, []byte{0x89, 0x50, 0x4E, 0x47}, pngData[0:4], "PNG should have PNG signature")

	t.Logf("Multi-format targets: WebP=%d bytes, JPEG=%d bytes, PNG=%d bytes",
		len(webpData), len(jpegData), len(pngData))
}

func TestTargetWithFileOutput(t *testing.T) {
	// Test target with actual file I/O
	testDir := ensureTestDir(t)
	testFile := filepath.Join(testDir, "target_output.webp")
	defer os.Remove(testFile)

	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Create file target
	file, err := os.Create(testFile)
	require.NoError(t, err)

	target := NewTarget(file) // file implements io.WriteCloser
	defer target.Close()

	// Save to file via target
	err = img.WebpsaveTarget(target, nil)
	require.NoError(t, err)

	// Close target to ensure file is flushed
	target.Close()

	// Verify file exists and has content
	fileInfo, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(100), "File should have content")

	t.Logf("File target save: %d bytes", fileInfo.Size())
}

func TestTargetMemoryLeaks(t *testing.T) {
	// Create and use multiple targets
	for i := 0; i < 10; i++ {
		img, err := createWhiteImage(50, 50)
		require.NoError(t, err)

		var buf bytes.Buffer
		target := NewTarget(&writeCloser{&buf})

		err = img.WebpsaveTarget(target, nil)
		require.NoError(t, err)

		// Proper cleanup
		target.Close()
		img.Close()

		// Force GC
		if i%3 == 0 {
			runtime.GC()
		}
	}

	// Final GC
	runtime.GC()
}

func TestTargetOptionStructures(t *testing.T) {
	// Test that generated *saveTargetOptions work correctly

	// Test WebP options
	webpOpts := DefaultWebpsaveTargetOptions()
	assert.NotNil(t, webpOpts)
	assert.IsType(t, &WebpsaveTargetOptions{}, webpOpts)

	// Test JPEG options
	jpegOpts := DefaultJpegsaveTargetOptions()
	assert.NotNil(t, jpegOpts)
	assert.IsType(t, &JpegsaveTargetOptions{}, jpegOpts)

	// Test PNG options
	pngOpts := DefaultPngsaveTargetOptions()
	assert.NotNil(t, pngOpts)
	assert.IsType(t, &PngsaveTargetOptions{}, pngOpts)

	// Test with actual saves
	img, err := createWhiteImage(50, 50)
	require.NoError(t, err)
	defer img.Close()

	// Test each format with custom options
	formats := []struct {
		name     string
		saveFunc func(*Target) error
	}{
		{
			name: "WebP",
			saveFunc: func(target *Target) error {
				return img.WebpsaveTarget(target, &WebpsaveTargetOptions{Q: 80, Lossless: false})
			},
		},
		{
			name: "JPEG",
			saveFunc: func(target *Target) error {
				return img.JpegsaveTarget(target, &JpegsaveTargetOptions{Q: 85, OptimizeCoding: true})
			},
		},
		{
			name: "PNG",
			saveFunc: func(target *Target) error {
				return img.PngsaveTarget(target, &PngsaveTargetOptions{Compression: 6, Interlace: true})
			},
		},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			var buf bytes.Buffer
			target := NewTarget(&writeCloser{&buf})
			defer target.Close()

			err := format.saveFunc(target)
			require.NoError(t, err, "%s custom options should work", format.name)
			assert.Greater(t, buf.Len(), 0, "%s should produce data", format.name)
		})
	}
}

func TestTargetWithSourceRoundTrip(t *testing.T) {
	// Test Target → Source round trip to verify output compatibility
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Save to target
	var webpBuf bytes.Buffer
	target := NewTarget(&writeCloser{&webpBuf})
	defer target.Close()

	err = img.WebpsaveTarget(target, &WebpsaveTargetOptions{Lossless: true})
	require.NoError(t, err)

	// Load back via source
	reader := bytes.NewReader(webpBuf.Bytes())
	source := NewSource(io.NopCloser(reader))
	defer source.Close()

	loadedImg, err := NewImageFromSource(source, nil)
	require.NoError(t, err)
	defer loadedImg.Close()

	// Verify basic properties are preserved
	assert.Equal(t, img.Width(), loadedImg.Width(), "Width should be preserved")
	assert.Equal(t, img.Height(), loadedImg.Height(), "Height should be preserved")

	t.Logf("Round trip successful: %dx%d image, %d bytes WebP",
		loadedImg.Width(), loadedImg.Height(), webpBuf.Len())
}

func TestNewThumbnail_Options(t *testing.T) {
	// Create a test image
	width, height := 400, 300
	pngData := createTestPngBuffer(t, width, height)

	// Create test file for file-based thumbnail
	testDir := ensureTestDir(t)
	testFile := filepath.Join(testDir, "test_thumb.png")
	err := os.WriteFile(testFile, pngData, 0644)
	require.NoError(t, err)
	defer os.Remove(testFile)

	testCases := []struct {
		name          string
		bufferOptions *ThumbnailBufferOptions
		fileOptions   *ThumbnailOptions
		sourceOptions *ThumbnailSourceOptions
		validate      func(*testing.T, *Image, string)
	}{
		{
			name:          "nil options",
			bufferOptions: nil,
			fileOptions:   nil,
			sourceOptions: nil,
			validate: func(t *testing.T, img *Image, method string) {
				assert.Equal(t, 200, img.Width(), "%s: width should be 200", method)
				assert.Equal(t, 150, img.Height(), "%s: height should be 150", method)
			},
		},
		{
			name: "with specific height",
			bufferOptions: &ThumbnailBufferOptions{
				Height: 100,
			},
			fileOptions: &ThumbnailOptions{
				Height: 100,
			},
			sourceOptions: &ThumbnailSourceOptions{
				Height: 100,
			},
			validate: func(t *testing.T, img *Image, method string) {
				assert.InDelta(t, 133, img.Width(), 2, "%s: width should be ~133", method)
				assert.Equal(t, 100, img.Height(), "%s: height should be 100", method)
			},
		},
		{
			name: "with size constraint",
			bufferOptions: &ThumbnailBufferOptions{
				Size: SizeBoth,
			},
			fileOptions: &ThumbnailOptions{
				Size: SizeBoth,
			},
			sourceOptions: &ThumbnailSourceOptions{
				Size: SizeBoth,
			},
			validate: func(t *testing.T, img *Image, method string) {
				// Both dimensions should be <= 200
				assert.LessOrEqual(t, img.Width(), 200, "%s: width should be <= 200", method)
				assert.LessOrEqual(t, img.Height(), 200, "%s: height should be <= 200", method)
			},
		},
		{
			name: "with fail on error",
			bufferOptions: &ThumbnailBufferOptions{
				FailOn: FailOnError,
			},
			fileOptions: &ThumbnailOptions{
				FailOn: FailOnError,
			},
			sourceOptions: &ThumbnailSourceOptions{
				FailOn: FailOnError,
			},
			validate: func(t *testing.T, img *Image, method string) {
				assert.Equal(t, 200, img.Width(), "%s: width should be 200", method)
				assert.InDelta(t, 150, img.Height(), 2, "%s: height should be ~150", method)
			},
		},
		{
			name: "with linear interpolation",
			bufferOptions: &ThumbnailBufferOptions{
				Linear: true,
			},
			fileOptions: &ThumbnailOptions{
				Linear: true,
			},
			sourceOptions: &ThumbnailSourceOptions{
				Linear: true,
			},
			validate: func(t *testing.T, img *Image, method string) {
				assert.Equal(t, 200, img.Width(), "%s: width should be 200", method)
				assert.InDelta(t, 150, img.Height(), 2, "%s: height should be ~150", method)
			},
		},
		{
			name: "with crop center",
			bufferOptions: &ThumbnailBufferOptions{
				Height: 200,
				Crop:   InterestingCentre,
			},
			fileOptions: &ThumbnailOptions{
				Height: 200,
				Crop:   InterestingCentre,
			},
			sourceOptions: &ThumbnailSourceOptions{
				Height: 200,
				Crop:   InterestingCentre,
			},
			validate: func(t *testing.T, img *Image, method string) {
				assert.Equal(t, 200, img.Width(), "%s: width should be 200", method)
				assert.Equal(t, 200, img.Height(), "%s: height should be 200 (cropped)", method)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test 1: NewThumbnailBuffer
			t.Run("buffer", func(t *testing.T) {
				thumbnail, err := NewThumbnailBuffer(pngData, 200, tc.bufferOptions)
				require.NoError(t, err)
				defer thumbnail.Close()

				tc.validate(t, thumbnail, "buffer")
				t.Logf("Buffer thumbnail with %s: %dx%d", tc.name, thumbnail.Width(), thumbnail.Height())
			})

			// Test 2: NewThumbnail (file)
			t.Run("file", func(t *testing.T) {
				thumbnail, err := NewThumbnail(testFile, 200, tc.fileOptions)
				require.NoError(t, err)
				defer thumbnail.Close()

				tc.validate(t, thumbnail, "file")
				t.Logf("File thumbnail with %s: %dx%d", tc.name, thumbnail.Width(), thumbnail.Height())
			})

			// Test 3: NewThumbnailSource
			t.Run("source", func(t *testing.T) {
				reader := bytes.NewReader(pngData)
				source := NewSource(io.NopCloser(reader))
				defer source.Close()

				thumbnail, err := NewThumbnailSource(source, 200, tc.sourceOptions)
				require.NoError(t, err)
				defer thumbnail.Close()

				tc.validate(t, thumbnail, "source")
				t.Logf("Source thumbnail with %s: %dx%d", tc.name, thumbnail.Width(), thumbnail.Height())
			})
		})
	}
}

func TestImage_SetArrayInt(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	testArray := []int{1, 2, 3, 4, 5}
	err = img.SetArrayInt("test-array", testArray)
	require.NoError(t, err)

	retrievedArray, err := img.GetArrayInt("test-array")
	require.NoError(t, err)
	assert.Equal(t, testArray, retrievedArray, "Retrieved array should match the set array")

	singleArray := []int{42}
	err = img.SetArrayInt("single-array", singleArray)
	require.NoError(t, err)

	retrievedSingle, err := img.GetArrayInt("single-array")
	require.NoError(t, err)
	assert.Equal(t, singleArray, retrievedSingle, "Retrieved single element array should match")

	negativeArray := []int{-1, -2, 0, 1, 2}
	err = img.SetArrayInt("negative-array", negativeArray)
	require.NoError(t, err)

	retrievedNegative, err := img.GetArrayInt("negative-array")
	require.NoError(t, err)
	assert.Equal(t, negativeArray, retrievedNegative, "Retrieved negative array should match")
}

func TestImage_SetArrayDouble(t *testing.T) {
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	testArray := []float64{1.1, 2.2, 3.3, 4.4, 5.5}
	err = img.SetArrayDouble("test-array", testArray)
	require.NoError(t, err)

	retrievedArray, err := img.GetArrayDouble("test-array")
	require.NoError(t, err)
	assert.Equal(t, testArray, retrievedArray, "Retrieved array should match the set array")

	singleArray := []float64{42.1}
	err = img.SetArrayDouble("single-array", singleArray)
	require.NoError(t, err)

	retrievedSingle, err := img.GetArrayDouble("single-array")
	require.NoError(t, err)
	assert.Equal(t, singleArray, retrievedSingle, "Retrieved single element array should match")

	negativeArray := []float64{-1.1, -2.1, 0, 1.1, 2.2}
	err = img.SetArrayDouble("negative-array", negativeArray)
	require.NoError(t, err)

	retrievedNegative, err := img.GetArrayDouble("negative-array")
	require.NoError(t, err)
	assert.Equal(t, negativeArray, retrievedNegative, "Retrieved negative array should match")
}

// TestOptionalOutputStructGeneration tests that optional output structs are generated correctly
func TestOptionalOutputStructGeneration(t *testing.T) {
	// Test the optional output struct exists and has correct types
	options := DefaultMosaicOptions()
	require.NotNil(t, options, "MosaicOptions should not be nil")

	// Verify struct field types (this tests the code generation)
	assert.IsType(t, 0, options.Dx0, "Dx0 should be int type")
	assert.IsType(t, 0, options.Dy0, "Dy0 should be int type")
	assert.IsType(t, float64(0), options.Scale1, "Scale1 should be float64 type")
	assert.IsType(t, float64(0), options.Angle1, "Angle1 should be float64 type")
	assert.IsType(t, float64(0), options.Dx1, "Dx1 should be float64 type")
	assert.IsType(t, float64(0), options.Dy1, "Dy1 should be float64 type")

	// Test that initial values are zero
	assert.Equal(t, 0, options.Dx0, "Initial Dx0 should be 0")
	assert.Equal(t, 0, options.Dy0, "Initial Dy0 should be 0")
	assert.Equal(t, 0.0, options.Scale1, "Initial Scale1 should be 0.0")
	assert.Equal(t, 0.0, options.Angle1, "Initial Angle1 should be 0.0")

	t.Log("✓ Mosaic optional output struct validation passed")

	// Test smartcrop optional outputs
	smartcropOptions := DefaultSmartcropOptions()
	require.NotNil(t, smartcropOptions, "SmartcropOptions should not be nil")

	// Verify field types
	assert.IsType(t, 0, smartcropOptions.AttentionX, "AttentionX should be int type")
	assert.IsType(t, 0, smartcropOptions.AttentionY, "AttentionY should be int type")

	// Test initial values
	assert.Equal(t, 0, smartcropOptions.AttentionX, "Initial AttentionX should be 0")
	assert.Equal(t, 0, smartcropOptions.AttentionY, "Initial AttentionY should be 0")

	t.Log("✓ Smartcrop optional output struct validation passed")

	// Test min/max optional outputs
	minOptions := DefaultMinOptions()
	require.NotNil(t, minOptions, "MinOptions should not be nil")
	assert.IsType(t, 0, minOptions.X, "Min X should be int type")
	assert.IsType(t, 0, minOptions.Y, "Min Y should be int type")

	maxOptions := DefaultMaxOptions()
	require.NotNil(t, maxOptions, "MaxOptions should not be nil")
	assert.IsType(t, 0, maxOptions.X, "Max X should be int type")
	assert.IsType(t, 0, maxOptions.Y, "Max Y should be int type")

	t.Log("✓ Min/Max optional output struct validation passed")
}

// TestSmartcropOptionalOutputs tests smartcrop's attention coordinates
func TestSmartcropOptionalOutputs(t *testing.T) {
	// Create an image with a bright spot in a known location
	width, height := 200, 200
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with dark background
	darkColor := color.RGBA{50, 50, 50, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, darkColor)
		}
	}

	// Add a bright spot at (150, 50) - this should attract attention
	brightColor := color.RGBA{255, 255, 255, 255}
	for y := 40; y < 60; y++ {
		for x := 140; x < 160; x++ {
			img.Set(x, y, brightColor)
		}
	}

	// Convert to vips image
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	vipsImg, err := NewImageFromBuffer(buf.Bytes(), nil)
	require.NoError(t, err)
	defer vipsImg.Close()

	// Test smartcrop with optional outputs
	cropWidth, cropHeight := 100, 100
	options := DefaultSmartcropOptions()
	require.NotNil(t, options)

	err = vipsImg.Smartcrop(cropWidth, cropHeight, options)
	require.NoError(t, err)

	t.Logf("Smartcrop attention coordinates: x=%d, y=%d", options.AttentionX, options.AttentionY)

	// Verify attention coordinates are within image bounds
	assert.GreaterOrEqual(t, options.AttentionX, 0, "AttentionX should be >= 0")
	assert.LessOrEqual(t, options.AttentionX, width, "AttentionX should be <= width")
	assert.GreaterOrEqual(t, options.AttentionY, 0, "AttentionY should be >= 0")
	assert.LessOrEqual(t, options.AttentionY, height, "AttentionY should be <= height")

	// The attention should be somewhere near our bright spot (150, 50)
	// Allow some tolerance since the algorithm may not pick the exact center
	assert.InDelta(t, 150, options.AttentionX, 50, "AttentionX should be near the bright spot")
	assert.InDelta(t, 50, options.AttentionY, 50, "AttentionY should be near the bright spot")
}

// TestMaxMinOptionalOutputs tests max/min operations with position outputs
func TestMaxMinOptionalOutputs(t *testing.T) {
	// Create a gradient image where we know the min/max locations
	width, height := 100, 80
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create gradient: darkest at (0,0), brightest at (width-1, height-1)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Linear gradient
			value := uint8(float64(x+y) / float64(width+height-2) * 255)
			img.Set(x, y, color.RGBA{value, value, value, 255})
		}
	}

	// Convert to vips image
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	require.NoError(t, err)

	vipsImg, err := NewImageFromBuffer(buf.Bytes(), nil)
	require.NoError(t, err)
	defer vipsImg.Close()

	// Test Min operation with position output
	minOptions := DefaultMinOptions()
	require.NotNil(t, minOptions)

	minValue, err := vipsImg.Min(minOptions)
	require.NoError(t, err)

	t.Logf("Min value: %.2f at position (%d, %d)", minValue, minOptions.X, minOptions.Y)

	// Min should be at (0,0) and close to 0
	assert.InDelta(t, 0.0, minValue, 10.0, "Min value should be close to 0")
	assert.Equal(t, 0, minOptions.X, "Min X should be 0")
	assert.Equal(t, 0, minOptions.Y, "Min Y should be 0")

	// Test Max operation with position output
	maxOptions := DefaultMaxOptions()
	require.NotNil(t, maxOptions)

	maxValue, err := vipsImg.Max(maxOptions)
	require.NoError(t, err)

	t.Logf("Max value: %.2f at position (%d, %d)", maxValue, maxOptions.X, maxOptions.Y)

	// Max should be at (width-1, height-1) and close to 255
	assert.InDelta(t, 255.0, maxValue, 10.0, "Max value should be close to 255")
	assert.Equal(t, width-1, maxOptions.X, "Max X should be width-1")
	assert.Equal(t, height-1, maxOptions.Y, "Max Y should be height-1")
}

// TestDrawFloodOptionalOutputs tests draw_flood's affected area outputs
func TestDrawFloodOptionalOutputs(t *testing.T) {
	// Create a white image with a small colored region
	width, height := 200, 150
	img, err := createWhiteImage(width, height)
	require.NoError(t, err)
	defer img.Close()

	// Draw a small red rectangle that we'll flood fill
	redColor := []float64{255, 0, 0}
	err = img.DrawRect(redColor, 50, 40, 30, 20, &DrawRectOptions{Fill: true})
	require.NoError(t, err)

	// Test flood fill with optional outputs
	options := DefaultDrawFloodOptions()
	require.NotNil(t, options)

	// Flood fill from a point inside the red rectangle
	floodColor := []float64{0, 255, 0} // Green
	startX, startY := 60, 50

	err = img.DrawFlood(floodColor, startX, startY, options)
	require.NoError(t, err)

	t.Logf("Flood fill affected area: left=%d, top=%d, width=%d, height=%d",
		options.Left, options.Top, options.Width, options.Height)

	// Verify the affected area bounds are reasonable
	assert.GreaterOrEqual(t, options.Left, 0, "Left should be >= 0")
	assert.GreaterOrEqual(t, options.Top, 0, "Top should be >= 0")
	assert.Greater(t, options.Width, 0, "Width should be > 0")
	assert.Greater(t, options.Height, 0, "Height should be > 0")

	// The affected area should encompass our red rectangle
	assert.LessOrEqual(t, options.Left, 50, "Left should be <= 50")
	assert.LessOrEqual(t, options.Top, 40, "Top should be <= 40")
	assert.GreaterOrEqual(t, options.Left+options.Width, 80, "Right edge should be >= 80")
	assert.GreaterOrEqual(t, options.Top+options.Height, 60, "Bottom edge should be >= 60")

	// Calculate total affected pixels
	totalPixels := options.Width * options.Height
	t.Logf("Total affected pixels: %d", totalPixels)
	assert.Greater(t, totalPixels, 0, "Should have affected some pixels")
}

// TestOptionalOutputsWithNilOptions tests that optional outputs work with nil options
func TestOptionalOutputsWithNilOptions(t *testing.T) {
	// Create test images with textured patterns for better mosaic matching
	img1, err := createCheckboardImage(t, 200, 150, 20)
	require.NoError(t, err)
	defer img1.Close()

	img2, err := createCheckboardImage(t, 150, 150, 20)
	require.NoError(t, err)
	defer img2.Close()

	// Test that operations work with nil options (should use defaults)
	// Use realistic tie points for mosaic operation
	err = img1.Mosaic(img2, DirectionHorizontal, 150, 75, 75, 75, nil)
	require.NoError(t, err, "Mosaic should work with nil options")

	// Test smartcrop with nil options
	err = img1.Smartcrop(50, 50, nil)
	require.NoError(t, err, "Smartcrop should work with nil options")

	// Test min/max with nil options
	_, err = img1.Min(nil)
	require.NoError(t, err, "Min should work with nil options")

	_, err = img1.Max(nil)
	require.NoError(t, err, "Max should work with nil options")

	// Test draw flood with nil options
	err = img1.DrawFlood([]float64{0, 255, 0}, 25, 25, nil)
	require.NoError(t, err, "DrawFlood should work with nil options")
}

// TestOptionalOutputCodeGeneration tests that the code generation for optional outputs is correct
func TestOptionalOutputCodeGeneration(t *testing.T) {
	// This test verifies that the gint conversion fix is properly implemented
	// by testing the struct generation and type safety

	// Test that mosaic options struct has correct field types
	options := DefaultMosaicOptions()
	require.NotNil(t, options)

	// Verify that dx0/dy0 are int types (not uint32 or other types)
	assert.IsType(t, int(0), options.Dx0, "Dx0 should be int type for signed values")
	assert.IsType(t, int(0), options.Dy0, "Dy0 should be int type for signed values")

	// Test that we can assign negative values (this would fail if they were uint)
	options.Dx0 = -100
	options.Dy0 = -50
	assert.Equal(t, -100, options.Dx0, "Should be able to store negative dx0")
	assert.Equal(t, -50, options.Dy0, "Should be able to store negative dy0")

	t.Log("✓ Mosaic optional output types support signed integers")

	// Test other optional output types for consistency
	smartcropOpts := DefaultSmartcropOptions()
	require.NotNil(t, smartcropOpts)

	// AttentionX/Y should also be int types
	assert.IsType(t, int(0), smartcropOpts.AttentionX, "AttentionX should be int type")
	assert.IsType(t, int(0), smartcropOpts.AttentionY, "AttentionY should be int type")

	t.Log("✓ Smartcrop optional output types are correct")

	// Test min/max position outputs
	minOpts := DefaultMinOptions()
	maxOpts := DefaultMaxOptions()
	require.NotNil(t, minOpts)
	require.NotNil(t, maxOpts)

	assert.IsType(t, int(0), minOpts.X, "Min X should be int type")
	assert.IsType(t, int(0), minOpts.Y, "Min Y should be int type")
	assert.IsType(t, int(0), maxOpts.X, "Max X should be int type")
	assert.IsType(t, int(0), maxOpts.Y, "Max Y should be int type")

	t.Log("✓ Min/Max optional output types are correct")

	// Test draw flood area outputs
	floodOpts := DefaultDrawFloodOptions()
	require.NotNil(t, floodOpts)

	assert.IsType(t, int(0), floodOpts.Left, "Flood Left should be int type")
	assert.IsType(t, int(0), floodOpts.Top, "Flood Top should be int type")
	assert.IsType(t, int(0), floodOpts.Width, "Flood Width should be int type")
	assert.IsType(t, int(0), floodOpts.Height, "Flood Height should be int type")
}
