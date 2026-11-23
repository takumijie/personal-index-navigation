// @vitest-environment happy-dom

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { adminApi, publicApi } from '@/api'
import ProfilePanel from './ProfilePanel.vue'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('ProfilePanel field limits', () => {
  it('aligns text limits and accepts 20 Unicode tags of 30 characters', async () => {
    vi.spyOn(publicApi, 'profile').mockResolvedValue({
      name: 'Name', username: 'user', description: 'Description', avatar: '',
      cover_mode: 'current', cover_url: '', tags: [],
    })
    const update = vi.spyOn(adminApi, 'updateProfile').mockImplementation(async (profile) => profile)
    const wrapper = mount(ProfilePanel)
    await flushPromises()

    expect(wrapper.get('input[placeholder="你的名字"]').attributes('maxlength')).toBe('100')
    expect(wrapper.get('textarea').attributes('maxlength')).toBe('1000')
    const tags = Array.from({ length: 20 }, () => '🙂'.repeat(30))
    await wrapper.get('input[placeholder="开发，摄影，咖啡"]').setValue(tags.join('，'))
    await wrapper.get('button[type="button"]').trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledTimes(1)
    expect(update.mock.calls[0][0].tags).toEqual(tags)

    await wrapper.get('input[placeholder="开发，摄影，咖啡"]').setValue('🙂'.repeat(31))
    await wrapper.get('button[type="button"]').trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('每个个人标签不能超过 30 字符')
    wrapper.unmount()
  })
})
