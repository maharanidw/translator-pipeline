let extractedData = null;

// Fungsi untuk membuat estimasi jumlah potongan text yang akan diproses
function calculateEstimatedChunks(text) {
    const paragraphs = text.split('\n\n');
    let chunks = 0;
    let currentLen = 0;

    for (let p of paragraphs) {
        p = p.trim();
        if (!p) continue;
        if (currentLen + p.length > 4500) {
            if (currentLen > 0) {
                chunks++;
                currentLen = 0;
            }
        }
        currentLen += p.length + 2; // tambahan estimasi untuk \n\n
    }
    if (currentLen > 0) {
        chunks++;
    }
    return chunks;
}

// Event Listener dropdown agar form nama novel hilang-timbul
document.getElementById('novelSelect').addEventListener('change', (e) => {
    if (e.target.value === "0") {
        document.getElementById('novelOverrideTitle').style.display = "block";
    } else {
        document.getElementById('novelOverrideTitle').style.display = "none";
    }
});

// 1. Tombol Ekstrak Data dari Tab Browser
document.getElementById('extractBtn').addEventListener('click', async () => {
    const statusText = document.getElementById('status');
    statusText.innerText = "Membaca teks dari halaman...";
    
    // Reset state jika extract lagi
    document.getElementById('previewSection').style.display = "none";
    document.getElementById('translateBtn').disabled = false;
    document.getElementById('translateBtn').innerText = "2. Mulai Terjemahkan";
  
    let [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    
    // Auto-check jika sudah ada di database sebelum extract (Untuk fitur RESUME)
    try {
        const checkRes = await fetch("https://exctracta-47dzy.ondigitalocean.app/api/v1/status?url=" + encodeURIComponent(tab.url));
        const checkData = await checkRes.json();
        
        if (checkData.success && checkData.data) {
            const { CurrentChunk, TotalChunks, IsSynced } = checkData.data;
            if (IsSynced) {
                statusText.innerHTML = "✅ <b>Selesai 100%.</b> Cek <a href='https://exctracta-47dzy.ondigitalocean.app/dashboard' target='_blank'>Dashboard</a>";
                return; // Berhenti karena sudah tamat
            } else if (TotalChunks > 0 && CurrentChunk > 0) {
                const percent = Math.round((CurrentChunk / TotalChunks) * 100);
                // ADA YANG NYANGKUT! MUNCULKAN TOMBOL RESUME!
                statusText.innerHTML = `⚠️ <b>Proses Terhenti di ${percent}% (${CurrentChunk}/${TotalChunks})</b>`;
                document.getElementById('extractBtn').style.display = "none"; 
                document.getElementById('resumeSection').style.display = "block";
                
                // Ubah behavior tombol resume
                document.getElementById('resumeBtn').onclick = () => {
                    document.getElementById('resumeSection').style.display = "none";
                    document.getElementById('extractBtn').style.display = "block";
                    startExtractionAndSendBackend(tab); // Lanjut rutinitas normal (karena backend otomatis nge-resume dari DB)
                };
                return; // Jeda di sini sampai user pencet "Lanjutkan"
            }
        }
    } catch(e) {}
    
    // Jika aman / novel baru, lanjut ke prosedur Normal (Mulai Ekstraksi)
    startExtractionAndSendBackend(tab);
});

// Fungsi Ekstraksi Inti (dipisahkan agar bisa dipanggil oleh tombol Resume atau Awal)
async function startExtractionAndSendBackend(tab) {
    const statusText = document.getElementById('status');
    // Tarik daftar List Novel dari database agar user bisa pilih gabung ke mana
    try {
        const novelsRes = await fetch("https://exctracta-47dzy.ondigitalocean.app/api/v1/novels");
        const novelsData = await novelsRes.json();
        if (novelsData.success && novelsData.data) {
            const selectEl = document.getElementById('novelSelect');
            // Bersihkan sisa opsi lama kecuali opsi index 0
            selectEl.innerHTML = '<option value="0">✨ Buat Seri Novel Baru</option>';
            novelsData.data.forEach(nvl => {
                const opt = document.createElement('option');
                opt.value = nvl.ID;
                opt.innerText = nvl.Title;
                selectEl.appendChild(opt);
            });
        }
    } catch(err) {
        console.warn("Gagal menarik data novel dari backend. Pastikan Go Server menyala.");
    }

    // Auto-check jika sudah ada di database
    // (Dihapus dari sini karena sudah dipindah ke awal-awal logika)
  
    chrome.scripting.executeScript({
      target: { tabId: tab.id },
      files: ['content.js']
    }, () => {
      chrome.tabs.sendMessage(tab.id, { action: "EXTRACT_TEXT" }, (response) => {
        if (chrome.runtime.lastError) {
          statusText.innerText = "Gagal menghubungi page. Muat ulang (refresh) halaman sumber.";
          return;
        }

        if (response && response.success) {
            extractedData = response;
            
            // Proses kalkulasi kasar
            const chunkEstimate = calculateEstimatedChunks(response.text);
            const totalChars = response.text.length;
            
            // Tampilkan UI Preview
            document.getElementById('previewSection').style.display = "block";
            // Masukkan teks ke dalam textarea agar user bisa melihat dan mengeditnya
            document.getElementById('extractedTextPreview').value = response.text;
            
            // Masukkan title default dari web
            document.getElementById('novelOverrideTitle').value = response.title;
            // Gunakan Judul Tab as default chapter title, tapi kalau bisa di-edit
            document.getElementById('chapterOverrideTitle').value = response.title;

            document.getElementById('previewText').innerHTML = `
                <div style="margin-bottom: 5px;"><b>🔍 Tinjauan AI:</b></div>
                Teks diekstrak: <b>${totalChars.toLocaleString('id-ID')} Karakter</b><br/>
                Perkiraan Request: <b>~${chunkEstimate} Potong</b><br/><br/>
                <span style="font-size: 11.5px; color:#ef4444; border-top: 1px dotted #ccc; padding-top: 5px; display:inline-block;">
                  <b>RATE LIMIT INFO:</b><br/>
                  • Gunakan <b>Gemini 3.1 Flash Lite</b> (Kuota Tinggi: 500 Req/Hari | 15 Req/Menit).<br/>
                  • Hati-hati <b>Gemini 2.5 Flash</b> sangat terbatas (Hanya 20 Req/Hari | 5 Req/Menit).<br/>
                  • Pemrosesan akan di-jeda otomatis oleh server agar tidak over-limit.
                </span>
            `;
            statusText.innerText = "";
        } else {
          statusText.innerText = "Gagal mengekstrak teks DOM.";
        }
      });
    });
} // Akhir dari Fungsi startExtractionAndSendBackend

// 2. Tombol Trigger Menembak ke Backend REST API
document.getElementById('translateBtn').addEventListener('click', () => {
    if (!extractedData) return;
    
    // Ambil hasil teks final yang mungkin sudah diedit/dibersihkan oleh user di textarea
    const finalCleanText = document.getElementById('extractedTextPreview').value;
    
    const statusText = document.getElementById('status');
    const selectedModel = document.getElementById('aiModel').value;
    const sourceLang = document.getElementById('sourceLang').value;
    const targetLang = document.getElementById('targetLang').value;
    
    // Konfigurasi Title & Novel
    const selectedNovelID = parseInt(document.getElementById('novelSelect').value);
    const customNovelTitle = document.getElementById('novelOverrideTitle').value || extractedData.title;
    const customChapterTitle = document.getElementById('chapterOverrideTitle').value || "Bab Tanpa Judul";

    if (selectedNovelID === 0 && document.getElementById('novelOverrideTitle').value.trim() === "") {
        alert("Judul Novel Baru wajib diisi!");
        return;
    }

    statusText.innerText = "Laporan dikirim. Tunggu...";
    document.getElementById('translateBtn').disabled = true;
    document.getElementById('translateBtn').innerText = "Mengontak Backend...";

    fetch("https://exctracta-47dzy.ondigitalocean.app/api/v1/extract", { 
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
            novel_id: selectedNovelID,
            novel_title: customNovelTitle,
            chapter_title: customChapterTitle,
            url: extractedData.url,
            text: finalCleanText, // Kirim teks yang sudah dibersihkan
            ai_model: selectedModel,
            source_lang: sourceLang,
            target_lang: targetLang
        })
    })
    .then(res => res.json())
    .then(data => {
        if (data.success) {
            statusText.innerText = "✅ Perintah Eksekusi Diterima Server! Sedang memproses...";
            document.getElementById('translateBtn').innerText = "Sedang diproses AI (0%)";
            
            // Lakukan Polling Status Terjemahan di Popup!
            const pollInterval = setInterval(async () => {
                try {
                    const statusRes = await fetch("https://exctracta-47dzy.ondigitalocean.app/api/v1/status?url=" + encodeURIComponent(extractedData.url));
                    const statusData = await statusRes.json();
                    
                    if (statusData.success && statusData.data) {
                        const { CurrentChunk, TotalChunks, IsSynced } = statusData.data;
                        
                        if (TotalChunks > 0) {
                            const percent = Math.round((CurrentChunk / TotalChunks) * 100);
                            statusText.innerText = `🔄 Menerjemahkan... (${CurrentChunk}/${TotalChunks} bagian) ${percent}%`;
                            document.getElementById('translateBtn').innerText = `Memproses (${percent}%)`;
                        } else {
                            statusText.innerText = "🔄 Menginisialisasi...";
                        }
                        
                        if (IsSynced) {
                            clearInterval(pollInterval);
                            statusText.innerText = "🎉 Terjemahan Selesai! Silakan cek di Dashboard.";
                            document.getElementById('translateBtn').innerText = "Selesai!";
                        }
                    }
                } catch(e) {
                    console.log("Polling error (Abaikan jika server restart) :", e);
                }
            }, 3000); // Cek tiap 3 detik
            
        } else {
            statusText.innerText = "❌ Backend menolak request.";
            document.getElementById('translateBtn').disabled = false;
        }
    })
    .catch(err => {
        statusText.innerText = "Error: Hubungi backend gagal! Pastikan terminal Go nyala. " + err.message;
        document.getElementById('translateBtn').disabled = false;
        document.getElementById('translateBtn').innerText = "Coba Lagi";
        console.error(err);
    });
});
