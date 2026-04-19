package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/maharanidw/translator-pipeline/internal/config"
	"github.com/maharanidw/translator-pipeline/internal/models"
)

func main() {
	// Load environment variables (.env)
	if err := godotenv.Load(); err != nil {
		log.Println("Peringatan: file .env tidak ditemukan, menggunakan variabel environment yang ada.")
	}

	// Inisialisasi koneksi Database
	config.InitDB()

	// Buka file json yang merupakan hasil export
	file, err := os.Open("database/missing_chapter.json")
	if err != nil {
		log.Fatalf("Gagal membuka file JSON: %v", err)
	}
	defer file.Close()

	// Karena format waktu (Timestamp) dari export JSON Supabase tidak sesuai dengan
	// parser standar JSON Go (RFC3339), kita decode saja field yang penting ke struct anonim,
	// lalu kita pindahkan isinya ke model asli agar Gorm otomatis reset ulang waktu created/updated_at-nya.
	type RawChapter struct {
		NovelID        uint   `json:"novel_id"`
		ChapterNumber  int    `json:"chapter_number"`
		Title          string `json:"title"`
		SourceURL      string `json:"source_url"`
		OriginalText   string `json:"original_text"`
		TranslatedText string `json:"translated_text"`
		LanguageSource string `json:"language_source"`
		AIModelUsed    string `json:"ai_model_used"`
		IsSynced       bool   `json:"is_synced"`
		CurrentChunk   int    `json:"current_chunk"`
		TotalChunks    int    `json:"total_chunks"`
	}

	var rawChapters []RawChapter
	if err := json.NewDecoder(file).Decode(&rawChapters); err != nil {
		log.Fatalf("Gagal melakukan decode JSON: %v", err)
	}

	for _, rawValue := range rawChapters {
		// Bentuk menjadi struct models asli yang siap masuk GORM
		chapter := models.Chapter{
			Title:          rawValue.Title,
			ChapterNumber:  rawValue.ChapterNumber,
			SourceURL:      rawValue.SourceURL,
			OriginalText:   rawValue.OriginalText,
			TranslatedText: rawValue.TranslatedText,
			LanguageSource: rawValue.LanguageSource,
			AIModelUsed:    rawValue.AIModelUsed,
			IsSynced:       rawValue.IsSynced,
			CurrentChunk:   rawValue.CurrentChunk,
			TotalChunks:    rawValue.TotalChunks,
		}

		// 1. Cari / Buat Ulang Novel yang hilang
		var novel models.Novel

		// Fallback URL jika source URL belum tertampung sempurna sebagai URL dasar novelnya
		// Tapi untuk data ini, kita ikuti pattern mencari source_url dari URL novel yang di parsing di ekstensi.
		// Jika tidak ketemu, kita buat baru.
		err := config.DB.Unscoped().Where("title = ? OR source_url = ?", chapter.Title, chapter.SourceURL).First(&novel).Error

		if err != nil {
			log.Printf("Novel '%s' rupanya sudah tidak ada di database, membuat novel baru...", chapter.Title)
			novel = models.Novel{
				Title:     chapter.Title, // Menggunakan judul chapter karena metadata judul novel hilang
				SourceURL: chapter.SourceURL,
			}
			if createErr := config.DB.Create(&novel).Error; createErr != nil {
				log.Fatalf("❌ Gagal membuat Novel baru: %v", createErr)
			}
			log.Printf("✅ Novel baru berhasil dibuat (ID: %d)", novel.ID)
		} else {
			log.Printf("ℹ️ Novel sudah ada dengan ID: %d", novel.ID)
			if novel.DeletedAt.Valid {
				// Restore jika novel soft-delete
				config.DB.Exec("UPDATE novels SET deleted_at = NULL WHERE id = ?", novel.ID)
				log.Println("♻️ Novel dikembalikan dari status terhapus (Soft Delete)")
			}
		}

		// 2. Modifikasi Data Chapter sebelum Inserts
		// Pastikan mereferensikan Novel ID terbaru
		chapter.NovelID = novel.ID

		// Kosongkan Primary ID (0) agar PostgreSQL otomatis meng-generate ID Auto-Increment baru,
		// mencegah bentrok ID (Duplicate Key) jika ID 1 sudah terpakai di Supabase
		chapter.ID = 0

		// Simpan utuh ke database berikut 100% fieldnya
		if err := config.DB.Create(&chapter).Error; err != nil {
			log.Printf("❌ Gagal merestore chapter '%s': %v\n", chapter.Title, err)
		} else {
			log.Printf("🎉 SUKSES! Chapter '%s' berhasil dikembalikan (Chapter ID Baru: %d) bernaung di (Novel ID Baru: %d)\n", chapter.Title, chapter.ID, novel.ID)
		}
	}
}
