//go:build race

package vips

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"sync"
	"testing"
)

// withSingleThreadedVips pins libvips to 1 worker thread for the duration of
// the test, restoring the previous setting on exit. Single-threaded libvips
// matches the moonglade fleet config where the race was first observed and
// reliably surfaces it under -race; with auto-concurrency libvips finishes
// callbacks fast enough that the race window almost always closes before
// Source.Close runs in the calling goroutine. Implemented in
// race_helpers.go so the cgo call is not in a _test.go file (Go disallows
// cgo in test files when the surrounding package already uses cgo).
func withSingleThreadedVips(t *testing.T) {
	t.Helper()
	prev := raceTestSetConcurrency(1)
	t.Cleanup(func() { raceTestSetConcurrency(prev) })
}

// TestSource_CloseCallbackRace stresses (*Source).Close concurrent with the
// goSourceRead callback fired by libvips while decoding through a custom
// source. On the unpatched library the race detector reports:
//
//	WARNING: DATA RACE
//	Write at 0x... by goroutine N:
//	  vips.(*Source).Close()    connection.go:52   // s.reader = nil
//	Previous read at 0x... by goroutine M:
//	  vips.goSourceRead()       callback.go:32     // source.reader.Read
//
// Source.Close releases s.lock and then writes s.reader = nil outside the
// lock; goSourceRead reads source.reader without holding any lock. After
// libvips has released its synchronous reference, callback "extra"
// goroutines (Go's cgo callback dispatchers) may still be unwinding when
// Close runs — that is the race window.
//
// The trigger pattern is `Close source while a vips.Image keeps libvips'
// reference alive`. shrink-on-load (NewThumbnailSource) reads enough bytes
// to produce the thumbnail synchronously, but libvips may still hold the
// source via the returned image until the image is destroyed; closing the
// source first surfaces the race against the still-pending callback
// goroutine. Build-tagged `race` so the load test is only compiled when
// the race detector is the test verdict.
func TestSource_CloseCallbackRace(t *testing.T) {
	withSingleThreadedVips(t)
	data := makeRaceTestJPEG(t, 800, 600)

	workers := 16
	const perWorker = 50

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for range perWorker {
				// Wrap the bytes reader in io.NopCloser so the Source falls
				// through the non-seekable branch — same shape libvips ends up
				// dispatching to a worker thread for.
				src := NewSource(io.NopCloser(chainReader{r: bytes.NewReader(data)}))
				// NewImageFromSource is lazy — the returned Image is an
				// operation graph that pulls bytes from source on demand.
				// JpegsaveTarget below forces the pull, firing goSourceRead
				// callbacks on libvips worker goroutines. Closing the source
				// while the image still holds the libvips reference is the
				// race window.
				im, err := NewImageFromSource(src, nil)
				if err != nil || im == nil {
					src.Close()
					continue
				}
				var out bytes.Buffer
				tgt := NewTarget(nopWriteCloser{w: &out})
				// Close source first — image is still alive, libvips
				// will still try to read through goSourceRead during
				// JpegsaveTarget below.
				src.Close()
				_ = im.JpegsaveTarget(tgt, nil)
				tgt.Close()
				im.Close()
			}
		}()
	}
	wg.Wait()
}

// TestTarget_CloseCallbackWriteRace mirrors the source test for Target.Close
// vs goTargetWrite. Source is closed last so the source side of the
// pipeline doesn't interfere; we close the Target while libvips may still
// be flushing through goTargetWrite from JpegsaveTarget.
func TestTarget_CloseCallbackWriteRace(t *testing.T) {
	withSingleThreadedVips(t)
	data := makeRaceTestJPEG(t, 800, 600)

	workers := 16
	const perWorker = 50

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for range perWorker {
				src := NewSource(io.NopCloser(chainReader{r: bytes.NewReader(data)}))
				im, err := NewThumbnailSource(src, 100, nil)
				if err != nil || im == nil {
					src.Close()
					continue
				}
				var out bytes.Buffer
				tgt := NewTarget(nopWriteCloser{w: &out})
				_ = im.JpegsaveTarget(tgt, nil)
				// Close target BEFORE im.Close so libvips can still hold
				// the target ref via the image during the unwind.
				tgt.Close()
				im.Close()
				src.Close()
			}
		}()
	}
	wg.Wait()
}

type nopWriteCloser struct{ w io.Writer }

func (n nopWriteCloser) Write(p []byte) (int, error) { return n.w.Write(p) }
func (n nopWriteCloser) Close() error                { return nil }

// chainReader wraps an io.Reader to mimic the wrapping-chain shape that
// real-world callers produce (validating wrappers, decompression streams,
// sniff buffers). The extra indirection slows Read just enough that the
// cgo callback goroutine has not finished unwinding when Source.Close runs
// in the calling goroutine — which is the race window. bytes.Reader.Read
// is so fast the window almost never opens.
type chainReader struct{ r io.Reader }

func (c chainReader) Read(p []byte) (int, error) { return c.r.Read(p) }

func makeRaceTestJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
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
		t.Fatalf("jpeg.Encode: %v", err)
	}
	return buf.Bytes()
}
