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
- `admin_users` menyimpan akun admin panel dengan `password_hash`, status aktif, dan histori login terakhir

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

Untuk admin panel, set credential bootstrap dan token secret berikut di `.env`:

```bash
ADMIN_USERNAME=admin
ADMIN_PASSWORD=change-me-to-a-strong-password
ADMIN_TOKEN_SECRET=change-me-to-a-long-random-secret-at-least-64-characters-long
ADMIN_TOKEN_TTL=4h
```

Saat API start dengan database aktif dan tabel `admin_users` masih kosong, aplikasi akan membuat admin pertama dari `ADMIN_USERNAME` dan `ADMIN_PASSWORD`. Password disimpan sebagai bcrypt hash, bukan plain text. Setelah admin user sudah ada, nilai username/password bootstrap tidak akan menimpa data di database.

Saat `APP_ENV=production`, aplikasi akan menolak start jika:
- `ADMIN_PASSWORD` kurang dari 12 karakter
- `ADMIN_TOKEN_SECRET` kurang dari 64 karakter
- `ADMIN_TOKEN_TTL` lebih dari 8 jam
- `DB_SSLMODE=disable` saat database aktif

Untuk production, set CORS origin secara eksplisit sesuai domain frontend:

```bash
CORS_ALLOWED_ORIGINS=https://admin.example.com,https://www.example.com
CORS_ALLOWED_METHODS=GET,POST,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Authorization,Content-Type
CORS_MAX_AGE=600
```

Catatan production:
- `CORS_ALLOWED_ORIGINS` wajib diisi saat `APP_ENV=production`
- jangan gunakan `*` untuk production
- pisahkan origin admin panel dan end-user frontend sesuai domain deploy Anda

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
curl -X POST http://localhost:8080/internal/worker/run \
  -H "Authorization: Bearer <admin_token>"
```

Status pipeline bisa dicek lewat:

```bash
curl http://localhost:8080/internal/worker/status \
  -H "Authorization: Bearer <admin_token>"
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
- endpoint admin/mutasi `/internal/*` wajib `Authorization: Bearer <admin_token>`, kecuali `POST /internal/auth/login`
- public job endpoints tanpa token hanya mengembalikan job `approved` atau `published`
- public job response memakai DTO aman dan tidak mengekspos field internal seperti `content_hash`, `status`, `wordpress_post_id`, atau `telegram_sent`
- login admin memiliki rate limit dasar untuk menahan brute-force

Endpoint internal utama yang aktif saat ini:
- `POST /analytics/events`
- `GET /internal/analytics/summary`
- `POST /internal/auth/login`
- `GET /internal/auth/me`
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
- `limit`
- `offset`

Contoh:

```bash
curl -X POST "http://localhost:8080/internal/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"admin\",\"password\":\"change-me\"}"

curl "http://localhost:8080/internal/jobs?location=jakarta&work_type=remote&sort=desc" \
  -H "Authorization: Bearer <admin_token>"

curl "http://localhost:8080/internal/jobs/categories" \
  -H "Authorization: Bearer <admin_token>"

curl "http://localhost:8080/internal/worker/scrape-health" \
  -H "Authorization: Bearer <admin_token>"
```

## Analytics MVP

Frontend end-user bisa mengirim event analytics tanpa login:

```bash
curl -X POST "http://localhost:8080/analytics/events" \
  -H "Content-Type: application/json" \
  -d '{
    "event_name": "job_view",
    "visitor_id": "anonymous-visitor-id",
    "session_id": "browser-session-id",
    "path": "/jobs/123",
    "job_id": 123,
    "metadata": {
      "category": "Engineering",
      "work_type": "remote"
    }
  }'
```

Event yang didukung:
- `page_view`
- `job_view`
- `search_performed`
- `filter_used`
- `apply_clicked`
- `category_clicked`
- `newsletter_interest`

Admin dashboard bisa membaca summary:

```bash
curl "http://localhost:8080/internal/analytics/summary?limit=5" \
  -H "Authorization: Bearer <admin_token>"
```

Summary berisi:
- `visitors_today`
- `page_views_today`
- `job_views_today`
- `apply_clicks_today`
- `searches_today`
- `conversion_rate`
- `top_viewed_jobs`
- `top_search_keywords`
