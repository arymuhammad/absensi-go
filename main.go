package main

import (
	"log"
	"net/http"

	"absensi-go/config"
	"absensi-go/routes"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("File .env tidak ditemukan, menggunakan environment variable sistem")
	}

	db, err := config.ConnectDB()
	if err != nil {
		log.Fatal("Koneksi database gagal: ", err)
	}
	defer db.Close()

	log.Println("Koneksi MySQL berhasil!")

	handler := routes.Setup(db)

	log.Println("Server berjalan di port 8080")

	err = http.ListenAndServe(":8080", handler)
	if err != nil {
		log.Fatal("HTTP Server gagal: ", err)
	}
}
