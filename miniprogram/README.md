# MeowHome 小程序端（uni-app）

## ⚠️ 最重要的一条：微信开发者工具导入哪个目录

**导入 `dist/build/mp-weixin`，不是本目录。**

```
✅  D:\MyProject\MeowHome\miniprogram\dist\build\mp-weixin     ← 编译产物，含 app.json
❌  D:\MyProject\MeowHome\miniprogram                          ← 本目录，uni-app 源码工程，没有 app.json
❌  D:\MyProject\MeowHome\miniprogram\src                       ← 源码，没有 app.json
❌  D:\MyProject\MeowHome                                      ← 仓库根，没有 app.json
```

导入错的目录时，开发者工具会报 **「app.json 文件在项目根目录未找到」** —— 这不是构建失败，是选错了目录。

`app.json` / `app.js` / `app.wxss` / `project.config.json` 都由 uni-app 编译生成，只存在于产物目录。

## 构建

```powershell
cd D:\MyProject\MeowHome\miniprogram
npm.cmd run build:mp-weixin
```

看到 `DONE  Build complete.` 即成功，产物在 `dist/build/mp-weixin`。

> 本机 PowerShell 执行策略为 Restricted，所以用 `npm.cmd` 而不是 `npm`（后者会走 `npm.ps1` 被拦）。

开发时可用增量编译（改源码自动重新产出，开发者工具里会自动刷新）：

```powershell
npm.cmd run dev:mp-weixin
```

## 后端

小程序接口指向 `http://127.0.0.1:8080`（见 `src/api/config.ts`），**验证前需先启动后端**：

```powershell
cd D:\MyProject\MeowHome\backend
go run ./cmd/server
```

`project.config.json` 里已设 `urlCheck: false`，因此在开发者工具中**无需**手动勾选「不校验合法域名」。

## 目录说明

| 路径                    | 作用                                                                  |
| ----------------------- | --------------------------------------------------------------------- |
| `src/pages/`            | 页面（对应 Web 端 `frontend/src/views/`）                             |
| `src/api/`              | 网络层：`client.ts`（uni.request 封装）、`endpoints.ts`、`adapter.ts` |
| `src/stores/`           | Pinia store                                                           |
| `src/utils/guard.ts`    | 路由守卫（Web 端 `router.beforeEach` 的等价实现）                     |
| `src/styles/`           | WXSS 样式（由 Web 端 CSS 适配而来）                                   |
| `src/components/app/`   | `AppIcon`（CSS 遮罩图标）                                             |
| `scripts/`              | 构建期辅助脚本 + 验证脚本                                             |
| `dist/build/mp-weixin/` | **编译产物，开发者工具导入这里**                                      |

## 可运行的验证脚本

```powershell
node scripts/verify-auth-flow.mjs
```

复刻 `src/api/client.ts` 的协议行为打真实后端（需后端已启动），验证认证链路、401 刷新、跨家庭隔离、PATCH 降级等。结果同时写入 `scripts/auth-flow-report.md`。

## 迁移进度

见仓库根目录 `MeowHome-小程序迁移步骤.md`（§10 进度追踪 / §12 开发记录）。
