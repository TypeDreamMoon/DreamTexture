<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { NButton, NInput, NSelect, NTag, NAlert, NSpin, useMessage, useDialog } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../api/client'
import { persisted } from '../persist'
import { health } from '../store'
import type { ManagerCapability, NodePack, NodeQueue } from '../api/types'

const message = useMessage()
const dialog = useDialog()

const cap = ref<ManagerCapability | null>(null)
const packs = ref<NodePack[]>([])
const total = ref(0)
const loading = ref(true)
const query = persisted<string>('nodes.query', '')
const state = ref<string>('')
const busy = ref('')
const queue = ref<NodeQueue | null>(null)
const needsRestart = ref(false)

const STATES = [
  { label: '全部', value: '' },
  { label: '已安装', value: 'installed' },
  { label: '未安装', value: 'not-installed' },
  { label: '已停用', value: 'disabled' },
]

const comfyURL = computed(() => health.value?.base_url ?? 'http://127.0.0.1:8188')

async function load() {
  loading.value = true
  try {
    const r = await api.nodes({ q: query.value.trim(), state: state.value, limit: 40 })
    packs.value = r.packs
    total.value = r.total
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    loading.value = false
  }
}

let timer: number | undefined
watch([query, state], () => {
  clearTimeout(timer)
  timer = window.setTimeout(load, 260)
})

// 安装是异步的（Manager 排队跑 git clone + pip install），轮询队列给个进度感。
//
// 判定完成不能用「不在处理且 total>0」：Manager 干完活会把计数清零，那个条件
// 永远等不到，界面就一直转圈。改成「见过在处理，之后不在处理了」就算完，
// 另加一个上限兜住任务瞬间结束、一次都没轮询到「处理中」的情况。
let poll: number | undefined
function startPolling() {
  clearInterval(poll)
  let sawProcessing = false
  let ticks = 0
  poll = window.setInterval(async () => {
    ticks++
    try {
      const q = await api.nodeQueue()
      queue.value = q
      if (q.is_processing) {
        sawProcessing = true
        return
      }
      if (sawProcessing || ticks >= 3) {
        clearInterval(poll)
        queue.value = null
        needsRestart.value = true
        await load()
      }
    } catch {
      /* ComfyUI 重启途中会断，忽略这一轮 */
    }
  }, 2000)
}
onBeforeUnmount(() => {
  clearTimeout(timer)
  clearInterval(poll)
})

onMounted(async () => {
  try {
    cap.value = await api.managerInfo()
  } catch {
    cap.value = { available: false, has_node_list: false, reason: '无法连接后端' }
  }
  if (cap.value?.available) await load()
  else loading.value = false
})

async function act(p: NodePack, action: string) {
  busy.value = p.id
  try {
    const r = await api.nodeAction(p.id, action)
    message.success(r.hint)
    startPolling()
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    busy.value = ''
  }
}

function confirmUninstall(p: NodePack) {
  dialog.warning({
    title: '卸载节点包',
    content: `确定卸载 ${p.title || p.id}？依赖它的工作流会无法运行。`,
    positiveText: '卸载',
    negativeText: '取消',
    onPositiveClick: () => act(p, 'uninstall'),
  })
}

async function restartComfy() {
  try {
    await api.comfyRestart()
    needsRestart.value = false
    message.success('ComfyUI 正在重启，稍等片刻')
  } catch (e) {
    message.error(String((e as Error).message))
  }
}

function stateLabel(s: string) {
  return s === 'enabled' ? '已启用' : s === 'disabled' ? '已停用' : '未安装'
}
</script>

<template>
  <div class="dt-page">
    <PageHeader
      title="节点"
      desc="ComfyUI 的自定义节点包。装上之后工作流里才能用到对应的节点，装完需要重启 ComfyUI 才会注册。"
    >
      <template #actions>
        <span v-if="cap?.available" class="ver dt-mono dt-faint">
          Manager {{ cap.version }} · {{ cap.api }}
        </span>
        <a :href="comfyURL" target="_blank" rel="noreferrer">
          <NButton size="small">打开 ComfyUI</NButton>
        </a>
      </template>
    </PageHeader>

    <NAlert v-if="cap && !cap.available" type="warning" :bordered="false">
      {{ cap.reason }}
    </NAlert>

    <NAlert
      v-else-if="cap && !cap.has_node_list"
      type="warning"
      :bordered="false"
      class="tip"
    >
      当前的 ComfyUI-Manager（{{ cap.version }}，{{ cap.api }} 接口）没有提供节点目录端点。
      用 <span class="dt-mono">--enable-manager-legacy-ui</span> 启动 ComfyUI 可以恢复它。
    </NAlert>

    <template v-else-if="cap?.available">
      <NAlert v-if="needsRestart" type="info" :bordered="false" closable class="tip"
              @close="needsRestart = false">
        节点变更已完成。<b>需要重启 ComfyUI 才会生效。</b>
        <NButton size="tiny" type="primary" class="inline-btn" @click="restartComfy">
          立即重启
        </NButton>
      </NAlert>

      <div class="bar">
        <NInput v-model:value="query" placeholder="搜索节点包（名称、作者、描述）" clearable class="search" />
        <NSelect v-model:value="state" :options="STATES" class="pick" />
        <span v-if="queue?.is_processing" class="working dt-faint">
          <NSpin :size="12" /> 正在处理 {{ queue.in_progress_count }} 项
        </span>
        <span class="count dt-faint dt-mono">{{ total }} 个匹配</span>
      </div>

      <div v-if="loading" class="center"><NSpin /></div>

      <div v-else-if="!packs.length" class="empty dt-faint">没有匹配的节点包</div>

      <div v-else class="list">
        <article
          v-for="(p, i) in packs"
          :key="p.id"
          class="pack dt-panel dt-enter"
          :class="{ on: p.state === 'enabled' }"
          :style="{ animationDelay: `${Math.min(i, 12) * 25}ms` }"
        >
          <div class="main">
            <div class="top">
              <span class="name">{{ p.title || p.id }}</span>
              <NTag
                size="tiny"
                :bordered="false"
                :type="p.state === 'enabled' ? 'success' : p.state === 'disabled' ? 'warning' : 'default'"
              >
                {{ stateLabel(p.state) }}
              </NTag>
              <span v-if="p.stars" class="dt-mono dt-faint stars">★ {{ p.stars }}</span>
              <span class="dt-faint author">{{ p.author }}</span>
            </div>
            <p class="desc dt-muted">{{ p.description }}</p>
            <p class="meta dt-faint dt-mono">
              {{ p.id }}<template v-if="p.last_update"> · 更新于 {{ p.last_update }}</template>
            </p>
          </div>

          <div class="side">
            <NButton
              v-if="p.state === 'not-installed'"
              size="tiny"
              type="primary"
              :loading="busy === p.id"
              @click="act(p, 'install')"
            >
              安装
            </NButton>
            <template v-else>
              <NButton
                size="tiny"
                :loading="busy === p.id"
                @click="act(p, p.state === 'enabled' ? 'disable' : 'enable')"
              >
                {{ p.state === 'enabled' ? '停用' : '启用' }}
              </NButton>
              <NButton size="tiny" tertiary :loading="busy === p.id" @click="act(p, 'update')">
                更新
              </NButton>
              <NButton size="tiny" tertiary @click="confirmUninstall(p)">卸载</NButton>
            </template>
            <a v-if="p.repository" :href="p.repository" target="_blank" rel="noreferrer" class="repo">
              仓库 ↗
            </a>
          </div>
        </article>
      </div>
    </template>
  </div>
</template>

<style scoped>
.ver {
  font-size: var(--dt-fs-xs);
  white-space: nowrap;
}
a {
  text-decoration: none;
}
.tip {
  margin-bottom: 14px;
  line-height: 1.7;
  font-size: var(--dt-fs-base);
}
.tip b {
  font-weight: 500;
}
.inline-btn {
  margin-left: 10px;
}

.bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}
.search {
  max-width: 340px;
}
.pick {
  width: 130px;
}
.working {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: var(--dt-fs-sm);
}
.count {
  margin-left: auto;
  font-size: var(--dt-fs-sm);
}

.center {
  display: grid;
  place-items: center;
  height: 30vh;
}
.empty {
  padding: 60px 0;
  text-align: center;
  border: 1px dashed var(--dt-border);
  border-radius: var(--dt-radius-card);
}

.list {
  display: flex;
  flex-direction: column;
  gap: 9px;
}
.pack {
  display: flex;
  gap: 16px;
  padding: 13px 15px;
  border-left-width: 3px;
  border-left-color: transparent;
  transition: border-color 0.2s ease;
}
.pack.on {
  border-left-color: var(--dt-accent);
}
.main {
  flex: 1;
  min-width: 0;
}
.top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.name {
  font-size: var(--dt-fs-md);
  font-weight: 500;
}
.stars,
.author {
  font-size: var(--dt-fs-xs);
}
.author {
  margin-left: auto;
}
.desc {
  margin: 6px 0 0;
  font-size: var(--dt-fs-base);
  line-height: 1.6;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.meta {
  margin: 5px 0 0;
  font-size: var(--dt-fs-2xs);
}
.side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 5px;
  flex: none;
}
.repo {
  font-size: var(--dt-fs-xs);
  color: var(--dt-accent);
  text-decoration: none;
}
.repo:hover {
  text-decoration: underline;
}
</style>
