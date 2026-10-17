package zeroprobe

import (
	"path/filepath"
	"testing"

	"github.com/cshum/vipsgen/v2/vips"
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
	_ = vips.TiffloadOptions{N: 0}                    // plain: n=0 is rejected
	_ = vips.TiffloadOptions{Subifd: 0}               // plain: subifd=0 is rejected
	_ = vips.SvgloadOptions{Scale: 0.0}               // plain: scale=0 is rejected
	_ = vips.WebploadOptions{Scale: 0.0}              // plain: scale=0 is rejected
	_ = vips.WebploadOptions{N: vips.Ptr(0)}          // pointer: n=0 is accepted here
	_ = vips.PngsaveOptions{Compression: vips.Ptr(0)} // pointer: uncompressed is real
	_ = vips.WebpsaveOptions{Effort: vips.Ptr(0)}     // pointer
	_ = vips.GifloadOptions{N: 0}                     // plain: n=0 is rejected
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

func TestExplicitZeroIsAcceptedWhereTheRuleSaysPointer(t *testing.T) {
	dir := t.TempDir()
	img, err := vips.NewBlack(1, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	png := filepath.Join(dir, "out.png")
	if got := outcome(
		img.Pngsave(png, &vips.PngsaveOptions{Compression: vips.Ptr(0)}) != nil,
		img.Pngsave(png, &vips.PngsaveOptions{}) != nil,
	); got != "ok" {
		t.Errorf("pngsave.compression: got %s, want ok", got)
	}
	webp := filepath.Join(dir, "in.webp")
	if err := img.Webpsave(webp, nil); err != nil {
		t.Fatal(err)
	}
	if got := outcome(
		func() error { _, e := vips.NewWebpload(webp, &vips.WebploadOptions{N: vips.Ptr(0)}); return e }() != nil,
		func() error { _, e := vips.NewWebpload(webp, &vips.WebploadOptions{}); return e }() != nil,
	); got != "ok" {
		t.Errorf("webpload.n: got %s, want ok - this is why n is not in the exception table", got)
	}
}
