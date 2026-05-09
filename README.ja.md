# 🚀 OpenAPI-manager

**OpenAPI-manager は、OpenAI 互換 API のためのゲートウェイ、リージョナルリレー、Provider ルーター、ロードバランサー、マルチモーダル中継基盤です。**

アプリや Agent は 1 つの Base URL だけを使い、OpenAPI-manager が provider の API Key、ルーティング、リトライ、RPM/TPM 制限、地域転送、互換性処理を裏側で管理します。

## 🌟 主な価値

- 🌏 ユーザーが中国・海外のどこにいても、上流 provider へのリクエストは OpenAPI-manager サーバーの所在地から送信されます。
- ⚖️ 複数 Key / 複数 Channel の負荷分散で、NVIDIA、OpenRouter、Gemini、Groq などの無料枠や低制限 provider を活用しやすくします。
- ⚡ Cloudflare Edge Proxy により、遅延に敏感なストリーミング経路を高速化できます。
- 🧩 海外向け provider と中国向け provider を 1 つの管理レイヤーで扱えます。
- 🖼️ Chat、Vision、画像生成、音声、動画タスク系の中継に対応します。実際の可用性は上流 provider と model に依存します。

```text
Client / App / Agent → OpenAPI-manager Server → Upstream Provider
```

## 🧩 Provider 例

グローバル provider：OpenAI、Claude、Gemini、Groq、OpenRouter、NVIDIA、Cohere、Mistral、xAI。

中国・中国フレンドリー provider：DeepSeek、Qwen / Tongyi、ERNIE / Wenxin Qianfan、Tencent Hunyuan、Doubao / Volcano Ark、Zhipu GLM、Kimi、Baichuan、MiniMax、iFlytek Spark、SiliconFlow、360、01.AI、StepFun。

## 🐳 Docker Hub

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/openapi-manager:/data heing8848/openapi-manager:latest
```

`http://localhost:3000` を開いてください。

初回ログイン：`root` / `123456`。ログイン後すぐに変更してください。

## ⚡ Edge Proxy

Cloudflare Edge Proxy は `/v1/*` リクエストを Worker で受け取り、OpenAPI-manager からルーティングと Key を取得し、Edge から上流 provider へ直接ストリーミングできます。

ガイド：[cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 Attribution / License

OpenAPI-manager は [one-api](https://github.com/songquanpeng/one-api) をベースにしています。元作者と contributors に感謝します。MIT License、詳細は [LICENSE](./LICENSE) を参照してください。
