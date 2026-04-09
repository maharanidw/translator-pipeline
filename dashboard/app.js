const API_BASE = "http://localhost:8888/api/v1";

document.addEventListener("DOMContentLoaded", () => {
    fetchNovels();
});

// View Manager
function showView(viewId) {
    document.getElementById("view-novels").style.display = "none";
    document.getElementById("view-chapters").style.display = "none";
    document.getElementById("view-reading").style.display = "none";
    
    // Check if view-logs elements exist (because I added them dynamically to standard DOM)
    let viewLogs = document.getElementById("view-logs");
    if(viewLogs) viewLogs.style.display = "none";

    document.getElementById(viewId).style.display = "block";
}

function showNovels() {
    stopLogPolling(); // Hentikan live update log kalau pindah tab
    showView("view-novels");
    fetchNovels(); // Auto refresh
}

// ======================== LIVE STATUS / LOGS API ========================
let logPollingInterval;

function showSystemLogs() {
    showView("view-logs");
    fetchLogs();
    // Refresh otomatis tiap 3 detik
    stopLogPolling(); 
    logPollingInterval = setInterval(fetchLogs, 3000); 
}

function stopLogPolling() {
    if (logPollingInterval) {
        clearInterval(logPollingInterval);
        logPollingInterval = null;
    }
}

async function fetchLogs() {
    const container = document.getElementById('log-container');
    if (!container) return; // fail safe

    try {
        const response = await fetch(`${API_BASE}/logs`);
        const result = await response.json();

        if (result.success) {
            container.innerText = result.data;
            // Auto scroll ke paling bawah jika konten bertambah
            container.scrollTop = container.scrollHeight;
        } else {
            container.innerText = "Error: " + result.error;
        }
    } catch (e) {
        container.innerText = "Gagal menghubungi server untuk menarik logs...";
    }
}

async function clearLogs() {
    const confirmDelete = confirm("Hapus semua isi log agar server lebih ringan? Log lama akan dibersihkan secara permanen.");
    if (!confirmDelete) return;

    try {
        const response = await fetch(`${API_BASE}/logs`, { method: 'DELETE' });
        const result = await response.json();
        if (result.success) {
            fetchLogs(); // Minta baris kosong lagi ke backend
        } else {
            alert("Gagal menghapus log: " + result.error);
        }
    } catch (e) {
        alert("Terjadi kesalahan jaringan.");
    }
}

// ======================== API CALLS ========================

// Mengambil Daftar Tabel "Novel"
async function fetchNovels() {
    const list = document.getElementById("novels-list");
    list.innerHTML = `<div class="loader">Sedang memuat kumpulan novel dari database...</div>`;

    try {
        const response = await fetch(`${API_BASE}/novels`);
        const result = await response.json();

        list.innerHTML = ""; // Clear loader

        if (!result.success || result.data.length === 0) {
            list.innerHTML = "<em>Belum ada Novel yang tersimpan. Coba ekstrak sesuatu dari Extension dulu!</em>";
            return;
        }

        result.data.forEach((novel) => {
            const item = document.createElement("div");
            item.className = "list-item";
            item.innerHTML = `
                <div style="display:flex; justify-content: space-between; align-items:flex-start;">
                    <div>
                        <strong>${novel.Title || 'Judul Tidak Diketahui'}</strong>
                        <div style="font-size: 14px; margin-top: 5px;">
                            <a href="${novel.SourceURL}" target="_blank" style="color:var(--text-color);">${novel.SourceURL}</a>
                        </div>
                    </div>
                    <button class="delete-btn" onclick="deleteNovel(event, ${novel.ID}, '${novel.Title.replace(/'/g, "\\'")}')">Hapus 🗑️</button>
                </div>
            `;
            item.onclick = (e) => {
                // Ignore the click if clicking directly on the link or delete button
                if(e.target.tagName.toLowerCase() !== 'a' && e.target.tagName.toLowerCase() !== 'button') {
                    fetchChapters(novel.ID, novel.Title);
                }
            };

            list.appendChild(item);
        });
    } catch (e) {
        list.innerHTML = `<div class="loader" style="color:red;">Error: Gagal memuat data dari server!</div>`;
    }
}

// Menghapus Data Novel + Chapters terkait (Fitur "Sisa Testing")
async function deleteNovel(event, novelId, title) {
    if (event) {
        event.stopPropagation(); // Biar list-item parent gak terpanggil onClick-nya
    }

    const confirmDelete = confirm(`Apakah Anda yakin ingin menghapus data novel "${title}" beserta semua chapternya secara permanen?`);
    if (!confirmDelete) return;

    try {
        const response = await fetch(`${API_BASE}/novels/${novelId}`, {
            method: 'DELETE'
        });
        const result = await response.json();

        if (result.success) {
            alert(result.message);
            fetchNovels(); // Refresh otomatis UI-nya
        } else {
            alert(`Gagal Menghapus: ${result.error}`);
        }
    } catch (e) {
        alert("Terjadi kesalahan jaringan saat mencoba menghapus novel: " + e.message);
    }
}

// Mengambil Semua "Chapters" dalam "Novel"
async function fetchChapters(novelId, title) {
    document.getElementById("novel-title-header").innerText = title;
    
    const list = document.getElementById("chapters-list");
    list.innerHTML = `<div class="loader">Memuat daftar chapter...</div>`;
    
    showView("view-chapters");

    try {
        const response = await fetch(`${API_BASE}/novels/${novelId}/chapters`);
        const result = await response.json();

        list.innerHTML = "";

        if (!result.success || result.data.length === 0) {
            list.innerHTML = "<em>Belum ada bab/chapter untuk novel ini.</em>";
            return;
        }

        result.data.forEach((ch) => {
            const item = document.createElement("div");
            item.className = "list-item";
            
            const total = ch.TotalChunks || Math.max(1, ch.CurrentChunk); // Hindari div by 0
            const percent = ch.TotalChunks > 0 ? Math.round((ch.CurrentChunk / ch.TotalChunks) * 100) : 0;
            
            const isPartiallyDone = (ch.CurrentChunk > 0 && ch.CurrentChunk < ch.TotalChunks && !ch.IsSynced);
            const isTranslating = (!ch.IsSynced && ch.CurrentChunk >= 0);
            
            let badgeClass = ch.IsSynced ? "status-sync" : isPartiallyDone ? "status-partial" : "status-pending";
            let badgeText = ch.IsSynced ? "Selesai 100%" : isPartiallyDone ? `Terpotong (${percent}%)` : `Proses: ${percent}%`;

            item.innerHTML = `
                📄 <strong>Chapter Teratas: ${ch.Title}</strong>
                <span class="badge ${badgeClass}">${badgeText}</span>
                <div style="font-size: 12px; margin-top:5px; color: gray;">
                   [${ch.AIModelUsed || "Proses Background"}] - Potongan: ${ch.CurrentChunk} / ${ch.TotalChunks}
                </div>
                <!-- Progress Bar -->
                <div style="width: 100%; background: #ddd; height: 5px; border-radius: 3px; margin-top: 8px;">
                    <div style="width: ${percent}%; background: ${ch.IsSynced ? '#4caf50' : '#3b82f6'}; height: 100%; border-radius: 3px; transition: width 0.3s;"></div>
                </div>
            `;
            
            item.onclick = async () => {
                if (ch.CurrentChunk > 0 || ch.IsSynced) {
                    if (!ch.IsSynced) {
                        alert(`Peringatan: Chapter ini belum selesai 100% (baru ${percent}%). Anda akan membaca teks sejauh yang sudah berhasil diproses saja.`);
                    }
                    await readChapter(ch.ID, novelId, title); // Load Reading Dashboard!
                } else {
                    alert("Chapter ini baru saja dimulai dan belum ada teks yang berhasil diterjemahkan. Tunggu sebentar lagi ya.");
                }
            };

            list.appendChild(item);
        });

    } catch (e) {
        list.innerHTML = `<div class="loader" style="color:red;">Error gagal memuat chapters...</div>`;
    }
}

// Mode "DASHBOARD MEMBACA" Penuh E-Ink
async function readChapter(chapterId, novelId, novelTitle) {
    const loadingScreen = document.getElementById("reading-content");
    
    showView("view-reading");
    loadingScreen.innerHTML = `<div class="loader">Sedang menarik teks yang panjang ini...</div>`;
    document.getElementById("reading-title").innerText = "Loading...";

    try {
        const response = await fetch(`${API_BASE}/chapters/${chapterId}`);
        const result = await response.json();

        if (!result.success) {
            loadingScreen.innerHTML = "<em>Buku ini nggak ketemu... Error 404.</em>";
            return;
        }

        const ch = result.data;
        document.getElementById("reading-title").innerText = ch.Title || "No Title";
        document.getElementById("reading-model").innerText = ch.AIModelUsed;
        
        loadingScreen.innerHTML = ch.TranslatedText;

        // Customise Back button
        const backBtn = document.getElementById("btn-back-chapter");
        backBtn.onclick = () => fetchChapters(novelId, novelTitle);
        window.scrollTo(0, 0); // Balik ke posisi atas di tab itu
        
    } catch (e) {
        loadingScreen.innerHTML = `<div class="loader" style="color:red;">Failed network. Pastikan Golang-mu on.</div>`;
    }
}