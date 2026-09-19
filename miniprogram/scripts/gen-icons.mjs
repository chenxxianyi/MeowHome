/**
 * 从小程序外部的 Web 端 iconPaths.ts 生成小程序端图标数据。
 *
 * ## 为什么需要这个脚本
 *
 * 小程序**不支持内联 `<svg>`，也不支持 `v-html`**，所以 Web 端
 * `AppIcon.vue` 的 `<svg v-html="content">` 实现无法照搬。
 *
 * 这里把每个图标的 path 数据原样内联进一个完整的 `<svg>`，再 base64 编码成
 * data URI，供 WXSS 的 `-webkit-mask-image` 使用。用遮罩而非直接贴图的原因是：
 * 遮罩只取 alpha 通道，颜色由 `background-color: currentColor` 决定，
 * 因此**颜色能跟随父元素文字色**——这正是 Web 端 `stroke="currentColor"` 的语义。
 *
 * ## 为什么不转图标字体（原方案的推荐做法）
 *
 * 这 57 个图标是**描边式**（`fill="none"` + `stroke`，且含 `polyline`、
 * 无填充 `circle`）。字体字形是**填充形状**，描边需要先做 outline 展开
 * （stroke → filled path）才能成为字形，否则渲染出来是空的。
 * 遮罩方案直接复用原始 path，无此问题，也不需要引入字体生成工具链。
 *
 * ## 用法
 *
 * ```bash
 * cd miniprogram
 * node scripts/gen-icons.mjs
 * ```
 *
 * Web 端图标有增删改时**必须重跑本脚本**，否则两端图标会漂移。
 */
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const SRC = resolve(here, '../../frontend/src/components/app/iconPaths.ts')
const OUT = resolve(here, '../src/components/app/iconData.ts')

/** 与 Web 端 AppIcon.vue 上的 svg 属性保持一致（stroke 颜色对遮罩无影响，仅取 alpha）。 */
const SVG_ATTRS = [
  'xmlns="http://www.w3.org/2000/svg"',
  'viewBox="0 0 24 24"',
  'fill="none"',
  'stroke="#000"',
  'stroke-width="1.8"',
  'stroke-linecap="round"',
  'stroke-linejoin="round"'
].join(' ')

function main() {
  const raw = readFileSync(SRC, 'utf8')

  // 匹配形如：  home: `<path .../><polyline .../>`,
  const re = /^\s{2}([A-Za-z][A-Za-z0-9_]*):\s*`([\s\S]*?)`,?\s*$/gm
  const entries = []
  let m
  while ((m = re.exec(raw)) !== null) {
    const name = m[1]
    const inner = m[2].replace(/\s+/g, ' ').trim()
    if (!inner) continue
    entries.push([name, inner])
  }

  if (entries.length === 0) {
    console.error(`未从 ${SRC} 解析到任何图标，请检查格式是否变化。`)
    process.exit(1)
  }

  const lines = entries.map(([name, inner]) => {
    const svg = `<svg ${SVG_ATTRS}>${inner}</svg>`
    const uri = `data:image/svg+xml;base64,${Buffer.from(svg, 'utf8').toString('base64')}`
    return `  ${name}: '${uri}'`
  })

  const out = `// ⚠️ 本文件由 scripts/gen-icons.mjs 自动生成，请勿手工编辑。
// 源：frontend/src/components/app/iconPaths.ts
// 重新生成：cd miniprogram && node scripts/gen-icons.mjs
//
// 每个值是 \`data:image/svg+xml;base64,...\`，供 AppIcon.vue 用作 CSS 遮罩。
// 遮罩只取 alpha，颜色由 background-color: currentColor 提供。

export const iconDataUris = {
${lines.join(',\n')}
} as const

export type AppIconName = keyof typeof iconDataUris

/** 全部图标名，供验证页遍历。 */
export const iconNames = Object.keys(iconDataUris) as AppIconName[]
`

  mkdirSync(dirname(OUT), { recursive: true })
  writeFileSync(OUT, out, 'utf8')

  const bytes = Buffer.byteLength(out, 'utf8')
  console.log(`已生成 ${OUT}`)
  console.log(`  图标数量 : ${entries.length}`)
  console.log(`  文件大小 : ${(bytes / 1024).toFixed(1)} KB`)
  console.log(`  图标名   : ${entries.map((e) => e[0]).join(', ')}`)
}

main()
