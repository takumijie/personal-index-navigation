<script setup lang="ts">
import { ref } from 'vue'
import { ArrowLeft, Eye, EyeOff, LockKeyhole, LogIn, UserRound } from 'lucide-vue-next'
import { adminApi } from '@/api'
import { errorMessage } from '@/api/client'
import AlertMessage from '@/components/common/AlertMessage.vue'
import FormField from '@/components/common/FormField.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

const emit = defineEmits<{ success: [username: string] }>()
const username = ref('')
const password = ref('')
const showPassword = ref(false)
const loading = ref(false)
const error = ref('')

async function login() {
  error.value = ''
  if (!username.value.trim() || !password.value) {
    error.value = '请输入用户名和密码'
    return
  }
  loading.value = true
  try {
    const result = await adminApi.login(username.value.trim(), password.value)
    emit('success', result.username || username.value.trim())
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="relative flex min-h-screen items-center justify-center overflow-hidden bg-[#0c0e1b] px-4 py-10">
    <div class="absolute left-1/2 top-[-20%] size-[600px] -translate-x-1/2 rounded-full bg-violet-700/15 blur-[120px]" />
    <div class="absolute bottom-[-30%] right-[-10%] size-[500px] rounded-full bg-indigo-700/10 blur-[110px]" />
    <section class="glass-card animate-enter relative w-full max-w-[420px] rounded-3xl p-7 min-[640px]:p-9">
      <RouterLink to="/" class="mb-8 inline-flex items-center gap-2 text-xs text-white/38 transition hover:text-white/75"><ArrowLeft :size="14" />返回主页</RouterLink>
      <div class="mb-7">
        <div class="mb-5 flex size-11 items-center justify-center rounded-2xl border border-violet-200/15 bg-violet-300/[0.08] text-violet-200"><LockKeyhole :size="20" /></div>
        <h1 class="text-2xl font-semibold tracking-tight text-white/90">欢迎回来</h1>
        <p class="mt-2 text-sm text-white/38">登录后管理你的数字角落</p>
      </div>
      <AlertMessage :message="error" type="error" />
      <form class="space-y-4" @submit.prevent="login">
        <FormField label="用户名">
          <div class="relative"><UserRound :size="15" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-white/25" /><input v-model="username" class="field pl-9" maxlength="100" autocomplete="username" placeholder="管理员用户名" :disabled="loading" /></div>
        </FormField>
        <FormField label="密码">
          <div class="relative">
            <LockKeyhole :size="15" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-white/25" />
            <input v-model="password" class="field px-9" maxlength="200" :type="showPassword ? 'text' : 'password'" autocomplete="current-password" placeholder="管理员密码" :disabled="loading" />
            <button class="absolute right-1 top-1/2 flex size-8 -translate-y-1/2 items-center justify-center text-white/30 hover:text-white/70" type="button" :aria-label="showPassword ? '隐藏密码' : '显示密码'" @click="showPassword = !showPassword"><EyeOff v-if="showPassword" :size="15" /><Eye v-else :size="15" /></button>
          </div>
        </FormField>
        <button class="primary-button mt-2 w-full py-3" type="submit" :disabled="loading">
          <LoadingSpinner v-if="loading" label="正在登录…" /><template v-else><LogIn :size="16" />登录管理后台</template>
        </button>
      </form>
      <p class="mt-7 text-center text-[10px] leading-5 text-white/22">会话由服务器安全验证，请妥善保管登录凭据</p>
    </section>
  </main>
</template>
