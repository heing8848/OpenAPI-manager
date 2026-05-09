# 🚀 OpenAPI-manager

**OpenAPI-manager es una pasarela API compatible con OpenAI, relay regional, router de providers, balanceador de carga y gestor de tráfico multimodal.**

Tus apps usan un único Base URL, mientras OpenAPI-manager gestiona claves, routing, reintentos, presión de límites RPM/TPM, forwarding regional y compatibilidad entre providers.

## 🌟 Por qué usarlo

- 🌏 Los usuarios pueden estar en China o en el extranjero; las peticiones al provider salen desde la región del servidor OpenAPI-manager.
- ⚖️ Balanceo entre múltiples keys y channels para providers con límites estrictos como NVIDIA, OpenRouter, Gemini y Groq.
- ⚡ Cloudflare Edge Proxy opcional para acelerar rutas streaming sensibles a latencia.
- 🧩 Unifica providers globales y providers compatibles con China.
- 🖼️ Relay para chat, vision, generación de imágenes, audio y tareas de video cuando el provider lo soporta.

```text
Client / App / Agent → OpenAPI-manager Server → Upstream Provider
```

## 🧩 Providers

Globales: OpenAI, Claude, Gemini, Groq, OpenRouter, NVIDIA, Cohere, Mistral, xAI.

China-friendly: DeepSeek, Qwen / Tongyi, ERNIE / Wenxin Qianfan, Tencent Hunyuan, Doubao / Volcano Ark, Zhipu GLM, Kimi, Baichuan, MiniMax, iFlytek Spark, SiliconFlow, 360, 01.AI, StepFun.

## 🐳 Docker Hub

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/openapi-manager:/data heing8848/openapi-manager:latest
```

Abre `http://localhost:3000`.

Primer login: `root` / `123456`. Cambia la contraseña inmediatamente.

## ⚡ Edge Proxy

Cloudflare Edge Proxy permite que Workers reciban `/v1/*`, consulten routing y key en OpenAPI-manager, y hagan streaming directo desde el edge al provider.

Guía: [cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 Créditos y licencia

OpenAPI-manager está basado en [one-api](https://github.com/songquanpeng/one-api). Gracias al autor original y contributors. Licencia MIT: [LICENSE](./LICENSE).
