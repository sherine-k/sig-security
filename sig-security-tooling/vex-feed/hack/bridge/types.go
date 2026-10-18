package main

import (
	"encoding/json"
	"time"
)

// VexflowIssue represents an issue to be created in kubernetes/.vexflow
type VexflowIssue struct {
	// Issue fields for .vexflow issue
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Labels []string `json:"labels,omitempty"`

	// Source information (for tracking)
	SourceRepo  string `json:"source_repo"`
	SourceIssue int    `json:"source_issue_number"`
	SourceURL   string `json:"source_url"`

	// Extracted metadata
	CVEIDs               []string   `json:"cve_ids"`
	Component            *Component `json:"component,omitempty"`
	Justification        string     `json:"justification,omitempty"`
	JustificationDetails string     `json:"justification_details,omitempty"`

	// Vexflow embedded metadata
	VexflowData *VexflowMetadata `json:"vexflow_data,omitempty"`
}

// Component represents a software component affected by a CVE
type Component struct {
	Type    string `json:"type"` // e.g., "golang"
	Name    string `json:"name"` // e.g., "golang.org/x/net"
	Version string `json:"version"`
	PURL    string `json:"purl"` // Package URL
}

// VexflowMetadata represents the hidden metadata embedded in the issue body
// This mirrors the structure expected by vexflow as seen in embed.go
type VexflowMetadata struct {
	BranchID      string             `json:"branch_id"`
	BackendID     string             `json:"backend_id"`
	Vulnerability *VulnerabilityInfo `json:"vulnerability"`
	Branch        *BranchInfo        `json:"branch"`
	Status        string             `json:"status"` // e.g., "WAITING_USER"
	CreatedAt     *time.Time         `json:"created_at,omitempty"`
}

// VulnerabilityInfo represents vulnerability details in vexflow metadata
type VulnerabilityInfo struct {
	ID        string     `json:"id"`      // e.g., "CVE-2026-25681"
	Aliases   []string   `json:"aliases"` // Alternative IDs
	Component *Component `json:"component"`
}

// BranchInfo represents branch information in vexflow metadata
type BranchInfo struct {
	Repo string `json:"repo"` // e.g., "github.com/kubernetes/kubernetes"
	Name string `json:"name"` // e.g., "master"
}

// VexflowIssueList represents a collection of vexflow issues to be created
type VexflowIssueList struct {
	Issues   []*VexflowIssue `json:"issues"`
	Summary  *Summary        `json:"summary"`
	Metadata *Metadata       `json:"metadata"`
}

// Summary provides statistics about the collected issues
type Summary struct {
	TotalIssues           int            `json:"total_issues"`
	UniqueCVEs            int            `json:"unique_cves"`
	UniqueComponents      int            `json:"unique_components"`
	IssuesByJustification map[string]int `json:"issues_by_justification"`
	CVEIDCounts           map[string]int `json:"cve_id_counts"` // CVE -> count of issues mentioning it
}

// Metadata provides information about when/how this was generated
type Metadata struct {
	GeneratedAt  time.Time `json:"generated_at"`
	SourceRepo   string    `json:"source_repo"`
	VexflowRepo  string    `json:"vexflow_repo"`
	TargetBranch string    `json:"target_branch"`
	Label        string    `json:"label"`
}

// KubernetesIssue represents a kubernetes/kubernetes GitHub issue (subset of fields)
type KubernetesIssue struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	URL      string `json:"url"`
	State    string `json:"state"`
	Comments []struct {
		Body string `json:"body"`
	} `json:"comments"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// MarshalJSON ensures the VexflowIssueList is marshaled in the proper order
func (vil *VexflowIssueList) MarshalJSON() ([]byte, error) {
	type Alias VexflowIssueList
	return json.MarshalIndent((*Alias)(vil), "", "  ")
}
