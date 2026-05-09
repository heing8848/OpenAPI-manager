# 🚀 OpenAPI-manager

**OpenAPI-manager is an OpenAI-compatible API gateway, regional relay, provider router, load balancer, and multimodal traffic manager.**

It gives your apps one stable Base URL while the server handles provider keys, channel selection, retries, usage control, RPM/TPM pressure, and provider compatibility.

> Built on top of [one-api](https://github.com/songquanpeng/one-api), with extra focus on multi-key routing, free-tier provider pooling, regional forwarding, Edge Proxy acceleration, and multimodal compatibility.

## 🌟 Why It Matters

AI provider access is fragmented:

- 🌏 Users may be in China, overseas, or both.
- 🧱 Some providers are overseas-only or unstable from certain networks.
- 🇨🇳 Some China-friendly providers work better from China or nearby regions.
- ⏱️ Free and low-cost tiers often have strict RPM and TPM limits.
- 🔑 One key is rarely enough for a team, an agent platform, or bursty workloads.
- 🧩 Providers differ in request/response shape even when they claim OpenAI compatibility.

OpenAPI-manager gives you a controlled relay point:

```text
Client / App / Agent
        ↓
OpenAPI-manager server
        ↓
Selected upstream provider
```

No matter where the request starts, the upstream provider request is sent from the OpenAPI-manager server location. You can choose a server region that best fits your provider mix, keep provider API keys server-side, and expose one OpenAI-compatible endpoint to all clients.

## ⚖️ Load Balancing For Tight RPM/TPM Providers

Free-tier and low-limit providers are useful, but their capacity is often split across many small quotas. OpenAPI-manager helps turn that fragmented capacity into one managed pool.

Good fit for providers and platforms such as:

- NVIDIA
- OpenRouter
- Google Gemini
- Groq
- OpenAI-compatible free or low-cost providers
- Self-hosted or private OpenAI-compatible endpoints

What OpenAPI-manager can help with:

- 🔁 Spread requests across multiple API keys under the same channel.
- 🧭 Route traffic across multiple channels and providers.
- 🧯 Retry or fall back when a channel is exhausted, unhealthy, or temporarily failing.
- 📊 Centralize usage logs, quotas, billing weights, and channel health.
- 🧑‍💻 Keep client apps simple: one Base URL and one user token.

## 🧭 Regional Relay + Edge Proxy

### Server-side relay

OpenAPI-manager works as a server-side relay:

```text
User in China      ┐
User overseas      ├──> OpenAPI-manager server ──> Provider
Agent / backend    ┘
```

Benefits:

- 🌍 Users in different regions can share the same API endpoint.
- 🔐 Provider API keys stay on the server instead of client devices.
- 🧭 Provider requests leave from your chosen server region.
- 🧩 You can mix overseas-only providers and China-friendly providers in one management layer.
- 🧾 Logs, quotas, retries, and routing decisions are centralized.

### ⚡ Cloudflare Edge Proxy

For latency-sensitive paths, Edge Proxy lets Cloudflare Workers receive `/v1/*` requests, ask OpenAPI-manager for the selected channel/key, then stream directly from the edge to the upstream provider.

This can reduce first-token latency and avoid pushing every streamed byte through the central server, especially when Cloudflare has a better cross-region route.

Guide:

[cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🧩 Provider Compatibility

OpenAPI-manager focuses on one client-facing OpenAI-compatible API surface while supporting many provider backends.

### Global providers

- OpenAI
- Anthropic Claude
- Google Gemini
- Groq
- OpenRouter
- NVIDIA
- Cohere
- Mistral
- Together AI
- xAI
- Other OpenAI-compatible endpoints

### China and China-friendly providers

- DeepSeek
- Alibaba Qwen / Tongyi / DashScope
- Baidu ERNIE / Wenxin Qianfan
- Tencent Hunyuan
- ByteDance Doubao / Volcano Engine Ark
- Zhipu GLM / ChatGLM
- Moonshot / Kimi
- Baichuan
- MiniMax
- iFlytek Spark
- SiliconFlow
- 360 AI
- 01.AI / Yi
- StepFun

Provider support depends on the model, upstream API, and channel configuration. OpenAPI-manager's job is to normalize routing, request handling, and operational control as much as possible.

## 🖼️ Multimodal Support

Where supported by upstream providers, OpenAPI-manager includes relay support for:

- 💬 Streaming and non-streaming chat completions
- 👁️ Vision/image-capable chat requests
- 🎨 Image generation response handling
- 🔊 Audio-related routes
- 🎬 Video-task style providers

Multimodal behavior still depends on the selected provider and model.

## 🐳 Docker Hub

Public image:

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
```

Quick start with SQLite:

```bash
docker run --name openapi-manager \
  -d \
  --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v /home/ubuntu/data/openapi-manager:/data \
  heing8848/openapi-manager:latest
```

Open:

```text
http://localhost:3000
```

Default first login:

```text
username: root
password: 123456
```

Change the default password immediately after first login.

## 🧱 Docker Compose

Copy the environment template:

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

Start:

```bash
docker compose up -d
```

More deployment notes:

[docs/open-source-deployment.md](./docs/open-source-deployment.md)

## 🔐 Security Checklist

Never publish:

```text
.env
data/
logs/
*.db
*.pem
*.key
*.crt
```

Run a quick scan before release:

```bash
rg -n -a "sk-[A-Za-z0-9_-]{8,}|sk-or-v1-|nvapi-|AIza|AKIA|Bearer\\s+[A-Za-z0-9._-]{10,}|http://[0-9]{1,3}(\\.[0-9]{1,3}){3}" . --glob "!data/**" --glob "!logs/**" --glob "!.git/**"
```

## 🙏 Attribution

OpenAPI-manager is based on [one-api](https://github.com/songquanpeng/one-api) by [songquanpeng](https://github.com/songquanpeng).

Thanks to the original one-api author and contributors for building the foundation this project is based on.

## 📄 License

MIT License. See [LICENSE](./LICENSE).
