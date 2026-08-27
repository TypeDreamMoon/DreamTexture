<script setup lang="ts">
import { computed } from 'vue'
import { NProgress, NButton } from 'naive-ui'
import { RouterLink } from 'vue-router'
import { fileURL } from '../api/client'
import { workflows } from '../store'
import type { Job } from '../api/types'

const props = defineProps<{ job: Job }>()
defineEmits<{ cancel: [string] }>()

const done = computed(() => props.job.status === 'succeeded')

// 产出的是材质还是图片，按任务自己的工作流判定，不能跟着生成台当前选的档走：
// 任务列表是不分档的，切到"材质"档时那些图片任务还在列表里，跟着档位走
// 会把它们全渲染成材质，点进去是个不存在的详情页。
const isImage = computed(
  () => workflows.value.find((w) => w.id === props.job.workflow_id)?.kind === 'image',
)
const link = computed(() =>
  isImage.value ? `/picture/${props.job.material_id}` : `/material/${props.job.material_id}`,
)
const thumb = computed(() =>
  isImage.value
    ? `/api/pictures/${props.job.material_id}/file`
    : fileURL(props.job.material_id, 'basecolor.png'),
)
const bad = computed(() => props.job.status === 'failed' || props.job.status === 'canceled')
const busy = computed(() => props.job.status === 'queued' || props.job.status === 'running')

const seed = computed(() => String(props.job.params?.seed ?? ''))

const tone = computed(() => {
  switch (props.job.status) {
    case 'succeeded':
      return { text: '完成', color: 'var(--dt-ok)' }
    case 'failed':
      return { text: '失败', color: 'var(--dt-danger)' }
    case 'canceled':
      return { text: '已取消', color: 'var(--dt-ink-faint)' }
    case 'running':
      return { text: props.job.stage || '生成中', color: 'var(--dt-accent)' }
    default:
      return { text: '排队中', color: 'var(--dt-ink-faint)' }
  }
})
</script>

<template>
  <div class="card dt-panel">
    <div class="thumb dt-swatch">
      <RouterLink v-if="done" :to="link" class="thumblink">
        <img :src="thumb" alt="" loading="lazy" />
      </RouterLink>
      <div v-else class="placeholder" :class="{ 'dt-sheen': busy }">
        <span :style="{ color: tone.color }" class="state">{{ tone.text }}</span>
        <NProgress
          v-if="busy"
          type="line"
          :percentage="Math.round(job.progress * 100)"
          :show-indicator="false"
          :height="4"
          class="bar"
        />
      </div>
    </div>

    <div class="meta">
      <div class="row">
        <span class="dt-mono seed">种子 {{ seed }}</span>
        <span v-if="busy" class="dt-mono pct">{{ Math.round(job.progress * 100) }}%</span>
      </div>

      <p v-if="bad && job.error" class="err" :title="job.error">{{ job.error }}</p>

      <div class="actions">
        <RouterLink v-if="done" :to="link">
          <NButton size="tiny" tertiary>{{ isImage ? '查看大图' : '查看通道' }}</NButton>
        </RouterLink>
        <NButton v-if="busy" size="tiny" tertiary @click="$emit('cancel', job.id)">取消</NButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition:
    transform 0.22s cubic-bezier(0.22, 0.8, 0.3, 1),
    border-color 0.22s ease,
    box-shadow 0.22s ease;
}
.card:hover {
  transform: translateY(-2px);
  border-color: var(--dt-border-strong);
  box-shadow: 0 6px 20px var(--dt-shadow);
}
.thumb {
  aspect-ratio: 1;
  position: relative;
  overflow: hidden;
}
.thumblink,
.thumblink img {
  display: block;
  width: 100%;
  height: 100%;
}
.thumblink img {
  object-fit: cover;
  transition: transform 0.4s cubic-bezier(0.22, 0.8, 0.3, 1);
}
.card:hover .thumblink img {
  transform: scale(1.04);
}
.placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: var(--dt-surface2);
}
.state {
  font-size: var(--dt-fs-base);
}
.bar {
  width: 60%;
}
.meta {
  padding: 9px 11px;
  border-top: 1px solid var(--dt-border);
  display: flex;
  flex-direction: column;
  gap: 7px;
}
.row {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.seed {
  font-size: var(--dt-fs-xs);
  color: var(--dt-ink-faint);
}
.pct {
  margin-left: auto;
  font-size: var(--dt-fs-xs);
  color: var(--dt-accent);
}
.err {
  margin: 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.5;
  color: var(--dt-danger);
  display: -webkit-box;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.actions {
  display: flex;
  gap: 6px;
}
</style>
