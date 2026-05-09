# 🚀 OpenAPI-manager

**OpenAPI-manager adalah API gateway kompatibel OpenAI, regional relay, provider router, load balancer, dan pengelola trafik multimodal.**

Aplikasi cukup memakai satu Base URL, sementara OpenAPI-manager menangani provider API key, routing, retry, batas RPM/TPM, regional forwarding, dan kompatibilitas provider.

## 🌟 Kenapa Berguna

- 🌏 Pengguna bisa berada di China atau luar negeri; request ke provider dikirim dari lokasi server OpenAPI-manager.
- ⚖️ Load balancing banyak key dan channel membantu provider dengan limit ketat seperti NVIDIA, OpenRouter, Gemini, dan Groq.
- ⚡ Cloudflare Edge Proxy opsional untuk mempercepat streaming yang sensitif latency.
- 🧩 Provider global dan China-friendly bisa dikelola dalam satu layer.
- 🖼️ Relay untuk chat, vision, image generation, audio, dan video-task jika didukung provider.

```text
Client / App / Agent → OpenAPI-manager Server → Upstream Provider
```

## 🧩 Provider

Global: OpenAI, Claude, Gemini, Groq, OpenRouter, NVIDIA, Cohere, Mistral, xAI.

China-friendly: DeepSeek, Qwen / Tongyi, ERNIE / Wenxin Qianfan, Tencent Hunyuan, Doubao / Volcano Ark, Zhipu GLM, Kimi, Baichuan, MiniMax, iFlytek Spark, SiliconFlow, 360, 01.AI, StepFun.

## 🐳 Docker Hub

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/openapi-manager:/data heing8848/openapi-manager:latest
```

Buka `http://localhost:3000`.

Login pertama: `root` / `123456`. Segera ubah password.

## ⚡ Edge Proxy

Cloudflare Edge Proxy memungkinkan Worker menerima `/v1/*`, meminta routing dan key ke OpenAPI-manager, lalu streaming langsung dari edge ke provider.

Panduan: [cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 Atribusi dan Lisensi

OpenAPI-manager berbasis [one-api](https://github.com/songquanpeng/one-api). Terima kasih kepada author asli dan contributors. MIT License: [LICENSE](./LICENSE).
