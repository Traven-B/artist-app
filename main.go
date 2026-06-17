package main

import (
	// "bufio"
	// "bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math" // Added for vector similarity calculations
	"net/http"
	"os"
	"path/filepath"
	"sort" // Added for ranking closest matches
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
	Features    string    // Feature data string
	Vector      []float64 // Server-side vector embeddings array (never leaves the server)
}

type FormData struct {
	Name         string
	OriginalName string
	Desc         string
	ImgURL       string
	Features     string // Form input data

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
	ToAdd      []string
	FormData   FormData
	IsTestMode bool
}

var templates *template.Template

// File-backed data
var globalMasterList []ArtistRecord
var globalToAddList []string
var globalFeatureVectors map[int][]float64 // Map artist ID to its feature vector

var dataDir = "data"     // Default prod
var imagesDir = "images" // Default prod

// --- Vector Math Engine (Server-Side Only) ---

// cosineSimilarity calculates how close two vectors are (returns a value between -1.0 and 1.0)
func cosineSimilarity(v1, v2 []float64) float64 {
	if len(v1) != len(v2) || len(v1) == 0 {
		return 0.0
	}
	var dotProduct, normA, normB float64
	for i := 0; i < len(v1); i++ {
		dotProduct += v1[i] * v2[i]
		normA += v1[i] * v1[i]
		normB += v2[i] * v2[i]
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

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
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "id:") {
				rec.ID, _ = strconv.Atoi(strings.TrimSpace(line[3:]))
			} else if strings.HasPrefix(line, "n:") {
				rec.Name = strings.TrimSpace(line[2:])
			} else if strings.HasPrefix(line, "d:") {
				rec.Description = strings.TrimSpace(line[2:])
			} else if strings.HasPrefix(line, "t:") {
				rec.Thumb = strings.TrimSpace(line[2:])
			} else if strings.HasPrefix(line, "f:") {
				rec.Features = strings.TrimSpace(line[2:])
			}
		}
		records = append(records, rec)
	}
	return records, nil
}

func ReadFeatureVectors(filename string) (map[int][]float64, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	featureVectors := make(map[int][]float64)
	blocks := strings.Split(string(data), "\n\n")
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		var id int
		var vector []float64
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "id:") {
				id, _ = strconv.Atoi(strings.TrimSpace(line[3:]))
			} else if strings.HasPrefix(line, "ef:") {
				vecStr := strings.TrimSpace(line[3:])
				if vecStr != "" {
					parts := strings.Split(vecStr, ",")
					for _, p := range parts {
						val, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
						if err == nil {
							vector = append(vector, val)
						} else {
							log.Printf("Error parsing float from feature vector string '%s': %v", p, err)
						}
					}
				}
			}
		}
		if id != 0 && len(vector) > 0 {
			featureVectors[id] = vector
		}
	}
	return featureVectors, nil
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

// --- Helper for fetching remote images ---
func downloadImage(urlStr string) (io.ReadCloser, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(urlStr)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}
	return resp.Body, nil
}

func isDuplicate(name string) bool {
	for _, rec := range globalMasterList {
		if strings.EqualFold(strings.TrimSpace(rec.Name), name) {
			return true
		}
	}
	for _, n := range globalToAddList {
		if strings.EqualFold(strings.TrimSpace(n), name) {
			return true
		}
	}
	return false
}

// --- Handlers ---

func addArtistPage(w http.ResponseWriter, r *http.Request) {
	data := AddArtistPageData{
		ToAdd:      globalToAddList,
		FormData:   FormData{},
		IsTestMode: dataDir == "test_data",
	}

	err := templates.ExecuteTemplate(w, "index", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

// Real-Time Vector Similarity Handler
func artistSimilarIDsHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/artists/similar-ids/")
	targetID, _ := strconv.Atoi(idStr)

	// 1. Locate the target artist from memory
	var targetArtist ArtistRecord
	found := false
	for _, a := range globalMasterList {
		if a.ID == targetID {
			targetArtist = a
			found = true
			break
		}
	}

	// Retrieve target artist's vector from globalFeatureVectors
	targetVector, hasVector := globalFeatureVectors[targetID]

	// Dynamic Fallback: If artist isn't found or doesn't have vectors yet, default to canned indexes safely
	if !found || !hasVector || len(targetVector) == 0 {
		cannedIDs := []int{targetID, targetID + 1, targetID + 2, targetID + 3}
		responseMap := map[string][]int{"ids": cannedIDs}
		jsonIDs, _ := json.Marshal(responseMap)
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"checkSimilarArtists": %s}`, string(jsonIDs)))
		w.WriteHeader(http.StatusOK)
		return
	}

	// Struct to keep track of dynamic comparison rankings
	type match struct {
		id    int
		score float64
	}
	var matches []match

	// 2. Compute similarity scores across all other 160+ artists
	for _, a := range globalMasterList {
		if a.ID == targetID {
			continue // Skip self
		}
		aVector, aHasVector := globalFeatureVectors[a.ID]
		if !aHasVector || len(aVector) == 0 {
			continue // Skip entries lacking embeddings
		}
		score := cosineSimilarity(targetVector, aVector)
		matches = append(matches, match{id: a.ID, score: score})
	}

	// 3. Sort closest neighbors (highest decimal scores first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	// 4. Extract top 3 matches
	limit := 3
	if len(matches) < limit {
		limit = len(matches)
	}

	// Include clicked artist ID first, followed by computed neighbors
	computedIDs := []int{targetID}
	for i := 0; i < limit; i++ {
		computedIDs = append(computedIDs, matches[i].id)
	}

	// 5. Serialize lightweight IDs string to fire custom UI event trigger
	responseMap := map[string][]int{"ids": computedIDs}
	jsonIDs, err := json.Marshal(responseMap)
	if err != nil {
		http.Error(w, "JSON error", 500)
		return
	}

	w.Header().Set("HX-Trigger", fmt.Sprintf(`{"checkSimilarArtists": %s}`, string(jsonIDs)))
	w.WriteHeader(http.StatusOK)
}

func galleryPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Artists    []ArtistRecord
		IsTestMode bool
	}{
		Artists:    globalMasterList,
		IsTestMode: dataDir == "test_data",
	}

	err := templates.ExecuteTemplate(w, "gallery_page", data)
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), 500)
	}
}

func populateFormHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	data := AddArtistPageData{
		FormData: FormData{
			Name:         name,
			OriginalName: name,
			NameMsg:      "",
			Features:     "",
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
	name := strings.TrimSpace(r.FormValue("name"))
	originalName := r.FormValue("original_name")
	msg := ""
	// Search master list for duplicate (case-insensitive)
	for _, rec := range globalMasterList {
		if strings.EqualFold(strings.TrimSpace(rec.Name), name) {
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
	addedCount := 0
	skippedCount := 0

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}

		if isDuplicate(name) {
			skippedCount++
		} else {
			globalToAddList = append(globalToAddList, name)
			addedCount++
		}
	}

	if addedCount > 0 {
		_ = os.WriteFile(filepath.Join(dataDir, "artists_to_add.txt"), []byte(strings.Join(globalToAddList, "\n")+"\n"), 0644)
	}

	data := struct {
		ToAdd   []string
		Added   int
		Skipped int
	}{
		ToAdd:   globalToAddList,
		Added:   addedCount,
		Skipped: skippedCount,
	}
	_ = templates.ExecuteTemplate(w, "todo_add_response", data)
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
	features := strings.TrimSpace(r.FormValue("features")) // Get features from form

	var nameMsg, descMsg, imgMsg string

	// Validation
	if name == "" {
		nameMsg = "Name is required."
	}
	if desc == "" {
		descMsg = "Description is required."
	}

	// Handle File Upload or URL
	file, _, fileErr := r.FormFile("image")
	if fileErr != nil && imgURL == "" {
		imgMsg = "Image file or URL is required."
	}

	// Check for duplicate in master list
	for _, rec := range globalMasterList {
		if strings.EqualFold(strings.TrimSpace(rec.Name), name) {
			nameMsg = "This name is already in the master list!"
			break
		}
	}

	// If initial validation failed, return form
	if nameMsg != "" || descMsg != "" || imgMsg != "" {
		data := AddArtistPageData{
			ToAdd: globalToAddList,
			FormData: FormData{
				Name:         name,
				OriginalName: originalName,
				Desc:         desc,
				ImgURL:       imgURL,
				Features:     features,
				NameMsg:      nameMsg,
				DescMsg:      descMsg,
				ImgMsg:       imgMsg,
			},
		}
		_ = templates.ExecuteTemplate(w, "submit_response", data)
		return
	}

	// Generate next ID and thumb filename
	maxID := 0
	for _, rec := range globalMasterList {
		if rec.ID > maxID {
			maxID = rec.ID
		}
	}
	newID := maxID + 1
	thumbFile := fmt.Sprintf("%d-%d.jpg", newID, time.Now().Unix())

	// Create thumbnail from source
	var reader io.ReadCloser
	if fileErr == nil {
		reader = file
		defer file.Close()
	} else {
		var err error
		reader, err = downloadImage(imgURL)
		if err != nil {
			log.Printf("download error: %v", err)
			imgMsg = "Failed to download image from URL."
		} else {
			defer reader.Close()
		}
	}

	if imgMsg == "" {
		if err := createThumbnailFromReader(reader, thumbFile); err != nil {
			log.Printf("thumbnail error: %v", err)
			imgMsg = "Failed to process image source."
		}
	}

	if imgMsg != "" {
		data := AddArtistPageData{
			ToAdd: globalToAddList,
			FormData: FormData{
				Name:         name,
				OriginalName: originalName,
				Desc:         desc,
				ImgURL:       imgURL,
				Features:     features,
				ImgMsg:       imgMsg,
			},
		}
		_ = templates.ExecuteTemplate(w, "submit_response", data)
		return
	}

	// Add artist to master list
	newRec := ArtistRecord{
		ID:          newID,
		Name:        name,
		Description: desc,
		Thumb:       thumbFile,
		Features:    features,
		Vector:      nil, // Vector is no longer stored in ArtistRecord, but will be generated/read from artists_feature_vectors.txt
	}
	globalMasterList = append(globalMasterList, newRec)

	saveMasterListInternal()

	// Remove name from to-do list if present
	if originalName != "" {
		newList := make([]string, 0, len(globalToAddList))
		for _, n := range globalToAddList {
			if !strings.EqualFold(n, originalName) {
				newList = append(newList, n)
			}
		}
		globalToAddList = newList
		_ = os.WriteFile(filepath.Join(dataDir, "artists_to_add.txt"), []byte(strings.Join(globalToAddList, "\n")+"\n"), 0644)
	}

	// Return updated form (cleared) + updated list via OOB swaps
	data := AddArtistPageData{
		ToAdd:    globalToAddList,
		FormData: FormData{},
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
	features := strings.TrimSpace(r.FormValue("features"))

	for i, rec := range globalMasterList {
		if rec.ID == id {
			var nameMsg, descMsg, imgMsg string
			if name == "" {
				nameMsg = "Name is required."
			}
			if desc == "" {
				descMsg = "Description is required."
			}

			if nameMsg == "" {
				for _, other := range globalMasterList {
					if other.ID != id && strings.EqualFold(strings.TrimSpace(other.Name), name) {
						nameMsg = "This name is already in the master list!"
						break
					}
				}
			}

			// Handle Optional File Upload or URL
			imgURL := strings.TrimSpace(r.FormValue("img_url"))
			file, _, fileErr := r.FormFile("image")
			var newThumb string

			if fileErr == nil || imgURL != "" {
				var reader io.ReadCloser
				if fileErr == nil {
					reader = file
					defer file.Close()
				} else {
					var err error
					reader, err = downloadImage(imgURL)
					if err != nil {
						log.Printf("download error: %v", err)
						imgMsg = "Failed to download image."
					} else {
						defer reader.Close()
					}
				}

				if imgMsg == "" {
					newThumb = fmt.Sprintf("%d-%d.jpg", id, time.Now().Unix())
					if err := createThumbnailFromReader(reader, newThumb); err != nil {
						log.Printf("thumbnail error: %v", err)
						imgMsg = "Failed to process image source."
						newThumb = ""
					}
				}
			} else if fileErr != http.ErrMissingFile {
				imgMsg = "Error uploading image."
			}

			if nameMsg != "" || descMsg != "" || imgMsg != "" {
				w.Header().Set("HX-Retarget", "#edit-form-target")
				w.Header().Set("HX-Reswap", "innerHTML")
				data := EditFormData{
					ArtistRecord: ArtistRecord{
						ID:          id,
						Name:        name,
						Description: desc,
						Thumb:       rec.Thumb,
						Features:    features,
					},
					NameMsg: nameMsg,
					DescMsg: descMsg,
					ImgMsg:  imgMsg,
				}
				_ = templates.ExecuteTemplate(w, "edit_form_content", data)
				return
			}

			// Success - Update record
			if newThumb != "" {
				oldThumb := globalMasterList[i].Thumb
				globalMasterList[i].Thumb = newThumb
				if oldThumb != "" && oldThumb != newThumb {
					_ = os.Remove(filepath.Join(imagesDir, oldThumb))
				}
			}

			globalMasterList[i].Name = name
			globalMasterList[i].Description = desc
			globalMasterList[i].Features = features
			globalMasterList[i].Vector = nil // Clear any potential old vector data

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
		// Feature vectors are now stored in a separate file, so we don't save 'ef:' here.
		builder.WriteString(fmt.Sprintf("id:%d\nn:%s\nd:%s\nt:%s\nf:%s\n\n", rec.ID, rec.Name, rec.Description, rec.Thumb, rec.Features))
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

	var err error
	globalMasterList, err = ReadMasterList(filepath.Join(dataDir, "artists_master.txt"))
	if err != nil {
		log.Fatal("Error reading master list:", err)
	}
	globalToAddList, err = ReadToAddList(filepath.Join(dataDir, "artists_to_add.txt"))
	if err != nil {
		log.Fatal("Error reading to-add list:", err)
	}

	globalFeatureVectors, err = ReadFeatureVectors(filepath.Join(dataDir, "artists_feature_vectors.txt"))
	if err != nil {
		log.Fatal("Error reading feature vectors:", err)
	}

	http.HandleFunc("/", addArtistPage)
	http.HandleFunc("/gallery", galleryPage)
	http.HandleFunc("/artists/similar-ids/", artistSimilarIDsHandler)
	http.HandleFunc("/populate-form", populateFormHandler)
	http.HandleFunc("/check-name", checkNameHandler)
	http.HandleFunc("/delete-todo-form", deleteTodoFormHandler)
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

func createThumbnailFromReader(reader io.Reader, filename string) error {
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return err
	}

	img, err := imaging.Decode(reader)
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
