package output

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/neilmpatterson/bug-butler/internal/domain"
)

// DisplayTrendStats renders the complete trend statistics report
func DisplayTrendStats(stats *domain.TrendStats) {
	if len(stats.MonthlyData) == 0 {
		fmt.Println("\n⚠️  No bug data available for the selected time range")
		return
	}

	displayHeader()
	displayUnresolvedSparkline(stats.MonthlyData)
	displayMonthlyTable(stats.MonthlyData)
	displayGoalProgress(stats)
	displayPriorityBreakdown(stats.MonthlyData)
	displaySprintStats(stats.SprintStats)
}

// displayHeader prints the report header
func displayHeader() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("  BUG BUTLER - TREND STATISTICS")
	fmt.Println(strings.Repeat("=", 80))
}

// displayUnresolvedSparkline shows a sparkline of unresolved bug counts
func displayUnresolvedSparkline(monthly []domain.MonthlyBugStats) {
	if len(monthly) == 0 {
		return
	}

	fmt.Println("\n📈 Unresolved Bug Backlog Trend (Last 24 Months)")

	// Extract unresolved counts
	values := make([]int, len(monthly))
	for i, m := range monthly {
		values[i] = m.TotalUnresolved
	}

	// Generate sparkline
	sparkline := generateSparkline(values)
	fmt.Printf("\n%s\n", sparkline)

	// Show first, middle, and last months with counts
	if len(monthly) >= 3 {
		first := monthly[0]
		middle := monthly[len(monthly)/2]
		last := monthly[len(monthly)-1]

		fmt.Printf("\n%s: %d bugs  →  %s: %d bugs  →  %s: %d bugs\n",
			first.Month.Format("Jan 2006"), first.TotalUnresolved,
			middle.Month.Format("Jan 2006"), middle.TotalUnresolved,
			last.Month.Format("Jan 2006"), last.TotalUnresolved,
		)
	}
}

// displayMonthlyTable shows month-by-month breakdown
func displayMonthlyTable(monthly []domain.MonthlyBugStats) {
	if len(monthly) == 0 {
		return
	}

	fmt.Println("\n📊 Monthly Bug Statistics")

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)

	// Set headers
	t.AppendHeader(table.Row{"Month", "Created", "Resolved", "Unresolved", "Trend"})

	// Show last 12 months for readability
	startIdx := 0
	if len(monthly) > 12 {
		startIdx = len(monthly) - 12
	}

	for i := startIdx; i < len(monthly); i++ {
		m := monthly[i]

		// Format trend indicator
		trend := "→"
		if m.ChangePercent > 5 {
			trend = "↑ " + fmt.Sprintf("+%.1f%%", m.ChangePercent)
		} else if m.ChangePercent < -5 {
			trend = "↓ " + fmt.Sprintf("%.1f%%", m.ChangePercent)
		}

		t.AppendRow(table.Row{
			m.Month.Format("Jan 2006"),
			m.TotalCreated,
			m.TotalResolved,
			m.TotalUnresolved,
			trend,
		})
	}

	t.Render()
}

// displayGoalProgress shows current month goal tracking
func displayGoalProgress(stats *domain.TrendStats) {
	if stats.CurrentMonth == nil || stats.LastYearSameMonth == nil {
		return
	}

	fmt.Println("\n🎯 Current Month Goal")

	currentMonthName := stats.CurrentMonth.Month.Format("January 2006")
	lastYearCount := stats.LastYearSameMonth.TotalCreated
	currentCount := stats.CurrentMonth.TotalCreated
	goalTarget := stats.GoalTarget

	// Calculate how we're doing
	var status string
	var statusColor text.Colors

	if currentCount <= goalTarget {
		percentBelow := ((float64(goalTarget-currentCount) / float64(goalTarget)) * 100)
		status = fmt.Sprintf("✓ On track (%.1f%% below target)", percentBelow)
		statusColor = text.Colors{text.FgGreen, text.Bold}
	} else {
		percentOver := ((float64(currentCount-goalTarget) / float64(goalTarget)) * 100)
		status = fmt.Sprintf("⚠ Over target (%.1f%% above)", percentOver)
		statusColor = text.Colors{text.FgYellow, text.Bold}
	}

	fmt.Printf("\n%s\n", currentMonthName)
	fmt.Printf("Last year: %d bugs created\n", lastYearCount)
	fmt.Printf("Target: ≤ %d bugs (%.0f%% reduction goal)\n", goalTarget, stats.ReductionGoal)
	fmt.Printf("Actual: %d bugs created so far\n", currentCount)
	fmt.Printf("Status: %s\n", text.Colors.Sprint(statusColor, status))
}

// displayPriorityBreakdown shows priority distribution over time
func displayPriorityBreakdown(monthly []domain.MonthlyBugStats) {
	if len(monthly) == 0 {
		return
	}

	fmt.Println("\n🔍 Priority Breakdown (Last 6 Months)")

	// Get last 6 months
	startIdx := 0
	if len(monthly) > 6 {
		startIdx = len(monthly) - 6
	}

	// Collect all priorities that appear
	prioritySet := make(map[string]bool)
	for i := startIdx; i < len(monthly); i++ {
		for priority := range monthly[i].ByPriority {
			prioritySet[priority] = true
		}
	}

	// Build table
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)

	// Build header with priorities in order: Critical, High, Medium, Low, Others
	priorityOrder := []string{"Critical", "High", "Medium", "Low"}
	headerRow := table.Row{"Month"}
	for _, p := range priorityOrder {
		if prioritySet[p] {
			headerRow = append(headerRow, p)
		}
	}
	// Add any other priorities
	for p := range prioritySet {
		found := false
		for _, known := range priorityOrder {
			if p == known {
				found = true
				break
			}
		}
		if !found {
			headerRow = append(headerRow, p)
			priorityOrder = append(priorityOrder, p)
		}
	}
	t.AppendHeader(headerRow)

	// Add rows
	for i := startIdx; i < len(monthly); i++ {
		m := monthly[i]
		row := table.Row{m.Month.Format("Jan 2006")}
		for _, p := range priorityOrder {
			if prioritySet[p] {
				count := m.ByPriority[p]
				row = append(row, count)
			}
		}
		t.AppendRow(row)
	}

	t.Render()
}

// generateSparkline creates an ASCII sparkline from values
func generateSparkline(values []int) string {
	if len(values) == 0 {
		return ""
	}

	// Unicode block characters for sparkline
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Find min and max
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Build sparkline
	var result strings.Builder
	for _, val := range values {
		// Normalize to 0-7 range
		var normalized int
		if max == min {
			normalized = 4 // Middle if all values are the same
		} else {
			ratio := float64(val-min) / float64(max-min)
			normalized = int(math.Round(ratio * 7))
			if normalized < 0 {
				normalized = 0
			}
			if normalized > 7 {
				normalized = 7
			}
		}
		result.WriteRune(blocks[normalized])
	}

	return result.String()
}

// displaySprintStats shows sprint-level bug statistics
func displaySprintStats(sprintStats []domain.SprintStats) {
	if len(sprintStats) == 0 {
		return
	}

	fmt.Println("\n🏃 Sprint Statistics")
	fmt.Printf("\nShowing bug density across %d sprints\n", len(sprintStats))

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)

	// Set headers
	t.AppendHeader(table.Row{
		"Sprint",
		"Bugs",
		"Other",
		"Total",
		"Bug %",
		"Bug Pts",
		"Total Pts",
		"Pts %",
	})

	// Add rows for each sprint
	for _, sprint := range sprintStats {
		// Format percentages
		bugPercent := fmt.Sprintf("%.1f%%", sprint.BugPercentage)
		pointsPercent := fmt.Sprintf("%.1f%%", sprint.PointsPercentage)

		// Color code bug percentage (higher is worse)
		var bugPercentColor text.Colors
		if sprint.BugPercentage > 50 {
			bugPercentColor = text.Colors{text.FgRed, text.Bold}
		} else if sprint.BugPercentage > 30 {
			bugPercentColor = text.Colors{text.FgYellow}
		} else {
			bugPercentColor = text.Colors{text.FgGreen}
		}

		t.AppendRow(table.Row{
			sprint.SprintName,
			sprint.BugCount,
			sprint.OtherCount,
			sprint.TotalCount,
			text.Colors.Sprint(bugPercentColor, bugPercent),
			fmt.Sprintf("%.1f", sprint.BugStoryPoints),
			fmt.Sprintf("%.1f", sprint.TotalStoryPoints),
			pointsPercent,
		})
	}

	t.Render()

	// Display summary statistics
	if len(sprintStats) > 0 {
		var totalBugs, totalOther int
		var totalBugPoints, totalAllPoints float64

		for _, s := range sprintStats {
			totalBugs += s.BugCount
			totalOther += s.OtherCount
			totalBugPoints += s.BugStoryPoints
			totalAllPoints += s.TotalStoryPoints
		}

		totalIssues := totalBugs + totalOther
		avgBugPercent := 0.0
		avgPointsPercent := 0.0

		if totalIssues > 0 {
			avgBugPercent = (float64(totalBugs) / float64(totalIssues)) * 100
		}
		if totalAllPoints > 0 {
			avgPointsPercent = (totalBugPoints / totalAllPoints) * 100
		}

		fmt.Printf("\nSummary:\n")
		fmt.Printf("  Total issues: %d (%d bugs, %d other)\n", totalIssues, totalBugs, totalOther)
		fmt.Printf("  Average bug density: %.1f%% of issues\n", avgBugPercent)
		fmt.Printf("  Average bug points: %.1f%% of story points\n", avgPointsPercent)
	}
}
