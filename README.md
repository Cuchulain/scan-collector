# ISBN Collector

HTTP server accepting ISBN scan records as JSON and appending them to daily CSV files.

## Run locally

```sh
docker compose up --build -d
```

The server listens on port `8765`. CSV files are written to `./data/isbn-YYYY-MM-DD.csv` and survive container recreation.

Send a record with:

```sh
curl -X POST http://localhost:8765/ \
  -H 'Content-Type: application/json' \
  -d '{"timestamp":"2026-09-25T12:00:00Z","content":"9780306406157","format":"EAN-13","deviceId":"scanner-1"}'
```

Accepted JSON fields are `timestamp`, `content`, `format`, and `deviceId`. The CSV header is `čas,isbn,formát,zařízení`.

Set `PORT` to change the host port. The container writes inside `/data`, mapped to `./data` by Compose.

## Pull the published image

After the GitHub Actions workflow has published the image, set `IMAGE` to the repository's GHCR image name:

```sh
IMAGE=ghcr.io/OWNER/REPOSITORY:latest docker compose up -d
```

For a private package, authenticate Docker to `ghcr.io` with an account that has package read access before pulling.

## Publish

Pushes to `main`, including merges into `main`, run tests and publish multi-platform `linux/amd64` and `linux/arm64` images to `ghcr.io/OWNER/REPOSITORY`, tagged `latest` and with the commit SHA. GitHub Actions uses the built-in `GITHUB_TOKEN`; ensure the repository permits Actions to write packages. The package visibility can be set to private in GitHub Packages.