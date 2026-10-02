# MeowHome 猫管家 Agent 开发任务文档

> 版本：v1.0  
> 编制日期：2026-09-25  
> 设计依据：[MeowHome-Agent设计方案.md](./MeowHome-Agent设计方案.md)  
> 面向对象：在本仓库中实现功能的 AI 编程助手及代码审查者  
> 当前状态：AG-00、AG-01 已完成；阶段 A/B/C 尚未完成，详见第 12 节执行日志。

## 1. 目标、范围和执行约定

本任务文档按指定设计方案的顺序交付：**阶段 A 主动巡检与消息展示 → 阶段 B 智能问答 → 阶段 C LLM 增强巡检正文**。补充的准备任务用于解决当前代码与设计示例之间的差距。

首轮交付覆盖 Go 后端和 `miniprogram/` 小程序。现有 Web `frontend/` 保持接口兼容；新增 Web Agent 页面、微信外部通知、OCR、语音、向量检索、多 Agent 和新建健康记录工具不在本任务范围内。设计中的“推送”首轮落实为持久化消息和应用内展示，外部渠道另开任务。

此前的 [Agent 实施建议](./docs/agent-implementation-plan.md) 仅作为参考；执行本任务时保持原方案的四个工具与 A/B/C 顺序，不自动扩大到七个工具或重新安排功能优先级。

### 1.1 给开发 AI 的工作规则

1. 先读本任务文档、设计方案和目标文件，再按依赖执行。一次完成一个可验证的任务；任务完成后继续下一个已具备条件的任务，用户指定阶段时仅执行指定范围。
2. 开始前查看 `git status --short`，保留已有改动。不要覆盖其他人的代码，不顺带重构认证、库存、加密等无关模块。
3. 所有文件路径均相对于仓库根目录。文档列出的“新增文件”允许依据当前结构做小幅调整，但必须记录实际位置。
4. 先验证接口与业务规则，再写界面。依赖任务缺失时先补依赖，不能用伪造数据宣称集成完成。
5. 可使用 Fake LLM 和测试数据完成开发；真实模型配置、真实账号或设备验证缺失时，继续完成独立工作，并记录具体未验证项。
6. 修改配置模板 `backend/.env.example`；不要覆盖 `backend/.env`，不要在报告、截图或日志中输出真实密钥和连接串。
7. 数据迁移、回滚、自动清理仅在明确隔离的测试数据库验证，不对现有业务库运行全量回滚。正式部署不是完成开发任务的必要步骤。
8. 普通工程细节按本文件默认约定推进；只有改变产品范围、无法保持数据兼容或真正缺少外部必要信息时才提问。
9. 每项任务完成后，勾选总表及子项，记录修改文件、实际执行命令、结果、跳过项和剩余问题。`SKIP`、只编译成功和 Mock 通过不能写成完整集成通过。
10. 全部验收满足后才标记阶段完成；不要仅因为页面存在、接口返回 200 或测试文件写好就勾选。

### 1.2 全局完成标准

- 功能可从真实 API 路径到达，服务已在 `backend/cmd/server/main.go` 注入，配置已读取，页面已注册。
- 家庭、猫咪、会话、消息的权限在服务端执行，不能只靠前端隐藏入口。
- 正式提醒只通过用户确认创建，重复确认最多创建一次；Agent 不新增正式健康记录写入路径。
- 规则和工具有针对业务边界的测试，数据库事务与唯一约束有实际 MySQL 验证。
- 小程序至少通过类型检查、现有自动测试、生产构建；关键交互完成微信开发者工具或真机验证。
- OpenAPI、配置模板和本任务文档与最终代码一致。

## 2. 已核对的代码现状与统一约定

本节依据当前工作区源码，实施时需再次核对。这里明确解释原方案中的示例和冲突，避免不同开发轮次各自猜测。

| 编号 | 现状或设计歧义 | 本任务的执行约定 |
|---|---|---|
| D01 | `AIService.Parse / Summary` 是规则解析与模板摘要；AI 配置尚未注入真实 Provider | A 完全不依赖 LLM；B 新增并注入 Provider，不声称填配置就能启用 |
| D02 | 原文同时出现复用与不复用 `ai_sessions` | 现有真实表名是 `ai_parse_sessions`；新建 `ai_agent_sessions`、`ai_agent_messages`，保留旧表职责 |
| D03 | 消息示例缺少 `session_id`、`role`，无法保存多轮历史 | 为消息补上会话、角色、工具调用关联与可见性字段；不只保存助手答案 |
| D04 | 原文有直接调用提醒接口与专用 confirm 接口两条链路 | Agent 卡片统一调用专用 confirm；后端在事务内复用提醒校验及持久化逻辑 |
| D05 | “草稿不入库”与确认后恢复状态存在歧义 | 草稿可随 Agent 消息持久化；确认前禁止插入正式 `reminders` |
| D06 | `Reminder` 主要保存 `TimeLabel / Rule`，没有可计算到期时间 | 补充可空 UTC 到期时间、时区和完成时间；历史未补时间的提醒不参与到期判断 |
| D07 | 原文 AI 开关语义相互冲突 | `AI_AGENT_ENABLED` 控制 Agent；`AI_ENABLED` 控制模型使用；Agent 开、AI 关时确定性巡检继续 |
| D08 | 设计写“复用 JOB / FamilyRepo.ListAll”，当前未发现对应实现 | A 新增小型调度器和分页扫描能力，不能引用不存在的接口或一次加载全部家庭 |
| D09 | 原文日巡检时刻与周巡检时刻混在一起 | 家庭本地 08:00/20:00 唤醒；日规则按日期去重；R-04 只在周日 20:00；新 danger 记录走事件扫描 |
| D10 | 当前记录类型为 `feeding`，不是示例 `feed` | 工具沿用真实枚举；腹泻由 `elimination.payload.form` 的明确值识别，不统计所有排泄记录 |
| D11 | 原文只预加载近七天记录，却要求判断十四天未称重 | 分别查近七天事件和每只猫最近一次体重，不用七天窗口判断十四天缺测 |
| D12 | `CareService.Trends` 缺测默认零/normal，且查询有条数上限 | Agent 查询结果补充缺测、覆盖和截断信息；不能直接将展示默认值解释为健康事实 |
| D13 | `Base` 使用 ULID/varchar(26)，设计 SQL 使用 varchar(36) | 新表沿用现有 ID 类型，模型与 SQL 一致；明确实现 `TableName()` |
| D14 | `RequireAdmin` 等中间件存在占位实现 | 巡检入口使用已验证的登录身份，并在应用层查询成员角色，不以占位中间件作为权限保证 |
| D15 | 小程序 `request()` 返回带 `success/data` 的结果，路由有显式映射 | 正确解包；新增 `/agent` 映射、页面注册和参数处理，不直接照抄示例 |
| D16 | `Base.DeletedAt` 是普通指针字段 | Agent 仓储及所读数据显式排除软删除项，不能假设 GORM 自动过滤 |

### 2.1 固定的权限与草稿约定

- 家庭巡检消息为 `visibility=family`；本家庭现有成员可见。确认或忽略影响该家庭消息的共享状态，记录实际操作者。
- 用户对话为 `visibility=private`，只有会话创建者可读写；同家庭其他成员也不能访问聊天或聊天生成的草稿。
- 后台巡检使用系统执行身份，并明确限制家庭范围；系统消息允许 `user_id` 为空，`actor_type=system`，不能伪装成所有者进行交互式操作。
- 一个消息最多携带一份可确认提醒草稿，可同时携带多个导航建议。多个提醒必须拆为多条消息，避免一个 `action_status` 管理多次写入。
- 提醒草稿状态：`pending → confirmed / dismissed / expired`。另设消息展示状态，导航点击不改变草稿状态。
- 草稿保存绝对时间、时区、明确猫咪 ID 或明确家庭级范围、版本、有效期和确认后的提醒 ID。编辑增加版本；确认旧版本返回冲突。
- 不能将未识别猫咪自动转换为当前猫或 `both`。家庭级提醒必须由用户明确选择；单猫提醒必须验证猫咪归属。
- 缺少记录只描述为“未记录”；原记录的 danger 标记和规则等级均标明来源，不输出诊断或自行决定药量。

### 2.2 配置行为矩阵

| `AI_AGENT_ENABLED` | `AI_ENABLED` | `AI_AGENT_LLM_ENHANCE` | 预期行为 |
|---|---|---|---|
| false | 任意 | 任意 | 停止巡检，不调用模型；历史可读、可忽略；不允许新草稿确认 |
| true | false | 任意 | 执行确定性巡检；对话返回明确降级；已有有效草稿可确认 |
| true | true | false | 执行巡检和模型问答；巡检正文保持规则模板 |
| true | true | true | 问答可用，允许增强普通巡检正文；danger 正文保持规则模板 |

正文增强开关统一使用原方案的 `AI_AGENT_LLM_ENHANCE`。开关默认关闭，所有现有非 Agent 功能继续遵守既有配置。

## 3. 任务总表和依赖

所有任务初始未完成。原方案 A/B/C 的 2/2/1 周是参考排期，准备工作和现有实现缺口应计入实际估时。

| 完成 | 任务 ID | 任务 | 依赖 | 对应设计章节 |
|---|---|---|---|---|
| [x] | AG-00 | 基线核对与测试环境 | 无 | 全文 |
| [x] | AG-01 | 冻结 DTO、状态、接口和规则语义 | AG-00 | §2–5、§7、§9 |
| [ ] | AG-A01 | Agent 数据模型、迁移与仓储 | AG-01 | §4、§5.1 |
| [ ] | AG-A02 | 提醒到期时间与归属校验 | AG-01 | §3.3、§7 |
| [ ] | AG-A03 | 六条确定性巡检规则 | AG-A01、AG-A02 | §3.1、§7 |
| [ ] | AG-A04 | 巡检消息生成、证据与去重 | AG-A03 | §3.1、§5.2 |
| [ ] | AG-A05 | 草稿编辑、确认、忽略与幂等 | AG-A01、AG-A02 | §3.3、§5.2 |
| [ ] | AG-A06 | 定时调度、事件提示与启动接线 | AG-A04 | §3.1、§5.5 |
| [ ] | AG-A07 | 巡检、消息、操作 HTTP API | AG-A04、AG-A05、AG-A06 | §5.4 |
| [ ] | AG-A08 | 小程序入口、消息与操作卡片 | AG-A07 | §6 |
| [ ] | AG-A09 | 阶段 A 集成验收 | AG-A08 | §8.A、§11 |
| [ ] | AG-B01 | LLM Provider 和模拟 Provider | AG-A09 | §5.3 |
| [ ] | AG-B02 | 四个受控工具 | AG-B01 | §3.2 |
| [ ] | AG-B03 | 会话历史、工具循环及降级 | AG-B02 | §3.2、§5.2 |
| [ ] | AG-B04 | Chat、会话恢复接口与重试 | AG-B03 | §5.4 |
| [ ] | AG-B05 | 小程序完整对话交互 | AG-B04 | §6.1 |
| [ ] | AG-B06 | 阶段 B 权限、故障与对话验收 | AG-B05 | §8.B、§11 |
| [ ] | AG-C01 | 巡检正文 LLM 增强 | AG-B06 | §3.1、§8.C |
| [ ] | AG-C02 | 灰度、观测与阶段 C 验收 | AG-C01 | §9、§12 |
| [ ] | AG-D01 | 总体验收与交接文档 | AG-C02 | §11–13 |

执行顺序：`AG-00 → AG-01 → 阶段 A → 阶段 B → 阶段 C → AG-D01`。任务内实现及修复可迭代，阶段验收必须完成后再勾选阶段交付。

## 4. 准备任务

### AG-00：基线核对与测试环境

**修改范围：**必要时调整 `backend/tests/test_helpers.go` 及测试迁移辅助代码；记录写入本文件末尾的执行日志。

- [x] 阅读设计方案、现有 `ai_service.go / care_service.go / reminder_service.go / scope.go`、模型、仓储、路由、小程序请求和导航封装。
- [x] 记录已有工作区变更、Go/Node 版本、依赖锁文件和已有测试结果，不打印 `.env`。
- [x] 建立不访问外部模型的测试路径，复用 `platform/clock.FixedClock`。
- [x] 若启用 MySQL 集成测试，使用明确可销毁的测试库。现有 helper 会创建/删除 `meowhome_test`，执行前必须隔离，必要时改为专属测试库配置并验证目标。
- [x] 修正测试迁移解析差异：当前 helper 直接执行整个 `.up.sql`，而早期文件包含 `-- +migrate Down`；测试也须仅执行 Up 部分。
- [x] 记录基线失败和环境阻塞；可以继续无数据库的单元开发，但不能将 MySQL 验证标为通过。

**验收：**后续 AI 能依据日志分辨原有失败和新增失败；数据库测试不会连接业务库，也不会执行嵌入的 Down 段。

### AG-01：冻结开发契约

**修改范围：**`backend/openapi/openapi.yaml`、拟新增 `backend/internal/app/agent_dto.go`、`miniprogram/src/types/agent.ts`；必要的契约说明可放 `docs/agent-api-contract.md`。

- [x] 按第 2 节明确字段命名、消息可见性、草稿状态、错误码、开关和时间窗口，定义 DTO/Schema 后再实现业务。
- [x] `AgentMessage` 增加 `session_id? / role / visibility / cat_id? / evidence / action_suggestions / draft_version? / action_status? / generated_at / model`，禁止 API 暴露完整工具原始响应。
- [x] 证据支持 `source_type + source_id + cat_ids + occurred_at + excerpt`；规则 R-03/R-05 引用提醒，R-04 引用最近称重或建档起点，R-06 记录“该家庭昨日无记录”的查询范围。
- [x] 列表固定返回 `{messages, next_cursor}`，聊天返回 `{session_id, message, degraded}`，确认返回 `{reminder, message_id, action_status}`，外层仍使用现有响应 Envelope。
- [x] 定义 `AGENT_DISABLED / AGENT_UNAVAILABLE / AGENT_INVALID_OUTPUT / AGENT_DRAFT_EXPIRED / AGENT_CONFLICT` 等码及 HTTP 映射；超出权限的 ID 统一按项目策略处理。
- [x] 明确内容长度、查询范围、分页上限、每轮调用预算以及模型输出校验失败行为，写入 OpenAPI 和配置默认值。

**验收：**后端 DTO、小程序类型和 OpenAPI 的样例可逐字段对应；不存在列表返回数组与 `{messages}` 混用的问题。

## 5. 阶段 A：主动巡检与消息展示

### AG-A01：数据模型、迁移与仓储

**修改范围：**新增 `backend/internal/domain/model/agent.go`、`backend/internal/domain/repository/agent_repo.go`、`backend/internal/infrastructure/persistence/mysql/agent_repo.go` 和下一组空闲 migration；更新必要测试。

- [x] 创建原方案的 `ai_agent_sessions / ai_agent_messages`，沿用 ULID、审计字段和软删除风格（源码与迁移已具备；真实 MySQL 建表仍列在本任务验收待办）。
- [x] 会话保存家庭、创建者、标题、状态；消息保存会话、角色、可见性、猫咪/家庭范围、类型、正文、证据、动作、规则版本和生成时间。
- [x] 增加 `dedup_key` 唯一约束、草稿版本/有效期/已确认提醒 ID，以及聊天的 `client_message_id` 和工具调用关联字段。
- [x] 未设置去重键时使用 NULL，不能让空字符串唯一索引阻塞普通聊天。索引同时覆盖家庭消息列表和私有会话分页（索引迁移已写，MySQL 执行效果待验证）。
- [ ] 单条查询、分页、更新和软删除均要求家庭范围；私有消息再校验会话拥有者。使用 `(generated_at,id)` 等稳定游标处理同一时间多消息。
- [x] 提供事务句柄注入或仓储事务执行接口，供 AG-A05 同一事务创建提醒并更新消息（`ConfirmMessageAndCreateReminder` 使用行锁和事务；回滚与并发仍待 MySQL 验证）。
- [ ] 显式实现 `TableName()`，验证模型、迁移字段长度和可空性一致；模型名不要误生成 `agent_messages`。

**测试/验收：**空测试库可建表；已有数据升级不丢失；仅新增 migration 的回滚在隔离库通过；跨家庭 ID、软删除、同时间分页、去重并发有测试。

### AG-A02：可计算的提醒时间与归属

**修改范围：**`backend/internal/domain/model/core.go`、`app/reminder_service.go`、提醒仓储/接口、新增 migration、OpenAPI；保持旧客户端兼容。

- [x] 为一次性提醒增加可空 `scheduled_at`、`timezone`、`completed_at`；旧 `time_label` 继续用于兼容展示，不能作为到期判断依据。
- [ ] 旧提醒默认 `scheduled_at=NULL`。无法确定日期的文本不自动回填；相关规则跳过并给出可识别的数据不足状态。
- [x] 扩展创建/完成流程及 DTO，校验 RFC3339、时区和猫咪归属；确认创建后保存 UTC 和展示时区。
- [x] Agent 单猫提醒要求明确猫咪；明确家庭级提醒可兼容现有 `both`，由服务端转换，不能将漏填猫咪当家庭级。
- [ ] 新提醒的计划时间必须在未来；相对时间经服务端基准时间转换，含糊表达先补充，不默认为现在。
- [x] 复用创建校验逻辑并支持事务内调用，避免 AG-A05 复制出一套不同的提醒规则。

**测试/验收：**历史提醒仍能显示与完成；未来/过去/不合法时间、UTC 跨日、其他家庭猫咪、缺少猫咪都有确定结果；准确查到超期待办与七天内到期项。

### AG-A03：确定性规则 R-01～R-06

**修改范围：**新增 `backend/internal/app/agent_rules.go` 及测试，必要的受控聚合查询。

- [x] 将规则写成可注入时钟的确定性函数，输入不包含模型输出，输出携带 `rule_id / rule_version / scope / window / evidence / actions`。
- [x] 日期窗口使用家庭时区的本地午夜和下一日本地午夜，转换 UTC 查询，采用左闭右开；不要用固定 24 小时替代所有时区的下一天。
- [x] 单猫规则逐猫判断；多猫事件按记录关联展开，各猫内按记录 ID 去重；家庭级 R-06 单独处理。
- [ ] 对缺字段、软删除、无记录、分页截断采用显式分支，不能把错误当零条记录触发提示。
- [x] 提醒建议保留为未确认动作；没有明确时间时返回待填写字段，不编造医疗安排或精确时间。

**固定规则语义：**下表是把原方案条件转成可测试约定，不是新增临床阈值。

| 规则 | 条件与边界 | 输出与测试重点 |
|---|---|---|
| R-01 | 昨日本地日内，同猫 `severity=danger` 记录至少一条 | danger，引用原记录并说明原记录标记；warning-only 不触发 R-01，可由其他规则覆盖 |
| R-02 | 最近七个本地日（含当天截至 now），同猫同类呕吐或腹泻记录至少三条 | warning；呕吐按记录条数统计；腹泻仅 `elimination.form ∈ {稀便,水样}`；两条/三条和正常排便边界 |
| R-03 | medication 提醒仍为 todo，且 `now-scheduled_at > 24h` | warning；描述“超期未确认完成”，不是断言漏服；刚好 24h、done、未知时间均不触发 |
| R-04 | 周日 20:00 检查：距最近一次称重超过十四个本地日 | warning，明确是记录缺失；从未称重则按建档日计，新建档不足十四日不误报；恰好十四日不触发 |
| R-05 | vaccine/deworm 待办满足 `now ≤ scheduled_at < now+7个本地日` | info；已完成、恰好七天、时间未知不触发；使用实际提醒 type 映射并固定测试 |
| R-06 | 昨日本地日，家庭所有有效猫咪总记录数为零 | info；有猫且家庭建档覆盖完整昨日才参与；不能将数据库错误当无记录 |

**验收：**六条规则均有命中、不命中、边界和证据断言；不访问模型；重放同一上下文得到同一业务结论。

### AG-A04：巡检消息、证据与去重

**修改范围：**新增 `backend/internal/app/agent_service.go`，扩展记录/家庭仓储的必要查询，使用 AG-A01 仓储。

- [x] 实现指定授权家庭的 `Patrol`；近七天事件、最后称重、待办分开查询，不用七天内“没称重”推断十四天没称重。
- [x] 统计使用完整聚合或分页；当前 `ListByFamily` 有 500/1000 条上限，需要补齐查询能力，证据展示可限量但计数不能截断（记录与计划提醒按游标读取；消息额度使用数据库计数）。
- [x] 将规则结果变为 `rule-engine-v1` 消息，健康提示附固定免责声明，保存猫咪、规则版本、时间窗口与证据来源。
- [x] 用“家庭 + 猫咪/家庭范围 + 规则 + 版本 + 日/周/事件窗口”生成唯一键；唯一冲突按已生成处理，不能靠先查后写保证去重（代码和内存测试已覆盖，数据库唯一索引待独立测试库验证）。
- [x] 重跑不复活已忽略消息；新窗口允许按规则再次提醒。非 danger 提示每日最多三条，danger 不因普通消息额度被隐藏；按危险度排序（按家庭行锁在事务内计数和写入，真实 MySQL 并发验收待办）。
- [x] 当用户点击创建提醒时，从服务端消息生成/补全唯一草稿。若直接在消息中预置完整草稿，仍须符合 AG-A05 的版本和确认机制（现用同一消息的 `PATCH draft` 首次版本 0→1）。

**测试/验收：**重复巡检、并发巡检、忽略后重跑、超量记录及不同家庭/猫咪互不干扰；检查实际消息行数与来源引用。

### AG-A05：草稿编辑、确认、忽略

**修改范围：**新增 `backend/internal/app/agent_action_service.go`，Agent 与提醒仓储事务适配、审计。

- [ ] 实现创建/编辑提醒草稿，校验标题、猫咪范围、时间和期限，服务器保存规范化内容；建议有效期默认 24 小时且可配置。
- [x] `ConfirmReminderDraft` 从消息加载草稿；确认请求只提交预期版本，不重新信任任意 `family_id / user_id / payload`。
- [x] 再次校验登录身份、家庭成员资格、消息可见性、猫咪归属、Agent 开关、草稿状态、有效期和计划时间（应用层及事务内复核；真实数据库并发验证待办）。
- [x] 在同一数据库事务中锁消息行、创建提醒、写审计、标记 confirmed 并保存 reminder ID。所有仓储使用同一个事务句柄（源码路径已接通，原子回滚待 MySQL 验证）。
- [x] 已确认消息的重复请求直接返回同一提醒；编辑与确认并发通过版本检查失败；拒绝和过期不能确认（业务测试通过，数据库并发仍待验收）。
- [ ] `DismissMessage` 幂等；忽略已确认消息仅隐藏该消息，不撤销正式提醒。事务失败不留下孤立提醒或假成功状态。

**测试/验收：**双击/并发确认只创建一条；注入提醒写入或消息更新失败时整体回滚；无权用户、旧版本、过去时间及过期草稿被拒绝。

### AG-A06：调度、事件提示和启动接线

**修改范围：**新增 `backend/internal/infrastructure/scheduler/agent_patrol.go` 或同级实现，`config.go`、`cmd/server/main.go`、家庭分页扫描和持久化调度进度；必要新增任务状态 migration。

- [x] 实现 `AI_AGENT_ENABLED / AI_AGENT_PATROL_TIMES / AI_AGENT_LLM_ENHANCE` 的默认值、显式环境变量绑定和校验；测试读取实际环境变量，不只测试默认值。
- [x] 单实例 `time.Ticker` 负责唤醒，按家庭时区判断任务。日规则在默认 08:00 和 20:00 均执行，消息按日期去重；R-04 周日 20:00 单独执行（按计划时刻计算补跑）。
- [x] 持久化最后成功调度时点与失败状态，包括“扫描成功但零消息”；08:00 和 20:00 的执行进度分别记录。启动按策略补跑最近遗漏窗口，禁止反复全历史回放（迁移/持久化真实库验收待办）。
- [x] 新增家庭游标分页，忽略已删除家庭。任务以受限系统作用域调用巡检；用户 API 仍走用户鉴权。
- [x] 新增 danger 记录尽快生成事件消息：可用持久化变更游标的短周期扫描，初始扫描间隔 30 秒；扫描依据 `created_at + id`，不能用业务发生时间漏掉补录（真实库索引/游标待验证）。
- [x] 事件消息引用新增记录，按 record ID 去重；进度只在处理成功后前移，支持重启重放。这里表示应用内提示，不发送外部通知。
- [x] 限制单家庭执行时间 30 秒、并发和每批数量；单家庭失败记录并继续其他家庭。应用退出时取消并等待任务，避免 goroutine 泄漏。
- [ ] 首版默认单个调度执行实例；部署成多实例前启用数据库领取/锁保护，不能假定消息去重同时解决所有调度状态竞争。

**测试/验收：**固定时钟验证 07:59/08:00、周日、跨日、重启补跑、零消息进度；模拟一家庭失败不影响下一家庭；关闭开关停止新任务；事件补录不漏报。

### AG-A07：HTTP 接口与权限

**修改范围：**`backend/internal/transport/http/handler/agent.go`、handler 构造器、路由、错误映射、OpenAPI、`main.go`。

- [x] 实现第 8 节阶段 A 的列表、详情、编辑、确认、忽略和巡检入口；校验分页、过滤、请求体长度与版本（16 KiB Agent JSON 上限；真实路由测试待执行）。
- [x] 用户 API 放在现有 `/api/v1/families/:familyId` 认证组；逐资源检查家庭和消息可见性。
- [x] `/api/v1/internal/agent/patrol` 保留原方案路径语义，限制为请求中目标家庭的真实 admin，手动执行只针对该家庭并受限流与审计；应用层复核角色（进程内每来源 IP 每分钟 3 次，联调待验收）。
- [x] 后台调度直接调用受限服务方法，不通过公开 HTTP 模拟 admin；不使用 `RequireAdmin` 占位实现作为唯一校验。
- [x] 对齐 Envelope、状态码和小程序 method override；确认端点不得让客户端再直接调用普通创建提醒接口。
- [x] 通过真实 handler/router 链路测试注入成功，不能只测试 service 后遗漏路由注册（认证/越权/参数/限流与合法编辑→确认→忽略路径已执行通过；真实 MySQL 另属 A09）。

**验收：**未登录、非成员、普通成员调用 admin 巡检、跨家庭消息 ID 均正确拒绝；合法列表、编辑、确认、忽略路径完整运行。

### AG-A08：小程序入口、消息和卡片

**修改范围：**`miniprogram/src/api/endpoints.ts`、`types/agent.ts`、`stores/agent.ts`、`pages.json`、`utils/navigation.ts`、`pages/today/index.vue`、新增 `pages/agent/index.vue` 与 Agent 组件。

- [x] 新增 `agentApi`，沿用统一 client 和响应解包；GET 显式设置 method，聊天后续使用 AI 超时客户端。
- [x] 注册页面和 `/agent` 路由，支持 `messageId / catId / sessionId` 参数；使用 `useProtectedPage` 和家庭初始化状态。
- [x] 今日页增加管家消息区，默认展示优先级最高的两条，提供“查看全部”；加载失败不阻断今日页原有功能。
- [x] A 阶段完成 Agent 消息列表/详情页，对话入口显示尚未启用，不让今日页跳到空白未注册页面。
- [x] 证据卡片展示来源时间、猫咪和内容；原始记录通过新增按 ID 详情接口定位，趋势进入对应猫咪趋势页；查不到的目标明确显示失败（真机导航待验证）。
- [x] 操作卡片支持编辑猫咪、标题和具体时间，展示确认/忽略、处理中、失败、已确认和过期状态；后端成功后更新提醒 store（设备时区选时，真机流程待验证）。
- [x] 导航仅允许已定义动作，处理 query 与 tab 页参数传递，不依赖无法识别的 `/records?cat=...` 映射。
- [x] 切换家庭/登出清理 Agent 状态；通过请求序号与家庭标识阻止旧请求晚到覆盖新家庭界面。

**测试/验收：**今日页 → 消息 → 查看证据 → 编辑提醒 → 确认 → 提醒页看见真实条目；双击、断网、401、切换家庭和忽略状态都可验证。

### AG-A09：阶段 A 验收

**修改范围：**后端规则/仓储/HTTP 集成测试、小程序相关行为测试、本文件执行日志。

- [ ] 建立家庭 A/B、A 下两只猫、A 的 admin/member、B 的成员，以及第 10 节的规则数据集。
- [ ] `AI_ENABLED=false` 且 Agent 开启时，验证六条规则、家庭时区、持久化去重和提醒确认均正常。
- [ ] 模拟服务重启、重复事件、草稿并发确认和数据库失败；验证无需 LLM 的完整闭环。
- [ ] 按第 9 节执行后端和小程序检查，完成微信开发者工具/真机操作证据；真实设备未验证则保持该项未完成。
- [ ] 用现有 `/ai/parse`、`/ai/summary`、手工记录与提醒流程做针对性回归，确认新功能关闭时老功能仍可用。

**阶段完成条件：**家庭昨日有 danger 记录 → 自动生成带证据消息 → 今日页可见 → 编辑并确认草稿 → reminders 中仅新增一条；跨家庭不可见，重启不重复。

## 6. 阶段 B：智能问答

### AG-B01：LLM Provider

**修改范围：**新增 `backend/internal/infrastructure/ai/` 实现、应用层 Provider 接口与类型、Fake Provider；扩展 `config.go` 和 `main.go`。

- [x] 定义 `LLMProvider.ChatWithTools` 及请求/响应类型，保留原方案接口意图；补齐工具调用 ID、工具结果关联、usage、停止原因及协议所需上下文项。
- [x] 实现可配置的 Chat Completions function-tools 适配器，复用 `AI_BASE_URL / AI_API_KEY / AI_MODEL / timeout / retries`；依据官方接口文档核对协议形状，仅适用于确实支持该协议的供应商，真实密钥联调待办。
- [x] 模型启用且配置缺失时给出 `ErrLLMUnavailable`；Agent 未启用或仅规则模式时不因缺 API Key 阻断核心服务启动。
- [x] 限制 HTTP 超时、响应体大小和有限重试，支持 context 取消；不重试无效参数或权限错误，不在错误信息输出授权头。
- [ ] 初始采用非流式工具调用；全轮聊天预算默认 25 秒，低于当前小程序 AI 请求 30 秒与后端写超时，重试不能超出全轮预算。
- [x] 用 `httptest.Server` 和 Fake Provider 的契约接口验证普通回复、单次/多次工具调用、错误 JSON、429、5xx、超时及取消（真实供应商联调另列 B06）。

**验收：**没有真实密钥也能运行 Provider 契约测试；测试证明会请求配置中的地址/模型，工具调用结果可完整往返；日志不含密钥或全文病历。

### AG-B02：四个受控工具

**修改范围：**新增 `backend/internal/app/agent_tools.go`，必要的 `agent_query_service.go`，复用猫咪、健康档案、记录、趋势和草稿服务。

- [x] 注册且只注册 `listRecords / getCatProfile / getTrends / createReminderDraft`，分别定义 JSON Schema 与服务端参数校验。
- [x] `family_id / user_id` 来自认证后的服务端上下文；客户端/模型提交这些越权参数时不覆盖服务端范围。
- [x] 每次按家庭核验猫咪和相关资源，返回字段白名单；通过查询门面读取，不向模型暴露任意 SQL、密钥、仓储对象或家庭成员联系方式。
- [x] `listRecords` 默认七天，days 限制 1–90、limit 限制 1–50，合法类型沿用项目枚举；返回证据 ID、完整时间、截断信息。
- [x] `getCatProfile` 合并基础信息和已有 `CatHealthProfileRepo`，区分无档案与查询失败。
- [x] `getTrends` 输出指定指标的确定性统计、单位、缺测和覆盖信息。可复用 `CareService` 查询代码，但对 Agent 增加完整聚合及证据，不能把默认 normal/0 作为观测结果。
- [x] `createReminderDraft` 复用 AG-A05，返回可追溯消息/草稿 ID；工具调用结束前数据库中不得出现正式提醒。此处由无数据库仓储替身验证，真实 MySQL 事务仍待验收。
- [ ] 会话初始化时提供经授权的猫咪 ID/名称候选，让模型能选择现有 ID；同名或指代不明时返回澄清，不新增第五个工具来规避设计范围。

**测试/验收：**四个工具分别覆盖有效参数、边界、未知工具/字段、跨家庭 cat ID、缺数据和读取失败；确认前正式提醒数量不变。

### AG-B03：会话、工具循环与降级

**修改范围：**`agent_service.go` 或新增 `agent_chat_service.go`、会话/消息仓储、Prompt/Schema 版本化文件及测试。

- [ ] 新会话绑定当前家庭和用户；续聊校验拥有者，不能凭会话 ID 加载其他人的历史。切换家庭不能继续旧会话。
- [ ] 保存用户输入、助手工具调用、工具结果及最终答案；工具 ID 匹配。默认近五条可见对话加关联工具项，按 token/大小预算截断时保留完整调用对。
- [ ] 服务端注入当前时间、家庭时区、明确选择的猫咪候选、能力边界；记录备注仅作为数据，不拼入可信系统指令。
- [ ] 实现“模型请求 → 调用校验 → 工具执行 → 结果回传 → 下一次模型请求”，直到最终答案、澄清或预算耗尽。
- [ ] 每轮最多四次模型请求、八次工具执行、25 秒总预算；重试计入预算，重复相同调用可复用本轮结果；达到上限必须终止。
- [ ] 结果包含回答、引用、导航/草稿建议和降级标志。引用必须来自当前授权查询结果，未知 ID 不展示；统计值由确定性结果提供。
- [ ] 提示和校验共同禁止“没记录=没发生”“喂食次数相同=进食正常”。要求诊断或药量时保持原产品边界。
- [ ] 意图尚未解析且模型失败：返回不可用状态和固定记录/趋势入口；查询成功后生成失败：显示真实工具结果；禁止虚构“已查到 N 条”。
- [ ] 已持久化的草稿在模型后续失败时保留为 pending，并返回恢复入口；不执行、不伪装成失败回滚后消失，也不靠重试重复创建。

**测试/验收：**Fake Provider 覆盖无工具、一次工具、多轮工具、澄清、重复调用、非法参数、伪造引用、超时与生成中断；每个分支有停止条件及可恢复状态。

### AG-B04：聊天、历史和重试 API

**修改范围：**Agent handler、router、DTO、仓储以及 OpenAPI。

- [ ] 实现原方案 `POST /agent/chat`，支持 `session_id? / message / client_message_id`，明确文字长度和空输入处理。
- [ ] 同时实现会话列表及本人会话消息分页，支持重新进入页面后恢复，不只依赖 Pinia 内存。
- [ ] 同一会话对 `client_message_id` 唯一约束；同一消息重试返回原结果，处理中返回带 turn 标识的状态，不再次生成草稿。
- [ ] 首轮未携带 `session_id` 时，按 `family_id + user_id + client_message_id` 恢复已经创建的会话和轮次，避免首轮响应丢失后新建重复会话。
- [ ] 为每轮对话保存 `pending / running / completed / failed` 与关联结果。可通过会话消息的 turn 元数据实现；若独立建运行表，记录必要性并更新 migration/契约。
- [ ] 防止同会话并发轮次交错破坏历史；首版可串行化并对忙碌会话返回明确冲突。进程重启后孤立 running 轮次可识别为中断并重试。
- [ ] 稳定重试同一消息时，草稿创建以 `turn_id + 草稿序号` 去重；相同标识不同内容返回冲突。
- [ ] 模型 usage、耗时、错误原因记录为脱敏诊断信息，禁止返回密钥和内部堆栈给小程序。

**测试/验收：**真实路由测试创建会话、续聊、恢复、重复提交、同会话并发及跨用户读取；中断重试不生成重复提醒草稿。

### AG-B05：小程序完整聊天页

**修改范围：**扩展 A 阶段 `pages/agent/index.vue`、`stores/agent.ts`、API/类型和组件。

- [ ] 添加输入框、推荐问题、发送状态、失败重试、消息分页和会话恢复，使用稳定 `client_message_id`。
- [ ] 对话显示当前家庭/猫咪上下文；缺少猫咪时提供明确选择。发送后保留用户消息，失败标记可重试，不伪造助手成功答案。
- [ ] 正确处理客户端超时与后端处理中状态；先查询会话消息结果，再决定是否重发同一 ID。
- [ ] 复用 A 阶段证据和提醒卡片，更新草稿版本后再允许确认，页面返回后保持服务端真实状态。
- [ ] 模型不可用时显示真实降级状态和可用入口；家庭消息与本人聊天视觉上可区分、接口权限也独立。
- [ ] 增加切换家庭、登录失效、后台切回前台、连续发送和重复点击测试；旧请求不能串到新会话。

**验收：**用户询问“小白这两天吃得怎么样”得到基于真实记录的说明；继续追问可正确关联；创建提醒须看见草稿并确认；退出页面后能恢复。

### AG-B06：阶段 B 验收

**修改范围：**Agent 工具/服务/HTTP 测试，业务对话 fixtures，人工试用记录。

- [ ] 建立至少 24 条固定业务样例：八条查询/追问、六条多猫与时间歧义、四条提醒草稿、四条越权/注入、两条模型故障。
- [ ] 自动断言对象、调用工具、数值、引用和动作状态，不要求自然语言逐字一致；真实模型措辞和健康边界保留人工审阅。
- [ ] 覆盖同家庭他人的聊天、已移除成员、已删除猫咪/记录和旧证据引用，避免只测跨家庭 ID。
- [ ] 执行真实模型可选联调，记录 Provider、模型配置、日期和成功/失败样例；没有密钥时写“未验证”，不能用 Fake 成绩代表真实模型质量。
- [ ] 按第 9 节完成代码检查，重新验证阶段 A 未退化。

**阶段完成条件：**四工具闭环、多轮上下文、证据点击、提醒确认、权限和降级均通过；正式健康记录未被 Agent 自动写入。

## 7. 阶段 C：LLM 增强及交接

### AG-C01：巡检正文增强

**修改范围：**Agent 巡检增强服务、Prompt/输出 Schema、消息生成流程和测试。

- [ ] 仅在 `AI_AGENT_ENABLED && AI_ENABLED && AI_AGENT_LLM_ENHANCE` 时执行；danger 消息直接使用规则模板。
- [ ] 输入只包含命中规则、受控统计和必要证据；要求输出仅有正文，不接收可变 severity、evidence、actions、cat ID 或时间。
- [ ] 规则消息先可用，再对普通提示做有限预算增强；失败、超时或输出不合法时保留原文，不能阻塞规则提醒。
- [ ] 保存原规则正文、增强正文、模型及 Prompt 版本和 usage，便于回退和解释；读接口选择最终允许展示的正文。
- [ ] 限制字数和内容，禁止生成新增诊断、药量、未在事实中出现的数字或延期紧急事项。无法校验时回退模板，不仅靠免责声明。
- [ ] 增强异步完成时使用消息版本检查，不能覆盖用户已经确认/忽略后的相关状态；同一规则消息同一版本避免重复花费。

**测试/验收：**启用/关闭、danger、模型失败、添加事实、改写数字、超长返回、并发编辑均有测试；证据、动作、严重度在增强前后完全一致。

### AG-C02：灰度、观测与阶段 C 验收

**修改范围：**`config.go`、日志/指标、`.env.example`、测试与运行说明。

- [ ] 增加可选家庭白名单作为整体 Agent 开关下的灰度条件，后端强制生效；前端不得自行决定权限。
- [ ] 记录巡检成功/失败、规则命中、去重命中、模型调用/耗时、工具错误、草稿确认/冲突；日志脱敏并带 request/turn/message ID。
- [ ] 定义模型预算与增强任务上限，达到限制时保留规则正文；不要把读取消息变成每次重新调用模型。
- [ ] 按第 2.2 节逐格验证配置，含开关关闭后历史消息、草稿和既有核心功能行为。
- [ ] 小范围真实模型人工审阅，确认正文保留事实与健康边界；将不合格输出作为后续回归样例。

**验收：**可单独关闭正文增强并立刻回到规则正文；关闭全部 Agent 不影响手工记录和现有提醒；所有运行状态可从脱敏日志定位。

### AG-D01：总体验收与交接

**修改范围：**本文件、OpenAPI、配置模板、新增 `docs/agent-runbook.md`；必要的现有项目说明链接。

- [ ] 逐项完成第 10 节验收矩阵，核对总表与真实代码，不把未完成工作改名移出范围来宣称完成。
- [ ] 运行说明包含配置、启动接线、迁移目标与顺序、单实例调度限制、模型故障降级、灰度开启和关闭步骤。
- [ ] 明确列出历史提醒 `scheduled_at` 为空的处理、未接入外部通知、真实模型/真机验证状态，以及任何已知问题。
- [ ] 核对现有 `/ai/parse / ai/summary / reminders / records` 的兼容性，变更枚举或字段时保留旧客户端使用方式。
- [ ] 最终报告列出已完成任务、修改文件、测试命令及结果、未验证项、运行方式；文档和代码一起交付供审查。

**最终完成条件：**A/B/C 核心路径和数据库/权限测试均通过；真实模型与客户端人工验收有证据。若缺外部环境，明确交付为“代码完成，指定联调待验证”，保留相应复选框。

## 8. API 实施清单

除内部巡检外，下表路径统一前缀 `/api/v1/families/:familyId`。新增接口是对原方案所需编辑、历史恢复和详情交互的最小补充。

| 阶段 | 方法 | 相对路径 | 输入/结果与约束 |
|---|---|---|---|
| A | GET | `/agent/messages` | `type?, status?, limit?, before?`；家庭巡检列表，排除私有聊天 |
| A | GET | `/agent/messages/:messageId` | 单条详情；校验可见性 |
| A | PATCH | `/agent/messages/:messageId/draft` | 创建或编辑该消息的唯一草稿，带 `expected_version`；未创建时预期版本为 0，首次保存版本为 1 |
| A | POST | `/agent/messages/:messageId/confirm` | `{expected_version}`；服务端加载草稿，返回已有/新建提醒 |
| A | POST | `/agent/messages/:messageId/dismiss` | 幂等，成功 204；不删除已创建提醒 |
| A | POST | `/api/v1/internal/agent/patrol`（完整路径） | `{family_id}`；目标家庭真实 admin，有限流与审计 |
| B | POST | `/agent/chat` | `{session_id?, client_message_id, message}`；完成返回聊天结果，处理中返回可查询状态 |
| B | GET | `/agent/sessions` | 仅当前用户在本家庭的会话 |
| B | GET | `/agent/sessions/:sessionId/messages` | 私有聊天历史/轮次状态及草稿，游标分页 |

合同要求：

- 正常成功数据使用 `{code, message, data, request_id}`；小程序的 `request()` 已再转换为 `{success,data,requestId}`，不得重复或漏解包。
- PATCH 通过现有小程序 method override 通道，后端保留 Gin 外层 method override 处理。
- `before` 游标不允许跨家庭或绕过权限；过滤字段无法识别时返回参数错误，不默认为全部数据。
- 所有 Agent 导航使用预定义动作类型，不接受模型生成的任意 URL。
- 草稿字段不足时保留 pending 补充状态并禁用确认；导航动作不要求业务写入确认。

## 9. 验证命令与执行条件

以下是未来开发时应执行的命令。本次编制任务文档没有执行这些业务测试，也没有修改运行中的配置。

### 9.1 后端（工作目录 `backend/`）

```powershell
go vet ./...
go build ./...
go test ./... -count=1
```

开发过程中优先运行当前任务涉及的包/用例，阶段结束再跑上面的完整检查。格式化限于实际修改文件，最终确认这些文件通过 `gofmt`。

MySQL 测试前先完成 AG-00：配置专属可销毁测试库、确认迁移仅执行 Up，且测试连接和业务环境隔离。当前 `MYSQL_TEST_DSN` 缺失会跳过集成测试，CI 也尚未提供 MySQL 服务；因此绿色 `go test` 不自动证明事务、索引或迁移通过。

新增数据库验收任务需补齐独立测试库/CI 服务，或提供本地隔离库的实际日志。命令输出必须区分通过和跳过。不要直接对日常使用的 `backend/.env` 执行 `go run ./cmd/migrate -down`：现有工具会回滚全部 Down 文件。

OpenAPI 的 Makefile 校验当前是占位任务；实现时使用实际 schema 校验和契约测试并记录命令，不能把占位输出当接口验证通过。

### 9.2 小程序（工作目录 `miniprogram/`）

```powershell
npm run type-check
npm test
npm run verify:migration
npm run build:mp-weixin
```

按项目现有规范运行变更文件的 ESLint/Prettier 检查，阶段验收再执行 `npm run lint`；已有无关格式问题单独记录，不为使全仓库变绿而批量格式化。

`npm run verify:release` 仅在准备实际发布时执行；它依赖真实 HTTPS 地址、appid 和发布页清理，不是本地 Agent 功能开发的前置条件。依赖安装沿用现有锁文件和包管理器，不随意升级项目依赖。

构建成功不能替代页面操作验证：在微信开发者工具/真机完成第 10 节交互检查，记录设备/工具版本、日期、操作和观察结果。

## 10. 最终验收矩阵

测试数据必须人工构造到隔离环境，使用固定时钟，例如家庭时区 `Asia/Shanghai`、当前时间 `2026-09-27T20:00:00+08:00`（周日），不要修改真实用户记录制造规则命中。

| 完成 | 编号 | 场景 | 验收结果 |
|---|---|---|---|
| [ ] | V01 | 昨日一条 danger；只存在 warning 的对照组 | R-01 仅匹配既定条件，证据属于正确猫咪 |
| [ ] | V02 | 同猫七天两条/三条呕吐；正常排便、稀便和水样对照 | R-02 条数、类型和窗口正确，不把多猫数据相加 |
| [ ] | V03 | 用药到期刚好 24h、超过 24h、已完成、无绝对时间 | R-03 边界正确，不将缺确认说成没服药 |
| [ ] | V04 | 最近称重 13/14/15 天；从未称重的新/旧档案 | R-04 仅在周日时点按建档与称重语义触发 |
| [ ] | V05 | 疫苗/驱虫在 0/6/7 天到期与已完成 | R-05 左闭右开，无未知时间误报 |
| [ ] | V06 | 完整昨日无记录、有一条、昨日后才建家庭、查询失败 | R-06 不把新家庭或数据库故障当漏记录 |
| [ ] | V07 | 日巡检重跑、并发、忽略后重跑 | 唯一消息，不复活已忽略状态 |
| [ ] | V08 | danger 记录补录、扫描失败和重启 | 事件消息不漏、不重复，进度可恢复 |
| [ ] | V09 | 同一提醒草稿并发确认与请求超时重试 | 正式提醒只有一个，返回同一 ID |
| [ ] | V10 | 草稿编辑后旧版本、过期、拒绝、撤销成员资格 | 拒绝确认，不产生正式提醒 |
| [ ] | V11 | 消息更新或审计写入失败 | 提醒与消息事务整体回滚，不显示假成功 |
| [ ] | V12 | 家庭 B 的猫咪/消息 ID；同家庭他人的聊天 | 全部不可访问，错误不泄露内容 |
| [ ] | V13 | 普通成员调用内部巡检、伪造 family_id | 应用层角色及范围校验生效 |
| [ ] | V14 | 两猫同名、代词不明、缺猫咪、跨时区明早 | 澄清并展示具体对象和绝对时间 |
| [ ] | V15 | 一周只记两天、查询超过 1000 条、缺测日期 | 不补成实际零值，不谎报完整数据或健康正常 |
| [ ] | V16 | 模型未知工具、越权参数、伪造证据、备注注入 | 工具和输出校验拒绝，不越权执行 |
| [ ] | V17 | 模型多轮工具、无限循环、超时、错误 JSON | 调用对正确，预算耗尽停止，可识别降级 |
| [ ] | V18 | 聊天响应丢失、同消息重试、并发续聊 | 可恢复，不重复创建草稿，历史不交错 |
| [ ] | V19 | 页面切换家庭、登出、401、后台恢复、证据导航 | 不串家庭，返回正确页面并恢复真实状态 |
| [ ] | V20 | LLM 增强变更数字/风险/建议或模型故障 | 回退规则正文；证据与动作不变 |
| [ ] | V21 | 第 2.2 节全部开关组合 | 模型、巡检、确认和历史读取行为一致 |
| [ ] | V22 | 老 AI 解析/摘要、手工记录、普通提醒 | 既有业务兼容，无新增强依赖 |

## 11. 可直接复制给开发 AI 的提示词

### 11.1 按阶段开发

```text
请在当前 MeowHome 仓库实现猫管家 Agent。

先阅读：
1. MeowHome-Agent设计方案.md
2. MeowHome-Agent开发任务.md

本轮范围：AG-00、AG-01 和阶段 A（AG-A01 至 AG-A09）。
保留设计方案 A → B → C 的顺序，本轮完成 A 的规则巡检、消息展示与提醒确认闭环。
按任务文档第 2 节处理原方案中的接口、时间、开关和数据模型歧义。

先查看已有代码和 git 状态，再逐项实现。常规实现细节自行推进，保留用户已有改动。
使用 Fake 依赖完成无需真实模型的开发，不读取或输出密钥，不修改 backend/.env。
每项完成后执行相关测试，更新任务总表、子项和执行日志。
数据库测试只能使用确认隔离的可销毁测试库；集成测试跳过必须明确记录。
不要以 TODO、空实现、硬编码业务结果或 Mock 页面代替真实后端集成。
依赖缺失先补依赖；只有无法继续的外部条件才记录阻塞，同时完成独立工作。

最终给出：完成的任务编号、修改文件、测试命令与结果、未验证项、运行方式。
```

完成 A 后将范围改为 `阶段 B（AG-B01 至 AG-B06）`；完成 B 后改为 `阶段 C（AG-C01、AG-C02）及 AG-D01`。若要求全程开发，可将范围写为 `AG-00 至 AG-D01，按依赖持续推进`。

### 11.2 从断点接续

```text
继续 MeowHome-Agent开发任务.md 中的 Agent 开发。
先检查任务总表、最后执行日志和实际代码；完成标记仅在验收证据成立时有效。
从第一个依赖已满足的未完成任务开始，保持现有接口和第 2 节约定。
不要重做已验证任务，也不要回退其他人的改动。
更新本文件状态，明确报告实际通过、失败、跳过与外部待验证项目。
```

### 11.3 完成后审查

```text
请审查当前 Agent 实现是否满足 MeowHome-Agent开发任务.md。
逐条核对任务、真实调用路径、第 10 节验收矩阵及测试证据。
重点检查：家庭/私有会话权限、草稿原子确认与幂等、规则时间窗口、
缺测语义、模型工具循环、超时降级、调度重启、小程序切换家庭和真实路由。
识别仅有代码骨架但没有 main/router/config 接线的功能。
将问题按影响程度列出，标明文件位置、复现条件与需要补充的测试。
不要把跳过的 MySQL/真实模型/真机验证算作已经通过。
```

## 12. 开发执行日志

文档初建时仅完成任务拆解；以下按日期追加实际开发记录，未满足的验收项继续保持未完成。

### 2026-09-28 / Codex：AG-01

状态：完成。实际修改：`backend/internal/app/agent_dto.go`、`agent_service.go`、`backend/internal/transport/http/handler/agent.go`、`backend/openapi/openapi.yaml`、`backend/.env.example`、`miniprogram/src/types/agent.ts`；新增 `docs/agent-api-contract.md`。统一 `next_cursor` 必有、分页 1–50、聊天幂等键必填及输入长度，并记录证据、权限、时间窗、错误码和模型预算。固定默认值写在 Go DTO 常量和配置模板注释中，目前不提供环境变量覆盖。

验证命令及工作目录：`backend/` 执行 `go test ./internal/app ./internal/transport/http/... -count=1` 通过（transport 包无测试）；`miniprogram/` 执行 `npm.cmd run type-check` 通过，并用 Node YAML 解析器解析 OpenAPI 成功。跳过：真实 MySQL、真实模型、微信开发者工具；这些是后续任务的集成验收，不计入本任务通过范围。与任务文档不同之处：契约补充说明放在 `docs/agent-api-contract.md`；现有业务实现已先于任务勾选存在，后续按依赖重新核验。剩余问题 / 下一任务：AG-A01，核对仓储作用域、游标与事务及迁移实测。

### 2026-09-28 / Codex：AG-A01～AG-A03 持续实施

状态：进行中，三个任务总表均未勾选。实际修改：`backend/internal/domain/model/agent.go`、`backend/internal/domain/repository/repository.go`、`backend/internal/infrastructure/persistence/mysql/agent_repo.go`、`backend/internal/app/agent_service.go`、`agent_rules.go`、`reminder_service.go` 与对应测试；迁移 `008_agent_idempotency.down.sql`、`009_agent_message_linkage.up.sql`、`009_agent_message_linkage.down.sql`；同步 `backend/openapi/openapi.yaml`、`miniprogram/src/types/agent.ts` 和契约说明。修复分页游标跳过一条、仓储单条操作缺家庭作用域、私有消息只信任消息 user_id、事务行锁、提醒新计划时间允许过去一分钟等问题；提醒校验在普通提醒与 Agent 间共用。R-01 按猫展开并去重，R-02 仅识别明确稀便/水样，R-04 取每猫全历史最近称重且证据指向记录或猫咪建档，R-06 记录查询范围。

验证：`backend/` 的 `go build ./...`、`go vet ./...` 通过；`go test ./internal/app -run 'TestEvaluateRules|TestReminderValidation|TestAgentMessagePage' -count=1` 通过；`miniprogram/` 的 `npm.cmd run type-check` 通过。`go test ./internal/app` 在修改前通过，修改后一次执行遭 Windows Application Control 拦截临时测试程序；针对性测试随后通过。`MYSQL_TEST_DSN` 未设置，未运行迁移、索引、事务与回滚的真实 MySQL 测试，也未触碰业务库。剩余：补齐仓储软删除与完整数据分页、提醒到期查询及状态、规则全边界和高记录量测试；在明确隔离的测试库执行迁移/并发/事务验收后才能勾选 AG-A01～A03。下一步继续 AG-A01 的数据库验证条件和 AG-A02 的到期查询能力。

### 2026-09-28 / Codex：AG-A02～AG-A05 继续实现

状态：进行中，任务总表和阶段 A 验收仍未勾选。实际修改：`backend/internal/app/agent_service.go`、`agent_service_test.go`、`reminder_service.go`、`reminder_validation_test.go`、`agent_rules.go`、`agent_rules_test.go`，仓储接口和 MySQL 实现，Agent 模型/DTO、OpenAPI、小程序类型与 store；新增 `010_agent_reminder_schedule_index.up/down.sql`、`011_agent_message_display_status.up/down.sql`。到期提醒改为按计划时间区间和 `(scheduled_at,id)` 游标读取，501 条待办测试覆盖原 500 条截断；家庭消息日额度改为数据库计数。草稿确认要求未来绝对时间，状态更新同时检查旧版本与旧状态，避免过期请求覆盖已确认提醒。忽略已确认消息仅写 `display_status=dismissed`，保留正式提醒与 `action_status=confirmed`；确认事务内复核成员/猫咪并写审计日志。

验证：`backend/` 的针对性命令 `go test ./internal/app -run 'TestConfirmDraft|TestDismissConfirmed|TestStaleDismiss' -count=1` 通过，`go build ./...` 通过；`miniprogram/` 的 `npm.cmd run type-check` 通过。另一次完整 `go test ./... -count=1` 中 `internal/app` 与 `scheduler` 通过，`backend/tests` 的临时可执行文件被 Windows Application Control 拦截，因此完整测试失败，不能记为通过。Go vet 和 OpenAPI YAML 解析通过。用户已选择继续无数据库开发；MySQL 迁移、索引、并发、事务回滚仍未验证。后续需要完成 A04 去重键规则版本与并发额度、A05 真实 MySQL 验收、A06 调度持久化和事件扫描、A07/A08 真正接口与小程序闭环，再继续 B/C。

补充验证（同日）：`backend/` 的 `go vet ./...`、`go build ./...`、`go test ./internal/app ./internal/infrastructure/scheduler -count=1` 均通过；完整 `go test ./... -count=1` 再次只在 `backend/tests` 被 Application Control 拦截，非业务断言失败。`miniprogram/` 的 `npm test` 通过 3/3，`npm run build:mp-weixin` 构建完成；OpenAPI YAML 解析及 Agent 字段检查通过。微信开发者工具/真机操作未做，不能以构建替代交互验收。

### 2026-09-28 / Codex：AG-A04 去重键与 AG-A06 调度事件扫描

状态：继续实施；AG-A04、AG-A06 总表保持未完成。实际修改：规则键加入 `RuleVersion`；巡检唯一冲突后读取已存消息，不把未写入对象作为成功结果。新增 `012_agent_task_progress.up/down.sql`、`agent_task_progress_repo.go` 和 `(created_at,id)` 记录扫描；调度进度以家庭和窗口键分别保存成功/失败、零消息成功及事件游标。启动立即补跑最近两个到期时段，后续每 30 秒扫描；按原计划时间计算周日 R-04 和跨日窗口。danger 补录事件按记录 ID 去重，成功处理后才推进游标；家庭失败不阻塞其他家庭，退出时取消并等待任务。配置增加真实环境变量绑定测试，Go 固定时钟测试覆盖 07:59/08:00、重启、零消息、周日补跑、事件失败重放及补录。

验证（`backend/`）：`go test ./internal/app ./internal/infrastructure/scheduler ./internal/platform/config -count=1`、`go vet ./...`、`go build ./...` 通过。`MYSQL_TEST_DSN` 仍未设置，迁移 012 及真实库进度/索引未执行；多实例领取锁和 A04 普通消息并发额度仍待实现，故不勾选总任务。用户要求先继续无数据库开发；未操作业务库。下一步继续 AG-A07/A08 的接口与小程序闭环，并在独立测试库可用时补 A01/A05/A06 集成验收。

### 2026-09-30 / Codex：AG-A07/A08 代码路径与界面

状态：继续实施；AG-A07/A08 总表保持未完成。后端新增原始记录按 ID 详情接口（家庭成员权限与软删除过滤），Agent 消息详情返回服务端保存的草稿供恢复编辑；Agent JSON 请求限制 16 KiB，手动巡检按真实 admin 校验、每来源 IP 每分钟限 3 次并写审计。真实 router 测试代码覆盖认证、跨家庭、错误分页、method override、手动巡检拒绝及限流。小程序今日页接入前两条巡检消息，Agent 页实现列表、证据详情、编辑猫咪/标题/时间、确认、忽略、状态与过期显示，新增按 ID 原始记录详情页；家庭切换和登出清空消息状态，请求序号避免旧响应覆盖。OpenAPI 和契约同步。

本轮先执行 Go 测试时，`TestChatRecoversSessionWhenFirstResponseIsRetried` 实际暴露一个幂等键复用漏洞：仓储测试替身随机先返回助手消息，导致不同内容可复用同一键。已改为先固定查询用户消息并检查原文，随后重跑的临时测试程序连续被 Windows Application Control 拦截，**修复后的运行结果尚未取得**；新 router 测试也受同一限制。`backend/` 的 `go vet ./...`、`go build ./...` 和 `go test -c` 编译 app/router 测试包通过，只能证明可编译。`miniprogram/` 的 `npm.cmd run type-check`、`npm.cmd run build:mp-weixin` 通过；OpenAPI YAML 解析通过。未设置 `MYSQL_TEST_DSN`，未运行迁移或事务测试；微信开发者工具/真机尚未验收，故 A 阶段闭环不标完成。下一步需要恢复 Go 测试执行条件、独立 MySQL 验证 A01/A05/A06，以及实机确认 A08，再进入 B 阶段。

补充实现（同日）：AG-A04 普通消息每日三条额度改为 `AgentRepo.CreatePatrolMessage` 内锁定家庭行、统计本地日窗口并写入消息的单事务路径；danger 跳过普通额度。并发同规则唯一冲突仍回查已存行，额度满时若该规则已写入也返回原行。`go vet ./...`、`go build ./...`、app/router 测试包编译通过；MySQL 并发执行尚未验证。

验证更新（同日）：随后 Windows Application Control 允许执行 Go 测试；`backend/` 的 `go test ./... -count=1` 全部通过，包括此前失败的聊天幂等键用例、Agent router、Provider 和 `backend/tests`。`backend/tests` 因 `MYSQL_TEST_DSN` 未设置而跳过数据库集成用例，因此绿色结果不代表迁移/事务验收。`miniprogram/` 的 `npm.cmd test` 通过 3/3；此前本轮 `type-check` 与微信构建通过。

### 2026-09-30 / Codex：AG-B01 Provider 基础实现

状态：代码与无密钥协议测试完成；任务总表暂不勾选，整轮共享 25 秒预算需要 B03 工具循环落地，真实供应商兼容性尚未联调。新增 `backend/internal/app/agent_llm.go` 的请求/响应和 Fake Provider，`backend/internal/infrastructure/ai/chat_completions.go` 非流式工具调用适配器及 `httptest.Server` 测试；`main.go` 注入 Provider。适配器保留工具调用 ID、原始参数、结果关联、usage 和停止原因；限制一次调用及重试合计 25 秒、响应 1 MiB、最多两次重试。远程地址强制 HTTPS，本地回环允许 HTTP，禁止带授权头重定向，错误不输出密钥或正文。配置继续复用现有 `AI_*`，缺失时仅模型调用返回可识别错误，规则巡检不受影响。协议选择与限制见 `docs/agent-llm-provider.md`，依据官方 OpenAI Chat Completions/function calling 文档；其他兼容供应商和需 Responses API 的模型未验证。

验证（`backend/`）：`go test ./internal/infrastructure/ai ./internal/transport/http/router -count=1`、`go vet ./...`、`go build ./...` 均通过。无真实模型密钥，未调用外部模型；B02/B03/B04/B05/B06 和 C 阶段仍未实现或验收。

### 2026-09-30 / Codex：AG-B02 四个受控工具（独立调用层）

状态：四个工具的查询、参数校验与草稿动作已实现并通过应用层测试；总表暂不勾选。新增 `backend/internal/app/agent_tools.go`、`agent_tool_actions.go`、`agent_tools_test.go`，`main.go` 注入健康档案仓储。模型参数不接受家庭/用户 ID，执行范围由服务端鉴权上下文提供；猫咪按家庭复核，读取只返回白名单。记录默认七天并限制 1–90 天、1–50 条；趋势按 `(occurred_at,id)` 分页聚合全部匹配记录，报告缺测日期、无数值记录与证据截断；健康档案区分不存在和读取故障。提醒草稿走既有 `EditDraft` 和版本校验，测试证明确认前没有正式提醒。工具调用需要 B03 会话循环传入服务端草稿目标；候选猫咪已提供受权查询方法，但尚未注入会话初始化，指代不明的澄清也尚未落地，因此最后一项保持未勾选。

验证（`backend/`）：`go test ./internal/app -run 'TestAgentTool|TestAgentListRecords|TestAgentGetCatProfile|TestAgentGetTrends|TestAgentReminderTool' -count=1`、`go vet ./...`、`go build ./...` 通过。随后 `go test ./... -count=1` 中 app、scheduler、router、config、tests 等包通过，只有 `internal/infrastructure/ai` 的临时测试可执行文件被 Windows Application Control 拦截；单独重跑该包仍被拦截，属于执行环境限制，不能记为全量测试通过。`MYSQL_TEST_DSN` 未设置，未运行真实 MySQL 迁移/事务，也未用真实模型调用工具。

补充修正（同日）：趋势读取现在区分记录中明确的 `0` 和无数值字段，保留有效零值观测并将缺字段计入 `records_without_value`。`go test ./internal/app -count=1` 通过。

### 2026-09-30 / Codex：AG-B03 模型工具循环核心（尚未接入聊天入口）

状态：进行中，总表与子项暂不勾选。新增 `backend/internal/app/agent_chat_loop.go` 和测试，固定版本提示词并注入当前时间、家庭时区、授权猫咪 ID/名称；提示把记录与候选当作数据、歧义先澄清、不能把缺记录当作没发生或提供诊断/药量。独立循环通过 Fake Provider 验证一次工具调用、重复调用复用、非法参数反馈、引用 ID 白名单、无效模型输出和取消；最多四次模型请求、八个工具调用，整轮 `context` 限 25 秒。最终答案要求结构化 JSON，未知证据 ID 不进入返回的证据数组。

验证（`backend/`）：`go test ./internal/app -count=1` 通过。**该循环尚未由现有 `Chat` 路径调用**；工具调用/结果持久化、历史截断、草稿目标及故障恢复尚未实现，因此不能作为阶段 B 功能验收。下一步先建立可重试的轮次持久化，再连接聊天入口，避免中断时生成重复草稿。

补充修复（同日）：检查 AG-B04 幂等索引时发现 `ai_agent_messages.client_message_id` 在迁移中可为 `NULL`，但 Go 模型原先使用普通 `string`；巡检消息未设置键时可能写成空串，导致同家庭多条巡检消息触发唯一冲突。现将模型字段改为 `*string`，聊天用户/助手消息显式提供键，巡检及工具消息保留 `NULL`；内存仓储替身同步按非空键比较；迁移 008 在加唯一索引前把历史空串归一化为 `NULL`。`go vet ./...`、`go build ./...` 通过。此次修改后的 `go test ./internal/app -count=1` 两次均因 Windows Application Control 拦截临时可执行文件而未运行，不能记为测试通过；真实 MySQL 唯一索引行为还需独立测试库验证。

补充实现（同日）：`createReminderDraft` 在收到服务端轮次 ID 时才懒创建家庭可见草稿目标，目标主键由轮次 ID 确定性派生；同轮同内容重试复用 pending 草稿，不同内容返回冲突，正式提醒仍未创建。`runAgentTurn` 在工具返回草稿后更新本轮目标与版本。`go test ./internal/app -run 'TestAgentReminderTool|TestAgentTurn' -count=1` 通过；真实 MySQL 去重、并发和轮次落库仍待验收。

验证更新（同日）：`go test ./... -count=1` 中 app、AI 适配器、router、config、tests 等包均通过，scheduler 测试临时可执行文件被 Windows Application Control 拦截；单独重跑 scheduler 仍被拦截，因此全量测试不能记为通过。`go vet ./...`、`go build ./...` 通过。`backend/tests` 的数据库用例仍因缺少 `MYSQL_TEST_DSN` 而跳过。

每完成一个任务追加一条，不覆盖上一轮记录：

```text
日期 / 执行者：
任务编号：
状态：未开始 / 进行中 / 完成 / 阻塞
实际修改文件：
实现说明：
验证命令及工作目录：
结果：通过 / 失败 / 跳过（分别列出）
人工验证证据：
与任务文档不同的实现及原因：
剩余问题 / 下一任务：
```
