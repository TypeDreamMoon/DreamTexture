<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { NInput, NSelect, NSpin, NSwitch } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import { api, fileURL } from '../api/client'
import { persisted, persistedEnum } from '../persist'
import type { MaterialIndex, Picture } from '../api/types'

// 材质与图片分档看：两者的卡片信息完全不同（材质关心风格与分辨率，
// 图片关心尺寸与花费），混在一个网格里只会互相干扰。
const kind = persistedEnum<'material' | 'picture'>('lib.kind', 'material', ['material', 'picture'])
const KINDS = [
  { label: '材质', value: 'material' as const },
  { label: '图片', value: 'picture' as const },
]

const items = ref<MaterialIndex[]>([])
const pics = ref<Picture[]>([])
const loading = ref(true)
const query = persisted<string>('lib.query', '')
const style = persisted<string | null>('lib.style', null)
const favOnly = persisted<boolean>('lib.fav', false)

const STYLES = [
  { label: '全部风格', value: '' },
  { label: '写实', value: 'realistic' },
  { label: '风格化', value: 'stylized' },
]

async function load() {
  loading.value = true
  try {
    if (kind.value === 'picture') {
      pics.value = (await api.pictures({
        q: query.value.trim() || undefined,
        fav: favOnly.value,
        limit: 120,
      })).pictures
    } else {
      items.value = (await api.materials({
        q: query.value.trim() || undefined,
        style: style.value || undefined,
        fav: favOnly.value,
        limit: 120,
      })).materials
    }
  } finally {
    loading.value = false
  }
}

// 边打字边搜，简单防抖即可，本地后端没有速率压力。
let timer: number | undefined
watch([query, style, favOnly], () => {
  clearTimeout(timer)
  timer = window.setTimeout(load, 220)
})
// 换档要立刻重拉，不走防抖——那是给打字准备的。
watch(kind, () => load())

const count = computed(() => (kind.value === 'picture' ? pics.value.length : items.value.length))
const empty = computed(() => count.value === 0)

onMounted(load)
</script>

<template>
  <div class="dt-page dt-page-wide">
    <PageHeader
      title="素材库"
      :desc="
        kind === 'material'
          ? '生成过的材质套装都在这儿。点开可以三维预览、逐通道检视、把参数回填去再来一张。'
          : '生成过的单张图片。可以收藏、下载，或者直接提升成参考图给下一轮用。'
      "
    >
      <template #actions>
        <span class="count dt-faint dt-mono">{{ count }} 项</span>
      </template>
    </PageHeader>

    <header class="bar">
      <div class="kinds">
        <button
          v-for="k in KINDS"
          :key="k.value"
          class="kind"
          :class="{ on: kind === k.value }"
          @click="kind = k.value"
        >
          {{ k.label }}
        </button>
      </div>
      <NInput
        v-model:value="query"
        placeholder="按名称或提示词搜索"
        clearable
        class="search"
      />
      <NSelect v-if="kind === 'material'" v-model:value="style" :options="STYLES" class="pick" />
      <label class="fav">
        <NSwitch v-model:value="favOnly" size="small" />
        <span class="dt-muted">只看收藏</span>
      </label>
    </header>

    <div v-if="loading && empty" class="center"><NSpin /></div>

    <div v-else-if="empty" class="empty dt-faint">
      <p>
        {{ query || favOnly || style ? '没有匹配的内容。'
           : kind === 'material' ? '还没有生成过材质。' : '还没有生成过图片。' }}
      </p>
      <RouterLink to="/generate" class="link">去生成台</RouterLink>
    </div>

    <div v-else-if="kind === 'material'" class="grid">
      <RouterLink
        v-for="(m, i) in items"
        :key="m.id"
        :to="`/material/${m.id}`"
        class="card dt-panel dt-enter"
        :style="{ animationDelay: `${Math.min(i, 10) * 30}ms` }"
      >
        <span class="thumb dt-swatch">
          <img :src="fileURL(m.id, 'preview.png')" alt="" loading="lazy" />
        </span>
        <span class="meta">
          <span class="name">{{ m.name }}</span>
          <span class="sub dt-faint dt-mono">
            {{ m.style === 'realistic' ? '写实' : '风格化' }} · {{ m.resolution }}²
            <span v-if="m.favorite" class="star">★</span>
          </span>
        </span>
      </RouterLink>
    </div>

    <div v-else class="grid">
      <RouterLink
        v-for="(p, i) in pics"
        :key="p.id"
        :to="`/picture/${p.id}`"
        class="card dt-panel dt-enter"
        :style="{ animationDelay: `${Math.min(i, 10) * 30}ms` }"
      >
        <span class="thumb dt-swatch">
          <img :src="`/api/pictures/${p.id}/file`" alt="" loading="lazy" />
        </span>
        <span class="meta">
          <span class="name">{{ p.name }}</span>
          <span class="sub dt-faint dt-mono">
            {{ p.width }}×{{ p.height }}
            <template v-if="p.cost_usd"> · ${{ p.cost_usd.toFixed(3) }}</template>
            <span v-if="p.favorite" class="star">★</span>
          </span>
        </span>
      </RouterLink>
    </div>
  </div>
</template>

<style scoped>
.bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.kinds {
  display: flex;
  gap: 4px;
  padding: 3px;
  border-radius: var(--dt-radius);
  background: var(--dt-surface2);
  flex: none;
}
.kind {
  font: inherit;
  font-size: var(--dt-fs-base);
  padding: 4px 16px;
  border: none;
  border-radius: calc(var(--dt-radius) - 2px);
  background: transparent;
  color: var(--dt-ink-muted);
  cursor: pointer;
  transition:
    background 0.16s ease,
    color 0.16s ease;
}
.kind:hover {
  color: var(--dt-ink);
}
.kind.on {
  background: var(--dt-surface);
  color: var(--dt-accent);
  font-weight: 500;
}
.search {
  max-width: 320px;
}
.pick {
  width: 140px;
}
.fav {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: var(--dt-fs-base);
  cursor: pointer;
}
.count {
  font-size: var(--dt-fs-sm);
  white-space: nowrap;
}

.center {
  display: grid;
  place-items: center;
  height: 40vh;
}
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 80px 0;
  border: 1px dashed var(--dt-border);
}
.empty p {
  margin: 0;
}
.link {
  color: var(--dt-accent);
  text-decoration: none;
  border-bottom: 1px solid var(--dt-accent);
  font-size: var(--dt-fs-base);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 14px;
}
.card {
  display: flex;
  flex-direction: column;
  text-decoration: none;
  color: inherit;
  overflow: hidden;
  transition:
    transform 0.22s cubic-bezier(0.22, 0.8, 0.3, 1),
    border-color 0.22s ease,
    box-shadow 0.22s ease;
}
.card:hover {
  transform: translateY(-2px);
  border-color: var(--dt-accent);
  box-shadow: 0 6px 20px var(--dt-shadow);
}
.thumb {
  display: block;
  aspect-ratio: 1;
  overflow: hidden;
}
.thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.4s cubic-bezier(0.22, 0.8, 0.3, 1);
}
.card:hover .thumb img {
  transform: scale(1.05);
}
.meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 9px 11px;
  border-top: 1px solid var(--dt-border);
}
.name {
  font-size: var(--dt-fs-base);
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.sub {
  font-size: var(--dt-fs-xs);
}
.star {
  color: var(--dt-accent);
  margin-left: 4px;
}
</style>
