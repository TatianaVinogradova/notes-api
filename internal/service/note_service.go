package service

import (
    "context"

    "notes-api/internal/models"
    "notes-api/internal/repository"
)

type NoteService struct {
    repo *repository.NoteRepository
}

func NewNoteService(repo *repository.NoteRepository) *NoteService {
    return &NoteService{repo: repo}
}

func (s *NoteService) GetAll(ctx context.Context) ([]models.Note, error) {
    return s.repo.GetAll(ctx)
}

func (s *NoteService) GetByID(ctx context.Context, id int) (*models.Note, error) {
    return s.repo.GetByID(ctx, id)
}

func (s *NoteService) Create(ctx context.Context, title, content string) (*models.Note, error) {
    return s.repo.Create(ctx, title, content)
}

func (s *NoteService) Update(ctx context.Context, id int, title, content string) (*models.Note, error) {
    return s.repo.Update(ctx, id, title, content)
}

func (s *NoteService) Delete(ctx context.Context, id int) error {
    return s.repo.Delete(ctx, id)
}