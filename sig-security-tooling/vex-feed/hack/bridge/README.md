# Bridge Binary: Translating Kubernetes Issues to Vexflow Format

## Overview

The bridge binary translates issues labeled `vex-unaffected-feed` from `kubernetes/kubernetes` into vexflow-formatted issues suitable for creation in `kubernetes/.vexflow`. This is a key component of the VEX publishing pipeline described in the [KEP](../kep_publish_vex.md).

**Current Status**: Alpha (iteration 1)
- ✅ Fetches and parses `kubernetes/kubernetes` issues
- ✅ Extracts CVE IDs and justifications
- ✅ Generates vexflow-formatted issue JSON
- ⏳ **Currently outputs JSON only (no issue creation yet)**

## Usage

### Basic Usage (Dry-Run - Default)

```bash
# Build the binary
go build -o bridge .

# Run in dry-run mode (outputs JSON)
./bridge --output output.json
```

### Output Files

The bridge generates a JSON file containing:

1. **Issues Array**: Each potential `.vexflow` issue with:
   - Title (CVE ID)
   - Body (with embedded vexflow metadata and instructions)
   - Labels (e.g., `vex-unaffected-feed`)
   - Source information (link back to original k/k issue)
   - Extracted metadata (CVEs, component, justification)
   - Vexflow data structure for parsing

2. **Summary**: Statistics about the collection:
   - Total issues to be created
   - Unique CVEs processed
   - Unique components identified
   - Distribution by justification type

3. **Metadata**: Context about when/how the output was generated

### Command-Line Flags

```
  -label string
        Label to search for on kubernetes/kubernetes issues (default "vex-unaffected-feed")
  -max-results int
        Maximum number of issues to process (default 100)
  -output string
        Output file for JSON results (default: stdout)
  -repo string
        Kubernetes repository (owner/name) (default "kubernetes/kubernetes")
  -verbose
        Enable verbose logging
  -dry-run
        If true, only output JSON (default true; currently always true)
```

### Examples

```bash
# Output to stdout
./bridge --verbose

# Save to file
./bridge --output vexflow_issues.json --max-results 50

# Process custom repository with custom label
./bridge \
  --repo my-org/my-repo \
  --label my-vex-label \
  --output output.json \
  --verbose
```

## Environment Variables

### Required

- **`GITHUB_TOKEN`**: GitHub personal access token with repo read access to `kubernetes/kubernetes`
  - Needed scopes: `public_repo` (for public repos) or `repo` (for private repos)

### Example Setup

```bash
export GITHUB_TOKEN=$(gh auth token)
./bridge --output output.json
```

## Output Format

### Full JSON Structure

```json
{
  "issues": [
    {
      "title": "CVE-2026-25681",
      "body": "# Exploitability triage for CVE-2026-25681\n\nThis issue tracks...",
      "labels": ["vex-unaffected-feed"],
      "source_repo": "github.com/kubernetes/kubernetes",
      "source_issue_number": 139221,
      "source_url": "https://github.com/kubernetes/kubernetes/issues/139221",
      "cve_ids": ["CVE-2026-25681"],
      "component": {
        "type": "golang",
        "name": "k8s.io/kubernetes",
        "version": "master",
        "purl": "pkg:golang/k8s.io/kubernetes@master"
      },
      "justification": "vulnerable_code_not_present",
      "justification_details": "The code path is not used in Kubernetes.",
      "vexflow_data": {
        "branch_id": "github.com/kubernetes/kubernetes@master",
        "backend_id": "",
        "vulnerability": {
          "id": "CVE-2026-25681",
          "aliases": ["CVE-2026-25681"],
          "component": {
            "type": "golang",
            "name": "k8s.io/kubernetes",
            "version": "master",
            "purl": "pkg:golang/k8s.io/kubernetes@master"
          }
        },
        "branch": {
          "repo": "github.com/kubernetes/kubernetes",
          "name": "master"
        },
        "status": "WAITING_USER",
        "created_at": "2026-01-09T18:30:00.000Z"
      }
    }
  ],
  "summary": {
    "total_issues": 1,
    "unique_cves": 1,
    "unique_components": 1,
    "issues_by_justification": {
      "vulnerable_code_not_present": 1
    },
    "cve_id_counts": {
      "CVE-2026-25681": 1
    }
  },
  "metadata": {
    "generated_at": "2026-01-09T18:30:00.000Z",
    "kubernetes_repo": "kubernetes/kubernetes",
    "vexflow_repo": "kubernetes/.vexflow",
    "target_branch": "master",
    "label": "vex-unaffected-feed"
  }
}
```

## Processing Logic

### 1. Issue Discovery

- Searches for issues in `kubernetes/kubernetes` with the label `vex-unaffected-feed`
- Uses GitHub Search API with query: `is:issue label:vex-unaffected-feed repo:kubernetes/kubernetes`

### 2. CVE Extraction

- Scans issue title, body, and all comments for CVE patterns
- Pattern: `CVE-YYYY-NNNNN` (case-insensitive)
- Deduplicates and sorts CVE IDs

### 3. Justification Extraction

- Looks for vexflow slash commands in comments:
  - `/not_affected:component_not_present`
  - `/not_affected:vulnerable_code_not_present`
  - `/not_affected:vulnerable_code_not_in_execute_path`
  - `/not_affected:vulnerable_code_cannot_be_controlled_by_adversary`
  - `/not_affected:inline_mitigations_already_exist`
- Extracts the justification type and any supporting details

### 4. Vexflow Metadata Generation

For each CVE found in an issue:
- Creates a vexflow issue with title = CVE ID
- Generates an issue body that includes:
  - Human-readable triage instructions
  - Reference to the source `kubernetes/kubernetes` issue
  - Embedded JSON metadata in HTML comment format (see below)
  - Examples of slash commands maintainers can use

### 5. Embedded Metadata Format

The issue body includes hidden JSON data in an HTML comment:

```html
<!--
### VEXFLOW==DATA ###
{"branch_id":"github.com/kubernetes/kubernetes@master","backend_id":"","vulnerability":{"id":"CVE-2026-25681",...},...}
### VEXFLOW==DATA ###
-->
```

This metadata matches the format expected by [vexflow](https://github.com/carabiner-dev/vexflow/blob/main/pkg/triage/github/embed.go) and includes:
- `branch_id`: Identifier of the target branch
- `vulnerability`: CVE ID, aliases, and component info
- `branch`: Repository and branch name
- `status`: Triage status (initially `WAITING_USER`)

## Integration with Vexflow

This bridge output integrates with the [vexflow](https://github.com/carabiner-dev/vexflow) tool:

1. **Issue Creation**: The JSON output from the bridge can be fed to a future tool that creates these issues in `kubernetes/.vexflow`
2. **Vexflow Processing**: Once issues exist in `.vexflow`, the vexflow tool can:
   - Parse the embedded metadata
   - Process slash commands added to issue comments
   - Generate OpenVEX documents
   - Sign and publish attestations via sigstore

## Testing

### Unit Tests

All core functionality is covered by unit tests:

```bash
go test -v
```

Tests cover:
- CVE extraction and deduplication
- Justification extraction from comments
- Metadata generation
- Issue rendering

### Manual Testing

To test without GitHub API calls, you can:

1. Check the generated JSON structure for correctness
2. Verify the embedded metadata can be parsed (use `jq` or similar JSON tools)
3. Validate the issue body format by checking for expected sections

Example:

```bash
# Check that embedded metadata is valid JSON
./bridge --max-results 5 | \
  jq '.issues[0].vexflow_data' | \
  jq .
```

## Future Work

### Phase 1 (Current): Alpha v1
- ✅ Fetch and parse kubernetes/kubernetes issues
- ✅ Extract CVEs and justifications
- ✅ Generate vexflow-formatted JSON output
- ⏳ **Remaining**: Implement issue creation in `.vexflow`

### Phase 2: Alpha v1.5 / Beta
- Support additional VEX statuses (`affected`, `fixed`)
- Consume `official-cve-feed` labeled issues from `kubernetes/kubernetes`
- Extract component information from OSV JSON blobs in issue bodies
- Generate `affected` and `fixed` VEX statements

### Phase 3: Future Enhancements
- Automated deduplication based on CVE + component combo
- Support for release branches (not just master)
- Integration with the prow job pipeline
- Consumer documentation for scanning tools

## Architecture Notes

### Key Components

1. **CVE Extraction** (`cve.go`):
   - Regex-based CVE pattern matching
   - Justification extraction from slash commands
   - Normalization of justification labels

2. **Vexflow Metadata** (`vexflow.go`):
   - Building the vexflow metadata structure
   - Generating human-readable issue bodies
   - Rendering complete vexflow issues

3. **GitHub Integration** (`github.go`):
   - GitHub API client with token-based auth
   - Issue search and detail fetching
   - Comment aggregation

4. **Main Logic** (`main.go`):
   - Orchestration of the pipeline
   - JSON output generation
   - Summary statistics

### Data Structures

See `types.go` for:
- `VexflowIssue`: Complete issue data ready for creation
- `VexflowMetadata`: The embedded JSON structure
- `Component`: PURL-based component identification
- `VexflowIssueList`: Output container with issues, summary, and metadata

## Troubleshooting

### "GITHUB_TOKEN environment variable not set"

```bash
export GITHUB_TOKEN=$(gh auth token)
```

### "GitHub API error (status 401): Bad credentials"

- Check that your token is valid: `gh auth status`
- Ensure token has repo read access: `gh auth refresh -s repo`

### No issues found

- Verify the label exists in `kubernetes/kubernetes`: 
  ```bash
  gh issue list --repo kubernetes/kubernetes --label vex-unaffected-feed --limit 5
  ```
- Check that issues have been properly labeled by maintainers

### Issues with CVE extraction

- Enable verbose mode to see which issues are being processed:
  ```bash
  ./bridge --verbose --max-results 10
  ```
- Verify CVE format in issue title/body (must match `CVE-YYYY-NNNNN`)

## References

- [Kubernetes VEX KEP](../kep_publish_vex.md)
- [GitHub Issue #208](https://github.com/kubernetes/sig-security/issues/208)
- [Vexflow Repository](https://github.com/carabiner-dev/vexflow)
- [OpenVEX Specification](https://github.com/openvex/spec)
