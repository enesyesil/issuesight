package parser

import (
	"errors"
	"testing"

	"github.com/issuesight/issuesight/internal/domain"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *domain.IssueURL
		wantErr error
	}{
		// Happy paths
		{
			name:  "valid https url",
			input: "https://github.com/kubernetes/kubernetes/issues/123",
			want:  &domain.IssueURL{Owner: "kubernetes", Repo: "kubernetes", Number: 123},
		},
		{
			name:  "valid http url",
			input: "http://github.com/owner/repo/issues/1",
			want:  &domain.IssueURL{Owner: "owner", Repo: "repo", Number: 1},
		},
		{
			name:  "url without protocol",
			input: "github.com/facebook/react/issues/42",
			want:  &domain.IssueURL{Owner: "facebook", Repo: "react", Number: 42},
		},
		{
			name:  "url with www",
			input: "https://www.github.com/golang/go/issues/999",
			want:  &domain.IssueURL{Owner: "golang", Repo: "go", Number: 999},
		},
		{
			name:  "url with trailing slash",
			input: "https://github.com/owner/repo/issues/123/",
			want:  &domain.IssueURL{Owner: "owner", Repo: "repo", Number: 123},
		},
		{
			name:  "url with query parameters",
			input: "https://github.com/owner/repo/issues/123?query=1",
			want:  &domain.IssueURL{Owner: "owner", Repo: "repo", Number: 123},
		},
		{
			name:  "url with hash fragment",
			input: "https://github.com/owner/repo/issues/123#issuecomment-456",
			want:  &domain.IssueURL{Owner: "owner", Repo: "repo", Number: 123},
		},
		{
			name:  "url with whitespace",
			input: "  https://github.com/owner/repo/issues/123  ",
			want:  &domain.IssueURL{Owner: "owner", Repo: "repo", Number: 123},
		},
		{
			name:  "large issue number",
			input: "https://github.com/owner/repo/issues/999999999",
			want:  &domain.IssueURL{Owner: "owner", Repo: "repo", Number: 999999999},
		},

		// Error cases
		{
			name:    "empty string",
			input:   "",
			wantErr: ErrEmptyURL,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: ErrEmptyURL,
		},
		{
			name:    "non-github host",
			input:   "https://gitlab.com/owner/repo/issues/123",
			wantErr: ErrInvalidHost,
		},
		{
			name:    "github enterprise host",
			input:   "https://github.mycompany.com/owner/repo/issues/123",
			wantErr: ErrInvalidHost,
		},
		{
			name:    "pull request url",
			input:   "https://github.com/owner/repo/pull/123",
			wantErr: ErrPullRequestURL,
		},
		{
			name:    "missing issue number",
			input:   "https://github.com/owner/repo/issues",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "missing issues segment",
			input:   "https://github.com/owner/repo/123",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "repo root only",
			input:   "https://github.com/owner/repo",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "user profile only",
			input:   "https://github.com/owner",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "github root",
			input:   "https://github.com/",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "non-numeric issue number",
			input:   "https://github.com/owner/repo/issues/abc",
			wantErr: ErrInvalidIssueNumber,
		},
		{
			name:    "zero issue number",
			input:   "https://github.com/owner/repo/issues/0",
			wantErr: ErrInvalidIssueNumber,
		},
		{
			name:    "negative issue number",
			input:   "https://github.com/owner/repo/issues/-1",
			wantErr: ErrInvalidIssueNumber,
		},
		{
			name:    "discussions url",
			input:   "https://github.com/owner/repo/discussions/123",
			wantErr: ErrInvalidPath,
		},
		{
			name:    "actions url",
			input:   "https://github.com/owner/repo/actions/runs/123",
			wantErr: ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)

			// Check error
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Parse() expected error %v, got nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			// No error expected
			if err != nil {
				t.Errorf("Parse() unexpected error: %v", err)
				return
			}

			// Check result
			if got.Owner != tt.want.Owner {
				t.Errorf("Parse() Owner = %v, want %v", got.Owner, tt.want.Owner)
			}
			if got.Repo != tt.want.Repo {
				t.Errorf("Parse() Repo = %v, want %v", got.Repo, tt.want.Repo)
			}
			if got.Number != tt.want.Number {
				t.Errorf("Parse() Number = %v, want %v", got.Number, tt.want.Number)
			}
		})
	}
}

func TestIssueURL_FullName(t *testing.T) {
	url := domain.IssueURL{Owner: "kubernetes", Repo: "kubernetes", Number: 123}
	if got := url.FullName(); got != "kubernetes/kubernetes" {
		t.Errorf("FullName() = %v, want kubernetes/kubernetes", got)
	}
}
