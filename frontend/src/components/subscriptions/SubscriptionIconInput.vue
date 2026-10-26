<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { NAlert, NButton, NInput, NModal, NTooltip } from 'naive-ui'
import { Search } from '@lucide/vue'
import { useApi } from '@/composables/useApi'
import type { ImageSearchItem } from '@/types/icons'

const props = defineProps<{
  name: string
  disabled?: boolean
  inputId?: string
  invalid?: boolean
  describedBy?: string
}>()
const model = defineModel<string | null>()
const previewIcon = computed(() => model.value?.trim() || '')
const imageFailed = ref(false)
const previewImage = computed(() => {
  try {
    const url = new URL(previewIcon.value)
    return (
      previewIcon.value.length <= 2048 &&
      url.protocol === 'https:' &&
      !url.username &&
      !url.password
    )
  } catch {
    return false
  }
})
const previewText = computed(() =>
  previewIcon.value &&
  !/^[a-z][a-z\d+.-]*:|^\/\//i.test(previewIcon.value) &&
  [...previewIcon.value].length <= 16
    ? previewIcon.value
    : Array.from(props.name.trim())[0] || '?',
)
watch(previewIcon, () => (imageFailed.value = false))
function onPreviewError(event: Event) {
  if ((event.target as HTMLImageElement).getAttribute('src') === previewIcon.value)
    imageFailed.value = true
}
const $api = useApi()
const open = ref(false)
const query = ref('')
const results = ref<ImageSearchItem[]>([])
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
watch(
  open,
  (value) => {
    if (!value) cancel()
  },
  { flush: 'sync' },
)
onBeforeUnmount(cancel)

function showSearch() {
  if (props.disabled) return
  query.value = props.name.trim().slice(0, 100)
  results.value = []
  searched.value = false
  errorMessage.value = ''
  open.value = true
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
  cancel()
  const current = generation
  controller = new AbortController()
  pending.value = true
  errorMessage.value = ''
  results.value = []
  searched.value = false
  try {
    const response = await $api<{ data: ImageSearchItem[] }>('/api/icons/search', {
      method: 'POST',
      body: { query: keyword },
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
    cancelImage = undefined
    if (current !== generation) return
    selecting.value = false
    if (success) {
      model.value = item.original
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
  <div class="flex w-full items-center gap-3">
    <NInput
      :value="model ?? ''"
      :input-props="{
        id: inputId,
        'aria-label': '图标内容或图片地址',
        'aria-invalid': invalid,
        'aria-describedby': describedBy,
      }"
      placeholder="文字、表情或图片链接"
      class="min-w-0 flex-1"
      :disabled="disabled"
      @update:value="model = $event || null"
    >
      <template #suffix>
        <NTooltip>
          <template #trigger>
            <NButton
              quaternary
              class="form-icon-action"
              :disabled="disabled"
              aria-label="搜索品牌图标"
              @click="showSearch"
              ><template #icon><Search :size="18" /></template
            ></NButton>
          </template>
          搜索品牌图标
        </NTooltip>
      </template>
    </NInput>
    <div class="subscription-icon" role="img" aria-label="图标预览">
      <img
        v-if="previewImage && !imageFailed"
        :key="previewIcon"
        :src="previewIcon"
        alt=""
        referrerpolicy="no-referrer"
        class="h-full w-full rounded-xl object-contain"
        @error="onPreviewError"
      />
      <span v-else class="truncate px-1">{{ previewText }}</span>
    </div>
  </div>
  <NModal
    v-model:show="open"
    preset="card"
    content-scrollable
    title="搜索品牌图标"
    aria-label="搜索品牌图标"
    style="width: min(800px, calc(100vw - 32px))"
  >
    <div class="space-y-4" @keydown.enter.stop="onSearchEnter">
      <div class="flex gap-3">
        <div class="min-w-0 flex-1">
          <NInput
            v-model:value="query"
            autofocus
            :input-props="{ 'aria-label': '图标搜索关键词' }"
            :maxlength="100"
            placeholder="搜索品牌图标"
            class="w-full"
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
      <NAlert v-if="errorMessage" type="error" role="alert">{{ errorMessage }}</NAlert>
      <p v-if="pending || selecting" role="status" class="text-sm text-muted">
        {{ pending ? '正在搜索图标…' : '正在加载图片…' }}
      </p>
      <p v-else-if="searched && !results.length" role="status" class="text-sm text-muted">
        未找到图标，请更换关键词
      </p>
      <div
        class="grid max-h-[min(692px,calc(100dvh-200px))] auto-rows-[164px] grid-cols-5 gap-3 overflow-y-auto"
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
            :src="item.thumbnail"
            alt=""
            loading="lazy"
            referrerpolicy="no-referrer"
            class="h-24 w-full rounded object-contain"
          />
          <span class="mt-2 block truncate text-sm" :title="item.title">{{ item.title }}</span>
          <span class="block truncate text-xs text-muted" :title="item.source">{{
            item.source
          }}</span>
        </button>
      </div>
    </div>
  </NModal>
</template>
