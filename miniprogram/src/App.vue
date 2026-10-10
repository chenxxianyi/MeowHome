<script setup lang="ts">
import { onLaunch } from '@dcloudio/uni-app'

import { API_BASE } from './api/config'
import { useAppStore } from './stores/app'

type ProbeRequestOptions = UniApp.RequestOptions & {
  success: NonNullable<UniApp.RequestOptions['success']>
  fail: NonNullable<UniApp.RequestOptions['fail']>
}

type RequestFn = (options: ProbeRequestOptions) => UniApp.RequestTask

declare const wx: { request: RequestFn }

interface ProbeResult {
  probe: string
  requestId: string
  phase: 'success' | 'fail' | 'deadline' | 'threw'
  elapsedMs: number
  status?: number
}

// 诊断内联到启动模块，避免远程 Console 上下文或独立诊断模块的导出影响启动。
const healthOrigin = API_BASE.replace(/\/api\/v1\/?$/, '')
let diagnosticsStarted = false

/** 只读健康请求，不携带登录信息；比较框架、原生和 JSON 解析回调。 */
function probe(
  request: RequestFn,
  name: string,
  dataType: 'json' | 'text',
  observeComplete = false
): Promise<ProbeResult> {
  const startedAt = Date.now()
  const requestId = `probe-${name}-${startedAt}-${Math.random().toString(16).slice(2, 8)}`
  console.info('[network-probe] start', { probe: name, requestId })
  return new Promise((resolve) => {
    let settled = false
    let task: UniApp.RequestTask | undefined
    const timer = setTimeout(() => {
      if (settled) return
      finish('deadline')
      try {
        task?.abort()
      } catch {
        // 已结束诊断；平台中止失败不影响其他探针。
      }
    }, 6000)

    function finish(phase: ProbeResult['phase'], status?: number) {
      if (settled) return
      settled = true
      clearTimeout(timer)
      const result: ProbeResult = {
        probe: name,
        requestId,
        phase,
        elapsedMs: Date.now() - startedAt,
        ...(status === undefined ? {} : { status })
      }
      console.info('[network-probe] result', JSON.stringify(result))
      resolve(result)
    }

    try {
      task = request({
        url: `${healthOrigin}/health/live`,
        method: 'GET',
        header: { 'X-Request-Id': requestId },
        dataType,
        responseType: 'text',
        timeout: 5000,
        success: (res) => finish('success', res.statusCode),
        fail: () => finish('fail'),
        ...(observeComplete
          ? { complete: () => console.info('[network-probe] complete', { probe: name, requestId }) }
          : {})
      })
    } catch {
      finish('threw')
    }
  })
}

async function runNetworkDiagnostics(): Promise<ProbeResult[]> {
  const uniRequest: RequestFn = (options) => uni.request(options)
  const probes = [probe(uniRequest, 'uni-json', 'json'), probe(uniRequest, 'uni-json-complete', 'json', true)]
  if (typeof wx !== 'undefined' && typeof wx.request === 'function') {
    const nativeRequest = wx.request.bind(wx)
    probes.push(probe(nativeRequest, 'wx-json', 'json'), probe(nativeRequest, 'wx-text', 'text'))
  }
  const results = await Promise.all(probes)
  console.info('[network-probe] summary', JSON.stringify(results))
  return results
}

function startLocalNetworkDiagnostics() {
  // 本地 HTTP 调试包启动时运行一次；正式 HTTPS 包不运行。
  if (
    diagnosticsStarted ||
    !/^http:\/\/(?:localhost|127\.0\.0\.1|192\.168\.|10\.|172\.(?:1[6-9]|2\d|3[01])\.)/.test(healthOrigin)
  ) {
    return
  }
  diagnosticsStarted = true
  console.info('[network-probe] auto.start')
  void runNetworkDiagnostics().catch(() => console.info('[network-probe] unavailable'))
}

onLaunch(() => {
  console.info('[app] MeowHome 小程序启动', { buildTime: __MEOWHOME_BUILD_TIME__ })

  // 小程序没有浏览器在线状态 API，网络状态必须主动查询 + 注册监听。
  // Web 端是在 store 的 state 初始化时同步读取的，这里改为异步校准。
  const app = useAppStore()
  app.syncNetworkStatus()
  app.listenNetworkStatus()

  // 诊断失败不能中断应用的网络状态初始化。
  try {
    startLocalNetworkDiagnostics()
  } catch {
    console.info('[network-probe] unavailable')
  }
})
</script>

<!--
  小程序没有 index.html，App.vue 不渲染任何 DOM，仅承载全局样式与生命周期。

  全局样式必须在 App.vue 的**非 scoped** style 中引入：
  uni-app 会把它编译成 app.wxss，对全部页面生效。

  ⚠️ pages.css 里的类名（.page-header / .cat-list-item / .btn-primary /
  .sheet-overlay 等）是 Web 端各视图直接使用的全局类，**必须在此全局引入**，
  否则页面样式会整体丢失。

  顺序不可调换：tokens（变量）→ reset（归零）→ global（基础）→ pages（业务类）。
-->
<style>
@import './styles/tokens.css';
@import './styles/reset.css';
@import './styles/global.css';
@import './styles/pages.css';
</style>
