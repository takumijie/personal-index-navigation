<script setup lang="ts">
import { computed, type Component } from 'vue'
import {
  BookOpen, BriefcaseBusiness, Camera, Code2, Cpu, Gamepad2, Github, Globe2,
  HardDrive, Headphones, Keyboard, Laptop, Link2, Mail, MessageCircle, Monitor,
  Mouse, Rss, Server, Smartphone, Tablet, Watch,
} from 'lucide-vue-next'

const props = withDefaults(defineProps<{ name?: string; size?: number; strokeWidth?: number }>(), { name: 'Link2', size: 20, strokeWidth: 1.8 })
const icons: Record<string, Component> = {
  BookOpen, BriefcaseBusiness, Camera, Code2, Cpu, Gamepad2, Github, Globe2,
  HardDrive, Headphones, Keyboard, Laptop, Link2, Mail, MessageCircle, Monitor,
  Mouse, Rss, Server, Smartphone, Tablet, Watch,
}
const icon = computed(() => icons[props.name] || Link2)
</script>

<template>
  <component :is="icon" :size="size" :stroke-width="strokeWidth" aria-hidden="true" />
</template>
