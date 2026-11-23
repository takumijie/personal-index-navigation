// @vitest-environment happy-dom

import { mount } from '@vue/test-utils'
import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { DEFAULT_SETTINGS } from '@/data/defaults'
import type { Settings } from '@/types'
import TerminalCard from './TerminalCard.vue'

vi.mock('@/composables/useTerminal', () => ({
  useTerminal: () => ({
    phase: ref('interactive'),
    lines: ref([]),
    input: ref(''),
    submitting: ref(false),
    prompt: ref('guest>'),
    output: ref(null),
    inputElement: ref(null),
    start: vi.fn(),
    submit: vi.fn(),
    navigateHistory: vi.fn(),
  }),
}))

function mountCard(settings: Settings) {
  return mount(TerminalCard, {
    props: {
      active: false,
      entries: [],
      settings,
      visitor: null,
      terminalError: '',
      loadTerminalData: vi.fn(),
    },
  })
}

describe('TerminalCard guestbook input limit', () => {
  it('uses a configured limit below 1000 instead of overriding it', () => {
    const wrapper = mountCard({ ...structuredClone(DEFAULT_SETTINGS), guestbook_max_length: 120 })

    expect(wrapper.get('input[aria-label="终端命令或留言"]').attributes('maxlength')).toBe('120')
    wrapper.unmount()
  })

  it('uses 1000 only when the setting is absent', async () => {
    const wrapper = mountCard(structuredClone(DEFAULT_SETTINGS))
    const settings = { ...structuredClone(DEFAULT_SETTINGS), guestbook_max_length: undefined } as unknown as Settings

    await wrapper.setProps({ settings })

    expect(wrapper.get('input[aria-label="终端命令或留言"]').attributes('maxlength')).toBe('1000')
    wrapper.unmount()
  })
})
