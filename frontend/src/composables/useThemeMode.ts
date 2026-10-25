import { computed, reactive, ref, watch } from 'vue'
import type { ThemeMode } from '@/types/theme'

const mode = reactive<{ preference: ThemeMode }>({ preference: 'system' })
const system = window.matchMedia('(prefers-color-scheme: dark)')
const systemDark = ref(system.matches)
export const isDark = computed(
  () => mode.preference === 'dark' || (mode.preference === 'system' && systemDark.value),
)
watch(isDark, (dark) => document.documentElement.classList.toggle('dark', dark), {
  immediate: true,
})
system.addEventListener('change', () => {
  systemDark.value = system.matches
})

export function useThemeMode() {
  return mode
}
