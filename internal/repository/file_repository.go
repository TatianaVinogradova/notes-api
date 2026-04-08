package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"notes-api/internal/models"
	"os"
	"strings"
	"sync"
	"time"
)

type FileRepository struct {
	datPath string
	idxPath string
	mu      sync.RWMutex
}

func NewFileRepository(datPath, idxPath string) *FileRepository {
	return &FileRepository{
		datPath: datPath,
		idxPath: idxPath,
	}
}

func (r *FileRepository) readAll() ([]models.Note, error) {
	file, err := os.OpenFile(r.datPath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var notes []models.Note
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var n models.Note
		if err := json.Unmarshal([]byte(line), &n); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, nil
}

func (r *FileRepository) writeAll(notes []models.Note) error {
	file, err := os.OpenFile(r.datPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, n := range notes {
		line, err := json.Marshal(n)
		if err != nil {
			return err
		}
		writer.WriteString(string(line) + "\n")
	}
	return writer.Flush()
}

func (r *FileRepository) rebuildIndex(notes []models.Note) error {
	file, err := os.OpenFile(r.idxPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, n := range notes {
		line := strings.ToLower(n.Title) + ":" + string(rune(n.ID+'0')) + "\n"
		writer.WriteString(line)
	}
	return writer.Flush()
}

func (r *FileRepository) nextID(notes []models.Note) int {
	maxID := 0
	for _, n := range notes {
		if n.ID > maxID {
			maxID = n.ID
		}
	}
	return maxID + 1
}

func (r *FileRepository) GetAll(ctx context.Context) ([]models.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.readAll()
}

func (r *FileRepository) GetByID(ctx context.Context, id int) (*models.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	notes, err := r.readAll()
	if err != nil {
		return nil, err
	}
	for _, n := range notes {
		if n.ID == id {
			return &n, nil
		}
	}
	return nil, errors.New("nota no encontrada")
}

func (r *FileRepository) Create(ctx context.Context, title, content string) (*models.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	notes, err := r.readAll()
	if err != nil {
		return nil, err
	}

	n := models.Note{
		ID:        r.nextID(notes),
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	notes = append(notes, n)

	if err := r.writeAll(notes); err != nil {
		return nil, err
	}
	if err := r.rebuildIndex(notes); err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *FileRepository) Update(ctx context.Context, id int, title, content string) (*models.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	notes, err := r.readAll()
	if err != nil {
		return nil, err
	}

	var updated *models.Note
	for i, n := range notes {
		if n.ID == id {
			notes[i].Title = title
			notes[i].Content = content
			notes[i].UpdatedAt = time.Now()
			updated = &notes[i]
			break
		}
	}

	if updated == nil {
		return nil, errors.New("nota no encontrada")
	}
	if err := r.writeAll(notes); err != nil {
		return nil, err
	}
	if err := r.rebuildIndex(notes); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *FileRepository) Delete(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	notes, err := r.readAll()
	if err != nil {
		return err
	}

	filtered := make([]models.Note, 0)
	for _, n := range notes {
		if n.ID != id {
			filtered = append(filtered, n)
		}
	}
	if err := r.writeAll(filtered); err != nil {
		return err
	}
	return r.rebuildIndex(filtered)
}

func (r *FileRepository) Search(ctx context.Context, query string) ([]models.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	notes, err := r.readAll()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var result []models.Note
	for _, n := range notes {
		if strings.Contains(strings.ToLower(n.Title), query) {
			result = append(result, n)
		}
	}
	return result, nil
}
