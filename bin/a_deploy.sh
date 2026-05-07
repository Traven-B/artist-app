#!/bin/bash

if [ ! -f "$(dirname "$0")/../.artist_app_root" ]; then
  echo "Error: This script must be run from a subdirectory of the project root." >&2
  exit 1
fi

TARGET_DIR="$(cd "$(dirname "$0")/../../gh_artist_pages" && pwd)"
cd "$TARGET_DIR" && rm -rf "$TARGET_DIR/.git" && git init && touch .nojekyll && git add . && git commit -m "Deploy $(date)" && git branch -M gh-pages && git remote add origin git@github.com:Traven-B/artist-app.git && git push -f origin gh-pages
