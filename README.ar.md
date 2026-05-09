<div dir="rtl">

# 🚀 OpenAPI-manager

**OpenAPI-manager هو بوابة API متوافقة مع OpenAI، وطبقة Relay إقليمية، وموجّه Providers، وموازن حمل، ومدير لحركة المرور متعددة الوسائط.**

يوفّر لتطبيقاتك عنوان Base URL واحداً، بينما يتولى الخادم إدارة مفاتيح providers، والتوجيه، وإعادة المحاولة، وضغط حدود RPM/TPM، والتحويل الإقليمي، والتوافق بين providers.

## 🌟 لماذا OpenAPI-manager؟

- 🌏 سواء كان المستخدم في الصين أو خارجها، يتم إرسال الطلب إلى provider من موقع خادم OpenAPI-manager.
- ⚖️ توزيع الحمل على عدة مفاتيح وقنوات يساعد مع providers ذات الحدود الصارمة مثل NVIDIA وOpenRouter وGemini وGroq.
- ⚡ يمكن استخدام Cloudflare Edge Proxy لتسريع مسارات البث الحساسة للزمن.
- 🧩 يمكن إدارة providers عالمية وproviders مناسبة للصين من طبقة واحدة.
- 🖼️ يدعم مسارات chat وvision وimage generation وaudio وvideo-task حيث يدعمها provider.

```text
Client / App / Agent → OpenAPI-manager Server → Upstream Provider
```

## 🧩 أمثلة Providers

عالمية: OpenAI، Claude، Gemini، Groq، OpenRouter، NVIDIA، Cohere، Mistral، xAI.

الصين أو المتوافقة معها: DeepSeek، Qwen / Tongyi، ERNIE / Wenxin Qianfan، Tencent Hunyuan، Doubao / Volcano Ark، Zhipu GLM، Kimi، Baichuan، MiniMax، iFlytek Spark، SiliconFlow، 360، 01.AI، StepFun.

## 🐳 Docker Hub

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/openapi-manager:/data heing8848/openapi-manager:latest
```

افتح `http://localhost:3000`.

تسجيل الدخول الأولي: `root` / `123456`. غيّر كلمة المرور فوراً.

## ⚡ Edge Proxy

يسمح Cloudflare Edge Proxy للـ Worker باستقبال طلبات `/v1/*`، ثم طلب معلومات التوجيه والمفتاح من OpenAPI-manager، وبعدها البث مباشرة من edge إلى provider.

الدليل: [cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 النسبة والترخيص

OpenAPI-manager مبني على [one-api](https://github.com/songquanpeng/one-api). شكراً للمؤلف الأصلي والمساهمين. الترخيص MIT، راجع [LICENSE](./LICENSE).

</div>
