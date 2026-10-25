<script setup lang="ts">
import { ref, computed, reactive, watch } from 'vue'
import { NAlert, NButton, NForm, NFormItem, NInput, NModal, useMessage } from 'naive-ui'
import DiscardChangesDialog from '@/components/common/DiscardChangesDialog.vue'
import { useFormErrors } from '@/composables/useFormErrors'
import UserAvatar from '@/components/common/UserAvatar.vue'
import { useApi } from '@/composables/useApi'
import { useUserSession } from '@/composables/useUserSession'
import type { User } from '@/types/auth'
import { profileSchema } from '@/schemas/profile'

const $api = useApi()
const session = useUserSession()
const message = useMessage()
const open = defineModel<boolean>('open', { default: false })
const { errors, clear, validate, setErrors } = useFormErrors('profile-form')
const draft = reactive({ nickname: '', avatar_url: '' })
const saved = ref({ nickname: '', avatar_url: '' })
const previewUrl = ref<string | null>(null)
const pending = ref(false)
const discardOpen = ref(false)
const errorMessage = ref('')
const dirty = computed(
  () => draft.nickname !== saved.value.nickname || draft.avatar_url !== saved.value.avatar_url,
)

watch(open, (value) => {
  if (!value) return
  saved.value = {
    nickname: session.user.value?.nickname ?? '',
    avatar_url: session.user.value?.avatar_url ?? '',
  }
  Object.assign(draft, saved.value)
  previewUrl.value = saved.value.avatar_url || null
  errorMessage.value = ''
  discardOpen.value = false
  clear()
})

function updatePreview() {
  const parsed = profileSchema.shape.avatar_url.safeParse(draft.avatar_url)
  previewUrl.value = parsed.success ? parsed.data : null
}

function requestClose(value = false) {
  if (value || pending.value) return
  if (dirty.value) discardOpen.value = true
  else open.value = false
}

function discard() {
  if (pending.value) return
  discardOpen.value = false
  open.value = false
}

async function save() {
  if (pending.value || discardOpen.value || !dirty.value) return
  const result = validate(profileSchema, draft)
  if (!result.success) return
  pending.value = true
  errorMessage.value = ''
  try {
    const response = await $api<{ data: User }>('/api/auth/profile', {
      method: 'PATCH',
      body: result.data,
      retry: false,
    })
    if (session.session.value)
      session.session.value = { ...session.session.value, user: response.data }
    message.success('资料已保存')
    open.value = false
  } catch (error) {
    const failure = error as {
      data?: { message?: string; data?: { fieldErrors?: Record<string, string> } }
    }
    errorMessage.value = failure.data?.message || '资料保存失败，输入已保留'
    const fields = failure.data?.data?.fieldErrors
    if (fields) setErrors(fields)
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <NModal
    :show="open"
    preset="card"
    content-scrollable
    class="form-modal"
    title="编辑资料"
    aria-label="编辑资料"
    :style="{ width: 'min(420px, calc(100vw - 32px))' }"
    :mask-closable="!pending"
    :close-on-esc="!pending && !discardOpen"
    :closable="!pending"
    @update:show="requestClose"
  >
    <NForm
      id="profile-form"
      :model="draft"
      :disabled="pending || discardOpen"
      @submit.prevent="save"
    >
      <NAlert v-if="errorMessage" type="error" role="alert" class="mb-4">{{ errorMessage }}</NAlert>
      <div class="mb-5 flex justify-center"><UserAvatar :src="previewUrl" :size="64" /></div>
      <NFormItem
        label="昵称"
        path="nickname"
        :label-props="{ for: 'profile-nickname' }"
        :validation-status="errors.nickname ? 'error' : undefined"
      >
        <NInput
          v-model:value="draft.nickname"
          placeholder="留空使用用户名"
          :input-props="{
            id: 'profile-nickname',
            'aria-invalid': !!errors.nickname,
            'aria-describedby': errors.nickname ? 'profile-nickname-error' : undefined,
            autocomplete: 'nickname',
          }"
          :maxlength="32"
        />
        <template v-if="errors.nickname" #feedback
          ><span id="profile-nickname-error">{{ errors.nickname }}</span></template
        ></NFormItem
      >
      <NFormItem
        label="头像地址"
        path="avatar_url"
        :label-props="{ for: 'profile-avatar-url' }"
        :validation-status="errors.avatar_url ? 'error' : undefined"
      >
        <NInput
          v-model:value="draft.avatar_url"
          placeholder="https://example.com/avatar.png"
          :input-props="{
            id: 'profile-avatar-url',
            'aria-invalid': !!errors.avatar_url,
            'aria-describedby': errors.avatar_url ? 'profile-avatar-url-error' : undefined,
            inputmode: 'url',
            autocomplete: 'off',
            autocapitalize: 'none',
            spellcheck: false,
          }"
          :maxlength="2048"
          @blur="updatePreview"
        />
        <template v-if="errors.avatar_url" #feedback
          ><span id="profile-avatar-url-error">{{ errors.avatar_url }}</span></template
        ></NFormItem
      >
    </NForm>
    <template #footer>
      <div class="modal-footer">
        <NButton :disabled="pending || discardOpen" @click="requestClose()">取消</NButton>
        <NButton
          attr-type="submit"
          form="profile-form"
          type="primary"
          :loading="pending"
          :disabled="pending || !dirty || discardOpen"
          >保存</NButton
        >
      </div>
    </template>
  </NModal>
  <DiscardChangesDialog v-model:open="discardOpen" @discard="discard" />
</template>
