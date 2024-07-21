package web

import (
	"html/template"
	"log"
	"net/http"
)

type Handler struct {
	tmpl *template.Template
}

func NewHandler() *Handler {
	return &Handler{
		tmpl: template.Must(template.ParseGlob("templates/*.html")),
	}
}

func (h *Handler) Render(w http.ResponseWriter, r *http.Request) {
	if err := h.tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		log.Printf("Error executing template: %v", err)
	}
}
