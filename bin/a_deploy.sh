#!/bin/bash
cd ~/work/go/toss/exported_pages && rm -rf .git && git init && touch .nojekyll && git add . && git commit -m "Deploy $(date)" && git branch -M gh-pages && git remote add origin git@github.com:Traven-B/artist-app.git && git push -f origin gh-pages
