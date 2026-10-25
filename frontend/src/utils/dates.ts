import { TZDate } from '@date-fns/tz'

export type IntervalUnit = 'day' | 'week' | 'month' | 'year'
export interface RenewalAnchor {
  anchor_day: number | null
  anchor_month: number | null
}

function daysInMonth(year: number, month: number): number {
  if (month === 2) return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0) ? 29 : 28
  return [4, 6, 9, 11].includes(month) ? 30 : 31
}

export function isLocalDate(value: unknown): value is string {
  if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return false
  const [year, month, day] = value.split('-').map(Number) as [number, number, number]
  return year >= 1 && month >= 1 && month <= 12 && day >= 1 && day <= daysInMonth(year, month)
}

function parts(date: string): [number, number, number] {
  if (!isLocalDate(date)) throw new RangeError('日期必须是有效的四位年 Gregorian 日期')
  return date.split('-').map(Number) as [number, number, number]
}

function formatDate(year: number, month: number, day: number): string {
  const result = `${String(year).padStart(4, '0')}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
  if (!isLocalDate(result)) throw new RangeError('日期超出 0001–9999 年范围')
  return result
}

export function dayNumber(date: string): number {
  const [year, month, day] = parts(date)
  const utc = new Date(0)
  utc.setUTCFullYear(year, month - 1, day)
  return utc.getTime() / 86_400_000
}

export function todayInTimezone(now: number | Date, timezone: string): string {
  const fields = new Intl.DateTimeFormat('en-US', {
    timeZone: timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    era: 'short',
  }).formatToParts(now)
  const get = (type: string) => fields.find((field) => field.type === type)?.value
  if (get('era') !== 'AD') throw new RangeError('日期超出 0001–9999 年范围')
  return formatDate(Number(get('year')), Number(get('month')), Number(get('day')))
}

export function nextBusinessMidnight(now: number | Date, timezone: string): number {
  const timestamp = typeof now === 'number' ? now : now.getTime()
  const midnight = new TZDate(timestamp, timezone)
  const next = midnight.setHours(24, 0, 0, 0)
  if (!Number.isFinite(timestamp) || !Number.isFinite(next) || next <= timestamp) {
    throw new RangeError('无法计算有效的下一业务日午夜')
  }
  return next
}

export function advanceDate(
  date: string,
  count: number,
  unit: IntervalUnit,
  anchor?: RenewalAnchor,
): string {
  const [year, month, day] = parts(date)
  if (!Number.isSafeInteger(count)) throw new RangeError('周期倍数必须是安全整数')
  if (unit === 'day' || unit === 'week') {
    const target = dayNumber(date) + count * (unit === 'week' ? 7 : 1)
    if (
      !Number.isSafeInteger(target) ||
      target < dayNumber('0001-01-01') ||
      target > dayNumber('9999-12-31')
    ) {
      throw new RangeError('日期超出 0001–9999 年范围')
    }
    const utc = new Date(target * 86_400_000)
    return formatDate(utc.getUTCFullYear(), utc.getUTCMonth() + 1, utc.getUTCDate())
  }
  const anchorDay = anchor?.anchor_day ?? day
  if (!Number.isInteger(anchorDay) || anchorDay < 1 || anchorDay > 31)
    throw new RangeError('续订日期配置无效')
  if (unit === 'month') {
    const target = year * 12 + month - 1 + count
    if (!Number.isSafeInteger(target) || target < 12 || target > 9999 * 12 + 11)
      throw new RangeError('日期超出 0001–9999 年范围')
    const targetYear = Math.floor(target / 12)
    const targetMonth = (target % 12) + 1
    return formatDate(
      targetYear,
      targetMonth,
      Math.min(anchorDay, daysInMonth(targetYear, targetMonth)),
    )
  }
  if (unit !== 'year') throw new RangeError('周期单位无效')
  const targetYear = year + count
  const targetMonth = anchor?.anchor_month ?? month
  if (
    !Number.isSafeInteger(targetYear) ||
    targetYear < 1 ||
    targetYear > 9999 ||
    !Number.isInteger(targetMonth) ||
    targetMonth < 1 ||
    targetMonth > 12
  )
    throw new RangeError('日期或续订锚点超出有效范围')
  return formatDate(
    targetYear,
    targetMonth,
    Math.min(anchorDay, daysInMonth(targetYear, targetMonth)),
  )
}

function cycleIndex(start: string, count: number, unit: IntervalUnit, today: string): number {
  if (!Number.isSafeInteger(count) || count < 1) throw new RangeError('周期倍数必须是正整数')
  const [startYear, startMonth] = parts(start)
  const [todayYear, todayMonth] = parts(today)
  const elapsed =
    unit === 'year'
      ? todayYear - startYear
      : unit === 'month'
        ? (todayYear - startYear) * 12 + todayMonth - startMonth
        : (dayNumber(today) - dayNumber(start)) / (unit === 'week' ? 7 : 1)
  const cycles = Math.max(0, Math.floor(elapsed / count))
  return cycles > 0 && advanceDate(start, cycles * count, unit) > today ? cycles - 1 : cycles
}

export function nextDueDate(
  start: string,
  count: number,
  unit: IntervalUnit,
  today: string,
): string {
  const index = cycleIndex(start, count, unit, today)
  const boundary = advanceDate(start, index * count, unit)
  return index > 0 && boundary === today ? boundary : advanceDate(start, (index + 1) * count, unit)
}

export function currentCycle(
  start: string,
  expiry: string,
  count: number,
  unit: IntervalUnit,
  today: string,
): { start: string; end: string; percentage: number } {
  parts(today)
  if (!isLocalDate(expiry) || expiry <= start) throw new RangeError('开始日期必须早于到期日期')
  const reference = today >= expiry ? advanceDate(expiry, -1, 'day') : today
  const index = cycleIndex(start, count, unit, reference)
  const cycleStart = advanceDate(start, index * count, unit)
  let end = expiry
  try {
    const boundary = advanceDate(start, (index + 1) * count, unit)
    if (boundary < end) end = boundary
  } catch (error) {
    // A valid expiry can truncate a cycle whose full end exceeds year 9999.
    if (!(error instanceof RangeError)) throw error
  }
  const percentage =
    today >= expiry
      ? 100
      : Math.max(
          0,
          Math.min(
            99,
            Math.round(
              ((dayNumber(today) - dayNumber(cycleStart)) /
                (dayNumber(end) - dayNumber(cycleStart))) *
                100,
            ),
          ),
        )
  return { start: cycleStart, end, percentage }
}
