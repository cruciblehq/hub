#!/bin/bash
set -e

# Cleanup script for Hub registry - deletes all namespaces, resources, and versions

HUB_URL="${HUB_URL:-http://hub.cruciblehq.xyz:8080}"

echo "Starting cleanup..."

# Get all namespaces
namespaces=$(curl -s "$HUB_URL/namespaces" | jq -r '.namespaces[]?.Name // empty')

if [[ -z "$namespaces" ]]; then
    echo "No namespaces found."
    exit 0
fi

# Process each namespace
for namespace in $namespaces; do
    echo ""
    echo "Processing namespace: $namespace"
    
    # Get all resources in this namespace
    resources=$(curl -s "$HUB_URL/namespaces/$namespace/resources" | jq -r '.resources[].Name')
    
    if [[ -z "$resources" ]]; then
        echo "  No resources in namespace $namespace"
    else
        for resource in $resources; do
            echo "  Processing resource: $resource"
            
            # Get all versions for this resource
            versions=$(curl -s "$HUB_URL/namespaces/$namespace/resources/$resource/versions" | jq -r '.versions[].String')
            
            if [[ -z "$versions" ]]; then
                echo "    No versions found"
            else
                for version in $versions; do
                    echo "    Deleting version: $version"
                    curl -s -X DELETE "$HUB_URL/namespaces/$namespace/resources/$resource/versions/$version" > /dev/null
                done
            fi
            
            # Delete the resource
            echo "    Deleting resource: $resource"
            curl -s -X DELETE "$HUB_URL/namespaces/$namespace/resources/$resource" > /dev/null
        done
    fi
    
    # Delete the namespace
    echo "  Deleting namespace: $namespace"
    curl -s -X DELETE "$HUB_URL/namespaces/$namespace" > /dev/null
done

echo ""
echo "Cleanup complete. All namespaces, resources, and versions deleted."
