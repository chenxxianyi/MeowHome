import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'

const tsOptions = { compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 } }
const dateSource = await readFile(new URL('../src/utils/date.ts', import.meta.url), 'utf8')
const dateJavaScript = ts.transpileModule(dateSource, tsOptions).outputText
const dateModuleUrl = `data:text/javascript;base64,${Buffer.from(dateJavaScript).toString('base64')}`
let calendarSource = await readFile(new URL('../src/utils/recordCalendar.ts', import.meta.url), 'utf8')
calendarSource = ts.transpileModule(calendarSource, tsOptions).outputText
calendarSource = calendarSource.replace(/from ['"]\.\/date['"]/, `from '${dateModuleUrl}'`)
const calendarModuleUrl = `data:text/javascript;base64,${Buffer.from(calendarSource).toString('base64')}`
const calendar = await import(calendarModuleUrl)

test('date keys reject impossible dates and accept leap days', () => {
  assert.equal(calendar.isValidDateKey('2024-02-29'), true)
  assert.equal(calendar.isValidDateKey('2026-02-29'), false)
  assert.equal(calendar.isValidDateKey('2026-13-01'), false)
})

test('month grid starts on Monday and includes leap day', () => {
  const october = calendar.buildMonthGrid('2026-10')
  assert.equal(october.length, 42)
  assert.deepEqual(october.slice(0, 3), [null, null, null])
  assert.equal(october[3].date, '2026-10-01')
  assert.equal(october.filter(Boolean).length, 31)

  const february = calendar.buildMonthGrid('2024-02')
  assert.equal(february.filter(Boolean).length, 29)
  assert.ok(february.some((cell) => cell?.date === '2024-02-29'))
})

test('month navigation crosses year boundaries', () => {
  assert.equal(calendar.shiftMonth('2026-01', -1), '2025-12')
  assert.equal(calendar.shiftMonth('2026-12', 1), '2027-01')
})

test('vaccine records use their explicit vaccination date', () => {
  assert.equal(calendar.recordCalendarDate({
    type: 'vaccine',
    occurred_at: '2026-10-22T01:00:00.000Z',
    payload: { vaccination_date: '2026-10-21' }
  }, 'Asia/Shanghai'), '2026-10-21')
})

test('ordinary records are assigned to the requested timezone date', () => {
  assert.equal(calendar.recordCalendarDate({
    type: 'feeding',
    occurred_at: '2026-10-21T16:30:00.000Z',
    payload: {}
  }, 'Asia/Shanghai'), '2026-10-22')
  assert.equal(calendar.recordCalendarDate({
    type: 'feeding',
    occurred_at: '2026-10-01T00:30:00.000Z',
    payload: {}
  }, 'America/Los_Angeles'), '2026-09-30')
})

test('month query bounds use local midnight and account for daylight saving changes', () => {
  assert.deepEqual(calendar.monthRange('2026-10', 'Asia/Shanghai'), {
    from: '2026-09-30T16:00:00.000Z',
    to: '2026-10-31T15:59:59.999Z'
  })
  assert.deepEqual(calendar.monthRange('2026-03', 'America/Los_Angeles'), {
    from: '2026-03-01T08:00:00.000Z',
    to: '2026-04-01T06:59:59.999Z'
  })
})
