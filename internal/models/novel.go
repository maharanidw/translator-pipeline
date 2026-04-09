package models

// informasi utama dari buku atau cerita yang dibaca
type Novel struct {
	ID        uint      `gorm:"primaryKey"`
	Title     string    `gorm:"size:255;not null;index"` // Judul cerita
	Author    string    `gorm:"size:100"`                // Penulis
	Source    string    `gorm:"size:50"`
	SourceURL string    `gorm:"uniqueIndex"` // URL utama seri/cerita
	Chapters  []Chapter // Relasi One-to-Many ke Chapters

	Timestamp
}

// Chapter menyimpan konten asli dan hasil terjemahan AI per bab.
type Chapter struct {
	ID            uint   `gorm:"primaryKey"`
	NovelID       uint   `gorm:"index"`       // Foreign Key ke Novel
	ChapterNumber int    `gorm:"index"`       // Urutan bab
	Title         string `gorm:"size:255"`    // Judul bab (jika ada)
	SourceURL     string `gorm:"uniqueIndex"` // URL spesifik bab ini

	// Content Section
	OriginalText   string `gorm:"type:text"` // Teks asli
	TranslatedText string `gorm:"type:text"` // Teks hasil terjemahan AI

	// Metadata untuk AI Context
	LanguageSource string `gorm:"size:50"`             // e.g., "ko", "zh"
	AIModelUsed    string `gorm:"size:50"`             // e.g., "gemini-1.5-pro"
	IsSynced       bool   `gorm:"default:false;index"` // Status apakah sudah didownload
	CurrentChunk   int    // Chunk/bagian saat ini yang berhasil diterjemahkan
	TotalChunks    int    // Total chunk/bagian teks

	Timestamp
}

// UserPreference (Optional) untuk menyimpan "Glossary" pribadi kamu.
type Glossary struct {
	ID          uint   `gorm:"primaryKey"`
	Term        string `gorm:"size:100;index"` // Kata asli (misal: nama karakter)
	Translation string `gorm:"size:100"`       // Terjemahan pilihanmu
	Language    string `gorm:"size:10"`        // "ko" atau "zh"
}
