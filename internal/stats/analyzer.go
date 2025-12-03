package stats

import (
	"math"
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
