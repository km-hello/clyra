import { z } from 'zod'
import { themeModes } from '@/types/theme'

export const settingsSchema = z.strictObject({
  scheduler_enabled: z.boolean({ error: '请选择是否开启自动提醒' }),
  default_reminder_days: z
    .number({ error: '提醒天数须为 0–365 的整数' })
    .int('提醒天数须为整数')
    .min(0, '提醒天数不能小于 0')
    .max(365, '提醒天数不能超过 365'),
  timezone: z.string().refine((value) => {
    try {
      const timezone = new Intl.DateTimeFormat('en-US', { timeZone: value }).resolvedOptions()
        .timeZone
      return !/^[+-]/.test(timezone)
    } catch {
      return false
    }
  }, '输入有效时区，如 Asia/Singapore'),
  notification_time: z.string().regex(/^(?:[01]\d|2[0-3]):[0-5]\d$/, '时间格式应为 HH:mm'),
  telegram_chat_id: z
    .string()
    .trim()
    .refine((value) => !value.includes('\0'), 'Chat ID 不能包含空字符')
    .transform((value) => value || null)
    .nullable(),
  theme: z.enum(themeModes),
})
