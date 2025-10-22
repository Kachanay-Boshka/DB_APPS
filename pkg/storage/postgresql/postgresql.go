package postgresql

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Storage struct {
	db *pgxpool.Pool
}

func New(connString string) (*Storage, error) {
	db, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return nil, err
	}
	return &Storage{db: db}, nil
}

func (s *Storage) Close() {
	s.db.Close()
}
