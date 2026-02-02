package adapter

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/logger"
)

// JiraClient implements port.JiraRepository
type JiraClient struct {
	baseURL    string
	email      string
	apiKey     string
	httpClient *http.Client
}

// NewJiraClient creates a new Jira client
func NewJiraClient(baseURL, email, apiKey string) *JiraClient {
	return &JiraClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		email:   email,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// issueResponse represents the Jira API issue response
type issueResponse struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string      `json:"summary"`
		Description interface{} `json:"description"`
		Attachment  []struct {
			ID       string `json:"id"`
			Filename string `json:"filename"`
			MimeType string `json:"mimeType"`
			Size     int64  `json:"size"`
			Content  string `json:"content"`
		} `json:"attachment"`
	} `json:"fields"`
}

// GetIssue fetches a Jira issue by its key
func (c *JiraClient) GetIssue(issueKey string) (*domain.JiraIssue, error) {
	logger.Debug("GetIssue: issueKey=%s, baseURL=%s", issueKey, c.baseURL)
	url := fmt.Sprintf("%s/rest/api/3/issue/%s", c.baseURL, issueKey)
	logger.Debug("GetIssue: requesting URL=%s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Debug("GetIssue: failed to create request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeader(req)
	req.Header.Set("Accept", "application/json")
	logger.Debug("GetIssue: auth header set for email=%s", c.email)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch issue: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var issueResp issueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issueResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	issue := &domain.JiraIssue{
		Key:         issueResp.Key,
		Summary:     issueResp.Fields.Summary,
		Description: parseADFToText(issueResp.Fields.Description),
		Link:        fmt.Sprintf("%s/browse/%s", c.baseURL, issueResp.Key),
	}

	for _, att := range issueResp.Fields.Attachment {
		issue.Attachments = append(issue.Attachments, domain.Attachment{
			ID:       att.ID,
			Filename: att.Filename,
			MimeType: att.MimeType,
			Size:     att.Size,
			URL:      att.Content,
		})
	}

	return issue, nil
}

// DownloadAttachment downloads an attachment from Jira
func (c *JiraClient) DownloadAttachment(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeader(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download attachment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed (status %d)", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *JiraClient) setAuthHeader(req *http.Request) {
	auth := base64.StdEncoding.EncodeToString([]byte(c.email + ":" + c.apiKey))
	req.Header.Set("Authorization", "Basic "+auth)
}

// parseADFToText converts Atlassian Document Format to plain text
func parseADFToText(adf interface{}) string {
	if adf == nil {
		return ""
	}

	adfMap, ok := adf.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("%v", adf)
	}

	var result strings.Builder
	parseADFNode(adfMap, &result)
	return strings.TrimSpace(result.String())
}

func parseADFNode(node map[string]interface{}, result *strings.Builder) {
	nodeType, _ := node["type"].(string)

	// Handle media nodes - insert placeholder for later replacement
	if nodeType == "mediaGroup" || nodeType == "mediaSingle" {
		if content, ok := node["content"].([]interface{}); ok {
			for _, child := range content {
				if childMap, ok := child.(map[string]interface{}); ok {
					if childMap["type"] == "media" {
						if attrs, ok := childMap["attrs"].(map[string]interface{}); ok {
							if filename, ok := attrs["alt"].(string); ok && filename != "" {
								result.WriteString(fmt.Sprintf("\n{{MEDIA:%s}}\n", filename))
							} else if id, ok := attrs["id"].(string); ok {
								result.WriteString(fmt.Sprintf("\n{{MEDIA_ID:%s}}\n", id))
							}
						}
					}
				}
			}
		}
		return
	}

	if content, ok := node["content"].([]interface{}); ok {
		for _, child := range content {
			if childMap, ok := child.(map[string]interface{}); ok {
				parseADFNode(childMap, result)
			}
		}
	}

	if nodeType == "text" {
		if text, ok := node["text"].(string); ok {
			result.WriteString(text)
		}
	}

	if nodeType == "paragraph" || nodeType == "listItem" || nodeType == "heading" || nodeType == "hardBreak" {
		result.WriteString("\n")
	}

	if nodeType == "bulletList" || nodeType == "orderedList" {
		result.WriteString("\n")
	}
}

// ExtractIssueKeyFromURL extracts the issue key from a Jira URL
func ExtractIssueKeyFromURL(url string) string {
	parts := strings.Split(url, "/")
	for i, part := range parts {
		if part == "browse" && i+1 < len(parts) {
			key := strings.Split(parts[i+1], "?")[0]
			return key
		}
	}

	for _, part := range parts {
		cleaned := strings.Split(part, "?")[0]
		if isIssueKey(cleaned) {
			return cleaned
		}
	}

	return ""
}

func isIssueKey(s string) bool {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return false
	}
	for _, c := range parts[0] {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	for _, c := range parts[1] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
