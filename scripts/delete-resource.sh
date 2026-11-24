#!/bin/bash
set -e

# Deletes a resource and all its versions from the Hub registry.
#
# Usage:
#   ./scripts/delete-resource.sh <namespace> <resource> [options]
#
# Arguments:
#   namespace: The namespace containing the resource
#   resource:  The name of the resource to delete
#
# Options:
#   --hub-url <url>   Hub URL (defaults to http://hub.cruciblehq.xyz:8080)
#   --yes             Skip confirmation prompt
#
# Examples:
#   ./scripts/delete-resource.sh cruciblehq my-service
#   ./scripts/delete-resource.sh cruciblehq my-service --yes
#   ./scripts/delete-resource.sh cruciblehq my-service --hub-url http://localhost:8080

if [ $# -lt 2 ]; then
    echo "Error: namespace and resource name required"
    echo "Usage: $0 <namespace> <resource> [options]"
    exit 1
fi

NAMESPACE=$1
RESOURCE=$2
shift 2

HUB_URL="http://hub.cruciblehq.xyz:8080"
SKIP_CONFIRM=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --hub-url)
            HUB_URL="$2"
            shift 2
            ;;
        --yes|-y)
            SKIP_CONFIRM=true
            shift
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

# Fetch versions to show what will be deleted
RESPONSE=$(curl -X GET "$HUB_URL/namespaces/$NAMESPACE/resources/$RESOURCE/versions" \
    -w "\n%{http_code}" \
    --max-time 10 \
    -s)

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "000" ]; then
    echo "Error: Could not connect to Hub at $HUB_URL"
    exit 1
elif [ "$HTTP_CODE" = "404" ]; then
    echo "Error: Resource '$NAMESPACE/$RESOURCE' not found"
    exit 1
elif [ "$HTTP_CODE" != "200" ]; then
    echo "Error: HTTP $HTTP_CODE"
    if [ -n "$BODY" ]; then
        echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
    fi
    exit 1
fi

VERSION_COUNT=$(echo "$BODY" | jq '.versions | length')
VERSIONS=$(echo "$BODY" | jq -r '.versions[].String // empty')

echo "Resource: $NAMESPACE/$RESOURCE"
echo "Versions: $VERSION_COUNT"
if [ -n "$VERSIONS" ]; then
    echo "$VERSIONS" | sed 's/^/  /'
fi
echo ""

# Confirm deletion
if [ "$SKIP_CONFIRM" = false ]; then
    read -r -p "Delete this resource and all its versions? [y/N] " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        echo "Aborted"
        exit 0
    fi
fi

# Delete all versions first
for version in $VERSIONS; do
    echo "Deleting version $version..."
    curl -s -X DELETE "$HUB_URL/namespaces/$NAMESPACE/resources/$RESOURCE/versions/$version" > /dev/null
done

# Delete the resource
echo "Deleting resource '$NAMESPACE/$RESOURCE'..."
RESPONSE=$(curl -X DELETE "$HUB_URL/namespaces/$NAMESPACE/resources/$RESOURCE" \
    -w "\n%{http_code}" \
    --max-time 10 \
    -s)

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "204" ] || [ "$HTTP_CODE" = "200" ]; then
    echo "Resource '$NAMESPACE/$RESOURCE' deleted successfully"
elif [ "$HTTP_CODE" = "404" ]; then
    echo "Error: Resource '$NAMESPACE/$RESOURCE' not found"
    exit 1
else
    echo "Error: HTTP $HTTP_CODE"
    if [ -n "$BODY" ]; then
        echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
    fi
    exit 1
fi
