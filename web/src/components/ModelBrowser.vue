<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { NButton, NInput, NSelect, NSpin, NTag, NAlert, useMessage } from 'naive-ui'
import { api } from '../api/client'
import { upsertDownload } from '../store'
import type { CatalogEntry } from '../api/types'

// 模型库浏览：精选清单 + Civitai 搜索，省得去别的网站找。
const message = useMessage()
const entries = ref<CatalogEntry[]>([])
const warnings = ref<string[]>([])
const dirs = ref<string[]>([])
const loading = ref(false)
const busy = ref('')

const query = ref('')
const kind = ref<string | null>('')
const source = ref<string | null>('')
// 用户可以改目标目录：启发式判断不一定对，猜错会让模型静默失效。
const dirOverride = ref<Record<string, string>>({})

const KINDS = [
  { label: '全部类型', value: '' },
  { label: '底模', value: 'checkpoints' },
  { label: 'LoRA', value: 'loras' },
  { label: 'VAE', value: 'vae' },
  { label: 'ControlNet', value: 'controlnet' },
  { label: '放大模型', value: 'upscale_models' },
  { label: 'Embedding', value: 'embeddings' },
]

const SOURCES = [
  { label: '全部来源', value: '' },
  { label: '精选清单', value: 'curated' },
  { label: 'Civitai', value: 'civitai' },
]

async function load() {
  loading.value = true
  try {
    const r = await api.catalog({
      q: query.value.trim(),
      kind: kind.value || undefined,
      source: source.value || undefined,
      limit: 24,
    })
    entries.value = r.entries
    warnings.value = r.warnings ?? []
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    loading.value = false
  }
}

let timer: number | undefined
watch([query, kind, source], () => {
  clearTimeout(timer)
  timer = window.setTimeout(load, 300)
})

onMounted(async () => {
  await load()
  try {
    dirs.value = await api.modelDirs()
  } catch {
    /* 目录列表拿不到就退化成只读显示 */
  }
})

const dirOptions = () => dirs.value.map((d) => ({ label: d, value: d }))

function keyOf(e: CatalogEntry) {
  return e.source + ':' + e.id
}

async function download(e: CatalogEntry) {
  const k = keyOf(e)
  busy.value = k
  try {
    const d = await api.catalogDownload({
      source: e.source,
      id: e.id,
      query: query.value.trim(),
      kind: kind.value || undefined,
      dir: dirOverride.value[k] || e.dir || undefined,
    })
    upsertDownload(d)
    message.success(`已加入下载队列：${e.filename}`)
  } catch (err) {
    message.error(String((err as Error).message))
  } finally {
    busy.value = ''
  }
}

function mb(n?: number) {
  if (!n) return ''
  return n >= 1 << 30 ? `${(n / (1 << 30)).toFixed(2)} GB` : `${(n / (1 << 20)).toFixed(0)} MB`
}
</script>

<template>
  <div class="browser">
    <div class="bar">
      <NInput v-model:value="query" placeholder="搜索模型名称" clearable class="search" />
      <NSelect v-model:value="kind" :options="KINDS" class="pick" />
      <NSelect v-model:value="source" :options="SOURCES" class="pick" />
    </div>

    <NAlert v-for="wtext in warnings" :key="wtext" type="warning" :bordered="false" class="warn">
      {{ wtext }}
    </NAlert>

    <div v-if="loading && !entries.length" class="center"><NSpin /></div>
    <div v-else-if="!entries.length" class="empty dt-faint">没有匹配的模型</div>

    <div v-else class="grid">
      <article
        v-for="(e, i) in entries"
        :key="keyOf(e)"
        class="item dt-panel dt-enter"
        :style="{ animationDelay: `${Math.min(i, 12) * 25}ms` }"
      >
        <span v-if="e.preview" class="thumb dt-swatch">
          <img :src="e.preview" alt="" loading="lazy" referrerpolicy="no-referrer" />
        </span>

        <div class="body">
          <div class="top">
            <span class="name" :title="e.name">{{ e.name }}</span>
            <NTag size="tiny" :bordered="false" :type="e.source === 'curated' ? 'success' : 'default'">
              {{ e.source === 'curated' ? '精选' : 'Civitai' }}
            </NTag>
            <NTag v-if="e.installed" size="tiny" type="info" :bordered="false">已装</NTag>
          </div>

          <p class="facts dt-faint dt-mono">
            <span v-if="e.base">{{ e.base }}</span>
            <span v-if="e.size_bytes">{{ mb(e.size_bytes) }}</span>
            <span v-if="e.downloads">↓{{ e.downloads }}</span>
            <span v-if="e.author">{{ e.author }}</span>
          </p>

          <p v-if="e.trained_words?.length" class="words dt-faint">
            触发词：<span class="dt-mono">{{ e.trained_words.slice(0, 2).join(' / ') }}</span>
          </p>
          <p v-else-if="e.description" class="words dt-muted">{{ e.description }}</p>

          <div class="row">
            <NSelect
              v-if="dirs.length"
              :value="dirOverride[keyOf(e)] ?? e.dir"
              :options="dirOptions()"
              size="tiny"
              filterable
              placeholder="选择目录"
              class="dirsel"
              :status="!(dirOverride[keyOf(e)] ?? e.dir) ? 'warning' : undefined"
              @update:value="(v: string) => (dirOverride[keyOf(e)] = v)"
            />
            <span v-else class="dt-mono dt-faint dirtext">{{ e.dir || '目录未知' }}</span>

            <NButton
              size="tiny"
              :type="e.installed ? 'default' : 'primary'"
              :disabled="e.installed"
              :loading="busy === keyOf(e)"
              @click="download(e)"
            >
              {{ e.installed ? '已装' : '下载' }}
            </NButton>
            <a v-if="e.page" :href="e.page" target="_blank" rel="noreferrer" class="page-link">页面 ↗</a>
          </div>

          <p v-if="e.auth !== 'none'" class="auth dt-faint">
            需要 {{ e.auth === 'civitai-login' ? 'Civitai' : 'HuggingFace' }} 令牌，可在下方设置里填
          </p>
        </div>
      </article>
    </div>
  </div>
</template>

<style scoped>
.browser {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.bar {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
.search {
  max-width: 300px;
}
.pick {
  width: 140px;
}
.warn {
  font-size: var(--dt-fs-base);
  line-height: 1.6;
}
.center {
  display: grid;
  place-items: center;
  height: 24vh;
}
.empty {
  padding: 50px 0;
  text-align: center;
  border: 1px dashed var(--dt-border);
  border-radius: var(--dt-radius-card);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 10px;
}
.item {
  display: flex;
  gap: 12px;
  padding: 11px 13px;
}
.thumb {
  flex: none;
  width: 68px;
  height: 68px;
  border-radius: var(--dt-radius);
  overflow: hidden;
  border: 1px solid var(--dt-border);
}
.thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.top {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.name {
  font-size: var(--dt-fs-base);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 200px;
}
.facts {
  margin: 0;
  font-size: var(--dt-fs-2xs);
  display: flex;
  gap: 9px;
  flex-wrap: wrap;
}
.words {
  margin: 0;
  font-size: var(--dt-fs-xs);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.row {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 4px;
}
.dirsel {
  width: 148px;
}
.dirtext {
  font-size: var(--dt-fs-xs);
}
.page-link {
  font-size: var(--dt-fs-xs);
  color: var(--dt-accent);
  text-decoration: none;
}
.auth {
  margin: 2px 0 0;
  font-size: var(--dt-fs-2xs);
}
</style>
