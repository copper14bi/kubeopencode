package cmd

import (
	"testing"
)

func TestServicesCmdHasExpectedUse(t *testing.T) {
	cmd := newServicesCmd()
	if cmd.Use != "services" {
		t.Errorf("expected Use 'services', got '%s'", cmd.Use)
	}
}

func TestServicesCmdHasExpectedAliases(t *testing.T) {
	cmd := newServicesCmd()
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "svc" {
		t.Errorf("expected aliases [svc], got %v", cmd.Aliases)
	}
}

func TestServicesCmdHasShortDescription(t *testing.T) {
	cmd := newServicesCmd()
	if cmd.Short == "" {
		t.Error("expected non-empty short description")
	}
}

func TestServicesCmdIsRegistered(t *testing.T) {
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "services" {
			return
		}
	}
	t.Error("services command not registered on rootCmd")
}
