# ArtistApp

A small Go + HTMX + Alpine.js app for collecting artist names, building a gallery,
and generating “art by …” prompt fragments for tools like Artbreeder.

The 'output' of the app is to have the Gallery page generate text like:

```
art by Tom Bagshaw,
art by Picasso,
```

This can be copied to the clipboard and then pasted into an Artbreeder prompt.


## Demo (GitHub Pages)

- **Gallery demo (works):** \\
  <https://traven-b.github.io/artist-app/Gallery.html>

- **Index / add-artists page (working except for delete buttons.):** \\
  <https://traven-b.github.io/artist-app/index.html>

Notes:

- The Gallery page works as intended, even as gh-pages demo.
- On the index page,
    - when served by app, the **Delete** buttons are not wired up.\\
      add name to form, do delete (and clear) button to remove name in list.
    - when served by gh-pages, the example index page only has working google search links.
- There is no update/edit flow and no delete for existing gallery entries.
- This is good enough for the intended workflow.

---

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

-   It lives in `data/artists_to_add.txt`

-   Whenever you stumble on a name, add it there


### Important caveat

You **should not edit `artists_to_add.txt` while the app is running**.

Why?

-   When you consume a name via the app, the in-memory todo list is rewritten
    back to disk
-   Any manual edits made while the app is running would be clobbered

Workflow workaround:

-   Keep a separate scratch file (for example `names.txt`)

-   Append new names to `artists_to_add.txt` **when the app is not running**


This is inconvenient, but works well enough.

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

### Duplicate checking

-   `Check Duplicates`:

    -   trims whitespace

    -   lowercases names

    -   compares against lowercased master list entries


If a duplicate is found and you don’t want to use the todo-list name:

-   `Delete (and clear)` removes it from the todo list and clears the form

-   `Clear` just clears the form


The Google search links in the form are populated after:

-   selecting a name from the todo list, or

-   running `Check Duplicates`


If you fix spelling, run `Check Duplicates` again to refresh links.

#### AI and JPG links?

Note the JPG link, that will open Google Images search for that artist in image search, asking for only JPGs.

Note the AI link. It does a search using Google AI mode and asks for a short description of the style of the artist. Clever me.  So, makes a good enough block of text for the description text area of the form. So you can spend time finding a jpg that will work.


### Why this page feels “modern”

The fun feature of this todo list and form page, is we use HTMX to get new renderings of the form and/or todo list, without doing a full page reload. So, we don’t scroll to top of page. A simple, old-fashioned trick to make it look like we are in the 21st century, and no real JS dependencies, build steps and so on.

* * *

## Gallery page

Once rendered, the gallery page is static.

Main idea:

-   Check artist cards

-   Click **Generate Prompt**

-   Get text like:


```
art by Tom Bagshaw,
art by Picasso,
```

This can be copied to the clipboard and then pasted into an Artbreeder prompt.


### Working list + copy flow


Selecting text is awkward due to checkboxes and controls, so:

Copy Checked copies only the checked art by … lines

You can:

reorder names with up/down arrows

include some or all of the working set

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






