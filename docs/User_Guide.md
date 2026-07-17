# User Guide / FAQ

## Cerita Kanak-kanak

Cerita Kanak-kanak boleh dibuka dari halaman utama melalui pautan **Cerita Kanak-kanak** atau terus melalui `/kids`.

### Cara guna

1. Masukkan nama watak utama.
2. Pilih umur antara `3` hingga `12`.
3. Pilih jenis cerita seperti **Misteri Dongeng**, **Sains Kanak-kanak**, atau **Pengembaraan Haiwan**.
4. Pilih peranan watak utama.
5. Tekan **Mula Cerita**.

### Panjang cerita ikut umur

- Umur `3-5`: cerita pendek dan mudah, sekitar `40-99` patah perkataan.
- Umur `6-8`: cerita sederhana, sekitar `100-200` patah perkataan.
- Umur `9-12`: cerita lebih panjang dan lebih kompleks, sekitar `200-300` patah perkataan.

Jika umur tidak diisi, sistem menggunakan umur lalai untuk mod kanak-kanak. Jika umur tidak sah, sistem menggunakan tier paling rendah supaya cerita kekal pendek dan mudah.

### Bahasa

Semua output Cerita Kanak-kanak mesti dalam Bahasa Malaysia standard. Sistem akan mengelakkan perkataan dan struktur ayat Indonesia seperti `gak`, `nggak`, `banget`, `gue`, `lu`, `ngapain`, `mau`, `uang`, `sepeda`, `apa kabar`, `rumah sakit`, `bego`, `jelek`, `dong`, dan `sih`.

## Pengekodan Warna Suku Kata (Syllable Color-Coding)

Bagi kanak-kanak berumur 4-5 tahun, teks cerita akan memaparkan suku kata dengan warna berselang-seli (Merah dan Hitam). Ciri ini membantu kanak-kanak membezakan suku kata dan menyokong kemahiran fonik (phonics) dalam proses belajar membaca.
Contoh: Perkataan "buku" akan dipaparkan sebagai **bu** (Merah) dan **ku** (Hitam).

*Nota Guru/Ibu Bapa: Ciri ini menyasarkan pra-pembaca (4-5 tahun) bagi memudahkan proses mengeja. Sila pastikan Mod Kanak-Kanak (`kids_mode`) dan Pengekodan Warna (`syllable_coloring`) diaktifkan dalam konfigurasi sistem.*

## Jika Cerita Gagal Dimuat

Jika API gagal, timeout, atau data cerita tidak lengkap:

1. Panel cerita akan memaparkan mesej ralat yang mesra pengguna.
2. Butang **Cuba Lagi** akan muncul.
3. Tekan **Cuba Lagi** untuk mencuba semula tanpa refresh halaman.
4. Butang tersebut dilumpuhkan selama 2 saat selepas ditekan untuk mengelakkan spam request.

Jika ralat berterusan selepas beberapa percubaan, semak sambungan internet atau hubungi pentadbir sistem.
