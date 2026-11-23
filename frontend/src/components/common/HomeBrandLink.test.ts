// @vitest-environment happy-dom

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { HomeButtonTheme } from '@/types'
import HomeBrandLink from './HomeBrandLink.vue'

const themes: HomeButtonTheme[] = ['classic', 'glass', 'terminal', 'aurora', 'image']

describe('HomeBrandLink', () => {
  it.each(themes)('渲染 %s 主题并保留返回主页行为', (theme) => {
    const wrapper = mount(HomeBrandLink, {
      props: { title: 'Personal Space', homeButton: { theme, image_url: theme === 'image' ? '/storage/images/brand.png' : '' } },
    })
    const link = wrapper.get('a')
    expect(link.classes()).toContain(`home-brand--${theme}`)
    expect(link.attributes('href')).toBe('/')
    expect(link.attributes('aria-label')).toBe('返回主页')
    expect(link.text()).toContain('Personal Space')
    if (theme === 'image') expect(wrapper.get('img').attributes('src')).toBe('/storage/images/brand.png')
    wrapper.unmount()
  })

  it('图片加载失败时回退经典菱形，并在图片地址变化后重试', async () => {
    const wrapper = mount(HomeBrandLink, {
      props: { title: 'Personal Space', homeButton: { theme: 'image', image_url: '/broken.png' } },
    })
    await wrapper.get('img').trigger('error')
    expect(wrapper.get('a').classes()).toContain('home-brand--classic')
    expect(wrapper.find('img').exists()).toBe(false)

    await wrapper.setProps({ homeButton: { theme: 'image', image_url: '/fixed.png' } })
    expect(wrapper.get('a').classes()).toContain('home-brand--image')
    expect(wrapper.get('img').attributes('src')).toBe('/fixed.png')
    wrapper.unmount()
  })
})
