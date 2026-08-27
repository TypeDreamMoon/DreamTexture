<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { NInput, NSelect, NSpin, NSwitch } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import { api, fileURL } from '../api/client'
import type { MaterialIndex } from '../api/types'

const items = ref<MaterialIndex[]>([])
const loading = ref(true)
const query = ref('')
const style = ref<string | null>(null)
const favOnly = ref(false)

const STYLES = [
  { label: '全部风格', value: '' },
  { label: '写实', value: 'realistic' },
  { label: '风格化', value: 'stylized' },
]

async function load() {
  loading.value = true
  try {
    const r = await api.materials({
      q: query.value.trim() || undefined,
      style: style.value || undefined,
      fav: favOnly.value,
      limit: 120,
    })
    items.value = r.materials
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

onMounted(load)
</script>

<template>
  <div class="dt-page dt-page-wide">
    <PageHeader title="素材库" desc="生成过的材质套装都在这儿。点开可以三维预览、逐通道检视、把参数回填去再来一张。">
      <template #actions>
        <span class="count dt-faint dt-mono">{{ items.length }} 项</span>
      </template>
    </PageHeader>

    <header class="bar">
      <NInput
        v-model:value="query"
        placeholder="按名称或提示词搜索"
        clearable
        class="search"
      />
      <NSelect v-model:value="style" :options="STYLES" class="pick" />
      <label class="fav">
        <NSwitch v-model:value="favOnly" size="small" />
        <span class="dt-muted">只看收藏</span>
      </label>
    </header>

    <div v-if="loading && !items.length" class="center"><NSpin /></div>

    <div v-else-if="!items.length" class="empty dt-faint">
      <p>{{ query || favOnly || style ? '没有匹配的素材。' : '素材库还是空的。' }}</p>
      <RouterLink to="/generate" class="link">去生成台</RouterLink>
    </div>

    <div v-else class="grid">
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
