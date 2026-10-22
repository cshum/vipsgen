# Fixtures

Inputs libvips can read but cannot write, so a generated round trip cannot make them.

The generated packages look for these at `../testdata/<name>`. A tree produced by the
`vipsgen` command has no such directory, so its subtests skip instead of failing: the
fixtures ship with this repository, not with the generator.

| file | used by | origin |
|---|---|---|
| `sample.pdf` | `pdfload`, `pdfload_source`, `pdfload_buffer` | copied from [imagor](https://github.com/cshum/imagor) `testdata/sample.pdf`, Apache-2.0 |
| `sample.svg` | `svgload`, `svgload_source`, `svgload_buffer` | copied from [imagor](https://github.com/cshum/imagor) `testdata/test.svg`, Apache-2.0 |
