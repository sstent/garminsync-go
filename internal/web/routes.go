package web

import (
	"net/http"
	"html/template"
	"path/filepath"
	"os"

	"github.com/yourusername/garminsync/internal/database"
)

type WebHandler struct {
	db *database.SQLiteDB
	templates map[string]*template.Template
}

func NewWebHandler(db *database.SQLiteDB) *WebHandler {
	return &WebHandler{
		db: db,
		templates: make(map[string]*template.Template),
	}
}

func (h *WebHandler) LoadTemplates(templateDir string) error {
	layouts, err := filepath.Glob(filepath.Join(templateDir, "layouts", "*.html"))
	if err != nil {
		return err
	}

	partials, err := filepath.Glob(filepath.Join(templateDir, "partials", "*.html"))
	if err != nil {
		return err
	}

	pages, err := filepath.Glob(filepath.Join(templateDir, "pages", "*.html"))
	if err != nil {
		return err
	}

	for _, page := range pages {
		name := filepath.Base(page)
		
		files := append([]string{page}, layouts...)
		files = append(files, partials...)
		
		h.templates[name], err = template.ParseFiles(files...)
		if err != nil {
			return err
		}
	}
	
	return nil
}

func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderTemplate(w, "index.html", stats)
}

func (h *WebHandler) ActivityList(w http.ResponseWriter, r *http.Request) {
	activities, err := h.db.GetActivities(50, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.renderTemplate(w, "activity_list.html", activities)
}

func (h *WebHandler) ActivityDetail(w http.ResponseWriter, r *http.Request) {
	// Extract activity ID from URL params
	activityID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid activity ID", http.StatusBadRequest)
		return
	}

	activity, err := h.db.GetActivity(activityID)
	if err != nil {
		http.Error(w, "Activity not found", http.StatusNotFound)
		return
	}

	h.renderTemplate(w, "activity_detail.html", activity)
}

func (h *WebHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	tmpl, ok := h.templates[name]
	if !ok {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
