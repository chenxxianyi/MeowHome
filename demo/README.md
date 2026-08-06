
# 猫宅 MeowHome — H5 前端 Demo

移动端优先的多猫家庭生活与健康管理系统 H5 原型。纯原生 H5/JS 实现，用于在转向 Vue 3 工程前快速验证设计方向、信息架构与交互。

## 启动 Demo

使用任意静态服务器打开 `demo/index.html`：

```bash
cd demo
npx serve .        # 或 python -m http.server 8080
# 然后浏览器访问 http://localhost:3000（serve 默认端口）
# 或 http://localhost:8080
```

> 提示：请通过 HTTP 访问（便于验证 hash 路由），不要直接双击打开 `file://`。

## 包含页面

| 路由（hash） | 页面 |
|---|---|
| `#today` | 今日首页（默认） |
| `#onboarding` | 初始化与添加猫咪（5 步） |
| `#records` | 记录中心 |
| `#quick-record?type=feeding` | 快速记录表单 |
| `#ai-confirm` | AI 自然语言解析确认 |
| `#cats` | 猫咪列表 |
| `#cat-detail?catId=cat-whit` | 猫咪详情 |
| `#trends?catId=cat-whit` | 健康趋势 |
| `#medical-upload` | 病历上传与识别 |
| `#reminders` | 提醒中心 |
| `#moments` | 时光时间线 |
| `#family` | 家庭 |
| `#inventory` | 库存管理 |
| `#expenses` | 养猫支出 |
| `#settings` | 设置 |

## 目录结构

```
demo/
├── index.html              # 唯一宿主页面（hash 路由）
├── assets/
│   ├── css/
│   │   ├── tokens.css      # Design Token（CSS 变量）
│   │   ├── reset.css
│   │   └── pages.css       # 组件化样式
│   └── js/
│       ├── mock.js         # 开发 Mock 数据
│       ├── icons.js        # Lucide 风格线性图标系统
│       ├── router.js       # hash 路由解析
│       ├── store.js        # 全局状态
│       ├── api.js          # Mock Service（模拟真实 API 接口）
│       ├── ui.js           # 渲染辅助
│       ├── app.js          # 壳层 + 底部导航 + 渲染分发
│       └── pages/          # 各页面控制器
├── tests/                  # 说明文档（Node 冒烟测试）
└── tools/check_links.js    # 文件存在性检查
```

## 设计系统

- Design Token 集中在 `tokens.css`（颜色、字号、圆角、间距、阴影）。
- 视觉风格：温暖、安静、可信赖；禁止玻璃拟态、大面积渐变、霓虹、重阴影。
- 移动端底部五栏导航；桌面端为主内容居中布局（预留三栏响应式样式）。
- 猫名/药名/长文本支持截断与换行。
- 触控目标 ≥ 44×44px，iOS 安全区已处理（`env(safe-area-inset-*)`）。

## 数据与 Mock

- 所有 Mock 数据标记为“开发数据”，集中在 `mock.js`。
- `api.js` 提供与真实接口同构的 Mock Service（Promise、延迟、统一返回结构），页面不直接依赖静态数据文件。
- 猫咪双档案（小白/小橘）完全独立，今日首页通过顶部切换器清晰区分，杜绝视觉混淆。

## 验证

在项目根目录运行：

```bash
# 1. JS 语法检查
cd demo && for f in assets/js/*.js assets/js/pages/*.js; do node --check "$f"; done

# 2. Mock 数据完整性
node -e "global.window=global;require('./assets/js/mock.js');require('./assets/js/icons.js');... "

# 3. 链接 / 文件存在性
node tools/check_links.js

# 4. HTTP 冒烟（页面可访问、非空）
node -e "..."   # 见上方测试脚本
```

## 下一步：转为 Vue 3

当前为纯 H5 验证阶段。确认方向后，将按既定前端开发方案迁移到 Vue 3 + TypeScript + Vite + Pinia + Vue Router + Vant 4 + ECharts 工程，拆分组件（AppShell、CatSwitcher、ReminderItem、AISummary、TimelineItem 等）。
