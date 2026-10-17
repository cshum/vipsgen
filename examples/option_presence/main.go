// Command option_presence shows how a vipsgen options struct says "not supplied".
//
// An optional argument is a pointer when libvips accepts its zero value and that zero
// differs from the option's default. nil means "let libvips decide", vips.Ptr(0) means
// "supplied, and the value is zero". Every other field stays a plain value, because
// there the zero value is either not a value libvips accepts or is what libvips would
// have used anyway.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cshum/vipsgen/v2/vips"
)

func main() {
	dir, err := os.MkdirTemp("", "vipsgen-presence")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// noise, so the compression level actually shows up in the output size
	img, err := vips.NewGaussnoise(512, 512, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer img.Close()

	// Compression is *int: 0 means "store uncompressed", which a default of 6 would
	// otherwise hide. Watch what each choice produces.
	var encoded []byte
	for _, tc := range []struct {
		label   string
		options *vips.PngsaveBufferOptions
	}{
		{"omitted", nil},
		{"Ptr(0)", &vips.PngsaveBufferOptions{Compression: vips.Ptr(0)}},
		{"Ptr(6)", &vips.PngsaveBufferOptions{Compression: vips.Ptr(6)}},
		{"Ptr(9)", &vips.PngsaveBufferOptions{Compression: vips.Ptr(9)}},
	} {
		buf, err := img.PngsaveBuffer(tc.options)
		if err != nil {
			log.Fatalf("%s: %v", tc.label, err)
		}
		encoded = buf
		fmt.Printf("compression %-8s %8d bytes\n", tc.label, len(buf))
	}

	// The generated constructor and the field comments both state the defaults, so
	// neither has to be looked up in libvips' documentation:
	//
	//	// Compression Compression factor (range 0-9, default 6)
	//	Compression *int
	fmt.Printf("\nDefaultPngsaveBufferOptions().Compression = %d\n", *vips.DefaultPngsaveBufferOptions().Compression)

	// A plain field needs no Ptr, because zero is not a height libvips accepts
	// (the comment says "range 1-100000000, default 1").
	thumb, err := vips.NewThumbnailBuffer(encoded, 100, &vips.ThumbnailBufferOptions{Height: 80})
	if err != nil {
		log.Fatalf("thumbnail: %v", err)
	}
	defer thumb.Close()
	fmt.Printf("thumbnail of Height: 80 -> %dx%d\n", thumb.Width(), thumb.Height())

	out := filepath.Join(dir, "presence.png")
	if err := thumb.Pngsave(out, &vips.PngsaveOptions{
		Compression: vips.Ptr(0),        // stored, not compressed
		Interlace:   true,               // a plain bool: false is the default
		Background:  []float64{0, 0, 0}, // a plain slice: empty would mean "unset"
	}); err != nil {
		log.Fatalf("save: %v", err)
	}
	fmt.Printf("wrote %s\n", out)
}
