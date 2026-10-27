<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { NButton, NInput, NTooltip } from 'naive-ui'
import { Search } from '@lucide/vue'
import SubscriptionIconSearchModal from './SubscriptionIconSearchModal.vue'

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
const open = ref(false)
function showSearch() {
  if (!props.disabled) open.value = true
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
  <SubscriptionIconSearchModal
    v-model:open="open"
    :name="name"
    :disabled="disabled"
    @select="model = $event"
  />
</template>
