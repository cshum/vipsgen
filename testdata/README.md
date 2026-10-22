# Fixtures

Inputs libvips can read but cannot write, so a generated round trip cannot make them.

The generated packages look for these at `../testdata/<name>`. A tree produced by the
`vipsgen` command has no such directory, so its subtests skip instead of failing: the
fixtures ship with this repository, not with the generator.

| file | used by | origin |
|---|---|---|
| `sample.pdf` | `pdfload`, `pdfload_source`, `pdfload_buffer` | written for this repository — one 200x100 page, 502 bytes |
| `sample.svg` | `svgload`, `svgload_source`, `svgload_buffer` | written for this repository — three shapes, 120x80 |
| `sample.mat` | `matload` | [scipy](https://github.com/scipy/scipy) `scipy/io/matlab/tests/data/testdouble_7.4_GLNX86.mat`, BSD-3 |
| `analyze.hdr`, `analyze.img` | `analyzeload` | written here: a 348-byte ANALYZE 7.5 header (MSB first, as the format requires) for an 8x8 uchar image, and its 64 bytes of data |
| `sample.uhdr.jpg` | `uhdrload`, `uhdrload_source`, `uhdrload_buffer` | written here: a 128x128 UltraHDR JPEG made with libultrahdr's `ultrahdr_app` v1.4.0 (encode scenario 4) from a three-band SDR image and a one-band gainmap generated for it, 2771 bytes. No third-party image is inside it |
| `sample.exr` | `openexrload` | [openexr-images](https://github.com/AcademySoftwareFoundation/openexr-images) `TestImages/GrayRampsHorizontal.exr`, BSD-3 |
