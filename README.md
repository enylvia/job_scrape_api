# Job Aggregator

Backend internal tool aggregator lowongan kerja berbasis Go.

This project is a backend-focused job scraping pipeline built with Go and PostgreSQL. It is designed to collect job listings from external sources, persist them into an internal database first, process them in a controlled pipeline, and prepare them for internal operations and future publishing flows.

Current implementation highlights:
- foundational backend structure for API, config, database, logger, models, repositories, and SQL migrations
- collector engine with modular source scraper support
- full pipeline flow ready to persist scraped jobs and raw source payloads into PostgreSQL
- internal API endpoints to trigger the pipeline and operate jobs or sources

Planned next steps include normalization, deduplication, internal operational APIs, publishing integrations, and scheduling automation.

## Menjalankan PostgreSQL Dengan Docker

1. Salin `.env.example` menjadi `.env`.
2. Ubah `DB_ENABLED=true` di `.env`.
3. Jalankan PostgreSQL:

```bash
make db-up
```

4. Cek status container:

```bash
docker compose ps
```

5. Untuk menghentikan PostgreSQL:

```bash
make db-down
```

6. Untuk menghentikan sekaligus menghapus volume data lokal:

```bash
make db-reset
```

`docker-compose.yml` memakai nilai `DB_PORT`, `DB_USER`, `DB_PASSWORD`, dan `DB_NAME` dari file `.env`, jadi config database aplikasi dan container tetap konsisten. Pada volume database yang masih baru, PostgreSQL akan otomatis menjalankan semua file SQL di folder `migrations/`, termasuk seed source default `dealls` dan `glints`.

Untuk Docker lokal di Windows, gunakan `DB_HOST=127.0.0.1` agar koneksi aplikasi tidak nyasar ke service PostgreSQL lain yang mungkin terpasang lewat WSL atau host machine.

Kalau Anda ingin mengulang init database dari nol setelah ada perubahan migration, jalankan:

```bash
make db-reset
make db-up
```

## Menjalankan API

1. Salin `.env.example` menjadi `.env` dan sesuaikan nilainya.
2. Jalankan server:

```bash
go run ./cmd/api
```

3. Cek health endpoint:

```bash
curl http://localhost:8080/health
```

Secara default `DB_ENABLED=false`, jadi API bisa dijalankan tanpa database saat bootstrap awal.

Saat API berhasil jalan, log startup akan menampilkan informasi seperti:
- `router initialized: health=/health internal=/internal/*`
- `http server is ready to accept requests on :8080`

Ini berguna untuk memastikan service sudah hidup saat dijalankan di VPS.

## Menjalankan Pipeline Lewat Internal API

Untuk mode `browser`, install Playwright browser lebih dulu:

```bash
go install github.com/playwright-community/playwright-go/cmd/playwright@latest
playwright install chromium
```

Setelah API hidup, pipeline bisa dijalankan lewat internal API:

```bash
curl -X POST http://localhost:8080/internal/worker/run
```

Status pipeline bisa dicek lewat:

```bash
curl http://localhost:8080/internal/worker/status
```

Karena trigger pipeline sekarang dilakukan lewat internal API, command `worker`, `collectorpreview`, dan `dbsetup` sudah tidak dipakai lagi sebagai entrypoint utama.
