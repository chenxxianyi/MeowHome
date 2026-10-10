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

当前调试接口指向电脑 WLAN 地址 `http://192.168.1.13:8080`（见 `src/api/config.ts`），模拟器和同一局域网内的手机均可使用。电脑 IP 变化后需更新此配置并重建；手机不能用 `127.0.0.1` 访问电脑后端。[uni.request 官方说明](https://uniapp.dcloud.net.cn/api/request/request)。**验证前需先启动后端**：

```powershell
cd D:\MyProject\MeowHome\backend
go run ./cmd/server
```

`project.config.json` 里已设 `urlCheck: false`，因此在开发者工具中**无需**手动勾选「不校验合法域名」。

### `ERR_CONNECTION_REFUSED`

`GET .../api/v1/me net::ERR_CONNECTION_REFUSED` 表示当前后端地址连接被拒绝，先确认后端已启动且 `backend/.env` 中 `APP_PORT` 与 `src/api/config.ts` 的端口一致。只启动小程序编译进程不会启动 Go 后端。

本机实际可用的启动方式（在单独的 PowerShell 中保持运行）：

```powershell
cd D:\Myproject\MeowHome\backend
$env:GOPATH = 'C:\Users\chenxianyi\go-work'
& 'C:\Users\chenxianyi\go\bin\go.exe' run ./cmd/server
```

其他机器使用自己的 Go 安装路径。

为便于持续真机调试，建议在你自己的独立 PowerShell 窗口运行项目启动脚本：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File D:\Myproject\MeowHome\backend\scripts\start-local.ps1 -Restart
```

脚本定位 backend 目录和本机 Go，读取现有 `.env` 并在当前终端运行服务。调试期间保持窗口打开，用 Ctrl+C 停止。`-Restart` 可将此前的本项目 `bin/meowhome-debug.exe` 后台实例停止，改为由你的终端运行；匹配可执行文件的完整路径。首次直接执行 ps1 被本机 Restricted 策略阻止，以上命令已验证可启动脚本，策略仅用于该次 PowerShell 进程；未更改系统策略。

启动后在另一个终端验证：

```powershell
curl.exe http://127.0.0.1:8080/health/live
curl.exe http://127.0.0.1:8080/health/ready
```

真机调试前让手机和电脑连接同一 Wi-Fi，在手机浏览器打开 `http://192.168.1.13:8080/health/live`，应返回 `SUCCESS` 和 `status: up`。若打不开，继续检查局域网可达性、路由器客户端隔离及 Windows 防火墙；仅电脑访问成功不能证明手机能访问。确认工具本地调试使用“忽略域名校验”，随后重新编译并生成新的真机调试码。正式发布仍须配置备案 HTTPS 域名及合法域名校验。

2026-10-07 手机注册“处理中”排查：截图为登录页正在提交注册；原地址是 `127.0.0.1`，后端没有收到对应手机注册请求。改为上述 WLAN 地址；普通请求增加 15 秒独立超时、刷新增加 10 秒独立超时，超时中止 RequestTask，忽略迟到回调并保留页面 finally 恢复按钮；重复提交直接返回。新增 5 项微信客户端产物回归，旧代码 2/5 通过，修复后 5/5。`test:mp-weixin` 合计 12/12，源码测试 13/13、类型/迁移检查通过；实际手机可达性与注册结果待人工确认。

### 登录返回 200、客户端仍超时：回调对照诊断

2026-10-07 项目方提供登录请求 `req-1791373911756-b19d97e0` 的前端日志：发送后 15002 毫秒触发独立超时，没有 `response.success` / `response.failed`，页面 finally 已结束提交。项目方确认同一手机浏览器健康接口显示 SUCCESS/up；后续服务器日志确认该请求 HTTP 200、耗时约 18 毫秒。新登录请求 `req-1791375567033-35c00e1e` 同样来自手机 `192.168.1.8`，HTTP 200、耗时约 20 毫秒，客户端仍在 15003 毫秒超时。排查重心为小程序收到响应并调用回调的环节；现有证据不支持服务端处理慢、数据库拒绝登录或同步存储阻塞。

只读检查显示 WLAN 公用配置开启防火墙，当前后端监听所有地址；本地规则存在与当前 server.exe 路径对应的公用 TCP Allow。端口过滤器查询因 Windows 权限失败，改用 netsh 和注册表读取；没有修改防火墙或网络配置。新的后端由项目方终端运行，旧后台日志文件不能用于断言该次请求未到达。

诊断当前直接放在 `src/App.vue`，已删除独立的 `network-diagnostics.ts` 模块和旧 globalThis Console 入口。上一轮自动入口在手机出现 `startLocalNetworkDiagnostics is not a function`，而磁盘模块存在该导出，实际微信模块加载差异的原因尚未确认；本轮移除这项模块依赖，并先初始化应用网络监听，再启动诊断，诊断异常不会中断正常启动。**本地 HTTP 调试包启动后自动运行一次**，正式 HTTPS 配置不运行。

结束旧真机调试，在工程 `dist/build/mp-weixin` 重新编译并重新扫码。本轮启动日志的 `buildTime` 应为 **`2026-10-07T12:26:55.333Z`**（北京时间 20:26:55）；后续重新构建会改变该值。Console 应显示 `[network-probe] auto.start`，约 6 秒内输出 `[network-probe] summary`，复制该行即可；没有 summary 时提供最后一条探针日志。不需要输入 Console 命令。

工具自动发出四个只读 `/health/live` 请求：`uni-json`、`uni-json-complete`（增加 complete 回调的对照）、`wx-json` 和 `wx-text`。不传账号、密码、登录令牌或业务写入；结果仅含探针名称、请求 ID、阶段、状态与耗时，超时中止且忽略迟到回调。**必须在独立的真机调试窗口收集手机日志**，电脑主窗口的模拟器结果不能代替手机结果。后端通过相同探针 ID 的 client_ip 辅助核对来源。

20:27:59 项目方提供的四个探针均 HTTP 200、耗时 101–108 毫秒，后端来源为电脑 `192.168.1.13`；项目方已确认日志来自电脑模拟器。这验证了模拟器的启动及健康响应，真机回调仍未验证。同次 `ENOENT ...utils/network-diagnostics.js` 来自微信工具尝试读取已删除模块；当前源码/编译目录已无该路径引用，属于工具持有旧引用的证据，不能据此断定登录超时的原因。

随后项目方重启仍报 ENOENT。已通过微信 CLI 验证：仅刷新文件列表、清编译缓存仍失败；进一步清文件缓存并关闭/重新打开当前工程后，**实际微信编译和预览成功**，20:42:21 生成 `dist/preview/meowhome-preview.png`，包体 694379 字节（678.1 KB）。当前缺失文件的构建错误已排除，手机登录仍待测试。可扫码该二维码作普通预览，对照真机远程调试结果；本地 HTTP 需同一 Wi-Fi、后端持续运行，若提示合法域名限制则在小程序菜单中开启调试。

工具升级或重建产物后再次持有旧文件引用，可在已开启服务端口的微信工具上执行以下项目限定命令。先结束真机调试，命令会重新打开当前工程：

```powershell
$wechatCli = 'D:\tool\微信web开发者工具\cli.bat'
$wechatProject = 'D:\Myproject\MeowHome\miniprogram\dist\build\mp-weixin'
& $wechatCli reset-fileutils --project $wechatProject
& $wechatCli cache --clean compile --project $wechatProject
& $wechatCli cache --clean file --project $wechatProject
& $wechatCli close --project $wechatProject
& $wechatCli open --project $wechatProject
```

服务端口关闭时 CLI 会拒绝执行；本轮由项目方在设置中开启。预览会向微信上传编译产物，本轮首次被自动审批拒绝（账号归属和上传授权不明确），项目方随后确认 AppID `wxb376a5d7842bf256` 属于其账号并明确同意临时预览上传，再执行成功。授权用于本轮预览验证；正式上线仍按发布流程完成。

| 结果 | 下一步 |
| --- | --- |
| 框架请求失败，原生 JSON 成功 | 核对 uni.request 的参数/回调封装，验证原生传输兼容方案 |
| 只有 complete 对照成功 | 核对回调注册差异，并在登录请求中验证 |
| 原生 JSON 失败、原生 text 成功 | 验证 JSON 自动解析与文本接收差异 |
| 四个探针均超时，后端对应请求均 200 | 继续检查微信调试桥接/原生响应交付，比较普通预览模式 |
| 四个均成功、登录仍失败 | 核对该次登录的服务器耗时和响应，与健康请求分别判断 |

本轮新增 5 项实际编译 App 的启动回归，以缺失诊断导出的替身复现手机异常：旧产物 0/5，新产物 5/5；覆盖正常启动、回调完全缺失、成功回调、平台同步抛错及 HTTPS 不运行诊断。微信构建及合计 **17/17** 产物回归、类型/迁移检查、定向 ESLint/Prettier 通过；测试的平台和网络为替身，不代替真机验收。原有构建循环警告保留。上一轮源码 13/13 的结果保留，本轮未重复运行；手机探针尚未取得，登录超时仍待定位。

### 注册与服务停止的历史检查

19:33 截图的后续检查：日志确认上述后台实例在 2026-10-07 **19:29:26** 记录 `server stopped` 后退出；19:33 操作时服务已关闭，本次检查 LAN 健康接口连接失败，文件日志没有该时段的手机注册/登录请求。已重新启动，LAN `/health/live` 和 `/health/ready` 均 HTTP 200。谁触发停止尚未确认，后台启动不意味着永久常驻，后续用上述脚本保持独立终端运行。手机浏览器访问 LAN 健康地址并登录已有账号的结果仍待确认。

截图中的 vConsole **System** 标签展示系统、网络和 UA，没有业务异常栈；`System: Unknown` 与 UA 中的 `OpenHarmony 7.0` 不能单独证明项目存在鸿蒙兼容故障。需要查看电脑真机调试窗口 Console 中的 `[api]` / `[auth]` 步骤，或手机 Log 中可见的业务日志；本轮依据服务停止时间定位，不将其归因于广告日志或 Storage 假设。

2026-10-07 17:26:37，后端收到手机 `192.168.1.8` 的 `POST /api/v1/auth/register`，请求 ID 为 `req-1791365198212-4a639a3a`，返回 HTTP 201，耗时约 25 毫秒，与项目方真机截图一致。**该次账号已创建，家庭创建尚未观察到；后续应使用已有账号登录。** 上文“没有收到手机请求”属于调整地址之前的检查结果。

项目方随后反馈出现连接超时；复查时原后端进程已不在、LAN 健康接口连接失败，已构建 `backend/bin/meowhome-debug.exe` 并通过 `Start-Process -WindowStyle Hidden` 在后台启动，stdout/stderr 写入同目录的 `server-*.log`，恢复后 LAN 健康接口 HTTP 200。电脑必须持续运行后端；这些本地二进制和日志均在忽略目录中，`.env` 未修改。

小程序启动日志新增 `buildTime`（UTC ISO 时间），用于确认手机加载了重新构建的代码。认证请求只记录接口路径、请求 ID、状态和耗时；认证动作记录响应、登录状态保存、家庭创建及页面提交结束步骤，不记录邮箱、密码、令牌、请求体或响应体。最新源码 13/13、微信产物 12/12、类型检查、迁移静态检查及修改源码的定向 ESLint/Prettier 通过；原有构建警告保留。

结束旧真机调试，在工程 `dist/build/mp-weixin` 中重新编译并生成调试码；取消真机调试窗口中的“使用工具端的 Storage”，使用已注册账号登录。该选项是需要验证的调试桥接因素，尚未证实为故障根因。通过 Console 的最后一条步骤日志继续定位：

- 有 `[api] request.dispatched`，随后只有 `request.timeout`，而相同请求 ID 在后端已返回 201/200：继续查原生响应回调、调试桥接和客户端环境；不能据此认为数据库没有创建账号。
- 有 `[auth] session.save.start`，没有 `session.save.done`：停在同步存储调用，比较取消工具端 Storage 后的结果。
- 有 `[auth] family.create.start`：检查家庭接口的请求 ID、后端状态及响应步骤，注册成功与家庭创建成功分别判断。
- 有 `[auth-page] login.finished` 或 `register.finished`：提交流程已结束，结合错误提示或跳转结果验收。

广告调优 `recoverTuoguanOptimizeAd / invalid scope` 没有对应的业务调用；其具体触发原因尚未确认，不应把它当作注册失败的依据。实际手机响应接收、登录及家庭创建的完整验收仍待项目方反馈。

2026-10-07 实测：首次请求 8080 被拒绝；读取现有 `.env` 启动后端后，两项健康接口均 HTTP 200，未携带 token 的 `/api/v1/me` 返回 HTTP 401 `AUTH_REQUIRED`，表明 HTTP 服务和鉴权入口已响应。该检查不代替登录及业务功能验收。

若网络失败后紧接着出现 `_state.sent(...).useAgentStore is not a function`，旧版登录恢复的异常清理仍在运行。当前编译产物使用静态 `require("./agent.js")`；核对微信工具的实际项目目录，选择 `dist/build/mp-weixin`，通过“清缓存 → 全部清除”后重新编译。`dist/dev/mp-weixin` 则对应 `dev:mp-weixin` 的开发产物，两者需要与各自构建命令一致。

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

认证与 Agent 状态的微信编译产物回归：

```powershell
npm.cmd test
npm.cmd run type-check
npm.cmd run test:mp-weixin
```

`test:mp-weixin` 先重新构建，再用 Node 加载实际微信 CommonJS 产物和真实 Pinia，网络与存储使用测试替身。覆盖无 token 启动的两种模块加载顺序、登录失效、家庭切换、创建家庭及退出失败时的本地清理；编译产物测试不能替代微信开发者工具和真机验收。

### `useAgentStore is not a function`

2026-10-07 修复：`src/stores/auth.ts` 的 `await import('./agent')` 在当前 uni-app 小程序编译器中被生成为 `await "./agent.js"`，得到路径字符串而非模块，首次无 token 启动以及其他会话清理流程会失败。改为静态导入，并仅在 action 中调用 store，符合 [Pinia 组合 Store 的用法](https://pinia.vuejs.org/cookbook/composing-stores.html)。

旧产物的 7 项回归全部失败；修复后重新构建，7/7 通过。当前构建仍报告上游 `finally` 循环引用警告，以及 `stores/auth -> stores/agent -> stores/auth` 的 chunk 循环警告；两种加载顺序的产物测试通过，微信运行仍需人工确认。

微信开发者工具导入 `dist/build/mp-weixin` 后重新编译。若还显示旧调用栈，清除工具缓存后再编译，并验证首次进入、登录失效、创建家庭、退出后重新登录。截图中的 `WA*Service*.js` 预加载提示与本次 store 调用错误不同，不需改业务代码处理。

接口协议联调：

```powershell
node scripts/verify-auth-flow.mjs
```

复刻 `src/api/client.ts` 的协议行为打真实后端（需后端已启动），验证认证链路、401 刷新、跨家庭隔离、PATCH 降级等。结果同时写入 `scripts/auth-flow-report.md`。

## 迁移进度

见仓库根目录 `MeowHome-小程序迁移步骤.md`（§10 进度追踪 / §12 开发记录）。
