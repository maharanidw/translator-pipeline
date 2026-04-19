# 📖 The Translator Pipeline (Project Blueprint)

**The Translator Pipeline** adalah sistem integrasi AI yang dirancang untuk mem-bypass batasan web-reading pada situs bacaan dan memberikan terjemahan berbasis konteks yang mendalam untuk bahasa tertenu.

---

## 🚀 Fitur Utama
* **Bypass Engine:** Web Extension (Chrome) mengambil teks dari elemen DOM yang diproteksi (anti-copy/right-click) secara otomatis.
* **Context-Aware AI:** Menerjemahkan fiksi dengan menjaga *honorifics* (panggilan) dan nuansa emosional menggunakan Google Gemini API.
* **Smart Rate Limiting & Background Task:** Memproses potongan (*chunking*) teks di background menggunakan *Global Mutex Rate Limiter* untuk mencegah error 429 (Too Many Requests).
* **Web Dashboard (E-Ink Friendly):** Sistem antarmuka pembaca sederhana sekaligus untuk memonitor log progres secara *real-time*.
* **PDF/HTML Export:** Kemampuan satu-klik mengekspor seluruh bab novel menjadi format dokumen PDF/HTML siap cetak.
* **Smart Caching:** Menyimpan hasil terjemahan di PostgreSQL untuk menghemat token API (Cache Hit/Miss).

---

## 🛠️ Tech Stack
* **Backend:** Go (Golang) dengan framework Gin.
* **Hosting:** DigitalOcean App Platform.
* **Database:** PostgreSQL (Managed) dengan GORM sebagai ORM.
* **Web Extension:** JavaScript (Manifest V3) untuk ekstraksi konten.
* **AI Engine:** Google Gemini SDK (Gemini 3.1 Flash Lite, 2.5 Flash, Pro).
* **Frontend:** Vanilla JS, HTML, CSS murni.

---

## 📐 Arsitektur Sistem

### 1. Workflow Data
1.  **Extension:** Ekstraksi teks asli dari tab browser yang aktif -> Kirim *payload* teks ke Go Backend di DigitalOcean.
2.  **Go Backend:** 
    * Validasi request & cek database (Cache Hit/Miss).
    * Jika teks baru, aplikasi melempar ke *background process* (*goroutine*).
    * Pemecahan *Chunking* teks tiap 4500 limit-karakter.
    * Panggil Gemini API dengan pelindung *anti-ban/ rate-limit*.
3.  **PostgreSQL DB:** Menyimpan metadata `Novel`, `Chapter` (termasuk resume progress chunk persentase), serta fitur pelacak target kuota API di tabel `DailyUsage`. 
4.  **Dashboard:** Antarmuka interaktif yang dipanggil *Client* untuk membaca koleksi secara komprehensif atau mengekspor PDF novel utuh.

### 2. Skema Database (GORM)
* `Novel`: Metadata utama buku/koleksi sumber (Judul, Source URL).
* `Chapter`: Menyimpan bagian/bab per terjemahan utuh (`OriginalText`, `TranslatedText`, indikator sinkronisasi chunking).
* `DailyUsage`: Pelacak konsumsi Token/Requests harian per model AI.

---

## 📅 Roadmap & Status Pengembangan
- [x] **Fase 1:** Setup Backend Go (Gin + GORM) & Migrasi DB PostgreSQL.
- [x] **Fase 2:** Pembuatan Extension Browser Chrome (Bypass DOM Content).
- [x] **Fase 3:** Integrasi API Gemini, Logika *Smart Chunking*, dan proteksi *Global Rate Limiter*.
- [x] **Fase 4:** Web Dashboard Backend & Frontend (E-Ink friendly) untuk koleksi novel.
- [x] **Fase 5:** Fitur Monitor *Live System Logs* & Export PDF.
- [ ] **Fase 6:** Implementasi fitur *Custom Glossary* (Kamus Pribadi).

