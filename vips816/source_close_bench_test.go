package vips

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"testing"
)

// BenchmarkProcessJPEG_NoOps measures end-to-end cost of decoding a JPEG
// through a custom Source and re-encoding through a custom Target — the
// hot path for image-pipeline applications. Each iteration fires many
// goSourceRead callbacks (decode) and goTargetWrite callbacks (encode),
// so it captures the per-callback lock overhead introduced by the
// Source/Target close-vs-callback race fix.
func BenchmarkProcessJPEG_NoOps(b *testing.B) {
	data := makeBenchJPEG(b, 800, 600)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := NewSource(io.NopCloser(bytes.NewReader(data)))
		im, err := NewImageFromSource(src, nil)
		if err != nil || im == nil {
			b.Fatalf("NewImageFromSource: %v", err)
		}
		var out bytes.Buffer
		tgt := NewTarget(benchNopWriteCloser{w: &out})
		if err := im.JpegsaveTarget(tgt, nil); err != nil {
			b.Fatalf("JpegsaveTarget: %v", err)
		}
		tgt.Close()
		im.Close()
		src.Close()
	}
}

// BenchmarkProcessJPEG_Thumbnail measures shrink-on-load — the same path
// as a typical thumbnail-generator service. Triggers many small reads via
// the source callback during the shrink-on-load phase.
func BenchmarkProcessJPEG_Thumbnail(b *testing.B) {
	data := makeBenchJPEG(b, 1600, 1200)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := NewSource(io.NopCloser(bytes.NewReader(data)))
		im, err := NewThumbnailSource(src, 200, &ThumbnailSourceOptions{
			Height: 150,
			Size:   SizeForce,
		})
		if err != nil || im == nil {
			b.Fatalf("NewThumbnailSource: %v", err)
		}
		var out bytes.Buffer
		tgt := NewTarget(benchNopWriteCloser{w: &out})
		if err := im.JpegsaveTarget(tgt, nil); err != nil {
			b.Fatalf("JpegsaveTarget: %v", err)
		}
		tgt.Close()
		im.Close()
		src.Close()
	}
}

// BenchmarkSourceCloseSync isolates the Source.Close + NewSource cycle to
// directly measure the cost of the per-callback sync.Mutex Lock/Unlock
// added by the fix. No actual decode work — just the lifecycle overhead.
func BenchmarkSourceCloseSync(b *testing.B) {
	data := makeBenchJPEG(b, 100, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src := NewSource(io.NopCloser(bytes.NewReader(data)))
		src.Close()
	}
}

type benchNopWriteCloser struct{ w io.Writer }

func (n benchNopWriteCloser) Write(p []byte) (int, error) { return n.w.Write(p) }
func (n benchNopWriteCloser) Close() error                { return nil }

func makeBenchJPEG(b *testing.B, w, h int) []byte {
	b.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 7) % 256),
				G: uint8((y * 13) % 256),
				B: uint8(((x + y) * 3) % 256),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		b.Fatalf("jpeg.Encode: %v", err)
	}
	return buf.Bytes()
}
