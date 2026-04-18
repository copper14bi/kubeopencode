package version

import (
	"testing"
)

func TestParse_ValidVersion(t *testing.T) {
	v, err := Parse("v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Major != 1 || v.Minor != 2 || v.Patch != 3 {
		t.Errorf("unexpected parsed values: %+v", v)
	}
	if v.Pre != "" {
		t.Errorf("expected empty pre-release, got %q", v.Pre)
	}
}

func TestParse_WithPreRelease(t *testing.T) {
	v, err := Parse("v2.0.0-beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Pre != "beta" {
		t.Errorf("expected pre-release 'beta', got %q", v.Pre)
	}
	if v.IsRelease() {
		t.Error("expected IsRelease() to be false for pre-release version")
	}
}

func TestParse_WithPreRelease_RC(t *testing.T) {
	// Added: also verify rc (release candidate) is treated as pre-release
	v, err := Parse("v1.0.0-rc1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Pre != "rc1" {
		t.Errorf("expected pre-release 'rc1', got %q", v.Pre)
	}
	if v.IsRelease() {
		t.Error("expected IsRelease() to be false for rc pre-release version")
	}
}

func TestParse_WithoutVPrefix(t *testing.T) {
	v, err := Parse("3.4.5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Major != 3 || v.Minor != 4 || v.Patch != 5 {
		t.Errorf("unexpected parsed values: %+v", v)
	}
}

func TestParse_InvalidVersion(t *testing.T) {
	cases := []string{"dev", "none", "1.2", "1.2.3.4", ""}
	for _, c := range cases {
		_, err := Parse(c)
		if err == nil {
			t.Errorf("expected error for input %q, got nil", c)
		}
	}
}

func TestSemVer_String(t *testing.T) {
	v := &SemVer{Major: 1, Minor: 0, Patch: 0}
	if v.String() != "v1.0.0" {
		t.Errorf("expected 'v1.0.0', got %q", v.String())
	}

	v.Pre = "alpha"
	if v.String() != "v1.0.0-alpha" {
		t.Errorf("expected 'v1.0.0-alpha', got %q", v.String())
	}
}

func TestIsRelease(t *testing.T) {
	v := &SemVer{Major: 1, Minor: 2, Patch: 3}
	if !v.IsRelease() {
		t.Error("expected IsRelease() to be true")
	}
}
