# Mencoba KlipForge dari awal

KlipForge mengelola kerja sama campaign: Brand menyiapkan brief dan budget, Clipper membuat serta memublikasikan konten, kemudian Brand/Admin meninjau hasil dan mencatat payout.

## Mulai dari sini

1. Buka http://localhost:3002/login.
2. Masuk sebagai `brand@klipforge.local`, password `Brand123!`.
3. Buka menu **How it works** untuk membaca alur dan peran akun yang sedang dipakai.
4. Klik **Create a practice draft**. Form sudah diisi contoh campaign kopi fiktif dengan budget USD 100, rate USD 10 per 1.000 views, dan batas USD 20 per clip.
5. Periksa judul, brief, ketentuan, tanggal, dan platform. Membuka form ini belum membuat campaign.
6. Tekan **Create draft** jika ingin menyimpannya. Draft masih privat. Tekan **Publish** pada kartu campaign setelah siap agar dapat diikuti Clipper.

## Melihat peran lain

Keluar dari akun sebelumnya, lalu masuk sebagai `clipper@klipforge.local` dengan password `Clipper123!`. How it works akan menjelaskan tugas Clipper. Explore campaigns membuka daftar campaign aktif. Pilih campaign yang tanggalnya masih berlaku, baca brief, lalu Join campaign. Submit a clip menerima link langsung video yang kamu publikasikan; KlipForge tidak membuat atau mengunggah videonya.

Kembali sebagai Brand untuk membuka Submissions, memilih campaign, lalu meninjau clip. Dashboard Your next step membantu membuka draft atau review campaign yang tersedia. Jika tanggal campaign sudah lewat, instruksi mengingatkan untuk menyelesaikan review sebelum melakukan Complete.

Akun `admin@klipforge.local` dengan password `Admin123!` memiliki panduan untuk antrean review dan pencatatan payout. Untuk latihan payout, campaign harus Completed dan clip harus Approved. Tunggu pengumpulan metrik simulasi berikutnya, lalu gunakan Preview amount, reserve, approve, dan pencatatan pembayaran eksternal sesuai alur yang ada.

## Hal yang perlu dipahami

- Semua akun di atas adalah fixture pengembangan lokal.
- Views dan engagement pada analytics campaign masih simulasi. Panel YouTube public metrics terpisah dapat memuat counter asli jika API key dikonfigurasi dan ada submission YouTube yang sudah Approved.
- Payout adalah catatan pembayaran manual; tombolnya tidak mengirim uang.
- Contoh USD 10 per 1.000 views menghasilkan USD 10 untuk 1.000 views. Dengan batas USD 20 per clip, 3.000 views menghasilkan USD 20, selama aturan kelayakan dan sisa budget terpenuhi.
- Campaign demo yang kamu simpan memakai database lokal dan dapat muncul dalam daftar akun fixture terkait.

Scope saat ini adalah demo portfolio: alur campaign sampai payout manual dapat dicoba, tanpa transfer uang. Panduan tiga tahap sebelumnya ada di [product-roadmap.md](product-roadmap.md).
