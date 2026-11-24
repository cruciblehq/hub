#!/bin/bash
set -e

# Lists all resources from the Hub registry with details.
#
# Usage:
#   ./scripts/list-resources.sh [options]
#
# Options:
#   --hub-url <url>       Hub URL (defaults to http://hub.cruciblehq.xyz:8080)
#   --namespace <name>    Filter by namespace
#   --type <type>         Filter by resource type (service|template|widget)
#
# Examples:
#   ./scripts/list-resources.sh
#   ./scripts/list-resources.sh --namespace cruciblehq
#   ./scripts/list-resources.sh --type service
#   ./scripts/list-resources.sh --namespace cruciblehq --type service

# Default values
HUB_URL="http://hub.cruciblehq.xyz:8080"
NAMESPACE=""
TYPE=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --hub-url)
            HUB_URL="$2"
            shift 2
            ;;
        --namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        --type)
            TYPE="$2"
            shift 2
            ;;
        -h|--help)
            grep "^#" "$0" | sed 's/^# \?//'
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo "Fetching resources from $HUB_URL..."
if [ -n "$NAMESPACE" ]; then
    echo "  Namespace: $NAMESPACE"
fi
if [ -n "$TYPE" ]; then
    echo "  Type: $TYPE"
fi
echo ""

# If namespace is specified, query that namespace
if [ -n "$NAMESPACE" ]; then
    RESPONSE=$(curl -X GET "$HUB_URL/namespaces/$NAMESPACE/resources" \
        -H "Accept: application/vnd.crucible.resource-list.v0+json" \
        -w "\n%{http_code}" \
        --max-time 30 \
        -s)
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" = "000" ]; then
        echo "Error: Could not connect to Hub at $HUB_URL"
        exit 1
    elif [ "$HTTP_CODE" != "200" ]; then
        echo "Error: HTTP $HTTP_CODE"
        echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
        exit 1
    fi
    
    # Filter by type if specified
    if [ -n "$TYPE" ]; then
        BODY=$(echo "$BODY" | jq --arg type "$TYPE" '.resources |= map(select(.type == $type)) | .resources = .resources')
    fi
    
    echo "$BODY" | jq '.'
    RESOURCE_COUNT=$(echo "$BODY" | jq '.resources | length')
    echo ""
    echo "Total resources: $RESOURCE_COUNT"
else
    # List all namespaces first
    NAMESPACES_RESPONSE=$(curl -X GET "$HUB_URL/namespaces" \
        -H "Accept: application/vnd.crucible.namespace-list.v0+json" \
        -w "\n%{http_code}" \
        --max-time 30 \
        -s)
    
    HTTP_CODE=$(echo "$NAMESPACES_RESPONSE" | tail -1)
    NAMESPACES_BODY=$(echo "$NAMESPACES_RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" = "000" ]; then
        echo "Error: Could not connect to Hub at $HUB_URL"
        exit 1
    elif [ "$HTTP_CODE" != "200" ]; then
        echo "Error: HTTP $HTTP_CODE"
        echo "$NAMESPACES_BODY" | jq '.' 2>/dev/null || echo "$NAMESPACES_BODY"
        exit 1
    fi
    
    # Extract namespace names
    NAMESPACE_LIST=$(echo "$NAMESPACES_BODY" | jq -r '.namespaces[].Name')
    
    # Collect all resources
    ALL_RESOURCES="[]"
    for NS in $NAMESPACE_LIST; do
        RESPONSE=$(curl -X GET "$HUB_URL/namespaces/$NS/resources" \
            -H "Accept: application/vnd.crucible.resource-list.v0+json" \
            -w "\n%{http_code}" \
            --max-time 30 \
            -s 2>/dev/null)
        
        HTTP_CODE=$(echo "$RESPONSE" | tail -1)
        BODY=$(echo "$RESPONSE" | sed '$d')
        
        if [ "$HTTP_CODE" = "200" ]; then
            RESOURCES=$(echo "$BODY" | jq '.resources')
            ALL_RESOURCES=$(echo "$ALL_RESOURCES" | jq --argjson new "$RESOURCES" '. + $new')
        fi
    done
    
    # Filter by type if specified
    if [ -n "$TYPE" ]; then
        ALL_RESOURCES=$(echo "$ALL_RESOURCES" | jq --arg type "$TYPE" 'map(select(.type == $type))')
    fi
    
    # Output results
    echo "$ALL_RESOURCES" | jq '.'
    RESOURCE_COUNT=$(echo "$ALL_RESOURCES" | jq 'length')
    echo ""
    echo "Total resources: $RESOURCE_COUNT"
fi
