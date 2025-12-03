package stats

import (
	"log/slog"
	"math"
	"regexp"
	"sort"
	"time"

	"github.com/neilmpatterson/bug-butler/internal/domain"
)

// Analyzer performs trend analysis on bug data
type Analyzer struct {
	reductionGoal   float64
	monthsToAnalyze int
}

// NewAnalyzer creates a new stats analyzer with configuration
func NewAnalyzer(reductionGoal float64, months int) *Analyzer {
	return &Analyzer{
		reductionGoal:   reductionGoal,
		monthsToAnalyze: months,
	}
}

// Analyze processes bugs and returns trend statistics
func (a *Analyzer) Analyze(bugs []*domain.Bug) (*domain.TrendStats, error) {
	// Group bugs by creation month
	grouped := groupByMonth(bugs)

	// Get list of months in chronological order
	months := make([]time.Time, 0, len(grouped))
	for month := range grouped {
		months = append(months, month)
	}
	sort.Slice(months, func(i, j int) bool {
		return months[i].Before(months[j])
	})

	// Build monthly statistics
	monthlyData := make([]domain.MonthlyBugStats, 0, len(months))
	var previousCreatedCount int

	for _, month := range months {
		bugsCreatedThisMonth := grouped[month]
		created := len(bugsCreatedThisMonth)
		priorityBreakdown := buildPriorityBreakdown(bugsCreatedThisMonth)

		// Calculate percentage change in created from previous month
		var changePercent float64
		if previousCreatedCount > 0 {
			changePercent = ((float64(created) - float64(previousCreatedCount)) / float64(previousCreatedCount)) * 100
		}

		// Calculate total unresolved bugs at end of this month
		// End of month is the last day of the month at 23:59:59
		monthEnd := time.Date(month.Year(), month.Month()+1, 0, 23, 59, 59, 0, time.UTC)
		unresolvedCount := countUnresolvedAtDate(bugs, monthEnd)

		// Count bugs resolved in this month (for future tracking)
		resolvedThisMonth := countResolvedInMonth(bugs, month)

		monthlyData = append(monthlyData, domain.MonthlyBugStats{
			Month:           month,
			TotalCreated:    created,
			TotalResolved:   resolvedThisMonth,
			TotalUnresolved: unresolvedCount,
			NetChange:       created - previousCreatedCount,
			ChangePercent:   changePercent,
			ByPriority:      priorityBreakdown,
		})

		previousCreatedCount = created
	}

	// Identify current month and last year's same month
	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastYearMonthStart := currentMonthStart.AddDate(-1, 0, 0)

	var currentMonth *domain.MonthlyBugStats
	var lastYearSameMonth *domain.MonthlyBugStats

	for i := range monthlyData {
		if monthlyData[i].Month.Equal(currentMonthStart) {
			currentMonth = &monthlyData[i]
		}
		if monthlyData[i].Month.Equal(lastYearMonthStart) {
			lastYearSameMonth = &monthlyData[i]
		}
	}

	// Calculate goal target and progress
	var goalTarget int
	var onTrack bool

	if lastYearSameMonth != nil && currentMonth != nil {
		goalTarget = calculateGoalTarget(lastYearSameMonth.TotalCreated, a.reductionGoal)
		onTrack = currentMonth.TotalCreated <= goalTarget
	}

	return &domain.TrendStats{
		MonthlyData:       monthlyData,
		CurrentMonth:      currentMonth,
		LastYearSameMonth: lastYearSameMonth,
		ReductionGoal:     a.reductionGoal,
		GoalTarget:        goalTarget,
		OnTrack:           onTrack,
		SprintStats:       []domain.SprintStats{}, // Will be populated separately if enabled
	}, nil
}

// groupByMonth groups bugs by their creation month
func groupByMonth(bugs []*domain.Bug) map[time.Time][]*domain.Bug {
	grouped := make(map[time.Time][]*domain.Bug)

	for _, bug := range bugs {
		// Normalize to first day of month (UTC)
		month := time.Date(bug.Created.Year(), bug.Created.Month(), 1, 0, 0, 0, 0, time.UTC)
		grouped[month] = append(grouped[month], bug)
	}

	return grouped
}

// calculateGoalTarget calculates the target bug count based on last year and reduction percentage
func calculateGoalTarget(lastYearCount int, reductionPercent float64) int {
	reduction := float64(lastYearCount) * (reductionPercent / 100.0)
	target := float64(lastYearCount) - reduction
	return int(math.Round(target))
}

// buildPriorityBreakdown creates a map of priority to count
func buildPriorityBreakdown(bugs []*domain.Bug) map[string]int {
	breakdown := make(map[string]int)

	for _, bug := range bugs {
		breakdown[bug.Priority]++
	}

	return breakdown
}

// countUnresolvedAtDate counts bugs that were unresolved at a specific date
// A bug is unresolved at date X if: created <= X AND (resolution is empty OR resolved > X)
func countUnresolvedAtDate(bugs []*domain.Bug, date time.Time) int {
	count := 0
	for _, bug := range bugs {
		// Bug must have been created before or at this date
		if bug.Created.After(date) {
			continue
		}

		// Bug is unresolved if it has no resolution or was resolved after this date
		if bug.Resolution == "" || (bug.ResolutionDate != nil && bug.ResolutionDate.After(date)) {
			count++
		}
	}
	return count
}

// countResolvedInMonth counts bugs that were resolved in a specific month
func countResolvedInMonth(bugs []*domain.Bug, month time.Time) int {
	// Calculate month boundaries
	monthStart := month
	monthEnd := time.Date(month.Year(), month.Month()+1, 0, 23, 59, 59, 0, time.UTC)

	count := 0
	for _, bug := range bugs {
		if bug.ResolutionDate != nil {
			// Check if resolution date falls within this month
			if !bug.ResolutionDate.Before(monthStart) && !bug.ResolutionDate.After(monthEnd) {
				count++
			}
		}
	}
	return count
}

// SprintInfo holds sprint ID and name for filtering
type SprintInfo struct {
	ID   string
	Name string
}

// ExtractSprintIDs extracts unique sprint IDs from bugs
func ExtractSprintIDs(bugs []*domain.Bug) []string {
	sprintMap := make(map[string]bool)

	for _, bug := range bugs {
		if bug.SprintID != "" {
			sprintMap[bug.SprintID] = true
		}
	}

	// Convert map to slice
	sprintIDs := make([]string, 0, len(sprintMap))
	for id := range sprintMap {
		sprintIDs = append(sprintIDs, id)
	}

	return sprintIDs
}

// ExtractAndFilterSprints extracts sprint info and filters by name pattern
// Returns only sprint IDs that match the filter criteria
func ExtractAndFilterSprints(bugs []*domain.Bug, sprintNameBeginsWith string, sprintNamePattern string) []string {
	// Extract unique sprints with their names
	sprintMap := make(map[string]string) // ID -> Name

	for _, bug := range bugs {
		if bug.SprintID != "" {
			sprintMap[bug.SprintID] = bug.SprintName
		}
	}

	slog.Debug("Extracted sprints from bugs",
		"total_sprint_count", len(sprintMap),
	)

	// Compile filter if provided
	var nameFilter *regexp.Regexp
	var err error

	if sprintNamePattern != "" {
		nameFilter, err = regexp.Compile(sprintNamePattern)
		if err != nil {
			slog.Warn("Invalid sprint name pattern, ignoring filter",
				"pattern", sprintNamePattern,
				"error", err,
			)
		} else {
			slog.Debug("Filtering sprints by regex pattern", "pattern", sprintNamePattern)
		}
	} else if sprintNameBeginsWith != "" {
		// Convert simple prefix to regex pattern
		escapedPrefix := regexp.QuoteMeta(sprintNameBeginsWith)
		pattern := "^" + escapedPrefix
		nameFilter, err = regexp.Compile(pattern)
		if err != nil {
			slog.Warn("Failed to compile prefix filter",
				"prefix", sprintNameBeginsWith,
				"error", err,
			)
		} else {
			slog.Debug("Filtering sprints by prefix", "prefix", sprintNameBeginsWith)
		}
	}

	// Filter sprint IDs by name
	filteredIDs := make([]string, 0)
	excludedCount := 0

	for sprintID, sprintName := range sprintMap {
		// Apply name filter if configured
		if nameFilter != nil {
			if nameFilter.MatchString(sprintName) {
				filteredIDs = append(filteredIDs, sprintID)
			} else {
				excludedCount++
				slog.Debug("Excluding sprint due to name filter",
					"sprint_id", sprintID,
					"sprint_name", sprintName,
				)
			}
		} else {
			// No filter, include all
			filteredIDs = append(filteredIDs, sprintID)
		}
	}

	slog.Debug("Sprint filtering complete",
		"total_sprints", len(sprintMap),
		"filtered_sprints", len(filteredIDs),
		"excluded_sprints", excludedCount,
	)

	return filteredIDs
}

// CalculateSprintStats analyzes all sprint issues and calculates statistics per sprint
// Parameters:
//   - sprintIssues: All issues from the sprints
//   - sprintNameBeginsWith: Simple prefix filter (e.g., "TOOLS Sprint")
//   - sprintNamePattern: Advanced regex pattern (overrides begins_with if set)
func (a *Analyzer) CalculateSprintStats(sprintIssues []*domain.Bug, sprintNameBeginsWith string, sprintNamePattern string) []domain.SprintStats {
	// Compile regex pattern
	var nameFilter *regexp.Regexp
	var err error

	// Priority: Use pattern if provided, otherwise convert begins_with to pattern
	if sprintNamePattern != "" {
		nameFilter, err = regexp.Compile(sprintNamePattern)
		if err != nil {
			slog.Warn("Invalid sprint name pattern, ignoring filter",
				"pattern", sprintNamePattern,
				"error", err,
			)
			nameFilter = nil
		} else {
			slog.Debug("Filtering sprints by regex pattern", "pattern", sprintNamePattern)
		}
	} else if sprintNameBeginsWith != "" {
		// Convert simple prefix to regex pattern
		escapedPrefix := regexp.QuoteMeta(sprintNameBeginsWith)
		pattern := "^" + escapedPrefix
		nameFilter, err = regexp.Compile(pattern)
		if err != nil {
			slog.Warn("Failed to compile prefix filter",
				"prefix", sprintNameBeginsWith,
				"error", err,
			)
			nameFilter = nil
		} else {
			slog.Debug("Filtering sprints by prefix", "prefix", sprintNameBeginsWith)
		}
	}

	// Group issues by sprint
	sprintGroups := make(map[string][]*domain.Bug)
	sprintNames := make(map[string]string)

	for _, issue := range sprintIssues {
		if issue.SprintID != "" {
			// Apply name filter if configured
			if nameFilter != nil && !nameFilter.MatchString(issue.SprintName) {
				slog.Debug("Excluding sprint due to name filter",
					"sprint_name", issue.SprintName,
					"pattern", sprintNamePattern,
				)
				continue
			}

			sprintGroups[issue.SprintID] = append(sprintGroups[issue.SprintID], issue)
			sprintNames[issue.SprintID] = issue.SprintName
		}
	}

	// Calculate stats for each sprint
	stats := make([]domain.SprintStats, 0, len(sprintGroups))

	for sprintID, issues := range sprintGroups {
		bugCount := 0
		otherCount := 0
		bugStoryPoints := 0.0
		totalStoryPoints := 0.0

		for _, issue := range issues {
			if issue.IssueType == "Bug" {
				bugCount++
				bugStoryPoints += issue.StoryPoints
			} else {
				otherCount++
			}
			totalStoryPoints += issue.StoryPoints
		}

		totalCount := bugCount + otherCount
		bugPercentage := 0.0
		pointsPercentage := 0.0

		if totalCount > 0 {
			bugPercentage = (float64(bugCount) / float64(totalCount)) * 100
		}

		if totalStoryPoints > 0 {
			pointsPercentage = (bugStoryPoints / totalStoryPoints) * 100
		}

		stats = append(stats, domain.SprintStats{
			SprintID:         sprintID,
			SprintName:       sprintNames[sprintID],
			BugCount:         bugCount,
			OtherCount:       otherCount,
			TotalCount:       totalCount,
			BugPercentage:    bugPercentage,
			BugStoryPoints:   bugStoryPoints,
			TotalStoryPoints: totalStoryPoints,
			PointsPercentage: pointsPercentage,
		})
	}

	// Sort by sprint name for consistent display
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].SprintName < stats[j].SprintName
	})

	return stats
}
