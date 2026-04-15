package cmd

import (
	"testing"
)

func TestDeploymentsCmdHasExpectedUse(t *testing.T) {
	cmd := newDeploymentsCmd()
	if cmd.Use != "deployments" {
		t.Errorf("expected use 'deployments', got '%s'", cmd.Use)
	}
}

func TestDeploymentsCmdHasExpectedAliases(t *testing.T) {
	cmd := newDeploymentsCmd()
	aliases := cmd.Aliases
	expected := map[string]bool{"deploy": true, "deployment": true}
	for _, a := range aliases {
		if !expected[a] {
			t.Errorf("unexpected alias: %s", a)
		}
	}
	if len(aliases) != len(expected) {
		t.Errorf("expected %d aliases, got %d", len(expected), len(aliases))
	}
}

func TestDeploymentsCmdHasShortDescription(t *testing.T) {
	cmd := newDeploymentsCmd()
	if cmd.Short == "" {
		t.Error("expected non-empty short description")
	}
}

func TestDeploymentsCmdIsRegistered(t *testing.T) {
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "deployments" {
			return
		}
	}
	t.Error("deployments command not registered on root")
}
