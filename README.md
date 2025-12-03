# Bug Butler

A CLI tool to monitor incoming bugs in Jira projects and identify bugs requiring attention based on configurable SLA rules.

Bug Butler helps you stay on top of your bug backlog by:
- Fetching unresolved bugs from your Jira project
- Evaluating them against priority and status-based SLA thresholds
- Grouping violations into severity buckets (Urgent, Attention Needed, Review Needed)
- Displaying results in color-coded terminal tables

## Features

- **SLA Monitoring**: Define flexible rules based on bug priority, status, and age
- **Bucket Categorization**: Automatically group bugs by severity for easy triage
- **Color-Coded Output**: Visual priority indicators using terminal colors
- **Environment Variables**: Secure API token management via environment variables
- **Configurable**: YAML-based configuration for easy customization
- **Zero Dependencies**: Single binary with no external runtime dependencies

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/neilmpatterson/bug-butler.git
cd bug-butler

# Build the binary
go build -o bug-butler ./cmd/bug-butler

# Move to your PATH (optional)
sudo mv bug-butler /usr/local/bin/
```

### Direct Build

```bash
go install github.com/neilmpatterson/bug-butler/cmd/bug-butler@latest
```

## Quick Start

### 1. Generate Jira API Token

1. Go to https://id.atlassian.com/manage-profile/security/api-tokens
2. Click "Create API token"
3. Give it a name (e.g., "bug-butler")
4. Copy the generated token

### 2. Set Environment Variable

```bash
export JIRA_API_TOKEN="your-api-token-here"
```

Add this to your `~/.bashrc`, `~/.zshrc`, or equivalent for persistence.

### 3. Configure Bug Butler

Copy the sample configuration and customize it:

```bash
cp config.sample.yaml config.yaml
```

Then edit `config.yaml` with your Jira details:

```yaml
jira:
  base_url: "https://yourcompany.atlassian.net"
  email: "your-email@company.com"
  api_token: "${JIRA_API_TOKEN}"

  # Monitor multiple projects (recommended)
  project_keys:
    - "PROJ1"
    - "PROJ2"

  # Or single project (legacy)
  # project_key: "YOURPROJECT"

sla_rules:
  - name: "Critical bugs need immediate triage"
    priority: "Critical"
    status: ["Needs Triage", "To Do"]  # Multiple statuses supported
    max_age_days: 0.25  # 6 hours
    bucket: "🔴 URGENT"
    severity: 1

  - name: "High priority backlog aging"
    priority: "High"
    status: ["Backlog", "To Do", "On Hold"]
    max_age_days: 3
    bucket: "🟡 ATTENTION NEEDED"
    severity: 2
```

**Note**: `config.yaml` is in `.gitignore` to protect your credentials. Only `config.sample.yaml` should be committed to version control.

### 4. Run Bug Butler

```bash
bug-butler check
```

## Configuration Guide

### Jira Settings

| Field | Description | Required |
|-------|-------------|----------|
| `base_url` | Your Jira Cloud URL (e.g., https://yourcompany.atlassian.net) | Yes |
| `email` | Your Jira account email | Yes |
| `api_token` | Jira API token (supports `${VAR}` interpolation) | Yes |
| `project_keys` | Array of Jira project keys/names to monitor | Yes* |
| `project_key` | Single project key (deprecated, use `project_keys`) | Yes* |

\* Either `project_keys` (recommended) or `project_key` must be provided

### SLA Rules

SLA rules are evaluated in order (first-match wins). Each rule defines:

| Field | Description | Type | Required |
|-------|-------------|------|----------|
| `name` | Descriptive name for the rule | string | Yes |
| `priority` | Bug priority to match (e.g., "Critical", "High") | string | No |
| `status` | Bug status(es) to match (e.g., "Backlog", ["Backlog", "To Do"]) | string or array | No |
| `max_age_days` | Maximum age in days before violation (supports decimals) | number | Yes |
| `bucket` | Which bucket to assign violations to | string | Yes |
| `severity` | Bucket display priority (1 = highest) | number | Yes |

**Note:** Status values are case-sensitive and must match your Jira instance exactly. Common statuses include "Backlog", "Needs Triage", "To Do", "In Progress", "On Hold", etc.

### Example SLA Rules

**Urgent Response for Critical Bugs** (6 hours):
```yaml
- name: "Critical bugs need immediate triage"
  priority: "Critical"
  status: "Needs Triage"
  max_age_days: 0.25
  bucket: "🔴 URGENT"
  severity: 1
```

**Prevent Backlog Aging** (3 days):
```yaml
- name: "High priority backlog aging"
  priority: "High"
  status: "Backlog"
  max_age_days: 3
  bucket: "🟡 ATTENTION NEEDED"
  severity: 2
```

**Stale Bugs** (7 days, multiple statuses):
```yaml
- name: "Medium priority stale"
  priority: "Medium"
  status: ["Backlog", "Needs Triage", "To Do"]
  max_age_days: 7
  bucket: "⚪ REVIEW NEEDED"
  severity: 3
```

**Multiple Projects**:
```yaml
jira:
  project_keys:
    - "PROJECT1"
    - "PROJECT2"
    - "PROJECT3"
```

## Usage

### Check Bugs

```bash
# Use default config.yaml in current directory
bug-butler check

# Use custom config file
bug-butler check --config /path/to/config.yaml

# Use short flag
bug-butler check -c my-config.yaml
```

### View Version

```bash
bug-butler version
```

### Get Help

```bash
bug-butler --help
bug-butler check --help
```

## Output Format

Bug Butler displays violations grouped by bucket, sorted by severity. Issue keys are **clickable links** in modern terminals:

```
================================================================================
  BUG BUTLER - SLA VIOLATION REPORT
================================================================================

🔴 URGENT (2 bugs)
╭──────────┬─────────────────────────┬──────────┬──────────────┬─────────╮
│ Key      │ Summary                 │ Priority │ Status       │ Age     │
├──────────┼─────────────────────────┼──────────┼──────────────┼─────────┤
│ PROJ-123 │ Critical login failure  │ Critical │ Needs Triage │ 18.5... │
│ PROJ-125 │ Data loss in export     │ Critical │ Backlog      │ 2.1 ... │
╰──────────┴─────────────────────────┴──────────┴──────────────┴─────────╯

🟡 ATTENTION NEEDED (5 bugs)
[...]

--------------------------------------------------------------------------------
  SUMMARY
--------------------------------------------------------------------------------

Total SLA violations: 7

Breakdown by bucket:
  🔴 URGENT: 2 bugs
  🟡 ATTENTION NEEDED: 5 bugs
```

**Clickable Links:** In supported terminals (iTerm2, VS Code, modern terminals), Cmd+Click or Ctrl+Click on issue keys to open them directly in Jira.

### Exit Codes

- `0`: No SLA violations found
- `1`: One or more SLA violations detected

This makes Bug Butler ideal for CI/CD integration or scheduled monitoring.

## Advanced Usage

### Scheduled Monitoring with Cron

Run Bug Butler every hour:

```bash
# Edit crontab
crontab -e

# Add this line
0 * * * * cd /path/to/bug-butler && ./bug-butler check >> /var/log/bug-butler.log 2>&1
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Check Jira SLAs
  env:
    JIRA_API_TOKEN: ${{ secrets.JIRA_API_TOKEN }}
  run: |
    bug-butler check
```

### Multiple Projects

Create separate config files for each project:

```bash
bug-butler check -c config-projectA.yaml
bug-butler check -c config-projectB.yaml
```

## Troubleshooting

### Authentication Failed

**Error**: `Authentication failed. Check JIRA_API_TOKEN`

**Solution**:
1. Verify `JIRA_API_TOKEN` environment variable is set: `echo $JIRA_API_TOKEN`
2. Confirm email matches your Jira account
3. Regenerate API token if expired

### Project Not Found

**Error**: `Project {key} not found`

**Solution**:
1. Verify project key is correct (case-sensitive)
2. Ensure you have access to the project
3. Check base_url points to the correct Jira instance

### No Bugs Found

**Message**: `No unresolved bugs found in project!`

**Reason**: The project has no unresolved bugs, or you don't have permission to view them.

### SSL/TLS Errors

If you encounter certificate errors with self-hosted Jira, ensure your base_url uses `https://` and the certificate is trusted.

## Development

### Project Structure

```
bug-butler/
├── cmd/bug-butler/        # Application entry point
├── internal/
│   ├── domain/            # Core domain models
│   ├── cli/               # CLI commands
│   ├── config/            # Configuration loading
│   ├── jira/              # Jira API integration
│   ├── sla/               # SLA rule evaluation
│   └── output/            # Terminal output formatting
├── config.sample.yaml     # Sample configuration (copy to config.yaml)
└── README.md
```

### Building

```bash
go build -o bug-butler ./cmd/bug-butler
```

### Testing

```bash
go test ./...
```

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT License - see LICENSE file for details

## Support

For issues, questions, or feature requests, please open an issue on GitHub.

## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Koanf](https://github.com/knadh/koanf) - Configuration management
- [go-jira](https://github.com/andygrunwald/go-jira) - Jira API client
- [go-pretty](https://github.com/jedib0t/go-pretty) - Terminal tables
