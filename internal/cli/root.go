package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "bug-butler",
	Short: "Monitor Jira bugs against SLA rules",
	Long: `Bug Butler helps you monitor incoming bugs in Jira projects and
categorize them into buckets based on configurable SLA rules.

It tracks bugs based on priority, status, and time since last activity
to help you identify what needs immediate attention.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("bug-butler v%s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
