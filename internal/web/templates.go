package web

import (
	"html/template"
	"io"
	"path/filepath"

	"github.com/sstent/garminsync-go/internal/database"
)

type WebHandler struct {
	templates *template.Template
	db        *database.SQLiteDB
}

func NewWebHandler(db *database.SQLiteDB) *WebHandler {
	return &WebHandler{
		db: db,
	}
}

func (h *WebHandler) LoadTemplates(templatesDir string) error {
	tmpl := template.New("base")
	tmpl = tmpl.Funcs(template.FuncMap{})

	// Load layouts
	layouts, err := filepath.Glob(filepath.Join(templatesDir, "layouts/*.html"))
	if err != nil {
		return err
	}

	// Load pages
	pages, err := filepath.Glob(filepath.Join(templatesDir, "pages/*.html"))
	if err != nil {
		return err
	}

	// Combine all templates
	files := append(layouts, pages...)

	h.templates, err = tmpl.ParseFiles(files...)
	return err
}

func (h *WebHandler) renderTemplate(w io.Writer, name string, data interface{}) error {
	return h.templates.ExecuteTemplate(w, name, data)
}
