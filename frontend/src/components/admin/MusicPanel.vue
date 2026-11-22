<script setup lang="ts">
import { ref } from 'vue'
import { Cloud, HardDrive, ListMusic, Music2 } from 'lucide-vue-next'
import PanelShell from '@/components/common/PanelShell.vue'
import PlaylistManager from '@/components/admin/music/PlaylistManager.vue'
import LocalMusicManager from '@/components/admin/music/LocalMusicManager.vue'
import NeteaseManager from '@/components/admin/music/NeteaseManager.vue'
import S3Manager from '@/components/admin/music/S3Manager.vue'

const tabs = [
  { id: 'playlists', label: '播放列表', icon: ListMusic },
  { id: 'local', label: '本地音乐', icon: HardDrive },
  { id: 'netease', label: '网易云音乐', icon: Music2 },
  { id: 's3', label: '对象存储', icon: Cloud },
] as const

const activeTab = ref<(typeof tabs)[number]['id']>('playlists')
</script>

<template>
  <PanelShell title="音乐管理" description="管理播放列表、本地音乐与歌词，或从网易云和 S3 兼容对象存储导入。">
    <div class="mb-6 flex gap-1 overflow-x-auto rounded-xl border border-white/[0.065] bg-black/10 p-1" role="tablist" aria-label="音乐来源">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-xs transition-colors"
        :class="activeTab === tab.id ? 'bg-violet-300/10 text-violet-100' : 'text-white/35 hover:bg-white/[0.035] hover:text-white/65'"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.id"
        @click="activeTab = tab.id"
      >
        <component :is="tab.icon" :size="14" />{{ tab.label }}
      </button>
    </div>
    <PlaylistManager v-if="activeTab === 'playlists'" />
    <LocalMusicManager v-else-if="activeTab === 'local'" />
    <NeteaseManager v-else-if="activeTab === 'netease'" />
    <S3Manager v-else />
  </PanelShell>
</template>
