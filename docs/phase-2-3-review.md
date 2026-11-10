# Review tahap 2 dan 3

Tanggal: 10 November 2026. Branch `feat/product-completion`. Pengguna mengizinkan kedua tahap dikerjakan tanpa checkpoint perantara. Belum ada commit, push, atau deployment.

## Tahap 2 - Alur kerja

- Dialog aksesibel menggantikan confirm/prompt browser pada partisipasi, lifecycle campaign, moderasi, dan payout. Escape/Cancel membatalkan sebelum penyimpanan; aksi berjalan mencegah pengiriman ganda; error tetap muncul di dialog.
- Review reason dan referensi pembayaran memiliki label, batas panjang, dan penjelasan yang jelas.
- Penjelasan status membantu membedakan draft, tanggal campaign, pending/flagged/approved/rejected, budget reserved, approved payout, dan pembayaran yang sudah dicatat.
- Campaign memeriksa urutan tanggal dan pilihan platform sebelum request API, mempertahankan input, memfokuskan field yang salah, dan menunjukkan contoh perhitungan. Backend tetap menentukan kelayakan dan nilai uang.
- URL video yang dikenali memilih platform yang cocok pada form pengiriman baru; validasi penuh tetap di backend.
- Filter Review status memudahkan menemukan Approved/Rejected setelah clip keluar dari antrean needs-review.

## Tahap 3 - YouTube

Integrasi public statistics YouTube sudah diimplementasikan sebagai panel read-only. Statistiknya terpisah dari analytics dan payout simulasi. Akses peran/ownership, cache, rate limit, app quota, error aman, dan provenance sudah ditangani. Counter yang tidak diberikan API tidak diubah menjadi nol.

Belum ada API key, sehingga belum ada aktivasi/pengambilan data dari YouTube sungguhan. UI menampilkan Not connected dan instruksi setup khusus Admin. Lihat [youtube-integration.md](youtube-integration.md).

## Validasi

- Prettier, ESLint, TypeScript dan build produksi: lulus.
- Frontend: 7 file / 29 unit test lulus.
- Backend: go vet dan seluruh go test -p 1 dengan PostgreSQL/Redis test terpisah lulus, termasuk provider HTTP uji dan isolasi cache/ownership/ledger.
- Browser build produksi: 2 workflow lulus (39,1 detik), mencakup tiga peran, 320-1440px, filter Approved, dialog payout, dan keamanan endpoint YouTube.
- Pemeriksaan manual browser: alasan kosong ditolak, Escape/Cancel menutup dialog, tidak ada mutasi review/payout saat membatalkan, tidak ada exception, dan status tanpa key eksplisit. Screenshot/hasil ada di .codex/phase-2-3/preview/ (diabaikan Git).
- YAML Compose valid. Docker build dan CI remote tidak dijalankan pada host ini.
- Tidak ada panggilan live Google dengan key pengguna dan tidak ada transfer uang nyata.

## Cara cek

Buka http://localhost:3002/login. Sebagai Brand, coba template campaign dengan tanggal akhir yang tidak valid atau tanpa platform; form menjelaskan kesalahan tanpa menyimpan. Sebagai Brand/Admin, buka Submissions, gunakan Review status, dan periksa dialog sebelum menyetujui keputusan. Sebagai Admin, buka Payouts untuk memeriksa rincian jumlah/recipient dan referensi pembayaran. Cancel tidak melakukan mutasi.

Di Analytics, periksa YouTube public metrics. Karena key belum tersedia, tampilannya harus Not connected. Detail setup hanya tersedia untuk Admin.

## Pekerjaan berikutnya

1. Sediakan API key dan uji satu video/akun sungguhan.
2. Jika ingin memakai metrik asli untuk payout, verifikasi kepemilikan creator/channel dan tetapkan periode/aturan perhitungan campaign lebih dahulu.
3. Uji dengan pengguna pilot, termasuk alur akun seperti pemulihan password/verifikasi email yang belum disediakan MVP ini.
4. Sebelum publik: jalankan CI, siapkan environment/DB produksi, hapus fixture publik dari deployment, tetapkan kebijakan trusted proxy/rate limit, backup, dan monitoring.
5. Integrasi transfer uang otomatis bersifat tahap lanjutan; pencatatan pembayaran manual tetap dapat dipakai pada pilot yang sesuai.
