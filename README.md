# 🚀 OpenAPI-manager

**One API gateway for global AI providers, regional relay, load balancing, Edge Proxy acceleration, and multimodal traffic.**

OpenAPI-manager gives apps and agents one OpenAI-compatible endpoint while the server handles provider keys, routing, retries, RPM/TPM pressure, regional forwarding, and provider compatibility.

## 🌍 Languages

- 🇺🇸 [English](./README.en.md)
- 🇨🇳 [简体中文](./README.zh-CN.md)
- 🇹🇼 [繁體中文](./README.zh-TW.md)
- 🇯🇵 [日本語](./README.ja.md)
- 🇰🇷 [한국어](./README.ko.md)
- 🇸🇦 [العربية](./README.ar.md)
- 🇹🇭 [ไทย](./README.th.md)
- 🇪🇸 [Español](./README.es.md)
- 🇧🇷 [Português do Brasil](./README.pt-BR.md)
- 🇫🇷 [Français](./README.fr.md)
- 🇩🇪 [Deutsch](./README.de.md)
- 🇮🇩 [Bahasa Indonesia](./README.id.md)
- 🇻🇳 [Tiếng Việt](./README.vi.md)

## ✨ Quick Picture

```text
Apps / Agents / Users
        ↓
OpenAPI-manager
        ↓
OpenAI / Claude / Gemini / Groq / OpenRouter / NVIDIA
DeepSeek / Qwen / ERNIE / Hunyuan / Doubao / GLM / Kimi / SiliconFlow / ...
```

## 🐳 Docker Hub

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager \
  -d \
  --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v /home/ubuntu/data/openapi-manager:/data \
  heing8848/openapi-manager:latest
```

Open `http://localhost:3000`.

Default first login: `root` / `123456`. Change it immediately.

## 🧭 Core Ideas

- 🌐 Users can be in China or overseas; provider requests leave from your OpenAPI-manager server region.
- ⚖️ Multi-key and multi-channel load balancing helps with strict RPM/TPM limits.
- ⚡ Optional Cloudflare Edge Proxy can speed up latency-sensitive streaming routes.
- 🧩 One OpenAI-compatible API surface can front many global and China-friendly providers.
- 🖼️ Multimodal relay paths cover chat, vision, image, audio, and video-task style providers where supported.

## 🙏 Attribution

OpenAPI-manager is based on [one-api](https://github.com/songquanpeng/one-api) by [songquanpeng](https://github.com/songquanpeng). Thanks to the original author and contributors for the foundation.

## 📄 License

MIT License. See [LICENSE](./LICENSE).
