import { z } from 'zod'
import { passwordValueSchema } from '@/schemas/auth'

export const passwordSchema = z
  .strictObject({
    current_password: passwordValueSchema,
    new_password: passwordValueSchema,
    new_password_confirm: z.string(),
  })
  .refine((value) => value.new_password === value.new_password_confirm, {
    path: ['new_password_confirm'],
    message: '两次密码不一致',
  })

export type PasswordInput = z.input<typeof passwordSchema>
