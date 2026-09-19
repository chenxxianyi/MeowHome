/**
 * 后端地址配置。
 *
 * ⚠️ 小程序**没有 Vite 代理**，请求必须直连后端，因此这里配的是绝对地址。
 * 三个场景的取值不同：
 *
 * | 场景 | 取值 | 前置条件 |
 * |---|---|---|
 * | 微信开发者工具模拟器 | `http://127.0.0.1:8080` | 需关闭「校验合法域名」（manifest.json 的 `mp-weixin.setting.urlCheck: false` 已设） |
 * | 真机预览 | 电脑的**局域网 IP**，如 `http://192.168.1.5:8080` | 手机与电脑同一网络；同样需关闭域名校验 |
 * | 正式发布 | 已备案的 **HTTPS** 域名，如 `https://api.example.com` | 必须 HTTPS、无端口、已在小程序后台配置 request 白名单 |
 *
 * 详见《MeowHome-小程序迁移步骤.md》步骤 5.1 / 5.2。
 */

/** 后端源站（不含 /api/v1）。改这里即可切换环境。 */
export const API_ORIGIN = 'http://127.0.0.1:8080'

/** 业务接口前缀（与后端路由 /api/v1 对应）。 */
export const API_BASE = `${API_ORIGIN}/api/v1`
