package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/neilmpatterson/bug-butler/internal/domain"
)

// DisplayBuckets renders the bucket groups as formatted terminal tables
func DisplayBuckets(bucketGroup *domain.BucketGroup) {
	if len(bucketGroup.Buckets) == 0 {
		fmt.Println("\n✅ All bugs are compliant with SLA rules!")
		fmt.Println("No bugs require immediate attention.")
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("  BUG BUTLER - SLA VIOLATION REPORT")
	fmt.Println(strings.Repeat("=", 80))

	// Display each bucket
	for _, bucket := range bucketGroup.Buckets {
		displayBucket(bucket)
	}

	// Display summary
	displaySummary(bucketGroup)
}

// displayBucket renders a single bucket as a table
func displayBucket(bucket *domain.Bucket) {
	fmt.Printf("\n%s (%d bugs)\n", bucket.Name, len(bucket.Bugs))

	if len(bucket.Bugs) == 0 {
		return
	}

	// Create table
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// Set style based on severity
	switch bucket.Severity {
	case 1:
		// Urgent - use rounded style with red colors
		t.SetStyle(table.StyleRounded)
		t.Style().Color.Header = text.Colors{text.BgRed, text.FgWhite, text.Bold}
		t.Style().Color.Row = text.Colors{text.FgHiRed}
	case 2:
		// Attention - use rounded style with yellow colors
		t.SetStyle(table.StyleRounded)
		t.Style().Color.Header = text.Colors{text.BgYellow, text.FgBlack, text.Bold}
		t.Style().Color.Row = text.Colors{text.FgHiYellow}
	default:
		// Review - default rounded style
		t.SetStyle(table.StyleRounded)
	}

	// Set headers
	t.AppendHeader(table.Row{"Key", "Summary", "Priority", "Status", "Age"})

	// Add rows with clickable URLs
	for _, bug := range bucket.Bugs {
		// Create clickable link using OSC 8 escape sequence (supported by modern terminals)
		clickableKey := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", bug.URL(), bug.Key)

		t.AppendRow(table.Row{
			clickableKey,
			truncateString(bug.Summary, 40),
			bug.Priority,
			bug.Status,
			formatAge(bug.AgeDays()),
		})
	}

	t.Render()
}

// displaySummary shows a summary of all violations
func displaySummary(bucketGroup *domain.BucketGroup) {
	fmt.Println("\n" + strings.Repeat("-", 80))
	fmt.Println("  SUMMARY")
	fmt.Println(strings.Repeat("-", 80))

	totalViolations := 0
	for _, bucket := range bucketGroup.Buckets {
		totalViolations += len(bucket.Bugs)
	}

	fmt.Printf("\nTotal SLA violations: %d\n", totalViolations)
	fmt.Println("\nBreakdown by bucket:")
	for _, bucket := range bucketGroup.Buckets {
		fmt.Printf("  %s: %d bugs\n", bucket.Name, len(bucket.Bugs))
	}

	fmt.Println()
}

// formatAge converts age in days to a human-readable string
func formatAge(days float64) string {
	if days < 1 {
		hours := days * 24
		if hours < 1 {
			minutes := hours * 60
			return fmt.Sprintf("%.0f minutes", minutes)
		}
		return fmt.Sprintf("%.1f hours", hours)
	} else if days < 7 {
		return fmt.Sprintf("%.1f days", days)
	} else {
		weeks := days / 7
		return fmt.Sprintf("%.1f weeks", weeks)
	}
}

// truncateString truncates a string to maxLen characters with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
