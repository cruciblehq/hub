#!/bin/bash
set -e

# Creates a namespace on the Hub registry.
#
# Usage:
#   ./scripts/create-namespace.sh <namespace> [hub-url]
#
# Arguments:
#   namespace: The name of the namespace to create
#   hub-url:   Optional Hub URL (defaults to http://hub.cruciblehq.xyz)
#
# Examples:
#   ./scripts/create-namespace.sh myorg
#   ./scripts/create-namespace.sh myorg http://localhost:8080

if [ $# -lt 1 ]; then
    echo "Error: namespace name required"
    echo "Usage: $0 <namespace> [hub-url]"
    exit 1
fi

NAMESPACE=$1
HUB_URL=${2:-http://hub.cruciblehq.xyz:8080}

echo "Creating namespace '$NAMESPACE' on $HUB_URL..."

# Create namespace via API
RESPONSE=$(curl -X POST "$HUB_URL/namespaces" \
    -H "Content-Type: application/vnd.crucible.namespace-info.v0+json" \
    -H "Accept: application/vnd.crucible.namespace.v0+json" \
    -d "{\"name\":\"$NAMESPACE\",\"description\":\"\"}" \
    -w "\n%{http_code}" \
    --max-time 10 \
    -s)

# Extract HTTP status code (last line) and body (all but last line)
HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

# Check status code
if [ "$HTTP_CODE" = "000" ]; then
    echo "Error: Could not connect to Hub at $HUB_URL. Make sure the Hub server is running"
    exit 1
elif [ "$HTTP_CODE" = "201" ]; then
    echo "$BODY" | jq .
    echo "Namespace '$NAMESPACE' created successfully"
else
    echo "Error: HTTP $HTTP_CODE"
    echo "$BODY" | jq .
    exit 1
fi
