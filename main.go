package main

import (
	// "bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

type ArtistRecord struct {
	ID          int
	Name        string
	Description string
	ImgURL      string
	Thumb       string
}

type FormData struct {
	Name         string
	OriginalName string
	Desc         string
	ImgURL       string

	NameMsg string
	DescMsg string
	ImgMsg  string
}

type EditFormData struct {
	ArtistRecord
	NameMsg string
	DescMsg string
	ImgMsg  string
}

type AddArtistPageData struct {
	ToAdd    []string
	FormData FormData
}

var templates *template.Template

// File-backed data
var globalMasterList []ArtistRecord
var globalToAddList []string

var dataDir = "data"     // Default prod
var imagesDir = "images" // Default prod

// --- File IO ---

func ReadMasterList(filename string) ([]ArtistRecord, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var records []ArtistRecord
	blocks := strings.Split(string(data), "\n\n")
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		var rec ArtistRecord
		for _, line := range lines {
			if strings.HasPrefix(line, "id:") {
				rec.ID, _ = strconv.Atoi(strings.TrimSpace(line[3:]))
			} else if strings.HasPrefix(line, "n:") {
				rec.Name = strings.TrimSpace(line[2:])
			} else if strings.HasPrefix(line, "d:") {
				rec.Description = strings.TrimSpace(line[2:])
			} else if strings.HasPrefix(line, "i:") {
				rec.ImgURL = strings.TrimSpace(line[2:])
			} else if strings.HasPrefix(line, "t:") {
				rec.Thumb = strings.TrimSpace(line[2:])
			}
		}
		records = append(records, rec)
	}
	return records, nil
}

func ReadToAddList(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var names []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

// --- Handlers ---

func addArtistPage(w http.ResponseWriter, r *http.Request) {
	data := AddArtistPageData{
		ToAdd:    globalToAddList,
		FormData: FormData{},
	}

	// not executing add_artist_page , doing flat top index , probably rename everything here eventually
	err := templates.ExecuteTemplate(w, "index", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

func galleryPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Artists []ArtistRecord
	}{Artists: globalMasterList}

	err := templates.ExecuteTemplate(w, "gallery_page", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

// htmx handler: populate form with selected name
func populateFormHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	data := AddArtistPageData{
		FormData: FormData{
			Name:         name,
			OriginalName: name,
			NameMsg:      "",
		},
	}
	// Only render the form partial
	err := templates.ExecuteTemplate(w, "artist_form", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

// htmx handler: check for duplicates and update the whole form
func checkNameHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name")) // <- trim spaces
	originalName := r.FormValue("original_name")
	msg := ""
	// Search master list for duplicate (case-insensitive)
	for _, rec := range globalMasterList {
		if strings.EqualFold(strings.TrimSpace(rec.Name), name) { // <- also trim stored name
			msg = "This name is already in the master list!"
			break
		}
	}

	data := AddArtistPageData{
		FormData: FormData{
			Name:         name,
			OriginalName: originalName,
			NameMsg:      msg,
		},
	}
	// Only render the form partial
	err := templates.ExecuteTemplate(w, "artist_form", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

func cancelAddFormHandler(w http.ResponseWriter, r *http.Request) {
	data := AddArtistPageData{
		FormData: FormData{}, // all fields zeroed/blank
	}
	err := templates.ExecuteTemplate(w, "artist_form", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

// htmx handler: show confirmation dialog for deleting from to-do list
func confirmDeleteTodoHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	data := struct {
		Name string
	}{Name: name}
	err := templates.ExecuteTemplate(w, "confirm_delete_content", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

// htmx handler: actually delete from to-do list and return updated list items
func deleteTodoItemHandler(w http.ResponseWriter, r *http.Request) {
	nameToDelete := strings.TrimSpace(r.FormValue("name"))
	if nameToDelete == "" {
		http.Error(w, "Name is required", 400)
		return
	}

	// Remove name from to-do list
	newList := make([]string, 0, len(globalToAddList))
	for _, name := range globalToAddList {
		if !strings.EqualFold(name, nameToDelete) {
			newList = append(newList, name)
		}
	}
	globalToAddList = newList

	// Save list
	err := os.WriteFile(filepath.Join(dataDir, "artists_to_add.txt"), []byte(strings.Join(globalToAddList, "\n")+"\n"), 0644)
	if err != nil {
		http.Error(w, "Failed to save to-do list", 500)
		return
	}

	// Return updated list items (inner HTML of <ul>)
	data := AddArtistPageData{
		ToAdd: globalToAddList,
	}
	err = templates.ExecuteTemplate(w, "todo_list_items", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

// htmx handler: add one or more names to the to-do list
func addToTodoListHandler(w http.ResponseWriter, r *http.Request) {
	rawNames := r.FormValue("names")
	lines := strings.Split(rawNames, "\n")
	updated := false

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name != "" {
			globalToAddList = append(globalToAddList, name)
			updated = true
		}
	}

	if updated {
		_ = os.WriteFile(filepath.Join(dataDir, "artists_to_add.txt"), []byte(strings.Join(globalToAddList, "\n")+"\n"), 0644)
	}

	data := AddArtistPageData{
		ToAdd: globalToAddList,
	}
	_ = templates.ExecuteTemplate(w, "todo_list_items", data)
}

// htmx handler: decide whether to show confirmation dialog or delete directly
func confirmDeleteTodoFormHandler(w http.ResponseWriter, r *http.Request) {
	originalName := strings.TrimSpace(r.FormValue("original_name"))

	if originalName == "" {
		// No name to delete, just clear form by calling deleteTodoFormHandler directly
		deleteTodoFormHandler(w, r)
		return
	}

	// Name exists → show confirmation dialog
	data := struct {
		Name string
	}{
		Name: originalName,
	}

	// err := templates.ExecuteTemplate(w, "confirm_delete_content", data)
	err := templates.ExecuteTemplate(w, "confirm_delete_and_clear_content", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
		return
	}
}

// htmx handler: delete name from to-do list based on original name inside form
func deleteTodoFormHandler(w http.ResponseWriter, r *http.Request) {
	originalName := strings.TrimSpace(r.FormValue("original_name"))
	if originalName == "" {
		// If called with no original_name (e.g. user typed name manually), we still return response
		data := AddArtistPageData{
			ToAdd:    globalToAddList, // unchanged
			FormData: FormData{},      // blank form to clear
		}
		_ = templates.ExecuteTemplate(w, "submit_response", data)
		return
	}

	// Remove name from to-do list
	newList := make([]string, 0, len(globalToAddList))
	for _, name := range globalToAddList {
		if !strings.EqualFold(name, originalName) {
			newList = append(newList, name)
		}
	}
	globalToAddList = newList

	// Save list
	err := os.WriteFile(filepath.Join(dataDir, "artists_to_add.txt"), []byte(strings.Join(globalToAddList, "\n")+"\n"), 0644)

	if err != nil {
		http.Error(w, "Failed to save to-do list", 500)
		return
	}

	// ✅ return full form + list response via out-of-band swaps
	data := AddArtistPageData{
		ToAdd:    globalToAddList,
		FormData: FormData{}, // clear the form
	}
	err = templates.ExecuteTemplate(w, "submit_response", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

func submitArtistAddFormHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	originalName := strings.TrimSpace(r.FormValue("original_name"))
	desc := strings.TrimSpace(r.FormValue("desc"))
	imgURL := strings.TrimSpace(r.FormValue("img_url"))

	var nameMsg, descMsg, imgMsg string

	// Validation
	if name == "" {
		nameMsg = "Name is required."
	}
	if desc == "" {
		descMsg = "Description is required."
	}
	if imgURL == "" {
		imgMsg = "Image URL is required."
	}

	// Check for duplicate in master list
	for _, rec := range globalMasterList {

		if strings.EqualFold(strings.TrimSpace(rec.Name), name) { // <- also trim stored name
			nameMsg = "This name is already in the master list!"
			break
		}
	}

	// If any validation failed, return form with all values preserved
	if nameMsg != "" || descMsg != "" || imgMsg != "" {
		data := AddArtistPageData{
			ToAdd: globalToAddList,
			FormData: FormData{
				Name:         name,
				OriginalName: originalName,
				Desc:         desc,
				ImgURL:       imgURL,
				NameMsg:      nameMsg,
				DescMsg:      descMsg,
				ImgMsg:       imgMsg,
			},
		}
		_ = templates.ExecuteTemplate(w, "submit_response", data)
		return
	}

	// Generate next ID
	maxID := 0
	for _, rec := range globalMasterList {
		if rec.ID > maxID {
			maxID = rec.ID
		}
	}
	newID := maxID + 1
	thumbFile := fmt.Sprintf("%d-%d.jpg", newID, time.Now().Unix())

	// Try to create thumbnail
	if !thumbnailExists(thumbFile) {
		if err := fetchAndCreateThumbnail(imgURL, thumbFile); err != nil {
			log.Printf("thumbnail error for %s: %v", imgURL, err)
			imgMsg = "Warning: could not create thumbnail from image URL."
			data := AddArtistPageData{
				ToAdd: globalToAddList,
				FormData: FormData{
					Name:         name,
					OriginalName: originalName,
					Desc:         desc,
					ImgURL:       imgURL,
					NameMsg:      nameMsg,
					DescMsg:      descMsg,
					ImgMsg:       imgMsg,
				},
			}
			_ = templates.ExecuteTemplate(w, "submit_response", data)
			return // stop processing further
		}
	}

	// Add artist to master list
	newRec := ArtistRecord{
		ID:          newID,
		Name:        name,
		Description: desc,
		ImgURL:      imgURL,
		Thumb:       thumbFile,
	}
	globalMasterList = append(globalMasterList, newRec)

	// Save master list to disk
	var builder strings.Builder
	for _, rec := range globalMasterList {
		builder.WriteString(fmt.Sprintf("id:%d\nn:%s\nd:%s\ni:%s\nt:%s\n\n", rec.ID, rec.Name, rec.Description, rec.ImgURL, rec.Thumb))
	}
	if err := os.WriteFile(filepath.Join(dataDir, "artists_master.txt"), []byte(builder.String()), 0644); err != nil {
		http.Error(w, "Error writing master list: "+err.Error(), 500)
		return
	}

	// Remove name from to-do list if present
	if originalName != "" {
		newList := make([]string, 0, len(globalToAddList))
		for _, n := range globalToAddList {
			if !strings.EqualFold(n, originalName) {
				newList = append(newList, n)
			}
		}
		globalToAddList = newList
		if err := os.WriteFile(filepath.Join(dataDir, "artists_to_add.txt"), []byte(strings.Join(globalToAddList, "\n")+"\n"), 0644); err != nil {
			http.Error(w, "Error writing to-do list: "+err.Error(), 500)
			return
		}
	}

	// Return updated form (cleared) + updated list via OOB swaps
	data := AddArtistPageData{
		ToAdd:    globalToAddList,
		FormData: FormData{}, // form cleared on success
	}
	_ = templates.ExecuteTemplate(w, "submit_response", data)
}

func deleteArtistHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/artists/delete/")
	id, _ := strconv.Atoi(idStr)

	for i, rec := range globalMasterList {
		if rec.ID == id {
			// Delete the thumbnail file from disk
			if rec.Thumb != "" {
				_ = os.Remove(filepath.Join(imagesDir, rec.Thumb))
			}
			// Remove the record
			globalMasterList = append(globalMasterList[:i], globalMasterList[i+1:]...)
			break
		}
	}

	// Save the updated master list
	saveMasterListInternal()

	// Signal to the frontend that this specific artist was deleted
	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"artist-deleted": {"id": "%s"}}`, idStr))

	// Return 200 OK with empty body. hx-swap="outerHTML" will remove the element.
	w.WriteHeader(http.StatusOK)
}

func editArtistHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/artists/edit/")
	id, _ := strconv.Atoi(idStr)

	var artist ArtistRecord
	found := false
	for _, rec := range globalMasterList {
		if rec.ID == id {
			artist = rec
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Artist not found", 404)
		return
	}

	data := EditFormData{
		ArtistRecord: artist,
	}

	err := templates.ExecuteTemplate(w, "edit_form_content", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

func updateArtistHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/artists/update/")
	id, _ := strconv.Atoi(idStr)

	name := strings.TrimSpace(r.FormValue("name"))
	desc := strings.TrimSpace(r.FormValue("desc"))
	imgURL := strings.TrimSpace(r.FormValue("img_url"))

	for i, rec := range globalMasterList {
		if rec.ID == id {
			// Validation
			var nameMsg, descMsg string
			if name == "" {
				nameMsg = "Name is required."
			}
			if desc == "" {
				descMsg = "Description is required."
			}

			// Check for duplicate in master list (excluding self) if name is not empty
			if nameMsg == "" {
				for _, other := range globalMasterList {
					if other.ID != id && strings.EqualFold(strings.TrimSpace(other.Name), name) {
						nameMsg = "This name is already in the master list!"
						break
					}
				}
			}

			if nameMsg != "" || descMsg != "" {
				w.Header().Set("HX-Retarget", "#edit-form-target")
				w.Header().Set("HX-Reswap", "innerHTML")
				data := EditFormData{
					ArtistRecord: ArtistRecord{
						ID:          id,
						Name:        name,
						Description: desc,
						ImgURL:      imgURL,
						Thumb:       rec.Thumb,
					},
					NameMsg: nameMsg,
					DescMsg: descMsg,
				}
				err := templates.ExecuteTemplate(w, "edit_form_content", data)
				if err != nil {
					http.Error(w, "Template error: "+err.Error(), 500)
				}
				return
			}

			// If URL changed and is not empty, fetch new image
			if imgURL != "" && imgURL != globalMasterList[i].ImgURL {
				oldThumb := globalMasterList[i].Thumb
				newThumb := fmt.Sprintf("%d-%d.jpg", id, time.Now().Unix())
				if err := fetchAndCreateThumbnail(imgURL, newThumb); err != nil {
					log.Printf("thumbnail error for %s: %v", imgURL, err)
					w.Header().Set("HX-Retarget", "#edit-form-target")
					w.Header().Set("HX-Reswap", "innerHTML")
					data := EditFormData{
						ArtistRecord: ArtistRecord{
							ID:          id,
							Name:        name,
							Description: desc,
							ImgURL:      imgURL,
							Thumb:       rec.Thumb,
						},
						ImgMsg: "Warning: could not create thumbnail from image URL.",
					}
					_ = templates.ExecuteTemplate(w, "edit_form_content", data)
					return
				}
				// Success
				globalMasterList[i].ImgURL = imgURL
				globalMasterList[i].Thumb = newThumb
				// Cleanup old thumb from disk
				if oldThumb != "" && oldThumb != newThumb {
					_ = os.Remove(filepath.Join(imagesDir, oldThumb))
				}
			}

			globalMasterList[i].Name = name
			globalMasterList[i].Description = desc

			saveMasterListInternal()

			// Reset the edit form area to its default state via OOB swap
			fmt.Fprint(w, `<div id="edit-form-target" hx-swap-oob="true"><p>Click "edit" on a card above to load its data here.</p></div>`)

			// Return just the updated grid item fragment
			err := templates.ExecuteTemplate(w, "grid_item", globalMasterList[i])
			if err != nil {
				http.Error(w, "Template error: "+err.Error(), 500)
			}
			return
		}
	}
}

// Helper to avoid code duplication
func saveMasterListInternal() {
	var builder strings.Builder
	for _, rec := range globalMasterList {
		builder.WriteString(fmt.Sprintf("id:%d\nn:%s\nd:%s\ni:%s\nt:%s\n\n", rec.ID, rec.Name, rec.Description, rec.ImgURL, rec.Thumb))
	}
	_ = os.WriteFile(filepath.Join(dataDir, "artists_master.txt"), []byte(builder.String()), 0644)
}

// --- Main ---

func main() {

	templates = template.Must(template.ParseFiles(
		"templates/index.tmpl",
		"templates/artist_form.tmpl",
		"templates/artist_list.tmpl",
		"templates/submit_response.tmpl",
		"templates/confirm_dialog.tmpl",
		"templates/gallery.tmpl",
	))

	// Read ENV vars FIRST
	if testMode := os.Getenv("TEST_MODE"); testMode != "" {
		if testMode == "true" || testMode == "1" {
			dataDir = "test_data"
			imagesDir = "test_images"
		}
	}

	// Log what we're using
	log.Printf("Using data dir: %s, images dir: %s", dataDir, imagesDir)

	// return

	// Load lists using NEW paths
	var err error
	globalMasterList, err = ReadMasterList(filepath.Join(dataDir, "artists_master.txt"))
	if err != nil {
		log.Fatal("Error reading master list:", err)
	}
	globalToAddList, err = ReadToAddList(filepath.Join(dataDir, "artists_to_add.txt"))
	if err != nil {
		log.Fatal("Error reading to-add list:", err)
	}

	// // Load lists from files
	// var err error
	// globalMasterList, err = ReadMasterList("data/artists_master.txt")
	// if err != nil {
	// 	log.Fatal("Error reading master list:", err)
	// }
	// globalToAddList, err = ReadToAddList("data/artists_to_add.txt")
	// if err != nil {
	// 	log.Fatal("Error reading to-add list:", err)
	// }

	http.HandleFunc("/", addArtistPage)
	http.HandleFunc("/gallery", galleryPage)
	http.HandleFunc("/populate-form", populateFormHandler)
	http.HandleFunc("/check-name", checkNameHandler)
	http.HandleFunc("/delete-todo-form", deleteTodoFormHandler) // we may still call this with htmxx but from are you sure dialog
	http.HandleFunc("/confirm-delete-todo-form", confirmDeleteTodoFormHandler)
	http.HandleFunc("/cancel-add-form", cancelAddFormHandler)
	http.HandleFunc("/submit-artist-add-form", submitArtistAddFormHandler)
	http.HandleFunc("/confirm-delete-todo", confirmDeleteTodoHandler)
	http.HandleFunc("/delete-todo-item", deleteTodoItemHandler)
	http.HandleFunc("/add-to-todo-list", addToTodoListHandler)
	http.HandleFunc("/artists/delete/", deleteArtistHandler)
	http.HandleFunc("/artists/edit/", editArtistHandler)
	http.HandleFunc("/artists/update/", updateArtistHandler)

	// main.go (add before http.ListenAndServe)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	// use next with imagesDir for test or prod images, if gallery not righ, do hard refresh Ctrl-Shift-R
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(imagesDir))))

	log.Println("Listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func thumbnailExists(filename string) bool {
	// thumbnailPath := filepath.Join("images", filename)

	thumbnailPath := filepath.Join(imagesDir, filename)
	if _, err := os.Stat(thumbnailPath); err == nil {
		return true
	}
	return false
}

func fetchAndCreateThumbnail(imageURL, filename string) error {
	// Ensure images dir exists
	// imagesDir := "images" / LOOK, using global now ...
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return err
	}

	resp, err := http.Get(imageURL)
	if err != nil {
		return fmt.Errorf("error fetching image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("error fetching image: status %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return fmt.Errorf("error reading image data: %v", err)
	}

	img, err := imaging.Decode(&buf)
	if err != nil {
		return fmt.Errorf("error decoding image: %v", err)
	}

	resized := imaging.Resize(img, 200, 0, imaging.Lanczos)
	outPath := filepath.Join(imagesDir, filename)

	if err := imaging.Save(resized, outPath); err != nil {
		return fmt.Errorf("error saving image: %v", err)
	}

	return nil
}
