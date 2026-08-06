# 猫宅 MeowHome 前端（Vue 3）

移动端优先的多猫家庭生活与健康管理系统前端。基于 H5 demo 高保真迁移到 **Vue 3 + TypeScript + Vite + Pinia + Vue Router + Vant 4 + ECharts + Axios + vite-plugin-pwa**。

## 技术栈

- **Vue 3**（Composition API、`<script setup lang="ts">`）
- **Vite 5** + **TypeScript**（严格模式）
- **Pinia** 状态管理
- **Vue Router** 分层懒加载路由
- **Vant 4** 移动端交互基础
- **Axios** 统一 API 客户端（业务 / AI 独立超时）
- **ECharts** 领域图表（迁移预留）
- **vite-plugin-pwa** 离线能力
- **Vitest + Vue Test Utils** 单元测试

## 安装与启动

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173
```

## 常用命令

| 命令 | 说明 |
|---|---|
| `npm run dev` | 启动开发服务器 |
| `npm run build` | 类型检查 + 生产构建 |
| `npm run preview` | 预览生产构建 |
| `npm run type-check` | TypeScript 严格检查 |
| `npm run lint` | ESLint 检查 |
| `npm run format` | Prettier 格式化 |
| `npm test` | 运行 Vitest 单元测试 |
| `npm run test:watch` | 测试监听模式 |

## 页面路由

| 路由 | 页面 |
|---|---|
| `/today` | 今日首页（默认） |
| `/onboarding` | 初始化与添加猫咪（5 步） |
| `/records` | 记录中心 |
| `/records/quick/:type` | 快速记录表单 |
| `/records/ai` | AI 自然语言输入 |
| `/records/ai/confirm` | AI 解析确认 |
| `/cats` | 猫咪列表 |
| `/cats/:catId` | 猫咪详情 |
| `/cats/:catId/trends` | 健康趋势（7/30/90 天） |
| `/medical/upload` | 病历上传与识别 |
| `/reminders` | 提醒中心 |
| `/moments` | 时光时间线 |
| `/family` | 家庭 |
| `/family/inventory` | 库存管理 |
| `/family/expenses` | 养猫支出 |
| `/settings` | 设置 |

## 目录结构

```
frontend/
├── public/
├── src/
│   ├── api/client.ts        # Axios 实例（认证、超时、request_id）
│   ├── components/
│   │   ├── app/             # AppShell、AppHeader、AppIcon
│   │   ├── ai/              # AIResultBadge、AIEvidencePanel
│   │   ├── cat/             # CatAvatar、CatSwitcher
│   │   ├── health/          # AISummary、HealthStatus
│   │   └── navigation/      # BottomNav
│   ├── mocks/               # Mock 数据（开发数据）
│   ├── router/              # 路由配置
│   ├── services/            # 业务服务（Mock Service）
│   ├── stores/              # Pinia 状态
│   ├── styles/              # tokens / reset / global
│   ├── types/               # 业务类型
│   ├── utils/               # 工具函数
│   └── views/               # 页面视图
├── tests/                   # Vitest 单元测试
├── .env.example
├── vite.config.ts
└── tsconfig.json
```

## Mock 数据切换

当前使用内置 Mock Service（`src/services/index.ts`）。后端完成后：

1. 复制 `.env.example` 为 `.env`，配置 `VITE_API_BASE_URL`。
2. 在 `services/index.ts` 中将 Mock 返回值替换为 `http` 请求。
3. 接口定义与返回结构保持一致，页面无需改动。

## 设计系统

- **Design Token**：集中在 `styles/tokens.css`（品牌色 `#C87345`、暖米色背景、低饱和状态色、4px 间距、圆角、阴影仅限弹层/导航）。
- **视觉规范**：温暖、安静、可信赖；禁止玻璃拟态、大面积渐变、霓虹、重阴影。
- **移动端**：底部五栏导航、iOS 安全区处理、触控目标 ≥44×44px。
- **桌面端**：底部导航隐藏，主内容居中（预留三栏布局）。
- **双猫隔离**：小白/小橘档案完全独立，切换器保证视觉清晰区分。
- **AI 可信**：所有 AI 结果带标识、依据、生成时间和免责声明；解析结果确认后才写入档案。

## 测试

```bash
npm test
```

覆盖：日期工具、Mock 数据完整性、业务服务、Pinia stores、类型形状等关键行为（23 个用例）。
