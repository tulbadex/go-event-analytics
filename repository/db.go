package repository

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func GetDB() *sql.DB {
	connStr := "postgres://postgres:password@localhost:5432/eventdb?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error connecting to DB: %v", err)
	}
	return db
}
