import { readdirSync, readFileSync, statSync } from 'node:fs'
import { dirname, resolve, relative } from 'node:path'
import { fileURLToPath } from 'node:url'
import { parse as parseSfc } from '@vue/compiler-sfc'
import { baseParse, NodeTypes } from '@vue/compiler-dom'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const src = resolve(root, 'src')
const excludedPages = new Set(['pages/icon-test/index.vue', 'pages/tag-test/index.vue'])
const htmlTags = new Set([
  'div',
  'span',
  'strong',
  'em',
  'i',
  'b',
  'img',
  'section',
  'header',
  'main',
  'nav',
  'article',
  'h1',
  'h2',
  'h3',
  'p',
  'a',
  'select',
  'option',
  'pre'
])
const errors = []

function files(dir) {
  return readdirSync(dir).flatMap((name) => {
    const path = resolve(dir, name)
    return statSync(path).isDirectory() ? files(path) : [path]
  })
}

function lineAt(source, offset) {
  return source.slice(0, offset).split(/\r?\n/).length
}

function visit(node, source, file, textDepth = 0, textContext = '') {
  if (node.type === NodeTypes.ELEMENT) {
    if (htmlTags.has(node.tag)) {
      errors.push(`${file}:${lineAt(source, node.loc.start.offset)} 禁止的 HTML 标签 <${node.tag}>`)
    }
    if (textDepth > 0 && node.tag !== 'text') {
      errors.push(`${file}:${lineAt(source, node.loc.start.offset)} ${textContext} 内嵌了 <${node.tag}>`)
    }
    if (node.tag === 'text') {
      textDepth += 1
      textContext = node.loc.source.split(/\r?\n/, 1)[0].slice(0, 100)
    }
  }
  for (const child of node.children || []) visit(child, source, file, textDepth, textContext)
}

for (const path of files(src).filter((file) => file.endsWith('.vue'))) {
  const file = relative(src, path).replaceAll('\\', '/')
  if (excludedPages.has(file)) continue
  const source = readFileSync(path, 'utf8')
  const { descriptor, errors: sfcErrors } = parseSfc(source, { filename: file })
  for (const error of sfcErrors) errors.push(`${file}: SFC 解析失败：${String(error)}`)
  if (descriptor.template) {
    const template = descriptor.template.content
    const ast = baseParse(template)
    visit(ast, template, file)
  }
  const script = descriptor.scriptSetup?.content || ''
  if (/from\s+['"]\.\.\/\.\.\/mocks\//.test(script)) errors.push(`${file}: 仍导入 mocks`)
  const executableScript = script.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/.*$/gm, '')
  if (/\b(document|window|navigator|localStorage)\s*\./.test(executableScript))
    errors.push(`${file}: 仍使用浏览器专有 API`)
  if (/页面待迁移|占位页 ——/.test(source)) errors.push(`${file}: 仍是占位页`)

  if (file.startsWith('pages/') && file !== 'pages/auth/index.vue') {
    const guarded = /useProtectedPage\s*\(|guardOnShow\s*\(/.test(script)
    if (!guarded) errors.push(`${file}: 未注册认证/家庭守卫`)
  }
}

if (errors.length) {
  console.error(`迁移静态检查失败（${errors.length} 项）：`)
  for (const error of errors) console.error(`- ${error}`)
  process.exit(1)
}

console.log('迁移静态检查通过：正式页面无 HTML 标签、非法 text 嵌套、mock import、浏览器 API、占位页或守卫缺失。')
