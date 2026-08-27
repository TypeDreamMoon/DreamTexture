<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NSpin, NTag, NAlert, useDialog, useMessage } from 'naive-ui'
import { api } from '../api/client'
import type { Picture, PictureMeta } from '../api/types'

const props = defineProps<{ id: string }>()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const rec = ref<Picture | null>(null)
const meta = ref<PictureMeta | null>(null)
const loading = ref(true)
const error = ref('')
const busy = ref('')

const src = computed(() => `/api/pictures/${props.id}/file`)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const r = await api.picture(props.id)
    rec.value = r.picture
    meta.value = r.meta
  } catch (e) {
    error.value = String((e as Error).message)
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function toggleFav() {
  if (!rec.value) return
  const next = !rec.value.favorite
  try {
    await api.favoritePicture(props.id, next)
    rec.value.favorite = next
  } catch (e) {
    message.error(String((e as Error).message))
  }
}

// 提升成参考图：这是图片库最有用的一条路——出了张满意的直接拿去当下一轮的
// 参考，不必先下载到本地再传回来。
async function toRef() {
  busy.value = 'ref'
  try {
    const r = await api.refFromPicture(props.id, rec.value?.name)
    message.success(`已加入参考图库：${r.ref.name}`)
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    busy.value = ''
  }
}

function confirmDelete() {
  dialog.warning({
    title: '删除图片',
    content: '文件和记录都会删掉，不可恢复。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.deletePicture(props.id)
        message.success('已删除')
        router.push('/library')
      } catch (e) {
        message.error(String((e as Error).message))
      }
    },
  })
}

// 回填参数去生成台再来一张。和素材详情用的是同一套 sessionStorage 约定。
function refill() {
  if (!meta.value) return
  const { seed: _seed, ...rest } = meta.value.params ?? {}
  sessionStorage.setItem(
    'dt.refill',
    JSON.stringify({ workflow_id: meta.value.workflow_id, params: rest }),
  )
  router.push('/generate')
}

const created = computed(() =>
  rec.value ? new Date(rec.value.created_at).toLocaleString('zh-CN') : '',
)
</script>

<template>
  <div class="dt-page dt-page-wide">
    <div v-if="loading" class="center"><NSpin /></div>
    <NAlert v-else-if="error" type="error" :bordered="false">{{ error }}</NAlert>

    <template v-else-if="rec && meta">
      <header class="head">
        <div class="htext">
          <h1>{{ rec.name }}</h1>
          <div class="tags">
            <NTag size="small" :bordered="false">{{ rec.width }}×{{ rec.height }}</NTag>
            <NTag v-if="meta.source" size="small" :bordered="false" type="warning">
              {{ meta.source.provider }} / {{ meta.source.model }}
            </NTag>
            <NTag v-else size="small" :bordered="false" type="success">本地</NTag>
            <NTag v-if="rec.cost_usd" size="small" :bordered="false" class="dt-mono">
              ${{ rec.cost_usd.toFixed(4) }}
            </NTag>
          </div>
        </div>
        <div class="hacts">
          <NButton size="small" :type="rec.favorite ? 'primary' : 'default'" @click="toggleFav">
            {{ rec.favorite ? '★ 已收藏' : '☆ 收藏' }}
          </NButton>
          <NButton size="small" :loading="busy === 'ref'" @click="toRef">存为参考图</NButton>
          <NButton size="small" @click="refill">用这套参数再来一张</NButton>
          <a :href="src" :download="`${rec.id}.png`"><NButton size="small" tertiary>下载</NButton></a>
          <NButton size="small" tertiary type="error" @click="confirmDelete">删除</NButton>
        </div>
      </header>

      <div class="body">
        <div class="viewer dt-swatch">
          <img :src="src" :alt="rec.name" />
        </div>

        <aside class="side">
          <div class="block dt-panel">
            <p class="dt-label">生成参数</p>
            <dl>
              <dt>提示词</dt><dd class="wrap">{{ meta.prompt }}</dd>
              <template v-if="meta.negative">
                <dt>负面词</dt><dd class="wrap dt-faint">{{ meta.negative }}</dd>
              </template>
              <template v-if="!meta.source">
                <dt>种子</dt><dd class="dt-mono">{{ meta.seed }}</dd>
              </template>
              <dt>工作流</dt><dd class="dt-mono">{{ meta.workflow_id }}</dd>
              <dt>生成时间</dt><dd>{{ created }}</dd>
            </dl>
            <p v-if="meta.source" class="note dt-faint">
              云端出图不支持种子，同样的参数再跑一次不会得到同一张。
            </p>
          </div>

          <div v-if="meta.source" class="block dt-panel">
            <p class="dt-label">云端</p>
            <dl>
              <dt>模型</dt>
              <dd class="dt-mono">{{ meta.source.provider }} / {{ meta.source.model }}</dd>
              <template v-if="meta.source.quality">
                <dt>画质档</dt><dd class="dt-mono">{{ meta.source.quality }}</dd>
              </template>
              <template v-if="meta.source.size">
                <dt>请求尺寸</dt><dd class="dt-mono">{{ meta.source.size }}</dd>
              </template>
              <template v-if="meta.source.cost_usd">
                <dt>实际花费</dt>
                <dd class="dt-mono">
                  ${{ meta.source.cost_usd.toFixed(4) }}
                  <span class="dt-faint">
                    ({{ meta.source.input_tokens }}+{{ meta.source.output_tokens }} tok)
                  </span>
                </dd>
              </template>
              <template v-if="meta.source.elapsed_ms">
                <dt>耗时</dt>
                <dd class="dt-mono">{{ (meta.source.elapsed_ms / 1000).toFixed(1) }}s</dd>
              </template>
            </dl>
            <p v-if="meta.source.revised_prompt" class="note dt-faint">
              服务端改写后：{{ meta.source.revised_prompt }}
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
  height: 40vh;
}
.head {
  display: flex;
  align-items: flex-start;
  gap: 18px;
  margin-bottom: 18px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--dt-border);
}
.htext {
  min-width: 0;
  flex: 1;
}
h1 {
  margin: 0 0 8px;
  font-size: var(--dt-fs-lg);
  font-weight: 500;
  line-height: 1.3;
}
.tags {
  display: flex;
  gap: 7px;
  flex-wrap: wrap;
}
.hacts {
  flex: none;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.hacts a {
  text-decoration: none;
}

.body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 330px;
  gap: 18px;
  align-items: start;
}
/* 图片按原比例显示、限高在视口内——单张图不像贴图那样需要平铺检查，
   一眼看全比看大更重要。 */
.viewer {
  display: grid;
  place-items: center;
  border-radius: var(--dt-radius-card);
  overflow: hidden;
  padding: 10px;
}
.viewer img {
  max-width: 100%;
  max-height: calc(100vh - 190px);
  object-fit: contain;
  display: block;
}

.side {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.block {
  padding: 13px 15px;
}
dl {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 6px 12px;
  margin: 9px 0 0;
  font-size: var(--dt-fs-sm);
}
dt {
  color: var(--dt-ink-faint);
  white-space: nowrap;
}
dd {
  margin: 0;
  min-width: 0;
}
.wrap {
  word-break: break-word;
  line-height: 1.65;
}
.note {
  margin: 10px 0 0;
  font-size: var(--dt-fs-xs);
  line-height: 1.65;
}

@media (max-width: 1000px) {
  .body {
    grid-template-columns: 1fr;
  }
}
</style>
