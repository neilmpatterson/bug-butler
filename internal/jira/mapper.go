package jira

import (
	"fmt"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/neilmpatterson/bug-butler/internal/domain"
)

// MapIssueToBug converts a Jira issue to a domain Bug
func MapIssueToBug(issue *jira.Issue, baseURL string) (*domain.Bug, error) {
	if issue == nil {
		return nil, fmt.Errorf("issue is nil")
	}

	// Extract priority name (handle nil priority)
	priority := "Unknown"
	if issue.Fields.Priority != nil {
		priority = issue.Fields.Priority.Name
	}

	// Extract status name (should always be present)
	status := "Unknown"
	if issue.Fields.Status != nil {
		status = issue.Fields.Status.Name
	}

	// Parse timestamps (go-jira Time type)
	created := time.Time(issue.Fields.Created)
	updated := time.Time(issue.Fields.Updated)

	// Extract resolution (may be nil if unresolved)
	resolution := ""
	if issue.Fields.Resolution != nil {
		resolution = issue.Fields.Resolution.Name
	}

	// Extract resolution date (may be zero if unresolved)
	var resolutionDate *time.Time
	if !time.Time(issue.Fields.Resolutiondate).IsZero() {
		t := time.Time(issue.Fields.Resolutiondate)
		resolutionDate = &t
	}

	return &domain.Bug{
		Key:            issue.Key,
		Summary:        issue.Fields.Summary,
		Priority:       priority,
		Status:         status,
		Created:        created,
		Updated:        updated,
		Resolution:     resolution,
		ResolutionDate: resolutionDate,
		BaseURL:        baseURL,
	}, nil
}
