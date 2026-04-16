// Package version provides utilities for working with semantic versions.
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer represents a parsed semantic version
type SemVer struct {
	Major int
	Minor int
	Patch int
	Pre   string
}

// Parse parses a semantic version string (e.g. "v1.2.3" or "1.2.3-beta").
// Both "v"-prefixed and bare version strings are accepted.
func Parse(v string) (*SemVer, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, "-", 2)
	core := parts[0]
	pre := ""
	if len(parts) == 2 {
		pre = parts[1]
	}

	segments := strings.Split(core, ".")
	if len(segments) != 3 {
		return nil, fmt.Errorf("invalid semver %q: expected major.minor.patch", v)
	}

	major, err := strconv.Atoi(segments[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := strconv.Atoi(segments[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %w", err)
	}
	patch, err := strconv.Atoi(segments[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %w", err)
	}

	// Disallow negative version numbers
	if major < 0 || minor < 0 || patch < 0 {
		return nil, fmt.Errorf("invalid semver %q: version numbers must be non-negative", v)
	}

	return &SemVer{Major: major, Minor: minor, Patch: patch, Pre: pre}, nil
}

// String returns the canonical string representation
func (s *SemVer) String() string {
	base := fmt.Sprintf("v%d.%d.%d", s.Major, s.Minor, s.Patch)
	if s.Pre != "" {
		return base + "-" + s.Pre
	}
	return base
}

// IsRelease returns true if the version has no pre-release suffix
func (s *SemVer) IsRelease() bool {
	return s.Pre == ""
}
