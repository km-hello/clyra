import { z } from 'zod'

export const profileSchema = z.strictObject({
  nickname: z
    .string()
    .trim()
    .max(32, '昵称最多 32 个字符')
    .refine((value) => !value.includes('\0'), '昵称不能包含空字符')
    .transform((value) => value || null)
    .nullable(),
  avatar_url: z
    .string()
    .trim()
    .max(2048, '头像地址最多 2048 个字符')
    .refine((value) => !value.includes('\0'), '头像地址不能包含空字符')
    .refine((value) => {
      if (!value) return true
      try {
        const url = new URL(value)
        return url.protocol === 'https:' && !url.username && !url.password
      } catch {
        return false
      }
    }, '头像须为 HTTPS 地址，不含账号密码')
    .transform((value) => value || null)
    .nullable(),
})
