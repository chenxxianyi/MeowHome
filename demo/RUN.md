# 猫宅 MeowHome H5 Demo — 运行说明

本目录为纯 H5 原型验证阶段产生的文件。此文档解释如何用任意静态服务器运行 demo。

## 方式一：Python 静态服务器（最简）

```bash
cd demo
python -m http.server 8080
# 浏览器访问 http://localhost:8080
```

## 方式二：Node 静态服务器

```bash
cd demo
npx --yes serve . -l 3000
# 浏览器访问 http://localhost:3000
```

## 方式三：Node 内置（无依赖）

根目录可执行：

```bash
node -e "const http=require('http'),fs=require('fs'),path=require('path');const MIME={'.html':'text/html','.css':'text/css','.js':'application/javascript'};const srv=http.createServer((q,s)=>{let p=decodeURIComponent(q.url.split('?')[0]);if(p==='/')p='/index.html';const f=path.join(process.cwd(),p);if(!fs.existsSync(f)){s.writeHead(404);s.end('404');return;}s.writeHead(200,{'Content-Type':MIME[path.extname(f)]||'text/plain'});fs.createReadStream(f).pipe(s);});srv.listen(8080,()=>console.log('http://localhost:8080'));"
```

## 路由控制

Demo 使用 hash 路由。桌面端可通过 URL 直接进入各页面：

```
http://localhost:8080/#today
http://localhost:8080/#records
http://localhost:8080/#cats
http://localhost:8080/#cat-detail?catId=cat-oran
http://localhost:8080/#trends?catId=cat-whit
http://localhost:8080/#reminders
http://localhost:8080/#moments
http://localhost:8080/#family
http://localhost:8080/#inventory
http://localhost:8080/#expenses
http://localhost:8080/#onboarding
```

移动端通过底部五栏导航切换（今日 / 记录 / 猫咪 / 时光 / 家庭）。

## 网络状态模拟

Demo 监听 `online` / `offline` 事件：
- 打开开发者工具 → Network → 切换 Offline，可看到离线提示。
- 快速记录在离线时保存为草稿（演示逻辑）。

## 说明

- 所有数据为开发 Mock，见 `assets/js/mock.js`。
- 首次打开默认进入今日首页；`#onboarding` 可查看五步引导。
- 本 demo 为 H5 验证阶段产物，验收后可迁移至 Vue 3 工程。
