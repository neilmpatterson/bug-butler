package jira

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/neilmpatterson/bug-butler/internal/domain"
)

// MapIssueToBug converts a Jira issue to a domain Bug
func MapIssueToBug(issue *jira.Issue, baseURL string, sprintFieldID string, storyPointsFieldID string) (*domain.Bug, error) {
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

	// Extract issue type (Bug, Story, Task, etc.)
	issueType := "Unknown"
	if issue.Fields.Type.Name != "" {
		issueType = issue.Fields.Type.Name
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

	// Extract sprint information (from Unknowns map - customfield_10020 is common for sprints)
	sprintID := ""
	sprintName := ""
	if issue.Fields.Unknowns != nil {
		// Log available custom fields for debugging (helps identify correct field IDs)
		slog.Debug("Custom fields available for issue",
			"issue_key", issue.Key,
			"field_count", len(issue.Fields.Unknowns),
		)

		// Sprint field - use configured field ID
		if sprintData, ok := issue.Fields.Unknowns[sprintFieldID]; ok && sprintData != nil {
			// Sprint can be an array of sprint objects
			if sprints, ok := sprintData.([]interface{}); ok && len(sprints) > 0 {
				// Take the first sprint (current sprint)
				if sprint, ok := sprints[0].(map[string]interface{}); ok {
					if id, ok := sprint["id"].(float64); ok {
						sprintID = fmt.Sprintf("%.0f", id)
					}
					if name, ok := sprint["name"].(string); ok {
						sprintName = name
					}
				}
			}
		} else {
			slog.Debug("Sprint field not found or null",
				"issue_key", issue.Key,
				"sprint_field_id", sprintFieldID,
				"field_exists", issue.Fields.Unknowns[sprintFieldID] != nil,
			)
		}
	}

	// Extract story points - use configured field ID
	storyPoints := 0.0
	if issue.Fields.Unknowns != nil {
		if points, ok := issue.Fields.Unknowns[storyPointsFieldID]; ok && points != nil {
			if pointsFloat, ok := points.(float64); ok {
				storyPoints = pointsFloat
			}
		}
	}

	return &domain.Bug{
		Key:            issue.Key,
		Summary:        issue.Fields.Summary,
		Priority:       priority,
		Status:         status,
		IssueType:      issueType,
		Created:        created,
		Updated:        updated,
		Resolution:     resolution,
		ResolutionDate: resolutionDate,
		SprintID:       sprintID,
		SprintName:     sprintName,
		StoryPoints:    storyPoints,
		BaseURL:        baseURL,
	}, nil
}
