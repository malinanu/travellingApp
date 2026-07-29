#!/bin/sh
set -e

/app/importspots
exec /app/api
