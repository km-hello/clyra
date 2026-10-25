<script setup lang="ts">
import { ref, computed, reactive, watch } from 'vue'
import { NAlert, NButton, NForm, NFormItem, NInput, NModal, useMessage } from 'naive-ui'
import DiscardChangesDialog from '@/components/common/DiscardChangesDialog.vue'
import { useFormErrors } from '@/composables/useFormErrors'
import { useApi } from '@/composables/useApi'
import { useUserSession } from '@/composables/useUserSession'
import { useThemeMode } from '@/composables/useThemeMode'
import { useRouter } from 'vue-router'
import { passwordSchema, type PasswordInput } from '@/schemas/password'

const $api = useApi()

const router = useRouter()

const open = defineModel<boolean>('open', { default: false })
const discardOpen = ref(false)
const state = reactive<PasswordInput>({
  current_password: '',
  new_password: '',
  new_password_confirm: '',
})
const pending = ref(false)
const changed = ref(false)
const message = useMessage()
const errorMessage = ref('')
const { errors, clear, validate, setErrors } = useFormErrors('password-form')
const session = useUserSession()
const colorMode = useThemeMode()
const dirty = computed(() => Object.values(state).some((value) => value !== ''))

function reset() {
  Object.assign(state, { current_password: '', new_password: '', new_password_confirm: '' })
  errorMessage.value = ''
  changed.value = false
  clear()
}
watch(open, (value) => {
  if (!value) return
  discardOpen.value = false
  reset()
})

function requestClose(value = false) {
  if (value || pending.value) return
  if (dirty.value) discardOpen.value = true
  else open.value = false
}

function discard() {
  if (pending.value) return
  reset()
  discardOpen.value = false
  open.value = false
}

async function submit() {
  if (pending.value || changed.value || discardOpen.value || !dirty.value) return
  const result = validate(passwordSchema, state)
  if (!result.success) return
  pending.value = true
  errorMessage.value = ''
  try {
    await $api('/api/auth/password', { method: 'PATCH', body: result.data, retry: false })
  } catch (error) {
    const failure = error as {
      data?: { message?: string; data?: { fieldErrors?: Record<string, string> } }
    }
    errorMessage.value = failure.data?.message || '密码修改失败，输入已保留'
    const fields = failure.data?.data?.fieldErrors
    if (fields) {
      setErrors(fields)
    }
    return
  } finally {
    pending.value = false
  }
  reset()
  changed.value = true
  session.session.value = {}
  colorMode.preference = 'system'
  message.success('密码已修改，请重新登录')
  try {
    await router.replace('/login')
  } catch {
    // The password is already changed; keep the success state and login link available.
  }
}
</script>

<template>
  <NModal
    :show="open"
    preset="card"
    content-scrollable
    class="form-modal"
    title="修改密码"
    aria-label="修改密码"
    :style="{ width: 'min(420px, calc(100vw - 32px))' }"
    :mask-closable="!pending"
    :close-on-esc="!pending && !discardOpen"
    :closable="!pending"
    @update:show="requestClose"
  >
    <NForm
      id="password-form"
      :model="state"
      :disabled="pending || changed || discardOpen"
      @submit.prevent="submit"
    >
      <NAlert v-if="changed" type="success" role="status" class="mb-4">
        密码已修改，请重新登录。
      </NAlert>
      <NAlert v-if="errorMessage" type="error" role="alert" class="mb-4">{{ errorMessage }}</NAlert>
      <NFormItem
        label="当前密码"
        path="current_password"
        :label-props="{ for: 'password-current-password' }"
        :validation-status="errors.current_password ? 'error' : undefined"
        required
      >
        <NInput
          v-model:value="state.current_password"
          placeholder="输入当前密码"
          type="password"
          show-password-on="click"
          :input-props="{
            id: 'password-current-password',
            'aria-invalid': !!errors.current_password,
            'aria-required': true,
            'aria-describedby': errors.current_password
              ? 'password-current-password-error'
              : undefined,
            autocomplete: 'current-password',
          }"
          :maxlength="32"
        />
        <template v-if="errors.current_password" #feedback
          ><span id="password-current-password-error">{{ errors.current_password }}</span></template
        ></NFormItem
      >
      <NFormItem
        label="新密码"
        path="new_password"
        :label-props="{ for: 'password-new-password' }"
        :validation-status="errors.new_password ? 'error' : undefined"
        required
      >
        <NInput
          v-model:value="state.new_password"
          placeholder="6–32 个字符"
          type="password"
          show-password-on="click"
          :input-props="{
            id: 'password-new-password',
            'aria-invalid': !!errors.new_password,
            'aria-required': true,
            'aria-describedby': errors.new_password ? 'password-new-password-error' : undefined,
            autocomplete: 'new-password',
          }"
          :maxlength="32"
        />
        <template v-if="errors.new_password" #feedback
          ><span id="password-new-password-error">{{ errors.new_password }}</span></template
        ></NFormItem
      >
      <NFormItem
        label="确认新密码"
        path="new_password_confirm"
        :label-props="{ for: 'password-new-password-confirm' }"
        :validation-status="errors.new_password_confirm ? 'error' : undefined"
        required
      >
        <NInput
          v-model:value="state.new_password_confirm"
          placeholder="再次输入新密码"
          type="password"
          show-password-on="click"
          :input-props="{
            id: 'password-new-password-confirm',
            'aria-invalid': !!errors.new_password_confirm,
            'aria-required': true,
            'aria-describedby': errors.new_password_confirm
              ? 'password-new-password-confirm-error'
              : undefined,
            autocomplete: 'new-password',
          }"
          :maxlength="32"
        />
        <template v-if="errors.new_password_confirm" #feedback
          ><span id="password-new-password-confirm-error">{{
            errors.new_password_confirm
          }}</span></template
        ></NFormItem
      >
    </NForm>
    <template #footer>
      <div class="modal-footer">
        <NButton :disabled="pending || discardOpen" @click="requestClose()">取消</NButton>
        <NButton
          v-if="!changed"
          attr-type="submit"
          form="password-form"
          type="primary"
          :loading="pending"
          :disabled="pending || discardOpen || !dirty"
          >保存</NButton
        >
        <NButton v-else tag="a" href="/login" type="primary">前往登录</NButton>
      </div>
    </template>
  </NModal>
  <DiscardChangesDialog v-model:open="discardOpen" @discard="discard" />
</template>
