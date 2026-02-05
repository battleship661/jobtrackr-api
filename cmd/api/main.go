package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, err := openDB()
	if err != nil {
		log.Fatalf("db open failed: %v", err)
	}
	defer db.Close()

	// Health endpoints
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	http.HandleFunc("/health/db", func(w http.ResponseWriter, r *http.Request) {
		if err := pingDB(db, 2*time.Second); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "db_down",
				"error":  err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "db_ok"})
	})

	// Applications collection: POST /applications, GET /applications
	http.HandleFunc("/applications", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromHeader(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		switch r.Method {
		case http.MethodPost:
			var req CreateApplicationRequest
			if err := readJSON(r, &req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
				return
			}
			app, err := createApplication(db, userID, req)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, app)

		case http.MethodGet:
			apps, err := listApplications(db, userID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list"})
				return
			}
			writeJSON(w, http.StatusOK, apps)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Applications item: GET /applications/{id}, DELETE /applications/{id}
	http.HandleFunc("/applications/", func(w http.ResponseWriter, r *http.Request) {
		userID, err := userIDFromHeader(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/applications/")
		if id == "" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			app, err := getApplication(db, userID, id)
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get"})
				return
			}
			writeJSON(w, http.StatusOK, app)

		case http.MethodPatch:
			var req UpdateApplicationRequest
			if err := readJSON(r, &req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
				return
			}

			app, err := updateApplication(db, userID, id, req)
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			if err != nil {
                                log.Printf("update failed: %v", err)
                        	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}

			writeJSON(w, http.StatusOK, app)

		case http.MethodDelete:
			err := deleteApplication(db, userID, id)
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete"})
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

/* -------------------- Models / DTOs -------------------- */

type Application struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Company     string `json:"company"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	Source      string `json:"source,omitempty"`
	AppliedDate string `json:"applied_date,omitempty"` // YYYY-MM-DD
	Notes       string `json:"notes,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type CreateApplicationRequest struct {
	Company     string `json:"company"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	Source      string `json:"source"`
	AppliedDate string `json:"applied_date"` // YYYY-MM-DD (optional)
	Notes       string `json:"notes"`
}
type UpdateApplicationRequest struct {
	Status      string `json:"status"`
	Notes       string `json:"notes"`
	AppliedDate string `json:"applied_date"`
}

/* -------------------- Helpers -------------------- */

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func userIDFromHeader(r *http.Request) (string, error) {
	uid := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if uid == "" {
		return "", errors.New("missing X-User-Id header")
	}
	return uid, nil
}

/* -------------------- DB: Connection -------------------- */

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

/* -------------------- DB: Applications -------------------- */

func createApplication(db *sql.DB, userID string, req CreateApplicationRequest) (Application, error) {
	if strings.TrimSpace(req.Company) == "" {
		return Application{}, errors.New("company is required")
	}
	if strings.TrimSpace(req.Role) == "" {
		return Application{}, errors.New("role is required")
	}
	if strings.TrimSpace(req.Status) == "" {
		req.Status = "applied"
	}

	var app Application
	app.UserID = userID

	query := `
		INSERT INTO applications (user_id, company, role, status, source, applied_date, notes)
		VALUES ($1,$2,$3,$4,$5, NULLIF($6,'')::date, $7)
		RETURNING id, company, role, status,
		          COALESCE(source,''),
		          COALESCE(to_char(applied_date,'YYYY-MM-DD'),''),
		          COALESCE(notes,''),
		          to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		          to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`
	err := db.QueryRow(query, userID, req.Company, req.Role, req.Status, req.Source, req.AppliedDate, req.Notes).
		Scan(&app.ID, &app.Company, &app.Role, &app.Status, &app.Source, &app.AppliedDate, &app.Notes, &app.CreatedAt, &app.UpdatedAt)
	return app, err
}

func listApplications(db *sql.DB, userID string) ([]Application, error) {
	query := `
		SELECT id, user_id, company, role, status,
		       COALESCE(source,''),
		       COALESCE(to_char(applied_date,'YYYY-MM-DD'),''),
		       COALESCE(notes,''),
		       to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM applications
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Application
	for rows.Next() {
		var a Application
		if err := rows.Scan(&a.ID, &a.UserID, &a.Company, &a.Role, &a.Status, &a.Source, &a.AppliedDate, &a.Notes, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func getApplication(db *sql.DB, userID, id string) (Application, error) {
	query := `
		SELECT id, user_id, company, role, status,
		       COALESCE(source,''),
		       COALESCE(to_char(applied_date,'YYYY-MM-DD'),''),
		       COALESCE(notes,''),
		       to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		       to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM applications
		WHERE user_id = $1 AND id = $2
	`
	var a Application
	err := db.QueryRow(query, userID, id).
		Scan(&a.ID, &a.UserID, &a.Company, &a.Role, &a.Status, &a.Source, &a.AppliedDate, &a.Notes, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

func deleteApplication(db *sql.DB, userID, id string) error {
	res, err := db.Exec(`DELETE FROM applications WHERE user_id=$1 AND id=$2`, userID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func updateApplication(db *sql.DB, userID, id string, req UpdateApplicationRequest) (Application, error) {
	query := `
		UPDATE applications
		SET
			status = COALESCE(NULLIF($3,''), status),
			notes = COALESCE(NULLIF($4,''), notes),
			applied_date = COALESCE(NULLIF($5,'')::date, applied_date),
			updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id, user_id, company, role, status,
		          COALESCE(source,''),
		          COALESCE(to_char(applied_date,'YYYY-MM-DD'),''),
		          COALESCE(notes,''),
		          to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		          to_char(updated_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`

	var a Application
	err := db.QueryRow(query, userID, id, req.Status, req.Notes, req.AppliedDate).
		Scan(&a.ID, &a.UserID, &a.Company, &a.Role, &a.Status,
			&a.Source, &a.AppliedDate, &a.Notes, &a.CreatedAt, &a.UpdatedAt)

	return a, err
}
