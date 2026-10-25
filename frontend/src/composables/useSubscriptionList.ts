import { computed, onBeforeUnmount, ref, shallowRef, watch, type Ref } from 'vue'
import type { SubscriptionListResponse } from '@/types/subscriptions'
import { useApi } from '@/composables/useApi'
import { useBusinessDateRefresh } from '@/composables/useBusinessDateRefresh'

export function useSubscriptionList(query: Ref<Record<string, unknown>>) {
  const api = useApi()
  const lastSuccess = shallowRef<{ key: string; response: SubscriptionListResponse }>()
  const status = ref<'idle' | 'pending' | 'success' | 'error'>('idle')
  const error = shallowRef<unknown>()
  const requestKey = computed(() =>
    JSON.stringify(Object.entries(query.value).sort(([a], [b]) => a.localeCompare(b))),
  )
  let controller: AbortController | undefined
  async function refresh() {
    controller?.abort()
    const request = new AbortController()
    controller = request
    const key = requestKey.value
    status.value = 'pending'
    error.value = undefined
    try {
      const response = await api<SubscriptionListResponse>('/api/subscriptions', {
        query: query.value,
        signal: request.signal,
        retry: false,
      })
      if (request.signal.aborted || key !== requestKey.value) return
      lastSuccess.value = { key, response }
      status.value = 'success'
    } catch (failure) {
      if (request.signal.aborted) return
      error.value = failure
      status.value = 'error'
    }
  }
  watch(
    requestKey,
    () => {
      void refresh()
    },
    { immediate: true },
  )
  onBeforeUnmount(() => controller?.abort())
  const data = computed(() =>
    lastSuccess.value?.key === requestKey.value ? lastSuccess.value.response : undefined,
  )
  useBusinessDateRefresh(() => lastSuccess.value?.response.meta, refresh)
  return { data, status, error, refresh }
}
