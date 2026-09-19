import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')

test('pages.json 中的每个页面都有源码文件', () => {
  const pages = JSON.parse(readFileSync(resolve(root, 'src/pages.json'), 'utf8')).pages
  assert.ok(pages.length >= 17)
  for (const page of pages) {
    assert.ok(existsSync(resolve(root, `src/${page.path}.vue`)), `${page.path}.vue 不存在`)
  }
})

test('tabBar 图标文件完整且页面均已注册', () => {
  const config = JSON.parse(readFileSync(resolve(root, 'src/pages.json'), 'utf8'))
  const registered = new Set(config.pages.map((page) => page.path))
  assert.equal(config.tabBar.list.length, 5)
  for (const tab of config.tabBar.list) {
    assert.ok(registered.has(tab.pagePath), `${tab.pagePath} 未注册`)
    assert.ok(existsSync(resolve(root, `src/${tab.iconPath}`)), `${tab.iconPath} 不存在`)
    assert.ok(existsSync(resolve(root, `src/${tab.selectedIconPath}`)), `${tab.selectedIconPath} 不存在`)
  }
})

test('57 个遮罩图标均为可解码的 SVG data URI', () => {
  const source = readFileSync(resolve(root, 'src/components/app/iconData.ts'), 'utf8')
  const uris = [...source.matchAll(/'data:image\/svg\+xml;base64,([^']+)'/g)].map((match) => match[1])
  assert.equal(uris.length, 57)
  for (const uri of uris) {
    const svg = Buffer.from(uri, 'base64').toString('utf8')
    assert.match(svg, /^<svg[\s\S]*<\/svg>$/)
  }
})
