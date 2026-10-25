import { createApp } from 'vue'
import { z } from 'zod'
import { zhCN } from 'zod/locales'
import App from '@/App.vue'
import { router } from '@/router/index'
import '@/styles/main.css'

z.config(zhCN())

const app = createApp(App)
app.use(router)
router
  .isReady()
  .then(() => app.mount('#app'))
  .catch(() => {
    const root = document.getElementById('app')!
    const message = document.createElement('p')
    message.textContent = '暂时无法连接服务，请刷新重试'
    const retry = document.createElement('button')
    retry.textContent = '重新加载'
    retry.onclick = () => window.location.reload()
    root.append(message, retry)
  })
