// @vitest-environment happy-dom

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { adminApi, publicApi } from '@/api'
import { DEFAULT_SETTINGS } from '@/data/defaults'
import SettingsPanel from './SettingsPanel.vue'

afterEach(() => { vi.restoreAllMocks() })

describe('站点图标与主页按钮设置', () => {
  it('实时预览主题并保存 favicon 和图片徽标配置', async () => {
    vi.spyOn(publicApi, 'settings').mockResolvedValue(structuredClone(DEFAULT_SETTINGS))
    const update = vi.spyOn(adminApi, 'updateSettings').mockImplementation(async (settings) => ({ ...settings, version: settings.version + 1 }))
    const wrapper = mount(SettingsPanel)
    await flushPromises()

    const select = wrapper.get('[data-testid="home-button-theme"]')
    expect(select.findAll('option')).toHaveLength(5)
    for (const theme of ['classic', 'glass', 'terminal', 'aurora']) {
      await select.setValue(theme)
      expect(wrapper.get('[data-testid="home-button-preview"]').find(`.home-brand--${theme}`).exists()).toBe(true)
    }

    await select.setValue('image')
    await wrapper.get('[data-testid="home-button-image"] input[type="text"]').setValue('/storage/images/brand.png')
    await wrapper.get('[data-testid="favicon-field"] input[type="text"]').setValue('/storage/images/favicon.ico')
    expect(wrapper.get('[data-testid="home-button-preview"]').find('.home-brand--image').exists()).toBe(true)

    await wrapper.findAll('button').find((button) => button.text() === '保存设置')?.trigger('click')
    await flushPromises()
    expect(update).toHaveBeenCalledWith(expect.objectContaining({
      favicon_url: '/storage/images/favicon.ico',
      home_button: { theme: 'image', image_url: '/storage/images/brand.png' },
    }))
    wrapper.unmount()
  })

  it('图片主题缺少图片时阻止保存并显示错误', async () => {
    vi.spyOn(publicApi, 'settings').mockResolvedValue(structuredClone(DEFAULT_SETTINGS))
    const update = vi.spyOn(adminApi, 'updateSettings')
    const wrapper = mount(SettingsPanel)
    await flushPromises()

    await wrapper.get('[data-testid="home-button-theme"]').setValue('image')
    await wrapper.findAll('button').find((button) => button.text() === '保存设置')?.trigger('click')
    expect(update).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="home-button-image"]').text()).toContain('图片徽标主题需要配置图片')
    wrapper.unmount()
  })
})
