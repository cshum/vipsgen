package generator

import "github.com/cshum/vipsgen/internal/introspection"

// Presence describes how a generated optional argument tells libvips whether the
// caller supplied it.
//
// v1 has a single mechanism: a plain Go value field, whose zero value means "not
// supplied" — except for the arguments argPresence marks PresenceZeroAllowed,
// where libvips treats zero as a real value and the wrapper passes it through.
// That exception list is why `PngsaveOptions{}` means compression 0 rather than
// the libvips default, and it is what v2 removes: v2 adds a pointer
// representation, where nil means "not supplied" and a non-nil pointer means the
// caller gave a value, whatever that value is.
type Presence int

const (
	// PresenceValue: the zero value means the caller did not supply the option.
	PresenceValue Presence = iota
	// PresenceZeroAllowed: a zero value is passed through to libvips.
	PresenceZeroAllowed
)

// argPresence classifies one optional input.
//
// v2 replaces the body with the metadata rule — a pointer when the type's zero
// value is one libvips accepts and differs from the recorded default — which
// needs no operation-name lists at all.
func argPresence(op introspection.Operation, opt introspection.Argument) Presence {
	if opt.IsEnum {
		// VIPS_INTENT_PERCEPTUAL is 0 while omitting intent lets the operation
		// choose its own default, so intent is the one enum v1 passes zero for.
		if opt.EnumType == "Intent" {
			return PresenceZeroAllowed
		}
		return PresenceValue
	}
	if opt.GoType != "int" {
		return PresenceValue
	}
	switch opt.Name {
	case "effort":
		// effort=0 is meaningful for the HEIF and WebP savers.
		switch op.Name {
		case "heifsave", "heifsave_buffer", "heifsave_target",
			"webpsave", "webpsave_buffer", "webpsave_mime", "webpsave_target":
			return PresenceZeroAllowed
		}
	case "compression":
		// PNG compression level 0 means "store uncompressed", not "unspecified".
		switch op.Name {
		case "pngsave", "pngsave_buffer", "pngsave_target":
			return PresenceZeroAllowed
		}
	}
	return PresenceValue
}
