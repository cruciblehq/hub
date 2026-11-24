#!/bin/bash
set -e

# Lists all namespaces from the Hub registry.
#
# Usage:
#   ./scripts/list-namespaces.sh [hub-url]
#
# Arguments:
#   hub-url: Optional Hub URL (defaults to http://hub.cruciblehq.xyz:8080)
#
# Examples:
#   ./scripts/list-namespaces.sh
#   ./scripts/list-namespaces.sh http://hub.cruciblehq.xyz:8080

HUB_URL=${1:-http://hub.cruciblehq.xyz:8080}

echo "Fetching namespaces from $HUB_URL..."
echo ""

RESPONSE=$(curl -X GET "$HUB_URL/namespaces" \
    -H "Accept: application/vnd.crucible.namespace-list.v0+json" \
    -w "\n%{http_code}" \
    --max-time 10 \
    -s)

# Extract HTTP status code (last line) and body (all but last line)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "000" ]; then
    echo "Error: Could not connect to Hub at $HUB_URL"
    echo "  Make sure the Hub server is running"
    exit 1
elif [ "$HTTP_CODE" = "200" ]; then
    echo "$BODY" | jq '.'
    COUNT=$(echo "$BODY" | jq '.namespaces | length')
    echo ""
    echo "Total namespaces: $COUNT"
else
    echo "Error: HTTP $HTTP_CODE"
    echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
    exit 1
fi
