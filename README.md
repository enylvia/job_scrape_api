# Job Aggregator

Backend internal tool aggregator lowongan kerja berbasis Go.

This project is a backend-focused job scraping pipeline built with Go and PostgreSQL. It is designed to collect job listings from external sources, persist them into an internal database first, process them in a controlled pipeline, and prepare them for internal operations and future publishing flows.

Current implementation highlights:
- foundational backend structure for API, worker, config, database, logger, models, repositories, and SQL migrations
- collector engine with modular source scraper support
- preview command to validate scraping without a database connection
- worker flow ready to persist scraped jobs and raw source payloads into PostgreSQL

Planned next steps include normalization, deduplication, internal operational APIs, publishing integrations, and scheduling automation.

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
go run ./cmd/worker
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
