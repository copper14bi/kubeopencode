package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information set via ldflags at build time
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// BuildInfo holds version metadata
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

// GetBuildInfo returns the current build information
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Print the version, commit hash, and build date of kubeopencode.`,
	Run: func(cmd *cobra.Command, args []string) {
		short, _ := cmd.Flags().GetBool("short")
		info := GetBuildInfo()
		if short {
			fmt.Println(info.Version)
			return
		}
		fmt.Printf("kubeopencode version %s\n", info.Version)
		fmt.Printf("  commit:     %s\n", info.Commit)
		fmt.Printf("  build date: %s\n", info.BuildDate)
	},
}

func init() {
	versionCmd.Flags().Bool("short", false, "Print only the version number")
	rootCmd.AddCommand(versionCmd)
}
