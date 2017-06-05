package database

import (
	"database/sql"
	"log"

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

func (db *DB) ResultCount(projectID int) (int, error) {
	var count int
	err := db.db.QueryRow("SELECT COUNT(*) FROM results WHERE project=$1", projectID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (db *DB) GetResults(projectID int) (<-chan *Result, error) {
	rows, err := db.db.Query("SELECT id, commit, path, caption, description FROM results WHERE project=$1", projectID)
	if err != nil {
		return nil, err
	}

	c := make(chan *Result, 20)

	go func(c chan<- *Result, rows *sql.Rows) {
		defer rows.Close()
		for rows.Next() {
			var result Result
			if err := rows.Scan(&result.ID, &result.Commit, &result.Path, &result.Caption, &result.Description); err != nil {
				log.Printf("Error scanning row: %s", err)
				continue
			}
			c <- &result
		}
		close(c)
	}(c, rows)

	return c, nil
}
