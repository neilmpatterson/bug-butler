package cli

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/neilmpatterson/bug-butler/internal/config"
	"github.com/neilmpatterson/bug-butler/internal/jira"
	"github.com/neilmpatterson/bug-butler/internal/output"
	"github.com/neilmpatterson/bug-butler/internal/sla"
)

var (
	configPath string
	debugMode  bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check bugs against SLA rules",
	Long: `Check fetches bugs from your Jira project and evaluates them against
configured SLA rules to identify bugs that need attention.

Bugs are grouped into buckets based on their SLA compliance status,
with the most urgent violations displayed first.`,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "Path to configuration file")
	checkCmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	// Set log level based on debug flag
	if debugMode {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("Debug mode enabled")
	}

	fmt.Println("🔍 Loading configuration...")

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	projectNames := cfg.Jira.ProjectKeys
	if len(projectNames) > 3 {
		projectNames = append(cfg.Jira.ProjectKeys[:3], fmt.Sprintf("... +%d more", len(cfg.Jira.ProjectKeys)-3))
	}
	fmt.Printf("📋 Projects: %s\n", strings.Join(projectNames, ", "))
	fmt.Printf("📏 SLA Rules: %d configured\n", len(cfg.SLARules))

	slog.Debug("Configuration loaded successfully",
		"jira_url", cfg.Jira.BaseURL,
		"project_count", len(cfg.Jira.ProjectKeys),
		"sla_rules", len(cfg.SLARules),
	)

	fmt.Println("\n🔐 Authenticating with Jira...")

	// Create Jira client
	jiraClient, err := jira.NewClient(cfg.Jira)
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %w", err)
	}

	fmt.Println("✓ Authenticated successfully")
	fmt.Print("\n📥 Fetching bugs...")

	// Fetch bugs from Jira
	bugs, err := jiraClient.FetchBugs()
	if err != nil {
		return fmt.Errorf("failed to fetch bugs: %w", err)
	}

	fmt.Printf(" found %d bugs\n", len(bugs))

	if len(bugs) == 0 {
		fmt.Println("\n✅ No unresolved bugs found!")
		return nil
	}

	fmt.Print("⚖️  Evaluating against SLA rules...")

	// Create SLA evaluator
	evaluator := sla.NewEvaluator(cfg.SLARules)

	// Evaluate bugs against SLA rules
	bucketGroup := evaluator.Evaluate(bugs)

	fmt.Println(" done")

	// Display results
	output.DisplayBuckets(bucketGroup)

	// Exit with error code if there are violations
	if len(bucketGroup.Buckets) > 0 {
		os.Exit(1)
	}

	return nil
}
