// 私有媒体读取：以登录态请求图片字节并转成 data URL。
//
// 采购截图只允许家庭成员访问，因此 <image> 不能直接指向接口地址
// （浏览器/小程序的 <image> 无法附加 Authorization 头）。这里用
// 带鉴权的 uni.request 取回字节，再转 data URL 交给 <image>。
import { apiBase, getToken } from './client'
import { ApiError } from './adapter'

// 微信小程序原生 API（条件编译块内使用，仅类型检查需要声明）。
declare const wx: { arrayBufferToBase64(buffer: ArrayBuffer): string }

const BASE64_TABLE = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/'

/** 手写 base64 编码（H5 端无 wx.arrayBufferToBase64 时使用）。 */
function manualBase64(bytes: Uint8Array): string {
  let out = ''
  for (let i = 0; i < bytes.length; i += 3) {
    const b0 = bytes[i] ?? 0
    const b1 = bytes[i + 1]
    const b2 = bytes[i + 2]
    out += BASE64_TABLE[b0 >> 2]
    out += BASE64_TABLE[((b0 & 3) << 4) | ((b1 ?? 0) >> 4)]
    out += b1 === undefined ? '=' : BASE64_TABLE[((b1 & 15) << 2) | ((b2 ?? 0) >> 6)]
    out += b2 === undefined ? '=' : BASE64_TABLE[b2 & 63]
  }
  return out
}

/** 由响应头推断图片 MIME；缺省按 PNG 处理。 */
function mimeOf(headers: Record<string, string> | undefined): string {
  const raw = headers?.['Content-Type'] || headers?.['content-type'] || 'image/png'
  return raw.split(';')[0]?.trim() || 'image/png'
}

/** ArrayBuffer → base64（微信小程序原生实现，H5 用手写编码兜底）。 */
export function bufferToBase64(buffer: ArrayBuffer): string {
  // #ifdef MP-WEIXIN
  return wx.arrayBufferToBase64(buffer)
  // #endif
  // #ifndef MP-WEIXIN
  return manualBase64(new Uint8Array(buffer))
  // #endif
}

/** 读取需要登录态的资源并返回 data URL。url 相对于 API 前缀。 */
export function fetchMediaDataUrl(url: string): Promise<string> {
  const token = getToken()
  return new Promise((resolve, reject) => {
    uni.request({
      url: apiBase + url,
      method: 'GET',
      responseType: 'arraybuffer',
      header: token ? { Authorization: `Bearer ${token}` } : {},
      timeout: 20000,
      success: (res) => {
        if (res.statusCode === 200 && res.data) {
          const mime = mimeOf(res.header as Record<string, string>)
          resolve(`data:${mime};base64,${bufferToBase64(res.data as ArrayBuffer)}`)
          return
        }
        reject(new ApiError(`HTTP_${res.statusCode}`, '读取截图失败', res.statusCode))
      },
      fail: () => reject(new ApiError('NETWORK_ERROR', '读取截图失败，请检查网络', 0))
    })
  })
}
