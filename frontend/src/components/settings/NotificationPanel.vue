<script setup lang="ts">
import { ref, watch, onBeforeUnmount, useTemplateRef } from 'vue'
import { NAlert, NButton, NTag } from 'naive-ui'
import { Send } from '@lucide/vue'
import { useApi } from '@/composables/useApi'
import type { SettingsResponse } from '@/types/settings'
import type { NotificationTestResponse } from '@/types/notifications'
import NotificationHistory from '@/components/settings/NotificationHistory.vue'

const props = defineProps<{
  active: boolean
  settings: SettingsResponse | null
  dirty: boolean
  disabled: boolean
}>()
const testing = defineModel<boolean>('testing', { default: false })
const $api = useApi()
const notificationHistory =
  useTemplateRef<InstanceType<typeof NotificationHistory>>('notificationHistory')
const testMessage = ref('')
const testFailed = ref(false)
let controller: AbortController | undefined

watch(
  () => props.active,
  () => {
    controller?.abort()
    testing.value = false
    testMessage.value = ''
    testFailed.value = false
  },
)
watch(
  () => props.settings,
  (settings) => {
    if (props.active && settings) void notificationHistory.value?.refresh()
  },
)
onBeforeUnmount(() => {
  controller?.abort()
  testing.value = false
})

async function sendTest() {
  if (!props.active || props.disabled || testing.value || !props.settings) return
  testFailed.value = false
  testMessage.value = ''
  if (props.dirty) {
    testFailed.value = true
    testMessage.value = '请先保存设置，再发送测试'
    return
  }
  const request = new AbortController()
  controller = request
  testing.value = true
  try {
    const response = await $api<NotificationTestResponse>('/api/notifications/test', {
      method: 'POST',
      signal: request.signal,
      retry: false,
    })
    if (!request.signal.aborted)
      testMessage.value = `测试消息已确认发送 ${response.data.message_count} 条`
  } catch (error) {
    if (!request.signal.aborted) {
      const failure = error as { data?: { message?: string } }
      testFailed.value = true
      testMessage.value = failure.data?.message || '发送结果未确认，未自动重试'
    }
  } finally {
    if (controller === request) testing.value = false
    if (!request.signal.aborted) void notificationHistory.value?.refresh()
  }
}
</script>

<template>
  <section v-if="settings" class="space-y-3 border-t border-default pt-4 text-sm">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <h3 class="font-semibold">Telegram 通知</h3>
        <NTag
          :type="settings.bot_configured ? 'success' : 'default'"
          :bordered="false"
          size="small"
        >
          {{ settings.bot_configured ? '已配置' : '未配置' }}
        </NTag>
      </div>
      <NButton
        class="ms-auto"
        size="small"
        secondary
        :loading="testing"
        :disabled="disabled"
        @click="sendTest"
      >
        <template #icon><Send :size="16" /></template>发送测试
      </NButton>
    </div>
    <p v-if="!settings.bot_configured" class="text-muted">请在服务器配置 Bot Token。</p>
    <NAlert v-if="testMessage" :type="testFailed ? 'error' : 'success'" role="status">{{
      testMessage
    }}</NAlert>
    <div class="border-t border-default pt-3">
      <NotificationHistory v-if="active" ref="notificationHistory" />
    </div>
  </section>
</template>
