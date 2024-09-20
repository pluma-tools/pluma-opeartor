#!/bin/bash

set -euo pipefail

# Enable debug logging
debug_log() {
    echo "[DEBUG] $1" >&2
}

debug_log "Script started"

# Define paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_YAML="${SCRIPT_DIR}/../istio/sample-charts/istio/Chart.yaml"
OUTPUT_FILE="${SCRIPT_DIR}/../istio/sample-charts/istio/values.yaml"

debug_log "CHART_YAML set to $CHART_YAML"
debug_log "OUTPUT_FILE set to $OUTPUT_FILE"

# Function to fetch values for a component
fetch_values() {
    local component=$1
    local version=$2
    debug_log "Fetching values for component: $component, version: $version"
    if [ "$component" == "istiod" ]; then
        component="istio-control/istio-discovery"
    elif [ "$component" == "cni" ]; then
        component="istio-cni"
    fi

    local url="https://raw.githubusercontent.com/istio/istio/${version}/manifests/charts/${component}/values.yaml"
    debug_log "Fetching from URL: $url"
    curl -sSL "$url" | sed '/^defaults:/,/^[^ ]/s/^  //g' | sed '/^defaults:/d'
}

# Initialize the output file
debug_log "Initializing output file: $OUTPUT_FILE"
echo "# Generated Istio values" >"$OUTPUT_FILE"

# Process dependencies
debug_log "Processing dependencies from $CHART_YAML"
dependencies=$(yq e '.dependencies' "$CHART_YAML")
for i in $(seq 0 $(($(echo "$dependencies" | yq e 'length' -) - 1))); do
    name=$(echo "$dependencies" | yq e ".[$i].name" -)
    version=$(echo "$dependencies" | yq e ".[$i].version" -)
    condition=$(echo "$dependencies" | yq e ".[$i].condition" -)

    debug_log "Processing dependency: $name (version $version)"

    # Fetch values for the component
    component_values=$(fetch_values "$name" "$version")

    # Append component values to output file
    debug_log "Appending component values for $name to $OUTPUT_FILE"
    echo "" >>"$OUTPUT_FILE"
    echo "# Values for $name" >>"$OUTPUT_FILE"
    echo "$name:" >>"$OUTPUT_FILE"
    echo "$component_values" | sed 's/^/  /' >>"$OUTPUT_FILE"

    # output
    echo "$component_values" >>"${SCRIPT_DIR}/../out/values-${name}.yaml"

    # Add condition if present
    if [ -n "$condition" ]; then
        debug_log "Adding condition for $name"
        yq e -i ".$name.enabled = false" "$OUTPUT_FILE"
    fi
done

# Validate the generated file
if ! yq e 'has("global")' "$OUTPUT_FILE" >/dev/null 2>&1; then
    debug_log "Error: Generated YAML is invalid"
    echo "Error: Generated YAML is invalid" >&2
    exit 1
fi

debug_log "Generated values file: $OUTPUT_FILE"
debug_log "Script completed"
