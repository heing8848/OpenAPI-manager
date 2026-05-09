# 🚀 OpenAPI-manager

**OpenAPI-manager est une passerelle API compatible OpenAI, un relais régional, un routeur de providers, un équilibreur de charge et un gestionnaire de trafic multimodal.**

Vos applications utilisent un seul Base URL, tandis qu'OpenAPI-manager gère les clés provider, le routage, les retries, les limites RPM/TPM, le forwarding régional et la compatibilité entre providers.

## 🌟 Pourquoi l'utiliser

- 🌏 Les utilisateurs peuvent être en Chine ou à l'étranger ; les requêtes provider partent de la région du serveur OpenAPI-manager.
- ⚖️ Load balancing entre plusieurs clés et channels pour les providers à limites strictes comme NVIDIA, OpenRouter, Gemini et Groq.
- ⚡ Cloudflare Edge Proxy optionnel pour accélérer les routes streaming sensibles à la latence.
- 🧩 Gestion unifiée des providers globaux et China-friendly.
- 🖼️ Relay pour chat, vision, génération d'images, audio et tâches vidéo lorsque le provider le supporte.

```text
Client / App / Agent → OpenAPI-manager Server → Upstream Provider
```

## 🧩 Providers

Globaux : OpenAI, Claude, Gemini, Groq, OpenRouter, NVIDIA, Cohere, Mistral, xAI.

China-friendly : DeepSeek, Qwen / Tongyi, ERNIE / Wenxin Qianfan, Tencent Hunyuan, Doubao / Volcano Ark, Zhipu GLM, Kimi, Baichuan, MiniMax, iFlytek Spark, SiliconFlow, 360, 01.AI, StepFun.

## 🐳 Docker Hub

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/openapi-manager:/data heing8848/openapi-manager:latest
```

Ouvrez `http://localhost:3000`.

Premier login : `root` / `123456`. Changez le mot de passe immédiatement.

## ⚡ Edge Proxy

Cloudflare Edge Proxy permet aux Workers de recevoir `/v1/*`, de demander le routing et la key à OpenAPI-manager, puis de streamer directement depuis l'edge vers le provider.

Guide : [cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 Attribution et licence

OpenAPI-manager est basé sur [one-api](https://github.com/songquanpeng/one-api). Merci à l'auteur original et aux contributors. Licence MIT : [LICENSE](./LICENSE).
