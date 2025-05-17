package main

import (
	"github.com/cshum/vipsgen/vips"
	"net/http"
	"os"
)

func main() {
	// manipulate images using libvips C bindings
	vips.Startup(nil)
	defer vips.Shutdown()

	// create source from io.ReadCloser
	resp, err := http.Get("https://raw.githubusercontent.com/cshum/imagor/master/testdata/dancing-banana.gif")
	if err != nil {
		panic(err)
	}
	source := vips.NewSource(resp.Body)
	defer source.Close() // source needs to remain available during the lifetime of image

	params := vips.NewLoadParams()
	params.NumPages.Set(-1)                               // enable animation
	image, err := vips.NewImageFromSource(source, params) // load image from source
	if err != nil {
		panic(err)
	}
	defer image.Close()
	if err = image.ExtractArea(30, 40, 50, 70); err != nil {
		panic(err)
	}
	if err = image.Flatten(); err != nil {
		panic(err)
	}
	buf, err := image.GifsaveBuffer()
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile("dancing-banana.gif", buf, 0666); err != nil {
		panic(err)
	}
}
