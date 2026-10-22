# Fixtures

Inputs libvips can read but cannot write, so a generated round trip cannot make them.

The generated packages look for these at `../testdata/<name>`. A tree produced by the
`vipsgen` command has no such directory, so its subtests skip instead of failing: the
fixtures ship with this repository, not with the generator.

| file | used by | origin |
|---|---|---|
| `sample.pdf` | `pdfload`, `pdfload_source`, `pdfload_buffer` | copied from [imagor](https://github.com/cshum/imagor) `testdata/sample.pdf`, Apache-2.0 |
| `sample.svg` | `svgload`, `svgload_source`, `svgload_buffer` | copied from [imagor](https://github.com/cshum/imagor) `testdata/test.svg`, Apache-2.0 |
| `sample.mat` | `matload` | [scipy](https://github.com/scipy/scipy) `scipy/io/matlab/tests/data/testdouble_7.4_GLNX86.mat`, BSD-3 |
| `analyze.hdr`, `analyze.img` | `analyzeload` | [nibabel](https://github.com/nipy/nibabel) `nibabel/tests/data/analyze.hdr` and `.img`, MIT |
| `sample.exr` | `openexrload` | [openexr-images](https://github.com/AcademySoftwareFoundation/openexr-images) `TestImages/GrayRampsHorizontal.exr`, BSD-3 |
