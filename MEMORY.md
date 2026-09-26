# Scan Collector: project handoff

## Application
- Go HTTP server listens on port `8765` and accepts JSON POST records with `timestamp`, `content`, `format`, and `deviceId` fields.
- Records are appended to daily `scan-YYYY-MM-DD.csv` files in `/data`; Docker Compose maps `./data:/data`.
- `GET /` shows dated sets and record counts. `GET /sets/YYYY-MM-DD` displays records; adding `.csv` downloads the original CSV.
- A login page protects the web interface and CSV downloads. Session tokens are random, stored as hashes in `/data/sessions.sqlite3`, and survive container restarts. Scan submissions to `POST /` do not require authentication. Configure `AUTH_USER` and `AUTH_PASSWORD` in a local `.env` file.
- `SESSION_TTL` defaults to `8h`; the login page checkbox selects `SESSION_EXTENDED_TTL`, default `720h` (30 days). `COOKIE_SECURE=true` enables the cookie Secure flag behind HTTPS; default is `false` for local HTTP.
- The request shape is intended to work with Binary Eye's HTTP POST integration.

## Local workflow
- Copy `.env.example` to `.env`, then run `docker compose up -d`.
- The `.env` file and generated CSV data are excluded from Git.
- The container image is published by `.github/workflows/publish-image.yml` on pushes to `main`.
