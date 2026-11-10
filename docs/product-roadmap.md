# Tahapan penyempurnaan KlipForge

Disepakati: 10 November 2026. Branch: `feat/product-completion`. Semua pengerjaan lokal; tidak ada commit, push, atau deployment tanpa permintaan pengguna.

Awalnya setiap tahap memiliki checkpoint review. Pada 10 November 2026 pengguna menyetujui tahap 1 dan mengizinkan tahap 2 serta tahap 3 dilanjutkan tanpa review di tengah. API key YouTube belum tersedia; pengguna meminta integrasinya disiapkan dahulu.

## Tahap 1 - Panduan penggunaan dan demo

Status: Selesai dan disetujui pengguna.

- Dashboard menampilkan tindakan berikutnya sesuai peran dan data campaign/submission yang tersedia.
- Menu How it works menjelaskan fungsi Brand, Clipper, Admin, lima langkah alur produk, istilah, dan contoh perhitungan payout.
- Brand dapat membuka template Demo: Coffee launch. Membuka template tidak menyimpan data; Create draft menyimpan campaign privat, dan Publish tetap merupakan tindakan terpisah.
- UI menjelaskan metrik simulasi dan pembayaran manual.
- Validasi meliputi unit test aturan rekomendasi, lint, TypeScript, build, dan pengujian browser tiga peran/mobile.

## Tahap 2 - Perapihan alur kerja

Status: Implementasi lokal selesai; validasi akhir dicatat di phase-2-3-review.md.

Rapikan form campaign, pengiriman konten, moderasi, dan payout agar status, konsekuensi tindakan, dan langkah selanjutnya lebih mudah dipahami. Detail perubahan ditetapkan berdasarkan review tahap 1; tidak mengubah aturan bisnis tanpa alasan yang jelas.

## Tahap 3 - Integrasi metrik asli

Status: Integrasi read-only YouTube disiapkan dan diuji dengan API uji; aktivasi data asli menunggu API key.

Pilih satu platform sosial, verifikasi kemampuan dan akses API resmi yang tersedia, lalu implementasikan integrasi sesuai akses tersebut. Kebutuhan akun developer, izin, atau kredensial harus ditentukan sebelum menjanjikan data live. Pembayaran tetap manual pada ruang lingkup ini.

Lima langkah di halaman How it works adalah alur penggunaan produk, berbeda dari tiga tahap pengerjaan di dokumen ini.

## Hasil dan batas ruang lingkup tahap 2-3

Tahap 2 menambahkan dialog konfirmasi dengan alasan/referensi, penjelasan status, validasi tanggal/platform campaign, pencocokan platform URL, dan filter status review.

Tahap 3 menambahkan panel public metrics YouTube yang terpisah dari analytics simulasi dan payout. API key tetap di server, respons yang tidak menyediakan counter memakai nilai tidak tersedia, cache 15 menit, dan pembatasan request berlaku. Integrasi tidak membuktikan kepemilikan video dan tidak mengganti dasar payout.

Aktivasi data asli, uji dengan key/akun milik pengguna, kesiapan produksi/CI/deployment, verifikasi kepemilikan creator, dan pembayaran otomatis adalah pekerjaan lanjutan sesuai kebutuhan. Tidak ada push atau deploy pada tahap ini.
