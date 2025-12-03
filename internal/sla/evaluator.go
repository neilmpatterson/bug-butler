package sla

import (
	"log/slog"

	"github.com/neilmpatterson/bug-butler/internal/config"
	"github.com/neilmpatterson/bug-butler/internal/domain"
)

// Evaluator applies SLA rules to bugs and groups them into buckets
type Evaluator struct {
	rules []domain.SLARule
}

// NewEvaluator creates a new SLA evaluator with the given rules
func NewEvaluator(rules []config.SLARule) *Evaluator {
	// Convert config rules to domain rules
	domainRules := make([]domain.SLARule, len(rules))
	for i, rule := range rules {
		// Handle single status string by converting to array
		statuses := rule.Status
		if len(statuses) == 0 && rule.Priority != "" {
			// If no status specified, match any status
			statuses = nil
		}

		domainRules[i] = domain.SLARule{
			Name:       rule.Name,
			Priority:   rule.Priority,
			Status:     statuses,
			MaxAgeDays: rule.MaxAgeDays,
			BucketName: rule.Bucket,
			Severity:   rule.Severity,
		}
	}

	return &Evaluator{
		rules: domainRules,
	}
}

// Evaluate applies SLA rules to bugs and returns grouped buckets
func (e *Evaluator) Evaluate(bugs []*domain.Bug) *domain.BucketGroup {
	bucketGroup := &domain.BucketGroup{}

	slog.Debug("Evaluating bugs against SLA rules", "bug_count", len(bugs), "rule_count", len(e.rules))

	// Track unique priorities and statuses for debugging
	priorities := make(map[string]int)
	statuses := make(map[string]int)
	for _, bug := range bugs {
		priorities[bug.Priority]++
		statuses[bug.Status]++
	}
	slog.Debug("Bug distribution by priority", "priorities", priorities)
	slog.Debug("Bug distribution by status", "statuses", statuses)

	violationCount := 0

	// Process each bug
	for _, bug := range bugs {
		// Try to match against rules in order (first-match wins)
		matched := false
		for _, rule := range e.rules {
			// Check if bug matches criteria (priority + status)
			if rule.Matches(bug) {
				// Bug matches criteria - check if it violates age threshold
				if rule.Violates(bug) {
					slog.Debug("Bug violates SLA rule",
						"bug_key", bug.Key,
						"rule", rule.Name,
						"priority", bug.Priority,
						"status", bug.Status,
						"age_days", bug.AgeDays(),
						"max_age", rule.MaxAgeDays,
					)
					bucketGroup.AddToBucket(rule.BucketName, rule.Severity, bug)
					matched = true
					violationCount++
					break // First-match wins
				} else {
					// Matched criteria but within SLA (too new)
					slog.Debug("Bug matches criteria but within SLA",
						"bug_key", bug.Key,
						"rule", rule.Name,
						"priority", bug.Priority,
						"status", bug.Status,
						"age_days", bug.AgeDays(),
						"max_age", rule.MaxAgeDays,
					)
				}
			}
		}

		if !matched {
			slog.Debug("Bug is compliant with all SLA rules",
				"bug_key", bug.Key,
				"priority", bug.Priority,
				"status", bug.Status,
				"age_days", bug.AgeDays(),
			)
		}
	}

	// Sort buckets by severity
	bucketGroup.Sort()

	slog.Debug("SLA evaluation complete",
		"total_bugs", len(bugs),
		"violations", violationCount,
		"buckets", len(bucketGroup.Buckets),
	)

	return bucketGroup
}

// GetViolationSummary returns a summary of SLA violations
func (e *Evaluator) GetViolationSummary(bucketGroup *domain.BucketGroup) map[string]int {
	summary := make(map[string]int)

	for _, bucket := range bucketGroup.Buckets {
		summary[bucket.Name] = len(bucket.Bugs)
	}

	return summary
}
