# 🚀 OpenAPI-manager

**OpenAPI-manager là API gateway tương thích OpenAI, regional relay, provider router, load balancer và nền tảng quản lý lưu lượng multimodal.**

Ứng dụng chỉ cần dùng một Base URL, còn OpenAPI-manager xử lý provider API key, routing, retry, giới hạn RPM/TPM, regional forwarding và tương thích provider.

## 🌟 Giá trị chính

- 🌏 Người dùng có thể ở Trung Quốc hoặc nước ngoài; request tới provider được gửi từ vị trí server OpenAPI-manager.
- ⚖️ Load balancing nhiều key và channel giúp tận dụng provider có giới hạn chặt như NVIDIA, OpenRouter, Gemini và Groq.
- ⚡ Cloudflare Edge Proxy tùy chọn để tăng tốc các luồng streaming nhạy cảm với latency.
- 🧩 Quản lý provider global và China-friendly trong cùng một lớp.
- 🖼️ Relay cho chat, vision, image generation, audio và video-task nếu provider hỗ trợ.

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

Mở `http://localhost:3000`.

Đăng nhập lần đầu: `root` / `123456`. Hãy đổi mật khẩu ngay.

## ⚡ Edge Proxy

Cloudflare Edge Proxy cho phép Worker nhận `/v1/*`, lấy routing và key từ OpenAPI-manager, rồi stream trực tiếp từ edge tới provider.

Hướng dẫn: [cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 Ghi nhận và giấy phép

OpenAPI-manager dựa trên [one-api](https://github.com/songquanpeng/one-api). Cảm ơn tác giả gốc và contributors. MIT License: [LICENSE](./LICENSE).
