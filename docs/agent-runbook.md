# 猫管家 Agent 运行与验收说明

更新日期：2026-10-06。任务状态以 [开发任务](../MeowHome-Agent开发任务.md) 和 [执行日志](agent-work-log.md) 为准。当前已实现聊天、四工具、草稿权限与确认、轮次恢复、小程序聊天、巡检增强及灰度；15 项独立 MySQL 集成用例与完整后端回归已通过，真实模型和微信交互验收尚未完成。

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
- 最终小程序 13 项自动测试、类型检查、迁移静态检查与微信构建通过；其中 7 项 store 测试覆盖丢失响应、运行中状态、重复点击、家庭切换、旧历史响应、重新进入与登录失效清理。OpenAPI YAML、116 个本地引用与新聊天响应契约检查通过。
- 最新 `go test ./... -count=1 -v` 完整命令通过；数据库 15 项顶层用例全部执行，`tests` 包 55.190s，无 SKIP。`go vet ./...`、`go build ./...` 和 Git diff 空白检查通过。早先的 Application Control/缓存失败与修正记录保留在执行日志。
- 真实供应商/模型未调用，微信开发者工具/真机未操作；这些验收仍待完成。任务总表与验收矩阵只按实际证据勾选，数据库通过不代表阶段全部完成。
- 历史提醒 `scheduled_at=NULL` 不参与到期规则；草稿编辑页面明确采用北京时间（Asia/Shanghai）。未接入微信订阅消息或其他外部通知；调度只支持单实例，多实例领取锁另需实现。

人工验收按任务文档第 10 节逐项记录：进食查询与追问、多猫/时间歧义、证据点击、草稿修改保存/确认/忽略/过期、重新进入、客户端超时、家庭切换、三个开关、真实模型措辞。发布前需补齐相应人工操作证据。
