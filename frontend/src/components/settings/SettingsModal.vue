<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import {
  NAlert,
  NButton,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSkeleton,
  NSwitch,
  useMessage,
} from 'naive-ui'
import DiscardChangesDialog from '@/components/common/DiscardChangesDialog.vue'
import { useFormErrors } from '@/composables/useFormErrors'
import NotificationPanel from '@/components/settings/NotificationPanel.vue'
import { useApi } from '@/composables/useApi'
import { settingsSchema } from '@/schemas/settings'
import type { SettingsValues, SettingsResponse } from '@/types/settings'

const $api = useApi()

const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ saved: [] }>()
const reminderSettingsSchema = settingsSchema.omit({ theme: true })
type ReminderSettingsValues = Omit<SettingsValues, 'theme'>
const message = useMessage()
const { errors, clear, setErrors, validate } = useFormErrors('reminder-settings-form')
const testing = ref(false)
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const errorMessage = ref('')
const savedSettings = ref<SettingsResponse | null>(null)
const draft = ref<ReminderSettingsValues | null>(null)
const discardOpen = ref(false)
let controller: AbortController | undefined
const busy = computed(() => saving.value || testing.value)
const dirty = computed(
  () =>
    !!draft.value &&
    Object.entries(draft.value).some(
      ([key, value]) =>
        !Object.is(value, savedSettings.value?.[key as keyof ReminderSettingsValues]),
    ),
)

async function load() {
  controller?.abort()
  const request = new AbortController()
  controller = request
  loading.value = true
  loadError.value = ''
  errorMessage.value = ''
  clear()
  savedSettings.value = null
  draft.value = null
  try {
    const response = await $api<{ data: SettingsResponse }>('/api/settings', {
      signal: request.signal,
      retry: false,
    })
    if (request.signal.aborted) return
    savedSettings.value = response.data
    const {
      scheduler_enabled,
      default_reminder_days,
      timezone,
      notification_time,
      telegram_chat_id,
      telegram_message_thread_id,
    } = response.data
    draft.value = {
      scheduler_enabled,
      default_reminder_days,
      timezone,
      notification_time,
      telegram_chat_id,
      telegram_message_thread_id,
    }
  } catch {
    if (!request.signal.aborted) loadError.value = '设置加载失败'
  } finally {
    if (controller === request) loading.value = false
  }
}
watch(open, (value) => {
  if (value) {
    discardOpen.value = false
    void load()
  } else {
    controller?.abort()
    loading.value = false
  }
})
onBeforeUnmount(() => controller?.abort())

function requestClose() {
  if (busy.value) return
  if (dirty.value) discardOpen.value = true
  else close()
}
function close() {
  discardOpen.value = false
  open.value = false
}
async function save() {
  if (busy.value || discardOpen.value || !dirty.value || !savedSettings.value || !draft.value)
    return
  const parsed = validate(reminderSettingsSchema, draft.value)
  if (!parsed.success) return
  const body = Object.fromEntries(
    Object.entries(parsed.data).filter(
      ([key, value]) =>
        !Object.is(value, savedSettings.value?.[key as keyof ReminderSettingsValues]),
    ),
  )
  if (!Object.keys(body).length) {
    draft.value = { ...parsed.data }
    open.value = false
    return
  }
  saving.value = true
  errorMessage.value = ''
  try {
    await $api('/api/settings', {
      method: 'PATCH',
      body,
      retry: false,
    })
    message.success('设置已保存')
    emit('saved')
    open.value = false
  } catch (error) {
    const failure = error as {
      data?: { message?: string; data?: { fieldErrors?: Record<string, string> } }
    }
    errorMessage.value = failure.data?.message || '设置保存失败，输入已保留'
    const fields = failure.data?.data?.fieldErrors
    if (fields) setErrors(fields)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <NModal
    :show="open"
    preset="card"
    content-scrollable
    title="提醒设置"
    aria-label="提醒设置"
    class="settings-modal form-modal"
    style="width: min(560px, calc(100vw - 32px))"
    :closable="!busy"
    :mask-closable="!busy"
    :close-on-esc="!busy && !discardOpen"
    @update:show="
      (value) => {
        if (!value) requestClose()
      }
    "
  >
    <div>
      <NAlert v-if="errorMessage" type="error" role="alert" class="mb-6">{{ errorMessage }}</NAlert>
      <div v-if="loading" class="space-y-4" aria-label="正在加载设置">
        <NSkeleton v-for="index in 4" :key="index" height="56px" :sharp="false" />
      </div>
      <NAlert v-else-if="loadError" type="error" role="alert">
        {{ loadError }}
        <div class="mt-3"><NButton @click="load">重试</NButton></div>
      </NAlert>
      <template v-else-if="draft && savedSettings">
        <NForm
          id="reminder-settings-form"
          novalidate
          :model="draft"
          :disabled="busy || discardOpen"
          @submit.prevent="save"
        >
          <div class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-6">
            <NFormItem
              label="自动提醒"
              path="scheduler_enabled"
              :label-props="{ for: 'reminder-scheduler_enabled' }"
              required
              :validation-status="errors.scheduler_enabled ? 'error' : undefined"
            >
              <div class="flex min-h-11 items-center">
                <NSwitch
                  id="reminder-scheduler_enabled"
                  v-model:value="draft.scheduler_enabled"
                  aria-label="自动提醒"
                  :aria-disabled="busy || discardOpen"
                  :aria-invalid="!!errors.scheduler_enabled"
                  :aria-describedby="
                    errors.scheduler_enabled ? 'reminder-scheduler_enabled-error' : undefined
                  "
                  :disabled="busy || discardOpen"
                />
              </div>
              <template v-if="errors.scheduler_enabled" #feedback>
                <span id="reminder-scheduler_enabled-error">{{ errors.scheduler_enabled }}</span>
              </template>
            </NFormItem>
            <NFormItem
              label="全局提醒天数"
              path="default_reminder_days"
              :label-props="{ for: 'reminder-default_reminder_days' }"
              required
              :validation-status="errors.default_reminder_days ? 'error' : undefined"
            >
              <NInputNumber
                placeholder="0–365"
                :show-button="false"
                :input-props="{
                  id: 'reminder-default_reminder_days',
                  required: true,
                  'aria-label': '全局提醒天数',
                  'aria-invalid': !!errors.default_reminder_days,
                  'aria-describedby': errors.default_reminder_days
                    ? 'reminder-default_reminder_days-error'
                    : undefined,
                }"
                :value="
                  Number.isFinite(draft.default_reminder_days) ? draft.default_reminder_days : null
                "
                :min="0"
                :max="365"
                :step="1"
                :precision="0"
                class="w-full"
                @update:value="
                  (value) => {
                    if (draft) draft.default_reminder_days = value ?? NaN
                  }
                "
              />
              <template v-if="errors.default_reminder_days" #feedback>
                <span id="reminder-default_reminder_days-error">{{
                  errors.default_reminder_days
                }}</span>
              </template>
            </NFormItem>
          </div>
          <NFormItem
            label="业务时区"
            path="timezone"
            :label-props="{ for: 'reminder-timezone' }"
            required
            :validation-status="errors.timezone ? 'error' : undefined"
          >
            <NInput
              v-model:value="draft.timezone"
              placeholder="Asia/Singapore"
              :input-props="{
                id: 'reminder-timezone',
                required: true,
                'aria-label': '业务时区',
                'aria-invalid': !!errors.timezone,
                'aria-describedby': errors.timezone ? 'reminder-timezone-error' : undefined,
              }"
            />
            <template v-if="errors.timezone" #feedback>
              <span id="reminder-timezone-error">{{ errors.timezone }}</span>
            </template>
          </NFormItem>
          <NFormItem
            label="每日提醒时间"
            path="notification_time"
            :label-props="{ for: 'reminder-notification_time' }"
            required
            :validation-status="errors.notification_time ? 'error' : undefined"
          >
            <input
              id="reminder-notification_time"
              v-model="draft.notification_time"
              type="time"
              required
              aria-label="每日提醒时间"
              :aria-invalid="!!errors.notification_time"
              :aria-describedby="
                errors.notification_time ? 'reminder-notification_time-error' : undefined
              "
              :disabled="busy || discardOpen"
              class="native-control"
            />
            <template v-if="errors.notification_time" #feedback>
              <span id="reminder-notification_time-error">{{ errors.notification_time }}</span>
            </template>
          </NFormItem>
          <div class="grid grid-cols-1 gap-x-4 min-[480px]:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
            <NFormItem
              label="Telegram Chat ID"
              path="telegram_chat_id"
              :label-props="{ for: 'reminder-telegram_chat_id' }"
              :validation-status="errors.telegram_chat_id ? 'error' : undefined"
            >
              <NInput
                :value="draft.telegram_chat_id ?? ''"
                :input-props="{
                  id: 'reminder-telegram_chat_id',
                  'aria-label': 'Telegram Chat ID',
                  'aria-invalid': !!errors.telegram_chat_id,
                  'aria-describedby': errors.telegram_chat_id
                    ? 'reminder-telegram_chat_id-error'
                    : undefined,
                }"
                placeholder="-1001234567890"
                @update:value="
                  (value) => {
                    if (draft) draft.telegram_chat_id = value || null
                  }
                "
              />
              <template v-if="errors.telegram_chat_id" #feedback>
                <span id="reminder-telegram_chat_id-error">{{ errors.telegram_chat_id }}</span>
              </template>
            </NFormItem>
            <NFormItem
              label="Telegram Topic ID"
              path="telegram_message_thread_id"
              :label-props="{ for: 'reminder-telegram_message_thread_id' }"
              :validation-status="errors.telegram_message_thread_id ? 'error' : undefined"
            >
              <NInputNumber
                v-model:value="draft.telegram_message_thread_id"
                placeholder="123"
                :show-button="false"
                class="w-full"
                :input-props="{
                  id: 'reminder-telegram_message_thread_id',
                  'aria-label': 'Telegram Topic ID',
                  'aria-invalid': !!errors.telegram_message_thread_id,
                  'aria-describedby': errors.telegram_message_thread_id
                    ? 'reminder-telegram_message_thread_id-error'
                    : undefined,
                }"
              />
              <template v-if="errors.telegram_message_thread_id" #feedback>
                <span id="reminder-telegram_message_thread_id-error">{{
                  errors.telegram_message_thread_id
                }}</span>
              </template>
            </NFormItem>
          </div>
        </NForm>
      </template>
      <NotificationPanel
        v-model:testing="testing"
        :active="open"
        :settings="savedSettings"
        :dirty="dirty"
        :disabled="busy || discardOpen"
      />
    </div>
    <template #footer>
      <div class="modal-footer">
        <NButton :disabled="busy || discardOpen" @click="requestClose()">取消</NButton>
        <NButton
          type="primary"
          attr-type="submit"
          form="reminder-settings-form"
          :loading="saving"
          :disabled="busy || discardOpen || loading || !draft || !dirty"
          >保存设置</NButton
        >
      </div>
    </template>
  </NModal>
  <DiscardChangesDialog v-model:open="discardOpen" @discard="close" />
</template>
