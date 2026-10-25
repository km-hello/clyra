<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { NAlert, NButton, NModal, useMessage } from 'naive-ui'
import { ArrowRight } from '@lucide/vue'
import { useApi } from '@/composables/useApi'
import type { Subscription } from '@/types/subscriptions'
import { advanceDate } from '@/utils/dates'

const $api = useApi()

const props = defineProps<{
  open: boolean
  subscription: Subscription | null
  action: 'deactivate' | 'delete' | 'renew' | null
}>()
const emit = defineEmits<{ close: []; saved: [] }>()
const message = useMessage()
const pending = ref(false)
const errorMessage = ref('')
watch(
  () => props.open,
  (open) => {
    if (open) errorMessage.value = ''
  },
)
const deleting = computed(() => props.action === 'delete')
const renewing = computed(() => props.action === 'renew')
const nextPayment = computed(() => {
  const item = props.subscription
  if (!renewing.value || !item?.interval_count || !item.interval_unit) return null
  try {
    return advanceDate(item.next_payment, item.interval_count, item.interval_unit, item)
  } catch {
    return null
  }
})

async function focusCancel() {
  await nextTick()
  document.getElementById('subscription-action-cancel')?.focus()
}

async function confirm() {
  if (!props.open || !props.subscription || pending.value || (renewing.value && !nextPayment.value))
    return
  pending.value = true
  errorMessage.value = ''
  try {
    const url = `/api/subscriptions/${props.subscription.id}` as const
    if (renewing.value) {
      await $api(`${url}/renew`, {
        method: 'POST',
        body: { expected_version: props.subscription.version },
        retry: false,
      })
    } else if (deleting.value) {
      await $api(url, {
        method: 'DELETE',
        query: { expected_version: props.subscription.version },
        retry: false,
      })
    } else {
      await $api(url, {
        method: 'PATCH',
        body: { expected_version: props.subscription.version, status: 'inactive' },
        retry: false,
      })
    }
    message.success(
      renewing.value ? '已续订一个周期' : deleting.value ? '订阅已删除' : '订阅已停用',
    )
    emit('saved')
    emit('close')
  } catch (error) {
    const failure = error as { statusCode?: number; data?: { message?: string } }
    errorMessage.value =
      failure.statusCode === 409
        ? '订阅已更新，请关闭弹窗并刷新列表'
        : failure.data?.message || '操作失败，请重试'
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
    :title="renewing ? '续订一次' : deleting ? '删除订阅' : '停用订阅'"
    :aria-label="renewing ? '续订一次' : deleting ? '删除订阅' : '停用订阅'"
    :mask-closable="!pending"
    :close-on-esc="!pending"
    :closable="!pending"
    style="width: min(440px, calc(100vw - 32px))"
    class="confirm-modal"
    @after-enter="focusCancel"
    @update:show="
      (value) => {
        if (!value && !pending) emit('close')
      }
    "
  >
    <NAlert v-if="errorMessage" type="error" class="mb-4" role="alert">{{ errorMessage }}</NAlert>
    <div class="space-y-5">
      <p class="break-words text-base font-semibold">{{ subscription?.name }}</p>
      <div
        v-if="renewing"
        class="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-3"
      >
        <dl class="space-y-1">
          <dt class="text-xs text-muted">当前到期</dt>
          <dd class="whitespace-nowrap text-base text-muted tabular-nums sm:text-lg">
            {{ subscription?.next_payment }}
          </dd>
        </dl>
        <ArrowRight :size="18" class="text-muted" aria-hidden="true" />
        <dl class="space-y-1 justify-self-end">
          <dt class="text-xs text-muted">续订后</dt>
          <dd class="whitespace-nowrap text-lg font-semibold text-primary tabular-nums sm:text-xl">
            {{ nextPayment ?? '超出范围' }}
          </dd>
        </dl>
      </div>
      <p v-else class="text-sm text-muted">
        {{ deleting ? '删除后无法恢复。' : '暂停提醒，可随时恢复。' }}
      </p>
      <NAlert v-if="renewing && !nextPayment" type="error">
        续订日期超出支持范围，请编辑周期或到期日。
      </NAlert>
    </div>
    <template #footer
      ><div class="modal-footer">
        <NButton id="subscription-action-cancel" :disabled="pending" @click="emit('close')"
          >取消</NButton
        ><NButton
          :type="deleting ? 'error' : 'primary'"
          :secondary="deleting"
          :loading="pending"
          :disabled="pending || (renewing && !nextPayment)"
          @click="confirm"
          >{{ renewing ? '确认续订' : deleting ? '删除' : '停用' }}</NButton
        >
      </div></template
    >
  </NModal>
</template>
