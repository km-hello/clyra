import type { z } from 'zod'
import type { settingsSchema } from '@/schemas/settings'

export type SettingsValues = z.output<typeof settingsSchema>
export type SettingsResponse = SettingsValues & {
  bot_configured: boolean
}
