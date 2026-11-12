<template>
  <button class="theme-toggle" @click="cycleThemeMode" :title="title">
    <Icon :name="icon" :size="15" />
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { themeMode, cycleThemeMode } from '../services/theme'
import Icon from './Icon.vue'

// 图标表示当前所处的档位；提示文字说明点击后切到哪一档
const MODE_META = {
  light: { icon: 'sun', label: '浅色', next: '深色' },
  dark: { icon: 'moon', label: '深色', next: '跟随系统' },
  system: { icon: 'monitor', label: '跟随系统', next: '浅色' },
}

const meta = computed(() => MODE_META[themeMode.value])
const icon = computed(() => meta.value.icon)
const title = computed(() => `主题：${meta.value.label}（点击切换到${meta.value.next}）`)
</script>

<style scoped>
.theme-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  border: none;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.theme-toggle:hover {
  background: var(--surface-card-hover);
  color: var(--text-primary);
}
</style>
