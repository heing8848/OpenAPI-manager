# 🚀 OpenAPI-manager

**OpenAPI-manager 是一个兼容 OpenAI API 格式的 AI API 网关、中转转发层、Provider 路由器、负载均衡器和多模态流量管理平台。**

它让你的应用只需要接入一个稳定的 Base URL，背后由 OpenAPI-manager 统一处理 provider API Key、渠道选择、重试、额度控制、RPM/TPM 限制、多区域转发和 provider 兼容性。

> 本项目基于 [one-api](https://github.com/songquanpeng/one-api) 继续开发，并重点增强了多 Key 路由、免费层 provider 容量池、中转转发、Cloudflare Edge Proxy 加速、多模态兼容和公开部署体验。

## 🌟 为什么需要 OpenAPI-manager

AI provider 的访问环境很碎片化：

- 🌏 用户可能在中国，也可能在海外。
- 🧱 有些 provider 更适合海外网络，有些 provider 更适合中国或亚洲网络。
- ⏱️ NVIDIA、OpenRouter、Gemini、Groq 等 provider 的免费层或低价层常常有严格 RPM/TPM 限制。
- 🔑 一个 API Key 很容易被多人、多个 agent 或突发请求打满。
- 🧩 不同 provider 即使号称兼容 OpenAI，实际请求/响应格式也会有差异。

OpenAPI-manager 在客户端和 provider 之间放置一个可控的中转点：

```text
客户端 / 应用 / Agent
        ↓
OpenAPI-manager 服务器
        ↓
被选中的上游 Provider
```

无论用户的 request point 身处中国还是海外，真正向 provider 发出的请求都会从 **OpenAPI-manager 服务器所在地** 发出。你可以根据上游 provider 的网络特性选择服务器地区，同时把 provider API Key 留在服务端，不需要暴露到每个客户端。

## ⚖️ 为低 RPM/TPM Provider 做负载均衡

很多免费层或低价 provider 并不是不能用，而是额度分散、限制严格、单 Key 容量太小。OpenAPI-manager 的重点之一就是把这些碎片化容量整合成一个可管理的 API 池。

特别适合：

- NVIDIA 免费层
- OpenRouter 免费/低价模型
- Google Gemini 免费层
- Groq 免费层
- 各类 OpenAI-compatible provider
- 自建模型或私有 OpenAI-compatible 服务

它可以帮你：

- 🔁 在同一个渠道下的多个 API Key 之间轮询分摊请求。
- 🧭 在多个渠道、多个 provider、多个账号之间做路由。
- 🧯 当某个渠道额度耗尽、健康状态异常或临时失败时，自动重试或绕行。
- 📊 集中记录日志、额度、计费权重、渠道健康度和调用结果。
- 🧑‍💻 让客户端保持简单：只需要一个 Base URL 和一个用户令牌。

## 🧭 中转转发与区域网络优势

OpenAPI-manager 的一个核心价值是 **服务端中转转发**：

```text
中国用户      ┐
海外用户      ├──> OpenAPI-manager 服务器 ──> 上游 Provider
后端 / Agent  ┘
```

这样做的好处：

- 🌍 中国和海外用户可以共用同一个 API endpoint。
- 🧭 上游 provider 请求从你选择的服务器地区发出，而不是从用户设备直接发出。
- 🔐 provider API Key 留在服务端，避免散落在客户端、脚本或 agent 环境里。
- 🧩 可以同时管理海外 only provider 和中国友好的 provider。
- 🧾 日志、额度、重试、路由、计费权重和健康检查都集中在一个地方。

例如：

- OpenAI、Claude、Gemini、Groq、OpenRouter、NVIDIA 等海外 provider 可以通过网络更稳定的服务器统一转发。
- DeepSeek、通义千问、文心千帆、腾讯混元、豆包、智谱 GLM、Kimi、硅基流动等中国或中国友好的 provider 可以通过更适合的渠道配置接入。

## ⚡ Cloudflare Edge Proxy 加速

对于延迟敏感的请求，OpenAPI-manager 还支持可选的 **Cloudflare Edge Proxy**。

Edge Proxy 的思路是：

```text
客户端 → Cloudflare Worker → 上游 Provider
             ↑
       向 OpenAPI-manager 获取路由与 Key
```

Worker 接收 `/v1/*` 请求后，先向 OpenAPI-manager 请求本次应该使用的渠道和 Key，然后直接从 Cloudflare 边缘节点向上游 provider 发起流式请求。

它的价值：

- ⚡ 降低首 token 延迟。
- 🌉 在某些中国/跨境网络路径下，比所有流量都绕回中心服务器更快。
- 📉 减少中心服务器转发大流式响应的带宽压力。
- 🔐 仍由 OpenAPI-manager 统一做鉴权、路由选择和用量回传。

完整教程：

[cloudflare-edge-proxy/README.md](./cloudflare-edge-proxy/README.md)

## 🧩 Provider 兼容性

OpenAPI-manager 面向客户端提供统一的 OpenAI-compatible API，同时尽量适配不同 provider 的请求差异。

### 全球/海外 Provider

- OpenAI
- Anthropic Claude
- Google Gemini
- Groq
- OpenRouter
- NVIDIA
- Cohere
- Mistral
- Together AI
- xAI
- 其他 OpenAI-compatible 服务

### 中国与中国友好 Provider

- DeepSeek
- 阿里通义千问 / DashScope / 百炼
- 百度文心千帆 / ERNIE
- 腾讯混元
- 字节豆包 / 火山方舟
- 智谱 GLM / ChatGLM
- Moonshot / Kimi
- 百川智能
- MiniMax
- 讯飞星火
- 硅基流动 SiliconFlow
- 360 智脑
- 零一万物 / Yi
- 阶跃星辰 StepFun

具体能力取决于上游 provider、模型和渠道配置。OpenAPI-manager 的目标是尽量统一路由、请求处理、响应处理和运维管理。

## 🖼️ 多模态支持

在上游 provider 支持的前提下，OpenAPI-manager 包含多模态中转能力：

- 💬 流式与非流式聊天补全
- 👁️ Vision / 图片理解类聊天请求
- 🎨 图片生成响应处理
- 🔊 音频相关接口中转
- 🎬 视频任务类 provider 中转

多模态功能是否可用，最终仍取决于具体 provider 和模型。

## 🐳 Docker Hub

公开镜像：

[heing8848/openapi-manager](https://hub.docker.com/r/heing8848/openapi-manager)

拉取镜像：

```bash
docker pull heing8848/openapi-manager:latest
```

使用 SQLite 快速启动：

```bash
docker run --name openapi-manager \
  -d \
  --restart always \
  -p 3000:3000 \
  -e TZ=Asia/Shanghai \
  -v /home/ubuntu/data/openapi-manager:/data \
  heing8848/openapi-manager:latest
```

打开：

```text
http://localhost:3000
```

首次登录默认账号：

```text
username: root
password: 123456
```

首次登录后请立即修改默认密码。

## 🧱 Docker Compose

复制环境变量模板：

```bash
cp .env.example .env
```

编辑 `.env`，替换所有占位符：

```env
SESSION_SECRET=<generate-a-long-random-session-secret>
MYSQL_ROOT_PASSWORD=<generate-a-strong-root-password>
MYSQL_PASSWORD=<generate-a-strong-database-password>
SQL_DSN=oneapi:<same-as-MYSQL_PASSWORD>@tcp(db:3306)/one-api
```

启动：

```bash
docker compose up -d
```

更多部署说明：

[docs/open-source-deployment.md](./docs/open-source-deployment.md)

## 🔐 发布前安全检查

不要公开：

```text
.env
data/
logs/
*.db
*.pem
*.key
*.crt
```

发布前可以快速扫描：

```bash
rg -n -a "sk-[A-Za-z0-9_-]{8,}|sk-or-v1-|nvapi-|AIza|AKIA|Bearer\\s+[A-Za-z0-9._-]{10,}|http://[0-9]{1,3}(\\.[0-9]{1,3}){3}" . --glob "!data/**" --glob "!logs/**" --glob "!.git/**"
```

正常情况下只应该看到占位符或文档里的检查命令。

## 🙏 致谢

OpenAPI-manager 基于 [one-api](https://github.com/songquanpeng/one-api) by [songquanpeng](https://github.com/songquanpeng) 继续开发。

感谢 one-api 原作者和贡献者提供的项目基础。

## 📄 许可证

本项目使用 MIT License。详见 [LICENSE](./LICENSE)。
