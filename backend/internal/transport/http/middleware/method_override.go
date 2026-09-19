package middleware

import "net/http"

// MethodOverrideHeader 是客户端用声明「真实 HTTP 方法」的请求头。
const MethodOverrideHeader = "X-HTTP-Method-Override"

// MethodOverride 把 `POST + X-HTTP-Method-Override: PATCH` 还原成 PATCH。
//
// # 为什么需要它
//
// 微信小程序的 `wx.request` **不支持 PATCH** —— 合法值只有
// GET/POST/PUT/DELETE/OPTIONS/HEAD/TRACE/CONNECT。而本项目有 3 个接口用 PATCH：
// 更新家庭、更新猫咪、完成提醒。小程序端因此降级为 POST + 覆盖头。
//
// # 为什么必须包在 engine 外面
//
// Gin 在**进入中间件链之前**就按 (method, path) 完成了路由匹配。若用 `engine.Use()`
// 注册，请求早已被判为「POST 无此路由」并返回 404，在中间件里再改 Method 已经来不及。
// 只有包在 `http.Handler` 外层、在 `ServeHTTP` 之前改写 `r.Method`，路由才会按 PATCH 匹配。
//
// # 为什么只放行 PATCH
//
// 不加限制的话，客户端可以用这个头把 POST 变成 DELETE —— 等于给所有写接口开了一条
// 绕过方法语义的后门，且 CORS 预检也拦不住（预检只校验头是否被允许，不校验值）。
// 这里只放行 PATCH，也就是 `wx.request` 唯一缺失的那一个方法，其余值一律忽略。
func MethodOverride(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.Header.Get(MethodOverrideHeader) == http.MethodPatch {
			r.Method = http.MethodPatch
		}
		next.ServeHTTP(w, r)
	})
}
