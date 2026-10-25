<script setup lang="ts">
import { useTemplateRef } from 'vue'
import { NButton, NModal } from 'naive-ui'

const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ discard: [] }>()
const cancel = useTemplateRef<InstanceType<typeof NButton>>('cancel')
</script>

<template>
  <NModal
    v-model:show="open"
    preset="card"
    content-scrollable
    class="confirm-modal discard-modal"
    title="放弃修改？"
    aria-label="放弃修改？"
    style="width: min(420px, calc(100vw - 32px))"
    @after-enter="cancel?.$el.focus()"
  >
    <p>未保存的修改将丢失。</p>
    <template #footer>
      <div class="modal-footer">
        <NButton ref="cancel" @click="open = false">继续编辑</NButton>
        <NButton type="error" secondary @click="emit('discard')">放弃修改</NButton>
      </div>
    </template>
  </NModal>
</template>
