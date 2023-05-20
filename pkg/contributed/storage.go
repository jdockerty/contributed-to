package contributed

import (
	"context"
	"database/sql"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	// SQLite database driver, this is the only one in use.
	DatabaseDriver = "sqlite"

	// Name of the SQLite database.
	DatabaseName = "/tmp/contributed-to.local.db"

	// Static query to return merged pull requests for a particular user
	QueryUserMergedPullRequests = "" // TODO

	// InitDBStatement is used to initialise the database with the relevant
	// tables and entity relationships.
	InitDBStatement = `
	CREATE TABLE Authors(
        id SERIAL PRIMARY KEY,
        author TEXT NOT NULL
    );

    CREATE TABLE RepoMetadata(
        repo_id SERIAL PRIMARY KEY,
        owner TEXT NOT NULL,
        repository TEXT NOT NULL,
        avatar_url TEXT
    );

    CREATE TABLE PullRequests(
        id SERIAL PRIMARY KEY,
        author_id INT NOT NULL,
        repository_id INT NOT NULL,

        CONSTRAINT fk_author FOREIGN KEY(author_id) REFERENCES Authors(id),
        CONSTRAINT fk_repository FOREIGN KEY(repository_id) REFERENCES RepoMetadata(id)
    );
	`
)

// Helper function for recognising an already initialised database.
func dbInitialised(err error) bool { return strings.Contains(err.Error(), "already exists") }

// Initialise the database with the correct tables and relationships.
func InitDB(db *sql.DB) error {

	_, err := db.Exec(InitDBStatement)
	if err != nil {
		// Catch an already initialised database
		if dbInitialised(err) {
			return nil
		}
		return err
	}

	return nil

}

// MergedPullRequestData is a wrapper struct around the
// merged pull request information and the database store.
type MergedPullRequestData struct {
	MergedPullRequests MergedPullRequestInfo
	DB                 *sql.DB
}

// FetchFromDB will retrieve the merged pull requests for a user from the database.
func FetchFromDB(ctx context.Context, db *sql.DB) error {

	//db, err := sql.Open(DatabaseDriver, DatabaseName)
	//if err != nil {
	//	return err
	//}
	//defer db.Close()

	if err := db.Ping(); err != nil {
		return err
	}

	return nil

}
