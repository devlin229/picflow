# PicFlow Frontend

PicFlow 的前端是一个 React + TypeScript 单页应用，不包含登录与账号体系。当前版本可在工作台配置图片裁剪、画布补边、可选 AI 换背景和规格表格，并下载结果；规格表格支持四个角落预设、鼠标自由拖动、四边及四角缩放，并可设置边框和文字、背景颜色。原图背景处理仅提供“不换背景”和“AI 换背景”，已移除本地换背景功能；后端接口统一位于 `/api`。

## 本地开发

```bash
npm install
npm run dev
```

开发环境通过 `VITE_DEV_API_TARGET` 配置后端地址，默认是 `http://localhost:8080`。

## 构建

```bash
npm run build
```

## 容器部署

生产环境不单独构建前端镜像。项目根目录的多阶段 `Dockerfile` 会先构建前端，再把 `dist/` 和 Go 后端放进同一个最终镜像，由 Go 服务同时提供静态页面、SPA fallback 和 `/api` 接口。部署命令见根目录 `README.md`。
