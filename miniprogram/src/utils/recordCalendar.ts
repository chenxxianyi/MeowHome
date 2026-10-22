import type { RecordDTO } from '../api/endpoints'
import { dateAtTimezone, deviceTimezone } from './date'

export interface CalendarCell {
  date: string
  day: number
}

export interface MonthRange {
  from: string
  to: string
}

export function isValidDateKey(value: unknown): value is string {
  if (typeof value !== 'string') return false
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (!match) return false
  const [, year, month, day] = match.map(Number)
  const date = new Date(Date.UTC(year, month - 1, day))
  return date.getUTCFullYear() === year && date.getUTCMonth() === month - 1 && date.getUTCDate() === day
}

export function dateKeyInTimezone(value: string | Date, timezone = deviceTimezone()): string {
  const date = value instanceof Date ? value.toISOString() : value
  return dateAtTimezone(date, timezone).slice(0, 10)
}

export function recordCalendarDate(record: RecordDTO, timezone = deviceTimezone()): string {
  const vaccineDate = record.type === 'vaccine' ? record.payload?.vaccination_date : undefined
  if (isValidDateKey(vaccineDate)) return vaccineDate
  return dateKeyInTimezone(record.occurred_at, timezone)
}

/** Build a six-week, Monday-first month grid. `null` cells are outside the month. */
export function buildMonthGrid(month: string): Array<CalendarCell | null> {
  const match = month.match(/^(\d{4})-(\d{2})$/)
  if (!match) return Array.from({ length: 42 }, () => null)
  const year = Number(match[1])
  const monthNumber = Number(match[2])
  if (monthNumber < 1 || monthNumber > 12) return Array.from({ length: 42 }, () => null)

  const firstWeekday = new Date(Date.UTC(year, monthNumber - 1, 1)).getUTCDay()
  const leadingCells = (firstWeekday + 6) % 7
  const daysInMonth = new Date(Date.UTC(year, monthNumber, 0)).getUTCDate()
  return Array.from({ length: 42 }, (_, index) => {
    const day = index - leadingCells + 1
    if (day < 1 || day > daysInMonth) return null
    const date = `${match[1]}-${match[2]}-${String(day).padStart(2, '0')}`
    return { date, day }
  })
}

export function shiftMonth(month: string, amount: number): string {
  const match = month.match(/^(\d{4})-(\d{2})$/)
  if (!match) return month
  const date = new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1 + amount, 1))
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`
}

function localDateTimeToUtc(value: string, timezone: string): number {
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})\.(\d{3})$/)
  if (!match) throw new Error(`Invalid local date time: ${value}`)
  const [, yearText, monthText, dayText, hourText, minuteText, secondText, millisecondText] = match
  const target = Date.UTC(
    Number(yearText), Number(monthText) - 1, Number(dayText),
    Number(hourText), Number(minuteText), Number(secondText), Number(millisecondText)
  )
  let candidate = target
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone: timezone,
    year: 'numeric', month: '2-digit', day: '2-digit',
    hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23'
  })

  // Correct the candidate until its wall-clock representation matches the requested zone time.
  for (let attempt = 0; attempt < 4; attempt += 1) {
    const parts = Object.fromEntries(formatter.formatToParts(new Date(candidate)).map((part) => [part.type, part.value]))
    const represented = Date.UTC(
      Number(parts.year), Number(parts.month) - 1, Number(parts.day),
      Number(parts.hour), Number(parts.minute), Number(parts.second), Number(millisecondText)
    )
    const difference = target - represented
    if (!difference) break
    candidate += difference
  }
  return candidate
}

/** Inclusive UTC query bounds for a local calendar month in the supplied IANA zone. */
export function monthRange(month: string, timezone = deviceTimezone()): MonthRange {
  const match = month.match(/^(\d{4})-(\d{2})$/)
  if (!match || Number(match[2]) < 1 || Number(match[2]) > 12) throw new Error(`Invalid month: ${month}`)
  const year = Number(match[1])
  const monthNumber = Number(match[2])
  const nextMonth = new Date(Date.UTC(year, monthNumber, 1))
  const startLocal = `${month}-01T00:00:00.000`
  const nextMonthLocal = `${nextMonth.getUTCFullYear()}-${String(nextMonth.getUTCMonth() + 1).padStart(2, '0')}-01T00:00:00.000`
  const start = localDateTimeToUtc(startLocal, timezone)
  const end = localDateTimeToUtc(nextMonthLocal, timezone) - 1
  return { from: new Date(start).toISOString(), to: new Date(end).toISOString() }
}
