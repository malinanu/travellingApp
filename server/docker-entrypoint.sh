#!/bin/sh
set -e

if ! /app/importsrilanka /app/sri_lanka_spots.json; then
  echo "WARNING: importsrilanka failed - starting API anyway with existing data" >&2
fi

exec /app/api
