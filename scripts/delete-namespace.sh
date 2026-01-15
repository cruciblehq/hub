#!/bin/bash
set -e

# Deletes a namespace from the Hub registry.
#
# Usage:
#   ./scripts/delete-namespace.sh <namespace> [hub-url]
#
# Arguments:
#   namespace: The name of the namespace to delete
#   hub-url:   Optional Hub URL (defaults to http://hub.cruciblehq.xyz)
#
# Examples:
#   ./scripts/delete-namespace.sh myorg
#   ./scripts/delete-namespace.sh myorg http://localhost:8080

if [ $# -lt 1 ]; then
    echo "Error: namespace name required"
    echo "Usage: $0 <namespace> [hub-url]"
    exit 1
fi

NAMESPACE=$1
HUB_URL=${2:-http://hub.cruciblehq.xyz:8080}

echo "Deleting namespace '$NAMESPACE' on $HUB_URL..."

# Delete namespace via API
RESPONSE=$(curl -X DELETE "$HUB_URL/namespaces/$NAMESPACE" \
    -w "\n%{http_code}" \
    --max-time 10 \
    -s)

# Extract HTTP status code (last line) and body (all but last line)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

# Check status code
if [ "$HTTP_CODE" = "000" ]; then
    echo "Error: Could not connect to Hub at $HUB_URL"
    echo "  Make sure the Hub server is running"
    exit 1
elif [ "$HTTP_CODE" = "204" ]; then
    echo "Namespace '$NAMESPACE' deleted successfully"
elif [ "$HTTP_CODE" = "404" ]; then
    echo "Error: Namespace '$NAMESPACE' not found"
    exit 1
else
    echo "Error: HTTP $HTTP_CODE"
    if [ -n "$BODY" ]; then
        echo "$BODY" | jq .
    fi
    exit 1
fi
