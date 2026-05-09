# OpenAPI-manager Open Source Deployment Guide

This guide explains how to run a fresh OpenAPI-manager checkout without relying on any private local files from the original development machine.

## 1. Prepare Environment Variables

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` and replace every placeholder:

```env
SESSION_SECRET=<generate-a-long-random-session-secret>
MYSQL_ROOT_PASSWORD=<generate-a-strong-root-password>
MYSQL_PASSWORD=<generate-a-strong-database-password>
SQL_DSN=oneapi:<same-as-MYSQL_PASSWORD>@tcp(db:3306)/one-api
```

Generate a strong session secret with one of these commands:

```bash
openssl rand -hex 32
```

or:

```bash
node -e "console.log(require('crypto').randomBytes(32).toString('hex'))"
```

Never commit `.env`.

## 2. Start With Docker Compose

```bash
docker compose up -d
```

The default compose file starts:

- OpenAPI-manager from `heing8848/openapi-manager:latest`
- OpenAPI-manager on `http://localhost:3000`
- Redis
- MySQL

Runtime data is created under `data/` and logs under `logs/`. These folders are ignored by Git and must not be published.

## 3. Use SQLite Instead Of MySQL

For a quick local test, leave `SQL_DSN` empty in `.env` and start only the app service with your preferred local setup. SQLite data will be stored under `data/`.

SQLite is convenient for local testing, but MySQL or PostgreSQL is recommended for real multi-user deployments.

## 4. Run The Docker Hub Image Directly

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/openapi-manager:/data heing8848/openapi-manager:latest
```

## 5. Configure Cloudflare Edge Proxy

Edge Proxy is optional. See:

```text
cloudflare-edge-proxy/README.md
```

Use placeholders in the repository and configure real values only in Cloudflare:

```text
ALFRED_API_HOST=https://<your-alfredapi-domain>
WORKER_ADMIN_TOKEN=<alfredapi-admin-access-token>
```

`WORKER_ADMIN_TOKEN` must be stored as a Cloudflare secret. Do not put it in source code, screenshots, public docs, or issue reports.

## 6. Before Publishing A Release

Run these checks from the repository root:

```bash
git status --short
rg -n -a "sk-[A-Za-z0-9_-]{8,}|sk-or-v1-|nvapi-|AIza|AKIA|Bearer\\s+[A-Za-z0-9._-]{10,}|http://[0-9]{1,3}(\\.[0-9]{1,3}){3}" . --glob "!data/**" --glob "!logs/**" --glob "!.git/**"
```

Confirm that these local-only paths are absent from the release artifact:

```text
data/
logs/
.env
*.db
*.pem
```

If any real token, private URL, database, or log file appears, stop and remove it before publishing.
