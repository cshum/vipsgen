package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/cshum/vipsgen/v2/vips"
)

func verdict(zero, none bool) string {
	if zero && !none { return "ZERO-REJECTED" }
	if none { return "INCONCLUSIVE" }
	return "ok"
}

var fileData []byte

func newSrc() *vips.Source { return vips.NewSource(io.NopCloser(bytes.NewReader(fileData))) }

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
	fileData, _ = os.ReadFile("/tmp/probe/in.webp")
	data := fileData
	_ = data
	{
		zErr := func() error { return img.Gifsave("/tmp/probe/out.gif", &vips.GifsaveOptions{Dither: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Gifsave("/tmp/probe/out.gif", &vips.GifsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "gifsave", "dither", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Gifsave("/tmp/probe/out.gif", &vips.GifsaveOptions{InterpaletteMaxerror: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Gifsave("/tmp/probe/out.gif", &vips.GifsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "gifsave", "interpalette_maxerror", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Gifsave("/tmp/probe/out.gif", &vips.GifsaveOptions{Keep: vips.Ptr(vips.Keep(0))}) }()
		nErr := func() error { return img.Gifsave("/tmp/probe/out.gif", &vips.GifsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "gifsave", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{Dither: vips.Ptr(0.0)}); return e }()
		nErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "gifsave_buffer", "dither", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{InterpaletteMaxerror: vips.Ptr(0.0)}); return e }()
		nErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "gifsave_buffer", "interpalette_maxerror", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "gifsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{Q: vips.Ptr(0)}) }()
		nErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave", "Q", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{AlphaQ: vips.Ptr(0)}) }()
		nErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave", "alpha_q", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{Effort: vips.Ptr(0)}) }()
		nErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave", "effort", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{Keep: vips.Ptr(vips.Keep(0))}) }()
		nErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{Kmax: vips.Ptr(0)}) }()
		nErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave", "kmax", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{Kmin: vips.Ptr(0)}) }()
		nErr := func() error { return img.Webpsave("/tmp/probe/out.webp", &vips.WebpsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave", "kmin", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{Q: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "Q", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{AlphaQ: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "alpha_q", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{Effort: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "effort", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{Kmax: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "kmax", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{Kmin: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "kmin", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := vips.NewCsvload("/tmp/probe/sample.csv", &vips.CsvloadOptions{Lines: vips.Ptr(0)}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := vips.NewCsvload("/tmp/probe/sample.csv", &vips.CsvloadOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "csvload", "lines", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := vips.NewCsvload("/tmp/probe/sample.csv", &vips.CsvloadOptions{Separator: vips.Ptr("")}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := vips.NewCsvload("/tmp/probe/sample.csv", &vips.CsvloadOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "csvload", "separator", "string", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := vips.NewCsvload("/tmp/probe/sample.csv", &vips.CsvloadOptions{Whitespace: vips.Ptr("")}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := vips.NewCsvload("/tmp/probe/sample.csv", &vips.CsvloadOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "csvload", "whitespace", "string", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := vips.NewCsvloadSource(newSrc(), &vips.CsvloadSourceOptions{Lines: vips.Ptr(0)}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := vips.NewCsvloadSource(newSrc(), &vips.CsvloadSourceOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "csvload_source", "lines", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := vips.NewCsvloadSource(newSrc(), &vips.CsvloadSourceOptions{Separator: vips.Ptr("")}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := vips.NewCsvloadSource(newSrc(), &vips.CsvloadSourceOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "csvload_source", "separator", "string", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := vips.NewCsvloadSource(newSrc(), &vips.CsvloadSourceOptions{Whitespace: vips.Ptr("")}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := vips.NewCsvloadSource(newSrc(), &vips.CsvloadSourceOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "csvload_source", "whitespace", "string", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := vips.NewJxlload("/tmp/probe/in.jxl", &vips.JxlloadOptions{N: vips.Ptr(0)}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := vips.NewJxlload("/tmp/probe/in.jxl", &vips.JxlloadOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlload", "n", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := vips.NewJxlloadBuffer(data, &vips.JxlloadBufferOptions{N: vips.Ptr(0)}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := vips.NewJxlloadBuffer(data, &vips.JxlloadBufferOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlload_buffer", "n", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := vips.NewJxlloadSource(newSrc(), &vips.JxlloadSourceOptions{N: vips.Ptr(0)}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := vips.NewJxlloadSource(newSrc(), &vips.JxlloadSourceOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlload_source", "n", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Jxlsave("/tmp/probe/out.jxl", &vips.JxlsaveOptions{Q: vips.Ptr(0)}) }()
		nErr := func() error { return img.Jxlsave("/tmp/probe/out.jxl", &vips.JxlsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlsave", "Q", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Jxlsave("/tmp/probe/out.jxl", &vips.JxlsaveOptions{Distance: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Jxlsave("/tmp/probe/out.jxl", &vips.JxlsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlsave", "distance", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Jxlsave("/tmp/probe/out.jxl", &vips.JxlsaveOptions{Keep: vips.Ptr(vips.Keep(0))}) }()
		nErr := func() error { return img.Jxlsave("/tmp/probe/out.jxl", &vips.JxlsaveOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlsave", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{Q: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlsave_buffer", "Q", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{Distance: vips.Ptr(0.0)}); return e }()
		nErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlsave_buffer", "distance", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := vips.NewMagickload("/tmp/probe/sample.svg", &vips.MagickloadOptions{N: vips.Ptr(0)}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := vips.NewMagickload("/tmp/probe/sample.svg", &vips.MagickloadOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "magickload", "n", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Affine(1.0, 1.0, 1.0, 1.0, &vips.AffineOptions{Extend: vips.Ptr(vips.Extend(0))}) }()
		nErr := func() error { return img.Affine(1.0, 1.0, 1.0, 1.0, &vips.AffineOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "affine", "extend", "Extend", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Canny(&vips.CannyOptions{Precision: vips.Ptr(vips.Precision(0))}) }()
		nErr := func() error { return img.Canny(&vips.CannyOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "canny", "precision", "Precision", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Clamp(&vips.ClampOptions{Max: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Clamp(&vips.ClampOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "clamp", "max", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.DzsaveBuffer(&vips.DzsaveBufferOptions{Id: vips.Ptr("")}); return e }()
		nErr := func() error { _, e := img.DzsaveBuffer(&vips.DzsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "dzsave_buffer", "id", "string", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.DzsaveBuffer(&vips.DzsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.DzsaveBuffer(&vips.DzsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "dzsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.DzsaveBuffer(&vips.DzsaveBufferOptions{Overlap: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.DzsaveBuffer(&vips.DzsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "dzsave_buffer", "overlap", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.DzsaveBuffer(&vips.DzsaveBufferOptions{SkipBlanks: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.DzsaveBuffer(&vips.DzsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "dzsave_buffer", "skip_blanks", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.DzsaveBuffer(&vips.DzsaveBufferOptions{Suffix: vips.Ptr("")}); return e }()
		nErr := func() error { _, e := img.DzsaveBuffer(&vips.DzsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "dzsave_buffer", "suffix", "string", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Flatten(&vips.FlattenOptions{MaxAlpha: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Flatten(&vips.FlattenOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "flatten", "max_alpha", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{Dither: vips.Ptr(0.0)}); return e }()
		nErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "gifsave_buffer", "dither", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{InterpaletteMaxerror: vips.Ptr(0.0)}); return e }()
		nErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "gifsave_buffer", "interpalette_maxerror", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.GifsaveBuffer(&vips.GifsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "gifsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.HeifsaveBuffer(&vips.HeifsaveBufferOptions{Effort: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.HeifsaveBuffer(&vips.HeifsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "heifsave_buffer", "effort", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.HeifsaveBuffer(&vips.HeifsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.HeifsaveBuffer(&vips.HeifsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "heifsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.HistEqual(&vips.HistEqualOptions{Band: vips.Ptr(0)}) }()
		nErr := func() error { return img.HistEqual(&vips.HistEqualOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "hist_equal", "band", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.HistFind(&vips.HistFindOptions{Band: vips.Ptr(0)}) }()
		nErr := func() error { return img.HistFind(&vips.HistFindOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "hist_find", "band", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.IccExport(&vips.IccExportOptions{Intent: vips.Ptr(vips.Intent(0))}) }()
		nErr := func() error { return img.IccExport(&vips.IccExportOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "icc_export", "intent", "Intent", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.IccImport(&vips.IccImportOptions{Intent: vips.Ptr(vips.Intent(0))}) }()
		nErr := func() error { return img.IccImport(&vips.IccImportOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "icc_import", "intent", "Intent", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.Jp2ksaveBuffer(&vips.Jp2ksaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.Jp2ksaveBuffer(&vips.Jp2ksaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jp2ksave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.Jp2ksaveBuffer(&vips.Jp2ksaveBufferOptions{SubsampleMode: vips.Ptr(vips.Subsample(0))}); return e }()
		nErr := func() error { _, e := img.Jp2ksaveBuffer(&vips.Jp2ksaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jp2ksave_buffer", "subsample_mode", "Subsample", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.JpegsaveBuffer(&vips.JpegsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.JpegsaveBuffer(&vips.JpegsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jpegsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{Q: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlsave_buffer", "Q", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{Distance: vips.Ptr(0.0)}); return e }()
		nErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlsave_buffer", "distance", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.JxlsaveBuffer(&vips.JxlsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "jxlsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.MagicksaveBuffer(&vips.MagicksaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.MagicksaveBuffer(&vips.MagicksaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "magicksave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Matrixprint(&vips.MatrixprintOptions{Keep: vips.Ptr(vips.Keep(0))}) }()
		nErr := func() error { return img.Matrixprint(&vips.MatrixprintOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "matrixprint", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Msb(&vips.MsbOptions{Band: vips.Ptr(0)}) }()
		nErr := func() error { return img.Msb(&vips.MsbOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "msb", "band", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.PngsaveBuffer(&vips.PngsaveBufferOptions{Q: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.PngsaveBuffer(&vips.PngsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "pngsave_buffer", "Q", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.PngsaveBuffer(&vips.PngsaveBufferOptions{Compression: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.PngsaveBuffer(&vips.PngsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "pngsave_buffer", "compression", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.PngsaveBuffer(&vips.PngsaveBufferOptions{Dither: vips.Ptr(0.0)}); return e }()
		nErr := func() error { _, e := img.PngsaveBuffer(&vips.PngsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "pngsave_buffer", "dither", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.PngsaveBuffer(&vips.PngsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.PngsaveBuffer(&vips.PngsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "pngsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Premultiply(&vips.PremultiplyOptions{MaxAlpha: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Premultiply(&vips.PremultiplyOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "premultiply", "max_alpha", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.RadsaveBuffer(&vips.RadsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.RadsaveBuffer(&vips.RadsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "radsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.RawsaveBuffer(&vips.RawsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.RawsaveBuffer(&vips.RawsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "rawsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Reduce(1.0, 1.0, &vips.ReduceOptions{Kernel: vips.Ptr(vips.Kernel(0))}) }()
		nErr := func() error { return img.Reduce(1.0, 1.0, &vips.ReduceOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "reduce", "kernel", "Kernel", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Reduceh(1.0, &vips.ReducehOptions{Kernel: vips.Ptr(vips.Kernel(0))}) }()
		nErr := func() error { return img.Reduceh(1.0, &vips.ReducehOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "reduceh", "kernel", "Kernel", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Reducev(1.0, &vips.ReducevOptions{Kernel: vips.Ptr(vips.Kernel(0))}) }()
		nErr := func() error { return img.Reducev(1.0, &vips.ReducevOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "reducev", "kernel", "Kernel", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Resize(1.0, &vips.ResizeOptions{Gap: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Resize(1.0, &vips.ResizeOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "resize", "gap", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Resize(1.0, &vips.ResizeOptions{Kernel: vips.Ptr(vips.Kernel(0))}) }()
		nErr := func() error { return img.Resize(1.0, &vips.ResizeOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "resize", "kernel", "Kernel", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Rot45(&vips.Rot45Options{Angle: vips.Ptr(vips.Angle45(0))}) }()
		nErr := func() error { return img.Rot45(&vips.Rot45Options{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "rot45", "angle", "Angle45", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Sharpen(&vips.SharpenOptions{M2: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Sharpen(&vips.SharpenOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "sharpen", "m2", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Sharpen(&vips.SharpenOptions{X1: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Sharpen(&vips.SharpenOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "sharpen", "x1", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Sharpen(&vips.SharpenOptions{Y2: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Sharpen(&vips.SharpenOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "sharpen", "y2", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Sharpen(&vips.SharpenOptions{Y3: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Sharpen(&vips.SharpenOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "sharpen", "y3", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Similarity(&vips.SimilarityOptions{Scale: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Similarity(&vips.SimilarityOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "similarity", "scale", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Smartcrop(1, 1, &vips.SmartcropOptions{Interesting: vips.Ptr(vips.Interesting(0))}) }()
		nErr := func() error { return img.Smartcrop(1, 1, &vips.SmartcropOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "smartcrop", "interesting", "Interesting", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Stdif(1, 1, &vips.StdifOptions{A: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Stdif(1, 1, &vips.StdifOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "stdif", "a", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Stdif(1, 1, &vips.StdifOptions{B: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Stdif(1, 1, &vips.StdifOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "stdif", "b", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Stdif(1, 1, &vips.StdifOptions{M0: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Stdif(1, 1, &vips.StdifOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "stdif", "m0", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Stdif(1, 1, &vips.StdifOptions{S0: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Stdif(1, 1, &vips.StdifOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "stdif", "s0", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.ThumbnailImage(1, &vips.ThumbnailImageOptions{Intent: vips.Ptr(vips.Intent(0))}) }()
		nErr := func() error { return img.ThumbnailImage(1, &vips.ThumbnailImageOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "thumbnail_image", "intent", "Intent", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.TiffsaveBuffer(&vips.TiffsaveBufferOptions{Depth: vips.Ptr(vips.DzDepth(0))}); return e }()
		nErr := func() error { _, e := img.TiffsaveBuffer(&vips.TiffsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "tiffsave_buffer", "depth", "DzDepth", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.TiffsaveBuffer(&vips.TiffsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.TiffsaveBuffer(&vips.TiffsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "tiffsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { p, e := img.Tilecache(&vips.TilecacheOptions{MaxTiles: vips.Ptr(0)}); if p != nil { p.Close() }; return e }()
		nErr := func() error { p, e := img.Tilecache(&vips.TilecacheOptions{}); if p != nil { p.Close() }; return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "tilecache", "max_tiles", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Unpremultiply(&vips.UnpremultiplyOptions{AlphaBand: vips.Ptr(0)}) }()
		nErr := func() error { return img.Unpremultiply(&vips.UnpremultiplyOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "unpremultiply", "alpha_band", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { return img.Unpremultiply(&vips.UnpremultiplyOptions{MaxAlpha: vips.Ptr(0.0)}) }()
		nErr := func() error { return img.Unpremultiply(&vips.UnpremultiplyOptions{}) }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "unpremultiply", "max_alpha", "float64", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{Q: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "Q", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{AlphaQ: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "alpha_q", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{Effort: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "effort", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{Keep: vips.Ptr(vips.Keep(0))}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "keep", "Keep", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{Kmax: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "kmax", "int", verdict(zErr != nil, nErr != nil))
	}
	{
		zErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{Kmin: vips.Ptr(0)}); return e }()
		nErr := func() error { _, e := img.WebpsaveBuffer(&vips.WebpsaveBufferOptions{}); return e }()
		fmt.Printf("%s\t%s\t%s\t%s\n", "webpsave_buffer", "kmin", "int", verdict(zErr != nil, nErr != nil))
	}
}
