import { onPullDownRefresh, onShareAppMessage, onShow } from '@dcloudio/uni-app'

import { currentPagePath, guardOnShow } from './guard'

/**
 * 注册受保护页面的统一 onShow 生命周期。
 * 守卫通过后才执行取数，避免会话恢复完成前误发家庭维度请求。
 */
export function useProtectedPage(load?: () => void | Promise<void>) {
  onShow(async () => {
    if (!(await guardOnShow())) return
    await load?.()
  })
}

/** 主页面通用原生能力：下拉刷新与转发。 */
export function usePageCapabilities(load: () => void | Promise<void>, title: string) {
  onPullDownRefresh(async () => {
    try {
      await load()
    } finally {
      uni.stopPullDownRefresh()
    }
  })

  onShareAppMessage(() => ({
    title,
    path: currentPagePath() || '/pages/today/index'
  }))
}
