package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func OpenFromEnv() (*sql.DB, error) {
	host := strings.TrimSpace(os.Getenv("PGHOST"))
	port := strings.TrimSpace(os.Getenv("PGPORT"))
	user := strings.TrimSpace(os.Getenv("PGUSER"))
	password := os.Getenv("PGPASSWORD")
	dbname := strings.TrimSpace(os.Getenv("PGNAME"))
	sslmode := strings.TrimSpace(os.Getenv("PGSSLMODE"))

	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}

	if user == "" || dbname == "" {
		return nil, fmt.Errorf("missing postgres configuration: PGUSER and PGNAME are required")
	}

	dsn := buildDSN(host, port, user, password, dbname, sslmode)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		if strings.Contains(strings.ToLower(err.Error()), "does not exist") {
			if createErr := createDatabase(host, port, user, password, dbname, sslmode); createErr != nil {
				return nil, createErr
			}
			db, err = sql.Open("postgres", dsn)
			if err != nil {
				return nil, err
			}
			if err := db.Ping(); err != nil {
				db.Close()
				return nil, err
			}
			return db, nil
		}
		return nil, err
	}
	return db, nil
}

func buildDSN(host, port, user, password, dbname, sslmode string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", host, port, user, password, dbname, sslmode)
}

func createDatabase(host, port, user, password, dbname, sslmode string) error {
	adminDSN := buildDSN(host, port, user, password, "postgres", sslmode)
	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return err
	}
	defer adminDB.Close()
	if err := adminDB.Ping(); err != nil {
		return err
	}

	quoted := `"` + strings.ReplaceAll(dbname, `"`, `""`) + `"`
	if _, err := adminDB.Exec("CREATE DATABASE " + quoted); err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "already exists") {
			return nil
		}
		return err
	}
	return nil
}
