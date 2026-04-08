package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"notes-api/internal/service"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type WebHandler struct {
	service   *service.NoteService
	indexTmpl *template.Template
	editTmpl  *template.Template
}

func getTemplatesDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "templates")
}

func NewWebHandler(service *service.NoteService) (*WebHandler, error) {
	dir := getTemplatesDir()

	indexTmpl, err := template.ParseFiles(filepath.Join(dir, "index.html"))
	if err != nil {
		return nil, err
	}

	editTmpl, err := template.ParseFiles(filepath.Join(dir, "edit.html"))
	if err != nil {
		return nil, err
	}

	return &WebHandler{
		service:   service,
		indexTmpl: indexTmpl,
		editTmpl:  editTmpl,
	}, nil
}

func getIDFromURL(path string) (int, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		return 0, fmt.Errorf("url inválida")
	}
	return strconv.Atoi(parts[3])
}

func (h *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	var err error
	var notes interface{}

	if query != "" {
		notes, err = h.service.Search(r.Context(), query)
	} else {
		notes, err = h.service.GetAll(r.Context())
	}

	if err != nil {
		http.Error(w, "error obteniendo notas", http.StatusInternalServerError)
		return
	}

	data := struct {
		Notes interface{}
		Query string
	}{
		Notes: notes,
		Query: query,
	}
	if err := h.indexTmpl.Execute(w, data); err != nil {
		http.Error(w, "error renderizado template", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Create(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	content := r.FormValue("content")

	_, err := h.service.Create(r.Context(), title, content)
	if err != nil {
		http.Error(w, "error creando nota", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/web", http.StatusSeeOther)
}

func (h *WebHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromURL(r.URL.Path)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	note, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "nota no encontrada", http.StatusNotFound)
		return
	}
	if err := h.editTmpl.Execute(w, note); err != nil {
		http.Error(w, "error renderizando template", http.StatusInternalServerError)
	}
}

func (h *WebHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromURL(r.URL.Path)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")

	_, err = h.service.Update(r.Context(), id, title, content)
	if err != nil {
		http.Error(w, "error actualizando nota", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/web", http.StatusSeeOther)
}

func (h *WebHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromURL(r.URL.Path)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		http.Error(w, "error eliminando nota", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/web", http.StatusSeeOther)
}
