package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/neilmpatterson/bug-butler/internal/config"
	"github.com/neilmpatterson/bug-butler/internal/domain"
)

// Client wraps the Jira API client
type Client struct {
	client         *jira.Client
	projectKeys    []string
	baseURL        string
	additionalJQL  string
	sprintFieldID  string
	storyPointsFieldID string
}

// NewClient creates a new Jira client with authentication
func NewClient(cfg config.JiraConfig) (*Client, error) {
	// Create basic auth transport
	tp := jira.BasicAuthTransport{
		Username: cfg.Email,
		Password: cfg.APIToken,
	}

	// Create Jira client
	client, err := jira.NewClient(tp.Client(), cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira client: %w", err)
	}

	// Verify authentication by fetching current user using API v3
	req, err := client.NewRequest("GET", "/rest/api/3/myself", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}

	_, err = client.Do(req, nil)
	if err != nil {
		return nil, fmt.Errorf("authentication failed (check email and API token): %w", err)
	}

	slog.Debug("Successfully authenticated with Jira", "base_url", cfg.BaseURL, "email", cfg.Email)

	return &Client{
		client:             client,
		projectKeys:        cfg.ProjectKeys,
		baseURL:            cfg.BaseURL,
		additionalJQL:      cfg.AdditionalJQL,
		sprintFieldID:      cfg.CustomFieldIDs.Sprint,
		storyPointsFieldID: cfg.CustomFieldIDs.StoryPoints,
	}, nil
}

// searchResponse represents the API v3 search/jql response with cursor pagination
type searchResponse struct {
	Issues        []jira.Issue `json:"issues"`
	NextPageToken string       `json:"nextPageToken"` // Cursor for next page
	Total         int          `json:"total"`
}

// FetchBugs retrieves all unresolved bugs from the configured project(s) using API v3
func (c *Client) FetchBugs() ([]*domain.Bug, error) {
	// Build JQL query to fetch unresolved bugs
	var jql string
	if len(c.projectKeys) == 1 {
		jql = fmt.Sprintf("project = %s AND statusCategory != done AND type = Bug", c.projectKeys[0])
	} else {
		// Multiple projects - use "project in (...)" syntax
		projects := ""
		for i, key := range c.projectKeys {
			if i > 0 {
				projects += ", "
			}
			projects += fmt.Sprintf("\"%s\"", key)
		}
		jql = fmt.Sprintf("project in (%s) AND statusCategory != done AND type = Bug", projects)
	}

	// Append additional JQL filters if configured
	if c.additionalJQL != "" {
		jql += " " + c.additionalJQL
	}

	jql += " ORDER BY updated DESC"

	slog.Debug("Fetching bugs from Jira", "jql", jql, "projects", c.projectKeys)

	var allBugs []*domain.Bug
	maxResults := 100 // Fetch in batches of 100
	var nextPageToken string
	pageNumber := 0

	for {
		pageNumber++

		// Build GET request URL with cursor-based pagination
		params := url.Values{}
		params.Set("jql", jql)
		params.Set("maxResults", strconv.Itoa(maxResults))
		params.Set("fields", "summary,priority,status,created,updated")

		// Add nextPageToken if we have one (not the first page)
		if nextPageToken != "" {
			params.Set("nextPageToken", nextPageToken)
		}

		apiURL := "/rest/api/3/search/jql?" + params.Encode()

		// Create GET request with cursor pagination
		req, err := c.client.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create search request: %w", err)
		}

		// Execute request and read response body
		var searchResp searchResponse
		resp, err := c.client.Do(req, &searchResp)
		if err != nil {
			// Try to read response body for more details
			if resp != nil && resp.Body != nil {
				bodyBytes, readErr := io.ReadAll(resp.Body)
				resp.Body.Close()
				if readErr == nil {
					slog.Error("API request failed",
						"status_code", resp.StatusCode,
						"response_body", string(bodyBytes),
						"request_url", req.URL.String(),
					)
					return nil, fmt.Errorf("failed to search for issues (status %d): %s", resp.StatusCode, string(bodyBytes))
				}
			}
			return nil, fmt.Errorf("failed to search for issues: %w", err)
		}
		defer resp.Body.Close()

		slog.Debug("Fetched page",
			"page", pageNumber,
			"count", len(searchResp.Issues),
		)

		// Convert Jira issues to domain bugs
		for _, issue := range searchResp.Issues {
			bug, err := MapIssueToBug(&issue, c.baseURL, c.sprintFieldID, c.storyPointsFieldID)
			if err != nil {
				slog.Warn("Failed to map issue to bug", "issue_key", issue.Key, "error", err)
				continue
			}
			allBugs = append(allBugs, bug)
		}

		// Check if there are more pages using nextPageToken
		if searchResp.NextPageToken == "" {
			break
		}

		// Update token for next iteration
		nextPageToken = searchResp.NextPageToken
	}

	slog.Debug("Successfully fetched bugs", "count", len(allBugs))
	return allBugs, nil
}

// FetchBugsByDateRange retrieves all bugs created within a date range (including resolved bugs)
func (c *Client) FetchBugsByDateRange(startDate, endDate time.Time) ([]*domain.Bug, error) {
	// Format dates for JQL: YYYY-MM-DD
	start := startDate.Format("2006-01-02")
	end := endDate.Format("2006-01-02")

	// Build JQL query to fetch ALL bugs in date range (no status filter)
	var jql string
	if len(c.projectKeys) == 1 {
		jql = fmt.Sprintf("project = %s AND type = Bug AND created >= %s AND created < %s",
			c.projectKeys[0], start, end)
	} else {
		// Multiple projects - use "project in (...)" syntax
		projects := ""
		for i, key := range c.projectKeys {
			if i > 0 {
				projects += ", "
			}
			projects += fmt.Sprintf("\"%s\"", key)
		}
		jql = fmt.Sprintf("project in (%s) AND type = Bug AND created >= %s AND created < %s",
			projects, start, end)
	}

	// Append additional JQL filters if configured
	if c.additionalJQL != "" {
		jql += " " + c.additionalJQL
	}

	jql += " ORDER BY created DESC"

	slog.Debug("Fetching bugs by date range", "jql", jql, "start", start, "end", end)

	var allBugs []*domain.Bug
	maxResults := 100
	var nextPageToken string
	pageNumber := 0

	for {
		pageNumber++

		// Build GET request URL with cursor-based pagination
		params := url.Values{}
		params.Set("jql", jql)
		params.Set("maxResults", strconv.Itoa(maxResults))
		params.Set("fields", fmt.Sprintf("priority,created,resolution,resolutiondate,issuetype,%s,%s", c.sprintFieldID, c.storyPointsFieldID))

		// Add nextPageToken if we have one (not the first page)
		if nextPageToken != "" {
			params.Set("nextPageToken", nextPageToken)
		}

		apiURL := "/rest/api/3/search/jql?" + params.Encode()

		// Create GET request with cursor pagination
		req, err := c.client.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create search request: %w", err)
		}

		// Execute request and read response body
		var searchResp searchResponse
		resp, err := c.client.Do(req, &searchResp)
		if err != nil {
			// Try to read response body for more details
			if resp != nil && resp.Body != nil {
				bodyBytes, readErr := io.ReadAll(resp.Body)
				resp.Body.Close()
				if readErr == nil {
					slog.Error("API request failed",
						"status_code", resp.StatusCode,
						"response_body", string(bodyBytes),
						"request_url", req.URL.String(),
					)
					return nil, fmt.Errorf("failed to search for issues (status %d): %s", resp.StatusCode, string(bodyBytes))
				}
			}
			return nil, fmt.Errorf("failed to search for issues: %w", err)
		}
		defer resp.Body.Close()

		slog.Debug("Fetched page",
			"page", pageNumber,
			"count", len(searchResp.Issues),
		)

		// Convert Jira issues to domain bugs
		for _, issue := range searchResp.Issues {
			bug, err := MapIssueToBug(&issue, c.baseURL, c.sprintFieldID, c.storyPointsFieldID)
			if err != nil {
				slog.Warn("Failed to map issue to bug", "issue_key", issue.Key, "error", err)
				continue
			}
			allBugs = append(allBugs, bug)
		}

		// Check if there are more pages using nextPageToken
		if searchResp.NextPageToken == "" {
			break
		}

		// Update token for next iteration
		nextPageToken = searchResp.NextPageToken
	}

	slog.Debug("Successfully fetched bugs by date range", "count", len(allBugs))
	return allBugs, nil
}

// FetchIssuesBySprints retrieves all done issues for the specified sprint IDs
func (c *Client) FetchIssuesBySprints(sprintIDs []string) ([]*domain.Bug, error) {
	if len(sprintIDs) == 0 {
		return []*domain.Bug{}, nil
	}

	// Build JQL query to fetch done issues for specific sprints
	// Sprint custom field is typically customfield_10020
	sprintList := ""
	for i, id := range sprintIDs {
		if i > 0 {
			sprintList += ", "
		}
		sprintList += id
	}

	var jql string
	if len(c.projectKeys) == 1 {
		jql = fmt.Sprintf("project = %s AND sprint in (%s) AND statusCategory = done",
			c.projectKeys[0], sprintList)
	} else {
		// Multiple projects - use "project in (...)" syntax
		projects := ""
		for i, key := range c.projectKeys {
			if i > 0 {
				projects += ", "
			}
			projects += fmt.Sprintf("\"%s\"", key)
		}
		jql = fmt.Sprintf("project in (%s) AND sprint in (%s) AND statusCategory = done",
			projects, sprintList)
	}

	// NOTE: We do NOT apply additional_jql here because sprint stats need ALL issues
	// (bugs + other types), not just filtered bugs. The additional_jql is meant for
	// bug-specific filtering and would incorrectly exclude Stories/Tasks/etc.

	jql += " ORDER BY resolutiondate DESC"

	slog.Debug("Fetching issues by sprints", "jql", jql, "sprint_count", len(sprintIDs))

	var allIssues []*domain.Bug
	maxResults := 100
	var nextPageToken string
	pageNumber := 0

	for {
		pageNumber++

		// Build GET request URL with cursor-based pagination
		params := url.Values{}
		params.Set("jql", jql)
		params.Set("maxResults", strconv.Itoa(maxResults))
		params.Set("fields", fmt.Sprintf("issuetype,resolution,resolutiondate,%s,%s", c.sprintFieldID, c.storyPointsFieldID))

		// Add nextPageToken if we have one (not the first page)
		if nextPageToken != "" {
			params.Set("nextPageToken", nextPageToken)
		}

		apiURL := "/rest/api/3/search/jql?" + params.Encode()

		// Create GET request with cursor pagination
		req, err := c.client.NewRequest("GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create search request: %w", err)
		}

		// Execute request and read response body
		var searchResp searchResponse
		resp, err := c.client.Do(req, &searchResp)
		if err != nil {
			// Try to read response body for more details
			if resp != nil && resp.Body != nil {
				bodyBytes, readErr := io.ReadAll(resp.Body)
				resp.Body.Close()
				if readErr == nil {
					slog.Error("API request failed",
						"status_code", resp.StatusCode,
						"response_body", string(bodyBytes),
						"request_url", req.URL.String(),
					)
					return nil, fmt.Errorf("failed to search for issues (status %d): %s", resp.StatusCode, string(bodyBytes))
				}
			}
			return nil, fmt.Errorf("failed to search for issues: %w", err)
		}
		defer resp.Body.Close()

		slog.Debug("Fetched page",
			"page", pageNumber,
			"count", len(searchResp.Issues),
		)

		// Convert Jira issues to domain bugs (reusing the same struct for all issue types)
		for _, issue := range searchResp.Issues {
			bug, err := MapIssueToBug(&issue, c.baseURL, c.sprintFieldID, c.storyPointsFieldID)
			if err != nil {
				slog.Warn("Failed to map issue", "issue_key", issue.Key, "error", err)
				continue
			}
			allIssues = append(allIssues, bug)
		}

		// Check if there are more pages using nextPageToken
		if searchResp.NextPageToken == "" {
			break
		}

		// Update token for next iteration
		nextPageToken = searchResp.NextPageToken
	}

	slog.Debug("Successfully fetched issues by sprints", "count", len(allIssues))
	return allIssues, nil
}

// parseSearchResponse parses the JSON response from API v3
func parseSearchResponse(data []byte) (*searchResponse, error) {
	var resp searchResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}
	return &resp, nil
}

// buildSearchURL builds the API v3 search URL with parameters
func buildSearchURL(jql string, startAt, maxResults int, fields []string) string {
	v := url.Values{}
	v.Set("jql", jql)
	v.Set("startAt", strconv.Itoa(startAt))
	v.Set("maxResults", strconv.Itoa(maxResults))
	if len(fields) > 0 {
		for _, field := range fields {
			v.Add("fields", field)
		}
	}
	return "/rest/api/3/search?" + v.Encode()
}
