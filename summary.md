# ArtistApp Summary

ArtistApp is a lightweight Go, HTMX, and Alpine.js application designed to collect artist names, construct a visual gallery, and generate "art by ..." prompt fragments for image generation tools (like Artbreeder).

## Key Features

- **Tech Stack:** Go (backend), HTMX (dynamic, partial page updates), Alpine.js (interactive gallery selection), and Pico.css/daft.css (styling). It requires no heavy database or JavaScript build toolchains.
- **AI Integration (Optional):** Supports Google's Gemini API (`GEMINI_API_KEY`) to generate text feature vector embeddings, enabling semantic searches and finding artist style similarities.
- **Todo List & Add Page (Index):** Manage missing artist names via `data/artists_to_add.txt`. Allows for validation, duplicate checking, custom features extraction, and downloading JPEG artist thumbnails.
- **Interactive Gallery:** Supports choosing artist cards, copying prompt lists to the clipboard, reordering items, and running local semantic search queries over the saved artists.

## Setup & Running

- Run backups: `./backup_test.sh`
- Build: `go build`
- Run in test mode: `TEST_MODE=true ./artistapp`
