
package routes

import (
	"database/sql"
	"net/http"

	"absensi-go/handlers"
)

func Setup(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	// Endpoint login.
	mux.HandleFunc("/auth", handlers.Auth(db))

	// Endpoint pemeriksaan koneksi database.
	mux.HandleFunc("/test_db", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("FAILED"))
			return
		}

		_, _ = w.Write([]byte("CONNECTED"))
	})

	return mux
}
