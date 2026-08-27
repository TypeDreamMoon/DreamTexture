<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NSpin, NTag, useMessage } from 'naive-ui'
import TileViewer from '../components/TileViewer.vue'
import MaterialPreview3D from '../components/MaterialPreview3D.vue'
import { api, fileURL } from '../api/client'
import { persistedEnum } from '../persist'
import { CHANNEL_LABEL, CHANNEL_ORDER } from '../api/types'
import type { Manifest, MaterialIndex } from '../api/types'

const props = defineProps<{ id: string }>()
const router = useRouter()
const message = useMessage()

const manifest = ref<Manifest | null>(null)
const index = ref<MaterialIndex | null>(null)
const loading = ref(true)
const error = ref('')
const channel = ref('basecolor')

// 默认进 3D：单看通道图判断不出材质好坏，打上光才看得出来。
const mode = persistedEnum<'3d' | 'channels'>('material.mode', '3d', ['3d', 'channels'])
const MODES = [
  { key: '3d' as const, label: '材质预览' },
  { key: 'channels' as const, label: '通道检视' },
]

watch(
  () => props.id,
  async (id) => {
    loading.value = true
    error.value = ''
    try {
      const r = await api.material(id)
      manifest.value = r.manifest
      index.value = r.index
      channel.value = r.manifest.maps['basecolor'] ? 'basecolor' : Object.keys(r.manifest.maps)[0]!
    } catch (e) {
      error.value = String((e as Error).message)
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

// 按约定顺序展示，manifest 里没有的通道自然跳过。
const channels = computed(() => {
  const m = manifest.value?.maps ?? {}
  const known = CHANNEL_ORDER.filter((c) => m[c])
  const extra = Object.keys(m).filter((c) => !CHANNEL_ORDER.includes(c as never))
  return [...known, ...extra]
})

// 3D 预览用到了哪些通道、各自起什么作用。没有的通道也列出来并标灰——
// 让人知道"这套材质没有金属度"，而不是以为界面漏了。
const SLOTS: { channel: string; label: string; role: string }[] = [
  { channel: 'basecolor', label: '基础色', role: '表面颜色' },
  { channel: 'normal', label: '法线', role: '微表面凹凸' },
  { channel: 'roughness', label: '粗糙度', role: '高光扩散' },
  { channel: 'metallic', label: '金属度', role: '导体 / 绝缘体' },
  { channel: 'ao', label: '环境光遮蔽', role: '缝隙暗部' },
  { channel: 'height', label: '高度', role: '几何置换' },
]

const slots = computed(() =>
  SLOTS.map((s) => ({ ...s, present: !!manifest.value?.maps[s.channel] })),
)

const normalNote = computed(() => {
  const y = manifest.value?.maps['normal']?.y
  if (!y) return ''
  return y.toLowerCase() === 'directx'
    ? '法线为 DirectX(Y−)，UE 导入无需翻 G；此处预览已按该方向渲染'
    : `法线为 ${y}`
})

const currentMap = computed(() => manifest.value?.maps[channel.value])
const currentSrc = computed(() =>
  manifest.value && currentMap.value ? fileURL(manifest.value.id, currentMap.value.file) : '',
)

const favorite = ref(false)
watch(index, (v) => (favorite.value = v?.favorite ?? false))

async function toggleFavorite() {
  if (!manifest.value) return
  const next = !favorite.value
  try {
    await api.setFavorite(manifest.value.id, next)
    favorite.value = next
  } catch (e) {
    message.error(String((e as Error).message))
  }
}

// 参数回填：带着这套材质的全部参数回生成台，改一改再来一张。
function regenerate() {
  if (!manifest.value) return
  sessionStorage.setItem(
    'dt.refill',
    JSON.stringify({
      workflow_id: manifest.value.workflow.id,
      params: manifest.value.params ?? {},
    }),
  )
  router.push('/generate')
}

const created = computed(() =>
  manifest.value ? new Date(manifest.value.created_at).toLocaleString('zh-CN') : '',
)

function kb(n: number) {
  return n >= 1024 * 1024 ? `${(n / 1024 / 1024).toFixed(1)} MB` : `${Math.round(n / 1024)} KB`
}
</script>

<template>
  <div class="dt-page dt-page-wide">
    <div v-if="loading" class="center"><NSpin size="large" /></div>
    <div v-else-if="error" class="center err">{{ error }}</div>

    <template v-else-if="manifest">
      <header class="head">
        <div>
          <h1>{{ manifest.name }}</h1>
          <div class="tags">
            <NTag size="small" :bordered="false">{{ manifest.style === 'realistic' ? '写实' : '风格化' }}</NTag>
            <NTag size="small" :bordered="false">{{ manifest.resolution }}²</NTag>
            <NTag v-if="manifest.tileable" size="small" :bordered="false">无缝</NTag>
            <NTag
              v-if="manifest.license_flags && !manifest.license_flags.commercial_use"
              size="small"
              type="warning"
              :bordered="false"
            >
              不可商用
            </NTag>
          </div>
        </div>
        <div class="acts">
          <NButton size="small" :type="favorite ? 'primary' : 'default'" tertiary @click="toggleFavorite">
            {{ favorite ? '已收藏' : '收藏' }}
          </NButton>
          <NButton size="small" @click="regenerate">用这套参数再来一张</NButton>
        </div>
      </header>

      <div class="body">
        <!-- 左：预览 -->
        <section class="main">
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

          <MaterialPreview3D v-if="mode === '3d'" :manifest="manifest" />

          <template v-else>
            <div class="channels">
              <button
                v-for="c in channels"
                :key="c"
                class="chan"
                :class="{ on: channel === c }"
                @click="channel = c"
              >
                <span class="chan-img dt-swatch">
                  <img :src="fileURL(manifest.id, manifest.maps[c]!.file)" alt="" loading="lazy" />
                </span>
                <span class="chan-name">{{ CHANNEL_LABEL[c] ?? c }}</span>
              </button>
            </div>

            <TileViewer :src="currentSrc" :alt="CHANNEL_LABEL[channel] ?? channel" />
          </template>
        </section>

        <!-- 右：元信息 -->
        <aside class="side">
          <div v-if="mode === '3d'" class="block dt-panel">
            <p class="dt-label">材质构成</p>
            <ul class="slots">
              <li v-for="s in slots" :key="s.channel" :class="{ off: !s.present }">
                <span class="s-dot" />
                <span class="s-name">{{ s.label }}</span>
                <span class="s-role dt-faint">{{ s.role }}</span>
              </li>
            </ul>
            <p v-if="normalNote" class="s-note dt-faint">{{ normalNote }}</p>
          </div>

          <div v-else class="block dt-panel">
            <p class="dt-label">当前通道</p>
            <dl v-if="currentMap">
              <dt>文件</dt><dd class="dt-mono">{{ currentMap.file }}</dd>
              <dt>色彩空间</dt><dd class="dt-mono">{{ currentMap.colorspace }}</dd>
              <template v-if="currentMap.y">
                <dt>法线方向</dt>
                <dd class="dt-mono">{{ currentMap.y }}<span class="dt-faint"> · UE 无需翻 G</span></dd>
              </template>
              <template v-if="currentMap.packing">
                <dt>通道打包</dt><dd class="dt-mono">{{ currentMap.packing }}</dd>
              </template>
              <dt>尺寸</dt><dd class="dt-mono">{{ currentMap.width }}×{{ currentMap.height }}</dd>
              <dt>大小</dt><dd class="dt-mono">{{ kb(currentMap.bytes) }}</dd>
            </dl>
            <a v-if="currentSrc" :href="currentSrc" :download="currentMap?.file" class="dl">
              下载这张
            </a>
          </div>

          <div class="block dt-panel">
            <p class="dt-label">生成参数</p>
            <dl>
              <dt>提示词</dt><dd class="wrap">{{ manifest.prompt }}</dd>
              <template v-if="manifest.negative">
                <dt>负面词</dt><dd class="wrap dt-faint">{{ manifest.negative }}</dd>
              </template>
              <dt>{{ manifest.source ? '本地重整种子' : '种子' }}</dt>
              <dd class="dt-mono">{{ manifest.seed }}</dd>
              <dt>工作流</dt><dd class="dt-mono">{{ manifest.workflow.id }} v{{ manifest.workflow.version }}</dd>
              <dt>生成时间</dt><dd>{{ created }}</dd>
            </dl>
            <p v-if="manifest.source" class="license dt-faint">
              底图来自云端模型，那一步不支持种子——同样的参数再跑一次不会得到同一张图。
            </p>
          </div>

          <div v-if="manifest.source" class="block dt-panel">
            <p class="dt-label">云端底图</p>
            <dl>
              <dt>模型</dt>
              <dd class="dt-mono">{{ manifest.source.provider }} / {{ manifest.source.model }}</dd>
              <template v-if="manifest.source.quality">
                <dt>画质档</dt><dd class="dt-mono">{{ manifest.source.quality }}</dd>
              </template>
              <template v-if="manifest.source.size">
                <dt>出图尺寸</dt><dd class="dt-mono">{{ manifest.source.size }}</dd>
              </template>
              <template v-if="manifest.source.cost_usd">
                <dt>实际花费</dt>
                <dd class="dt-mono">
                  ${{ manifest.source.cost_usd.toFixed(4) }}
                  <span class="dt-faint">
                    ({{ manifest.source.input_tokens }}+{{ manifest.source.output_tokens }} tok)
                  </span>
                </dd>
              </template>
              <template v-if="manifest.source.elapsed_ms">
                <dt>云端耗时</dt>
                <dd class="dt-mono">{{ (manifest.source.elapsed_ms / 1000).toFixed(1) }}s</dd>
              </template>
              <dt>亮度场</dt>
              <dd class="dt-mono">
                <template v-if="manifest.source.flattened">
                  已压平 {{ manifest.source.falloff_before?.toFixed(3) }} →
                  {{ manifest.source.falloff_after?.toFixed(3) }}
                </template>
                <template v-else>本来就均匀，未处理</template>
              </dd>
            </dl>
            <p v-if="manifest.source.revised_prompt" class="license dt-faint">
              服务端改写后的提示词：{{ manifest.source.revised_prompt }}
            </p>
          </div>

          <div class="block dt-panel">
            <p class="dt-label">生成器</p>
            <dl>
              <template v-if="manifest.generator.checkpoint">
                <dt>底模</dt><dd class="dt-mono wrap">{{ manifest.generator.checkpoint }}</dd>
              </template>
              <template v-if="manifest.generator.pbr_estimator">
                <dt>PBR 估计</dt><dd class="dt-mono wrap">{{ manifest.generator.pbr_estimator }}</dd>
              </template>
              <template v-if="manifest.generator.comfyui">
                <dt>ComfyUI</dt><dd class="dt-mono">{{ manifest.generator.comfyui }}</dd>
              </template>
            </dl>
            <p v-if="manifest.license_flags?.reason" class="license dt-faint">
              {{ manifest.license_flags.reason }}
            </p>
          </div>
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped>
.center {
  display: grid;
  place-items: center;
  height: 60vh;
}
.err {
  color: var(--dt-danger);
}

.head {
  display: flex;
  align-items: flex-start;
  gap: 20px;
  margin-bottom: 18px;
  flex-wrap: wrap;
}
h1 {
  margin: 0 0 8px;
  font-size: var(--dt-fs-lg);
  font-weight: 500;
  line-height: 1.3;
}
.tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.acts {
  display: flex;
  gap: 8px;
  margin-left: auto;
}

.body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 18px;
  align-items: start;
}

.modes {
  display: flex;
  gap: 4px;
  margin-bottom: 14px;
  padding: 3px;
  border-radius: var(--dt-radius);
  background: var(--dt-surface2);
  width: fit-content;
}
.mode {
  font: inherit;
  font-size: var(--dt-fs-base);
  padding: 5px 16px;
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

.slots {
  list-style: none;
  margin: 10px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 7px;
  font-size: var(--dt-fs-base);
}
.slots li {
  display: flex;
  align-items: center;
  gap: 8px;
}
.slots li.off {
  opacity: 0.4;
}
.s-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--dt-accent);
  flex: none;
}
.slots li.off .s-dot {
  background: var(--dt-ink-faint);
}
.s-role {
  margin-left: auto;
  font-size: var(--dt-fs-xs);
}
.s-note {
  margin: 12px 0 0;
  font-size: var(--dt-fs-xs);
  line-height: 1.6;
}

.channels {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 14px;
}
.chan {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 0;
  background: transparent;
  border: none;
  cursor: pointer;
}
.chan-img {
  display: block;
  width: 60px;
  height: 60px;
  border: 1px solid var(--dt-border);
}
.chan-img img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.chan.on .chan-img {
  border-color: var(--dt-accent);
  box-shadow: inset 0 0 0 1px var(--dt-accent);
}
.chan-name {
  font-size: var(--dt-fs-xs);
  color: var(--dt-ink-faint);
  text-align: center;
}
.chan.on .chan-name {
  color: var(--dt-accent);
}

.side {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.block {
  padding: 14px 16px;
}
dl {
  display: grid;
  grid-template-columns: 70px 1fr;
  gap: 7px 12px;
  margin: 10px 0 0;
  font-size: var(--dt-fs-base);
  line-height: 1.6;
}
dt {
  color: var(--dt-ink-faint);
}
dd {
  margin: 0;
  word-break: break-word;
}
.wrap {
  line-height: 1.65;
}
.dl {
  display: inline-block;
  margin-top: 12px;
  font-size: var(--dt-fs-sm);
  color: var(--dt-accent);
  text-decoration: none;
  border-bottom: 1px solid transparent;
}
.dl:hover {
  border-bottom-color: var(--dt-accent);
}
.license {
  margin: 10px 0 0;
  font-size: var(--dt-fs-xs);
  line-height: 1.6;
}

@media (max-width: 1000px) {
  .body {
    grid-template-columns: 1fr;
  }
}
</style>
