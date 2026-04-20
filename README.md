# Job Aggregator

Backend internal tool aggregator lowongan kerja berbasis Go.

Project ini adalah backend job aggregation yang melakukan scrape lowongan dari beberapa source, menyimpannya ke PostgreSQL, lalu menyiapkan data tersebut untuk kebutuhan admin panel dan pipeline internal.

Saat ini implementasinya sudah mencakup:
- collector multi-source untuk `dealls` dan `glints`
- pipeline `collector -> normalizer -> deduplicator`
- penyimpanan raw scrape payload dan metrik histori run
- internal API untuk admin panel, worker trigger, dan monitoring

## Struktur Data Saat Ini

Schema database pada folder `migrations/` sudah disesuaikan dengan perubahan backend terbaru:
- `jobs` menyimpan field tambahan seperti `company_profile_image_url`, `work_type`, `employment_type`, dan `category`
- `scrape_run_metrics` menyimpan histori run untuk dashboard internal 24 jam dan recent runs
- `about_pages` disederhanakan hanya menjadi `id`, `title`, dan `body`

Karena migration di-load saat volume PostgreSQL masih baru, setiap perubahan schema perlu diikuti reset volume lokal:

```bash
make db-reset
make db-up
```

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

## Internal API Ringkas

Semua endpoint internal menggunakan format response berikut:

```json
{
  "api_message": "jobs fetched successfully",
  "count": 94,
  "data": []
}
```

Catatan:
- `count` adalah total data dari DAO/repository, jadi bisa dipakai frontend untuk pagination
- `data` bisa berupa object tunggal, list, atau `null`

Endpoint internal utama yang aktif saat ini:
- `GET /internal/about`
- `POST /internal/about`
- `GET /internal/about/{id}`
- `PATCH /internal/about/{id}`
- `DELETE /internal/about/{id}`
- `GET /internal/jobs`
- `GET /internal/jobs/categories`
- `GET /internal/jobs/{id}`
- `PATCH /internal/jobs/{id}`
- `POST /internal/jobs/{id}/approve`
- `POST /internal/jobs/{id}/reject`
- `POST /internal/jobs/{id}/archive`
- `GET /internal/sources`
- `POST /internal/worker/run`
- `GET /internal/worker/status`
- `GET /internal/worker/scrape-health`
- `GET /internal/worker/runs`

Filter list job yang tersedia saat ini:
- `status`
- `search`
- `category`
- `location`
- `work_type`
- `role_type`
- `sort=asc|desc`
- `source_id`

Contoh:

```bash
curl "http://localhost:8080/internal/jobs?location=jakarta&work_type=remote&sort=desc"
curl "http://localhost:8080/internal/jobs/categories"
curl "http://localhost:8080/internal/worker/scrape-health"
```
