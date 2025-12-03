package domain

import "time"

// Bug represents a Jira issue with relevant fields for SLA monitoring
type Bug struct {
	Key            string     // Jira issue key (e.g., "PROJ-123")
	Summary        string     // Issue title/summary
	Priority       string     // Priority level (Critical, High, Medium, Low)
	Status         string     // Current status (Backlog, Needs Triage, etc.)
	Created        time.Time  // When the bug was created
	Updated        time.Time  // When the bug was last updated
	Resolution     string     // Resolution status (empty if unresolved)
	ResolutionDate *time.Time // When the bug was resolved (nil if unresolved)
	BaseURL        string     // Jira base URL for building links
}

// URL returns the full URL to the bug in Jira
func (b *Bug) URL() string {
	return b.BaseURL + "/browse/" + b.Key
}

// Age calculates how long ago the bug was last updated
func (b *Bug) Age() time.Duration {
	return time.Since(b.Updated)
}

// AgeDays returns the age of the bug in days
func (b *Bug) AgeDays() float64 {
	return b.Age().Hours() / 24
}

// SLARule defines a threshold for bug age based on priority and status
type SLARule struct {
	Name        string   // Descriptive name for the rule
	Priority    string   // Priority to match (e.g., "Critical")
	Status      []string // Status(es) to match (e.g., ["Backlog", "Needs Triage"])
	MaxAgeDays  float64  // Maximum allowed age in days
	BucketName  string   // Which bucket to assign violations to
	Severity    int      // Bucket display priority (1 = highest)
}

// Matches checks if a bug matches this rule's criteria
func (r *SLARule) Matches(bug *Bug) bool {
	// Check priority match
	if r.Priority != "" && r.Priority != bug.Priority {
		return false
	}

	// Check status match (OR logic for multiple statuses)
	if len(r.Status) > 0 {
		statusMatch := false
		for _, status := range r.Status {
			if status == bug.Status {
				statusMatch = true
				break
			}
		}
		if !statusMatch {
			return false
		}
	}

	return true
}

// Violates checks if a bug violates this rule (matches criteria and exceeds age)
func (r *SLARule) Violates(bug *Bug) bool {
	return r.Matches(bug) && bug.AgeDays() > r.MaxAgeDays
}

// Bucket represents a category of bugs based on SLA status
type Bucket struct {
	Name     string  // Display name (e.g., "🔴 URGENT")
	Severity int     // Display priority (1 = highest)
	Bugs     []*Bug  // Bugs in this bucket
}

// BucketGroup is a collection of buckets sorted by severity
type BucketGroup struct {
	Buckets []*Bucket
}

// AddToBucket adds a bug to a named bucket, creating it if needed
func (bg *BucketGroup) AddToBucket(bucketName string, severity int, bug *Bug) {
	// Find existing bucket
	for _, bucket := range bg.Buckets {
		if bucket.Name == bucketName {
			bucket.Bugs = append(bucket.Bugs, bug)
			return
		}
	}

	// Create new bucket
	bg.Buckets = append(bg.Buckets, &Bucket{
		Name:     bucketName,
		Severity: severity,
		Bugs:     []*Bug{bug},
	})
}

// Sort sorts buckets by severity (ascending, so 1 comes first)
func (bg *BucketGroup) Sort() {
	// Simple bubble sort is fine for small number of buckets
	n := len(bg.Buckets)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if bg.Buckets[j].Severity > bg.Buckets[j+1].Severity {
				bg.Buckets[j], bg.Buckets[j+1] = bg.Buckets[j+1], bg.Buckets[j]
			}
		}
	}
}

// MonthlyBugStats represents bug metrics for a single month
type MonthlyBugStats struct {
	Month           time.Time      // First day of the month
	TotalCreated    int            // Total bugs created in this month
	TotalResolved   int            // Total bugs resolved in this month
	TotalUnresolved int            // Total unresolved bugs at end of this month (backlog size)
	NetChange       int            // Created - Resolved
	ChangePercent   float64        // % change in created from previous month
	ByPriority      map[string]int // Created count by priority level
}

// TrendStats represents complete trend analysis over a time period
type TrendStats struct {
	MonthlyData       []MonthlyBugStats // Monthly statistics ordered chronologically
	CurrentMonth      *MonthlyBugStats  // In-progress month (partial data)
	LastYearSameMonth *MonthlyBugStats  // Same month from last year (for goal comparison)
	ReductionGoal     float64           // Target reduction percentage
	GoalTarget        int               // Calculated bug count target for current month
	OnTrack           bool              // Whether current month is meeting the goal
}
