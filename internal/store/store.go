package store

import "database/sql"

var DB *sql.DB

type Store struct {
	db    *sql.DB
	labDB *sql.DB
	repo  *Repo
}

func New(db *sql.DB) *Store {
	return &Store{
		db: db,
	}
}

func (s *Store) SetLabDB(labDB *sql.DB) {
	s.labDB = labDB
	if s.repo != nil {
		s.repo.store = s
	}
}

func (s *Store) LabDB() *sql.DB {
	return s.labDB
}

func (s *Store) Repo() *Repo {
	if s.repo != nil {
		return s.repo
	}
	s.repo = &Repo{
		store: s,
	}
	return s.repo
}

type Repo struct {
	store *Store
}