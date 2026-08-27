<script setup lang="ts">
import { computed, ref } from 'vue'
import { NButton, NInput, NSelect, NModal, NCheckbox, NTag, NAlert, useMessage } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../api/client'
import { workflows, health } from '../store'

const message = useMessage()
const busy = ref('')
const reloading = ref(false)
const lastExport = ref<{ id: string; file: string } | null>(null)

const comfyURL = computed(() => health.value?.base_url ?? 'http://127.0.0.1:8188')

async function refresh() {
  reloading.value = true
  try {
    await api.reloadWorkflows()
    workflows.value = await api.workflows()
    message.success('已重新加载')
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    reloading.value = false
  }
}

// 把模板转成编辑器格式送进 ComfyUI 的工作流列表。
// 直接把 API 格式丢过去是打不开的（编辑器会得到一张空图），所以这一步必须经后端转换。
async function editInComfy(id: string) {
  busy.value = id
  try {
    const r = await api.openInComfy(id)
    lastExport.value = { id, file: r.file }
    window.open(comfyURL.value, '_blank', 'noopener')
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    busy.value = ''
  }
}

// ---- 导入 ----
const showImport = ref(false)
const form = ref({ id: '', name: '', style: 'custom', override: false })
const graphText = ref('')
const importing = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

const STYLES = [
  { label: '写实', value: 'realistic' },
  { label: '风格化', value: 'stylized' },
  { label: '自定义', value: 'custom' },
]

function pickFile(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  const reader = new FileReader()
  reader.onload = () => {
    graphText.value = String(reader.result ?? '')
    if (!form.value.id) form.value.id = f.name.replace(/\.json$/i, '').toLowerCase()
  }
  reader.readAsText(f)
}

async function doImport() {
  let graph: unknown
  try {
    graph = JSON.parse(graphText.value)
  } catch {
    message.error('内容不是合法 JSON')
    return
  }
  importing.value = true
  try {
    await api.importWorkflow({ ...form.value, graph })
    workflows.value = await api.workflows()
    message.success(`已导入 ${form.value.id}`)
    showImport.value = false
    graphText.value = ''
    form.value = { id: '', name: '', style: 'custom', override: false }
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    importing.value = false
  }
}
</script>

<template>
  <div class="dt-page">
    <PageHeader
      title="工作流"
      desc="生成台上的每个风格预设就是这里的一套工作流。可以在 ComfyUI 里改完再导回来。"
    >
      <template #actions>
        <a :href="comfyURL" target="_blank" rel="noreferrer">
          <NButton size="small">打开 ComfyUI</NButton>
        </a>
        <NButton size="small" :loading="reloading" @click="refresh">重新加载</NButton>
        <NButton size="small" type="primary" @click="showImport = true">导入</NButton>
      </template>
    </PageHeader>

    <NAlert v-if="lastExport" type="success" :bordered="false" closable class="tip"
            @close="lastExport = null">
      <b>{{ lastExport.id }}</b> 已放进 ComfyUI 的工作流列表。在 ComfyUI 左侧「工作流」面板里打开
      <span class="dt-mono">{{ lastExport.file }}</span>，改完用「工作流 → 导出(API)」导出，
      再回这里点「导入」覆盖同名 id 即可生效。
    </NAlert>

    <div class="list">
      <article v-for="(w, i) in workflows" :key="w.id" class="wf dt-panel dt-enter"
               :style="{ animationDelay: `${i * 40}ms` }">
        <div class="top">
          <div class="title">
            <span class="name">{{ w.name }}</span>
            <NTag size="tiny" :bordered="false">{{ w.style }}</NTag>
            <NTag size="tiny" :bordered="false">{{ w.resolution }}²</NTag>
            <NTag v-if="w.tileable" size="tiny" :bordered="false">无缝</NTag>
            <NTag v-if="w.license_notice && !w.license_notice.commercial" size="tiny"
                  type="warning" :bordered="false">不可商用</NTag>
          </div>
          <span class="id dt-mono dt-faint">{{ w.id }} v{{ w.version }}</span>
        </div>

        <p class="desc dt-muted">{{ w.description }}</p>

        <dl class="facts">
          <dt>可调参数</dt>
          <dd>{{ w.params.length }} 基础 · {{ w.advanced.length }} 高级</dd>
          <dt>输出通道</dt>
          <dd class="dt-mono">{{ Object.keys(w.outputs).sort().join(' ') }}</dd>
          <template v-if="w.node_packs?.length">
            <dt>节点包</dt>
            <dd class="dt-mono">{{ w.node_packs.join(' · ') }}</dd>
          </template>
        </dl>

        <div class="row">
          <NButton size="tiny" :loading="busy === w.id" @click="editInComfy(w.id)">
            在 ComfyUI 中编辑
          </NButton>
          <a :href="`/api/workflows/${w.id}/template`" :download="`${w.id}.json`">
            <NButton size="tiny" tertiary>下载模板</NButton>
          </a>
        </div>
      </article>
    </div>

    <NModal
      v-model:show="showImport"
      preset="card"
      title="导入工作流"
      style="max-width: 620px"
    >
      <p class="hint dt-muted">
        在 ComfyUI 里用<b>「工作流 → 导出(API)」</b>导出的 JSON。普通的保存/导出是 UI 格式，
        缺少 DreamTexture 需要的结构，导入会被拒绝。
      </p>
      <p class="hint dt-faint">
        输出节点需命名为 <span class="dt-mono">dt.out.basecolor</span> 这样的标题，
        DreamTexture 靠它把产物对上通道；提示词节点命名为
        <span class="dt-mono">dt.positive</span> 会自动生成对应的可调参数。
      </p>

      <div class="form">
        <label>
          <span class="dt-label">id</span>
          <NInput v-model:value="form.id" placeholder="小写字母、数字、. _ -" size="small" />
        </label>
        <label>
          <span class="dt-label">名称</span>
          <NInput v-model:value="form.name" placeholder="留空则用 id" size="small" />
        </label>
        <label>
          <span class="dt-label">风格</span>
          <NSelect v-model:value="form.style" :options="STYLES" size="small" />
        </label>
      </div>

      <div class="filerow">
        <NButton size="small" @click="fileInput?.click()">选择 JSON 文件</NButton>
        <NCheckbox v-model:checked="form.override" size="small">覆盖同名工作流</NCheckbox>
      </div>
      <input ref="fileInput" type="file" accept="application/json,.json" hidden @change="pickFile" />

      <NInput
        v-model:value="graphText"
        type="textarea"
        :autosize="{ minRows: 6, maxRows: 12 }"
        placeholder="也可以直接把 API 格式 JSON 粘贴到这里"
        class="ta"
      />

      <template #footer>
        <div class="footer">
          <NButton size="small" @click="showImport = false">取消</NButton>
          <NButton
            size="small"
            type="primary"
            :loading="importing"
            :disabled="!form.id || !graphText"
            @click="doImport"
          >
            导入
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
a {
  text-decoration: none;
}
.tip {
  margin-bottom: 16px;
  line-height: 1.7;
  font-size: var(--dt-fs-base);
}

.list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.wf {
  padding: 16px 18px;
}
.top {
  display: flex;
  align-items: baseline;
  gap: 12px;
  flex-wrap: wrap;
}
.title {
  display: flex;
  align-items: center;
  gap: 7px;
  flex-wrap: wrap;
}
.name {
  font-size: var(--dt-fs-md);
  font-weight: 500;
}
.id {
  margin-left: auto;
  font-size: var(--dt-fs-xs);
}
.desc {
  margin: 9px 0 0;
  font-size: var(--dt-fs-base);
  line-height: 1.65;
}
.facts {
  display: grid;
  grid-template-columns: 76px 1fr;
  gap: 5px 12px;
  margin: 12px 0 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.6;
}
.facts dt {
  color: var(--dt-ink-faint);
}
.facts dd {
  margin: 0;
  word-break: break-word;
}
.row {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}
.row a {
  text-decoration: none;
}

.hint {
  margin: 0 0 8px;
  font-size: var(--dt-fs-base);
  line-height: 1.7;
}
.hint b {
  font-weight: 500;
  color: var(--dt-ink);
}
.form {
  display: grid;
  grid-template-columns: 1fr 1fr 120px;
  gap: 10px;
  margin: 14px 0 12px;
}
.form label {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.filerow {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 10px;
}
.ta {
  font-family: 'JetBrains Mono', Consolas, monospace;
  font-size: var(--dt-fs-sm);
}
.footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
