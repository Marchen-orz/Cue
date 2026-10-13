<template>
  <div class="knowledge-manager">
    <template v-if="!editing">
      <div class="kb-intro">
        <p class="hint-text">
          把简历、八股笔记、项目资料放进来，作为背景资料提供给模型。每份文档可单独选择用于「搜题」「面试」或「两者」，默认仅面试；用于搜题的文档会附加在每次截图搜题的请求里，建议只开启真正需要的。文档会加密保存在本地，增删改和范围切换立即生效，无需点击"保存"。
        </p>
        <div class="kb-quota">
          <div class="kb-quota-bar">
            <div class="kb-quota-fill" :class="{ warn: usedPercent >= 90 }" :style="{ width: usedPercent + '%' }"></div>
          </div>
          <span class="kb-quota-text" :class="{ warn: usedPercent >= 90 }">{{ summary.used }} / {{ summary.limit }} 字</span>
        </div>
        <div class="kb-actions">
          <button class="kb-btn primary" :disabled="busy" @click="importResume">
            <Icon name="download" :size="13" /><span>{{ hasResume ? '替换简历' : '导入简历' }}</span>
          </button>
          <button class="kb-btn" :disabled="busy" @click="importFiles">
            <Icon name="file" :size="13" /><span>导入文档</span>
          </button>
          <button class="kb-btn" :disabled="busy" @click="newNote">
            <Icon name="plus" :size="13" /><span>新建笔记</span>
          </button>
        </div>
        <p v-if="busy" class="hint-text kb-busy">
          <Icon name="loader" :size="12" class="kb-spin" />
          正在导入。PDF 会调用你配置的模型转写为 Markdown，请稍候…
        </p>
      </div>

      <div v-if="!summary.docs.length" class="kb-empty">
        <Icon name="book-open" :size="26" />
        <p>知识库是空的</p>
        <p class="hint-text">支持 PDF / DOCX / MD / TXT。简历至多一份，再次导入会替换。</p>
      </div>

      <ul v-else class="kb-list">
        <li v-for="doc in summary.docs" :key="doc.id" class="kb-item">
          <div class="kb-row">
            <button class="kb-main" @click="openDoc(doc)">
              <Icon :name="doc.kind === 'note' ? 'pen-line' : 'file'" :size="14" class="kb-kind-icon" />
              <span class="kb-name">{{ doc.name }}</span>
              <span class="kb-badge" :class="doc.kind">{{ kindLabel(doc.kind) }}</span>
              <span class="kb-size">{{ doc.size }} 字</span>
            </button>
            <button class="kb-icon-btn danger" title="删除" :disabled="busy" @click="remove(doc)">
              <Icon name="trash" :size="14" />
            </button>
          </div>
          <div class="kb-scope-row">
            <span class="kb-scope-label">用于</span>
            <KnowledgeScopeSelect :model-value="doc.scope" @update:model-value="setScope(doc, $event)" />
          </div>
        </li>
      </ul>
    </template>

    <template v-else>
      <div class="kb-editor-head">
        <button class="kb-btn" :disabled="saving" @click="closeEditor">
          <Icon name="chevron-left" :size="13" /><span>返回</span>
        </button>
        <span class="kb-count" :class="{ warn: overLimit }">{{ contentRunes }} / {{ available }} 字</span>
        <button class="kb-btn primary" :disabled="saving || overLimit || !contentRunes" @click="save">
          <span>{{ saving ? '保存中…' : '保存' }}</span>
        </button>
      </div>
      <input
        v-model="editing.name"
        class="kb-input"
        maxlength="60"
        :placeholder="editing.kind === 'resume' ? '简历' : '笔记标题'"
        autocomplete="off"
      />
      <div class="kb-scope-row">
        <span class="kb-scope-label">用于</span>
        <KnowledgeScopeSelect v-model="editing.scope" />
      </div>
      <textarea
        v-model="editing.content"
        class="kb-editor"
        placeholder="在此输入或粘贴 Markdown / 纯文本内容…"
      ></textarea>
      <p v-if="overLimit" class="hint-text kb-over">超出知识库剩余额度，请精简内容或先删除其它文档。</p>
    </template>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { api } from '../services/api'
import { useUIStore } from '../stores/ui'
import { useSettingsStore } from '../stores/settings'
import Icon from './Icon.vue'
import KnowledgeScopeSelect from './KnowledgeScopeSelect.vue'

const ui = useUIStore()
const settingsStore = useSettingsStore()

const summary = reactive({ docs: [], used: 0, limit: 0 })
const busy = computed({
  get: () => settingsStore.isKnowledgeImporting,
  set: (v) => { settingsStore.isKnowledgeImporting = v },
})
const saving = ref(false)
// 正在编辑的文档：id 为空 = 新建笔记；origSize 用于计算"替换自己"后的可用额度
const editing = ref(null)

const hasResume = computed(() => summary.docs.some(d => d.kind === 'resume'))
const usedPercent = computed(() =>
  summary.limit ? Math.min(100, Math.round((summary.used / summary.limit) * 100)) : 0
)

const contentRunes = computed(() => (editing.value ? Array.from(editing.value.content.trim()).length : 0))
const available = computed(() => (editing.value ? summary.limit - summary.used + editing.value.origSize : 0))
const overLimit = computed(() => contentRunes.value > available.value)

const kindLabel = (k) => (k === 'resume' ? '简历' : k === 'note' ? '笔记' : '文档')
const errText = (e) => e?.message || String(e)

async function refresh() {
  try {
    const s = await api.getKnowledge()
    summary.docs = s?.docs || []
    summary.used = s?.used || 0
    summary.limit = s?.limit || 0
  } catch (e) {
    ui.showToast(`读取知识库失败：${errText(e)}`, 'error')
  }
}

onMounted(refresh)

async function runImport(fn) {
  busy.value = true
  try {
    const res = await fn()
    const ok = res?.imported?.length || 0
    const failed = res?.failed || []
    if (ok) ui.showToast(`已导入 ${ok} 个文档`, 'success')
    failed.forEach(msg => ui.showToast(`导入失败：${msg}`, 'error', 4000))
    if (ok || failed.length) await refresh()
  } catch (e) {
    ui.showToast(`导入失败：${errText(e)}`, 'error', 4000)
  } finally {
    busy.value = false
  }
}

const importResume = () => runImport(api.importKnowledgeResume)
const importFiles = () => runImport(api.importKnowledgeFiles)

function newNote() {
  editing.value = { id: '', name: '', kind: 'note', scope: 'interview', content: '', origSize: 0 }
}

async function openDoc(doc) {
  try {
    const full = await api.getKnowledgeDoc(doc.id)
    editing.value = {
      id: full.id,
      name: full.name,
      kind: full.kind,
      scope: full.scope || 'interview',
      content: full.content || '',
      origSize: full.size || 0,
    }
  } catch (e) {
    ui.showToast(`读取失败：${errText(e)}`, 'error')
  }
}

function closeEditor() {
  editing.value = null
}

async function save() {
  const doc = editing.value
  if (!doc) return
  saving.value = true
  try {
    const saved = await api.saveKnowledgeDoc(doc.id, doc.name, doc.content)
    // 保存接口不带使用范围：新建笔记默认仅面试，编辑器里选了别的就紧接着补设
    let scopeFailed = ''
    if (saved?.id && saved.scope !== doc.scope) {
      try { await api.setKnowledgeScope(saved.id, doc.scope) } catch (e) { scopeFailed = errText(e) }
    }
    if (scopeFailed) ui.showToast(`已保存，但使用范围设置失败：${scopeFailed}`, 'warning', 4000)
    else ui.showToast('已保存', 'success')
    editing.value = null
    await refresh()
  } catch (e) {
    ui.showToast(`保存失败：${errText(e)}`, 'error', 4000)
  } finally {
    saving.value = false
  }
}

async function setScope(doc, scope) {
  try {
    await api.setKnowledgeScope(doc.id, scope)
    doc.scope = scope
  } catch (e) {
    ui.showToast(`设置失败：${errText(e)}`, 'error')
    await refresh()
  }
}

async function remove(doc) {
  try {
    await api.deleteKnowledgeDoc(doc.id)
    await refresh()
  } catch (e) {
    ui.showToast(`删除失败：${errText(e)}`, 'error')
  }
}
</script>

<style scoped>
.knowledge-manager { display: flex; flex-direction: column; gap: var(--sp-3); min-height: 100%; }
.hint-text { margin: 0; font-size: var(--text-xs); color: var(--text-muted); line-height: var(--leading-relaxed); }

.kb-quota { display: flex; align-items: center; gap: var(--sp-2); margin-top: var(--sp-2); }
.kb-quota-bar { flex: 1; height: 4px; border-radius: var(--radius-full); background: var(--border-default); overflow: hidden; }
.kb-quota-fill { height: 100%; background: var(--accent); transition: width var(--duration-base) ease; }
.kb-quota-fill.warn { background: var(--color-warning); }
.kb-quota-text { font-size: var(--text-xs); font-family: var(--font-mono); color: var(--text-muted); flex-shrink: 0; }
.kb-quota-text.warn { color: var(--color-warning); }

.kb-actions { display: flex; flex-wrap: wrap; gap: var(--sp-2); margin-top: var(--sp-2); }
.kb-btn {
  display: inline-flex; align-items: center; gap: var(--sp-1-5, 6px);
  padding: var(--sp-1-5, 6px) var(--sp-3);
  background: var(--surface-card); color: var(--text-secondary);
  border: 1px solid var(--border-default); border-radius: var(--radius-sm);
  font-size: var(--text-xs); font-weight: var(--weight-medium); cursor: pointer;
  transition: all var(--duration-fast) ease;
}
.kb-btn:hover:not(:disabled) { background: var(--surface-card-hover); color: var(--accent); }
.kb-btn.primary { background: var(--accent); color: #fff; border-color: transparent; }
.kb-btn.primary:hover:not(:disabled) { background: var(--accent-hover); color: #fff; }
.kb-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.kb-busy { display: flex; align-items: center; gap: var(--sp-1-5, 6px); margin-top: var(--sp-2); color: var(--accent); }
.kb-spin { animation: kb-rotate 0.8s linear infinite; }
@keyframes kb-rotate { to { transform: rotate(360deg); } }

.kb-empty {
  display: flex; flex-direction: column; align-items: center; gap: var(--sp-1);
  padding: var(--sp-6) 0; color: var(--text-muted); text-align: center;
}
.kb-empty p { margin: 0; font-size: var(--text-sm); }
.kb-empty .hint-text { margin-top: var(--sp-1); }

.kb-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--sp-2); }
.kb-item {
  display: flex; flex-direction: column; gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  background: var(--surface-card); border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}
.kb-row { display: flex; align-items: center; gap: var(--sp-2); }
.kb-scope-row { display: flex; align-items: center; gap: var(--sp-2); }
.kb-scope-label { font-size: var(--text-xs); color: var(--text-muted); flex-shrink: 0; }
.kb-main {
  flex: 1; min-width: 0; display: flex; align-items: center; gap: var(--sp-2);
  background: transparent; border: none; padding: 0; cursor: pointer;
  color: var(--text-primary); text-align: left;
}
.kb-kind-icon { color: var(--text-muted); flex-shrink: 0; }
.kb-name { font-size: var(--text-sm); font-weight: var(--weight-semibold); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.kb-badge {
  flex-shrink: 0; font-size: 10px; font-weight: var(--weight-bold);
  padding: 1px 5px; border-radius: var(--radius-xs);
  background: var(--surface-input); color: var(--text-muted);
}
.kb-badge.resume { background: var(--accent-muted); color: var(--accent); }
.kb-size { margin-left: auto; font-size: var(--text-xs); color: var(--text-muted); flex-shrink: 0; }
.kb-icon-btn {
  width: 26px; height: 26px; display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  background: transparent; border: none; border-radius: var(--radius-sm);
  color: var(--text-muted); cursor: pointer;
}
.kb-icon-btn.danger:hover:not(:disabled) { color: var(--color-error); background: var(--surface-card-hover); }
.kb-icon-btn:disabled { opacity: 0.4; cursor: not-allowed; }

.kb-editor-head { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-2); }
.kb-count { font-size: var(--text-xs); font-family: var(--font-mono); color: var(--text-muted); }
.kb-count.warn { color: var(--color-error); }
.kb-input {
  width: 100%; box-sizing: border-box;
  padding: 7px 12px; border-radius: var(--radius-md);
  border: 1px solid var(--border-default); background: var(--surface-card);
  color: var(--text-primary); font-size: var(--text-sm); outline: none;
  transition: border-color var(--duration-fast) ease;
}
.kb-input:focus { border-color: var(--accent); }
.kb-editor {
  flex: 1; width: 100%; min-height: 280px; box-sizing: border-box;
  padding: var(--sp-3); border-radius: var(--radius-md);
  border: 1px solid var(--border-default); background: var(--surface-card);
  color: var(--text-primary); font-family: var(--font-mono); font-size: var(--text-sm);
  line-height: var(--leading-relaxed); resize: vertical; outline: none;
  transition: border-color var(--duration-fast) ease;
}
.kb-editor:focus { border-color: var(--accent); }
.kb-editor::placeholder, .kb-input::placeholder { color: var(--text-muted); }
.kb-over { color: var(--color-error); }
</style>
