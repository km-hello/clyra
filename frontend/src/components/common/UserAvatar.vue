<script setup lang="ts">
import { ref, watch } from 'vue'
import { UserRound } from '@lucide/vue'
const props = withDefaults(defineProps<{ src?: string | null; size?: number }>(), {
  src: null,
  size: 32,
})
const loaded = ref(false)
const failed = ref(false)
watch(
  () => props.src,
  () => {
    loaded.value = false
    failed.value = false
  },
)
</script>

<template>
  <span
    class="relative inline-flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-muted text-muted"
    :style="{ width: `${size}px`, height: `${size}px` }"
    aria-hidden="true"
  >
    <UserRound v-if="!loaded" class="size-1/2" />
    <img
      v-if="src && !failed"
      :key="src"
      :src="src"
      alt=""
      referrerpolicy="no-referrer"
      class="absolute inset-0 size-full object-cover"
      :class="{ 'opacity-0': !loaded }"
      @load="loaded = true"
      @error="
        () => {
          failed = true
          loaded = false
        }
      "
    />
  </span>
</template>
