package vips

import (
	"path/filepath"
	"testing"
)

// gifload.n and friends are plain fields because libvips rejects an explicit zero for
// them, which was measured rather than inferred: a pspec range cannot tell an accepted
// zero from a meaningful one, and webpload.n accepts a zero while gifload.n answers
// "bad page number". The measurements that drove the table in
// internal/generator/presence.go are reproducible with the probe harness recorded there.
//
// These are compile-time pins. If the rule or the table changes so that one of these
// arguments is classified the other way, this file stops building.

var (
	_ = TiffloadOptions{N: 0}               // plain: n=0 is rejected
	_ = TiffloadOptions{Subifd: 0}          // plain: subifd=0 is rejected
	_ = SvgloadOptions{Scale: 0.0}          // plain: scale=0 is rejected
	_ = WebploadOptions{Scale: 0.0}         // plain: scale=0 is rejected
	_ = WebploadOptions{N: Ptr(0)}          // pointer: n=0 is accepted here
	_ = PngsaveOptions{Compression: Ptr(0)} // pointer: uncompressed is real
	_ = WebpsaveOptions{Effort: Ptr(0)}     // pointer
	_ = MagickloadOptions{N: 0}             // plain: n=0 is rejected
	_ = DzsaveOptions{Suffix: ""}           // plain: an empty suffix is rejected
	_ = SimilarityOptions{Scale: 0.0}       // plain: scale=0 is rejected
	_ = HeifloadOptions{N: 0}               // plain: n=0 is rejected ("bad seek")
	_ = PdfloadOptions{N: 0}                // plain: n=0 is rejected ("pages out of range")
	_ = GifloadOptions{N: 0}                // plain: n=0 is rejected
)

// The two that remain pointers are still probeable at run time, and they are the reason
// the rule stays: an explicit zero is a real request for them.
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
	webp := filepath.Join(dir, "in.webp")
	if err := img.Webpsave(webp, nil); err != nil {
		t.Fatal(err)
	}
	if got := outcome(
		func() error {
			i, e := NewWebpload(webp, &WebploadOptions{N: Ptr(0)})
			if i != nil {
				i.Close()
			}
			return e
		}() != nil,
		func() error {
			i, e := NewWebpload(webp, &WebploadOptions{})
			if i != nil {
				i.Close()
			}
			return e
		}() != nil,
	); got != "ok" {
		t.Errorf("webpload.n: got %s, want ok - this is why n is not in the exception table", got)
	}
}
