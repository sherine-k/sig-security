package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// Parse command-line flags
	outputFile := flag.String("output", "", "Output file for JSON results (default: stdout)")
	sourceRepo := flag.String("repo", "kubernetes/kubernetes", "Source repository (owner/name)")
	label := flag.String("label", "vex-unaffected-feed", "Label to search for on kubernetes/kubernetes issues")
	dryRun := flag.Bool("dry-run", true, "If true, only output JSON (don't create issues)")
	maxResults := flag.Int("max-results", 100, "Maximum number of issues to process")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")

	flag.Parse()

	if !*dryRun {
		log.Println("ERROR: Creating issues in kubernetes/.vexflow is not yet implemented")
		log.Println("Please use -dry-run=false only when ready for actual issue creation")
		os.Exit(1)
	}

	if *verbose {
		log.Printf("Bridge binary starting with parameters:")
		log.Printf("  repo: %s", *sourceRepo)
		log.Printf("  label: %s", *label)
		log.Printf("  max-results: %d", *maxResults)
		log.Printf("  dry-run: %v", *dryRun)
		log.Printf("  output: %s", *outputFile)
	}

	// Create GitHub client
	ghClient, err := NewGitHubClient()
	if err != nil {
		log.Fatalf("Failed to create GitHub client: %v", err)
	}

	// Parse repository
	owner, repo, err := ParseOwnerRepo(*sourceRepo)
	if err != nil {
		log.Fatalf("Failed to parse repo: %v", err)
	}

	// Build search query
	query := BuildSearchQuery(*sourceRepo, *label)
	if *verbose {
		log.Printf("Searching for issues with query: %s", query)
	}

	// Search for issues
	issues, err := ghClient.SearchIssues(query)
	if err != nil {
		log.Fatalf("Failed to search for issues: %v", err)
	}

	if *verbose {
		log.Printf("Found %d issues with label %s", len(issues), *label)
	}

	// Process issues and collect vexflow issues
	now := time.Now()
	vexflowIssues := []*VexflowIssue{}
	processedCVEs := make(map[string]int)
	processedComponents := make(map[string]int)
	justificationCounts := make(map[string]int)

	for i, kubeIssue := range issues {
		if i >= *maxResults {
			break
		}

		if *verbose {
			log.Printf("Processing issue #%d: %s", kubeIssue.Number, kubeIssue.Title)
		}

		// Get full issue details with comments
		fullIssue, err := ghClient.GetIssueDetails(owner, repo, kubeIssue.Number)
		if err != nil {
			log.Printf("Warning: Failed to get details for issue #%d: %v", kubeIssue.Number, err)
			continue
		}

		// Extract comment bodies
		commentBodies := make([]string, len(fullIssue.Comments))
		for j, c := range fullIssue.Comments {
			commentBodies[j] = c.Body
		}

		// Extract CVEs from issue
		cves := ExtractCVEs(fullIssue.Title, fullIssue.Body, commentBodies)
		if len(cves) == 0 {
			if *verbose {
				log.Printf("  No CVEs found in issue #%d", fullIssue.Number)
			}
			continue
		}

		if *verbose {
			log.Printf("  Found CVEs: %v", cves)
		}

		// Create a vexflow issue for each CVE
		// For now, we create one issue per CVE per kubernetes/kubernetes issue
		// In future, we might need to deduplicate by CVE+component combo
		for _, cveID := range cves {
			// Use default Kubernetes component
			component := &Component{
				Type:    "golang",
				Name:    "k8s.io/kubernetes",
				Version: "master",
				PURL:    "pkg:golang/k8s.io/kubernetes@master",
			}

			// Render the vexflow issue
			vexIssue, err := RenderVexflowIssue(fullIssue, cveID, component, *sourceRepo+"@thisismaster", now)
			if err != nil {
				log.Printf("Warning: Failed to render vexflow issue for #%d/%s: %v",
					fullIssue.Number, cveID, err)
				continue
			}

			vexflowIssues = append(vexflowIssues, vexIssue)
			processedCVEs[cveID]++

			componentKey := component.PURL
			processedComponents[componentKey]++

			if vexIssue.Justification != "" {
				justificationCounts[vexIssue.Justification]++
			}
		}
	}

	if *verbose {
		log.Printf("Generated %d vexflow issues", len(vexflowIssues))
	}

	// Build summary
	summary := &Summary{
		TotalIssues:           len(vexflowIssues),
		UniqueCVEs:            len(processedCVEs),
		UniqueComponents:      len(processedComponents),
		IssuesByJustification: justificationCounts,
		CVEIDCounts:           processedCVEs,
	}

	// Build output structure
	output := &VexflowIssueList{
		Issues:  vexflowIssues,
		Summary: summary,
		Metadata: &Metadata{
			GeneratedAt:  now,
			SourceRepo:   *sourceRepo,
			VexflowRepo:  "kubernetes/.vexflow",
			TargetBranch: "master",
			Label:        *label,
		},
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal output to JSON: %v", err)
	}

	// Write output
	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, jsonBytes, 0o644); err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		if *verbose {
			log.Printf("Output written to %s", *outputFile)
		}
		fmt.Printf("Output written to %s\n", *outputFile)
	} else {
		fmt.Println(string(jsonBytes))
	}

	// Print summary
	fmt.Fprintf(os.Stderr, "\n=== Summary ===\n")
	fmt.Fprintf(os.Stderr, "Total vexflow issues to be created: %d\n", summary.TotalIssues)
	fmt.Fprintf(os.Stderr, "Unique CVEs: %d\n", summary.UniqueCVEs)
	fmt.Fprintf(os.Stderr, "Unique components: %d\n", summary.UniqueComponents)
	if len(summary.IssuesByJustification) > 0 {
		fmt.Fprintf(os.Stderr, "Issues by justification:\n")
		for just, count := range summary.IssuesByJustification {
			fmt.Fprintf(os.Stderr, "  %s: %d\n", just, count)
		}
	}
}
