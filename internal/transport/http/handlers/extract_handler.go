package handlers

import (
	"log"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/maharanidw/translator-pipeline/internal/config"
	"github.com/maharanidw/translator-pipeline/internal/models"
	"github.com/maharanidw/translator-pipeline/internal/service"
)

type ExtractPayload struct {
	NovelID      uint   `json:"novel_id"`
	NovelTitle   string `json:"novel_title"`
	ChapterTitle string `json:"chapter_title"`
	URL          string `json:"url"`
	Text         string `json:"text"`
	AIModel      string `json:"ai_model"`
	SourceLang   string `json:"source_lang"`
	TargetLang   string `json:"target_lang"`
}

func ExtractText(c *gin.Context) {
	var payload ExtractPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data tidak valid"})
		return
	}

	if payload.SourceLang == "" {
		payload.SourceLang = "Auto Detect"
	}
	if payload.TargetLang == "" {
		payload.TargetLang = "Indonesian"
	}
	if payload.ChapterTitle == "" {
		payload.ChapterTitle = "Bab Tanpa Judul"
	}

	charCount := utf8.RuneCountInString(payload.Text)

	log.Printf("Menerima teks dari URL: %s | Judul Bab: %s | Ukuran: %d Karakter\n",
		payload.URL, payload.ChapterTitle, charCount)

	// 1. SMART PENGELOMPOKAN NOVEL
	var novel models.Novel
	if payload.NovelID > 0 {
		// User memilih novel yang sudah ada dari dropdown
		if err := config.DB.First(&novel, payload.NovelID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Novel pilihan tidak ada di Database"})
			return
		}
	} else {
		// Membuat Novel Baru
		if err := config.DB.Where("title = ?", payload.NovelTitle).FirstOrCreate(&novel, models.Novel{
			Title:     payload.NovelTitle,
			SourceURL: payload.URL, // Menggunakan URL bab pertama sebagai root
		}).Error; err != nil {
			log.Printf("❌ Gagal membuat novel: %v\n", err)
		}
	}

	var chapter models.Chapter
	if err := config.DB.Where("source_url = ?", payload.URL).FirstOrCreate(&chapter, models.Chapter{
		NovelID:        novel.ID,
		Title:          payload.ChapterTitle,
		SourceURL:      payload.URL,
		OriginalText:   payload.Text,
		LanguageSource: payload.SourceLang,
		IsSynced:       false,
	}).Error; err != nil {
		log.Printf("❌ Gagal membuat chapter dengan FirstOrCreate: %v\n", err)
		// Fallback jika terjadi race condition
		if err := config.DB.Where("source_url = ?", payload.URL).First(&chapter).Error; err != nil {
			log.Printf("❌ Gagal mencari chapter tersembunyi: %v\n", err)
		}
	}

	if chapter.ID == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan chapter ke database"})
		return
	}

	if chapter.IsSynced {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Novel ini sudah pernah diterjemahkan dan selesai sepenuhnya! (Cache Hit🚀)",
		})
		return
	}

	startChunk := chapter.CurrentChunk
	previousTranslation := chapter.TranslatedText

	// 2. CONTEXT-AWARE AI CHUNKING (Berjalan di Background)
	go func(chapterID uint, txt string, selectedModel string, sourceLang string, targetLang string, start int, prevTrans string) {
		if selectedModel == "" {
			selectedModel = "gemini-3.1-flash-lite-preview"
		}

		log.Printf("--> 🚀 [Background Task] Memulai proses terjemahan AI (mulai dari chunk %d) dengan %s...\n", start, selectedModel)

		// Callback ini akan dipanggil tiap kali 1 chunk berhasil dari ai_service
		onChunkSuccess := func(currentTranslated string, currentTokens int, currentReq int, totalReq int) {
			// Simpan teks partial ke database
			config.DB.Model(&models.Chapter{}).Where("id = ?", chapterID).Updates(models.Chapter{
				TranslatedText: currentTranslated,
				AIModelUsed:    selectedModel,
				IsSynced:       false, // Masih pending karena belum 100%
				CurrentChunk:   currentReq,
				TotalChunks:    totalReq,
			})
			log.Printf("   [DB Sync] Progress sinkronisasi ke-%d/%d berhasil disimpan ke DB...", currentReq, totalReq)
		}

		translatedTxt, requestsCount, totalTokens, err := service.TranslateText(txt, selectedModel, sourceLang, targetLang, start, prevTrans, onChunkSuccess)

		// 3. Simpan hasil terjemahan ke DB SEBERAPAPUN HASILNYA (Full / Terpotong Quota)
		config.DB.Model(&models.Chapter{}).Where("id = ?", chapterID).Updates(models.Chapter{
			TranslatedText: translatedTxt,
			AIModelUsed:    selectedModel,
			IsSynced:       err == nil, // True jika sukses 100%, False jika terpotong error
		})

		// 4. Update data KUOTA Harian
		today := time.Now().Truncate(24 * time.Hour)
		var usage models.DailyUsage
		config.DB.Where(&models.DailyUsage{Date: today, AIModel: selectedModel}).FirstOrCreate(&usage)

		usage.Requests += requestsCount
		usage.Tokens += totalTokens
		config.DB.Save(&usage)

		if err != nil {
			log.Println("--> ⚠️ [Background Task] TERHENTI/GAGAL:", err)
			return
		}

		log.Printf("--> ✅ [Background Task] SELESAI Penuh! Hasil terjemahan disimpan (ID: %d)\n", chapterID)
		log.Printf("--> 📊 [Penggunaan Kuota %s Hari Ini] Total Request: %d | Total Token: %d\n", selectedModel, usage.Requests, usage.Tokens)
	}(chapter.ID, payload.Text, payload.AIModel, payload.SourceLang, payload.TargetLang, startChunk, previousTranslation)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Data berhasil diterima. Server sedang menerjemahkannya di background (cek terminal atau database nanti).",
	})
}
