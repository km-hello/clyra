export const currencies = [
  'CNY',
  'USD',
  'EUR',
  'GBP',
  'JPY',
  'HKD',
  'TWD',
  'SGD',
  'AUD',
  'CAD',
] as const

export const currencyLabels: Record<(typeof currencies)[number], string> = {
  CNY: '人民币',
  USD: '美元',
  EUR: '欧元',
  GBP: '英镑',
  JPY: '日元',
  HKD: '港元',
  TWD: '新台币',
  SGD: '新加坡元',
  AUD: '澳大利亚元',
  CAD: '加拿大元',
}

export function parseAmountMinor(input: string): number {
  const value = input.trim()
  if (!value) throw new Error('请输入金额')
  if (!/^\d+(?:\.\d{1,2})?$/.test(value)) throw new Error('金额不能为负数，最多两位小数')
  const [whole, fraction = ''] = value.split('.')
  const amount = Number(whole + fraction.padEnd(2, '0'))
  if (!Number.isSafeInteger(amount)) throw new Error('金额超出支持范围')
  return amount
}

export function formatAmountMinor(amount: number, grouped = false): string {
  if (!Number.isSafeInteger(amount) || amount < 0) throw new Error('金额无效')
  const digits = String(amount).padStart(3, '0')
  const whole = digits.slice(0, -2)
  return `${grouped ? whole.replace(/\B(?=(\d{3})+(?!\d))/g, ',') : whole}.${digits.slice(-2)}`
}
