<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { NAlert, NButton, NTag } from 'naive-ui'
import { useApi } from '@/composables/useApi'
import type { NotificationRunsResponse } from '@/types/notifications'

const $api = useApi()

const loading = ref(false)
const errorMessage = ref('')
const history = ref<NotificationRunsResponse | null>(null)
let controller: AbortController | undefined
const statuses = { running: '运行中', succeeded: '成功', failed: '失败' }

async function refresh() {
  controller?.abort()
  const request = new AbortController()
  controller = request
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await $api<NotificationRunsResponse>('/api/notifications/runs', {
      signal: request.signal,
      retry: false,
    })
    if (!request.signal.aborted) history.value = response
  } catch {
    if (!request.signal.aborted) errorMessage.value = '通知记录加载失败'
  } finally {
    if (controller === request) loading.value = false
  }
}

function formatTime(timestamp: number): string {
  return new Intl.DateTimeFormat('zh-CN', {
    timeZone: history.value!.meta.timezone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23',
  }).format(timestamp)
}

onMounted(() => void refresh())
onBeforeUnmount(() => controller?.abort())
defineExpose({ refresh })
</script>

<template>
  <div class="space-y-3">
    <p v-if="loading" class="text-sm text-muted" role="status">正在加载通知记录…</p>
    <NAlert v-if="errorMessage" type="error" role="alert">
      {{ errorMessage }}
      <div class="mt-3"><NButton @click="refresh">重试</NButton></div>
    </NAlert>
    <template v-if="history">
      <p v-if="!history.data.length && !loading" class="py-4 text-center text-sm text-muted">
        暂无通知记录
      </p>
      <ul v-else class="divide-y divide-default" aria-label="最近通知记录">
        <li
          v-for="run in history.data"
          :key="run.id"
          class="space-y-1 py-3 text-sm first:pt-0 last:pb-0"
        >
          <div class="flex items-center justify-between gap-2">
            <p class="font-medium">
              {{ run.run_type === 'scheduled' ? '自动提醒' : '手动测试' }}
              <span v-if="run.scheduled_time" class="font-normal text-muted">
                · {{ run.scheduled_time }}</span
              >
            </p>
            <NTag
              :type="
                run.status === 'failed'
                  ? 'error'
                  : run.status === 'succeeded'
                    ? 'success'
                    : 'default'
              "
              :bordered="false"
              size="small"
              >{{ statuses[run.status] }}</NTag
            >
          </div>
          <div class="flex flex-wrap justify-between gap-x-3 gap-y-1 text-xs text-muted">
            <time class="shrink-0" :datetime="new Date(run.started_at).toISOString()">{{
              formatTime(run.started_at)
            }}</time>
            <span class="ms-auto text-right"
              >{{ run.matched_count }} 个订阅 · 已确认发送 {{ run.message_count }} 条</span
            >
          </div>
          <p v-if="run.error_message" class="break-words text-xs text-error">
            {{ run.error_message }}
          </p>
          <p v-if="run.status === 'running'" class="text-xs text-muted">
            任务尚未完成，可能仍在运行或已中断；不会自动重复执行。
          </p>
        </li>
      </ul>
    </template>
  </div>
</template>
