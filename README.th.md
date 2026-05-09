# 🚀 OpenAPI-manager

**OpenAPI-manager คือ API gateway ที่เข้ากันได้กับ OpenAI, regional relay, provider router, load balancer และแพลตฟอร์มจัดการทราฟฟิกแบบ multimodal**

แอปและ agent ใช้ Base URL เดียว ส่วน OpenAPI-manager จัดการ provider API key, routing, retry, ข้อจำกัด RPM/TPM, regional forwarding และความเข้ากันได้ของ provider ให้เบื้องหลัง

## 🌟 จุดเด่น

- 🌏 ไม่ว่าผู้ใช้อยู่จีนหรือต่างประเทศ คำขอไปยัง provider จะออกจากตำแหน่งของเซิร์ฟเวอร์ OpenAPI-manager
- ⚖️ load balancing หลาย key / หลาย channel เหมาะกับ provider ที่มี free tier หรือ limit ต่ำ เช่น NVIDIA, OpenRouter, Gemini, Groq
- ⚡ ใช้ Cloudflare Edge Proxy เพื่อเร่ง response แบบ streaming ที่ไวต่อ latency
- 🧩 จัดการ provider ฝั่ง global และ provider ที่เหมาะกับจีนในชั้นเดียว
- 🖼️ รองรับ relay สำหรับ chat, vision, image generation, audio และ video-task เมื่อ provider รองรับ

```text
Client / App / Agent → OpenAPI-manager Server → Upstream Provider
```

## 🧩 ตัวอย่าง Provider

Global: OpenAI, Claude, Gemini, Groq, OpenRouter, NVIDIA, Cohere, Mistral, xAI.

China-friendly: DeepSeek, Qwen / Tongyi, ERNIE / Wenxin Qianfan, Tencent Hunyuan, Doubao / Volcano Ark, Zhipu GLM, Kimi, Baichuan, MiniMax, iFlytek Spark, SiliconFlow, 360, 01.AI, StepFun.

## 🐳 Docker Hub

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/openapi-manager:/data heing8848/openapi-manager:latest
```

เปิด `http://localhost:3000`

บัญชีเริ่มต้น: `root` / `123456` กรุณาเปลี่ยนรหัสผ่านทันทีหลังเข้าสู่ระบบครั้งแรก

## ⚡ Edge Proxy

Cloudflare Edge Proxy ให้ Worker รับคำขอ `/v1/*`, ขอ routing และ key จาก OpenAPI-manager แล้ว stream จาก edge ไปยัง upstream provider โดยตรง

คู่มือ: [cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 Attribution / License

OpenAPI-manager พัฒนาต่อยอดจาก [one-api](https://github.com/songquanpeng/one-api) ขอบคุณผู้เขียนเดิมและ contributors ใช้ MIT License ดู [LICENSE](./LICENSE)
