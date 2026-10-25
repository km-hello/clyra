<script setup lang="ts">
import { RouterView } from 'vue-router'
import {
  NConfigProvider,
  NGlobalStyle,
  NMessageProvider,
  NNotificationProvider,
  darkTheme,
  zhCN,
  dateZhCN,
} from 'naive-ui'
import { isDark } from '@/composables/useThemeMode'
import { themeOverrides } from '@/theme'
</script>

<template>
  <NConfigProvider
    :theme="isDark ? darkTheme : null"
    :theme-overrides="themeOverrides"
    :locale="zhCN"
    :date-locale="dateZhCN"
  >
    <NGlobalStyle />
    <NMessageProvider>
      <NNotificationProvider>
        <RouterView />
      </NNotificationProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>
