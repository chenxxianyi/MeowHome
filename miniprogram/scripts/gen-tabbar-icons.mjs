/**
 * 生成微信小程序 tabBar 图标（81×81 PNG，普通态 + 选中态）。
 *
 * ## 为什么需要
 *
 * 微信原生 tabBar 的 `iconPath` 只接受**本地图片文件**（png/jpg，建议 81×81，
 * 单个 < 40 KB），不接受 SVG、网络图片或字体图标——所以 `AppIcon.vue` 那套
 * CSS 遮罩方案在这里用不上，必须落成图片。
 *
 * 本脚本从 Web 端**同一份** `iconPaths.ts` 取 path 数据，用与 app 一致的
 * 描边参数（`stroke-width: 1.8`、`linecap/linejoin: round`）渲染，
 * 因此 tabBar 图标与页面内图标视觉一致。
 *
 * ## 颜色
 *
 * 取自 `tokens.css`，与 `pages.json` 的 tabBar 配置保持一致：
 *   普通态 `--color-text-tertiary` = #9B948C
 *   选中态 `--color-brand`         = #C87345
 *
 * ## 用法
 *
 * ```bash
 * cd miniprogram
 * node scripts/gen-tabbar-icons.mjs
 * ```
 *
 * Web 端图标或主题色变更时重跑。
 */
import { readFileSync, writeFileSync, mkdirSync, statSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import sharp from 'sharp'

const here = dirname(fileURLToPath(import.meta.url))
const SRC = resolve(here, '../../frontend/src/components/app/iconPaths.ts')
const OUT = resolve(here, '../src/static/tabbar')

/** tabBar 项 → 图标名。顺序与 pages.json 的 tabBar.list 一致。 */
const TABS = [
  { file: 'today', icon: 'home' },
  { file: 'records', icon: 'record' },
  { file: 'cats', icon: 'cat' },
  { file: 'moments', icon: 'timeline' },
  { file: 'family', icon: 'family' }
]

const COLOR = '#9B948C' // --color-text-tertiary
const COLOR_ACTIVE = '#C87345' // --color-brand

const SIZE = 81
/** 视图外扩量：viewBox 由 24 → 30，给图标留出约 10% 的内边距 */
const PAD = 3

function parseIcons() {
  const raw = readFileSync(SRC, 'utf8')
  const re = /^\s{2}([A-Za-z][A-Za-z0-9_]*):\s*`([\s\S]*?)`,?\s*$/gm
  const map = {}
  let m
  while ((m = re.exec(raw)) !== null) map[m[1]] = m[2].replace(/\s+/g, ' ').trim()
  return map
}

function buildSvg(inner, color) {
  const vb = `${-PAD} ${-PAD} ${24 + PAD * 2} ${24 + PAD * 2}`
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${SIZE}" height="${SIZE}" viewBox="${vb}" fill="none" stroke="${color}" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">${inner}</svg>`
}

async function main() {
  const icons = parseIcons()
  mkdirSync(OUT, { recursive: true })

  const rows = []
  for (const tab of TABS) {
    const inner = icons[tab.icon]
    if (!inner) {
      console.error(`✗ 找不到图标 "${tab.icon}"（${tab.file}）`)
      process.exitCode = 1
      continue
    }
    for (const [suffix, color] of [
      ['', COLOR],
      ['-active', COLOR_ACTIVE]
    ]) {
      const file = `${tab.file}${suffix}.png`
      const path = resolve(OUT, file)
      // 不传 density：sharp 默认 72 DPI，SVG 按声明尺寸（SIZE×SIZE）渲染。
      // 曾误传 density:384 导致输出 432×432，与微信建议的 81×81 不符。
      await sharp(Buffer.from(buildSvg(inner, color)))
        .png({ compressionLevel: 9 })
        .toFile(path)
      rows.push({ file, icon: tab.icon, color, bytes: statSync(path).size })
    }
  }

  console.log(`已生成到 ${OUT}`)
  for (const r of rows) {
    const warn = r.bytes > 40 * 1024 ? '  ⚠ 超过 40 KB！' : ''
    console.log(`  ${r.file.padEnd(18)} ${r.icon.padEnd(10)} ${r.color}  ${(r.bytes / 1024).toFixed(1)} KB${warn}`)
  }
  console.log(`  合计 ${rows.length} 个文件，${(rows.reduce((s, r) => s + r.bytes, 0) / 1024).toFixed(1)} KB`)
}

main()
