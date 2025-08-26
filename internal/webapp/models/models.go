package models

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	db *pgxpool.Pool
}

type SnippetTuple struct {
	id int
}

type Store interface {
	ListSnippetTuples() ([]SnippetTuple, error)
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &PostgresStore{db: pool}, nil
}

func (ps *PostgresStore) ListSnippetTuples() ([]SnippetTuple, error) {
	return nil, errors.New("Not implemented")
}
