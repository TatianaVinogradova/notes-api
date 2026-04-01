package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"notes-api/internal/models"
	"notes-api/internal/service"
)

type NoteHandler struct {
	service *service.NoteService
}

func NewNoteHandler(service *service.NoteService) *NoteHandler {
	return &NoteHandler{service: service}
}

// extrae el ID de la URL, por ejemplo /notes/42 -> 42
func getIDFromPath(path string) (int, error) {
	parts := strings.Split(path, "/")
	return strconv.Atoi(parts[len(parts)-1])
}

func (h *NoteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path == "/notes" || r.URL.Path == "/notes/" {
		switch r.Method {
		case http.MethodGet:
			h.getAll(w, r)
		case http.MethodPost:
			h.create(w, r)
		default:
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		}
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 3 {
		http.NotFound(w, r)
		return
	}

	id, err := getIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getByID(w, r, id)
	case http.MethodPut:
		h.update(w, r, id)
	case http.MethodDelete:
		h.delete(w, r, id)
	default:
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
	}
}

func (h *NoteHandler) getAll(w http.ResponseWriter, r *http.Request) {
	notes, err := h.service.GetAll(r.Context())
	if err != nil {
		http.Error(w, "error obteniendo notas", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(notes)
}

func (h *NoteHandler) getByID(w http.ResponseWriter, r *http.Request, id int) {
	note, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "nota no encontrada", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(note)
}

func (h *NoteHandler) create(w http.ResponseWriter, r *http.Request) {
	var body models.Note
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request inválido", http.StatusBadRequest)
		return
	}

	note, err := h.service.Create(r.Context(), body.Title, body.Content)
	if err != nil {
		http.Error(w, "error creando nota", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

func (h *NoteHandler) update(w http.ResponseWriter, r *http.Request, id int) {
	var body models.Note
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request inválido", http.StatusBadRequest)
		return
	}

	note, err := h.service.Update(r.Context(), id, body.Title, body.Content)
	if err != nil {
		http.Error(w, "error actualizando nota", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(note)
}

func (h *NoteHandler) delete(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.service.Delete(r.Context(), id); err != nil {
		http.Error(w, "error eliminando nota", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
