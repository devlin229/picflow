# PicFlow Frontend

PicFlow 的前端是一个 React + TypeScript 单页应用，不包含登录与账号体系。当前版本可直接在浏览器中完成图片标准化、结果下载和尺寸图生成；后端接口统一预留在 `/api`。

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
