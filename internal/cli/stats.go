package cli

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/neilmpatterson/bug-butler/internal/config"
	"github.com/neilmpatterson/bug-butler/internal/jira"
	"github.com/neilmpatterson/bug-butler/internal/output"
	"github.com/neilmpatterson/bug-butler/internal/stats"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display bug trend statistics over 24 months",
	Long: `Stats fetches bugs created over the past 24 months and displays
trend analysis including monthly creation rates, unresolved bug trends,
priority breakdowns, and progress toward reduction goals.

The stats command shows:
- Sparkline of unresolved bug backlog over time
- Monthly breakdown of created, resolved, and unresolved bugs
- Current month goal tracking (comparing to same month last year)
- Priority distribution trends

This helps track whether your team is making progress on reducing
the overall bug backlog and meeting reduction goals.`,
	RunE: runStats,
}

func init() {
	statsCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "Path to configuration file")
	statsCmd.Flags().BoolVar(&debugMode, "debug", false, "Enable debug logging")
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("📋 Projects: %d configured\n", len(cfg.Jira.ProjectKeys))
	fmt.Printf("📊 Analysis Period: Last %d months\n", cfg.Stats.MonthsToAnalyze)
	fmt.Printf("🎯 Reduction Goal: %.0f%%\n", cfg.Stats.ReductionGoalPercent)

	slog.Debug("Configuration loaded successfully",
		"project_count", len(cfg.Jira.ProjectKeys),
		"months_to_analyze", cfg.Stats.MonthsToAnalyze,
		"reduction_goal", cfg.Stats.ReductionGoalPercent,
	)

	fmt.Println("\n🔐 Authenticating with Jira...")

	// Create Jira client
	jiraClient, err := jira.NewClient(cfg.Jira)
	if err != nil {
		return fmt.Errorf("failed to create Jira client: %w", err)
	}

	fmt.Println("✓ Authenticated successfully")

	// Calculate date range: last N months + current month
	now := time.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// We need to fetch ALL bugs from the beginning to calculate unresolved counts
	// But for display we'll only show the last N months
	// Fetch from 3 years ago to ensure we have enough history
	startDate := currentMonth.AddDate(-3, 0, 0)

	fmt.Printf("\n📥 Fetching bug data...\n")
	fmt.Printf("  Date range: %s to %s\n", startDate.Format("2006-01-02"), now.Format("2006-01-02"))

	// Fetch bugs from Jira
	bugs, err := jiraClient.FetchBugsByDateRange(startDate, now)
	if err != nil {
		return fmt.Errorf("failed to fetch bugs: %w", err)
	}

	fmt.Printf("  Found %d bugs\n", len(bugs))

	if len(bugs) == 0 {
		fmt.Println("\n⚠️  No bug data available for the selected time range")
		return nil
	}

	fmt.Print("\n📈 Analyzing trends...")

	// Create analyzer with config
	analyzer := stats.NewAnalyzer(cfg.Stats.ReductionGoalPercent, cfg.Stats.MonthsToAnalyze)

	// Analyze bugs
	trendStats, err := analyzer.Analyze(bugs)
	if err != nil {
		return fmt.Errorf("failed to analyze trends: %w", err)
	}

	fmt.Println(" done")

	// Display results
	output.DisplayTrendStats(trendStats)

	return nil
}
