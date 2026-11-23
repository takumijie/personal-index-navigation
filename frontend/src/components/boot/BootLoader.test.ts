// @vitest-environment happy-dom

import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { DEFAULT_SETTINGS } from '@/data/defaults'
import BootLoader from './BootLoader.vue'

vi.mock('@/utils/system', () => ({ detectSystem: vi.fn().mockResolvedValue(null) }))

const wrappers: VueWrapper[] = []
function mountLoader(theme: 'dark' | 'beige' = 'dark') {
  const loader = { ...structuredClone(DEFAULT_SETTINGS.loader), theme }
  const wrapper = mount(BootLoader, { props: { ready: false, loader, visitor: null } })
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({ matches: false }))
})

afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('BootLoader initial theme', () => {
  it('renders beige on the first mount', () => {
    const wrapper = mountLoader('beige')
    expect(wrapper.get('.boot-root').classes()).toContain('theme-beige')
  })
})
