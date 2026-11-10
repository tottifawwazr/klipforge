# Review tahap 1 - Panduan penggunaan dan demo

Tanggal: 10 November 2026. Branch `feat/product-completion`. Belum ada commit, push, atau deployment.

## Yang berubah pada tahap ini

- Overview memprioritaskan kartu Your next step sesuai peran dan data yang berhasil dimuat.
- Brand diarahkan ke draft yang tepat atau review campaign aktif; campaign yang tanggalnya berakhir mendapat arahan yang sesuai.
- Clipper diarahkan ke review yang masih berjalan, feedback penolakan, payout, atau penelusuran campaign sesuai data submission yang tersedia.
- Admin mendapat pintu masuk ke antrean moderasi.
- Menu How it works menjelaskan peran akun aktif, lima langkah alur penggunaan, glossary, contoh hitungan payout, metrik simulasi, dan pembayaran manual.
- Template Demo: Coffee launch menyediakan brief fiktif, requirement, contoh budget USD 100, CPM USD 10, maksimum USD 20 per clip, tiga platform, dan tanggal yang masih berlaku saat form dibuka.
- Membuka template tidak memanggil endpoint pembuatan campaign. Create draft tetap menyimpan draft privat melalui API yang sudah ada; Publish tetap merupakan tindakan terpisah.
- Browser Playwright bawaan digunakan agar pengujian tidak tergantung pada kecocokan versi Chrome lokal.

Tidak ada perubahan backend, skema database, provider metrik, atau aturan payout pada tahap ini. Perubahan tahap sebelumnya tetap dipertahankan.

## Validasi

- Prettier: lulus.
- ESLint: lulus tanpa warning.
- TypeScript: lulus.
- Vitest: 5 file / 19 test lulus, termasuk prioritas rekomendasi, batas tanggal, dan template lintas tahun.
- Build produksi Next.js: lulus.
- Browser responsive: panduan dan seluruh tab Brand, Clipper, Admin lulus pada 320, 390, 768, dan 1440px, tanpa exception browser.
- Browser workflow: template demo disimpan melalui aksi eksplisit, lalu campaign diedit/dipublikasikan, clip dikirim/ditinjau, payout dicatat, notifikasi dibaca, dan session refresh diverifikasi. Lulus setelah test memilih platform yang cocok dengan URL video pada template multi-platform.
- Screenshot dan hasil inspeksi manual tersedia di `.codex/phase-1/preview/` (diabaikan Git).

## Cara review

Buka http://localhost:3002/login. Masuk sebagai Brand (`brand@klipforge.local` / `Brand123!`), cek Your next step, buka How it works, lalu Create a practice draft. Tinjau isi form; kamu tidak perlu menekan Create draft untuk memeriksa template.

Coba juga How it works dari akun Clipper (`clipper@klipforge.local` / `Clipper123!`) atau Admin (`admin@klipforge.local` / `Admin123!`) untuk melihat penjelasan dan tombol sesuai peran. Kredensial ini hanya fixture lokal.

Panduan lebih rinci: [panduan-demo.md](panduan-demo.md). Batas pengerjaan dan approval: [product-roadmap.md](product-roadmap.md). Tahap 2 dan 3 belum dimulai.
