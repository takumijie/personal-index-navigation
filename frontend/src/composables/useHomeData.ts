import { onMounted, ref } from 'vue'
import { publicApi } from '@/api'
import { errorMessage } from '@/api/client'
import { DEFAULT_SETTINGS } from '@/data/defaults'
import type { Device, GuestbookEntry, NavigationItem, Profile, Settings, Track, Visitor } from '@/types'

export function useHomeData() {
  const profile = ref<Profile | null>(null)
  const navigation = ref<NavigationItem[]>([])
  const devices = ref<Device[]>([])
  const settings = ref<Settings>(structuredClone(DEFAULT_SETTINGS))
  const guestbook = ref<GuestbookEntry[]>([])
  const tracks = ref<Track[]>([])
  const visitor = ref<Visitor | null>(null)
  const loading = ref(true)
  const error = ref('')

  async function load() {
    loading.value = true
    error.value = ''
    try {
      const [profileData, navigationData, deviceData, settingsData, guestbookData, trackData, visitorData] = await Promise.all([
        publicApi.profile(), publicApi.navigation(), publicApi.devices(), publicApi.settings(),
        publicApi.guestbook(), publicApi.playlist(), publicApi.visitor(),
      ])
      profile.value = profileData
      navigation.value = navigationData
      devices.value = deviceData
      settings.value = settingsData
      guestbook.value = guestbookData
      tracks.value = trackData
      visitor.value = visitorData
    } catch (reason) {
      if (import.meta.env.DEV) {
        const mock = await import('@/api/mock')
        profile.value = mock.mockProfile
        navigation.value = mock.mockNavigation
        devices.value = mock.mockDevices
        settings.value = mock.mockSettings
        guestbook.value = mock.mockGuestbook
        tracks.value = mock.mockTracks
        visitor.value = mock.mockVisitor
        error.value = '当前为开发预览数据，后端连接不可用'
      } else {
        error.value = errorMessage(reason)
      }
    } finally {
      loading.value = false
    }
  }

  onMounted(load)
  return { profile, navigation, devices, settings, guestbook, tracks, visitor, loading, error, reload: load }
}
