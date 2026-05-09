# 🚀 OpenAPI-manager

**OpenAPI-manager는 OpenAI 호환 API를 위한 게이트웨이, 지역 릴레이, Provider 라우터, 로드 밸런서, 멀티모달 트래픽 관리 플랫폼입니다.**

앱과 에이전트는 하나의 안정적인 Base URL만 사용하고, OpenAPI-manager가 provider API Key, 라우팅, 재시도, RPM/TPM 제한, 지역별 전달, provider 호환성을 처리합니다.

## 🌟 핵심 가치

- 🌏 사용자가 중국에 있든 해외에 있든, upstream provider 요청은 OpenAPI-manager 서버가 위치한 지역에서 전송됩니다.
- ⚖️ 여러 Key / 여러 Channel 로드 밸런싱으로 NVIDIA, OpenRouter, Gemini, Groq 같은 무료 또는 저한도 provider를 더 효율적으로 사용할 수 있습니다.
- ⚡ Cloudflare Edge Proxy로 지연에 민감한 스트리밍 경로를 가속할 수 있습니다.
- 🧩 해외 전용 provider와 중국 친화 provider를 하나의 계층에서 관리할 수 있습니다.
- 🖼️ Chat, Vision, 이미지 생성, 오디오, 비디오 작업 스타일 relay를 지원합니다. 실제 지원 여부는 provider와 model에 따라 달라집니다.

```text
Client / App / Agent → OpenAPI-manager Server → Upstream Provider
```

## 🧩 Provider 예시

글로벌 provider: OpenAI, Claude, Gemini, Groq, OpenRouter, NVIDIA, Cohere, Mistral, xAI.

중국 및 중국 친화 provider: DeepSeek, Qwen / Tongyi, ERNIE / Wenxin Qianfan, Tencent Hunyuan, Doubao / Volcano Ark, Zhipu GLM, Kimi, Baichuan, MiniMax, iFlytek Spark, SiliconFlow, 360, 01.AI, StepFun.

## 🐳 Docker Hub

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

```bash
docker pull heing8848/openapi-manager:latest
docker run --name openapi-manager -d --restart always -p 3000:3000 -e TZ=Asia/Shanghai -v /home/ubuntu/data/openapi-manager:/data heing8848/openapi-manager:latest
```

`http://localhost:3000` 을 엽니다.

초기 로그인: `root` / `123456`. 로그인 후 즉시 비밀번호를 변경하세요.

## ⚡ Edge Proxy

Cloudflare Edge Proxy는 Worker가 `/v1/*` 요청을 받고, OpenAPI-manager에서 라우팅과 Key를 받은 뒤, Edge에서 upstream provider로 직접 스트리밍합니다.

가이드: [cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🙏 Attribution / License

OpenAPI-manager는 [one-api](https://github.com/songquanpeng/one-api)를 기반으로 합니다. 원作者와 contributors에게 감사드립니다. MIT License이며 자세한 내용은 [LICENSE](./LICENSE)를 참고하세요.
