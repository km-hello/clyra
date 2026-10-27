<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { NAlert, NButton, NInput, NModal, NTab, NTabs } from 'naive-ui'
import { useApi } from '@/composables/useApi'
import type { ImageSearchItem } from '@/types/icons'

const props = defineProps<{
  name: string
  disabled?: boolean
}>()
const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ select: [url: string] }>()
const $api = useApi()
const query = ref('')
const source = ref('serpapi')
const sources = [
  { value: 'serpapi', label: 'SerpAPI' },
  { value: 'selfhst', label: 'selfh.st' },
  { value: 'dashboardicons', label: 'Dashboard Icons' },
]
const results = ref<(ImageSearchItem & { thumbnailFailed?: boolean })[]>([])
const searched = ref(false)
const pending = ref(false)
const selecting = ref(false)
const errorMessage = ref('')
let controller: AbortController | undefined
let cancelImage: (() => void) | undefined
let generation = 0

function cancel() {
  generation++
  controller?.abort()
  controller = undefined
  cancelImage?.()
  cancelImage = undefined
  pending.value = false
  selecting.value = false
}
function resetSearch() {
  cancel()
  results.value = []
  searched.value = false
  errorMessage.value = ''
}
watch(
  open,
  (value) => {
    resetSearch()
    if (value) {
      source.value = 'serpapi'
      query.value = props.name.trim().slice(0, 100)
    }
  },
  { immediate: true, flush: 'sync' },
)
onBeforeUnmount(cancel)

function updateQuery(value: string) {
  query.value = value
  resetSearch()
}
function changeSource(value: string | number) {
  if (typeof value !== 'string' || value === source.value) return
  resetSearch()
  source.value = value
  void search()
}
function onSearchEnter(event: KeyboardEvent) {
  if (event.isComposing) return
  event.preventDefault()
  void search()
}
async function search() {
  if (pending.value || selecting.value || !open.value) return
  const keyword = query.value.trim()
  if (!keyword || keyword.length > 100) {
    errorMessage.value = '请输入 1–100 个字符的关键词'
    return
  }
  resetSearch()
  const current = generation
  controller = new AbortController()
  pending.value = true
  try {
    const response = await $api<{ data: ImageSearchItem[] }>('/api/icons/search', {
      method: 'POST',
      body: { query: keyword, source: source.value },
      signal: controller.signal,
      retry: false,
    })
    if (current !== generation) return
    results.value = response.data
    searched.value = true
  } catch (error) {
    if (current !== generation) return
    const failure = error as { data?: { message?: string } }
    errorMessage.value = failure.data?.message || '图标搜索失败，请重试'
  } finally {
    if (current === generation) pending.value = false
  }
}
function select(item: ImageSearchItem) {
  if (selecting.value || pending.value || !open.value || props.disabled) return
  const current = generation
  selecting.value = true
  errorMessage.value = ''
  const image = new Image()
  image.referrerPolicy = 'no-referrer'
  const timer = setTimeout(() => finish(false), 15000)
  function cleanup() {
    clearTimeout(timer)
    image.onload = null
    image.onerror = null
    image.src = ''
  }
  function finish(success: boolean) {
    cleanup()
    if (current !== generation) return
    cancelImage = undefined
    selecting.value = false
    if (success) {
      emit('select', item.original)
      open.value = false
    } else errorMessage.value = '图片加载失败，请重新选择'
  }
  cancelImage = cleanup
  image.onload = () => finish(image.naturalWidth > 0)
  image.onerror = () => finish(false)
  image.src = item.original
}
</script>

<template>
  <NModal
    v-model:show="open"
    preset="card"
    content-scrollable
    title="搜索品牌图标"
    aria-label="搜索品牌图标"
    style="width: min(800px, calc(100vw - 32px))"
  >
    <div class="space-y-4">
      <div class="flex gap-3">
        <div class="min-w-0 flex-1">
          <NInput
            :value="query"
            autofocus
            :input-props="{ 'aria-label': '图标搜索关键词' }"
            :maxlength="100"
            placeholder="搜索品牌图标"
            class="w-full"
            @update:value="updateQuery"
            @keydown.enter.stop="onSearchEnter"
          />
        </div>
        <NButton
          type="primary"
          class="min-w-22"
          :loading="pending"
          :disabled="pending || selecting"
          @click="search"
          >搜索</NButton
        >
      </div>
      <NTabs
        :value="source"
        type="line"
        role="group"
        aria-label="图标搜索来源"
        @update:value="changeSource"
      >
        <NTab v-for="entry in sources" :key="entry.value" :name="entry.value">
          <button
            type="button"
            :aria-pressed="source === entry.value"
            class="rounded focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-primary"
          >
            {{ entry.label }}
          </button>
        </NTab>
      </NTabs>
      <div class="space-y-4">
        <NAlert v-if="errorMessage" type="error" role="alert">{{ errorMessage }}</NAlert>
        <p v-if="pending || selecting" role="status" class="text-sm text-muted">
          {{ pending ? '正在搜索图标…' : '正在加载图片…' }}
        </p>
        <p v-else-if="searched && !results.length" role="status" class="text-sm text-muted">
          未找到图标，请更换关键词
        </p>
        <div
          class="grid max-h-[min(692px,calc(100dvh-200px))] auto-rows-[164px] grid-cols-2 gap-3 overflow-y-auto sm:grid-cols-3 md:grid-cols-5"
        >
          <button
            v-for="item in results"
            :key="item.original"
            type="button"
            class="min-w-0 rounded-lg border border-default p-2 text-left hover:bg-accented focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-primary disabled:opacity-50"
            :disabled="pending || selecting"
            :aria-label="`选择 ${item.title}`"
            @click="select(item)"
            @keydown.enter.stop
          >
            <img
              v-if="!item.thumbnailFailed"
              :src="item.thumbnail"
              alt=""
              loading="lazy"
              referrerpolicy="no-referrer"
              class="h-24 w-full rounded object-contain"
              @error="item.thumbnailFailed = true"
            />
            <span
              v-else
              class="flex h-24 w-full items-center justify-center rounded bg-accented px-2 text-center text-xs text-muted"
              >缩略图加载失败</span
            >
            <span class="mt-2 block truncate text-sm" :title="item.title">{{ item.title }}</span>
            <span class="block truncate text-xs text-muted" :title="item.source">{{
              item.source
            }}</span>
          </button>
        </div>
      </div>
    </div>
  </NModal>
</template>
