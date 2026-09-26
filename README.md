# Scan Collector

Small HTTP server for collecting scanned data from Android applications. It accepts JSON records with `timestamp`, `content`, `format`, and `deviceId` fields, then appends them to daily CSV files. The request shape is compatible with Binary Eye's HTTP POST integration.

Open `http://localhost:8765/` to view saved daily sets and their record counts. Each set has a detail page with per-record and bulk copy buttons, plus a CSV download.

## Run locally

Create a `.env` file next to `compose.yaml` and set a username and a long, unique password:

```dotenv
AUTH_USER=your-username
AUTH_PASSWORD=replace-with-a-long-random-password
```

The server refuses to start if either value is missing. Keep `.env` private. HTTP Basic authentication protects the dashboard, detail pages, CSV downloads, and record submission. Use HTTPS through a reverse proxy before exposing the service beyond a trusted local network, because plain HTTP does not encrypt credentials or records.

Start the server:

```sh
docker compose up --build -d
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

After the GitHub Actions workflow has published the image, set `IMAGE` to the GHCR image name:

```sh
IMAGE=ghcr.io/OWNER/REPOSITORY:latest docker compose up -d
```

For a private package, authenticate Docker to `ghcr.io` with an account that has package read access before pulling.

## Publish

Pushes to `main`, including merges into `main`, run tests and publish multi-platform `linux/amd64` and `linux/arm64` images to `ghcr.io/OWNER/REPOSITORY`, tagged `latest` and with the commit SHA. GitHub Actions uses the built-in `GITHUB_TOKEN`; ensure the repository permits Actions to write packages. The package visibility can be set to private in GitHub Packages.
