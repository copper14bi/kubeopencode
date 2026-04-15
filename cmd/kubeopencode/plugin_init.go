// Copyright Contributors to the KubeOpenCode project

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Environment variable names for plugin-init
const (
	envPluginPackages = "PLUGIN_PACKAGES" // JSON array of npm package specifiers
	envPluginDir      = "PLUGIN_DIR"      // Directory to install plugins into (default: /plugins)
)

// Default values for plugin-init
const (
	defaultPluginDir = "/plugins"
)

func init() {
	rootCmd.AddCommand(pluginInitCmd)
}

var pluginInitCmd = &cobra.Command{
	Use:   "plugin-init",
	Short: "Install OpenCode plugin dependencies via npm",
	Long: `plugin-init installs OpenCode plugins and their dependencies into a shared volume.

It runs 'npm install --production' to download plugins from the npm registry.
The installed packages are placed in /plugins/node_modules/ and shared with the
executor container via an emptyDir volume. The executor container does not need
npm installed — it only needs to read the pre-installed packages.

Environment variables:
  PLUGIN_PACKAGES   JSON array of npm package specifiers (required)
                    Example: ["cc-safety-net","@aexol/opencode-tui@^1.0.0"]
  PLUGIN_DIR        Target directory for installation (default: /plugins)`,
	RunE: runPluginInit,
}

func runPluginInit(cmd *cobra.Command, args []string) error {
	// Get plugin packages from environment
	packagesJSON := os.Getenv(envPluginPackages)
	if packagesJSON == "" {
		return fmt.Errorf("%s environment variable is required", envPluginPackages)
	}

	var packages []string
	if err := json.Unmarshal([]byte(packagesJSON), &packages); err != nil {
		return fmt.Errorf("failed to parse %s: %w", envPluginPackages, err)
	}

	if len(packages) == 0 {
		fmt.Println("plugin-init: No plugins to install")
		return nil
	}

	pluginDir := getEnvOrDefault(envPluginDir, defaultPluginDir)

	fmt.Println("plugin-init: Installing OpenCode plugins...")
	fmt.Printf("  Directory: %s\n", pluginDir)
	fmt.Printf("  Packages: %v\n", packages)

	// Ensure plugin directory exists
	if err := os.MkdirAll(pluginDir, 0755); err != nil { //nolint:gosec // Needs group/others access for random UID environments
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Initialize package.json if it doesn't exist
	packageJSONPath := filepath.Join(pluginDir, "package.json")
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		initJSON := []byte(`{"private":true}`)
		if err := os.WriteFile(packageJSONPath, initJSON, 0644); err != nil { //nolint:gosec // Needs to be readable
			return fmt.Errorf("failed to create package.json: %w", err)
		}
	}

	// Run npm install with all packages at once
	npmArgs := []string{"install", "--production", "--no-audit", "--no-fund"}
	npmArgs = append(npmArgs, packages...)

	npmCmd := exec.Command("npm", npmArgs...) //nolint:gosec // args from controlled env var
	npmCmd.Dir = pluginDir
	npmCmd.Stdout = os.Stdout
	npmCmd.Stderr = os.Stderr

	if err := npmCmd.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	// Verify installation
	nodeModulesDir := filepath.Join(pluginDir, "node_modules")
	entries, err := os.ReadDir(nodeModulesDir)
	if err != nil {
		return fmt.Errorf("failed to read node_modules: %w", err)
	}

	fmt.Println("plugin-init: Installation complete!")
	fmt.Printf("  Installed %d top-level packages\n", len(entries))

	// Set permissions for the executor container (which may run as a different user)
	chmodCmd := exec.Command("chmod", "-R", "a+rX", pluginDir) //nolint:gosec // pluginDir from controlled env var
	if err := chmodCmd.Run(); err != nil {
		fmt.Printf("plugin-init: Warning: could not set permissions: %v\n", err)
	}

	return nil
}
