<script setup lang="ts">
import { computed, onMounted, ref, type Component } from 'vue'
import {
  ArrowUpRight, BookOpenCheck, Boxes, ChevronRight, Compass, LoaderCircle,
  LogOut, Music2, Settings2, UserRound,
} from 'lucide-vue-next'
import { adminApi } from '@/api'
import { errorMessage } from '@/api/client'
import LoginForm from '@/components/admin/LoginForm.vue'
import ProfilePanel from '@/components/admin/ProfilePanel.vue'
import NavigationPanel from '@/components/admin/NavigationPanel.vue'
import DevicesPanel from '@/components/admin/DevicesPanel.vue'
import GuestbookPanel from '@/components/admin/GuestbookPanel.vue'
import SettingsPanel from '@/components/admin/SettingsPanel.vue'
import MusicPanel from '@/components/admin/MusicPanel.vue'
import AlertMessage from '@/components/common/AlertMessage.vue'

interface Tab { key: string; label: string; icon: Component; component: Component }
const tabs: Tab[] = [
  { key: 'profile', label: '个人资料', icon: UserRound, component: ProfilePanel },
  { key: 'navigation', label: '导航管理', icon: Compass, component: NavigationPanel },
  { key: 'devices', label: '设备管理', icon: Boxes, component: DevicesPanel },
  { key: 'guestbook', label: '留言审核', icon: BookOpenCheck, component: GuestbookPanel },
  { key: 'music', label: '音乐管理', icon: Music2, component: MusicPanel },
  { key: 'settings', label: '站点设置', icon: Settings2, component: SettingsPanel },
]
const auth = ref<'checking' | 'login' | 'authenticated'>('checking')
const username = ref('')
const activeTab = ref('profile')
const logoutLoading = ref(false)
const error = ref('')
const currentTab = computed(() => tabs.find((tab) => tab.key === activeTab.value) || tabs[0])

onMounted(async () => {
  try {
    const session = await adminApi.session()
    if (session.authenticated === false) {
      auth.value = 'login'
      return
    }
    username.value = session.username
    auth.value = 'authenticated'
  } catch {
    auth.value = 'login'
  }
})

async function logout() {
  logoutLoading.value = true
  error.value = ''
  try {
    await adminApi.logout()
    auth.value = 'login'
    username.value = ''
  } catch (reason) {
    error.value = errorMessage(reason)
  } finally {
    logoutLoading.value = false
  }
}
</script>

<template>
  <div v-if="auth === 'checking'" class="flex min-h-screen items-center justify-center bg-[#0c0e1b] text-white/45">
    <LoaderCircle :size="22" class="mr-3 animate-spin" /><span class="text-sm">正在验证管理会话…</span>
  </div>
  <LoginForm v-else-if="auth === 'login'" @success="username = $event; auth = 'authenticated'" />

  <div v-else class="min-h-screen bg-[#0d0f1d] text-white">
    <div class="pointer-events-none fixed inset-0 bg-[radial-gradient(circle_at_80%_0%,rgba(108,77,180,0.13),transparent_38%),radial-gradient(circle_at_0%_100%,rgba(57,78,148,0.09),transparent_35%)]" />
    <header class="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-white/[0.07] bg-[#0d0f1d]/85 px-4 backdrop-blur-xl min-[640px]:px-7">
      <div class="flex items-center gap-3">
        <span class="flex size-8 items-center justify-center rounded-xl bg-violet-400/10 text-violet-200"><Settings2 :size="16" /></span>
        <div><h1 class="text-sm font-medium text-white/85">站点管理</h1><p class="text-[9px] tracking-wider text-white/28">PERSONAL SPACE CONSOLE</p></div>
      </div>
      <div class="flex items-center gap-1.5">
        <RouterLink to="/" class="secondary-button px-3 py-2 text-xs" title="查看主页"><ArrowUpRight :size="14" /><span class="hidden min-[640px]:inline">查看主页</span></RouterLink>
        <span class="mx-2 hidden text-xs text-white/35 min-[640px]:inline">{{ username }}</span>
        <button class="icon-button" type="button" title="退出登录" aria-label="退出登录" :disabled="logoutLoading" @click="logout"><LoaderCircle v-if="logoutLoading" :size="15" class="animate-spin" /><LogOut v-else :size="15" /></button>
      </div>
    </header>

    <div class="relative mx-auto grid w-full max-w-[1440px] gap-5 p-4 min-[640px]:p-6 lg:grid-cols-[220px_minmax(0,1fr)] lg:p-7">
      <aside class="glass-card h-max rounded-2xl p-2 lg:sticky lg:top-[92px]">
        <nav class="flex gap-1 overflow-x-auto lg:flex-col" aria-label="管理功能">
          <button v-for="tab in tabs" :key="tab.key" type="button" class="group flex shrink-0 items-center gap-3 rounded-xl px-3.5 py-3 text-left text-xs transition lg:w-full" :class="activeTab === tab.key ? 'bg-violet-300/[0.1] text-violet-100' : 'text-white/42 hover:bg-white/[0.045] hover:text-white/75'" @click="activeTab = tab.key">
            <component :is="tab.icon" :size="16" /><span>{{ tab.label }}</span><ChevronRight v-if="activeTab === tab.key" :size="12" class="ml-auto hidden lg:block" />
          </button>
        </nav>
      </aside>
      <main class="min-w-0">
        <AlertMessage :message="error" type="error" />
        <KeepAlive><component :is="currentTab.component" :key="currentTab.key" /></KeepAlive>
      </main>
    </div>
  </div>
</template>
