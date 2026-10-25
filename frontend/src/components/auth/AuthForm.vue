<script setup lang="ts">
import { ref, reactive } from 'vue'
import { NAlert, NButton, NCard, NForm, NFormItem, NInput } from 'naive-ui'
import { ArrowRight, Link, LockKeyhole, UserRound } from '@lucide/vue'
import ClyraBrand from '@/components/common/ClyraBrand.vue'
import { useFormErrors } from '@/composables/useFormErrors'
import { useUserSession } from '@/composables/useUserSession'
import { useRouter } from 'vue-router'
import { ofetch as $fetch } from 'ofetch'
import { loginSchema, setupSchema } from '@/schemas/auth'

const router = useRouter()

const props = defineProps<{ setup?: boolean }>()
const state = reactive(
  props.setup
    ? { username: '', password: '', password_confirm: '', nickname: '', avatar_url: '' }
    : { username: '', password: '' },
)
const { errors, validate, setErrors } = useFormErrors()
const pending = ref(false)
const errorMessage = ref('')
const session = useUserSession()
async function submit() {
  if (pending.value) return
  const result = props.setup ? validate(setupSchema, state) : validate(loginSchema, state)
  if (!result.success) return
  pending.value = true
  errorMessage.value = ''
  try {
    await $fetch(props.setup ? '/api/setup' : '/api/auth/login', {
      method: 'POST',
      body: result.data,
      retry: false,
    })
    await session.fetch()
    if (!session.loggedIn.value) {
      errorMessage.value = '会话恢复失败，请重试登录'
      return
    }
    await router.push('/')
  } catch (error) {
    const failure = error as {
      data?: { message?: string; data?: { fieldErrors?: Record<string, string> } }
    }
    errorMessage.value = failure.data?.message || '连接失败，请重试'
    const fields = failure.data?.data?.fieldErrors
    if (fields) {
      setErrors(fields)
    }
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <main class="auth-wrap">
    <NCard class="auth-card" :bordered="false">
      <ClyraBrand />
      <h1 class="auth-title">{{ setup ? '创建管理员账号' : '登录' }}</h1>
      <NForm :model="state" :disabled="pending" class="auth-form" @submit.prevent="submit">
        <NAlert v-if="errorMessage" type="error" role="alert" class="mb-4">{{
          errorMessage
        }}</NAlert>
        <NFormItem
          label="用户名"
          path="username"
          :label-props="{ for: 'auth-username' }"
          :validation-status="errors.username ? 'error' : undefined"
          required
        >
          <NInput
            v-model:value="state.username"
            :placeholder="setup ? '4–32 位小写字母' : '输入用户名'"
            :input-props="{
              id: 'auth-username',
              'aria-invalid': !!errors.username,
              'aria-required': true,
              'aria-describedby': errors.username ? 'auth-username-error' : undefined,
              autocomplete: 'username',
              autocapitalize: 'none',
              spellcheck: false,
            }"
            :maxlength="32"
          >
            <template #prefix><UserRound :size="18" /></template>
          </NInput>
          <template v-if="errors.username" #feedback
            ><span id="auth-username-error">{{ errors.username }}</span></template
          ></NFormItem
        >
        <NFormItem
          label="密码"
          path="password"
          :label-props="{ for: 'auth-password' }"
          :validation-status="errors.password ? 'error' : undefined"
          required
        >
          <NInput
            v-model:value="state.password"
            type="password"
            show-password-on="click"
            :placeholder="setup ? '6–32 个字符' : '输入密码'"
            :input-props="{
              id: 'auth-password',
              'aria-invalid': !!errors.password,
              'aria-required': true,
              'aria-describedby': errors.password ? 'auth-password-error' : undefined,
              autocomplete: setup ? 'new-password' : 'current-password',
            }"
            :maxlength="32"
          >
            <template #prefix><LockKeyhole :size="18" /></template>
          </NInput>
          <template v-if="errors.password" #feedback
            ><span id="auth-password-error">{{ errors.password }}</span></template
          ></NFormItem
        >
        <NFormItem
          v-if="setup"
          label="确认密码"
          path="password_confirm"
          :label-props="{ for: 'auth-password-confirm' }"
          :validation-status="errors.password_confirm ? 'error' : undefined"
          required
        >
          <NInput
            v-model:value="state.password_confirm"
            type="password"
            show-password-on="click"
            placeholder="再次输入密码"
            :input-props="{
              id: 'auth-password-confirm',
              'aria-invalid': !!errors.password_confirm,
              'aria-required': true,
              'aria-describedby': errors.password_confirm
                ? 'auth-password-confirm-error'
                : undefined,
              autocomplete: 'new-password',
            }"
            :maxlength="32"
          >
            <template #prefix><LockKeyhole :size="18" /></template>
          </NInput>
          <template v-if="errors.password_confirm" #feedback
            ><span id="auth-password-confirm-error">{{ errors.password_confirm }}</span></template
          ></NFormItem
        >
        <NFormItem
          v-if="setup"
          label="昵称"
          path="nickname"
          :label-props="{ for: 'auth-nickname' }"
          :validation-status="errors.nickname ? 'error' : undefined"
        >
          <NInput
            v-model:value="state.nickname"
            placeholder="留空使用用户名"
            :input-props="{
              id: 'auth-nickname',
              'aria-invalid': !!errors.nickname,
              'aria-describedby': errors.nickname ? 'auth-nickname-error' : undefined,
              autocomplete: 'nickname',
            }"
            :maxlength="32"
          >
            <template #prefix><UserRound :size="18" /></template>
          </NInput>
          <template v-if="errors.nickname" #feedback
            ><span id="auth-nickname-error">{{ errors.nickname }}</span></template
          ></NFormItem
        >
        <NFormItem
          v-if="setup"
          label="头像地址"
          path="avatar_url"
          :label-props="{ for: 'auth-avatar-url' }"
          :validation-status="errors.avatar_url ? 'error' : undefined"
        >
          <NInput
            v-model:value="state.avatar_url"
            placeholder="https://example.com/avatar.png"
            :input-props="{
              id: 'auth-avatar-url',
              'aria-invalid': !!errors.avatar_url,
              'aria-describedby': errors.avatar_url ? 'auth-avatar-url-error' : undefined,
              inputmode: 'url',
              autocomplete: 'off',
              autocapitalize: 'none',
              spellcheck: false,
            }"
            :maxlength="2048"
          >
            <template #prefix><Link :size="18" /></template>
          </NInput>
          <template v-if="errors.avatar_url" #feedback
            ><span id="auth-avatar-url-error">{{ errors.avatar_url }}</span></template
          ></NFormItem
        >
        <NButton
          attr-type="submit"
          type="primary"
          block
          :loading="pending"
          :disabled="pending"
          class="mt-2"
        >
          {{ setup ? '创建管理员账号' : '登录' }}
          <template v-if="setup" #icon><ArrowRight :size="18" /></template>
        </NButton>
      </NForm>
    </NCard>
  </main>
</template>
