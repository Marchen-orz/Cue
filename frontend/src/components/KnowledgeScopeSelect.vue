<template>
  <div class="scope-select" role="radiogroup" aria-label="使用范围">
    <button
      v-for="opt in options"
      :key="opt.value"
      type="button"
      role="radio"
      class="scope-opt"
      :class="[opt.value, { on: modelValue === opt.value }]"
      :aria-checked="modelValue === opt.value"
      :disabled="disabled"
      :title="opt.title"
      @click="select(opt.value)"
    >
      <Icon v-if="opt.icon" :name="opt.icon" :size="11" />
      <span>{{ opt.label }}</span>
    </button>
  </div>
</template>

<script setup>
import Icon from './Icon.vue'

const props = defineProps({
  // 'interview' | 'solve' | 'both'
  modelValue: { type: String, default: 'interview' },
  disabled: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const options = [
  { value: 'solve', label: '搜题', icon: 'camera', title: '仅截图搜题时使用' },
  { value: 'interview', label: '面试', icon: 'mic', title: '仅面试模式回答时使用' },
  { value: 'both', label: '两者', icon: '', title: '截图搜题与面试都使用' },
]

function select(value) {
  if (props.disabled || value === props.modelValue) return
  emit('update:modelValue', value)
}
</script>

<style scoped>
.scope-select {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  background: var(--surface-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}
.scope-opt {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 3px 10px;
  background: transparent; border: none; border-radius: var(--radius-xs);
  color: var(--text-muted);
  font-size: var(--text-xs); font-weight: var(--weight-medium);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.scope-opt:hover:not(:disabled):not(.on) { color: var(--text-secondary); }
.scope-opt.on {
  background: var(--surface-elevated);
  color: var(--accent);
  font-weight: var(--weight-semibold);
  box-shadow: var(--shadow-sm);
}
.scope-opt:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
