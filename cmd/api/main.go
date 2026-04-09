package main

import (
	"io"
	"log"
	"os"

	"github.com/maharanidw/translator-pipeline/internal/config"
	"github.com/maharanidw/translator-pipeline/internal/transport/http/routes"

	"github.com/joho/godotenv"
)

func main() {
	// 1. SETUP LOGGING KE FILE (Agar bisa dibaca saat aplikasi berjalan di Background)
	f, err := os.OpenFile("translator.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Gagal membuka atau membuat file log: %v", err)
	}
	defer f.Close()

	// Menulis log ke File DAN ke layar Terminal sekaligus
	multiWriter := io.MultiWriter(os.Stdout, f)
	log.SetOutput(multiWriter)
	
	// Menambahkan tanggal, waktu, dan asal file pada tiap baris log
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			log.Panicf("failed to load .env file: %v", err)
		}
	}


	config.InitDB()


	r := routes.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888" // Default port
	}

	log.Printf("SERVER READY || Listening on port http://localhost:%s\n", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
