# 🚀 OpenAPI-manager

**OpenAPI-manager 是一個相容 OpenAI API 格式的 AI API 閘道、中轉轉發層、Provider 路由器、負載均衡器與多模態流量管理平台。**

它讓應用程式只需要接入一個穩定的 Base URL，背後由 OpenAPI-manager 統一處理 provider API Key、渠道選擇、重試、RPM/TPM 限制、多區域轉發和 provider 相容性。

## 🌟 核心價值

- 🌏 使用者無論在中國或海外，上游 provider 請求都可以從 OpenAPI-manager 伺服器所在地發出。
- ⚖️ 多 Key / 多渠道負載均衡，適合 NVIDIA、OpenRouter、Gemini、Groq 等免費層或低限額 provider。
- ⚡ 可選 Cloudflare Edge Proxy，加速延遲敏感的串流回應。
- 🧩 同時管理海外 only provider 與中國友善 provider。
- 🖼️ 支援聊天、Vision、圖片生成、音訊與影片任務類中轉能力，具體取決於上游模型。

```text
客戶端 / App / Agent → OpenAPI-manager 伺服器 → 上游 Provider
```

## 🧩 Provider 範例

全球 provider：OpenAI、Claude、Gemini、Groq、OpenRouter、NVIDIA、Cohere、Mistral、xAI。

中國與中國友善 provider：DeepSeek、通義千問、文心千帆、騰訊混元、豆包/火山方舟、智譜 GLM、Kimi、百川、MiniMax、訊飛星火、矽基流動、360、零一萬物、階躍星辰。

## 🐳 Docker Hub

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/openapi-manager:/data heing8848/openapi-manager:latest
```

開啟 `http://localhost:3000`。

首次登入：`root` / `123456`。請立即修改預設密碼。

## ⚡ Edge Proxy

Cloudflare Edge Proxy 可讓 Worker 接收 `/v1/*` 請求，向 OpenAPI-manager 取得路由與 Key，然後直接從邊緣節點串流到上游 provider。

完整教學：[cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 致謝與授權

OpenAPI-manager 基於 [one-api](https://github.com/songquanpeng/one-api) 繼續開發。感謝原作者與貢獻者。本專案使用 MIT License，詳見 [LICENSE](./LICENSE)。
