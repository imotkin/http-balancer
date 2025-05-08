package migrations

import (
	"database/sql"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"

	"github.com/pressly/goose/v3"
)

func Up(db *sql.DB, dialect string, dir string) error {
	err := goose.SetDialect(dialect)
	if err != nil {
		return err
	}

	err = goose.Up(db, dir)
	if err != nil {
		return err
	}

	return nil
}

func Down(db *sql.DB, dialect string, dir string) error {
	err := goose.SetDialect(dialect)
	if err != nil {
		return err
	}

	err = goose.Down(db, dir)
	if err != nil {
		return err
	}

	return nil
}
