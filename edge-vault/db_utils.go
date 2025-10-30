package main

import (
	"database/sql"
	"fmt"
	"sync"
)

type DBManager struct {
	db_sqlite *sql.DB
	db_pgsql  *sql.DB
	mu        sync.Mutex
}

const (
	host     = "192.168.230.1"
	port     = 5432
	user     = "su_admin"
	password = "changeme"
	dbname   = "cp_integration"
)

var psqlInfo = fmt.Sprintf("host=%s port=%d user=%s "+
	"password=%s dbname=%s sslmode=disable",
	host, port, user, password, dbname)

func NewDBManager() (*DBManager, error) {
	db1, err := sql.Open("sqlite", "sqlite.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database 1: %w", err)
	}
	db2, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to open database 2: %w", err)
	}

	// Ping to verify connections.
	if err := db1.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database 1: %w", err)
	}
	if err := db2.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database 2: %w", err)
	}

	return &DBManager{
		db_sqlite: db1,
		db_pgsql:  db2,
	}, nil
}
