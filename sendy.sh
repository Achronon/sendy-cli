#!/usr/bin/env bash
set -euo pipefail

SENDY_URL="${SENDY_URL:-https://sendy.md}"

usage() {
  echo "Usage: sendy [FILE]"
  echo ""
  echo "Paste text to sendy.md and get a shareable link."
  echo ""
  echo "  sendy FILE        Upload file contents"
  echo "  command | sendy    Pipe stdin"
  echo ""
  echo "Environment:"
  echo "  SENDY_URL   Override API base (default: https://sendy.md)"
  exit 1
}

# Check dependencies
for cmd in curl jq; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "error: $cmd is required but not installed" >&2
    exit 1
  fi
done

# Read content from file arg or stdin
if [ $# -gt 0 ]; then
  if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    usage
  fi
  if [ ! -f "$1" ]; then
    echo "error: file not found: $1" >&2
    exit 1
  fi
  content=$(cat "$1")
else
  if [ -t 0 ]; then
    echo "error: no input — pipe something or pass a file" >&2
    usage
  fi
  content=$(cat)
fi

# POST to API
response=$(echo "$content" | jq -Rs '{content: .}' | \
  curl -s -w "\n%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    -d @- \
    "${SENDY_URL}/api/pastes")

http_code=$(echo "$response" | tail -1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" != "201" ]; then
  error=$(echo "$body" | jq -r '.error // "unknown error"')
  echo "error: $error" >&2
  exit 1
fi

url=$(echo "$body" | jq -r '.url')

# Copy to clipboard if available
if command -v pbcopy &>/dev/null; then
  echo -n "$url" | pbcopy
  echo "$url (copied to clipboard)"
else
  echo "$url"
fi
