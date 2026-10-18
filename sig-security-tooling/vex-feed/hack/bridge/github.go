package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v60/github"
)

// GitHubClient handles interactions with the GitHub API
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new GitHub API client using the GITHUB_TOKEN environment variable
func NewGitHubClient() (*GitHubClient, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable not set")
	}

	// Create authenticated client
	client := github.NewClient(nil).WithAuthToken(token)

	return &GitHubClient{
		client: client,
	}, nil
}

// SearchIssues searches for issues using GitHub's search API with the given query
func (gc *GitHubClient) SearchIssues(query string) ([]KubernetesIssue, error) {
	ctx := context.Background()

	// Search for issues
	result, _, err := gc.client.Search.Issues(ctx, query, &github.SearchOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("searching for issues: %w", err)
	}

	// Convert github.Issue to KubernetesIssue
	issues := make([]KubernetesIssue, len(result.Issues))
	for i, ghIssue := range result.Issues {
		issues[i] = KubernetesIssue{
			Number: ghIssue.GetNumber(),
			Title:  ghIssue.GetTitle(),
			Body:   ghIssue.GetBody(),
			URL:    ghIssue.GetHTMLURL(),
			State:  ghIssue.GetState(),
		}
	}

	return issues, nil
}

// GetIssueDetails fetches detailed information about a specific issue,
// including all comments
func (gc *GitHubClient) GetIssueDetails(owner, repo string, issueNumber int) (*KubernetesIssue, error) {
	ctx := context.Background()

	// Fetch the issue
	ghIssue, _, err := gc.client.Issues.Get(ctx, owner, repo, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("fetching issue #%d: %w", issueNumber, err)
	}

	// Fetch all comments for this issue
	ghComments, _, err := gc.client.Issues.ListComments(ctx, owner, repo, issueNumber, &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("fetching comments for issue #%d: %w", issueNumber, err)
	}

	// Convert to KubernetesIssue
	issue := &KubernetesIssue{
		Number: ghIssue.GetNumber(),
		Title:  ghIssue.GetTitle(),
		Body:   ghIssue.GetBody(),
		URL:    ghIssue.GetHTMLURL(),
		State:  ghIssue.GetState(),
		Comments: make([]struct {
			Body string `json:"body"`
		}, len(ghComments)),
	}

	// Convert comments
	for i, comment := range ghComments {
		issue.Comments[i] = struct {
			Body string `json:"body"`
		}{
			Body: comment.GetBody(),
		}
	}

	return issue, nil
}

// BuildSearchQuery constructs a GitHub search query for vex-unaffected-feed issues
func BuildSearchQuery(repo, label string) string {
	return fmt.Sprintf("is:issue label:%s repo:%s", label, repo)
}

// ParseOwnerRepo splits a repo string (e.g., "kubernetes/kubernetes") into owner and repo
func ParseOwnerRepo(repoStr string) (owner, repo string, err error) {
	parts := strings.Split(repoStr, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repo format: %s (expected owner/repo)", repoStr)
	}
	return parts[0], parts[1], nil
}
