import { z } from 'zod'
import { advanceDate } from '@/utils/dates'
import {
  createSubscriptionSchema,
  patchSubscriptionSchema,
  subscriptionFieldsSchema,
} from '@/schemas/subscriptions'
import type {
  CreateSubscriptionInput,
  PatchSubscriptionInput,
  Subscription,
  SubscriptionFields,
} from '@/types/subscriptions'

export function normalizeSubscription(
  input: CreateSubscriptionInput | PatchSubscriptionInput,
  current?: Subscription,
): SubscriptionFields {
  const parsed = current
    ? patchSubscriptionSchema.parse(input)
    : createSubscriptionSchema.parse(input)
  const { expected_version: _expectedVersion, ...changes } = parsed as PatchSubscriptionInput
  const base = current
    ? subscriptionFieldsSchema.strip().parse(current)
    : {
        icon: null,
        website_url: null,
        notes: null,
        notify_enabled: true,
        reminder_days: null,
        status: 'active' as const,
        start_date: null,
        interval_count: 1,
        interval_unit: 'month' as const,
        anchor_day: null,
        anchor_month: null,
      }
  const value = {
    ...base,
    ...Object.fromEntries(Object.entries(changes).filter(([, field]) => field !== undefined)),
  } as SubscriptionFields
  if (value.subscription_type === 'one_time') {
    value.interval_count =
      value.interval_unit =
      value.start_date =
      value.anchor_day =
      value.anchor_month =
        null
  } else {
    const cycleChanged =
      !current ||
      (['next_payment', 'subscription_type', 'interval_unit'] as const).some(
        (key) => value[key] !== current[key],
      )
    if (cycleChanged) {
      const [, month, day] = value.next_payment.split('-').map(Number) as [number, number, number]
      value.anchor_day =
        changes.anchor_day !== undefined
          ? changes.anchor_day
          : value.interval_unit === 'month' || value.interval_unit === 'year'
            ? day
            : null
      value.anchor_month =
        changes.anchor_month !== undefined
          ? changes.anchor_month
          : value.interval_unit === 'year'
            ? month
            : null
    }
  }
  const normalized = subscriptionFieldsSchema.parse(value)
  if (
    normalized.subscription_type === 'recurring' &&
    advanceDate(normalized.next_payment, 0, normalized.interval_unit!, normalized) !==
      normalized.next_payment
  ) {
    throw new z.ZodError([
      { code: 'custom', path: ['next_payment'], message: '到期日与续订日期配置不一致' },
    ])
  }
  return normalized
}
