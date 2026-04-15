package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestGetBuildInfo(t *testing.T) {
	info := GetBuildInfo()
	if info.Version == "" {
		t.Error("expected Version to be non-empty")
	}
	if info.Commit == "" {
		t.Error("expected Commit to be non-empty")
	}
	if info.BuildDate == "" {
		t.Error("expected BuildDate to be non-empty")
	}
}

func TestVersionCmdHasExpectedUse(t *testing.T) {
	if versionCmd.Use != "version" {
		t.Errorf("expected Use to be 'version', got %q", versionCmd.Use)
	}
}

func TestVersionCmdHasShortFlag(t *testing.T) {
	flag := versionCmd.Flags().Lookup("short")
	if flag == nil {
		t.Fatal("expected --short flag to exist")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected default value 'false', got %q", flag.DefValue)
	}
}

func TestVersionCmdOutputContainsVersion(t *testing.T) {
	origVersion := Version
	Version = "v1.2.3"
	defer func() { Version = origVersion }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})
	_ = rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "v1.2.3") {
		t.Errorf("expected output to contain 'v1.2.3', got: %q", output)
	}
}

func TestVersionCmdShortOutput(t *testing.T) {
	origVersion := Version
	Version = "v0.9.0"
	defer func() { Version = origVersion }()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version", "--short"})
	_ = rootCmd.Execute()

	output := strings.TrimSpace(buf.String())
	if output != "v0.9.0" {
		t.Errorf("expected short output 'v0.9.0', got: %q", output)
	}
}
