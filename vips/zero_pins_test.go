package vips

import (
	"path/filepath"
	"testing"
)

// gifload.n and friends are plain fields because libvips rejects an explicit zero for them,
// which was measured rather than inferred: a pspec range cannot tell an accepted zero from a
// meaningful one, and a saver's compression accepts zero as "uncompressed" while a loader's
// page number answers "bad page number". The verdict can also depend on the fixture: n accepts
// zero on a single-page file and rejects it once there is a second page to ask about, so probe
// with a file where the argument can matter. The table is internal/generator/presence.go, and
// AGENTS.md says how to re-measure it.
//
// These are compile-time pins. If the rule or the table changes so that one of these
// arguments is classified the other way, this file stops building. Keep them hand-written:
// generated pins would assert whatever the generator emitted and catch nothing.

var (
	_ = TiffloadOptions{N: 0}               // plain: n=0 is rejected
	_ = TiffloadOptions{Subifd: 0}          // plain: subifd=0 is rejected
	_ = SvgloadOptions{Scale: 0.0}          // plain: scale=0 is rejected
	_ = WebploadOptions{Scale: 0.0}         // plain: scale=0 is rejected
	_ = WebploadOptions{N: 0}               // plain: n=0 is rejected on a multi-page file
	_ = CsvloadOptions{Lines: 0}            // plain: 0 is out of range, dropped silently
	_ = CsvloadSourceOptions{Lines: 0}      // plain: same argument, source form
	_ = PngsaveOptions{Compression: Ptr(0)} // pointer: uncompressed is real
	_ = WebpsaveOptions{Effort: Ptr(0)}     // pointer
	_ = MagickloadOptions{N: 0}             // plain: n=0 is rejected
	_ = DzsaveOptions{Suffix: ""}           // plain: an empty suffix is rejected
	_ = SimilarityOptions{Scale: 0.0}       // plain: scale=0 is rejected
	_ = HeifloadOptions{N: 0}               // plain: n=0 is rejected ("bad seek")
	_ = JxlloadOptions{N: 0}                // plain: n=0 is rejected ("bad page number")
	_ = PdfloadOptions{N: 0}                // plain: n=0 is rejected ("pages out of range")
	_ = GifloadOptions{N: 0}                // plain: n=0 is rejected
)

// The saver zero stays probeable at run time and is the reason the rule exists: there an
// explicit zero is a real request, not an absent value.
func outcome(zero, none bool) string {
	if zero && !none {
		return "ZERO-REJECTED"
	}
	if none {
		return "INCONCLUSIVE"
	}
	return "ok"
}

func TestZeroClassificationPins(t *testing.T) {
	dir := t.TempDir()
	img, err := NewBlack(1, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer img.Close()
	png := filepath.Join(dir, "out.png")
	if got := outcome(
		img.Pngsave(png, &PngsaveOptions{Compression: Ptr(0)}) != nil,
		img.Pngsave(png, &PngsaveOptions{}) != nil,
	); got != "ok" {
		t.Errorf("pngsave.compression: got %s, want ok", got)
	}
}
