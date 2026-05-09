# 🚀 OpenAPI-manager

**OpenAPI-manager ist ein OpenAI-kompatibles API-Gateway, regionaler Relay, Provider-Router, Load Balancer und Manager für multimodalen Traffic.**

Apps verwenden eine stabile Base URL, während OpenAPI-manager Provider-Keys, Routing, Retries, RPM/TPM-Limits, regionales Forwarding und Provider-Kompatibilität verwaltet.

## 🌟 Warum OpenAPI-manager

- 🌏 Nutzer können in China oder im Ausland sein; Provider-Anfragen verlassen die Region des OpenAPI-manager-Servers.
- ⚖️ Load Balancing über mehrere Keys und Channels hilft bei strengen Limits von NVIDIA, OpenRouter, Gemini und Groq.
- ⚡ Optionaler Cloudflare Edge Proxy beschleunigt latenzkritische Streaming-Routen.
- 🧩 Globale und China-freundliche Provider lassen sich in einer Ebene verwalten.
- 🖼️ Relay für Chat, Vision, Bildgenerierung, Audio und Video-Task-Provider, sofern unterstützt.

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

Öffnen Sie `http://localhost:3000`.

Erster Login: `root` / `123456`. Bitte sofort ändern.

## ⚡ Edge Proxy

Cloudflare Edge Proxy kann `/v1/*` in Workers empfangen, Routing und Key von OpenAPI-manager abrufen und direkt vom Edge zum Provider streamen.

Guide: [cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 Attribution und Lizenz

OpenAPI-manager basiert auf [one-api](https://github.com/songquanpeng/one-api). Danke an den ursprünglichen Autor und Contributors. MIT License: [LICENSE](./LICENSE).
