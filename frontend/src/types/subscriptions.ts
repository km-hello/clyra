import type { z } from 'zod'
import type { ThemeMode } from '@/types/theme'
import type {
  createSubscriptionSchema,
  patchSubscriptionSchema,
  subscriptionFieldsSchema,
} from '@/schemas/subscriptions'

export type SubscriptionFields = z.output<typeof subscriptionFieldsSchema>
export type Subscription = SubscriptionFields & {
  id: string
  version: number
  created_at: number
  updated_at: number
}
export type CreateSubscriptionInput = z.input<typeof createSubscriptionSchema>
export type PatchSubscriptionInput = z.input<typeof patchSubscriptionSchema>
export interface SubscriptionListResponse {
  data: Subscription[]
  meta: {
    page: number
    page_size: number
    total: number
    total_pages: number
    has_any_subscriptions: boolean
    today: string
    timezone: string
    default_reminder_days: number
    theme: ThemeMode
  }
}
