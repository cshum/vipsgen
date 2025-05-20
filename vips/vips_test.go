package vips

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
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
	bands := 3 // RGB
	data := make([]byte, width*height*bands)

	// Data is already initialized to zeros (black)

	return NewImageFromMemory(data, width, height, bands)
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
	img, err := NewImageFromBuffer(pngData, nil)
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
	imgFromSource, err := NewImageFromSource(source, nil)
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
	img, err := NewImageFromBuffer(createTestPNG(t, width, height), nil)
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
	// Skip the test if JPEG is not supported
	jpegSupported := false
	for _, mime := range ImageMimeTypes {
		if mime == "image/jpeg" {
			jpegSupported = true
			break
		}
	}
	if !jpegSupported {
		t.Skip("JPEG format not supported, skipping test")
	}

	// Create a simple white image
	img, err := createWhiteImage(100, 80)
	require.NoError(t, err)
	defer img.Close()

	// 1. First save as JPEG with minimal options
	jpegBuf, err := img.JpegsaveBuffer(&JpegsaveBufferOptions{
		Q: 90,
	})
	require.NoError(t, err)
	require.NotEmpty(t, jpegBuf)
	t.Logf("JPEG save produced %d bytes", len(jpegBuf))

	// 2. Load the JPEG
	jpegImg, err := NewImageFromBuffer(jpegBuf, nil)
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
	err = img2.Resize(1.5, nil)
	require.NoError(t, err)
	assert.Equal(t, 75, img2.Width())
	assert.Equal(t, 75, img2.Height())

	// 2. Try to composite images (if supported)
	err = img1.Composite2(img2, BlendModeOver, &Composite2Options{X: 10, Y: 10})
	if err != nil {
		t.Logf("Composite operation failed: %v (may not be supported in this libvips build)", err)
	} else {
		t.Log("Composite operation succeeded")
	}

	// Try to composite array of images (if supported)
	images := []*Image{img1, img2}

	composite, err := NewComposite(images, []BlendMode{BlendModeOver}, &CompositeOptions{X: []int{20}, Y: []int{20}})
	if err != nil {
		t.Logf("Composite2 operation failed: %v (may not be supported in this libvips build)", err)
	} else {
		t.Log("Composite2 operation succeeded")
	}
	composite.Close()
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
	// Skip if we can't load a JPEG file
	jpegSupported := false
	for _, mime := range ImageMimeTypes {
		if mime == "image/jpeg" {
			jpegSupported = true
			break
		}
	}
	if !jpegSupported {
		t.Skip("JPEG format not supported, skipping EXIF test")
	}

	// Create a JPEG with some basic structure
	jpegData := createTestJPEG(t, 120, 80)

	// Load JPEG
	img, err := NewImageFromBuffer(jpegData, nil)
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

// TestAllFormatsSupport tests saving in all supported formats
func TestAllFormatsSupport(t *testing.T) {
	// Create a test image
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Try saving in each supported format
	t.Log("Testing all supported save formats:")

	// 1. PNG
	pngBuf, err := img.PngsaveBuffer(nil)
	if err != nil {
		t.Logf("  - PNG save failed: %v", err)
	} else {
		t.Logf("  - PNG save succeeded: %d bytes", len(pngBuf))
		assert.NotEmpty(t, pngBuf)
	}

	// 2. JPEG
	jpegBuf, err := img.JpegsaveBuffer(nil)
	if err != nil {
		t.Logf("  - JPEG save failed: %v", err)
	} else {
		t.Logf("  - JPEG save succeeded: %d bytes", len(jpegBuf))
		assert.NotEmpty(t, jpegBuf)
	}

	webpBuf, err := img.WebpsaveBuffer(nil)
	if err != nil {
		t.Logf("  - WebP save failed: %v", err)
	} else {
		t.Logf("  - WebP save succeeded: %d bytes", len(webpBuf))
		assert.NotEmpty(t, webpBuf)
	}

	// 4. TIFF (if supported)
	tiffBuf, err := img.TiffsaveBuffer(nil)
	if err != nil {
		t.Logf("  - TIFF save failed: %v", err)
	} else {
		t.Logf("  - TIFF save succeeded: %d bytes", len(tiffBuf))
		assert.NotEmpty(t, tiffBuf)
	}

	gifBuf, err := img.GifsaveBuffer(nil)
	if err != nil {
		t.Logf("  - GIF save failed: %v", err)
	} else {
		t.Logf("  - GIF save succeeded: %d bytes", len(gifBuf))
		assert.NotEmpty(t, gifBuf)
	}
}

// TestErrorHandling tests error handling mechanisms
func TestErrorHandling(t *testing.T) {
	// Test invalid parameter for resize
	img, err := createWhiteImage(100, 100)
	require.NoError(t, err)
	defer img.Close()

	// Try invalid operation (resize by scale factor 0)
	err = img.Resize(0, nil)
	assert.Error(t, err, "Resize by scale factor 0 should fail")

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

// TestImageWithAlpha tests operations on images with alpha channel
func TestImageWithAlpha(t *testing.T) {
	// Create an image with alpha channel
	width, height := 100, 100
	bands := 4 // RGBA
	data := make([]byte, width*height*bands)

	// Fill with semi-transparent pixels
	for i := 0; i < len(data); i += 4 {
		data[i] = 255   // R
		data[i+1] = 255 // G
		data[i+2] = 255 // B
		data[i+3] = 128 // A (semi-transparent)
	}

	img, err := NewImageFromMemory(data, width, height, bands)
	require.NoError(t, err)
	defer img.Close()

	// Check alpha detection
	assert.True(t, img.HasAlpha(), "Should detect alpha channel")

	// Test flatten operation
	err = img.Flatten(&FlattenOptions{
		Background: []float64{0, 0, 0},
	})
	if err != nil {
		t.Logf("Flatten operation failed: %v", err)
	} else {
		t.Log("Flatten operation succeeded")
		assert.False(t, img.HasAlpha(), "Should not have alpha channel after flatten")
	}
}
