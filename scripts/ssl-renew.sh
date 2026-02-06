#!/bin/bash
# SSL Certificate Auto-Renewal Script
# Run this via cron or systemd timer

# Configuration
API_URL="${MICROPANEL_URL:-http://localhost:8080}"
API_KEY="${MICROPANEL_API_KEY:-}"

# Check if API key is set
if [ -z "$API_KEY" ]; then
    echo "Error: MICROPANEL_API_KEY environment variable is not set"
    exit 1
fi

# Try to renew certificates via API
echo "Starting SSL certificate renewal..."
response=$(curl -s -X POST \
    -H "Authorization: Bearer $API_KEY" \
    "$API_URL/ssl/renew")

if echo "$response" | grep -q "renewed"; then
    echo "SSL certificates renewed successfully"
else
    echo "SSL renewal response: $response"
fi

# Alternative: Direct certbot renewal
# Uncomment if you prefer direct certbot instead of API

# certbot renew --quiet
# if [ $? -eq 0 ]; then
#     echo "Certbot renewal successful"
#     nginx -t && nginx -s reload
# else
#     echo "Certbot renewal failed"
#     exit 1
# fi

echo "Done."
