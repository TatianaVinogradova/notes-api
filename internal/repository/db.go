package repository

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5"
)

func Connect() (*pgx.Conn, error) {
    connStr := "postgres://dimavinogradov@localhost:5432/notes"
    conn, err := pgx.Connect(context.Background(), connStr)
    if err != nil {
        return nil, fmt.Errorf("unable to connect to database: %w", err)
    }
    return conn, nil
}