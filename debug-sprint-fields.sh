#!/bin/bash
# Helper script to diagnose sprint statistics issues

echo "=== Bug Butler Sprint Diagnostics ==="
echo ""
echo "This script will run the stats command with debug logging to help"
echo "identify the correct custom field IDs for your Jira instance."
echo ""

# Check if JIRA_API_TOKEN is set
if [ -z "$JIRA_API_TOKEN" ]; then
    echo "❌ JIRA_API_TOKEN is not set"
    echo ""
    echo "Please run: source ~/.zshrc"
    echo "Or set it directly: export JIRA_API_TOKEN='your-token-here'"
    exit 1
fi

echo "✓ JIRA_API_TOKEN is set"
echo ""

# Run stats with debug logging and capture output
echo "Running: ./bin/bug-butler stats --debug 2>&1 | grep -A 5 'Custom fields'"
echo ""
echo "This will show the custom field IDs available in your Jira issues."
echo "Look for fields that contain sprint data (usually customfield_100XX)."
echo ""
echo "================================================================================\n"

./bin/bug-butler stats --debug 2>&1 | head -200 | grep -E "(Custom fields|Sprint|sprint|field_count|No sprints found)"

echo ""
echo "================================================================================\n"
echo ""
echo "If you see 'No sprints found', check the debug output above for:"
echo "  1. Custom field IDs that might contain sprint data"
echo "  2. Whether 'customfield_10020' appears in the list"
echo ""
echo "To find your sprint field ID:"
echo "  1. Go to Jira → Settings → Issues → Custom fields"
echo "  2. Search for 'Sprint' field"
echo "  3. Click on it - the ID will be in the URL"
echo ""
echo "Then update the field ID in:"
echo "  internal/jira/mapper.go:64 (sprint field)"
echo "  internal/jira/mapper.go:86 (story points field)"
