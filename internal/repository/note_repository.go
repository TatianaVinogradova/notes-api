package repository

import (
	"context"

	"notes-api/internal/models"

	"github.com/jackc/pgx/v5"
)

type NoteRepository interface {
	GetAll(ctx context.Context) ([]models.Note, error)
	GetByID(ctx context.Context, id int) (*models.Note, error)
	Create(ctx context.Context, title, content string) (*models.Note, error)
	Update(ctx context.Context, id int, title, content string) (*models.Note, error)
	Delete(ctx context.Context, id int) error
	Search(ctx context.Context, query string) ([]models.Note, error)
}

type PostgresRepository struct {
	db *pgx.Conn
}

func NewPostgresRepository(db *pgx.Conn) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetAll(ctx context.Context) ([]models.Note, error) {
	rows, err := r.db.Query(ctx, "SELECT id, title, content, created_at, updated_at FROM notes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int) (*models.Note, error) {
	var n models.Note
	err := r.db.QueryRow(ctx,
		"SELECT id, title, content, created_at, updated_at FROM notes WHERE id = $1", id,
	).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *PostgresRepository) Create(ctx context.Context, title, content string) (*models.Note, error) {
	var n models.Note
	err := r.db.QueryRow(ctx,
		"INSERT INTO notes (title, content) VALUES ($1, $2) RETURNING id, title, content, created_at, updated_at",
		title, content,
	).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id int, title, content string) (*models.Note, error) {
	var n models.Note
	err := r.db.QueryRow(ctx,
		"UPDATE notes SET title=$1, content=$2, updated_at=NOW() WHERE id=$3 RETURNING id, title, content, created_at, updated_at",
		title, content, id,
	).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "DELETE FROM notes WHERE id = $1", id)
	return err
}

func (r *PostgresRepository) Search(ctx context.Context, query string) ([]models.Note, error) {
	searchPattern := "%" + query + "%"
	rows, err := r.db.Query(ctx,
		"SELECT id, title, content, created_at, updated_at FROM notes WHERE title ILIKE $1",
		searchPattern,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, nil
}
