# Scan Collector

Small HTTP server primarily intended as a backend for the Android app [Binary Eye](https://github.com/markusfisch/BinaryEye). It accepts JSON records with `timestamp`, `content`, `format`, and `deviceId` fields, then appends them to daily CSV files. Other clients can use it too, as long as they send requests in the expected JSON format.

Open `http://localhost:8765/` to view saved daily sets and their record counts. Each set has a detail page with per-record and bulk copy buttons, plus a CSV download.

## Run locally

Copy `.env.example` to `.env` next to `compose.yaml`, then set a username and a long, unique password:

```sh
cp .env.example .env
```

The server refuses to start if either value is missing. Keep `.env` private. HTTP Basic authentication protects the dashboard, detail pages, CSV downloads, and record submission. Use HTTPS through a reverse proxy before exposing the service beyond a trusted local network, because plain HTTP does not encrypt credentials or records.

Start the server:

```sh
docker compose up -d
```

The server listens on port `8765`. CSV files are written to `./data/scan-YYYY-MM-DD.csv` and survive container recreation.

Send a record with:

```sh
curl --user your-username -X POST http://localhost:8765/ \
  -H 'Content-Type: application/json' \
  -d '{"timestamp":"2026-09-25T12:00:00Z","content":"sample scanned data","format":"QR_CODE","deviceId":"scanner-1"}'
```

`curl` prompts for the password. Set `PORT` to change the host port. The container writes inside `/data`, mapped to `./data` by Compose. CSV columns are `čas,obsah,formát,zařízení`.

## Pull the published image

The Compose file uses the published image `ghcr.io/cuchulain/scan-collector:latest` by default. To use another tag, set `IMAGE`:

```sh
IMAGE=ghcr.io/cuchulain/scan-collector:latest docker compose up -d
```

For a private package, authenticate Docker to `ghcr.io` with an account that has package read access before pulling.

## Publish

Pushes to `main`, including merges into `main`, run tests and publish multi-platform `linux/amd64` and `linux/arm64` images to `ghcr.io/cuchulain/scan-collector`, tagged `latest` and with the commit SHA. GitHub Actions uses the built-in `GITHUB_TOKEN`; ensure the repository permits Actions to write packages. The package visibility can be set to private in GitHub Packages.
