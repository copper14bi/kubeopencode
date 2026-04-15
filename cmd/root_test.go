package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmdHasExpectedUse(t *testing.T) {
	assert.Equal(t, "kubeopencode", rootCmd.Use)
}

func TestRootCmdHasExpectedShortDescription(t *testing.T) {
	assert.Contains(t, rootCmd.Short, "kubeopencode")
}

func TestRootCmdPersistentFlagKubeconfig(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("kubeconfig")
	require.NotNil(t, flag)
	assert.Equal(t, "string", flag.Value.Type())
	assert.Equal(t, "", flag.DefValue)
}

func TestRootCmdPersistentFlagNamespace(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("namespace")
	require.NotNil(t, flag)
	assert.Equal(t, "string", flag.Value.Type())
	assert.Equal(t, "default", flag.DefValue)
}

func TestRootCmdPersistentFlagVerbose(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("verbose")
	require.NotNil(t, flag)
	assert.Equal(t, "bool", flag.Value.Type())
	assert.Equal(t, "false", flag.DefValue)
}

func TestRootCmdPersistentFlagConfig(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, flag)
	assert.Equal(t, "string", flag.Value.Type())
}

func TestRootCmdHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "kubeopencode")
}

func TestExecuteDoesNotPanic(t *testing.T) {
	rootCmd.SetArgs([]string{})
	assert.NotPanics(t, func() {
		_ = rootCmd.Execute()
	})
}
