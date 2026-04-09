# 📖 The Translator Pipeline (Project Blueprint)

**The Translator Pipeline** adalah sistem integrasi AI yang dirancang untuk mem-bypass batasan web-reading pada situs fiksi (seperti Postype, AO3, AFF) dan memberikan terjemahan berbasis konteks yang mendalam untuk bahasa Korea dan Mandarin ke Bahasa Inggris

---

## 🚀 Fitur Utama
* **Bypass Engine:** Mengambil teks dari elemen DOM yang diproteksi (anti-copy/right-click) secara otomatis.
* **Context-Aware AI:** Menerjemahkan fiksi dengan menjaga *honorifics* (panggilan) dan nuansa emosional menggunakan LLM.
* **Smart Caching:** Menyimpan hasil terjemahan di PostgreSQL untuk menghemat token API dan akses cepat di kemudian hari.
* **Custom Glossary:** Kamus pribadi untuk menjaga konsistensi nama karakter dan istilah khusus.

---

## 🛠️ Tech Stack
* **Backend:** Go (Golang) dengan framework Gin.
* **Database:** PostgreSQL dengan GORM sebagai ORM.
* **Web Extension:** JavaScript (Manifest V3) untuk ekstraksi konten.
* **AI Engine:** Gemini 1.5 Pro / GPT-4o API.
* **Scraper:** Colly & Playwright-Go (untuk bypass situs berat).
* **Target Device:** Android

---

## 📐 Arsitektur Sistem

### 1. Workflow Data
1.  **Extension:** Ekstraksi teks asli dari browser -> Kirim ke Go Backend.
2.  **Go Backend:** * Validasi request & cek database (Cache Hit/Miss).
    * Proses *Chunking* teks (jika bab terlalu panjang).
    * Panggil AI API dengan *System Prompt* khusus.
3.  **PostgreSQL:** Simpan metadata novel, bab, dan teks terjemahan.
4.  **Sync Device:** Perangkat Android mengambil data terbaru via REST API.

### 2. Skema Database (GORM)
* `Novel`: Simpan metadata (Judul, Author, Source URL).
* `Chapter`: Simpan `OriginalText`, `TranslatedText`, dan urutan bab.
* `Glossary`: Simpan preferensi kata/istilah khusus.

---

## 📅 Roadmap Pengembangan
- [ ] **Fase 1:** Setup Backend Go (Gin + GORM) & Migrasi DB.
- [ ] **Fase 2:** Pembuatan Extension Browser (Bypass Postype DOM).
- [ ] **Fase 3:** Integrasi API AI & Logika *Smart Chunking*.
- [ ] **Fase 4:** Dashboard pembaca sederhana (Web App) untuk Onyx Boox.
- [ ] **Fase 5:** Fitur *Sync Offline* ke perangkat.

---

## 💡 Catatan Implementasi
> **Penting:** Untuk Postype, fokus pada penggunaan `querySelectorAll` pada class konten utama untuk mendapatkan teks tanpa terhalang proteksi script pada layer atas.