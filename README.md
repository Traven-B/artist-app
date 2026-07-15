# ArtistApp

👉 **[Try the gallery demo](https://traven-b.github.io/artist-app/artist-gallery.html)**

Perfect for the computer hobbyist or Artbreeder enthusiast!

A small Go + HTMX + Alpine.js app for collecting artist names, building a gallery,
and generating “art by …” prompt fragments for image generation tools.

No JavaScript framework build system, no database, just compile and go.

A Google Gemini API key is optional. The app runs without one, but enabling it
adds AI-generated feature vectors for semantic search and artist similarity.
The key is free to obtain, and the embeddings endpoint is available within
Google's free usage limits for hobby-scale use.

The output of the app is a Gallery page that can generate text like:

```
art by Tom Bagshaw,
art by Picasso,
```

This can be copied to the clipboard and pasted into an Artbreeder prompt.

## Quick start note on API Key

The app can be run without any external API keys.

Without a Gemini API key you can still:

- add and edit artists
- upload images
- browse the gallery
- use existing precomputed similarity vectors

When a Gemini API key is available, the app generates embeddings for new or
updated artist feature descriptions and enables semantic search.

If embedding generation fails, artist data is still saved. The missing embedding
is recorded for later repair.

A Gemini API key is free to obtain and free within Google's usage limits.

## Demo (GitHub Pages)

Static preview of the UI:

- **Gallery demo:** \\
  <https://traven-b.github.io/artist-app/artist-gallery.html>

- **Index / add-artists page:** \\
  <https://traven-b.github.io/artist-app/index.html>

The GitHub Pages demo is a static frontend snapshot. Backend features require
running the Go application locally.

### Obtaining a Google Gemini API Key

A Gemini API key is required for generating new feature vectors and using semantic search queries.

You can typically obtain one from the Google AI Studio platform (aistudio.google.com). Look for options to generate or manage API keys.
The free tier is sufficient for hobby-scale use.

---

### Setting the GEMINI_API_KEY

Before running the application, you need to set your Google Gemini API key as an environment variable named `GEMINI_API_KEY`. This is crucial for adding new artists with feature vectors and for using the semantic search on the gallery page.

Example (Linux/macOS):
```bash
export GEMINI_API_KEY="YOUR_API_KEY_HERE"
```

## Clone the repo

```bash
git clone --depth 1 https://github.com/Traven-B/artist-app.git
cd artist-app/
```

* * *

## Test data and backups

There is existing test data (text files and thumbnails) in:

-   `test_data/`
-   `test_images/`


Back these up before running the app:

```
./backup_test.sh
```


---

## Build

```
go build
```


This:

-   Compiles the Go source into a binary named `artistapp`
-   Automatically fetches required packages listed in `go.mod`


Tip: after editing imports, you can run:

```
go mod tidy
```

to sync `go.mod` with the code.

---

## Run (test mode)

Instead of a command-line flag, the app uses an environment variable to switch to test data and images:

```
TEST_MODE=true ./artistapp
```

You should see something like:

```
2026/02/02 17:44:38 Using data dir: test_data, images dir: test_images
2026/02/02 17:44:38 Listening on http://localhost:8080
```

Open this in your browser:

```
http://localhost:8080
```

You’ll land on the **index page**, with a link to the **gallery page**.


---

## Handy test command

When iterating, this is useful:

```
go build && ./restore_test.sh && TEST_MODE=true ./artistapp
```

That means:

-   build
-   restore test backups
-   run the server \\
    (each step only runs if the previous one succeeded)


---

## Index page (todo list + add form)

The index page is a todo list of artist names yet to be added to the master list.

Where does that list come from?

-   It lives in `data/artists_to_add.txt`.

-   You can add one or more names (one per line, no commas) via the **Artists To Add** text area on the index page while the app is running.


### Important caveat

You **should not manually edit `artists_to_add.txt` with a text editor while the app is running**.

Why?

-   When you consume or add a name via the app, the in-memory todo list is rewritten back to disk.
-   Any manual edits made to the file while the app is running would be clobbered.

Workflow workaround:

-   Keep a separate scratch file (for example `names.txt`).

-   When the app is running, paste lines from your scratch file into the bulk add text area on the index page.

-   Append new names manually to `artists_to_add.txt` only **when the app is not running**.

### What the form does

-   Adds artist name, description, and thumbnail to the master list

-   Fetches and decodes **JPG images only**

-   Removes the consumed name from:

    -   the rendered todo list

    -   in-memory state

    -   `data/artists_to_add.txt` on disk


If you slightly change the spelling of a name from the todo list, the original
entry is still matched and removed.

You can also enter a name directly (not from the todo list).

No matter how a name is entered (from the todo list "Add" button, or typed directly, or even a corrected spelling),
always press "Check Duplicates" as the next step. This is crucial for:
-   Ensuring the name isn't already in the master list or todo list.
-   Refreshing the Google search links with the current name.
Be aware that if a duplicate name is detected after entering data into other form fields,
the form will refresh, and you might lose unsaved input.

### Duplicate checking

-   `Check Duplicates`:

    -   trims whitespace

    -   lowercases names

    -   compares against lowercased master list entries


#### Image Upload and URLs

When searching for images, if you use the Google Image Search 'JPG' link, you can right-click on a desired image in the results panel (e.g., in Firefox) and select "Open image in new tab". Copying the URL from this new tab and pasting it into the "Image (File or URL)" field will, with near certainty, allow the application to successfully fetch and decode the image to create a thumbnail.


If a duplicate is found and you don’t want to use the todo-list name:

-   `Delete (and clear)` removes it from the todo list and clears the form

-   `Clear` just clears the form


The Google search links in the form are populated after:

-   selecting a name from the todo list, or

-   running `Check Duplicates`


If you fix spelling, run `Check Duplicates` again to refresh links.

#### AI and JPG links?

Note the JPG link, that will open Google Images search for that artist in image search, asking for only JPGs.

Note the AI link (for Description) and the new **AI Features** link. These links open Google AI mode searches in new tabs. Clicking them will generate telegraphic text relevant to the artist's style or a detailed "Visual DNA" profile. You can then copy this generated text and paste it into the respective "Description" or "Features" text areas in the form. **The results of the "AI Features" search should be pasted into the "Features" text input area.**

The 'Features' data is preserved and can be edited. See `TODO.md` for ideas on how to further utilize this augmented feature data.


### Why this page feels “modern”

The fun feature of this todo list and form page, is we use HTMX to get new renderings of the form and/or todo list, without doing a full page reload. So, we don’t scroll to top of page. A simple, old-fashioned trick to make it look like we are in the 21st century, and no real JS dependencies, build steps and so on.

* * *

## Gallery page

Once rendered, the gallery page is static.

### Interactive Gallery Features

The gallery provides several interactive ways to build your prompt list:

*   **Check artist cards**: Simply checking an artist card adds it to the working list at the bottom of the page.
*   **Ctrl-Click on artist cards**: Performing a `Ctrl-Click` on an artist card will add that artist along with its three most semantically similar artists to the working list at the bottom of the page.
*   **Semantic Search**: Enter an arbitrary search term into the search input bar above the working list and click "Search". The application will find artists whose features are semantically similar to your search term and add the top matches to the working list.

### Working list + copy flow


Selecting text is awkward due to checkboxes and controls, so:

*   **Copy Checked**: This button copies only the text from the checked "art by..." lines in the working list to your clipboard.

You can:

*   reorder names with up/down arrows
*   include some or all of the working set (by checking/unchecking the artist's entry)

If order matters in your prompt, you control it here.

### Navigation tricks


- Clicking an artist name jumps back to that gallery card
- Selecting the radio button shows:
    - thumbnail
    - name
    - description
    - Google search link in a side panel

New selections append to the working list.

To remove a name from the working list:

1. Click the artist name (jump to the gallery)
2. Uncheck it
3. Scroll down and click Generate Prompt again

To clear everything:

- Click Clear

This behavior is driven by Alpine.js, with no build pipeline.

### Styling

- Uses pico.css (via a fork: daft.css)
- Additional styles live as partials in scss/

You **do not**  need Sass to build or run the app.

If you want to adjust styling:

- Install Dart Sass (standalone binary, no JS package manager required)
- Rebuild the CSS

This app isn’t perfect, but it does what it needs to do:
add names, show a gallery, and generate usable prompt text quickly.






