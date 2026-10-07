# MeowHome Agent 开发执行日志

> 本文件记录 Agent 开发任务（AG-00 ~ AG-D01）的执行过程。
> 任务总表与验收矩阵见 [MeowHome-Agent开发任务.md](../MeowHome-Agent开发任务.md)。

## 环境阻塞记录

### 2026-10-06 最终完整后端回归（真实数据库启用）

`backend/` 设置进程变量 `MYSQL_TEST_DSN=''`、`MYSQL_TEST_ENV_FILE=(Resolve-Path .env).Path`、`GIN_MODE=release`，`go test ./... -count=1 -v` 完整命令退出 0。所有包通过，`tests` 包耗时 55.190s，15 项顶层数据库用例全部实际执行，无 SKIP：11 项 Agent、3 项原有权限/审计与 1 项基础套件。每项独立测试库清理成功。最终 `go vet ./...`、`go build ./...` 和 Git diff 空白检查通过。

本轮新增七项 Agent 集成用例及显式 .env 数据库配置入口，配置模板、运行说明和任务状态同步。已按证据完成 AG-A01、AG-B04 与 V08/V09/V11；尚未调用真实供应商模型或操作微信工具/真机，阶段总体仍未完成。没有修改 `backend/.env` 或向业务库写测试数据、执行迁移/回滚。早先失败记录保留，最新完整验证已通过。

### 2026-10-06 AG-B04 聊天恢复与调度真实数据库验证

`go test ./tests -run TestAgentMySQLChatRouteInterruptedDraftRecovery -count=1 -v` 通过。真实 router/JWT/MySQL 覆盖处理中返回、同会话忙碌冲突、租约到期恢复原 session/turn、重试复用已保存 pending 草稿、完成答案幂等、旧 worker 拒绝提交、工具结果隐藏和续聊，无自动正式提醒。AG-B04 勾选。

调度/计划提醒命令曾因 Application Control 以及 Go 缓存访问被阻止，授权缓存访问后的原命令 `go test ./tests -run 'TestAgentMySQL(Scheduler|Scheduled)' -count=1 -v` 两项通过。零消息成功窗口、45 天前业务事件补录、事件失败游标不前移、重建调度器及写消息后游标未保存重放去重通过，V08 勾选；计划提醒窗口、NULL、done、软删除、跨家庭和同时间分页通过。各临时库均清理成功。

### 2026-10-06 AG-A01 数据库验收与并发确认

`agent_persistence_test.go` 的并发巡检、并发确认/写入回滚、同时间稳定分页三个用例通过。12 同键巡检请求只保存一条；普通提示最多三条，已忽略提示继续计额度；danger 仍可保存。12 同时确认只生成一条提醒和一条审计，返回同一 ID。提醒/消息写入失败回滚，确认后重复忽略不撤销提醒。

升级/回滚用例首次因测试的 information_schema 列别名错误失败，修正后单独重跑通过；验证旧提醒 NULL 计划时间、旧草稿 private 修正、空客户端键 NULL、字段元数据、007–014 回滚与重建，原家庭/提醒保留。勾选 AG-A01、V09、V11。新增调度/计划提醒边界用例首次被 Application Control 拦截，保持未验证并继续原命令重跑。测试库均清理成功。

### 2026-10-06 使用授权的 .env 完成首批 MySQL 实测

用户已授权使用 `backend/.env` 的数据库凭据。测试 helper 新增显式 `MYSQL_TEST_ENV_FILE` 入口，只读取数据库连接字段，不读取业务库名或加载 AI/认证配置；默认仍不自动连接。连接 MySQL 8.0.41，启用 STRICT_TRANS_TABLES、ONLY_FULL_GROUP_BY 等服务器现有 SQL mode。

在 `backend/` 设置进程环境 `MYSQL_TEST_DSN=''`、`MYSQL_TEST_ENV_FILE=(Resolve-Path .env).Path` 后执行 `go test ./tests -run TestAgentMySQL -count=1 -v`，4/4 通过，无 SKIP。空库 001–014、重复迁移、并发首轮幂等、私有草稿权限、审计触发器失败的原子回滚、重复确认、租约 fencing、增强状态竞争与真实 HTTP 鉴权均实际执行。四个由本次测试创建的临时库均记录清理成功；没有在业务库执行迁移或写入。

后续继续补齐迁移升级/回滚、稳定分页、巡检额度与并发确认；真实模型和微信交互仍未验收。
### 2026-10-06 调度器最终补测

`go test ./internal/infrastructure/scheduler -count=1` 单独重跑通过，最终各 Go 测试包均有通过记录；保留先前完整命令被策略拦截而失败的原记录。Go vet/build、小程序 13/13、类型检查、静态迁移检查、微信构建与 OpenAPI 引用检查通过。MySQL 测试全部 SKIP、真实 Provider/模型和微信交互未验证；任务总表不把这些外部验收标为完成。

### 2026-10-06 最终预算与检查

内部 HTTP 重试共享每轮四次请求预算，巡检增强只有一次 HTTP 请求；新增 Provider 模拟服务器测试通过。最终 Go vet/build 通过；最新全量测试 app/ai/config/router/tests 通过、scheduler 被 Windows Application Control 拦截，不能记为该次全量通过。MySQL 集成用例 DSN 缺失 SKIP。小程序测试 13/13、类型检查、静态迁移检查与微信构建通过；OpenAPI 116 个本地引用及新响应契约检查通过。真实 MySQL、真实模型和微信交互验收仍未完成。

### 2026-10-06 业务验收补充与交接

新增 24 条固定业务 fixtures 及受控数字、诊断/药量、无查询统计校验，模拟应用测试通过；小程序实际 store 的超时、运行中、重复点击、家庭切换、重新进入、旧响应及登录失效清理测试通过，总计 13/13。补充移除成员、删除猫咪/记录、聊天原文不能改为草稿、趋势失败时显示实际观测、增强超时与 worker 停止、开关矩阵下草稿行为。数据库测试 helper 只创建/清理本次唯一临时库；新增四项 MySQL 集成测试但 DSN 缺失全部 SKIP。Provider 后续测试被 Windows Application Control 拦截，不能宣称最新全量 Go 测试通过。新增 `docs/agent-runbook.md`，任务实现子项与接口契约同步，真实数据库/模型/微信验收继续保留未完成。

### 2026-10-06 巡检正文增强及灰度

新增迁移 014，保留规则正文、增强正文、状态、模型、Prompt 版本及 usage。规则消息先入库；单线程、队列 8 条、单条 5 秒异步增强，danger 不调用模型，读取不会重新调用。首版仅接受原规则正文加批准前后缀；自由改写、新诊断、药量、数字变化等拒绝并保留模板。版本/动作/忽略状态检查防止覆盖后续编辑，同消息只认领一次；进程中断的增强保留模板，不自动重新付费。环境配置增加家庭白名单，未纳入家庭禁止新 Agent 操作，但历史仍可读取/忽略；模型与增强双开关在 main 接线。增加脱敏日志与 request/turn/message ID。

验证：增强事实保护、danger、关闭开关、重复调用、并发忽略与灰度历史读取应用测试通过；`go test ./internal/platform/config -count=1` 单独重跑通过。加入日志后的全量内部测试中 app/ai 临时程序被 Windows Application Control 拦截，其他包通过，尚不能宣称该次全量通过。独立数据库、真实 Provider 与真机仍未验证。

### 2026-10-06 小程序聊天页面接入

猫管家页增加家庭巡检/本人对话切换、选猫、推荐问题、发送和重复点击保护、历史会话与消息分页、降级标识、同键恢复结果；证据及私有提醒草稿复用详情页。未保存修改禁止确认。草稿编辑统一明确显示北京时间并保存 Asia/Shanghai，修正设备时区与服务端时区可能不一致的问题。小程序 `request<T>()` 实际直接返回解包后的 T，已纠正文档 D15。`npm.cmd run type-check`、`npm.cmd test`（6/6）、`npm.cmd run build:mp-weixin` 通过；新增恢复合并、草稿版本与幂等键测试。微信开发者工具/真机交互待验证。

### 2026-10-06 接通真实聊天与持久化轮次

`Chat` 接入四工具循环，保存内部工具结果和 usage；上下文限制为五轮完成的问答。轮次元数据保存在用户消息中，事务认领与 35 秒租约避免同会话交错；同键处理中返回 running，完成后复用答案，过期租约可恢复。模型总预算 25 秒。模型故障返回真实查询状态及草稿恢复入口，不自动创建正式提醒。新增迁移 013；012 索引创建改为可重复执行。`go test ./internal/... -count=1` 通过，覆盖真实服务入口、上下文、幂等、降级、租约和旧 worker 拒绝提交。MySQL 并发、迁移和真实 Provider 尚未验证，小程序接入继续实施。

### 2026-10-06 聊天草稿权限修复

聊天草稿改为私有，会话创建者可编辑和确认；同家庭其他成员不能读取、编辑或确认。MySQL 确认事务补充相同权限检查，包含重复确认路径。应用层权限、家庭列表隔离、重复确认测试通过：`go test ./internal/app -run 'TestPrivate|TestAgentReminderTool|TestConfirmDraft' -count=1`。数据库事务仍待独立环境验证，B 阶段整体继续实施。


### 2026-09-25 环境核对（AG-00 前置）

- 工作区状态：`MeowHome-Agent开发任务.md`、`MeowHome-Agent设计方案.md`、`docs/agent-implementation-plan.md` 为未跟踪新文件，无其他未提交改动。
- Node：v24.19.0，可用。
- Go：`D:\tool\bin\go.exe` 存在但**无法执行**（Windows 组织 Device Guard 策略拦截，cmd/PowerShell/bash 均失败，权限读取正常）。
- 影响：AG-00 的 Go/后端验证部分（`go vet / go build / go test`）目前无法在本机执行；代码编写不受阻，但所有"通过"记录必须区分"仅人工/静态审查"与"命令实际运行"。

### 2026-09-25 环境修复（B 方案：用户目录安装官方 Go）

- 执行方式：下载官方 Go 1.22.12 zip 至用户目录并解压，完全绕开被 Device Guard 拦截的 `D:\tool\bin\go.exe`，不修改组织级策略。
- 安装位置：`C:\Users\chenxianyi\go`（GOROOT）；`C:\Users\chenxianyi\go-work`（独立 GOPATH，避免与 `$HOME/go` 冲突告警）。
- 代理配置：`go env -w GOPROXY=https://goproxy.cn,direct GOSUMDB=off`（proxy.golang.org 在本机网络不可达，goproxy.cn 可达）。该配置写入 `C:\Users\chenxianyi\go\env` 用户级文件，未修改项目文件。
- 基线结果（工作目录 `backend/`，命令实际执行）：
  - `go vet ./...`：通过（无输出）
  - `go build ./...`：通过
  - `go test ./... -count=1`：通过；仅 `tests` 包有测试文件（0.99s ok），其余包均无测试文件
- 遗留说明：MySQL 集成测试依赖 `MYSQL_TEST_DSN`，当前缺失时自动跳过，绿色测试不代表事务/索引/迁移已在真实 MySQL 上验证（任务文档 AG-00 第 4 条要求）。

## 执行条目

### 2026-09-25 AG-00：基线核对与测试环境

- 状态：完成（环境阻塞已通过 B 方案修复，基线全绿）
- 实际修改文件：
  - `backend/tests/test_helpers.go`：`applyMigrations` 改为调用新增的 `loadMigration(path, down)`（与 `cmd/migrate/main.go` 同一套截断逻辑），执行前剔除 `.up.sql` 中嵌入的 `-- +migrate Down` 段，并打印逐文件执行结果；保留 `MYSQL_TEST_DSN` 未设置时整体跳过的行为。
  - `backend/.env.example`：新增 `MYSQL_TEST_DSN=`（默认空）及注释，说明必须使用可销毁的独立测试库、切勿指向 `meowhome`。
- 实现说明：
  - 已通读设计方案、`ai_service.go`（规则解析与模板摘要，未接真实 LLM，对应 D01）、`care_service.go`、`reminder_service.go`（`Reminder` 仅存 `TimeLabel/Rule`，无绝对到期时间，对应 D06）、`scope.go`、`model/core.go`（`Base` 使用 ULID/varchar(26)，对应 D13）、`clock.go`（`FixedClock` 已存在，AG-A03 直接复用）。
  - 环境修复：`D:\tool\bin\go.exe` 被 Device Guard 拦截 → 下载官方 Go 1.22.12 至 `C:\Users\chenxianyi\go`（GOROOT）+ 独立 `GOPATH=C:\Users\chenxianyi\go-work`；`go env -w GOPROXY=https://goproxy.cn,direct GOSUMDB=off` 持久化到用户级 `go/env`，未改项目文件。
- 验证命令及工作目录（`backend/`，全部实际执行）：
  - `go vet ./...` → 通过（无输出）
  - `go build ./...` → 通过
  - `go test ./tests/... -count=1` → `ok github.com/meowhome/backend/tests 1.878s`（`MYSQL_TEST_DSN` 未设置，集成用例按设计跳过，非失败）
  - `go test ./... -count=1` → 通过；除 `tests` 包外各包无测试文件
- 结果：通过。MySQL 集成路径保持"跳过"，未连接 `meowhome` 业务库，未执行任何 Down 迁移。
- 与任务文档不同的实现及原因：
  - 测试库隔离采用"环境变量 `MYSQL_TEST_DSN` + helper 内建 `meowhome_test` 数据库"而非新建独立数据库用户，因当前 Windows 环境无 Docker，MySQL 为本地 127.0.0.1 安装，`root/123456` 权限已验证可用；在 `MYSQL_TEST_DSN` 指向 `meowhome_test` 而非 `meowhome` 的前提下满足隔离要求。
- 剩余问题 / 下一任务：
  - `go env -w` 的代理配置写入的是用户级 `C:\Users\chenxianyi\go\env`，新环境需重做（已记录在案）。
  - 进入 AG-01：冻结 DTO/契约（`agent_dto.go` + `types/agent.ts` + OpenAPI）。

### 2026-09-28 Codex 接续

- AG-01 已完成；后续 AG-A01～AG-A03 仍在实施。逐项状态、修改文件、实际命令和跳过原因已追加到 [任务文档第 12 节](../MeowHome-Agent开发任务.md#12-开发执行日志)。
- 本轮没有设置 `MYSQL_TEST_DSN`，没有对日常业务库执行迁移或回滚。Windows Application Control 偶发阻止 Go 临时测试可执行文件；通过的针对性测试与被拦截的运行已分别记录。
- 后续补充：小程序 `npm test` 通过 3 项、`npm run build:mp-weixin` 构建完成；Go app/scheduler 包测试通过。完整 Go 测试仍被 Application Control 阻止运行 `backend/tests` 包，详情见任务文档第 12 节。
- 2026-09-28 继续：AG-A06 已接入持久化时段进度与 danger 创建时间游标，迁移 012 尚未在独立 MySQL 测试库执行；固定时钟单元测试、Go vet/build 通过。AG-A04 去重键加入规则版本；真实并发额度仍待做。详情见任务文档第 12 节。
- 2026-09-30 继续：AG-A07/A08 的接口、真实路由测试代码、今日页、Agent 草稿与证据详情入口已接入。Go vet/build 与测试包编译通过；测试运行被 Windows Application Control 拦截，MySQL/微信实机验收未做。一次实际测试暴露的聊天幂等键复用问题已修复，修复后的运行验证仍待办。详情见任务文档第 12 节。
- 2026-09-30 补充：AG-A04 的非 danger 日额度改为家庭行锁事务内计数和创建，代码编译/vet 通过；真实 MySQL 并发验收未做。
- 2026-09-30 验证更新：稍后 `go test ./... -count=1` 实际执行通过，聊天幂等键修复和合法 Agent 路由链均通过；`backend/tests` 仍因无 `MYSQL_TEST_DSN` 跳过数据库部分。小程序 `npm test` 3/3 通过。
- 2026-09-30 B01：增加 Chat Completions function-tools Provider、Fake 接口与无密钥 `httptest`；针对性 Go 测试、vet/build 通过。真实模型与 B03 整轮预算待办，详见任务文档第 12 节和 [Provider 协议说明](agent-llm-provider.md)。
- 2026-09-30 B02：四个受控工具的独立执行层、白名单、家庭作用域、完整趋势分页及提醒草稿版本校验已实现；应用层针对性测试、vet/build 通过。会话初始化候选猫咪与歧义澄清需 B03 接入；MySQL/真实模型未联调。全量测试仅 AI 包临时程序被 Windows Application Control 拦截，详见任务文档第 12 节。
- 2026-09-30 B03 进行中：增加独立模型工具循环和 Fake Provider 测试，涵盖授权猫咪上下文、四次请求/八次调用/25 秒预算、重复调用缓存、无效输出和引用过滤；尚未接入 `Chat`，轮次持久化、历史和草稿恢复待做。详情见任务文档第 12 节。
- 2026-09-30 幂等索引修复：将 Agent 消息的可选 `client_message_id` 改成可空指针，聊天消息仍传实际键；迁移 008 将历史空串归一化为 `NULL`，巡检消息不再占用唯一索引。vet/build 通过；此修改后的 Go 测试进程两次被 Windows Application Control 拦截，MySQL 唯一键行为尚待独立库验证。
- 2026-09-30 B02/B03 补充：提醒工具按服务端轮次 ID 懒创建确定性草稿目标；同内容重试复用、不同内容冲突，确认前不创建正式提醒。针对性 app 测试通过；数据库并发验证仍待独立库。
- 2026-09-30 验证更新：全量 Go 测试除 scheduler 临时程序被 Application Control 拦截外其余包通过；单独重跑 scheduler 仍被拦截。Go vet/build 通过，MySQL 用例仍跳过。
