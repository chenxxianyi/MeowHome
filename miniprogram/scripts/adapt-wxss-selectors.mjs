/**
 * 把 Web 端 CSS 的 **HTML 元素选择器** 转换为小程序等价物。
 *
 * ## 为什么需要
 *
 * 计划 §5.5 只覆盖了**模板标签**的替换，漏掉了 **CSS 元素选择器**。
 * 实测：`pages.css` 里有大量 `.today-cover-date span`、`.today-cover-metrics span + span`
 * 这类后代元素选择器，而 **WXSS 不支持 HTML 标签选择器**，构建时会逐个报警告，
 * 且规则会静默失效。
 *
 * ## 映射规则
 *
 * | Web 端元素 | 小程序 | 说明 |
 * |---|---|---|
 * | `span` `strong` `i` `em` `b` `small` | `text` | 行内文本；注意 uni-app 会把 `<span>` 映射成 `<label>`，所以模板侧应显式写 `<text>` |
 * | `div` `p` `h1`~`h6` `section` `header` `footer` `nav` `main` `summary` `ul` `ol` `li` | `view` | 块级 |
 * | `a` | `navigator` | `<a>` 被 uni-app 映射为 `<navigator>` |
 * | `img` | `image` | — |
 * | `select` | `picker` | 小程序用 `<picker>` 替代 |
 * | `svg` `table` `tr` `td` `th` | 删除该选择器项 | 小程序无对应标签；相关规则属 Web 专有 |
 *
 * 属性选择器、`:nth-child` 等伪类保留不动（WXSS 支持）。
 * 独立伪元素 `::after` / `::before` 保留。
 *
 * ## 用法
 *
 * ```bash
 * cd miniprogram
 * node scripts/adapt-wxss-selectors.mjs src/styles/pages.css
 * ```
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

/** 元素 → 小程序等价物。小程序没有对应标签的，替换为永不匹配的哨兵类。 */
const UNSUPPORTED = '.wxss-unsupported'

const MAP = {
  span: 'text',
  strong: 'text',
  i: 'text',
  em: 'text',
  b: 'text',
  small: 'text',
  div: 'view',
  p: 'view',
  h1: 'view',
  h2: 'view',
  h3: 'view',
  h4: 'view',
  h5: 'view',
  h6: 'view',
  section: 'view',
  header: 'view',
  footer: 'view',
  nav: 'view',
  main: 'view',
  summary: 'view',
  ul: 'view',
  ol: 'view',
  li: 'view',
  a: 'navigator',
  img: 'image',
  select: 'picker',
  // 小程序无对应标签。**不能删除该选择器项**——曾试过删除，结果是选择器变空、
  // 换行丢失，产出 `}{` 这类非法结构（行数 4108 → 4065）。
  // 改为替换成永不匹配的哨兵类：规则变成死代码，但 CSS 结构完好。
  svg: UNSUPPORTED,
  table: UNSUPPORTED,
  tr: UNSUPPORTED,
  td: UNSUPPORTED,
  th: UNSUPPORTED
}

/** 匹配选择器中的元素名：前面不能是 . # - 或单词字符，后面不能是单词字符或 - */
const ELEMENT_RE = new RegExp(`(?<![\\w.#-])(${Object.keys(MAP).join('|')})(?![\\w-])`, 'g')

/**
 * 只改写选择器部分。做法：按 `{` / `}` 切块，块首到 `{` 之间是选择器。
 * **不删除任何内容**，只做等长或近似等长的名称替换，保证行结构不变。
 */
function transform(css) {
  const changes = {}
  let out = ''
  let i = 0
  let segStart = 0

  while (i < css.length) {
    const ch = css[i]
    if (ch === '{') {
      const segment = css.slice(segStart, i)
      const atIdx = segment.lastIndexOf('@')
      const isAtRule = atIdx >= 0 && /@[a-z-]+\s*[^{]*$/.test(segment)
      if (!isAtRule) {
        const rewritten = segment.replace(ELEMENT_RE, (m) => {
          const to = MAP[m]
          const key = `${m} → ${to}`
          changes[key] = (changes[key] || 0) + 1
          return to
        })
        out += rewritten + ch
        segStart = i + 1
        i++
        continue
      }
    }
    i++
  }
  out += css.slice(segStart)
  return { css: out, changes }
}

const target = process.argv[2]
if (!target) {
  console.error('用法: node scripts/adapt-wxss-selectors.mjs <css 文件路径>')
  process.exit(1)
}

const path = resolve(process.cwd(), target)
const before = readFileSync(path, 'utf8')
const { css: after, changes } = transform(before)
writeFileSync(path, after, 'utf8')

console.log(`已处理 ${path}`)
const keys = Object.keys(changes).sort()
if (keys.length === 0) {
  console.log('  无需改动')
} else {
  for (const k of keys) console.log(`  ${k.padEnd(20)} ${changes[k]} 处`)
}
console.log(`  文件大小 ${(before.length / 1024).toFixed(1)} KB → ${(after.length / 1024).toFixed(1)} KB`)
