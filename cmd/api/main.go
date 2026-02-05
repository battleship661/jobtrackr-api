package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, err := openDB()
	if err != nil {
		log.Fatalf("db open failed: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	http.HandleFunc("/health/db", func(w http.ResponseWriter, r *http.Request) {
		ctxTimeout := 2 * time.Second
		if err := pingDB(db, ctxTimeout); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(fmt.Sprintf(`{"status":"db_down","error":%q}`, err.Error())))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"db_ok"}`))
	})

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func openDB() (*sql.DB, error) {
	host := getenv("DB_HOST", "localhost")
	port := getenv("DB_PORT", "5434")
	user := getenv("DB_USER", "jobtrackr")
	pass := getenv("DB_PASSWORD", "jobtrackr_password")
	name := getenv("DB_NAME", "jobtrackr")
	ssl := getenv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, ssl,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	// Reasonable defaults for a small service
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	return db, nil
}

func pingDB(db *sql.DB, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := db.Ping(); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return db.Ping()
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

