import { z } from 'zod'
import { profileSchema } from '@/schemas/profile'

const username = z
  .string()
  .trim()
  .min(4, { error: (issue) => (issue.input === '' ? '请输入用户名' : '用户名至少 4 个字符') })
  .max(32, '用户名最多 32 个字符')
  .regex(/^[a-z]*$/, '用户名仅支持小写英文字母（a–z）')
export const passwordValueSchema = z
  .string()
  .min(6, { error: (issue) => (issue.input === '' ? '请输入密码' : '密码至少 6 个字符') })
  .max(32, '密码最多 32 个字符')

export const loginSchema = z.strictObject({
  username,
  password: passwordValueSchema,
})

export const setupSchema = z
  .strictObject({
    username,
    password: passwordValueSchema,
    password_confirm: z.string(),
    nickname: profileSchema.shape.nickname.default(null),
    avatar_url: profileSchema.shape.avatar_url.default(null),
  })
  .refine((value) => value.password === value.password_confirm, {
    path: ['password_confirm'],
    message: '两次密码不一致',
  })
