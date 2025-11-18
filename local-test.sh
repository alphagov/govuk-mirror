#!/bin/bash

# Local testing script for GOV.UK mirror
set -euo pipefail

# Configuration
SITE="https://www.gov.uk/sitemap.xml"
ALLOWED_DOMAINS="www.gov.uk,assets.publishing.service.gov.uk"
DISALLOWED_URL_RULES="/apply-for-a-licence(/|\$),/business-finance-support(/|\$),/drug-device-alerts\.atom,/drug-safety-update\.atom,/foreign-travel-advice\.atom,/government/announcements\.atom,/government/publications\.atom,/government/statistics\.atom,/licence-finder/,/search(/|\$),\.csv/preview\$"
CONCURRENCY="2"
RATE_LIMIT_TOKEN="your-test-token"
HEADERS="rate-limit-token:${RATE_LIMIT_TOKEN}"

# Local directories
DATA_DIR="./local-mirror-data"
WWW_DOMAIN="www.gov.uk"
ASSETS_DOMAIN="assets.publishing.service.gov.uk"

# Remove existing data directory if it exists and create fresh one
if [ -d "${DATA_DIR}" ]; then
    echo "Removing existing data directory..."
    rm -rf "${DATA_DIR}"
fi
mkdir -p "${DATA_DIR}"

echo "Starting local GOV.UK mirror test..."
echo "Data directory: ${DATA_DIR}"
echo "Site: ${SITE}"

# Export environment variables for the scraper
export SITE
export ALLOWED_DOMAINS
export DISALLOWED_URL_RULES
export CONCURRENCY
export RATE_LIMIT_TOKEN
export HEADERS
export SKIP_VALIDATION="true"  # Skip validation for local testing
export LOG_LEVEL="INFO"        # Enable info logging to see progress
export PROMETHEUS_PUSHGATEWAY_URL="http://localhost:9091"

# Change to data directory
cd "${DATA_DIR}"

echo "Running scraper..."
echo "Note: This will run indefinitely. Press Ctrl+C to stop after it downloads enough files for testing."
echo "Starting in 3 seconds..."
sleep 3

# Build and run the Go scraper
go run ../cmd/main.go

echo "Scraping complete. Checking results..."

# Check what was downloaded
if [ -d "${WWW_DOMAIN}" ]; then
    echo "WWW domain files: $(find "${WWW_DOMAIN}" -type f | wc -l) files"
    echo "Sample files:"
    find "${WWW_DOMAIN}" -type f | head -10
else
    echo "No WWW domain files found"
fi

if [ -d "${ASSETS_DOMAIN}" ]; then
    echo "Assets domain files: $(find "${ASSETS_DOMAIN}" -type f | wc -l) files"
    echo "Sample files:"
    find "${ASSETS_DOMAIN}" -type f | head -10
else
    echo "No assets domain files found"
fi

# Create freshness marker
mkdir -p "${WWW_DOMAIN}"
date -Iseconds > "${WWW_DOMAIN}/last-updated.txt"
echo "Created freshness marker: ${WWW_DOMAIN}/last-updated.txt"

echo "Local mirror test complete!"
echo "You can now inspect the downloaded files in: ${DATA_DIR}"
echo ""
echo "To clean up, run: rm -rf ${DATA_DIR}"