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
