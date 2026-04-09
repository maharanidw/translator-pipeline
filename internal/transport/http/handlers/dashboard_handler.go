package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/maharanidw/translator-pipeline/internal/config"
	"github.com/maharanidw/translator-pipeline/internal/models"
)

// Mengambil sistem log dari file translator.log
func GetSystemLogs(c *gin.Context) {
	data, err := os.ReadFile("translator.log")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": "Membaca log ditunda... (File translator.log belum dibuat oleh sistem)"})
		return
	}

	lines := strings.Split(string(data), "\n")

	// Kita ambil 50-60 baris terakhir saja agar tidak terlalu berat memonitor log
	start := len(lines) - 60
	if start < 0 {
		start = 0
	}

	res := strings.Join(lines[start:], "\n")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": res})
}

// Membersihkan / mereset isi file log
func ClearSystemLogs(c *gin.Context) {
	err := os.WriteFile("translator.log", []byte(""), 0666)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Gagal membersihkan isi log"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Log berhasil dibersihkan!"})
}

// Ambil daftar Novel yang ada
func GetNovels(c *gin.Context) {
	var novels []models.Novel
	config.DB.Order("id desc").Find(&novels)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": novels})
}

// Menghapus Novel beserta semua Chapter di dalamnya
func DeleteNovel(c *gin.Context) {
	novelID := c.Param("id")

	// 1. Hapus semua chapter yang terkait dengan Novel ini
	if err := config.DB.Where("novel_id = ?", novelID).Delete(&models.Chapter{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Gagal menghapus chapters dari novel ini"})
		return
	}

	// 2. Hapus Novel-nya
	if err := config.DB.Where("id = ?", novelID).Delete(&models.Novel{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Gagal menghapus novel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Novel beserta semua isinya berhasil dihapus"})
}

// Ambil daftar Chapter dari sebuah Novel
func GetChaptersByNovelID(c *gin.Context) {
	novelID := c.Param("id")
	var chapters []models.Chapter
	// Jangan ambil OriginalText & TranslatedText agar API ringan saat melist chapter
	config.DB.Select("id", "novel_id", "chapter_number", "title", "source_url", "is_synced", "ai_model_used", "current_chunk", "total_chunks").
		Where("novel_id = ?", novelID).
		Order("chapter_number asc").
		Find(&chapters)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": chapters})
}

// Ambil isi lengkap teks hasil terjemahan dari spesifik Chapter
func GetChapterByID(c *gin.Context) {
	chapterID := c.Param("id")
	var chapter models.Chapter
	if err := config.DB.First(&chapter, chapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Chapter tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": chapter})
}

// Ambil status progres chapter dari URL spesifik (Untuk Extension Popup)
func GetChapterStatusByURL(c *gin.Context) {
	sourceURL := c.Query("url")
	if sourceURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "URL parameter wajib diisi"})
		return
	}

	var chapter models.Chapter
	if err := config.DB.Select("is_synced", "current_chunk", "total_chunks").Where("source_url = ?", sourceURL).First(&chapter).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Belum ada chapter di URL ini"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": chapter})
}
