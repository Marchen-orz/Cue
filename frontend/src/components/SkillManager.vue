<template>
  <div class="skill-manager">
    <div class="sk-intro">
      <p class="hint-text">
        导入 Claude 格式的技能（含 SKILL.md 的文件夹、.zip 或单个 .md）。答题时会先按题目自动挑选相关技能，并注入提示词；面试模式与截图搜题均生效。
      </p>
      <div class="sk-import">
        <button class="sk-btn primary" :disabled="busy" @click="importFile">
          <Icon name="download" :size="13" /><span>导入文件</span>
        </button>
        <button class="sk-btn" :disabled="busy" @click="importDir">
          <Icon name="file" :size="13" /><span>导入文件夹</span>
        </button>
      </div>
    </div>

    <div v-if="!skills.length" class="sk-empty">
      <Icon name="sparkles" :size="26" />
      <p>还没有技能</p>
      <p class="hint-text">没有启用的技能时，答题流程不会产生任何额外模型调用。</p>
    </div>

    <ul v-else class="sk-list">
      <li v-for="sk in skills" :key="sk.id" class="sk-item" :class="{ off: !sk.enabled }">
        <div class="sk-row">
          <button class="sk-main" @click="togglePreview(sk)">
            <Icon :name="openId === sk.id ? 'chevron-down' : 'chevron-right'" :size="13" />
            <span class="sk-name">{{ sk.name }}</span>
            <span class="sk-size">{{ sk.size }} 字</span>
          </button>
          <label class="sk-switch" :title="sk.enabled ? '已启用' : '已停用'">
            <input type="checkbox" :checked="sk.enabled" @change="setEnabled(sk, $event.target.checked)" />
            <span class="sk-slider"></span>
          </label>
          <button class="sk-icon-btn danger" title="删除" @click="remove(sk)">
            <Icon name="trash" :size="14" />
          </button>
        </div>
        <p class="sk-desc">{{ sk.description }}</p>
        <pre v-if="openId === sk.id" class="sk-preview">{{ preview }}</pre>
      </li>
    </ul>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../services/api'
import { useUIStore } from '../stores/ui'
import Icon from './Icon.vue'

const ui = useUIStore()
const skills = ref([])
const busy = ref(false)
const openId = ref('')
const preview = ref('')

async function refresh() {
  try { skills.value = (await api.listSkills()) || [] } catch { skills.value = [] }
}

onMounted(refresh)

async function runImport(fn) {
  busy.value = true
  try {
    const sk = await fn()
    if (sk && sk.id) {
      ui.showToast(`已导入技能：${sk.name}`, 'success')
      await refresh()
    }
  } catch (e) {
    ui.showToast(`导入失败：${e?.message || e}`, 'error', 3500)
  } finally {
    busy.value = false
  }
}

const importFile = () => runImport(api.importSkillFile)
const importDir = () => runImport(api.importSkillDir)

async function setEnabled(sk, on) {
  try {
    await api.setSkillEnabled(sk.id, on)
    sk.enabled = on
  } catch (e) {
    ui.showToast(`操作失败：${e?.message || e}`, 'error')
    await refresh()
  }
}

async function remove(sk) {
  try {
    await api.deleteSkill(sk.id)
    if (openId.value === sk.id) openId.value = ''
    await refresh()
  } catch (e) {
    ui.showToast(`删除失败：${e?.message || e}`, 'error')
  }
}

async function togglePreview(sk) {
  if (openId.value === sk.id) { openId.value = ''; return }
  try {
    const full = await api.getSkill(sk.id)
    preview.value = full.body || ''
    openId.value = sk.id
  } catch (e) {
    ui.showToast(`读取失败：${e?.message || e}`, 'error')
  }
}
</script>

<style scoped>
.skill-manager { display: flex; flex-direction: column; gap: var(--sp-3); }
.hint-text { margin: 0; font-size: var(--text-xs); color: var(--text-muted); line-height: var(--leading-relaxed); }
.sk-import { display: flex; gap: var(--sp-2); margin-top: var(--sp-2); }
.sk-btn {
  display: inline-flex; align-items: center; gap: var(--sp-1-5, 6px);
  padding: var(--sp-1-5, 6px) var(--sp-3);
  background: var(--surface-card); color: var(--text-secondary);
  border: 1px solid var(--border-default); border-radius: var(--radius-sm);
  font-size: var(--text-xs); font-weight: var(--weight-medium); cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.sk-btn:hover:not(:disabled) { background: var(--surface-card-hover); color: var(--accent); }
.sk-btn.primary { background: var(--accent); color: #fff; border-color: transparent; }
.sk-btn.primary:hover:not(:disabled) { background: var(--accent-hover); color: #fff; }
.sk-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.sk-empty {
  display: flex; flex-direction: column; align-items: center; gap: var(--sp-1);
  padding: var(--sp-6) 0; color: var(--text-muted); text-align: center;
}
.sk-empty p { margin: 0; font-size: var(--text-sm); }
.sk-empty .hint-text { margin-top: var(--sp-1); }

.sk-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--sp-2); }
.sk-item {
  padding: var(--sp-2) var(--sp-3);
  background: var(--surface-card); border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}
.sk-item.off .sk-name, .sk-item.off .sk-desc { opacity: 0.5; }
.sk-row { display: flex; align-items: center; gap: var(--sp-2); }
.sk-main {
  flex: 1; min-width: 0; display: flex; align-items: center; gap: var(--sp-1);
  background: transparent; border: none; padding: 0; cursor: pointer;
  color: var(--text-primary); text-align: left;
}
.sk-name { font-size: var(--text-sm); font-weight: var(--weight-semibold); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sk-size { font-size: var(--text-xs); color: var(--text-muted); flex-shrink: 0; }
.sk-desc {
  margin: var(--sp-1) 0 0 calc(13px + var(--sp-1));
  font-size: var(--text-xs); color: var(--text-muted); line-height: var(--leading-relaxed);
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
}
.sk-preview {
  margin: var(--sp-2) 0 0; padding: var(--sp-2) var(--sp-3);
  max-height: 220px; overflow: auto; white-space: pre-wrap; word-break: break-word;
  background: var(--surface-input); border-radius: var(--radius-sm);
  font-family: var(--font-mono); font-size: var(--text-xs); color: var(--text-secondary);
}
.sk-icon-btn {
  width: 26px; height: 26px; display: flex; align-items: center; justify-content: center;
  background: transparent; border: none; border-radius: var(--radius-sm);
  color: var(--text-muted); cursor: pointer;
}
.sk-icon-btn.danger:hover { color: var(--color-error); background: var(--surface-card-hover); }

.sk-switch { position: relative; width: 30px; height: 16px; flex-shrink: 0; cursor: pointer; }
.sk-switch input { opacity: 0; width: 0; height: 0; }
.sk-slider {
  position: absolute; inset: 0; background: var(--border-default); border-radius: 16px;
  transition: background var(--duration-fast) ease;
}
.sk-slider::before {
  content: ''; position: absolute; width: 12px; height: 12px; left: 2px; top: 2px;
  background: #fff; border-radius: 50%; transition: transform var(--duration-fast) ease;
}
.sk-switch input:checked + .sk-slider { background: var(--accent); }
.sk-switch input:checked + .sk-slider::before { transform: translateX(14px); }
</style>
