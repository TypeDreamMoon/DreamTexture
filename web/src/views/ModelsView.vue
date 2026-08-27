<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NButton, NProgress, NSpin, NTag, NAlert, useMessage } from 'naive-ui'
import { RouterLink } from 'vue-router'
import ModelBrowser from '../components/ModelBrowser.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../api/client'
import { downloads, upsertDownload } from '../store'
import type { Inventory, ModelRequirement } from '../api/types'

const message = useMessage()
const inv = ref<Inventory | null>(null)
const loading = ref(true)
const scanning = ref(false)
const error = ref('')

// 「本机」看已有的和工作流缺什么，「浏览」去库里找新的。
const mode = ref<'local' | 'browse'>('local')
const MODES = [
  { key: 'local' as const, label: '本机' },
  { key: 'browse' as const, label: '浏览下载' },
]



async function load(refresh = false) {
  if (refresh) scanning.value = true
  error.value = ''
  try {
    const m = await api.models(refresh)
    inv.value = m.inventory
    m.downloads.forEach(upsertDownload)
  } catch (e) {
    error.value = String((e as Error).message)
  } finally {
    loading.value = false
    scanning.value = false
  }
}

onMounted(() => load())

const missing = computed(() => inv.value?.requirements.filter((r) => !r.present) ?? [])
const present = computed(() => inv.value?.requirements.filter((r) => r.present) ?? [])

// 某个模型正在下载中的记录。
function downloadOf(r: ModelRequirement) {
  return downloads.value.find(
    (d) => d.file === r.file && d.dir === r.dir && (d.state === 'queued' || d.state === 'running'),
  )
}

function lastFailure(r: ModelRequirement) {
  return downloads.value.find((d) => d.file === r.file && d.dir === r.dir && d.state === 'failed')
}

async function start(r: ModelRequirement) {
  try {
    const d = await api.downloadModel(r.file, r.dir)
    upsertDownload(d)
    message.success(`已加入下载队列：${r.file}`)
  } catch (e) {
    message.error(String((e as Error).message))
  }
}

async function cancel(id: string) {
  try {
    await api.cancelDownload(id)
  } catch (e) {
    message.error(String((e as Error).message))
  }
}




function gb(n: number) {
  if (n >= 1 << 30) return `${(n / (1 << 30)).toFixed(2)} GB`
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(0)} MB`
  return `${(n / 1024).toFixed(0)} KB`
}
</script>

<template>
  <div class="dt-page">
    <PageHeader
      title="模型"
      desc="工作流需要哪些模型、本机有没有、占了多少盘。缺的能直接下，也可以去库里找新的。"
    >
      <template #actions>
        <div class="modes">
          <button
            v-for="m in MODES"
            :key="m.key"
            class="mode"
            :class="{ on: mode === m.key }"
            @click="mode = m.key"
          >
            {{ m.label }}
          </button>
        </div>
        <NButton v-if="mode === 'local'" size="small" :loading="scanning" @click="load(true)">
          重新盘点
        </NButton>
      </template>
    </PageHeader>

    <ModelBrowser v-if="mode === 'browse'" />
    <template v-else>

    <div v-if="loading" class="center"><NSpin /></div>
    <NAlert v-else-if="error" type="error" :bordered="false">{{ error }}</NAlert>

    <template v-else-if="inv">
      <!-- 缺失的排最前，红标 -->
      <section v-if="missing.length" class="block">
        <p class="dt-label">工作流所需，尚未就位（{{ missing.length }}）</p>
        <div class="reqs">
          <div v-for="r in missing" :key="r.dir + r.file" class="req dt-panel bad">
            <div class="req-main">
              <div class="req-top">
                <span class="file dt-mono">{{ r.file }}</span>
                <NTag size="tiny" :bordered="false">{{ r.dir }}</NTag>
                <span v-if="r.size_bytes" class="dt-faint dt-mono size">{{ gb(r.size_bytes) }}</span>
              </div>
              <p class="used dt-faint">被 {{ r.workflow_ids.join('、') }} 使用</p>
              <p v-if="r.note" class="note">{{ r.note }}</p>
              <p v-if="lastFailure(r)" class="fail">{{ lastFailure(r)!.error }}</p>

              <div v-if="downloadOf(r)" class="dl">
                <NProgress
                  type="line"
                  :percentage="downloadOf(r)!.total ? Math.round((downloadOf(r)!.received / downloadOf(r)!.total) * 100) : 0"
                  :height="5"
                  :show-indicator="false"
                />
                <span class="dt-mono dt-faint prog">
                  {{ gb(downloadOf(r)!.received) }}<template v-if="downloadOf(r)!.total"> / {{ gb(downloadOf(r)!.total) }}</template>
                </span>
              </div>
            </div>

            <div class="req-act">
              <NButton v-if="downloadOf(r)" size="tiny" tertiary @click="cancel(downloadOf(r)!.id)">
                取消
              </NButton>
              <NButton
                v-else-if="r.download_url"
                size="tiny"
                :type="r.auth === 'none' ? 'primary' : 'default'"
                @click="start(r)"
              >
                {{ r.auth === 'none' ? '下载' : '下载（需令牌）' }}
              </NButton>
              <a
                v-if="r.source && (!downloadOf(r) || r.auth !== 'none')"
                :href="r.source"
                target="_blank"
                rel="noreferrer"
                class="manual"
              >
                打开来源页
              </a>
            </div>
          </div>
        </div>
      </section>

      <section v-if="present.length" class="block">
        <p class="dt-label">工作流所需，已就位（{{ present.length }}）</p>
        <div class="reqs">
          <div v-for="r in present" :key="r.dir + r.file" class="req dt-panel ok">
            <div class="req-main">
              <div class="req-top">
                <span class="file dt-mono">{{ r.file }}</span>
                <NTag size="tiny" :bordered="false">{{ r.dir }}</NTag>
                <span class="dt-faint dt-mono size">{{ gb(r.actual_bytes ?? 0) }}</span>
              </div>
              <p class="used dt-faint">被 {{ r.workflow_ids.join('、') }} 使用</p>
            </div>
          </div>
        </div>
      </section>

      <!-- 磁盘占用 -->
      <section class="block">
        <p class="dt-label">磁盘占用 · 合计 {{ gb(inv.total_bytes) }}</p>
        <div class="folders">
          <div v-for="f in inv.folders" :key="f.name" class="folder dt-panel">
            <div class="f-top">
              <span class="f-name dt-mono">{{ f.name }}</span>
              <span class="dt-faint dt-mono">{{ f.count }} 个 · {{ gb(f.total_bytes) }}</span>
            </div>
            <p v-for="p in f.paths" :key="p" class="path dt-mono dt-faint">{{ p }}</p>
          </div>
        </div>
      </section>

      <!-- 令牌与接口地址已挪到设置页：那里集中放令牌、网络、环境这些
           全局性的东西，模型页只管模型本身。 -->
      <section class="block">
        <p class="dt-label">访问令牌</p>
        <p class="tip dt-muted">
          下载受限模型需要的令牌（HuggingFace / Civitai）已移到
          <RouterLink to="/settings" class="lnk">设置</RouterLink> 页统一管理。
        </p>
      </section>
    </template>
    </template>
  </div>
</template>

<style scoped>
.lnk {
  color: var(--dt-accent);
}
.modes {
  display: flex;
  gap: 4px;
  padding: 3px;
  border-radius: var(--dt-radius);
  background: var(--dt-surface2);
}
.mode {
  font: inherit;
  font-size: var(--dt-fs-base);
  padding: 4px 14px;
  border: none;
  border-radius: calc(var(--dt-radius) - 2px);
  background: transparent;
  color: var(--dt-ink-muted);
  cursor: pointer;
  transition:
    background 0.2s ease,
    color 0.2s ease;
}
.mode:hover {
  color: var(--dt-ink);
}
.mode.on {
  background: var(--dt-surface);
  color: var(--dt-ink);
  box-shadow: 0 1px 4px var(--dt-shadow);
}
.center {
  display: grid;
  place-items: center;
  height: 40vh;
}

.block + .block {
  margin-top: 26px;
}
.block .dt-label {
  margin: 0 0 10px;
}

.reqs {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.req {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 12px 14px;
  border-left-width: 3px;
}
.req.bad {
  border-left-color: var(--dt-danger);
}
.req.ok {
  border-left-color: var(--dt-ok);
}
.req-main {
  flex: 1;
  min-width: 0;
}
.req-top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.file {
  font-size: var(--dt-fs-base);
  word-break: break-all;
}
.size {
  font-size: var(--dt-fs-xs);
}
.used {
  margin: 5px 0 0;
  font-size: var(--dt-fs-sm);
}
.note {
  margin: 5px 0 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.55;
  color: var(--dt-warn);
}
.fail {
  margin: 6px 0 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.55;
  color: var(--dt-danger);
}
.dl {
  margin-top: 9px;
  display: flex;
  align-items: center;
  gap: 10px;
}
.dl .n-progress {
  flex: 1;
}
.prog {
  font-size: var(--dt-fs-xs);
  white-space: nowrap;
}
.req-act {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  flex: none;
}
.manual {
  font-size: var(--dt-fs-xs);
  color: var(--dt-accent);
  text-decoration: none;
  border-bottom: 1px solid transparent;
}
.manual:hover {
  border-bottom-color: var(--dt-accent);
}

.folders {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 8px;
}
.folder {
  padding: 11px 13px;
}
.f-top {
  display: flex;
  align-items: baseline;
  gap: 10px;
}
.f-name {
  font-size: var(--dt-fs-base);
}
.f-top span:last-child {
  margin-left: auto;
  font-size: var(--dt-fs-xs);
}
.path {
  margin: 5px 0 0;
  font-size: var(--dt-fs-2xs);
  word-break: break-all;
  line-height: 1.5;
}

.tip {
  margin: 0 0 12px;
  font-size: var(--dt-fs-base);
  line-height: 1.6;
}
.tokens {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 10px;
}
.token {
  padding: 13px 15px;
}
.t-top {
  display: flex;
  align-items: center;
  gap: 9px;
}
.t-name {
  font-size: var(--dt-fs-base);
  font-weight: 500;
}
.t-hint {
  margin: 5px 0 10px;
  font-size: var(--dt-fs-sm);
  line-height: 1.6;
}
.t-sub {
  margin: 14px 0 0;
  padding-top: 12px;
  border-top: 1px solid var(--dt-border);
}
.t-now {
  margin: 7px 0 0;
  font-size: var(--dt-fs-2xs);
}
.t-now b {
  font-weight: 500;
  color: var(--dt-ink-muted);
}
.t-row {
  display: flex;
  gap: 8px;
}
</style>
