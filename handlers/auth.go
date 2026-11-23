
package handlers

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// writeJSON mengirim response dalam format JSON.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	result, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return
	}

	_, _ = w.Write(result)
}

// getPost membaca input dari JSON atau form-urlencoded.
func getPost(r *http.Request, key string) (string, error) {
	contentType := strings.TrimSpace(
		strings.Split(r.Header.Get("Content-Type"), ";")[0],
	)

	if contentType == "application/json" {
		var input map[string]any

		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			return "", err
		}

		value, exists := input[key]
		if !exists || value == nil {
			return "", nil
		}

		return fmt.Sprint(value), nil
	}

	err := r.ParseForm()
	if err != nil {
		return "", err
	}

	return r.FormValue(key), nil
}

// isEmptyPHP meniru pengecekan kosong PHP untuk string.
func isEmptyPHP(value string) bool {
	return value == "" || value == "0"
}

// isMD5 memeriksa apakah password sudah berbentuk MD5.
func isMD5(value string) bool {
	if len(value) != 32 {
		return false
	}

	_, err := hex.DecodeString(value)
	return err == nil
}

// normalizeValue mengubah tipe data hasil MySQL menjadi
// tipe data yang aman untuk dikirim sebagai JSON.
func normalizeValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v.Format("2006-01-02")
	default:
		return value
	}
}

// Auth menangani proses login.
func Auth(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"success": false,
				"message": "Method not allowed",
			})
			return
		}

		username, err := getPost(r, "username")
		if err != nil {
			writeJSON(w, 400, map[string]any{
				"success": false,
				"message": "Request tidak valid",
			})
			return
		}

		password, err := getPost(r, "password")
		if err != nil {
			writeJSON(w, 400, map[string]any{
				"success": false,
				"message": "Request tidak valid",
			})
			return
		}

		// Validasi username terlebih dahulu, mengikuti CI3.
		if isEmptyPHP(username) {
			writeJSON(w, 422, map[string]any{
				"success": false,
				"message": "Username harus diisi",
			})
			return
		}

		if isEmptyPHP(password) {
			writeJSON(w, 422, map[string]any{
				"success": false,
				"message": "Password harus diisi",
			})
			return
		}

		// Mempertahankan aturan MD5 dari CI3.
		passwordMD5 := password

		if !isMD5(password) {
			hash := md5.Sum([]byte(password))
			passwordMD5 = fmt.Sprintf("%x", hash)
		}

		query := `
			SELECT
				A.*,
				B.nama_cabang,
				B.lat,
				B.long,
				B.area_coverage,
				B.area_coverage_qr,
				C.nama AS level_user,
				C.visit,
				C.cek_stok,
				C.parent_id,
				D.nama_parent
			FROM tbl_user A
			INNER JOIN tbl_cabang B
				ON B.kode_cabang = A.kode_cabang
			INNER JOIN tbl_level_user C
				ON C.id = A.level
			LEFT JOIN tbl_parent_level D
				ON D.id = C.parent_id
			WHERE A.username = ?
				AND A.password = ?
		`

		rows, err := db.QueryContext(
			r.Context(),
			query,
			username,
			passwordMD5,
		)

		if err != nil {
			writeJSON(w, 500, map[string]any{
				"success": false,
				"message": "Terjadi kesalahan server",
			})
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			writeJSON(w, 500, map[string]any{
				"success": false,
				"message": "Terjadi kesalahan server",
			})
			return
		}

		if !rows.Next() {
			if err := rows.Err(); err != nil {
				writeJSON(w, 500, map[string]any{
					"success": false,
					"message": "Terjadi kesalahan server",
				})
				return
			}

			writeJSON(w, 404, map[string]any{
				"success": false,
				"message": "Username atau password salah",
			})
			return
		}

		// Menyiapkan tempat untuk menampung seluruh kolom hasil query.
		values := make([]any, len(columns))
		scanArgs := make([]any, len(columns))

		for i := range values {
			scanArgs[i] = &values[i]
		}

		err = rows.Scan(scanArgs...)
		if err != nil {
			writeJSON(w, 500, map[string]any{
				"success": false,
				"message": "Terjadi kesalahan server",
			})
			return
		}

		// Mengubah hasil query menjadi map JSON.
		data := make(map[string]any)

		for i, column := range columns {
			data[column] = normalizeValue(values[i])
		}

		// Mempertahankan pengecekan active dari CI3.
		if fmt.Sprint(data["active"]) == "0" {
			writeJSON(w, 403, map[string]any{
				"success": false,
				"message": "User tidak aktif",
			})
			return
		}

		writeJSON(w, 200, map[string]any{
			"success": true,
			"data":    data,
		})
	}
}
