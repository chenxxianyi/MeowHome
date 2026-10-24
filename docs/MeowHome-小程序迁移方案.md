# MeowHome 小程序迁移方案

> **状态：待审核**
> **目标**：在保留现有 Web 端（`frontend/`）完全不动的前提下，独立开发一份视觉与交互一致的小程序
> **文档性质**：任务步骤与决策清单，供审核后再开工

---

## 1. 背景与目标

### 1.1 目标

1. 新增独立目录 `miniprogram/`，与 `frontend/` 并存，**互不干扰**
2. 小程序端与 Web 端**视觉、交互、数据一致**（同一套后端 API）
3. 现有 Web 端**零改动**：不改代码、不改构建、不改依赖

### 1.2 非目标（明确不做）

- ❌ 不抽取共享 npm 包、不建 monorepo、不改动 `frontend/`
- ❌ 不做原生小程序重写（成本过高，见 §2.1）
- ❌ 本期不做微信一键登录（列为待决策项，见 §11）
- ❌ 不做小程序端独有的新功能，只做等价还原

### 1.3 核心约束

| 约束 | 说明 |
|---|---|
| 代码双份 | 两份独立代码库，Web 端每次改动需手动同步到小程序端 |
| 后端共用 | 同一套 Go 后端，API 不变 |
| 上线门槛 | 微信要求 HTTPS + ICP 备案域名，**备案需 1–2 周，不占开发时间但卡上线** |

---

## 2. 技术选型

### 2.1 路线对比

| 方案 | 适配度 | 说明 |
|---|---|---|
| **uni-app (Vue3 + Vite)** | ⭐ **最高** | 与现有技术栈完全一致（Vue3 + `<script setup>` + Pinia + TS + Vite）。template/style 大部分可近乎原样搬移，Pinia 官方支持 |
| Taro (Vue3) | 中 | 同样支持 Vue3，但生态在本场景不如 uni-app 顺手 |
| 原生小程序重写 | 低 | 27 个 `.vue` 全部重写为 `wxml/wxss/js`，工作量最大（但包体与性能最优） |
| web-view 套壳 | 最低 | 直接嵌现有 H5。用不了原生能力、体验差，且**纯 web-view 小程序有审核政策限制** |

### 2.2 结论

**采用 uni-app（Vue3 + Vite + TS）**。

选型依据：本项目 27 个 `.vue` 文件中，template 与 style 占比远大于逻辑，且逻辑层已按 `api/` / `stores/` / `views/` 分层。真正需要重写的只有**路由、网络层、存储、平台 API**四块，集中度很高（见 §4.2）。

### 2.3 工程初始化

```bash
cd D:\MyProject\MeowHome
npx degit dcloudio/uni-preset-vue#vite-ts miniprogram
cd miniprogram
npm install
npm run dev:mp-weixin      # 产出 dist/dev/mp-weixin
```

> 微信开发者工具打开的是 **`miniprogram/dist/dev/mp-weixin`**，不是项目根目录。

---

## 3. 目录结构

```
MeowHome/
├── backend/                     # 不动
├── frontend/                    # 不动（Web 端）
├── docs/                        # 不动
├── demo/                        # 不动
└── miniprogram/                 # ★ 新增
    ├── package.json
    ├── vite.config.ts
    ├── tsconfig.json
    ├── index.html
    ├── project.config.json      # 微信开发者工具项目配置（appid 等）
    ├── scripts/
    │   └── sync-shared.mjs      # 从 frontend 同步共享层（见 §10）
    └── src/
        ├── main.ts
        ├── App.vue
        ├── pages.json           # 页面注册 + 原生 tabBar（替代 vue-router）
        ├── manifest.json        # appid / 平台配置
        ├── uni.scss
        ├── pages/               # 由 frontend/src/views/ 迁来（17 个）
        ├── components/          # 由 frontend/src/components/ 迁来（9 个）
        ├── stores/              # 由 frontend/src/stores/ 复制（9 个）
        ├── api/                 # 复制后重写 client.ts
        ├── styles/              # tokens / reset / global / pages
        ├── types/               # 直接复制
        └── static/              # tabBar 图标、字体文件
```

---

## 4. 现状盘点（实测数据）

> 以下数字均为对现有 `frontend/` 的实际扫描结果，非估算。

### 4.1 可原样复用

| 资产 | 实测 | 处理 |
|---|---|---|
| `types/index.ts` | 纯类型定义 | ✅ 原样复制 |
| `api/adapter.ts` | 无浏览器依赖 | ✅ 原样复制 |
| `stores/*.ts` | 9 个，Pinia | ✅ 基本原样 |
| `api/endpoints.ts` | 仅需换 import | ✅ 改动小 |
| `styles/tokens.css` | 67 行纯 CSS 变量 | ✅ 仅 `:root` → `page` |
| `styles/pages.css` | 330 类，**269 在用（82%）** | ⚠️ 搬 82%，删 61 个死类 |

**关键利好**：`tokens.css` 全用 `px`（实测 **800 处 px，0 处 rem/rpx**），且 WXSS 支持 CSS 变量。小程序中 `px` 即逻辑像素，375 设计稿可 **1:1 对上**，无需 px→rpx 换算（rpx 会随屏宽缩放，反而不利于「一模一样」）。

### 4.2 必须改造

| 项 | 实测数量 | 位置 |
|---|---|---|
| `vue-router` 引用 | **29 处** | 全部视图 |
| `axios` 引用 | **15 处** | 集中在 `api/` |
| `localStorage` | **9 处** | **全部在 `api/client.ts` 一个文件** |
| `<AppIcon>` | **71 处**（58 个图标） | 遍布所有视图 |
| `dvh` 单位 | 10 处 | 样式 |
| `document.*` 手搓 toast | 3 处 | `MedicalUploadView`、`TodayView` |
| `Suspense` + 动态 `<component :is>` | 各 1 处 | `App.vue`（仅 13 行） |
| `window.addEventListener` | 1 处 | `CatsView` 的 Esc 关闭 |
| `navigator.onLine` | 2 处 | `stores/app.ts` |

> `api/client.ts` 把浏览器专有 API **全部集中在一个文件**，改完它就解决了大半平台差异。这是现有代码结构最有利的一点。

### 4.3 顺带清理（搬移时不带过去）

| 项 | 实测 | 说明 |
|---|---|---|
| `vant` | 仅 `main.ts` 一行 CSS import，**零组件使用** | 死重 **264 KB**，占构建产物近一半 |
| `echarts` | **零引用** | 未使用依赖 |
| `vite-plugin-pwa` | — | 小程序无意义 |
| `mocks/` | — | 小程序端不需要 |
| 61 个死 CSS 类 | — | 含 `.offline-banner`、`.masthead-*`、`.desktop-*` 等 |

---

## 5. 改造清单

### 5.1 构建与工程

- [ ] uni-app 工程初始化，TypeScript 配置对齐 `frontend/tsconfig.app.json`
- [ ] 移除 `vant` / `echarts` / `vite-plugin-pwa`
- [ ] 配置 `manifest.json`（appid、`mp-weixin` 段）
- [ ] 配置 `pages.json`（页面注册、原生 tabBar、窗口样式）
- [ ] 配置 `project.config.json`（微信开发者工具）

### 5.2 路由（29 处）

| Web | 小程序 |
|---|---|
| `router.push('/cats')` | `uni.switchTab({ url: '/pages/cats/index' })` |
| `router.push('/cats/xxx')` | `uni.navigateTo({ url: '/pages/cat-detail/index?id=xxx' })` |
| `router.replace(...)` | `uni.redirectTo(...)` / `uni.reLaunch(...)` |
| `<router-link>` | `<navigator>` 或 `@click` + `uni.navigateTo` |
| `route.params.catId` | `onLoad(options)` 取 `options.id` |
| `route.query.catId` | 同上 |

- [ ] tab 页用 `switchTab`，非 tab 页用 `navigateTo`
- [ ] **注意：`switchTab` 不能带参数**，tab 间传参走 storage 或 Pinia
- [ ] 路由守卫逻辑迁移：现有 `beforeEach` 的认证与家庭校验，改为 `App.vue onLaunch` + 各页 `onShow` 检查

### 5.3 网络层（15 处）

- [ ] `api/client.ts` 重写：`axios` → `uni.request` 封装
- [ ] 保持对外函数签名不变（`request` / `requestEnvelope` / `saveTokens` 等），使 `endpoints.ts` 与 `stores/` 无需改动
- [ ] 保留 401 自动刷新逻辑（并发共享单次刷新，防令牌反复轮换）
- [ ] 保留 `X-Request-Id` 注入
- [ ] 超时配置：`AI_TIMEOUT` / `API_TIMEOUT` 从 `import.meta.env` 改读 `manifest.json` 或常量

### 5.4 存储与平台 API

- [ ] `localStorage`（9 处）→ `uni.setStorageSync` / `getStorageSync` / `removeStorageSync`
- [ ] `location.hash = '#/login'` → `uni.reLaunch({ url: '/pages/auth/index' })`
- [ ] `crypto.randomUUID()` → 时间戳 + 随机串兜底
- [ ] `navigator.onLine` → `uni.getNetworkType` + `uni.onNetworkStatusChange`
- [ ] `document.createElement` 手搓 toast（3 处）→ `uni.showToast`
- [ ] `document.getElementById`（`QuickRecordView`）→ 模板 `ref`
- [ ] `window.addEventListener('keydown')`（`CatsView` Esc 关闭）→ 直接删除

### 5.5 标签与样式（约 560 处）

实测标签使用量：

| 标签 | 数量 | 替换为 | 注意事项 |
|---|---|---|---|
| `<div>` | **324** | `<view>` | — |
| `<span>` | **112** | `<text>` | ⚠️ `<text>` 内**不能嵌套 `<view>`**，需重构嵌套结构 |
| `<button>` | **73** | `<button>` | ⚠️ 小程序 button 有**默认边框/背景/圆角**，必须 reset，否则全部走形 |
| `<label>` | 24 | `<label>` | 小程序原生支持，可直接用 |
| `<input>` | 20 | `<input>` | ⚠️ **原生组件**，层级恒在最上，会盖住 `position: fixed` 浮层 |
| `<h2>` / `<h1>` | 17 / 4 | `<view>` + 样式 | 无标题语义标签 |
| `<img>` | 7 | `<image>` | ⚠️ 需补 `mode` 属性，否则默认不裁剪、比例会变 |
| `<p>` | 7 | `<view>` / `<text>` | — |
| `<svg>` | 1（`AppIcon`） | 见 §6.1 | 🔴 小程序不支持内联 SVG |

样式：

- [ ] `dvh` → `vh`（10 处，WXSS 不支持 `dvh`）
- [ ] `tokens.css` 的 `:root` → `page`
- [ ] 清理 61 个死 CSS 类
- [ ] 删除 `AppShell.vue`（导航栏与 tabBar 由小程序原生接管）
- [ ] 删除 `BottomNav.vue`（改用 `pages.json` 的 `tabBar`）
- [ ] 逐个确认 `position: fixed`（4 处）在 `scroll-view` 内的行为
- [ ] 验证 `env(safe-area-inset-*)` 在当前基础库版本的支持情况

### 5.6 组件（9 个）

- [ ] `AppShell` / `BottomNav` — 删除（原生接管）
- [ ] `AppIcon` — 重做，见 §6.1
- [ ] `AppHeader` / `CatAvatar` / `CatSwitcher` / `AIEvidencePanel` / `AIResultBadge` / `AISummary` — 标签替换 + 样式适配
- [ ] 确认具名插槽 / `<Teleport>` / `<KeepAlive>` 的支持情况（`AppShell` 用了默认插槽，无问题）

---

## 6. 三大风险点

### 6.1 🔴 `AppIcon`：内联 SVG（最高风险）

`AppIcon.vue` 用 `<svg v-html="content">` 渲染 **58 个图标，被使用 71 次，遍布所有页面**。小程序 WXML 无 `<svg>` 标签，`v-html` 也不支持。

| 方案 | 还原度 | 代价 |
|---|---|---|
| base64 塞进 `<image>` | 高，但**丢失 `currentColor`**（无法随文字变色） | 71 处需逐个处理变色 |
| **图标字体（iconfont）** ⭐ | 高，**支持颜色** | 需生成字体文件 + WXSS `font-family` |
| 小程序原生 icon 组件 | 低（图标集不匹配） | 视觉不一致 |
| canvas 自绘 | 最高 | 过度设计 |

**建议图标字体**——唯一能同时保住「颜色跟随文字」与视觉还原的方案，独立工作量 2–3 天。

### 6.2 🟡 标签替换（约 560 处）

数量大但机械。主要坑：`<button>` 默认样式、`<input>` 原生组件层级、`<text>` 嵌套限制。建议写脚本做初筛，再人工核对边界情况。

### 6.3 🟡 路由模型差异

`switchTab` 不能带参；`navigateTo` 页面栈上限 10 层。现有依赖 URL 传参的页面（`QuickRecordView`、`CatDetailView`、`TrendsView`）需改造。

---

## 7. 后端改造

Web 端不受影响（同一套 API）。小程序上线需：

- [ ] **HTTPS + ICP 备案域名** — 微信要求合法域名必须 HTTPS、已备案、**不能是 IP、不能带端口**（现为 `http://127.0.0.1:8080`）
- [ ] **微信公众平台配置白名单** — `request` / `uploadFile` / `downloadFile`
- [ ] **CORS 无需改动** — 小程序**不发 `Origin` 头、不受同源策略约束**，`ALLOWED_ORIGINS` 仅对 Web 端有意义
- [ ] **开发期**使用微信开发者工具「不校验合法域名」开关连本地后端
- [ ] 若采用微信登录：`users` 表加 `openid` 字段 + 新增 `code2Session` 接口（见 §11）

---

## 8. 任务步骤

### 阶段 0：工程与共享层（2–3 天）

- [ ] 初始化 uni-app 工程，验证 `dev:mp-weixin` 能产出并跑通空白页
- [ ] 复制 `types/`、`api/adapter.ts`、`styles/`（含 61 个死类清理）
- [ ] `tokens.css` 的 `:root` → `page`，`dvh` → `vh`
- [ ] `api/client.ts` 重写为 `uni.request` + `uni.*Storage`
- [ ] **技术验证：选定并实现 `AppIcon` 图标字体方案**
- [ ] **技术验证：实测 uni-app 对 HTML 标签的容错边界**，确定是否需全量替换标签

> 阶段 0 的产出是「风险确认」，不是「功能」。

### 阶段 1：骨架与认证（3–4 天）

- [ ] `pages.json` 注册 5 个 tab 页 + 12 个普通页 + 原生 tabBar 图标
- [ ] 迁移 `stores/`（9 个）
- [ ] 迁移认证链路：登录/注册页、token 持久化、401 刷新、路由守卫等价逻辑
- [ ] 跑通：注册 → 登录 → 创建家庭 → `/me` 返回 familyId

### 阶段 2：5 个主 tab 页（5–8 天）

- [ ] 今日（`TodayView`）
- [ ] 记录（`RecordsView`）
- [ ] 猫咪（`CatsView`，含添加猫咪抽屉）
- [ ] 时光（`MomentsView`）
- [ ] 家庭（`FamilyView`）

每页验收：视觉与 Web 端一致、数据来自真实接口、交互可用。

### 阶段 3：12 个次级页（5–8 天）

- [ ] onboarding、快捷记录、AI 输入、AI 确认
- [ ] 猫咪详情、趋势
- [ ] 医疗上传
- [ ] 提醒
- [ ] 库存、账目
- [ ] 设置
- [ ] **`AppIcon` 71 处替换完成**

### 阶段 4：平台能力与真机适配（2–3 天）

- [ ] 原生 tabBar 图标与选中态
- [ ] `uni.showToast` 统一替换
- [ ] 网络状态监听
- [ ] 分享、下拉刷新
- [ ] 真机适配（不同屏宽、安全区、暗色模式可选）
- [ ] 包体检查（微信主包上限 2 MB）

### 阶段 5：上线准备

- [ ] 域名备案完成
- [ ] HTTPS 证书部署
- [ ] 微信公众平台配置合法域名
- [ ] 提审

---

## 9. 工时估算

| 阶段 | 内容 | 估时 |
|---|---|---|
| 0 | 工程 + 共享层 + 技术验证 | 2–3 天 |
| 1 | 骨架 + 认证 | 3–4 天 |
| 2 | 5 个主 tab 页 | 5–8 天 |
| 3 | 12 个次级页 + 图标替换 | 5–8 天 |
| 4 | 平台能力 + 真机适配 | 2–3 天 |
| **合计** | | **17–26 天（3–5 周）** |

> **修正说明**：早期口头估算曾为「两周左右」，按实测的 560 处标签替换与 71 处图标替换重算后，**3–5 周更接近真实**。大头集中在阶段 3 的机械替换。

备案（1–2 周）与开发并行，不占用上述工时，但**卡上线**，建议第一天启动。

---

## 10. 双端维护策略

因是完全独立目录，**Web 端每次改动都需手动同步到小程序端**（API 加字段、样式调整、bug 修复全部双份）。

**推荐做法**：在 `miniprogram/scripts/sync-shared.mjs` 放一个同步脚本，构建前从 `frontend/src/` 拷贝「无平台依赖」的部分：

| 可安全同步 | 不可同步（存在平台差异） |
|---|---|
| `types/index.ts` | `views/` |
| `api/adapter.ts` | `components/` |
| `styles/tokens.css` | `stores/`（部分含平台 API） |
| | `api/client.ts` |

这样 Web 端保持零改动，同时共享层不会漂移。**代价**：`views/`、`components/` 的改动仍需双份维护，这是「不抽共享包」这一约束的必然成本。

---

## 11. 待决策事项

| # | 事项 | 选项 | 影响 |
|---|---|---|---|
| 1 | **登录方式** | ① 沿用邮箱密码 ② 微信一键登录 | ②需后端加 `openid` 字段 + `code2Session` 接口，是**唯一需要动数据库**的改动 |
| 2 | **本期平台范围** | 仅微信 / 微信+支付宝+抖音 | 影响 `manifest.json` 与部分 API 选型 |
| 3 | **Web 端是否长期并行** | 长期双端 / 小程序上线后 Web 降级为内部 | 决定 §10 的同步策略是否值得投入 |
| 4 | **备案域名** | 是否已有可用备案域名 | 无则需先启动备案 |
| 5 | **图标方案** | 字体 / base64 | 阶段 0 验证后定 |

---

## 12. 验收标准

**功能等价**
- [ ] 5 个 tab 页与 12 个次级页功能与 Web 端一致
- [ ] 全部数据来自真实后端接口，无 mock 残留
- [ ] 登录 / 注册 / token 刷新 / 401 处理行为一致

**视觉一致**
- [ ] 逐页与 Web 端截图比对（375px 宽度基准）
- [ ] 颜色、字号、间距、圆角与 `tokens.css` 完全一致
- [ ] 图标视觉与 Web 端一致

**工程**
- [ ] `type-check` 通过
- [ ] 单元测试通过（可复用 Web 端测试思路）
- [ ] 微信主包 < 2 MB
- [ ] 真机（iOS + Android）无布局错位

---

## 13. 风险登记表

| 风险 | 等级 | 影响 | 对策 |
|---|---|---|---|
| `AppIcon` 内联 SVG 不可用 | 🔴 高 | 71 处图标，影响全部页面 | 阶段 0 优先验证字体方案 |
| `<button>` 默认样式导致视觉走形 | 🟡 中 | 73 处按钮 | 全局 reset 样式 |
| `<input>` 原生组件盖住浮层 | 🟡 中 | 抽屉内输入框 | 避免浮层内输入，或调整交互 |
| `<text>` 嵌套限制 | 🟡 中 | 112 处 span | 重构嵌套结构 |
| 备案周期不可控 | 🟡 中 | 阻塞上线 | 第一天启动 |
| 双端同步遗漏 | 🟡 中 | 两端行为不一致 | 同步脚本 + 改动清单 |
| `switchTab` 不能传参 | 🟢 低 | 少量页面改造 | 走 storage / Pinia |
| uni-app 版本兼容问题 | 🟢 低 | 个别 API 差异 | 锁定版本，阶段 0 验证 |

---

## 14. 建议的第一步

**先做阶段 0 中的「单页技术验证」**：只迁 `CatsView.vue` 一个页面，跑通「列表 + 添加猫咪」完整链路。

这一个页面即可暴露全部核心风险：`AppIcon`、`<button>` 默认样式、`<input>` 原生组件层级、`uni.request`、`switchTab` 传参。**验证通过再铺开 17 个页面**，比直接开工稳得多。

预估：0.5–1 天。
