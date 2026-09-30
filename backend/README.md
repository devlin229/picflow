# PicFlow 后端

PicFlow 是面向电商商品图的轻量图片处理服务。V0.2 提供批量上传、contain/cover 缩放裁剪、单一背景换色、格式转换、JPG 压缩、主图模板、可定位规格表格、单张下载和 ZIP 下载，并提供可选 Qwen / OpenAI AI 换背景扩展。

## AI 换背景

在 `backend/.env` 配置以下环境变量并重启后端：

```dotenv
LLM_PROTOCOL=qwen
LLM_API_KEY=填写你的密钥
LLM_BASE_URL=https://maas.qianwenaiapi.com/api/v1
LLM_MODEL=qwen-image-3.0
LLM_TIMEOUT_SECONDS=600
LLM_REQUESTS_PER_MINUTE=20
```

基础地址须与密钥所属平台一致；百炼工作空间使用对应的 API Host 加 `/api/v1`。密钥为空时关闭 AI 能力，原有本地处理可继续使用。Compose 部署支持同名变量。工作台选择“AI 换背景”，背景描述留空时使用所选纯色，也可填写场景描述。

切换为 OpenAI 时，在 `backend/.env` 同时替换以下四项，然后重启后端（本地可重新运行根目录 `./dev.sh`）：

```dotenv
LLM_PROTOCOL=openai
LLM_API_KEY=填写你的OpenAI密钥
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL=gpt-image-2.5-sunburst
```

`LLM_BASE_URL` 和 `LLM_MODEL` 留空时按协议选用默认值：Qwen 使用上述地址和模型；OpenAI 使用上述 OpenAI 地址和模型。支持按账户权限填写其他 GPT Image 模型名（例如 `gpt-image-2.5-flare`、`gpt-image-2`、`gpt-image-1.5`）。OpenAI 适配器使用 `/images/edits` 上传原图，固定只生成一张、`quality=medium`、自动尺寸、PNG，后续裁剪和 JPG 压缩仍由本地处理。GPT Image 1/1.5 显式设置高输入保真，较新型号不发送该参数。OpenAI 兼容服务需支持该图片编辑接口并返回 `data[].b64_json`，不使用聊天接口或自动回退其他模型。切换供应商后请新建任务；旧任务重试仍优先复用已保存图片。

适配实现位于 `internal/llm`，参考 check-img 的模型接口结构，实现 Qwen DashScope 和 OpenAI Images 图片编辑，共享限流、超时及输入校验。先编辑背景，再执行本地裁剪、尺寸和规格表格处理。明确的“换成黑色”等指令会同步为输出画布颜色，避免补边颜色冲突。纯色结果会进行边缘颜色初步校验，明显未换色时报告失败，不自动再次调用模型。

AI 原图上限 10MB，默认每分钟 20 次。模型返回图片保存于 `data/outputs/<任务ID>/<图片ID>_ai.png`，随任务删除；同一任务重试复用已保存图片。没有成功保存返回图的图片，手动重试仍可能重新收费；新建任务也会重新请求模型。日志记录供应商请求 ID，Qwen 记录输入/输出图片数量，OpenAI 记录输入/输出 token 用量，便于核对额度；这些日志不等于最终账单。模型不保证商品细节完全一致，边缘校验不代替人工验收。当前不提供透明 alpha 抠图。

接口依据：[Qwen Image 3.0 图片编辑文档](https://help.aliyun.com/zh/model-studio/qwen-image-generation-and-editing-api-reference)。

OpenAI 接口依据：[官方图片编辑文档](https://developers.openai.com/api/docs/guides/image-generation)。

## 技术结构

- Gin：HTTP API
- GORM + SQLite：任务和模板数据
- Viper：`.env` 配置
- 进程内 Worker：异步批量处理
- 本地文件存储：原图与输出文件
- 可选静态文件服务：统一镜像中直接托管前端 SPA

主要目录：

```text
internal/api/handler     HTTP Handler
internal/api/middleware  通用中间件
internal/service         业务流程
internal/repository      GORM 数据访问
internal/model           数据模型
internal/imageproc       规则型图片处理
internal/storage         本地文件存储
internal/worker          异步任务 Worker
```

## 本地运行

```bash
cp .env.example .env
go mod tidy
go run . serve
```

默认监听 `http://localhost:8080`，数据写入 `./data`。健康检查：

```bash
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
```

## 配置

| 配置项 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `8080` | HTTP 端口 |
| `DATA_DIR` | `./data` | SQLite、原图和输出目录 |
| `MAX_UPLOAD_SIZE_MB` | `20` | 单张图片大小上限 |
| `WORKER_COUNT` | `2` | 图片处理 Worker 数量 |
| `CORS_ALLOWED_ORIGINS` | 空 | 允许跨域访问的前端来源，多个值用逗号分隔 |
| `WEB_DIR` | 空 | 前端构建产物目录；本地开发留空，统一镜像中为 `/app/web` |

`.env` 文件是可选的；容器中可完全通过环境变量配置。生产部署统一使用项目根目录的 `Dockerfile` 和 `docker-compose.yml`，不单独构建后端镜像。

## 接口

JSON 接口统一返回：

```json
{"code":0,"msg":"success","data":{}}
```

失败时保留对应 HTTP 状态码，并返回 `code`、`msg`、`data`；图片和 ZIP 下载直接返回二进制内容。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/tasks` | 批量上传并创建图片处理任务，可选同时绘制规格表格 |
| `GET` | `/api/tasks/:taskID` | 查询任务状态和输出 |
| `POST` | `/api/tasks/:taskID/retry` | 重试失败任务 |
| `DELETE` | `/api/tasks/:taskID` | 清空任务 |
| `DELETE` | `/api/tasks/:taskID/assets/:assetID` | 删除任务中的单张原图 |
| `POST` | `/api/tasks/:taskID/size-charts` | 兼容旧版的二次规格图生成接口 |
| `GET` | `/api/tasks/:taskID/download.zip` | 下载全部结果 |
| `GET` | `/api/outputs/:outputID/download` | 下载单张结果 |
| `GET` | `/api/templates` | 查询模板 |
| `POST` | `/api/templates` | 保存模板 |
| `DELETE` | `/api/templates/:templateID` | 删除自定义模板 |

更完整的字段定义见 `openapi.yaml`。
