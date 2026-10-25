<script setup lang="ts">
import { ref, computed, reactive, watch, nextTick, useTemplateRef } from 'vue'
import {
  NAlert,
  NButton,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NRadioButton,
  NRadioGroup,
  NSwitch,
  NTooltip,
  useMessage,
} from 'naive-ui'
import { Calendar, RefreshCw } from '@lucide/vue'
import SubscriptionIconInput from '@/components/subscriptions/SubscriptionIconInput.vue'
import DiscardChangesDialog from '@/components/common/DiscardChangesDialog.vue'
import { useFormErrors } from '@/composables/useFormErrors'
import { useApi } from '@/composables/useApi'
import { z } from 'zod'
import { nextDueDate, todayInTimezone, isLocalDate, type IntervalUnit } from '@/utils/dates'
import { normalizeSubscription } from '@/utils/subscriptions'
import { currencies, currencyLabels, formatAmountMinor, parseAmountMinor } from '@/utils/money'
import type {
  CreateSubscriptionInput,
  PatchSubscriptionInput,
  Subscription,
} from '@/types/subscriptions'

type FormError = { name: string; message: string }
type SubscriptionDraft = Omit<CreateSubscriptionInput, 'amount_minor'> & { amount: string }

const $api = useApi()

const props = defineProps<{
  subscription: Subscription | null
  defaultReminderDays: number
  timezone: string | undefined
}>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ saved: [] }>()
const message = useMessage()
const { errors: fieldErrors, clear: clearErrors, setErrors } = useFormErrors('subscription-form')
const form = useTemplateRef('form')
const discardOpen = ref(false)
const pending = ref(false)
const errorMessage = ref('')
const conflict = ref(false)
const draft = reactive<SubscriptionDraft>({
  name: '',
  amount: '',
  currency: 'CNY',
  next_payment: '',
  subscription_type: 'recurring',
  interval_count: 1,
  interval_unit: 'month',
  start_date: null,
  icon: null,
  website_url: null,
  notes: null,
  notify_enabled: true,
  reminder_days: null,
})
const initialDraft = ref<SubscriptionDraft>({ ...draft })
const calculatedSchedule = ref<{
  interval_unit: IntervalUnit
  next_payment: string
  anchor_day: number | null
  anchor_month: number | null
} | null>(null)
let recurringDraft: Pick<
  CreateSubscriptionInput,
  'start_date' | 'interval_count' | 'interval_unit'
> | null = null
const units: { label: string; value: IntervalUnit }[] = [
  { label: '天', value: 'day' },
  { label: '周', value: 'week' },
  { label: '月', value: 'month' },
  { label: '年', value: 'year' },
]
const calculatedAnchor = computed(() => {
  const schedule = calculatedSchedule.value
  return schedule &&
    draft.subscription_type === 'recurring' &&
    draft.interval_unit === schedule.interval_unit &&
    draft.next_payment === schedule.next_payment
    ? { anchor_day: schedule.anchor_day, anchor_month: schedule.anchor_month }
    : {}
})
const dirty = computed(
  () =>
    Object.entries(draft).some(
      ([key, value]) => !Object.is(value, initialDraft.value[key as keyof SubscriptionDraft]),
    ) ||
    Object.entries(calculatedAnchor.value).some(
      ([key, value]) => value !== props.subscription?.[key as 'anchor_day' | 'anchor_month'],
    ),
)
const reminderMode = computed({
  get: () => (draft.reminder_days === null ? 'global' : 'custom'),
  set: (value: string) => {
    draft.reminder_days = value === 'global' ? null : props.defaultReminderDays
  },
})

watch(open, (value) => {
  if (!value) return
  const item = props.subscription
  Object.assign(draft, {
    name: item?.name ?? '',
    amount: item ? formatAmountMinor(item.amount_minor) : '',
    currency: item?.currency ?? 'CNY',
    next_payment: item?.next_payment ?? '',
    subscription_type: item?.subscription_type ?? 'recurring',
    interval_count: item ? item.interval_count : 1,
    interval_unit: item ? item.interval_unit : 'month',
    start_date: item?.start_date ?? null,
    icon: item?.icon ?? null,
    website_url: item?.website_url ?? null,
    notes: item?.notes ?? null,
    notify_enabled: item?.notify_enabled ?? true,
    reminder_days: item?.reminder_days ?? null,
  })
  initialDraft.value = { ...draft }
  recurringDraft = null
  calculatedSchedule.value = null
  discardOpen.value = false
  errorMessage.value = ''
  conflict.value = false
  clearErrors()
})
function updateType() {
  if (draft.subscription_type === 'one_time') {
    recurringDraft = {
      start_date: draft.start_date,
      interval_count: draft.interval_count,
      interval_unit: draft.interval_unit,
    }
    draft.interval_count = null
    draft.interval_unit = null
    draft.start_date = null
  } else {
    Object.assign(
      draft,
      recurringDraft ?? { start_date: null, interval_count: 1, interval_unit: 'month' },
    )
  }
}
function calculateNextPayment() {
  if (
    pending.value ||
    discardOpen.value ||
    draft.subscription_type !== 'recurring' ||
    !props.timezone
  )
    return
  const errors: FormError[] = []
  if (!isLocalDate(draft.start_date))
    errors.push({ name: 'start_date', message: '请先填写有效的开始日期' })
  if (
    !Number.isSafeInteger(draft.interval_count) ||
    !draft.interval_count ||
    draft.interval_count < 1
  )
    errors.push({ name: 'interval_count', message: '请输入正整数周期倍数' })
  if (!draft.interval_unit) errors.push({ name: 'interval_unit', message: '请选择周期单位' })
  if (!errors.length) {
    try {
      const calculated = nextDueDate(
        draft.start_date!,
        draft.interval_count!,
        draft.interval_unit!,
        todayInTimezone(Date.now(), props.timezone),
      )
      if (isLocalDate(draft.next_payment) && draft.next_payment >= calculated) {
        message.info('已保留当前到期日期')
      } else {
        const [, month, day] = draft.start_date!.split('-').map(Number) as [number, number, number]
        draft.next_payment = calculated
        calculatedSchedule.value = {
          interval_unit: draft.interval_unit!,
          next_payment: calculated,
          anchor_day:
            draft.interval_unit === 'month' || draft.interval_unit === 'year' ? day : null,
          anchor_month: draft.interval_unit === 'year' ? month : null,
        }
      }
    } catch {
      errors.push({
        name: 'next_payment',
        message: '推算日期超出 0001–9999 年范围，请调整开始日期或周期',
      })
    }
  }
  for (const name of ['start_date', 'interval_count', 'interval_unit', 'next_payment'])
    delete fieldErrors[name]
  for (const error of errors) fieldErrors[error.name] = error.message
}
function fieldA11y(name: string) {
  return {
    'aria-invalid': !!fieldErrors[name],
    'aria-describedby': fieldErrors[name] ? `subscription-${name}-error` : undefined,
  }
}
function payload(): CreateSubscriptionInput | PatchSubscriptionInput {
  const { amount, ...fields } = draft
  const amountMinor = parseAmountMinor(amount)
  const anchor = calculatedAnchor.value
  if (!props.subscription) return { ...fields, ...anchor, amount_minor: amountMinor }
  return {
    expected_version: props.subscription.version,
    ...anchor,
    ...(amount !== initialDraft.value.amount ? { amount_minor: amountMinor } : {}),
    ...Object.fromEntries(
      Object.entries(fields).filter(
        ([key, value]) => value !== initialDraft.value[key as keyof SubscriptionDraft],
      ),
    ),
  }
}
function validate(): FormError[] {
  if (props.subscription && !dirty.value) return []
  try {
    parseAmountMinor(draft.amount)
  } catch (error) {
    return [{ name: 'amount_minor', message: (error as Error).message }]
  }
  try {
    normalizeSubscription(payload(), props.subscription ?? undefined)
    return []
  } catch (error) {
    if (error instanceof z.ZodError)
      return error.issues.map((issue) => ({ name: issue.path.join('.'), message: issue.message }))
    return [{ name: 'next_payment', message: '请检查到期日与周期配置' }]
  }
}
function requestClose(value = false) {
  if (value || pending.value) return
  if (dirty.value) discardOpen.value = true
  else open.value = false
}
function discard() {
  if (pending.value) return
  discardOpen.value = false
  open.value = false
}
function reloadAfterConflict() {
  open.value = false
  emit('saved')
}
async function submit() {
  if (pending.value || discardOpen.value || (props.subscription && !dirty.value)) return
  const errors = validate()
  setErrors(Object.fromEntries(errors.map(({ name, message }) => [name, message])))
  if (errors.length) {
    await nextTick()
    form.value?.$el
      .querySelector(
        `[data-field="${CSS.escape(errors[0]!.name)}"] input, [data-field="${CSS.escape(errors[0]!.name)}"] button, [data-field="${CSS.escape(errors[0]!.name)}"] select, [data-field="${CSS.escape(errors[0]!.name)}"] textarea`,
      )
      ?.focus()
    return
  }
  pending.value = true
  errorMessage.value = ''
  conflict.value = false
  try {
    const body = payload()
    if (props.subscription)
      await $api(`/api/subscriptions/${props.subscription.id}`, {
        method: 'PATCH',
        body,
        retry: false,
      })
    else await $api('/api/subscriptions', { method: 'POST', body, retry: false })
    message.success(props.subscription ? '订阅已保存' : '订阅已新增')
    open.value = false
    emit('saved')
  } catch (error) {
    const failure = error as {
      statusCode?: number
      data?: { message?: string; data?: { fieldErrors?: Record<string, string> } }
    }
    conflict.value = failure.statusCode === 409
    errorMessage.value = conflict.value
      ? '订阅已更新，当前草稿已保留。请刷新后重新编辑。'
      : failure.data?.message || '保存失败，输入已保留'
    const fields = failure.data?.data?.fieldErrors
    if (fields) {
      setErrors(fields)
    }
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <NModal
    :show="open"
    preset="card"
    content-scrollable
    class="form-modal"
    :title="subscription ? '编辑订阅' : '新增订阅'"
    :aria-label="subscription ? '编辑订阅' : '新增订阅'"
    :mask-closable="!pending"
    :close-on-esc="!pending && !discardOpen"
    :closable="!pending"
    style="width: min(720px, calc(100vw - 32px))"
    @update:show="requestClose"
  >
    <NForm id="subscription-form" ref="form" :model="draft" @submit.prevent="submit">
      <NAlert v-if="errorMessage" type="error" class="mb-5" role="alert"
        >{{ errorMessage
        }}<NButton v-if="conflict" type="error" secondary class="mt-3" @click="reloadAfterConflict"
          >放弃草稿并刷新</NButton
        ></NAlert
      >
      <NAlert v-if="fieldErrors._form || fieldErrors['']" type="error" class="mb-5" role="alert">{{
        fieldErrors._form || fieldErrors['']
      }}</NAlert>
      <NFormItem
        label="名称"
        path="name"
        data-field="name"
        :label-props="{ for: 'subscription-name', id: 'subscription-name-label' }"
        required
        :validation-status="fieldErrors.name ? 'error' : undefined"
        ><NInput
          v-model:value="draft.name"
          :input-props="{ id: 'subscription-name', ...fieldA11y('name') }"
          autofocus
          placeholder="Cloud Storage"
          :maxlength="100"
          :disabled="pending || discardOpen"
        /><template v-if="fieldErrors.name" #feedback
          ><span id="subscription-name-error">{{ fieldErrors.name }}</span></template
        ></NFormItem
      >
      <NFormItem
        label="图标"
        path="icon"
        data-field="icon"
        :label-props="{ for: 'subscription-icon', id: 'subscription-icon-label' }"
        :validation-status="fieldErrors.icon ? 'error' : undefined"
        ><SubscriptionIconInput
          v-model="draft.icon"
          input-id="subscription-icon"
          :invalid="!!fieldErrors.icon"
          :described-by="fieldErrors.icon ? 'subscription-icon-error' : undefined"
          :name="draft.name"
          :disabled="pending || discardOpen"
        /><template v-if="fieldErrors.icon" #feedback
          ><span id="subscription-icon-error">{{ fieldErrors.icon }}</span></template
        ></NFormItem
      >
      <NFormItem
        label="链接"
        path="website_url"
        data-field="website_url"
        :label-props="{ for: 'subscription-website_url', id: 'subscription-website_url-label' }"
        :validation-status="fieldErrors.website_url ? 'error' : undefined"
        ><NInput
          :value="draft.website_url ?? ''"
          :input-props="{
            id: 'subscription-website_url',
            inputmode: 'url',
            ...fieldA11y('website_url'),
          }"
          placeholder="https://example.com"
          :disabled="pending || discardOpen"
          @update:value="draft.website_url = $event || null"
        /><template v-if="fieldErrors.website_url" #feedback
          ><span id="subscription-website_url-error">{{ fieldErrors.website_url }}</span></template
        ></NFormItem
      >
      <div class="border-t border-default pt-6">
        <NFormItem
          label="类型"
          path="subscription_type"
          data-field="subscription_type"
          :label-props="{
            for: 'subscription-subscription_type',
            id: 'subscription-subscription_type-label',
          }"
          :validation-status="fieldErrors.subscription_type ? 'error' : undefined"
          ><NRadioGroup
            id="subscription-subscription_type"
            v-model:value="draft.subscription_type"
            class="subscription-type-switch"
            name="subscription-type"
            aria-labelledby="subscription-subscription_type-label"
            v-bind="fieldA11y('subscription_type')"
            :disabled="pending || discardOpen"
            @update:value="updateType"
            ><NRadioButton value="recurring"
              ><span class="subscription-type-label"
                ><RefreshCw :size="18" aria-hidden="true" />周期性</span
              ></NRadioButton
            ><NRadioButton value="one_time"
              ><span class="subscription-type-label"
                ><Calendar :size="18" aria-hidden="true" />一次性</span
              ></NRadioButton
            ></NRadioGroup
          ><template v-if="fieldErrors.subscription_type" #feedback
            ><span id="subscription-subscription_type-error">{{
              fieldErrors.subscription_type
            }}</span></template
          ></NFormItem
        >
        <div class="grid gap-x-4 sm:grid-cols-2">
          <NFormItem
            :label="draft.subscription_type === 'recurring' ? '每期金额' : '单次金额'"
            path="amount"
            data-field="amount_minor"
            :label-props="{
              for: 'subscription-amount_minor',
              id: 'subscription-amount_minor-label',
            }"
            required
            :validation-status="fieldErrors.amount_minor ? 'error' : undefined"
            ><NInput
              v-model:value="draft.amount"
              :input-props="{
                id: 'subscription-amount_minor',
                inputmode: 'decimal',
                ...fieldA11y('amount_minor'),
              }"
              placeholder="0.00"
              :disabled="pending || discardOpen"
            /><template v-if="fieldErrors.amount_minor" #feedback
              ><span id="subscription-amount_minor-error">{{
                fieldErrors.amount_minor
              }}</span></template
            ></NFormItem
          >
          <NFormItem
            label="币种"
            path="currency"
            data-field="currency"
            :label-props="{ for: 'subscription-currency', id: 'subscription-currency-label' }"
            required
            :validation-status="fieldErrors.currency ? 'error' : undefined"
            ><select
              id="subscription-currency"
              v-model="draft.currency"
              v-bind="fieldA11y('currency')"
              class="native-control"
              :disabled="pending || discardOpen"
            >
              <option v-for="currency in currencies" :key="currency" :value="currency">
                {{ currency }} · {{ currencyLabels[currency] }}
              </option></select
            ><template v-if="fieldErrors.currency" #feedback
              ><span id="subscription-currency-error">{{ fieldErrors.currency }}</span></template
            ></NFormItem
          >
        </div>
        <div v-if="draft.subscription_type === 'recurring'" class="grid grid-cols-2 gap-x-4">
          <NFormItem
            label="周期倍数"
            path="interval_count"
            data-field="interval_count"
            :label-props="{
              for: 'subscription-interval_count',
              id: 'subscription-interval_count-label',
            }"
            required
            :validation-status="fieldErrors.interval_count ? 'error' : undefined"
            ><NInputNumber
              placeholder="1"
              :show-button="false"
              :value="Number.isFinite(draft.interval_count) ? (draft.interval_count ?? null) : null"
              :input-props="{
                id: 'subscription-interval_count',
                inputmode: 'numeric',
                ...fieldA11y('interval_count'),
              }"
              :min="1"
              :precision="0"
              :disabled="pending || discardOpen"
              class="w-full"
              @update:value="draft.interval_count = $event ?? NaN"
            /><template v-if="fieldErrors.interval_count" #feedback
              ><span id="subscription-interval_count-error">{{
                fieldErrors.interval_count
              }}</span></template
            ></NFormItem
          >
          <NFormItem
            label="周期单位"
            path="interval_unit"
            data-field="interval_unit"
            :label-props="{
              for: 'subscription-interval_unit',
              id: 'subscription-interval_unit-label',
            }"
            required
            :validation-status="fieldErrors.interval_unit ? 'error' : undefined"
            ><select
              id="subscription-interval_unit"
              :value="draft.interval_unit ?? ''"
              v-bind="fieldA11y('interval_unit')"
              class="native-control"
              :disabled="pending || discardOpen"
              @change="
                draft.interval_unit = ($event.target as HTMLSelectElement).value as IntervalUnit
              "
            >
              <option v-for="unit in units" :key="unit.value" :value="unit.value">
                {{ unit.label }}
              </option></select
            ><template v-if="fieldErrors.interval_unit" #feedback
              ><span id="subscription-interval_unit-error">{{
                fieldErrors.interval_unit
              }}</span></template
            ></NFormItem
          >
        </div>
        <div class="grid gap-x-4 sm:grid-cols-2">
          <NFormItem
            v-if="draft.subscription_type === 'recurring'"
            label="开始日期"
            required
            path="start_date"
            data-field="start_date"
            :label-props="{
              for: 'subscription-start_date',
              id: 'subscription-start_date-label',
            }"
            :validation-status="fieldErrors.start_date ? 'error' : undefined"
            ><input
              id="subscription-start_date"
              :value="draft.start_date ?? ''"
              v-bind="fieldA11y('start_date')"
              type="date"
              min="0001-01-01"
              max="9999-12-31"
              aria-label="开始日期"
              class="native-control"
              :disabled="pending || discardOpen"
              @input="draft.start_date = ($event.target as HTMLInputElement).value || null"
            /><template v-if="fieldErrors.start_date" #feedback
              ><span id="subscription-start_date-error">{{
                fieldErrors.start_date
              }}</span></template
            ></NFormItem
          >
          <NFormItem
            label="到期日期"
            path="next_payment"
            data-field="next_payment"
            :label-props="{
              for: 'subscription-next_payment',
              id: 'subscription-next_payment-label',
            }"
            required
            :validation-status="fieldErrors.next_payment ? 'error' : undefined"
            ><div class="flex w-full items-center gap-2">
              <input
                id="subscription-next_payment"
                v-model="draft.next_payment"
                v-bind="fieldA11y('next_payment')"
                type="date"
                min="0001-01-01"
                max="9999-12-31"
                aria-label="到期日期"
                class="native-control min-w-0 flex-1"
                :disabled="pending || discardOpen"
              /><NTooltip v-if="draft.subscription_type === 'recurring'"
                ><template #trigger
                  ><NButton
                    quaternary
                    class="form-icon-action"
                    :disabled="
                      pending || discardOpen || !timezone || !isLocalDate(draft.start_date)
                    "
                    aria-label="根据开始日期和周期推算到期日期"
                    @click="calculateNextPayment"
                    ><template #icon><RefreshCw :size="18" /></template></NButton></template
                >根据开始日期和周期推算到期日</NTooltip
              >
            </div>
            <template v-if="fieldErrors.next_payment" #feedback
              ><span id="subscription-next_payment-error">{{
                fieldErrors.next_payment
              }}</span></template
            ></NFormItem
          >
        </div>
      </div>
      <div class="grid gap-x-4 border-t border-default pt-6 sm:grid-cols-[auto_minmax(0,1fr)]">
        <NFormItem
          label="开启提醒"
          path="notify_enabled"
          data-field="notify_enabled"
          :label-props="{
            for: 'subscription-notify_enabled',
            id: 'subscription-notify_enabled-label',
          }"
          :validation-status="fieldErrors.notify_enabled ? 'error' : undefined"
          ><div class="flex min-h-11 items-center">
            <NSwitch
              id="subscription-notify_enabled"
              v-model:value="draft.notify_enabled"
              v-bind="fieldA11y('notify_enabled')"
              :disabled="pending || discardOpen"
              aria-label="开启提醒"
            />
          </div>
          <template v-if="fieldErrors.notify_enabled" #feedback
            ><span id="subscription-notify_enabled-error">{{
              fieldErrors.notify_enabled
            }}</span></template
          ></NFormItem
        >
        <NFormItem
          label="提醒天数"
          path="reminder_days"
          data-field="reminder_days"
          :label-props="{
            for: 'subscription-reminder_days',
            id: 'subscription-reminder_days-label',
          }"
          :validation-status="fieldErrors.reminder_days ? 'error' : undefined"
          ><div class="grid w-full gap-3" :class="{ 'sm:grid-cols-2': reminderMode === 'custom' }">
            <select
              id="subscription-reminder_days"
              v-bind="fieldA11y('reminder_days')"
              v-model="reminderMode"
              class="native-control"
              :disabled="pending || discardOpen"
            >
              <option value="global">使用全局设置</option>
              <option value="custom">自定义</option></select
            ><NInputNumber
              v-if="reminderMode === 'custom'"
              placeholder="0–365"
              :show-button="false"
              :value="Number.isFinite(draft.reminder_days) ? (draft.reminder_days ?? null) : null"
              :input-props="{
                id: 'subscription-custom-reminder-days',
                ...fieldA11y('reminder_days'),
                'aria-describedby': fieldErrors.reminder_days
                  ? 'subscription-reminder_days-hint subscription-reminder_days-error'
                  : 'subscription-reminder_days-hint',
                inputmode: 'numeric',
                'aria-label': '自定义提醒天数',
              }"
              :min="0"
              :max="365"
              :precision="0"
              aria-label="自定义提醒天数"
              :disabled="pending || discardOpen"
              @update:value="draft.reminder_days = $event ?? NaN"
            />
          </div>
          <template #feedback>
            <span
              v-if="reminderMode === 'custom'"
              id="subscription-reminder_days-hint"
              class="block text-muted"
              >0 表示到期当天</span
            >
            <span v-if="fieldErrors.reminder_days" id="subscription-reminder_days-error">{{
              fieldErrors.reminder_days
            }}</span>
          </template></NFormItem
        >
      </div>
      <NFormItem
        label="备注"
        path="notes"
        data-field="notes"
        :label-props="{ for: 'subscription-notes', id: 'subscription-notes-label' }"
        :validation-status="fieldErrors.notes ? 'error' : undefined"
        ><NInput
          :value="draft.notes ?? ''"
          :input-props="{ id: 'subscription-notes', ...fieldA11y('notes') }"
          type="textarea"
          placeholder="补充说明"
          :maxlength="2000"
          :autosize="{ minRows: 2, maxRows: 5 }"
          :disabled="pending || discardOpen"
          @update:value="draft.notes = $event || null"
        /><template v-if="fieldErrors.notes" #feedback
          ><span id="subscription-notes-error">{{ fieldErrors.notes }}</span></template
        ></NFormItem
      >
    </NForm>
    <template #footer
      ><div class="modal-footer">
        <NButton :disabled="pending || discardOpen" @click="requestClose()">取消</NButton
        ><NButton
          type="primary"
          attr-type="submit"
          form="subscription-form"
          :loading="pending"
          :disabled="pending || discardOpen || (!!subscription && !dirty)"
          >保存</NButton
        >
      </div></template
    >
  </NModal>
  <DiscardChangesDialog v-model:open="discardOpen" @discard="discard" />
</template>

<style scoped>
.subscription-type-switch {
  display: flex;
  width: 100%;
  height: auto;
  gap: 4px;
  padding: 4px;
  border-radius: 10px;
  background: var(--ui-bg-muted);
}

.subscription-type-switch :deep(.n-radio-group__splitor),
.subscription-type-switch :deep(.n-radio-button__state-border) {
  display: none;
}

.subscription-type-switch :deep(.n-radio-button) {
  flex: 1;
  min-width: 0;
  height: 44px;
  line-height: 42px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--ui-text-muted);
  text-align: center;
  font-weight: 600;
}

.subscription-type-switch :deep(.n-radio-button:not(.n-radio-button--disabled):hover) {
  background: var(--ui-bg-elevated);
  color: var(--ui-text);
}

.subscription-type-switch :deep(.n-radio-button--checked),
.subscription-type-switch :deep(.n-radio-button--checked:not(.n-radio-button--disabled):hover) {
  border-color: var(--ui-primary);
  background: color-mix(in srgb, var(--ui-primary) 8%, var(--ui-bg-elevated));
  color: var(--ui-primary);
  box-shadow: none;
}

.subscription-type-switch :deep(.n-radio-button:has(input:focus-visible)) {
  outline: 2px solid var(--ui-primary);
  outline-offset: 1px;
}

.subscription-type-label {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 100%;
}
</style>
