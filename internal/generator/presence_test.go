package generator

import (
	"testing"

	"github.com/cshum/vipsgen/v2/internal/introspection"
)

// TestArgPresenceFollowsMetadata checks the v2 rule against the metadata of every
// operation the installed libvips exposes, and pins the invariants that make the
// pointer fields safe:
//
//   - a pointer field's type zero is a value libvips accepts, so Ptr(0) is a
//     meaningful request rather than an error;
//   - a pointer field's zero differs from the argument's default, so the two cases
//     the field exists to tell apart really are different;
//   - nothing that carries a pointer, image, array, source or target becomes a
//     pointer, because presence for those has no separate meaning.
//
// Values from libvips are not pinned, so a libvips upgrade shows up as different
// counts rather than a failure.
func TestArgPresenceFollowsMetadata(t *testing.T) {
	i := introspection.NewIntrospection(false)
	operations := i.DiscoverOperations()
	if len(operations) == 0 {
		t.Fatal("no operations discovered")
	}

	var pointer, value int
	for _, op := range operations {
		for _, opt := range op.OptionalInputs {
			switch argPresence(opt) {
			case PresencePointer:
				pointer++
				if !zeroValueAccepted(opt) {
					t.Errorf("%s.%s is a pointer but its zero value is not one libvips accepts", op.Name, opt.Name)
				}
				if isZeroDefault(opt.DefaultValue) {
					t.Errorf("%s.%s is a pointer but its zero value equals its default", op.Name, opt.Name)
				}
				if opt.IsArray || opt.IsImage || opt.IsSource || opt.IsTarget || opt.IsOutput {
					t.Errorf("%s.%s is a pointer but does not hold a scalar", op.Name, opt.Name)
				}
			case PresenceValue:
				value++
			}
		}
	}
	if pointer == 0 {
		t.Fatal("no pointer optional inputs found; the rule matched nothing")
	}
	if value == 0 {
		t.Fatal("no value optional inputs found; the rule matched everything")
	}
	t.Logf("optional inputs: %d pointer, %d value", pointer, value)
}

// TestArgPresenceSpotsTheZeroValueCases checks the arguments where v1 needed a
// hand-written exception, on operations present in every supported libvips.
func TestArgPresenceSpotsTheZeroValueCases(t *testing.T) {
	i := introspection.NewIntrospection(false)
	operations := i.DiscoverOperations()

	find := func(opName, argName string) (introspection.Argument, bool) {
		for _, op := range operations {
			if op.Name != opName {
				continue
			}
			for _, opt := range op.OptionalInputs {
				if opt.Name == argName {
					return opt, true
				}
			}
		}
		return introspection.Argument{}, false
	}

	// libvips accepts 0 for these, and their defaults are non-zero, so they have to
	// be pointers: this is exactly what the v1 allowlist was compensating for.
	for _, want := range []struct{ op, arg string }{
		{"webpsave_buffer", "effort"},
		{"pngsave_buffer", "compression"},
		{"thumbnail", "intent"},
	} {
		opt, ok := find(want.op, want.arg)
		if !ok {
			t.Fatalf("%s.%s not found; libvips changed", want.op, want.arg)
		}
		if argPresence(opt) != PresencePointer {
			t.Errorf("%s.%s should be a pointer (zero is a value, default is %v)", want.op, want.arg, opt.DefaultValue)
		}
	}

	// A boolean whose default is false has no ambiguity: false is the default.
	if opt, ok := find("thumbnail", "no_rotate"); ok {
		if argPresence(opt) != PresenceValue {
			t.Errorf("thumbnail.no_rotate should stay a value (default %v)", opt.DefaultValue)
		}
	}
	// A string with an empty default has no ambiguity either.
	if opt, ok := find("pngsave", "profile"); ok {
		if argPresence(opt) != PresenceValue {
			t.Errorf("pngsave.profile should stay a value (default %q)", opt.DefaultValue)
		}
	}
}
