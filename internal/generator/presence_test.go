package generator

import (
	"testing"

	"github.com/cshum/vipsgen/internal/introspection"
)

// v1ZeroAllowedReference is the released behaviour, written out independently of
// argPresence so the classifier has to match a specification rather than itself.
// It is the list that existed as shouldIntAllowZero before v2 started.
func v1ZeroAllowedReference(op introspection.Operation, opt introspection.Argument) bool {
	if opt.IsEnum {
		return opt.EnumType == "Intent"
	}
	if opt.GoType != "int" {
		return false
	}
	switch opt.Name {
	case "effort":
		switch op.Name {
		case "heifsave", "heifsave_buffer", "heifsave_target",
			"webpsave", "webpsave_buffer", "webpsave_mime", "webpsave_target":
			return true
		}
	case "compression":
		switch op.Name {
		case "pngsave", "pngsave_buffer", "pngsave_target":
			return true
		}
	}
	return false
}

// TestArgPresenceMatchesV1Allowlist pins the v1 policy to the behaviour that
// shipped: every optional input of every operation the installed libvips exposes
// must classify identically to the hand-written list. It is version-independent —
// the list names operations and enum types, not libvips values — so it runs on
// every libvips in the CI matrix, and any v2 change to the policy has to be
// deliberate rather than incidental.
func TestArgPresenceMatchesV1Allowlist(t *testing.T) {
	introspection := introspection.NewIntrospection(false)
	operations := introspection.DiscoverOperations()
	if len(operations) == 0 {
		t.Fatal("no operations discovered")
	}

	var checked, zeroAllowed int
	for _, op := range operations {
		for _, opt := range op.OptionalInputs {
			checked++
			got := argPresence(op, opt) == PresenceZeroAllowed
			want := v1ZeroAllowedReference(op, opt)
			if got != want {
				t.Errorf("%s.%s: zero-allowed = %v, want %v (v1 behaviour)", op.Name, opt.Name, got, want)
			}
			if want {
				zeroAllowed++
			}
		}
	}
	if checked == 0 {
		t.Fatal("no optional inputs discovered")
	}
	if zeroAllowed == 0 {
		t.Fatal("no zero-allowed optional inputs found; the v1 rule matched nothing")
	}
	t.Logf("optional inputs classified: %d, zero-allowed: %d", checked, zeroAllowed)
}
