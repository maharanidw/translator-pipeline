// Script yang akan dieksekusi di konteks halaman web
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.action === "EXTRACT_TEXT") {
        let extractedText = "";

        // STRATEGI BYPASS: Kita cari elemen utamanya langsung
        // Postype dan situs similar sering membungkus konten dalam article/div khusus
        
        // Coba deteksi spesifik struktur halaman Postype
        const isPostype = window.location.href.includes('postype.com');
        let articleBody = null;

        if (isPostype) {
            // Artikel utama di Postype biasanya ada di dalam section id "post-content" atau class "post-content"
            articleBody = document.querySelector('#post-content, .post-content, .post-body');
        } else {
            // Fallback untuk situs fiksi lain (AO3, dll)
            articleBody = document.querySelector('.userstuff, article, .story-text, .post-body');
        }
        
        if (articleBody) {
            // Ekstrak teks hanya dari elemen <p> untuk menghindari metadata, tombol navigasi, 
            // author info, atau komentar
            const paragraphs = articleBody.querySelectorAll('p');
            
            // Opsional: Coba filter paragraf yang sepertinya adalah teks non-fiksi (seperti info donasi dll)
            paragraphs.forEach(p => {
                const text = p.innerText.trim();
                
                // Lewati jika teks kosong atau terlalu pendek (kadang merupakan pembatas)
                if (text === '') return;

                // Heuristic sederhana untuk menghapus teks promosi/navigasi yang ganjil
                const isNavText = text.includes('이전 포스트') || 
                                  text.includes('다음 포스트') || 
                                  text.includes('구독자') ||
                                  text.includes('댓글');
                
                if (!isNavText) {
                    extractedText += text + "\n\n";
                }
            });
        } else {
            // Fallback (misal di AO3 atau situs tidak dikenal)
            extractedText = document.body.innerText;
        }

        console.log("[Translator Pipeline] Teks terekstrak berukuran:", extractedText.length, "karakter.");
        sendResponse({ 
            success: true, 
            text: extractedText,
            url: window.location.href,
            title: document.title
        });
    }
    return true; // Agar async response bisa berjalan
});
