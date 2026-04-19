package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/maharanidw/translator-pipeline/internal/config"
	"github.com/maharanidw/translator-pipeline/internal/models"
)

// Menggabungkan seluruh hasil terjemahan dalam satu novel ke dalam halaman khusus untuk di Print PDF
func ExportNovel(c *gin.Context) {
	novelID := c.Param("id")

	var novel models.Novel
	if err := config.DB.First(&novel, novelID).Error; err != nil {
		c.String(http.StatusNotFound, "Kesalahan: Data Novel tidak ditemukan")
		return
	}

	var chapters []models.Chapter
	// Urutkan berdasarkan ID secara Ascending (Berdasarkan jadwal narik dari URL / waktu input)
	if err := config.DB.Where("novel_id = ?", novelID).Order("id asc").Find(&chapters).Error; err != nil {
		c.String(http.StatusInternalServerError, "Kesalahan: Gagal mengambil susunan chapters")
		return
	}

	if len(chapters) == 0 {
		c.String(http.StatusNotFound, "Kesalahan: Novel ini belum memiliki chapter sama sekali.")
		return
	}

	var sb strings.Builder

	// Build Minimalist HTML strictly for Printing / E-Reader
	sb.WriteString("<!DOCTYPE html>\n<html lang='id'>\n<head>\n")
	sb.WriteString("<meta charset='UTF-8'>\n")
	sb.WriteString(fmt.Sprintf("<title>%s - Export</title>\n", novel.Title))
	sb.WriteString("<style>\n")
	sb.WriteString(`
		body {
			font-family: 'Georgia', serif;
			line-height: 1.8;
			color: #000;
			max-width: 800px;
			margin: 0 auto;
			padding: 40px 20px;
		}
		h1 {
			text-align: center;
			font-size: 28px;
			margin-bottom: 20px;
			page-break-before: always;
		}
		.novel-title {
			text-align: center;
			font-size: 42px;
			font-weight: bold;
			margin-bottom: 300px;
			margin-top: 15rem;
			page-break-after: always;
		}
		.chapter-content {
			font-size: 18px;
			text-align: justify;
			margin-top: 30px;
			white-space: pre-wrap;
		}
		
		/* Hilangkan elemen yang gak penting saat print */
		@media print {
			body { max-width: 100%; margin: 0; padding: 0; }
			@page { margin: 2.5cm; }
		}
	`)
	sb.WriteString("</style>\n</head>\n<body>\n")

	// Halaman Sampul Depan (Cover Page)
	sb.WriteString(fmt.Sprintf("<div class='novel-title'>%s</div>\n", novel.Title))

	// Halaman Bab (Chapters content)
	for _, ch := range chapters {
		sb.WriteString(fmt.Sprintf("<h1>%s</h1>\n", ch.Title))
		if ch.IsSynced {
			sb.WriteString(fmt.Sprintf("<div class='chapter-content'>%s</div>\n", ch.TranslatedText))
		} else {
			sb.WriteString(fmt.Sprintf("<div class='chapter-content'><i>[Catatan: Chapter ini belum selesai 100%%]</i>\n\n%s</div>\n", ch.TranslatedText))
		}
	}

	// Auto Trigger fitur Print browser
	sb.WriteString("<script>\n")
	sb.WriteString("window.onload = function() {\n")
	sb.WriteString("   setTimeout(function() { window.print(); }, 500);\n")
	sb.WriteString("};\n")
	sb.WriteString("</script>\n")

	sb.WriteString("</body>\n</html>")

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(sb.String()))
}
