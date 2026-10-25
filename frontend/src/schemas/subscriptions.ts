import { z } from 'zod'
import { isLocalDate } from '@/utils/dates'
import { currencies } from '@/utils/money'
import { isSubscriptionLink } from '@/utils/subscriptionLink'

export const localDateSchema = z.string().refine(isLocalDate, '日期须为 YYYY-MM-DD 格式且有效')
const positiveInteger = z
  .number({ error: '请输入有效的正整数' })
  .int('请输入整数')
  .positive('数值必须大于 0')
  .max(Number.MAX_SAFE_INTEGER, '数值超出支持范围')
const intervalUnit = z.enum(['day', 'week', 'month', 'year'], { error: '请选择周期单位' })
const nullableText = (max: number, label: string) =>
  z
    .string()
    .trim()
    .max(max, `${label}最多 ${max} 个字符`)
    .transform((value) => value || null)
    .nullable()
const icon = nullableText(2048, '图标').refine((value) => {
  if (value === null) return true
  try {
    const url = new URL(value)
    return url.protocol === 'https:' && !url.username && !url.password
  } catch {
    return !/^[a-z][a-z\d+.-]*:/i.test(value) && !value.startsWith('//') && [...value].length <= 16
  }
}, '图标须为 16 字以内的文字、表情，或不含账号密码的 HTTPS 图片链接')

const fields = {
  name: z
    .string()
    .trim()
    .min(1, '请输入订阅名称')
    .max(100, '名称最多 100 个字符')
    .refine((value) => !value.includes('\0'), '名称不能包含空字符'),
  icon,
  website_url: nullableText(2048, '链接').refine(
    (value) => value === null || isSubscriptionLink(value),
    '链接须为 HTTP/HTTPS，不含账号密码',
  ),
  amount_minor: z
    .number({ error: '请输入有效金额' })
    .int('金额最多两位小数')
    .min(0, '金额不能为负数')
    .max(Number.MAX_SAFE_INTEGER, '金额超出支持范围'),
  currency: z.enum(currencies, { error: '请选择支持的币种' }),
  subscription_type: z.enum(['one_time', 'recurring'], { error: '请选择订阅类型' }),
  start_date: localDateSchema.nullable(),
  next_payment: localDateSchema,
  interval_count: positiveInteger.nullable(),
  interval_unit: intervalUnit.nullable(),
  notify_enabled: z.boolean(),
  reminder_days: z
    .number({ error: '提醒天数须为 0–365 的整数' })
    .int('提醒天数须为整数')
    .min(0, '提醒天数不能小于 0')
    .max(365, '提醒天数不能超过 365')
    .nullable(),
  notes: nullableText(2000, '备注'),
  status: z.enum(['active', 'inactive']),
}

export const subscriptionFieldsSchema = z
  .strictObject({
    ...fields,
    anchor_month: z.number().int().min(1).max(12).nullable(),
    anchor_day: z.number().int().min(1).max(31).nullable(),
  })
  .superRefine((value, context) => {
    const issue = (path: string, message: string) =>
      context.addIssue({ code: 'custom', path: [path], message })
    if (value.subscription_type === 'one_time') {
      for (const key of [
        'interval_count',
        'interval_unit',
        'start_date',
        'anchor_day',
        'anchor_month',
      ] as const) {
        if (value[key] !== null) issue(key, '一次性订阅不包含周期配置')
      }
      return
    }
    if (value.interval_count === null) issue('interval_count', '请输入正整数周期倍数')
    if (value.interval_unit === null) issue('interval_unit', '请选择周期单位')
    if (value.start_date === null) issue('start_date', '请选择开始日期')
    else if (value.start_date >= value.next_payment) issue('start_date', '开始日期必须早于到期日期')
    const needsDay = value.interval_unit === 'month' || value.interval_unit === 'year'
    if (needsDay !== (value.anchor_day !== null)) issue('anchor_day', '续订日期配置无效')
    if ((value.interval_unit === 'year') !== (value.anchor_month !== null))
      issue('anchor_month', '续订日期配置无效')
  })

const inputFields = {
  name: fields.name,
  amount_minor: fields.amount_minor,
  currency: fields.currency,
  next_payment: fields.next_payment,
  subscription_type: fields.subscription_type,
  interval_count: fields.interval_count.optional(),
  interval_unit: fields.interval_unit.optional(),
  start_date: fields.start_date.optional(),
  anchor_day: subscriptionFieldsSchema.shape.anchor_day.optional(),
  anchor_month: subscriptionFieldsSchema.shape.anchor_month.optional(),
  icon: fields.icon.optional(),
  website_url: fields.website_url.optional(),
  notes: fields.notes.optional(),
  notify_enabled: fields.notify_enabled.optional(),
  reminder_days: fields.reminder_days.optional(),
}

export const createSubscriptionSchema = z
  .strictObject(inputFields)
  .superRefine((value, context) => {
    if (value.subscription_type === 'recurring') {
      if (value.start_date == null)
        context.addIssue({ code: 'custom', path: ['start_date'], message: '请选择开始日期' })
      if (value.interval_count == null)
        context.addIssue({ code: 'custom', path: ['interval_count'], message: '请输入周期倍数' })
      if (value.interval_unit == null)
        context.addIssue({ code: 'custom', path: ['interval_unit'], message: '请选择周期单位' })
    }
  })

export const patchSubscriptionSchema = z
  .strictObject({
    ...z.strictObject(inputFields).partial().shape,
    status: fields.status.optional(),
    expected_version: positiveInteger,
  })
  .refine(
    (value) =>
      Object.keys(value).some(
        (key) => key !== 'expected_version' && value[key as keyof typeof value] !== undefined,
      ),
    '请至少修改一个字段',
  )
