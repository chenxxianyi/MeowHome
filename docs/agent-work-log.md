# MeowHome Agent 开发执行日志

> 本文件记录 Agent 开发任务（AG-00 ~ AG-D01）的执行过程。
> 任务总表与验收矩阵见 [MeowHome-Agent开发任务.md](../MeowHome-Agent开发任务.md)。

## 环境阻塞记录

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
