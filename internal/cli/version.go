package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the version, build time, and other build information for llmconf.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println()
		fmt.Println("llmconf - LLM Provider Configuration CLI")
		fmt.Println()
		fmt.Printf("Version:    %s\n", Version)
		fmt.Printf("Build Time: %s\n", BuildTime)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Println()
	},
}
