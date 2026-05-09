# AlfredAPI Cloudflare Edge Proxy

Edge Proxy is an optional Cloudflare Worker deployment for AlfredAPI. It receives OpenAI-compatible `/v1/*` requests, asks the AlfredAPI server which upstream channel and key should be used, then sends the request directly from Cloudflare to the upstream provider.

Use this only for channels where you accept the trade-off: lower latency and less traffic through the central AlfredAPI server, but the Worker becomes a privileged component that can see prepared upstream request metadata.

## How It Works

1. The client sends a normal OpenAI-compatible request to the Worker, for example `/v1/chat/completions`.
2. The Worker sends request metadata to the AlfredAPI server endpoint `/api/worker/prepare_v2`.
3. AlfredAPI authenticates the user token, checks quota, chooses a channel/key, and returns the prepared upstream request.
4. The Worker calls the upstream provider directly and streams the response back to the client.
5. After the response finishes, the Worker reports usage to `/api/worker/callback_v2`.

Non-relay paths such as `/api/*` and `/dashboard/*` are forwarded back to `ALFRED_API_HOST` without Edge Proxy routing.

## Required Settings

Configure these values in Cloudflare Worker variables/secrets. Do not commit real values to Git.

| Name | Type | Example | Notes |
| --- | --- | --- | --- |
| `ALFRED_API_HOST` | Variable | `https://<your-alfredapi-domain>` | Your AlfredAPI server origin. Do not include a trailing slash. |
| `WORKER_ADMIN_TOKEN` | Secret | `<alfredapi-admin-access-token>` | Create a dedicated AlfredAPI admin token for the Worker. Store it as a Cloudflare secret. |

`WORKER_ADMIN_TOKEN` is highly privileged because it is used for worker callback accounting. Rotate it immediately if it is exposed.

## Deploy From The Cloudflare Dashboard

1. Open Cloudflare Dashboard.
2. Go to Workers & Pages.
3. Create a Worker.
4. Replace the generated code with the contents of `worker.js`.
5. Open Settings -> Variables and Secrets.
6. Add `ALFRED_API_HOST` as a variable.
7. Add `WORKER_ADMIN_TOKEN` as a secret.
8. Deploy the Worker.
9. In AlfredAPI channel settings, enable Edge Proxy only for the channels that should use this mode.
10. In your client app, set the Base URL to the Worker URL, for example `https://<worker-name>.<your-subdomain>.workers.dev`.

The client still uses its normal AlfredAPI user token. Do not put upstream provider keys into the client.

## Deploy With Wrangler

Install Wrangler and log in:

```bash
npm install -g wrangler
wrangler login
```

Create a Worker project or copy `worker.js` into an existing Worker project, then set secrets:

```bash
wrangler secret put WORKER_ADMIN_TOKEN
```

Set `ALFRED_API_HOST` in `wrangler.toml` or the Cloudflare dashboard:

```toml
[vars]
ALFRED_API_HOST = "https://<your-alfredapi-domain>"
```

Deploy:

```bash
wrangler deploy
```

## Security Checklist

- Do not commit `.env`, real Worker variables, admin tokens, upstream provider API keys, database files, or logs.
- Use a dedicated AlfredAPI admin token for `WORKER_ADMIN_TOKEN`.
- Rotate `WORKER_ADMIN_TOKEN` after testing or when changing operators.
- Enable Edge Proxy only on channels that are safe to call from Cloudflare.
- Treat streaming interruptions as a possible source of incomplete usage callback data.
- Keep the Worker account trusted: the Worker receives the prepared upstream URL, headers, and body from AlfredAPI.

## Troubleshooting

- `500 Missing Edge Proxy configuration`: check `ALFRED_API_HOST` and `WORKER_ADMIN_TOKEN`.
- Normal dashboard/API paths do not use Edge Proxy routing; they should be proxied to `ALFRED_API_HOST`.
- If a channel does not support Edge Proxy, AlfredAPI can reject the prepare request and the Worker will fall back to the original origin request where possible.
