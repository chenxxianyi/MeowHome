# 猫管家 Agent 运行与验收说明

更新日期：2026-10-07。任务状态以 [开发任务](../MeowHome-Agent开发任务.md) 和 [执行日志](agent-work-log.md) 为准。当前已实现聊天、四工具、草稿权限与确认、轮次恢复、小程序聊天、巡检增强及灰度；15 项独立 MySQL 集成用例与完整后端回归已通过；小程序会话恢复动态导入故障已修复并通过编译产物回归，真实模型和微信交互验收尚未完成。

## 配置与启动

在 `backend/.env.example` 查阅配置；保留现有 `backend/.env`，不要把密钥写进文档。Provider 复用 `AI_ENABLED / AI_BASE_URL / AI_API_KEY / AI_MODEL / AI_TIMEOUT / AI_MAX_RETRIES`，仅支持配置的供应商确实兼容 Chat Completions function-tools 的情况。参考 [Provider 说明](agent-llm-provider.md) 与 [OpenAI function calling 文档](https://developers.openai.com/api/docs/guides/function-calling)。

| 配置 | 行为 |
|---|---|
| `AI_AGENT_ENABLED=false` | 停止 Agent 聊天、工具、编辑/确认和巡检；历史查询、忽略仍可用 |
| Agent 开，`AI_ENABLED=false` | 规则巡检可用；聊天明确降级，不发外部模型请求 |
| Agent/AI 开，`AI_AGENT_LLM_ENHANCE=false` | 聊天可用；巡检显示原规则正文 |
| 三个开关均开 | 普通巡检允许受限异步增强；danger 永远使用原规则模板 |
| `AI_AGENT_ALLOWED_FAMILIES=` | 留空覆盖全部家庭；指定逗号分隔家庭 ID 时只允许这些家庭的新 Agent 操作，历史仍可读/忽略 |
| `AI_AGENT_PATROL_TIMES=08:00,20:00` | 按家庭时区触发，单实例每 30 秒扫描并补跑最近两个时段 |

配置在服务启动时读取，修改后重启。建议先限定一个测试家庭、关闭增强，再开启聊天；完成模型人工验收后再开增强。关闭增强并重启后立即读取原规则正文，保留增强快照用于排查；关闭全部 Agent 不撤销已经确认的提醒。

在 `backend/` 执行 `go run ./cmd/server`。main 已接线 Agent 仓储、Provider、健康档案、日志、调度和增强 worker，并在退出时取消和等待。需要可用 MySQL 及已迁移表；不要将无数据库的启动降级模式当作 Agent 可用状态。小程序在 `miniprogram/` 执行 `npm.cmd run build:mp-weixin`，微信开发者工具导入 `dist/build/mp-weixin`，进入猫管家页面。

## 迁移与独立数据库测试

按编号执行 `.up.sql`：Agent 原有 007–012，新增 013（轮次、usage、历史私有草稿修正）和 014（规则/增强正文）。012 索引补充重复执行保护。仓库迁移器会重放全部 Up 文件，没有迁移版本账本；应先在隔离副本验证重复迁移。001/002 文件内含 Down 段，不能把整个文件直接送给 MySQL 客户端。

`go run ./cmd/migrate -dry` 只打印解析后的 SQL；正式执行前核对环境目标与备份。`-down` 会逆序回滚全部 Down 文件，不能用于日常业务库。关闭 Agent 应通过配置完成。

通过进程环境设置 `MYSQL_TEST_DSN`，或明确授权后设置 `MYSQL_TEST_ENV_FILE` 复用指定文件中的数据库凭据。在 `backend/` 执行以下命令可选用本地 `.env`：

```powershell
$env:MYSQL_TEST_DSN = ''
$env:MYSQL_TEST_ENV_FILE = (Resolve-Path -LiteralPath '.env').Path
go test ./tests -run TestAgentMySQL -count=1 -v
```

helper 忽略 DSN 中的数据库名；文件入口只读取 MYSQL_HOST/PORT/USER/PASSWORD/TLS，进程中的同名变量优先，`MYSQL_TEST_DSN` 优先于文件入口。每个测试创建唯一的 `meowhome_agent_test_<ULID>`，并且只清理由该测试创建的库。账号需具备创建/删除测试库和创建 trigger 的权限。两个入口都缺失时明确 SKIP；显式配置后的连接或权限错误使测试失败，不跳过。设置环境变量后可运行 `go test ./... -count=1 -v` 验证全部后端包与数据库用例。

2026-10-06 连接 MySQL 8.0.41，15 项数据库用例全部通过（11 项 Agent、3 项原有权限/审计、1 项基础套件），无 SKIP，临时库全部清理。Agent 覆盖空库与重复迁移、已有数据升级、007–014 回滚重建、字段元数据、同时间消息/会话分页、软删除及权限隔离、12 请求并发去重/额度/确认、提醒/消息/审计写入失败回滚、调度重启和补录事件、提醒窗口、聊天中断与草稿恢复、租约旧 token 拒绝、增强竞争和真实 HTTP 鉴权。模型使用 Fake 验证契约；没有调用真实 Provider。

## 聊天和故障恢复

接口见 [API 契约](agent-api-contract.md)。只有四个工具：`listRecords / getCatProfile / getTrends / createReminderDraft`；身份来自服务器，不接受模型指定家庭或用户。提醒工具只保存私有 pending 草稿，用户核对猫咪、标题和未来时间后才能确认。

每轮执行最多五轮完成历史、四次请求（包含供应商内部 HTTP 重试）、八次工具执行、25 秒总预算；单次请求输出上限 1200 tokens。内部工具结果不直接返回客户端。输出过滤未知证据 ID，数值检查只从工具受控字段取值，记录备注不能授权数字；诊断/药量、无查询统计及由次数推断进食正常触发降级。此校验采用保守规则，会拒绝部分合法措辞，不能替代真实模型的语言和健康边界评测。

用户消息保存 running/completed/failed、turn ID、35 秒租约和客户端幂等键。pending 由事务直接认领为 running，不设独立等待队列。同键处理中返回 `status=running,message=null`；完成后返回原答案。另一个同会话轮次忙碌返回冲突。进程中断后租约到期允许同键恢复，旧 token 不能发布结果。小程序在超时后先查历史，再使用原 ID 重试；首轮丢失响应也能恢复原会话。重新进入页面从服务端用户消息恢复未完成请求。

模型失效前没有可靠查询时，返回明确不可用说明及记录/趋势入口；查询成功后失效，显示真实查询数量/观测、缺测说明和证据。已经保存的草稿仍在私有历史中，不因模型失败创建正式提醒或重复草稿。

## 巡检增强与观察

规则消息先保存，再由一个 worker 处理普通提示。队列容量 8，每条最多 5 秒、一次 HTTP 请求，一条规则消息只认领一次。增强仅允许原规则正文加批准前后缀；自由改写、新增事实、数字变化、诊断/剂量、额外字段等回退原文。规则正文、增强正文、模型、Prompt 版本、usage 分开保存；版本/动作/忽略状态变化后不发布旧增强。队列满或进程退出时保留模板，尚未实现持久队列和自动重新增强。

服务日志使用 request/turn/message/family ID 关联 `agent_chat_finished / agent_tool_finished / agent_draft_edit / agent_draft_confirm / agent_patrol_rule / agent_patrol_dedup / agent_patrol_finished / agent_enhancement_finished / agent_scheduler_failure`，记录固定错误类别、tokens 和耗时；不记录问题、备注、授权头或模型原文。数据库中的内部工具结果属于会话私有数据，不应作为公共日志导出。

## 验证记录与待验收项

- 应用层 24 个固定业务样例、聊天入口与草稿故障恢复、五轮上下文、数值聚合、租约和增强/灰度/开关矩阵已加入测试；Fake 的意图选择是脚本设定，只证明执行契约。
- 2026-10-07 小程序 13 项源码自动测试、12 项微信编译产物测试、类型检查、迁移静态检查、定向 ESLint/Prettier 与微信构建通过。产物测试使用 Node 加载实际 CommonJS 模块、真实 Pinia 和网络/存储测试替身；7 项覆盖会话清理，新增 5 项覆盖原生回调缺失、成功/失败计时器清理、迟到成功和刷新超时后迟到凭据隔离。此前 OpenAPI YAML、116 个本地引用与新聊天响应契约检查通过。
- 最新 `go test ./... -count=1 -v` 完整命令通过；数据库 15 项顶层用例全部执行，`tests` 包 55.190s，无 SKIP。`go vet ./...`、`go build ./...` 和 Git diff 空白检查通过。早先的 Application Control/缓存失败与修正记录保留在执行日志。
- 真实供应商/模型未调用，微信开发者工具/真机未操作；这些验收仍待完成。任务总表与验收矩阵只按实际证据勾选，数据库通过不代表阶段全部完成。
- 历史提醒 `scheduled_at=NULL` 不参与到期规则；草稿编辑页面明确采用北京时间（Asia/Shanghai）。未接入微信订阅消息或其他外部通知；调度只支持单实例，多实例领取锁另需实现。

人工验收按任务文档第 10 节逐项记录：进食查询与追问、多猫/时间歧义、证据点击、草稿修改保存/确认/忽略/过期、重新进入、客户端超时、家庭切换、三个开关、真实模型措辞。发布前需补齐相应人工操作证据。

## 小程序启动报错排查

`useAgentStore is not a function`：2026-10-07 定位为 `auth.ts` 中动态导入被当前编译器生成为路径字符串，已改成静态导入，保持在 action 中才创建 store。旧微信产物 0/7 通过，修复后 `miniprogram/` 执行 `npm.cmd run test:mp-weixin` 重建并回归，7/7 通过。类型检查及直接转译源码的测试不会验证小程序编译器如何改写动态导入，因此需要保留产物回归。

导入最新 `miniprogram/dist/build/mp-weixin` 后在微信开发者工具重新编译；旧调用栈仍出现时清除工具缓存再编译。当前构建有上游 `finally` 循环引用和 auth/agent chunk 循环警告，测试已覆盖两种模块加载顺序；实际微信运行仍待人工确认。首次进入、失效会话、创建家庭和退出后重新登录均需验收。

2026-10-07 连接排查：8080 初始连接被拒绝，随后使用本机可运行的 Go 安装及 `GOPATH=C:\Users\chenxianyi\go-work` 在 `backend/` 启动 `cmd/server`，按现有 `.env` 监听 8080。`/health/live`、`/health/ready` HTTP 200；未携带 token 的 `/api/v1/me` HTTP 401 `AUTH_REQUIRED`，HTTP 连接恢复。初次沙箱启动被 Go 缓存访问权限阻止，授权缓存访问后的启动成功；未修改 `.env`、未执行迁移。微信产物重新构建并回归 7/7 通过；工具里的实际项目目录与缓存更新仍需人工核对，不能把磁盘产物通过写成微信交互通过。

## 手机注册等待与本地数据库升级

项目方随后确认工具使用 `miniprogram/dist/build/mp-weixin`。手机注册“处理中”时，接口仍指向 localhost，后端没有观察到对应注册请求；已改为电脑 WLAN `http://192.168.1.13:8080`。手机与电脑需同一局域网，并在手机浏览器访问 `/health/live` 验证可达性。电脑访问 LAN 地址成功不代表真机请求成功，手机完整注册流程仍待验收。

普通请求使用 15 秒、刷新使用 10 秒原生与独立超时；计时器到期结束 Promise 并中止请求，迟到回调不更新结果或凭据，页面 finally 恢复按钮。5 项客户端产物回归由修复前 2/5 变为 5/5，合计产物测试 12/12；源码测试 13/13 和类型/迁移检查通过。

后端日志同时发现 `ai_agent_messages` 缺表。2026-10-07 只读核对确认 Agent 三表均不存在、提醒调度字段缺失；审查并预演现有 007–014 Up 脚本后，在当前本地开发库成功执行这 8 项升级。升级后三表均存在、提醒 3 个调度字段齐全，Agent 消息过滤查询成功。六张业务表升级前后行数一致：users/families/family_members 各 1，cats/daily_records/reminders 各 0；未执行 Down 或业务数据回滚。Agent 开关仍按现有配置读取，结构升级不代替模型或真机验收。

广告调优 `recoverTuoguanOptimizeAd / invalid scope` 的具体触发原因尚未确认，当前业务源码未接入对应广告调用；注册排查依据实际网络请求和后端响应，不能用该内部日志判定注册结果。

## 真机注册响应与后端持续运行：最新证据

2026-10-07 17:26:37，截图请求 `req-1791365198212-4a639a3a` 在后端记录为手机 `192.168.1.8` 的注册 POST，HTTP 201，约 25 毫秒。账号已创建；没有观察到后续家庭创建，完整前端流程仍未完成。此前“手机可达性待验证”由这项实际请求证据更新，后续应登录已有账号，避免重复注册。

随后用户报告网络超时；复查原后端进程已退出，LAN 健康接口无法连接。构建本地忽略目录中的 `backend/bin/meowhome-debug.exe`，通过隐藏的后台进程启动并保存 `server-*.stdout.log` / `server-*.stderr.log`，恢复后 LAN 健康接口 HTTP 200。未修改 `.env`、Agent 开关或真实用户数据。

小程序加入启动构建时间、认证请求回调、登录状态同步保存、家庭创建和提交结束日志，限定为固定步骤、路径、请求 ID、状态与耗时，不输出敏感请求/响应内容。新版构建、13 项源码测试、12 项微信产物回归、类型/迁移检查及定向 ESLint/Prettier 通过；微信回调、完整登录和家庭创建尚待实际手机确认。

人工排查：结束旧真机调试，重新编译并扫码，确认启动 `buildTime`，取消“使用工具端的 Storage”后登录。只有请求发送和超时而后端已成功时检查回调/桥接；停在 `session.save.start` 时检查同步存储；出现 `family.create.start` 时结合家庭接口日志检查。Storage 桥接是待验证假设，不能写成已确认根因。详细步骤见 [小程序 README](../miniprogram/README.md)。

## 19:33 真机超时与本地启动脚本

2026-10-07 新截图的操作时间为 19:33；后台日志在 **19:29:26** 已写入 `server stopped`，对应进程退出，LAN 健康检查无法连接，未找到该时段的实际手机注册/登录记录。当前证据指向后端已停止，停止的发起方未知；不能把后台启动当作永久服务。已恢复后端，LAN 两项健康接口 HTTP 200，本轮未改变小程序源码或数据库。

新增 `backend/scripts/start-local.ps1`，可在项目方自己的 PowerShell 中运行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File D:\Myproject\MeowHome\backend\scripts\start-local.ps1 -Restart
```

脚本使用本机 Go 与现有 `.env`，按完整路径识别并可停止本项目的旧 debug 实例，再在当前终端运行后端；保持窗口打开，Ctrl+C 结束。PowerShell 语法解析通过，直接执行因本机 Restricted 策略失败，进程级策略命令验证通过（已有实例提示分支），系统策略未修改。服务新启和两项健康检查通过，不将此写成手机完整登录通过。

vConsole System 截图是环境信息，未提供业务请求日志。手机浏览器访问 `http://192.168.1.13:8080/health/live` 验证当前可达性，再使用先前已注册的账号登录；仍失败时比对同次 Console 的步骤日志与服务器请求 ID。

## 登录状态 200 但前端回调未到达

2026-10-07 项目方提供 `req-1791373911756-b19d97e0`：前端 request.dispatched 后 15002 毫秒独立超时，未出现成功/失败回调；后续终端日志确认同 ID 登录状态 200、latency 0.0182501 秒。新请求 `req-1791375567033-35c00e1e` 为手机 `192.168.1.8`，状态 200、latency 0.019777 秒，前端仍 15003 毫秒超时，排除服务端慢处理。手机浏览器健康已成功，当前后端监听所有地址、电脑 localhost/LAN 健康均 200；旧日志属于已停止的后台实例，新请求看项目方当前终端。防火墙公用 TCP Allow 存在对应程序路径，未修改规则；端口过滤器只读查询被 Windows 权限拒绝，netsh/注册表读取成功。

旧全局 Console 入口不可见；随后模块自动入口又在手机触发 `startLocalNetworkDiagnostics is not a function`。当前诊断已内联到 `App.vue`，独立模块已删除，不再导入或求值全局诊断函数。先完成应用网络监听初始化，再以异常隔离启动探针。本地 HTTP 包在 onLaunch 自动运行一次，正式 HTTPS 不运行。四个只读健康探针比较 uni-json、uni-json-complete、wx-json、wx-text；只返回状态、请求 ID、阶段和耗时，不携带凭据或读写家庭业务。收集 summary 与服务端 latency 后，分别验证框架封装、complete 注册、JSON 解析或微信调试桥接，不将候选直接判定为兼容故障。

结束旧真机调试，在 `miniprogram/dist/build/mp-weixin` 重新编译并扫码。本轮 `buildTime` 为 `2026-10-07T12:26:55.333Z`，后续构建会更新。**在独立真机调试窗口**等待约 6 秒，提供 `[network-probe] summary`；缺失时提供最后一条探针日志，并通过后端相同 ID 的 client_ip 核对测试来源。

本轮微信构建、12 项产物回归、13 项源码测试、类型/迁移检查及定向 ESLint/Prettier 通过；回调类型的首次检查失败已修正，格式检查失败后重新格式化。未修改正式请求传输、数据库、.env、防火墙或私有调试设置；完整登录仍待实际手机探针结果。

自动诊断调整后，微信构建及 12 项产物回归、类型/迁移检查、定向 ESLint/Prettier 再次通过。实际失败是 Console 入口不可见，未据此认定网络层故障原因；仍需手机启动自动探针的结果。

2026-10-07 模块入口异常修正：新增编译 App 启动回归，以缺失导出的替身复现同名异常，旧产物 0/5；内联后 5/5，合计 17/17 产物回归通过，类型/迁移和定向 ESLint/Prettier 通过。当前 LAN 健康 200；没有修改正在运行的后端、数据库或防火墙。磁盘原模块确有导出，实际微信加载差异尚未查明；已移除该依赖，但实际手机诊断和登录修复验收仍待反馈。

20:27:59 四个健康探针均 200、101–108 毫秒，后端 client_ip 为电脑 `192.168.1.13`；项目方确认来自电脑模拟器。该结果不验证手机回调。微信工具同时报已删除诊断文件 ENOENT，当前源码和编译目录无其引用，尚未因模拟器成功更改正式登录传输。

项目方重启仍报 ENOENT 后，使用当前安装的微信 CLI 查明支持 reset-fileutils / cache。首次沙箱调用因 .cli 文件访问权限失败，授权调用后发现微信服务端口关闭；项目方已开启。刷新文件列表、清编译缓存后，实际 preview 编译仍报同样 ENOENT；继续清文件缓存、close/open 当前工程后，preview 成功，20:42:21 二维码保存于 `miniprogram/dist/preview/meowhome-preview.png`，微信包体 694379 字节。缺失文件构建已排除，普通手机预览登录和真机回调仍待验证。重复操作命令见 [小程序 README](../miniprogram/README.md)。

生成预览会上传构建产物到微信，首次自动审批因目标 AppID 归属/上传授权未明确而拒绝；项目方随后确认账号归属并明确授权临时预览，重试成功。未通过间接方式绕过拒绝，也未进行正式版本发布。
