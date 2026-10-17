package main

import (
	"fmt"
	"os"

	"github.com/cshum/vipsgen/v2/vips"
)

func verdict(zero, none bool) string {
	if zero && !none { return "ZERO-REJECTED" }
	if none { return "INCONCLUSIVE" }
	return "ok"
}

func main() {
	os.MkdirAll("/tmp/probe", 0o755)
	img, err := vips.NewBlack(1, 1, nil)
	if err != nil { panic(err) }
	defer img.Close()
	_ = err
	_ = func() error { return img.Gifsave("/tmp/probe/in.gif", nil) }()
	_ = func() error { return img.Webpsave("/tmp/probe/in.webp", nil) }()
	_ = func() error { return img.Tiffsave("/tmp/probe/in.tiff", nil) }()
	_ = func() error { return img.Jxlsave("/tmp/probe/in.jxl", nil) }()
	data, _ := os.ReadFile("/tmp/probe/in.webp")
	_ = data

}
