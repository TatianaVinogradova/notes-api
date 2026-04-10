package repository

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"notes-api/internal/models"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const recordSize = 512
const deletedFlag = byte(1)
const activeFlag = byte(0)

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

func encodeRecord(n models.Note) ([]byte, error) {
	data, err := json.Marshal(n)
	if err != nil {
		return nil, err
	}
	if len(data) > recordSize-1 {
		return nil, errors.New("Nota demaciado larga")
	}

	record := make([]byte, recordSize)
	record[0] = activeFlag
	copy(record[1:], data)
	return record, nil
}

func decodeRecord(record []byte) (*models.Note, bool, error) {
	deleted := record[0] == deletedFlag
	data := bytes.TrimRight(record[1:], "\x00 ")
	var n models.Note
	if err := json.Unmarshal(data, &n); err != nil {
		return nil, deleted, err
	}
	return &n, deleted, nil
}

func offsetForID(id int) int64 {
	return int64((id - 1) * recordSize)
}

func (r *FileRepository) nextID() (int, error) {
	info, err := os.Stat(r.datPath)
	if err != nil {
		return 1, nil
	}
	return int(info.Size()/recordSize) + 1, nil
}

func (r *FileRepository) GetAll(ctx context.Context) ([]models.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := os.ReadFile(r.datPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Note{}, nil
		}
		return nil, err
	}

	var notes []models.Note
	for i := 0; i+recordSize <= len(data); i += recordSize {
		record := data[i : i+recordSize]
		n, deleted, err := decodeRecord(record)
		if err != nil || deleted {
			continue
		}
		notes = append(notes, *n)
	}
	return notes, nil
}

func (r *FileRepository) GetByID(ctx context.Context, id int) (*models.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	file, err := os.OpenFile(r.datPath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = file.Seek(offsetForID(id), 0)
	if err != nil {
		return nil, err
	}

	record := make([]byte, recordSize)
	_, err = file.Read(record)
	if err != nil {
		return nil, errors.New("nota no encontrada")
	}

	n, deleted, err := decodeRecord(record)
	if err != nil || deleted {
		return nil, errors.New("nota no encontrada")
	}
	return n, nil
}

func (r *FileRepository) Create(ctx context.Context, title, content string) (*models.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id, err := r.nextID()
	if err != nil {
		return nil, err
	}

	n := models.Note{
		ID:        id,
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	record, err := encodeRecord(n)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(r.datPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if _, err := file.Write(record); err != nil {
		return nil, err
	}
	if err := r.addToIndex(n); err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *FileRepository) Update(ctx context.Context, id int, title, content string) (*models.Note, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.OpenFile(r.datPath, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = file.Seek(offsetForID(id), 0)
	if err != nil {
		return nil, err
	}
	record := make([]byte, recordSize)
	if _, err := file.Read(record); err != nil {
		return nil, errors.New("Nota no encontrada")
	}
	existing, deleted, err := decodeRecord(record)
	if err != nil || deleted {
		return nil, errors.New("Nota no encontrada")
	}

	existing.Title = title
	existing.Content = content
	existing.UpdatedAt = time.Now()

	newRecord, err := encodeRecord(*existing)
	if err != nil {
		return nil, err
	}
	_, err = file.Seek(offsetForID(id), 0)
	if err != nil {
		return nil, err
	}
	if _, err := file.Write(newRecord); err != nil {
		return nil, err
	}
	return existing, r.rebuildIndex()
}

func (r *FileRepository) Delete(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.OpenFile(r.datPath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Seek(offsetForID(id), 0)
	if err != nil {
		return err
	}
	_, err = file.Write([]byte{deletedFlag})
	return err
}

func (r *FileRepository) Search(ctx context.Context, query string) ([]models.Note, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	idxFile, err := os.OpenFile(r.idxPath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer idxFile.Close()

	query = strings.ToLower(query)
	matchedIDs := make(map[int]bool)

	scanner := bufio.NewScanner(idxFile)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		title := parts[0]
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		if strings.Contains(title, query) {
			matchedIDs[id] = true
		}
	}

	file, err := os.OpenFile(r.datPath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []models.Note
	record := make([]byte, recordSize)

	for id := range matchedIDs {
		_, err = file.Seek(offsetForID(id), 0)
		if err != nil {
			continue
		}
		if _, err := file.Read(record); err != nil {
			continue
		}
		n, deleted, err := decodeRecord(record)
		if err != nil || deleted {
			continue
		}
		result = append(result, *n)
	}
	return result, nil
}

func (r *FileRepository) addToIndex(n models.Note) error {
	file, err := os.OpenFile(r.idxPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	writer.WriteString(fmt.Sprintf("%s:%d\n", strings.ToLower(n.Title), n.ID))
	return writer.Flush()
}

func (r *FileRepository) rebuildIndex() error {
	data, err := os.ReadFile(r.datPath)
	if err != nil {
		return err
	}

	idxFile, err := os.OpenFile(r.idxPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer idxFile.Close()

	writer := bufio.NewWriter(idxFile)
	for i := 0; i+recordSize <= len(data); i += recordSize {
		record := data[i : i+recordSize]
		n, deleted, err := decodeRecord(record)
		if err != nil || deleted {
			continue
		}
		writer.WriteString(fmt.Sprintf("%s:%d\n", strings.ToLower(n.Title), n.ID))
	}
	return writer.Flush()
}
