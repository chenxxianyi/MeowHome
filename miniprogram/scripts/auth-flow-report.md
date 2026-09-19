# 1.5 认证链路联调 —— 协议镜像验证报告

- 运行时间：2026-09-18T05:22:13.970Z
- 目标后端：`http://127.0.0.1:8080`
- 用例：**通过 35 / 失败 0**（100.0%）
- 请求数：27，刷新调用：3

> 说明：本报告由 `scripts/verify-auth-flow.mjs` 生成。它复刻 `src/api/client.ts` 的
> 协议行为打真实后端，**不覆盖** uni.request 本身与页面跳转（那部分需 DevTools 目视）。

| 分组 | 检查项 | 结果 | 实测值 |
|---|---|---|---|
| 注册 → 建家庭 → /me | POST /auth/register 返回 access_token | ✅ |  |
| 注册 → 建家庭 → /me | POST /auth/register 返回 refresh_token | ✅ |  |
| 注册 → 建家庭 → /me | access_token 段数 | ✅ | 2 (expect 2) |
| 注册 → 建家庭 → /me | 注册返回 user | ✅ | user.id=01M2SFE6R58W7RJQHZX2XFSVE0 |
| 注册 → 建家庭 → /me | user 字段名为 snake_case（与其余接口一致） | ✅ | "email,id,name" (expect "email,id,name") |
| 注册 → 建家庭 → /me | 注册响应未下发密码哈希 | ✅ | keys=id,email,name |
| 注册 → 建家庭 → /me | user.email 与注册邮箱一致 | ✅ | "mp-alice-1789708933867@example.com" (expect "mp-alice-1789708933867@example.com") |
| 注册 → 建家庭 → /me | user.name 与注册昵称一致 | ✅ | "爱丽丝" (expect "爱丽丝") |
| 注册 → 建家庭 → /me | 注册后 /me 可访问 | ✅ |  |
| 注册 → 建家庭 → /me | 注册后尚未加入家庭（family_id 为空 → 守卫应送去 onboarding） | ✅ | family_id=undefined |
| 注册 → 建家庭 → /me | POST /families 返回家庭 id | ✅ | id=01M2SFE6RKRRMGXBN378Q0HDCQ |
| 注册 → 建家庭 → /me | /me 返回 family_id 且已注入 | ✅ | "01M2SFE6RKRRMGXBN378Q0HDCQ" (expect "01M2SFE6RKRRMGXBN378Q0HDCQ") |
| 注册 → 建家庭 → /me | GET cats 可访问 | ✅ | length=0 |
| 令牌过期 → 自动刷新 | 已铸造签名合法但过期的 token | ✅ |  |
| 令牌过期 → 自动刷新 | 过期令牌下 /me 仍成功（说明走完了刷新+重试） | ✅ | id=01M2SFE6R58W7RJQHZX2XFSVE0 |
| 令牌过期 → 自动刷新 | 刷新被调用 1 次 | ✅ | 1 (expect 1) |
| 令牌过期 → 自动刷新 | 存储中的 access_token 已轮换 | ✅ |  |
| 令牌过期 → 自动刷新 | 存储中的 refresh_token 已轮换（后端单次使用） | ✅ |  |
| 并发 401 单飞 | 4 个并发请求全部成功 | ✅ | 4 (expect 4) |
| 并发 401 单飞 | 刷新仅触发 1 次（未被 4 个请求各刷一次） | ✅ | 1 (expect 1) |
| 刷新令牌失效 | 请求最终失败（未静默成功） | ✅ | TOKEN_EXPIRED |
| 刷新令牌失效 | access_token 已被清空 | ✅ | null (expect null) |
| 刷新令牌失效 | refresh_token 已被清空 | ✅ | null (expect null) |
| 刷新令牌失效 | familyId 已被清空 | ✅ | null (expect null) |
| 刷新令牌失效 | 已触发跳登录页 | ✅ | 1 (expect 1) |
| 跨家庭隔离 | 读他人家庭被拒 403 | ✅ | FAMILY_FORBIDDEN / HTTP 403 |
| 跨家庭隔离 | 读他人猫咪列表被拒 403 | ✅ | FAMILY_FORBIDDEN / HTTP 403 |
| 跨家庭隔离 | 鲍勃的家庭列表为空（未泄漏他人家庭） | ✅ | 0 (expect 0) |
| 非 Envelope 响应处理 | 未注册路径会抛错（不再把 404 文本当成功） | ✅ | code=HTTP_404 raw="404 page not found" |
| 非 Envelope 响应处理 | 抛出的是 HTTP_404 而非业务码 | ✅ | "HTTP_404" (expect "HTTP_404") |
| 非 Envelope 响应处理 | 对照：健康检查确实不是 Envelope（验证上面用例的前提成立） | ✅ | HTTP 200 |
| 阻塞项 #7：PATCH 覆盖头 | POST + X-HTTP-Method-Override: PATCH 能真正改到数据 | ✅ | name 爱丽丝的猫宅 -> 覆盖名 |
| 阻塞项 #7：PATCH 覆盖头 | 对照：原生 PATCH 返回 200 且真正生效 | ✅ | HTTP 200，name=原生改名 |
| 阻塞项 #7：PATCH 覆盖头 | 覆盖头只放行 PATCH：POST+override:DELETE 不会误删数据 | ✅ | HTTP 404，猫仍在=true |
| 阻塞项 #7：PATCH 覆盖头 | 结论：§11 #7 已解除（覆盖头可用） | ✅ | PATCH 降级链路打通 |
