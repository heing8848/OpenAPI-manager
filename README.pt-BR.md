# 🚀 OpenAPI-manager

**OpenAPI-manager é um gateway de API compatível com OpenAI, relay regional, roteador de providers, balanceador de carga e gerenciador de tráfego multimodal.**

Suas aplicações usam um único Base URL, enquanto o OpenAPI-manager gerencia chaves, roteamento, retries, pressão de limites RPM/TPM, encaminhamento regional e compatibilidade entre providers.

## 🌟 Por que usar

- 🌏 Usuários podem estar na China ou no exterior; as chamadas ao provider saem da região do servidor OpenAPI-manager.
- ⚖️ Balanceamento entre múltiplas keys e channels para providers com limites rígidos, como NVIDIA, OpenRouter, Gemini e Groq.
- ⚡ Cloudflare Edge Proxy opcional para acelerar rotas de streaming sensíveis a latência.
- 🧩 Gerencia providers globais e China-friendly na mesma camada.
- 🖼️ Relay para chat, vision, geração de imagens, áudio e tarefas de vídeo quando suportado pelo provider.

```text
Client / App / Agent → OpenAPI-manager Server → Upstream Provider
```

## 🧩 Providers

Globais: OpenAI, Claude, Gemini, Groq, OpenRouter, NVIDIA, Cohere, Mistral, xAI.

China-friendly: DeepSeek, Qwen / Tongyi, ERNIE / Wenxin Qianfan, Tencent Hunyuan, Doubao / Volcano Ark, Zhipu GLM, Kimi, Baichuan, MiniMax, iFlytek Spark, SiliconFlow, 360, 01.AI, StepFun.

## 🐳 Docker Hub

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/openapi-manager:/data heing8848/openapi-manager:latest
```

Acesse `http://localhost:3000`.

Login inicial: `root` / `123456`. Altere a senha imediatamente.

## ⚡ Edge Proxy

Cloudflare Edge Proxy permite que Workers recebam `/v1/*`, consultem roteamento e key no OpenAPI-manager, e façam streaming direto do edge para o provider.

Guia: [cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 Créditos e licença

OpenAPI-manager é baseado em [one-api](https://github.com/songquanpeng/one-api). Obrigado ao autor original e contributors. Licença MIT: [LICENSE](./LICENSE).
