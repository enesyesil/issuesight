package transformer

import (
	"testing"
)

func TestTransformStreamMessage(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]interface{}
		wantErr bool
		check   func(t *testing.T, payload *StreamIssuePayload)
	}{
		{
			name: "valid payload with all fields",
			payload: map[string]interface{}{
				"issue_id":         "12345",
				"issue_number":     "42",
				"owner":            "testowner",
				"repo":             "testrepo",
				"full_name":        "testowner/testrepo",
				"title":            "Test Issue Title",
				"body":             "This is the issue body",
				"labels":           `["bug","feature"]`,
				"state":            "open",
				"html_url":         "https://github.com/testowner/testrepo/issues/42",
				"repo_id":          "67890",
				"repo_language":    "Go",
				"repo_stars":       "100",
				"repo_description": "Example repository description",
				"collected_at":     "2024-01-15T10:30:00Z",
			},
			wantErr: false,
			check: func(t *testing.T, p *StreamIssuePayload) {
				if p.IssueID != 12345 {
					t.Errorf("IssueID = %d, want 12345", p.IssueID)
				}
				if p.IssueNumber != 42 {
					t.Errorf("IssueNumber = %d, want 42", p.IssueNumber)
				}
				if p.Owner != "testowner" {
					t.Errorf("Owner = %s, want testowner", p.Owner)
				}
				if p.Repo != "testrepo" {
					t.Errorf("Repo = %s, want testrepo", p.Repo)
				}
				if p.Title != "Test Issue Title" {
					t.Errorf("Title = %s, want 'Test Issue Title'", p.Title)
				}
				if len(p.Labels) != 2 {
					t.Errorf("Labels length = %d, want 2", len(p.Labels))
				}
				if p.RepoLanguage != "Go" {
					t.Errorf("RepoLanguage = %s, want Go", p.RepoLanguage)
				}
				if p.RepoDescription != "Example repository description" {
					t.Errorf("RepoDescription = %s, want example description", p.RepoDescription)
				}
			},
		},
		{
			name: "valid payload with minimal required fields",
			payload: map[string]interface{}{
				"issue_id": int64(12345),
				"owner":    "testowner",
				"repo":     "testrepo",
				"title":    "Test Issue",
			},
			wantErr: false,
			check: func(t *testing.T, p *StreamIssuePayload) {
				if p.IssueID != 12345 {
					t.Errorf("IssueID = %d, want 12345", p.IssueID)
				}
				if p.Owner != "testowner" {
					t.Errorf("Owner = %s, want testowner", p.Owner)
				}
			},
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: true,
		},
		{
			name:    "empty payload",
			payload: map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "missing required owner",
			payload: map[string]interface{}{
				"issue_id": "12345",
				"repo":     "testrepo",
				"title":    "Test Issue",
			},
			wantErr: true,
		},
		{
			name: "missing required repo",
			payload: map[string]interface{}{
				"issue_id": "12345",
				"owner":    "testowner",
				"title":    "Test Issue",
			},
			wantErr: true,
		},
		{
			name: "missing required title",
			payload: map[string]interface{}{
				"issue_id": "12345",
				"owner":    "testowner",
				"repo":     "testrepo",
			},
			wantErr: true,
		},
		{
			name: "missing required issue_id",
			payload: map[string]interface{}{
				"owner": "testowner",
				"repo":  "testrepo",
				"title": "Test Issue",
			},
			wantErr: true,
		},
		{
			name: "invalid labels JSON - should not fail",
			payload: map[string]interface{}{
				"issue_id": "12345",
				"owner":    "testowner",
				"repo":     "testrepo",
				"title":    "Test Issue",
				"labels":   "not-valid-json",
			},
			wantErr: false,
			check: func(t *testing.T, p *StreamIssuePayload) {
				if len(p.Labels) != 0 {
					t.Errorf("Labels should be empty for invalid JSON, got %v", p.Labels)
				}
			},
		},
		{
			name: "numeric fields as float64",
			payload: map[string]interface{}{
				"issue_id":     float64(12345),
				"issue_number": float64(42),
				"owner":        "testowner",
				"repo":         "testrepo",
				"title":        "Test Issue",
				"repo_stars":   float64(100),
			},
			wantErr: false,
			check: func(t *testing.T, p *StreamIssuePayload) {
				if p.IssueID != 12345 {
					t.Errorf("IssueID = %d, want 12345", p.IssueID)
				}
				if p.RepoStars != 100 {
					t.Errorf("RepoStars = %d, want 100", p.RepoStars)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := TransformStreamMessage(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransformStreamMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, payload)
			}
		})
	}
}

func TestStreamIssuePayload_ToIssue(t *testing.T) {
	payload := &StreamIssuePayload{
		IssueID:     12345,
		IssueNumber: 42,
		Owner:       "testowner",
		Repo:        "testrepo",
		FullName:    "testowner/testrepo",
		Title:       "Test Issue",
		Body:        "Test body",
		Labels:      []string{"bug"},
		State:       "open",
		HTMLURL:     "https://github.com/testowner/testrepo/issues/42",
	}

	issue := payload.ToIssue()

	if issue.ID != payload.IssueID {
		t.Errorf("ID = %d, want %d", issue.ID, payload.IssueID)
	}
	if issue.RepoOwner != payload.Owner {
		t.Errorf("RepoOwner = %s, want %s", issue.RepoOwner, payload.Owner)
	}
	if issue.RepoName != payload.Repo {
		t.Errorf("RepoName = %s, want %s", issue.RepoName, payload.Repo)
	}
	if issue.Title != payload.Title {
		t.Errorf("Title = %s, want %s", issue.Title, payload.Title)
	}
}

func TestStreamIssuePayload_ToRepository(t *testing.T) {
	payload := &StreamIssuePayload{
		RepoID:       67890,
		FullName:     "testowner/testrepo",
		Owner:        "testowner",
		Repo:         "testrepo",
		RepoLanguage: "Go",
		RepoStars:    100,
	}

	repo := payload.ToRepository()

	if repo.ID != payload.RepoID {
		t.Errorf("ID = %d, want %d", repo.ID, payload.RepoID)
	}
	if repo.Language != payload.RepoLanguage {
		t.Errorf("Language = %s, want %s", repo.Language, payload.RepoLanguage)
	}
	if repo.Stars != payload.RepoStars {
		t.Errorf("Stars = %d, want %d", repo.Stars, payload.RepoStars)
	}
}

func TestTransformStreamMessage_FieldLengthLimits(t *testing.T) {
	longString := func(n int) string {
		s := make([]byte, n)
		for i := range s {
			s[i] = 'a'
		}
		return string(s)
	}

	tests := []struct {
		name    string
		payload map[string]interface{}
		wantErr bool
		check   func(t *testing.T, p *StreamIssuePayload)
	}{
		{
			name: "owner exceeds max length",
			payload: map[string]interface{}{
				"issue_id": int64(12345),
				"owner":    longString(150), // Exceeds MaxOwnerLength (100)
				"repo":     "testrepo",
				"title":    "Test Issue",
			},
			wantErr: true,
		},
		{
			name: "repo exceeds max length",
			payload: map[string]interface{}{
				"issue_id": int64(12345),
				"owner":    "testowner",
				"repo":     longString(300), // Exceeds MaxRepoLength (256)
				"title":    "Test Issue",
			},
			wantErr: true,
		},
		{
			name: "title gets truncated instead of failing",
			payload: map[string]interface{}{
				"issue_id": int64(12345),
				"owner":    "testowner",
				"repo":     "testrepo",
				"title":    longString(2000), // Exceeds MaxTitleLength (1024)
			},
			wantErr: false,
			check: func(t *testing.T, p *StreamIssuePayload) {
				if len(p.Title) != MaxTitleLength {
					t.Errorf("Title length = %d, want %d (truncated)", len(p.Title), MaxTitleLength)
				}
			},
		},
		{
			name: "body gets truncated instead of failing",
			payload: map[string]interface{}{
				"issue_id": int64(12345),
				"owner":    "testowner",
				"repo":     "testrepo",
				"title":    "Test Issue",
				"body":     longString(600000), // Exceeds MaxBodyLength (500000)
			},
			wantErr: false,
			check: func(t *testing.T, p *StreamIssuePayload) {
				if len(p.Body) != MaxBodyLength {
					t.Errorf("Body length = %d, want %d (truncated)", len(p.Body), MaxBodyLength)
				}
			},
		},
		{
			name: "url gets truncated instead of failing",
			payload: map[string]interface{}{
				"issue_id": int64(12345),
				"owner":    "testowner",
				"repo":     "testrepo",
				"title":    "Test Issue",
				"html_url": longString(3000), // Exceeds MaxURLLength (2048)
			},
			wantErr: false,
			check: func(t *testing.T, p *StreamIssuePayload) {
				if len(p.HTMLURL) != MaxURLLength {
					t.Errorf("HTMLURL length = %d, want %d (truncated)", len(p.HTMLURL), MaxURLLength)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := TransformStreamMessage(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransformStreamMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, payload)
			}
		})
	}
}
