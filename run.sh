#!/usr/bin/env bash
set -e

MODE="$1"

if [ "$MODE" = "test" ]; then
  go build && ./restore_test.sh && figlet test && TEST_MODE=true ./artistapp

elif [ "$MODE" = "prod" ]; then
  go build && ./backup_prod.sh && figlet PROD && ./artistapp

else
  echo "Usage: $0 [test|prod]"
  exit 1
fi
