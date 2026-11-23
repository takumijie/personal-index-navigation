<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Home } from 'lucide-vue-next'
import type { HomeButtonSettings } from '@/types'

const props = defineProps<{ title: string; homeButton: HomeButtonSettings }>()
const imageFailed = ref(false)

watch(() => [props.homeButton.theme, props.homeButton.image_url], () => { imageFailed.value = false })

const showImage = computed(() => props.homeButton.theme === 'image' && !!props.homeButton.image_url && !imageFailed.value)
const effectiveTheme = computed(() => props.homeButton.theme === 'image' && !showImage.value ? 'classic' : props.homeButton.theme)
</script>

<template>
  <a href="/" class="home-brand" :class="`home-brand--${effectiveTheme}`" aria-label="返回主页">
    <span class="home-brand__icon" aria-hidden="true">
      <img v-if="showImage" :src="homeButton.image_url" alt="" class="home-brand__image" @error="imageFailed = true" />
      <Home v-else-if="effectiveTheme === 'glass'" :size="15" :stroke-width="1.6" />
      <span v-else-if="effectiveTheme === 'terminal'" class="home-brand__prompt">~/</span>
      <span v-else class="home-brand__diamond" />
    </span>
    <span class="home-brand__copy">
      <span class="home-brand__title">{{ title }}</span>
      <span class="home-brand__subtitle">A QUIET DIGITAL CORNER</span>
    </span>
  </a>
</template>

<style scoped>
.home-brand {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
  color: inherit;
  text-decoration: none;
  outline: none;
}

.home-brand__icon {
  display: flex;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  align-items: center;
  justify-content: center;
  border: 1px solid rgb(255 255 255 / 10%);
  border-radius: 12px;
  background: rgb(255 255 255 / 7%);
  box-shadow: 0 8px 24px rgb(0 0 0 / 18%);
  backdrop-filter: blur(16px);
  transition: transform 180ms ease, border-color 180ms ease, box-shadow 180ms ease;
}

.home-brand__copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.home-brand__title {
  max-width: clamp(110px, 38vw, 220px);
  overflow: hidden;
  color: rgb(255 255 255 / 75%);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.025em;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home-brand__subtitle {
  color: rgb(255 255 255 / 30%);
  font-size: 8px;
  letter-spacing: 0.22em;
  line-height: 1.5;
  white-space: nowrap;
}

.home-brand__diamond {
  width: 10px;
  height: 10px;
  transform: rotate(45deg);
  border: 1px solid rgb(221 214 254 / 70%);
  border-radius: 3px;
  background: rgb(196 181 253 / 20%);
  transition: transform 220ms ease;
}

.home-brand--classic:hover .home-brand__diamond,
.home-brand--aurora:hover .home-brand__diamond {
  transform: rotate(135deg);
}

.home-brand:focus-visible .home-brand__icon {
  outline: 2px solid rgb(196 181 253 / 80%);
  outline-offset: 3px;
}

.home-brand--glass .home-brand__icon {
  border-color: rgb(255 255 255 / 20%);
  background: linear-gradient(145deg, rgb(255 255 255 / 18%), rgb(255 255 255 / 5%));
  color: rgb(255 255 255 / 80%);
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 16%), 0 10px 28px rgb(15 23 42 / 20%);
}

.home-brand--glass:hover .home-brand__icon {
  transform: translateY(-2px);
  border-color: rgb(255 255 255 / 35%);
}

.home-brand--terminal .home-brand__icon {
  border-color: rgb(74 222 128 / 28%);
  border-radius: 8px;
  background: rgb(3 18 13 / 72%);
  box-shadow: inset 0 0 16px rgb(34 197 94 / 5%), 0 8px 24px rgb(0 0 0 / 18%);
}

.home-brand__prompt {
  color: rgb(134 239 172 / 85%);
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: -0.08em;
}

.home-brand--terminal .home-brand__title {
  color: rgb(209 250 229 / 75%);
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
}

.home-brand--aurora .home-brand__icon {
  border-color: transparent;
  background: linear-gradient(#19162f, #19162f) padding-box,
    linear-gradient(135deg, rgb(167 139 250 / 85%), rgb(96 165 250 / 65%), rgb(45 212 191 / 55%)) border-box;
  box-shadow: 0 0 22px rgb(139 92 246 / 20%), inset 0 0 14px rgb(96 165 250 / 7%);
}

.home-brand--aurora .home-brand__diamond {
  border-color: rgb(191 219 254 / 85%);
  background: linear-gradient(135deg, rgb(196 181 253 / 45%), rgb(125 211 252 / 30%));
}

.home-brand--image .home-brand__icon {
  background: rgb(15 15 25 / 55%);
}

.home-brand__image {
  width: 100%;
  height: 100%;
  border-radius: inherit;
  object-fit: cover;
}

@media (max-width: 380px) {
  .home-brand__title { max-width: 104px; }
  .home-brand__subtitle { font-size: 7px; letter-spacing: 0.16em; }
}
</style>
