<template>
  <div class="model-list-wrap">
    <div class="list-search">
      <Icon name="search" :size="14" class="search-icon" />
      <input v-model="searchQuery" class="search-input" placeholder="搜索模型..." @keydown.enter.prevent="selectFirstFiltered" />
      <span v-if="models.length" class="list-count">{{ filteredModels.length }}/{{ models.length }}</span>
    </div>

    <div ref="listRef" class="list-scroll">
      <div v-if="loading" class="list-state">
        <Icon name="loader" :size="14" spinning />
        <span>加载中...</span>
      </div>
      <div v-else-if="models.length === 0" class="list-state">
        <span>{{ placeholder }}</span>
      </div>
      <div v-else-if="filteredModels.length === 0" class="list-state">
        <span>未找到匹配的模型</span>
      </div>
      <template v-else>
        <div v-for="model in filteredModels" :key="model" class="model-row"
          :class="{ selected: modelValue === model }" :data-model="model" @click="emit('update:modelValue', model)">
          <div class="provider-logo">
            <div v-if="isInlineSvg(model)" v-html="getProviderLogo(model)"></div>
            <img v-else :src="getProviderLogo(model)" alt="logo" />
          </div>
          <div class="model-info">
            <span class="model-name">{{ model }}</span>
            <span class="provider-name">{{ getProviderName(model) }}</span>
          </div>
          <div class="capability-tags">
            <span v-if="getModelCapabilities(model).image" class="cap-tag" title="支持图片"><Icon name="image" :size="11" /></span>
            <span v-if="getModelCapabilities(model).file" class="cap-tag" title="支持文件"><Icon name="file" :size="11" /></span>
            <span v-if="getModelCapabilities(model).audio" class="cap-tag" title="支持音频"><Icon name="mic" :size="11" /></span>
            <span v-if="getModelCapabilities(model).video" class="cap-tag" title="支持视频">
              <svg class="cap-icon" viewBox="0 0 16 16" fill="none">
                <rect x="1" y="4" width="10" height="8" rx="1" stroke="currentColor" stroke-width="1.2"/>
                <path d="M11 7l4-2v6l-4-2v-2z" stroke="currentColor" stroke-width="1.2" stroke-linejoin="round"/>
              </svg>
            </span>
            <span v-if="getModelCapabilities(model).contextLength" class="cap-tag cap-context" title="上下文长度">
              {{ formatContextLength(getModelCapabilities(model).contextLength) }}
            </span>
          </div>
          <span class="check-slot"><Icon v-if="modelValue === model" name="check" :size="14" /></span>
        </div>
      </template>
    </div>

    <div v-if="showEffort" class="effort-section">
      <div class="effort-head">
        <span class="effort-label"><Icon name="brain" :size="12" />思考强度</span>
        <span class="effort-hint">仅截图搜题生效 · 默认跟随模型</span>
      </div>
      <div class="effort-options">
        <button v-for="opt in effortOptions" :key="opt.value" type="button" class="effort-chip"
          :class="{ active: effort === opt.value }" @click="emit('update:effort', opt.value)">
          {{ opt.label }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { getProviderLogo, getProviderName, getModelCapabilities, formatContextLength } from '../utils/modelCapabilities'
import Icon from './Icon.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  models: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  placeholder: { type: String, default: '暂未获取到模型' },
  // 思考强度（搜题模式）：'' = 跟随模型/网关默认
  effort: { type: String, default: '' },
  showEffort: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue', 'update:effort'])

// 思考强度档位（与后端 config.ReasoningEffort 取值一致）
const effortOptions = [
  { value: '', label: '默认' },
  { value: 'none', label: '关闭' },
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' },
  { value: 'xhigh', label: '超高' },
  { value: 'max', label: '最大' },
  { value: 'ultra', label: 'Ultra' },
]

const searchQuery = ref('')
const listRef = ref(null)

// 搜索过滤：匹配模型 ID 或提供商名（不区分大小写）
const filteredModels = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return props.models
  return props.models.filter(m =>
    m.toLowerCase().includes(q) ||
    getProviderName(m).toLowerCase().includes(q)
  )
})

function isInlineSvg(model) {
  const logo = getProviderLogo(model)
  return !!logo && logo.trim().startsWith('<svg')
}

function selectFirstFiltered() {
  if (filteredModels.value.length > 0) emit('update:modelValue', filteredModels.value[0])
}

// 把当前选中项滚到列表中间（只滚列表自身，不带动外层弹窗）
function scrollToSelected() {
  const box = listRef.value
  if (!box || box.clientHeight === 0) return
  const row = box.querySelector('.model-row.selected')
  if (!row) return
  box.scrollTop = row.offsetTop - (box.clientHeight - row.offsetHeight) / 2
}

// 设置页的标签页用 v-show 切换：不可见时无法测量，等它从隐藏变为可见时再定位
let resizeObserver = null
let wasVisible = false
onMounted(() => {
  resizeObserver = new ResizeObserver(() => {
    const visible = !!listRef.value && listRef.value.clientHeight > 0
    if (visible && !wasVisible) scrollToSelected()
    wasVisible = visible
  })
  if (listRef.value) resizeObserver.observe(listRef.value)
})
onBeforeUnmount(() => resizeObserver?.disconnect())

// 模型列表（重新）加载完成后定位一次
watch(() => props.models, () => nextTick(scrollToSelected))
</script>

<style scoped>
.model-list-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
}

.list-search {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: 0 var(--sp-3);
  height: 34px;
  background: var(--surface-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  transition: border-color var(--duration-fast) ease;
}

.list-search:focus-within { border-color: var(--accent); box-shadow: var(--shadow-glow); }

.search-icon { color: var(--text-tertiary); flex-shrink: 0; }

.search-input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--text-primary);
  font-size: var(--text-sm);
}
.search-input::placeholder { color: var(--text-tertiary); }

.list-count {
  flex-shrink: 0;
  font-size: 10px;
  font-family: var(--font-mono);
  color: var(--text-tertiary);
}

.list-scroll {
  position: relative;
  flex: 1;
  min-height: 72px;
  overflow-y: auto;
  background: var(--surface-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.list-scroll::-webkit-scrollbar { width: 6px; }
.list-scroll::-webkit-scrollbar-track { background: transparent; }
.list-scroll::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); border-radius: 3px; }

.list-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  height: 100%;
  min-height: 72px;
  color: var(--text-tertiary);
  font-size: var(--text-sm);
}

.model-row {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-2) var(--sp-3);
  cursor: pointer;
  border-bottom: 1px solid var(--border-subtle);
  transition: background var(--duration-fast) ease;
}
.model-row:last-child { border-bottom: none; }
.model-row:hover { background: var(--surface-card-hover); }
.model-row.selected { background: var(--accent-muted); }

.provider-logo {
  width: 24px; height: 24px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  color: var(--text-primary);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.provider-logo :deep(svg) { width: 100%; height: 100%; display: block; max-width: 24px; max-height: 24px; }
.provider-logo img { width: 100%; height: 100%; object-fit: contain; }

.model-info {
  flex: 1; min-width: 0;
  display: flex; flex-direction: column; gap: 1px;
}
.model-name {
  font-size: var(--text-sm); font-weight: 500;
  color: var(--text-primary);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.provider-name { font-size: 10px; color: var(--text-tertiary); }

.capability-tags { display: flex; gap: 4px; flex-shrink: 0; align-items: center; }
.cap-tag {
  display: flex; align-items: center; justify-content: center;
  width: 18px; height: 18px; border-radius: 4px;
  background: var(--accent-muted);
  color: var(--accent);
}
.cap-tag.cap-context { width: auto; padding: 0 5px; font-size: 9px; font-weight: 600; font-family: var(--font-mono); }
.cap-icon { width: 11px; height: 11px; }

.check-slot {
  width: 16px; flex-shrink: 0;
  display: flex; align-items: center; justify-content: center;
  color: var(--accent);
}

/* 思考强度：列表下方 */
.effort-section {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  background: var(--surface-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}

.effort-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-2);
}

.effort-label {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
}

.effort-hint { font-size: 10px; color: var(--text-tertiary); }

.effort-options { display: flex; gap: 4px; }

.effort-chip {
  flex: 1 1 0;
  min-width: 0;
  padding: 4px 0;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-default);
  background: transparent;
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.4;
  white-space: nowrap;
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.effort-chip:hover { border-color: var(--accent); color: var(--accent); }
.effort-chip.active {
  background: var(--accent-muted);
  border-color: var(--accent);
  color: var(--accent);
  font-weight: 600;
}
</style>
