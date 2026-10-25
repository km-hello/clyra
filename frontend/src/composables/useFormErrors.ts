import { reactive, watch } from 'vue'
import type { z } from 'zod'

export function useFormErrors(formId?: string) {
  const errors = reactive<Record<string, string>>({})
  if (formId) {
    watch(
      errors,
      () => {
        document
          .getElementById(formId)
          ?.querySelector<HTMLElement>('[aria-invalid="true"]')
          ?.focus()
      },
      { flush: 'post' },
    )
  }
  function clear() {
    for (const name of Object.keys(errors)) delete errors[name]
  }
  function setErrors(fields: Record<string, string>) {
    clear()
    Object.assign(errors, fields)
  }
  function validate<T>(schema: z.ZodType<T>, value: unknown) {
    clear()
    const result = schema.safeParse(value)
    if (!result.success) {
      for (const issue of result.error.issues) {
        const name = issue.path.join('.')
        errors[name] ??= issue.message
      }
    }
    return result
  }
  return { errors, clear, setErrors, validate }
}
