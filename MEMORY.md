# Scan Collector: project handoff

## Application
- Go HTTP server listens on port `8765` and accepts JSON POST records with `timestamp`, `content`, `format`, and `deviceId` fields.
- Records are appended to daily `scan-YYYY-MM-DD.csv` files in `/data`; Docker Compose maps `./data:/data`.
- `GET /` shows dated sets and record counts. `GET /sets/YYYY-MM-DD` displays records; adding `.csv` downloads the original CSV.
- HTTP Basic authentication is required for all routes. Configure `AUTH_USER` and `AUTH_PASSWORD` in a local `.env` file.
- The request shape is intended to work with Binary Eye's HTTP POST integration.

## Local workflow
- Run `docker compose up --build -d` after creating `.env`.
- The `.env` file and generated CSV data are excluded from Git.
- The container image is published by `.github/workflows/publish-image.yml` on pushes to `main`.
