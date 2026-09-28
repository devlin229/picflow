# PicFlow 后端

PicFlow 是面向电商商品图的轻量图片处理服务。V1 提供批量上传、等比缩放与补边、主图模板、尺寸图、单张下载和 ZIP 下载，不包含登录、权限、计费、AI 或 ComfyUI。

## 技术结构

- Gin：HTTP API
- GORM + SQLite：任务和模板数据
- Viper：`.env` 配置
- 进程内 Worker：异步批量处理
- 本地文件存储：原图与输出文件

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

## 接口

JSON 接口统一返回：

```json
{"code":0,"msg":"success","data":{}}
```

失败时保留对应 HTTP 状态码，并返回 `code`、`msg`、`data`；图片和 ZIP 下载直接返回二进制内容。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/tasks` | 批量上传并创建标准化任务 |
| `GET` | `/api/tasks/:taskID` | 查询任务状态和输出 |
| `DELETE` | `/api/tasks/:taskID` | 清空任务 |
| `DELETE` | `/api/tasks/:taskID/assets/:assetID` | 删除任务中的单张原图 |
| `POST` | `/api/tasks/:taskID/size-charts` | 生成尺寸图 |
| `GET` | `/api/tasks/:taskID/download.zip` | 下载全部结果 |
| `GET` | `/api/outputs/:outputID/download` | 下载单张结果 |
| `GET` | `/api/templates` | 查询模板 |
| `POST` | `/api/templates` | 保存模板 |
| `DELETE` | `/api/templates/:templateID` | 删除自定义模板 |

更完整的字段定义见 `openapi.yaml`。
