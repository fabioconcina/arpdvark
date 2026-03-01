#!/usr/bin/env bash
# Downloads and trims the IEEE OUI database to (hex) lines only.
# Run via: make update-oui

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="$SCRIPT_DIR/../vendor_db/oui.txt"

echo "Downloading IEEE OUI database..."
curl -fsSL https://standards-oui.ieee.org/oui/oui.txt \
  | grep '(hex)' > "$OUT"

COUNT=$(wc -l < "$OUT")
if [ "$COUNT" -lt 1000 ]; then
  echo "Error: downloaded OUI database has only $COUNT entries (expected 10000+). File may be truncated or corrupt." >&2
  rm -f "$OUT"
  exit 1
fi
echo "Done. $COUNT entries written to vendor_db/oui.txt"
