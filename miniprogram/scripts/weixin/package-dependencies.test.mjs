import test from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const buildRoot = fileURLToPath(new URL('../../dist/build/mp-weixin/', import.meta.url))

test('compiled WeChat package contains the calendar modules and registered auth page', () => {
  const app = JSON.parse(readFileSync(resolve(buildRoot, 'app.json'), 'utf8'))
  const calendarModule = resolve(buildRoot, 'utils/recordCalendar.js')

  assert.ok(existsSync(calendarModule), 'recordCalendar.js must be present in the package')

  for (const page of ['pages/records/index.js', 'pages/quick-record/index.js']) {
    const pagePath = resolve(buildRoot, page)
    const source = readFileSync(pagePath, 'utf8')
    assert.match(source, /require\(["']\.\.\/\.\.\/utils\/recordCalendar\.js["']\)/)
    assert.ok(existsSync(resolve(dirname(pagePath), '../../utils/recordCalendar.js')))
  }

  assert.ok(app.pages.includes('pages/auth/index'))
  assert.ok(existsSync(resolve(buildRoot, 'pages/auth/index.js')))
})
