package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var (
	globalAIMutex sync.Mutex
	lastAPIcall   time.Time
)

// ChunkText memecah teks panjang menjadi beberapa bagian (batas maxChars)
// agar hasil terjemahan dari AI tidak terpotong akibat output limit (8k token).
func ChunkText(text string, maxChars int) []string {
	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current strings.Builder

	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Jika paragraf ini ditambahkan akan melebihi batas, simpan chunk saat ini
		if current.Len()+len(p) > maxChars {
			if current.Len() > 0 {
				chunks = append(chunks, current.String())
				current.Reset()
			}
		}
		current.WriteString(p)
		current.WriteString("\n\n")
	}
	// Masukkan sisa teks yang belum tersimpan
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

// TranslateText menerjemahkan teks menggunakan Gemini API,
// mengembalikan teks hasil, jumlah request, total token, dan error (jika ada).
func TranslateText(text string, modelName string, sourceLang string, targetLang string, startChunk int, previousTranslation string, onChunkSuccess func(currentText string, currentTokens int, currentRequest int, totalRequests int)) (string, int, int, error) {
	ctx := context.Background()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return previousTranslation, 0, 0, fmt.Errorf("GEMINI_API_KEY belum dikonfigurasi di file .env")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return previousTranslation, 0, 0, err
	}
	defer client.Close()

	if modelName == "" || modelName == "gemini-1.5-flash-latest" {
		modelName = "gemini-2.5-flash"
	}
	model := client.GenerativeModel(modelName)

	// MATIKAN SEMUA FILTER KETAT (Bypass BlockReason)
	// Karena fiksi seringkali memuat adegan aksi, kekerasan ringan, atau emosi kuat
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockNone,
		},
	}

	var translationPrompt string
	if sourceLang == "Auto Detect" || sourceLang == "" {
		translationPrompt = fmt.Sprintf("Translate the following fiction text accurately to %s.", targetLang)
	} else {
		translationPrompt = fmt.Sprintf("Translate the following fiction text from %s to %s accurately.", sourceLang, targetLang)
	}

	// System Instruction khusus (Context-Aware AI)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text(fmt.Sprintf("You are an expert translator specializing in web novels and fan fiction. %s Maintain the emotional nuance, tone, standard formatting, and honorifics (e.g., Hyung, Oppa, Sunbae, shidi). Do NOT add any conversational filler, translator notes, or extra explanations. Return ONLY the translated %s text.", translationPrompt, targetLang)),
		},
	}

	chunks := ChunkText(text, 4500) // Pecah teks tiap ~4500 karakter
	var translated strings.Builder
	translated.WriteString(previousTranslation)
	if previousTranslation != "" {
		translated.WriteString("\n\n")
	}

	log.Printf("[AI Service] Memulai terjemahan dengan %s, total %d chunks...\n", modelName, len(chunks))
	totalTokens := 0
	retryCount := 0

	for i := startChunk; i < len(chunks); i++ {
		chunk := chunks[i]
		log.Printf("[AI Service] Menerjemahkan chunk %d dari %d...\n", i+1, len(chunks))

		// 🚦 SMART GLOBAL RATE LIMITER (Mencegah Spike API Call dari Background Tasks)
		// Memaksa seluruh terjemahan yang berjalan bersamaan (concurrent) untuk patuh pada 1 antrean
		globalAIMutex.Lock()
		elapsed := time.Since(lastAPIcall)
		
		var minDelay time.Duration
		if strings.Contains(modelName, "3.1-flash-lite") {
			minDelay = 11 * time.Second // Aman untuk ~5-6 RPM
		} else if strings.Contains(modelName, "2.5-flash") && !strings.Contains(modelName, "lite") {
			minDelay = 25 * time.Second // Aman batas Free Tier
		} else if strings.Contains(modelName, "pro") {
			minDelay = 45 * time.Second // Sangat mahal limitnya
		} else {
			minDelay = 12 * time.Second
		}

		if elapsed < minDelay {
			waitDur := minDelay - elapsed
			log.Printf("⏳ [Rate Limit Antrean] Menunda API Request selama %v detik (Model: %s)...\n", waitDur.Round(time.Second), modelName)
			time.Sleep(waitDur)
		}
		
		// Update timer terakhir ketika API benar-benar dipanggil
		lastAPIcall = time.Now()
		globalAIMutex.Unlock()

		resp, err := model.GenerateContent(ctx, genai.Text(chunk))

		// 🚀 RETRY MECHANISM: Jika terkena Error 429 (Rate Limit Quota)
		if err != nil {
			// Tangani RTO (Read TCP Timeout) atau Error Koneksi Jaringan
			// "A connection attempt failed..." / "wsarecv" / "timeout"
			if strings.Contains(strings.ToLower(err.Error()), "read tcp") || strings.Contains(err.Error(), "wsarecv") || strings.Contains(strings.ToLower(err.Error()), "timeout") || strings.Contains(strings.ToLower(err.Error()), "connection") {
				if retryCount >= 5 {
					log.Printf("❌  [AI Service] Max retries koneksi tercapai untuk chunk %d. Internet tidak stabil.\n", i+1)
					return strings.TrimSpace(translated.String()), int(i), totalTokens, fmt.Errorf("gagal terhubung ke API Gemini setelah 5 kali coba: %v", err)
				}

				log.Printf("⚠️  [AI Service] Koneksi terputus/Timeout pada chunk %d. Menunggu %d detik sebelum reconect...\n", i+1, 20*(retryCount+1))
				time.Sleep(time.Duration(20*(retryCount+1)) * time.Second)
				retryCount++
				i-- // Mundur ke chunk yang sama
				continue
			}

			// FALLBACK: Terjegat Sensor Bawaan Google (Prohibited Content)
			if strings.Contains(err.Error(), "blocked") || strings.Contains(err.Error(), "BlockReason") {
				log.Printf("⚠️  [AI Service] Chunk %d diblokir mentah-mentah oleh Filter Inti Gemini (BlockReason). Melompati chunk ini...\n", i+1)
				translated.WriteString("\n\n[⚠️ TERJEMAHAN DIBLOKIR OLEH SENSOR GOOGLE GEMINI (DILUAR KENDALI SISTEM). BERIKUT ADALAH TEKS ASLINYA:]\n\n")
				translated.WriteString(chunk) // Biarkan teks asli menyambung
				translated.WriteString("\n\n")

				// Langsung kabari backend agar DB ter-update dan lanjut ke chunk berikutnya
				if onChunkSuccess != nil {
					onChunkSuccess(strings.TrimSpace(translated.String()), totalTokens, i+1, len(chunks))
				}
				retryCount = 0
				time.Sleep(3 * time.Second) // Sedikit jeda napas
				continue
			}

			if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "Quota") || strings.Contains(strings.ToLower(err.Error()), "rate limit") {
				if retryCount >= 3 {
					log.Printf("❌  [AI Service] Max retries tercapai untuk chunk %d. Kemungkinan Daily Quota habis.\n", i+1)
					// Mengembalikan teks yang SUDAH DITERJEMAHKAN sejauh ini alih-alih membuang semuanya
					return strings.TrimSpace(translated.String()), int(i), totalTokens, fmt.Errorf("rate limit API Gemini membatalkan proses di chunk %d: %v", i+1, err)
				}

				// Exponential Backoff santai: 35s, lalu 65s, lalu 125s...
				sleepTime := time.Duration(35+(retryCount*30)) * time.Second
				log.Printf("⚠️  [AI Service] Hit Rate Limit (429) pada chunk %d. Menunggu %v sebelum retry...\n", i+1, sleepTime)

				time.Sleep(sleepTime)
				retryCount++
				i-- // Mundurkan index agar chunk ini diulang
				continue
			}
			log.Printf("❌  [AI Service] Error fatal pada chunk %d: %v\n", i+1, err)
			return strings.TrimSpace(translated.String()), int(i), totalTokens, err
		}

		retryCount = 0 // Reset setelah sukses, maju ke chunk selanjutnya

		// Menghitung jumlah token yang dipakai Gemini (Prompt + Candidate token)
		if resp.UsageMetadata != nil {
			totalTokens += int(resp.UsageMetadata.TotalTokenCount)
		}

		for _, part := range resp.Candidates[0].Content.Parts {
			if txt, ok := part.(genai.Text); ok {
				translated.WriteString(string(txt))
				translated.WriteString("\n\n")
			}
		}

		// BERHASIL 1 CHUNK! Langsung kabari backend agar DB diupdate "Live"
		if onChunkSuccess != nil {
			onChunkSuccess(strings.TrimSpace(translated.String()), totalTokens, i+1, len(chunks))
		}

		// Chunk berhasil dan status terupdate, siap maju ke chunk berikutnya (jika ada).
		// Note: Rate Limit sudah di-handle di awal iterasi menggunakan Global Mutex (globalAIMutex).
	}

	return strings.TrimSpace(translated.String()), len(chunks), totalTokens, nil
}
