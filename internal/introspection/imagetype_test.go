package introspection

import "testing"

// TestNormalizeFormatName pins what an operation name contributes to a format: the aliases
// libvips spells differently from its loaders, and the names that are not formats at all.
func TestNormalizeFormatName(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"png", "png"},
		{"JPG", "jpeg"},
		{"tif", "tiff"},
		{"j2k", "jp2k"},
		{"jp2", "jp2k"},
		{"matlab", "mat"},
		{"nifti", "nii"},
		{"foreignload", "load"},
		{"", ""},
		{"profile", ""},
		{"foreign", ""},
		{"icc", ""},
		{"colourspace", ""},
		{"colorspace", ""},
		{"trailing_", ""},
	} {
		if got := normalizeFormatName(c.in); got != c.want {
			t.Errorf("normalizeFormatName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestGetMimeType pins the MIME type an image type reports, which is what a caller serving
// a format needs, and that an unknown format reports nothing rather than a guess.
func TestGetMimeType(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"jpeg", "image/jpeg"},
		{"JPEG", "image/jpeg"},
		{"png", "image/png"},
		{"pdf", "application/pdf"},
		{"notaformat", ""},
	} {
		if got := getMimeType(c.in); got != c.want {
			t.Errorf("getMimeType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestCachedCString pins that the same name maps to the same C string, which is what makes
// the cache worth having, and that different names do not collide.
func TestCachedCString(t *testing.T) {
	first := cachedCString("VipsImage")
	again := cachedCString("VipsImage")
	other := cachedCString("VipsOperation")

	if first != again {
		t.Error("the same name should return the cached pointer")
	}
	if first == other {
		t.Error("different names should not share a pointer")
	}
}
