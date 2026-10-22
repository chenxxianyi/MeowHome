# MeowHome

MeowHome 是面向养猫家庭的照护记录工具，当前处于内测阶段。微信小程序是主要验收端；Vue H5 用于内部预览，静态 Demo 仅用于交互演示。

## 项目结构

- `backend/`：Go、Gin、GORM 和 MySQL API。
- `miniprogram/`：uni-app 微信小程序。
- `frontend/`：Vue 3 H5 预览端。
- `demo/`：静态交互演示，不连接权威业务数据。
- `backend/openapi/openapi.yaml`：当前 HTTP API 契约。
- `MeowHome-项目优化与迭代开发方案.md`：阶段顺序、发布门槛和迭代记录。

## 本地启动

需要 Go 1.22+、Node.js 20+、npm 和 Docker Compose。首次设置后端配置并启动 MySQL：

```powershell
Copy-Item backend/.env.example backend/.env
docker compose -f backend/docker-compose.yml up -d mysql
```

确认 `backend/.env` 中的 MySQL 地址和凭据与 Compose 配置一致，然后显式执行数据库迁移并启动 API：

```powershell
Set-Location backend
go run ./cmd/migrate
go run ./cmd/server
```

就绪检查会验证 MySQL 连通性及迁移版本。开发 API 默认在 `http://127.0.0.1:8080`；手机调试时还需把小程序 API 地址改为电脑局域网地址，并确保防火墙允许访问。

## 构建与测试

```powershell
Set-Location backend
go test ./...
go vet ./...
go build ./...
```

```powershell
Set-Location miniprogram
npm ci
npm run type-check
npm test
npm run build:mp-weixin
```

```powershell
Set-Location frontend
npm ci
npm run type-check
npm test
npm run build
```

MySQL 集成测试可设置 `MYSQL_TEST_DSN`，或设置 `MYSQL_TEST_ENV_FILE` 指向含测试 MySQL 凭据的文件。测试会创建独立临时数据库并在结束时删除；不要让测试账号指向生产数据库。未配置测试 MySQL 时，这些用例会显示为跳过。

## 数据库迁移

`go run ./cmd/migrate` 只执行迁移账本中尚未应用的版本。`go run ./cmd/migrate -down` 只回滚最近一次已应用版本；回滚前先备份并确认对应 Down SQL 的数据影响。

为已有数据库启用迁移账本前，先核对结构确实已到对应版本，再运行例如 `go run ./cmd/migrate -baseline 015`。baseline 只登记历史，不会执行 SQL；不要用它跳过尚未应用的迁移。迁移仍由显式部署步骤执行，服务启动和多实例扩容不会自动改表。

## 微信开发者工具

从 `miniprogram/` 构建微信包，开发者工具导入 `miniprogram/dist/build/mp-weixin`。需要真实微信能力或通知验收时，按对应迭代文档在开发者工具和真机记录结果；本地构建成功不等于微信发布验收完成。
