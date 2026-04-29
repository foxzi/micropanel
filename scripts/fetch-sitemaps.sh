#!/bin/bash
# Fetch robots.txt and sitemaps from original servers for all sites
# and save them into each site's public directory on panel2.
#
# Run on panel2.nevoua.com:
#   bash /tmp/fetch-sitemaps.sh
#
# Requires: curl, grep, python3 (for XML parsing)

set -euo pipefail

API_URL="http://localhost:8080/api/v1/sites"
API_TOKEN="093765ae98aa17b92f657c0537743b00d3163f6f10c95d7b4d077f722173eefd"
SITES_PATH="/var/www/panel/sites"
TIMEOUT=10
LOG="/tmp/fetch-sitemaps.log"
STATS_OK=0
STATS_SKIP=0
STATS_FAIL=0

log() {
    echo "[$(date '+%H:%M:%S')] $*" | tee -a "$LOG"
}

# Download a file, preserving its path relative to domain root
# Usage: download_to SITE_ID URL
download_to() {
    local site_id="$1"
    local url="$2"
    local dest_dir="$SITES_PATH/$site_id/public"

    # Extract path from URL (e.g., https://example.com/foo/bar.xml -> /foo/bar.xml)
    local url_path
    url_path=$(echo "$url" | sed -E 's|^https?://[^/]+||')
    # Default to /sitemap.xml if path is empty
    [ -z "$url_path" ] && url_path="/sitemap.xml"

    local dest_file="$dest_dir$url_path"
    local dest_subdir
    dest_subdir=$(dirname "$dest_file")

    mkdir -p "$dest_subdir"

    if curl -sfL --max-time "$TIMEOUT" -o "$dest_file" "$url"; then
        log "    OK: $url -> $dest_file ($(wc -c < "$dest_file") bytes)"
        return 0
    else
        rm -f "$dest_file"
        return 1
    fi
}

# Extract sitemap URLs from a sitemap index (XML with <sitemap><loc>...</loc></sitemap>)
# Also handles regular sitemaps that have no sub-sitemaps (returns nothing)
extract_sub_sitemaps() {
    local file="$1"
    # Look for <loc> inside <sitemap> entries (sitemap index format)
    # Use grep+sed for simplicity, works for well-formed XML
    if grep -qi '<sitemapindex' "$file" 2>/dev/null; then
        grep -oP '<loc>\K[^<]+' "$file" 2>/dev/null || true
    fi
}

# Extract Sitemap URLs from robots.txt
extract_sitemaps_from_robots() {
    local file="$1"
    grep -iE '^Sitemap:' "$file" 2>/dev/null | sed -E 's/^[Ss]itemap:\s*//' | tr -d '\r' || true
}

# Process one site
process_site() {
    local site_id="$1"
    local domain="$2"
    local dest_dir="$SITES_PATH/$site_id/public"

    # Try HTTPS first, fall back to HTTP
    local base_url="https://$domain"
    local robots_url="$base_url/robots.txt"

    log "[$site_id] $domain"

    # Download robots.txt
    local robots_file="$dest_dir/robots.txt"
    mkdir -p "$dest_dir"

    local fetched_robots=0
    if curl -sfL --max-time "$TIMEOUT" -o "$robots_file" "$robots_url"; then
        fetched_robots=1
    else
        # Try HTTP
        robots_url="http://$domain/robots.txt"
        if curl -sfL --max-time "$TIMEOUT" -o "$robots_file" "$robots_url"; then
            fetched_robots=1
            base_url="http://$domain"
        fi
    fi

    if [ "$fetched_robots" -eq 0 ]; then
        log "  no robots.txt, trying default /sitemap.xml"
        rm -f "$robots_file"
    else
        # Check it's actually text, not an HTML error page
        if file -b "$robots_file" | grep -qi 'html\|empty'; then
            log "  robots.txt is HTML or empty, skipping"
            rm -f "$robots_file"
            fetched_robots=0
        else
            log "  robots.txt saved ($(wc -c < "$robots_file") bytes)"
        fi
    fi

    # Collect sitemap URLs
    local -a sitemap_urls=()

    if [ "$fetched_robots" -eq 1 ]; then
        while IFS= read -r url; do
            [ -n "$url" ] && sitemap_urls+=("$url")
        done < <(extract_sitemaps_from_robots "$robots_file")
    fi

    # If no sitemaps found in robots.txt, try default location
    if [ ${#sitemap_urls[@]} -eq 0 ]; then
        sitemap_urls=("$base_url/sitemap.xml")
    fi

    local any_sitemap=0

    for sitemap_url in "${sitemap_urls[@]}"; do
        log "  sitemap: $sitemap_url"
        if download_to "$site_id" "$sitemap_url"; then
            any_sitemap=1

            # Check if it's a sitemap index with sub-sitemaps
            local url_path
            url_path=$(echo "$sitemap_url" | sed -E 's|^https?://[^/]+||')
            [ -z "$url_path" ] && url_path="/sitemap.xml"
            local local_file="$dest_dir$url_path"

            local sub_urls
            sub_urls=$(extract_sub_sitemaps "$local_file")

            if [ -n "$sub_urls" ]; then
                log "    sitemap index, downloading sub-sitemaps..."
                while IFS= read -r sub_url; do
                    [ -z "$sub_url" ] && continue
                    # Fix relative URLs
                    if [[ "$sub_url" != http* ]]; then
                        sub_url="$base_url$sub_url"
                    fi
                    log "    sub: $sub_url"
                    download_to "$site_id" "$sub_url" || log "    FAIL: $sub_url"
                done <<< "$sub_urls"
            fi
        else
            log "    FAIL: could not download $sitemap_url"
        fi
    done

    if [ "$any_sitemap" -eq 1 ]; then
        STATS_OK=$((STATS_OK + 1))
    elif [ "$fetched_robots" -eq 1 ]; then
        STATS_SKIP=$((STATS_SKIP + 1))
    else
        STATS_FAIL=$((STATS_FAIL + 1))
    fi
}

# Main
log "=== Fetching sitemaps for all sites ==="
log ""

# Get all sites from API
sites_json=$(curl -sf -H "Authorization: Bearer $API_TOKEN" "$API_URL")

# Parse into id:name pairs
site_list=$(echo "$sites_json" | python3 -c '
import json, sys
data = json.load(sys.stdin)
for s in data:
    if s.get("is_enabled", True):
        print(str(s["id"]) + " " + s["name"])
')

total=$(echo "$site_list" | wc -l)
current=0

while IFS=' ' read -r site_id domain; do
    current=$((current + 1))
    log "--- [$current/$total] ---"
    process_site "$site_id" "$domain" || log "  ERROR processing $domain"
done <<< "$site_list"

log ""
log "=== Done ==="
log "OK (got sitemaps): $STATS_OK"
log "Skip (robots but no sitemaps): $STATS_SKIP"
log "Fail (no robots, no sitemaps): $STATS_FAIL"
log "Total: $total"
