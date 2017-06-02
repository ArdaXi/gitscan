package database

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Result struct {
	ID          int
	Project     int
	Commit      string
	Path        string
	Caption     string
	Description string
}

type DB struct {
	db *sqlx.DB
}

func Connect(dsn string) (*DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

func (db *DB) GetLastScanned(projectID int) (string, error) {
	var lastScannedID sql.NullString
	err := db.db.QueryRow("SELECT last_scanned_id FROM projects WHERE id=$1", projectID).Scan(
		&lastScannedID,
	)
	if err == sql.ErrNoRows {
		_, err = db.db.Exec(
			"INSERT INTO projects (id) VALUES ($1)",
			projectID,
		)
	}
	return lastScannedID.String, err
}

func (db *DB) SetLastScanned(projectID int, commitID string) error {
	_, err := db.db.Exec("UPDATE projects SET last_scanned_id=$1 WHERE id=$2", commitID, projectID)
	return err
}

func (db *DB) AddResult(result *Result) error {
	_, err := db.db.Exec(
		"INSERT INTO results (project, commit, path, caption, description) VALUES ($1, $2, $3, $4, $5)",
		result.Project,
		result.Commit,
		result.Path,
		result.Caption,
		result.Description,
	)
	return err
}
