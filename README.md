# Job Aggregator

Backend internal tool aggregator lowongan kerja berbasis Go.

This project is a backend-focused job scraping pipeline built with Go and PostgreSQL. It is designed to collect job listings from external sources, persist them into an internal database first, process them in a controlled pipeline, and prepare them for internal operations and future publishing flows.

Current implementation highlights:
- foundational backend structure for API, worker, config, database, logger, models, repositories, and SQL migrations
- collector engine with modular source scraper support
- preview command to validate scraping without a database connection
- worker flow ready to persist scraped jobs and raw source payloads into PostgreSQL

Planned next steps include normalization, deduplication, internal operational APIs, publishing integrations, and scheduling automation.

## Menjalankan PostgreSQL Dengan Docker

1. Salin `.env.example` menjadi `.env`.
2. Ubah `DB_ENABLED=true` di `.env`.
3. Jalankan PostgreSQL:

```bash
make db-up
```

4. Jalankan migration:

```bash
make migrate
```

5. Seed source bawaan:

```bash
make seed-sources
```

6. Cek status container:

```bash
docker compose ps
```

7. Untuk setup penuh sekaligus migration + seed:

```bash
make db-setup
```

8. Untuk menghentikan PostgreSQL:

```bash
make db-down
```

9. Untuk menghentikan sekaligus menghapus volume data lokal:

```bash
make db-reset
```

`docker-compose.yml` memakai nilai `DB_PORT`, `DB_USER`, `DB_PASSWORD`, dan `DB_NAME` dari file `.env`, jadi config database aplikasi dan container tetap konsisten.
Untuk Docker lokal di Windows, gunakan `DB_HOST=127.0.0.1` agar koneksi aplikasi tidak nyasar ke service PostgreSQL lain yang mungkin terpasang lewat WSL atau host machine.
`make migrate` akan menjalankan semua SQL di folder `migrations/`, sedangkan `make seed-sources` akan membuat atau meng-update source default `dealls` dan `glints`.

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

## Menjalankan Worker Collector

1. Pastikan migration sudah dijalankan di PostgreSQL dan isi tabel `sources` dengan source aktif.
2. Untuk mode `browser`, install Playwright browser lebih dulu:

```bash
go install github.com/playwright-community/playwright-go/cmd/playwright@latest
playwright install chromium
```

3. Jalankan worker:

```bash
make worker
```

Untuk setup database lalu langsung menjalankan worker end-to-end:

```bash
make e2e-worker
```

## Test Run Tanpa Database

Untuk preview hasil scrape tanpa koneksi database:

```bash
go run ./cmd/collectorpreview
```

Opsional:

```bash
COLLECTOR_PREVIEW_PAGES=1
COLLECTOR_PREVIEW_PRINT_LIMIT=3
go run ./cmd/collectorpreview
```

Command ini akan menjalankan preview collector aktif, scrape detail page yang dibutuhkan, lalu print hasil beberapa job pertama ke terminal dalam format JSON.

Untuk memilih source tertentu saat preview:

```bash
COLLECTOR_PREVIEW_SOURCE=dealls
go run ./cmd/collectorpreview
```

```bash
COLLECTOR_PREVIEW_SOURCE=glints
COLLECTOR_PREVIEW_PAGES=1
COLLECTOR_PREVIEW_PRINT_LIMIT=2
go run ./cmd/collectorpreview
```

Catatan:
- `glints` memakai `browser` collector dan sekarang dijalankan headless secara default.
- Jika perlu override perilaku browser, gunakan `BROWSER_COLLECTOR_HEADLESS=true|false` atau `GLINTS_BROWSER_HEADLESS=true|false`.
