<script setup lang="ts">
import { h, onMounted, ref } from 'vue'
import { NButton, NSlider, NModal, NInput, useDialog, useMessage } from 'naive-ui'
import { api } from '../api/client'
import type { RefItem } from '../api/types'

// 参考图入口。留空即纯文生图；给了图就变成在它基础上重绘。
//
// showDenoise 为 false 时不显示重绘幅度：云端底图的管线里，参考图是发给
// 云端做图生图的，重绘强度由那边自己定，本地没有对应的旋钮。
const props = withDefaults(
  defineProps<{ modelValue: string; denoise: number; showDenoise?: boolean; note?: string }>(),
  { showDenoise: true, note: '' },
)
const emit = defineEmits<{
  'update:modelValue': [string]
  'update:denoise': [number]
}>()

const preview = ref('')
const filename = ref('')
const uploading = ref(false)
const dragging = ref(false)
const input = ref<HTMLInputElement | null>(null)
const message = useMessage()

// 留着刚拖进来的那个 File，"存进库"要用它——走到那一步时原始文件已经
// 交给后端了，手里只剩一个文件名。
const lastFile = ref<File | null>(null)

async function take(file: File | undefined) {
  if (!file) return
  uploading.value = true
  lastFile.value = file
  try {
    const form = new FormData()
    form.append('image', file)
    const resp = await fetch('/api/uploads', { method: 'POST', body: form })
    const body = await resp.json()
    if (!resp.ok) throw new Error(body?.error ?? `HTTP ${resp.status}`)
    emit('update:modelValue', body.name)
    filename.value = body.original ?? file.name
    // 本地预览，不用再从 ComfyUI 拉一次。
    URL.revokeObjectURL(preview.value)
    preview.value = URL.createObjectURL(file)
    // 有了参考图，1.0 的重绘幅度等于把它完全盖掉，先给一个有意义的默认值。
    if (props.denoise >= 0.99) emit('update:denoise', 0.6)
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    uploading.value = false
  }
}

function clear() {
  URL.revokeObjectURL(preview.value)
  preview.value = ''
  filename.value = ''
  lastFile.value = null
  emit('update:modelValue', '')
  emit('update:denoise', 1)
}

function onDrop(e: DragEvent) {
  dragging.value = false
  take(e.dataTransfer?.files?.[0])
}

// ── 参考图库 ──────────────────────────────────────────────────
//
// 拖进来的图用完就散，同一张要反复用就得反复传。库里的能留着，
// 也能把满意的产物提升上来当下一轮的参考。
const dialog = useDialog()
const libOpen = ref(false)
const refs = ref<RefItem[]>([])
const libLoading = ref(false)

async function loadRefs() {
  libLoading.value = true
  try {
    refs.value = (await api.refs()).refs
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    libLoading.value = false
  }
}
onMounted(loadRefs)

function openLib() {
  libOpen.value = true
  loadRefs()
}

// 选中库里的一张：每次都重新推进 ComfyUI 的 input 目录。
// 记住上次的文件名会在 ComfyUI 被清空或换实例后失效，而那个失败要等到
// 提交工作流才出现，报错还看不出跟参考图库有关。
async function pick(r: RefItem) {
  try {
    const res = await api.useRef(r.id)
    emit('update:modelValue', res.name)
    filename.value = r.name
    URL.revokeObjectURL(preview.value)
    preview.value = `/api/refs/${r.id}/file`
    if (props.denoise >= 0.99) emit('update:denoise', 0.6)
    libOpen.value = false
  } catch (e) {
    message.error(String((e as Error).message))
  }
}

function renameRef(r: RefItem) {
  const name = ref(r.name)
  dialog.create({
    title: '重命名',
    content: () => h(NInput, { value: name.value, 'onUpdate:value': (v: string) => (name.value = v) }),
    positiveText: '保存',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.renameRef(r.id, name.value)
        await loadRefs()
      } catch (e) {
        message.error(String((e as Error).message))
      }
    },
  })
}

function removeRef(r: RefItem) {
  dialog.warning({
    title: '删除参考图',
    content: `确定从库里删掉「${r.name}」？`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.deleteRef(r.id)
        await loadRefs()
      } catch (e) {
        message.error(String((e as Error).message))
      }
    },
  })
}

// 把当前这张存进库里，下次不用再传。
const savingToLib = ref(false)
async function saveToLib(file: File) {
  savingToLib.value = true
  try {
    const form = new FormData()
    form.append('image', file)
    form.append('name', file.name.replace(/\.[^.]+$/, ''))
    const resp = await fetch('/api/refs', { method: 'POST', body: form })
    const body = await resp.json()
    if (!resp.ok) throw new Error(body?.error ?? `HTTP ${resp.status}`)
    await loadRefs()
    message.success('已存进参考图库')
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    savingToLib.value = false
  }
}

function kb(n: number) {
  return n >= 1 << 20 ? `${(n / (1 << 20)).toFixed(1)} MB` : `${Math.round(n / 1024)} KB`
}

// 重绘幅度的语义对新手不直观，用文字说清楚当前档位意味着什么。
function denoiseHint(v: number): string {
  if (v <= 0.35) return '几乎照抄原图，只做细节翻新'
  if (v <= 0.55) return '保留原图结构，换材质表现'
  if (v <= 0.75) return '借用大色调与构成，重新生成'
  return '基本无视原图'
}
</script>

<template>
  <div class="ref">
    <div class="head">
      <span class="name">参考图</span>
      <span class="opt dt-faint">可选</span>
      <button class="lib" @click="openLib">
        图库<span v-if="refs.length" class="n dt-mono">{{ refs.length }}</span>
      </button>
      <button v-if="modelValue" class="clear" @click="clear">移除</button>
    </div>

    <div
      v-if="!modelValue"
      class="drop"
      :class="{ on: dragging, busy: uploading }"
      @dragover.prevent="dragging = true"
      @dragleave="dragging = false"
      @drop.prevent="onDrop"
      @click="input?.click()"
    >
      <p class="d1">{{ uploading ? '上传中…' : '拖入图片，或点击选择' }}</p>
      <p class="d2 dt-faint">{{ note || '留空则为纯文生图' }}</p>
    </div>

    <div v-else class="picked">
      <span class="thumb dt-swatch">
        <img :src="preview" alt="" />
      </span>
      <div class="info">
        <p class="fn dt-mono" :title="filename">{{ filename }}</p>
        <template v-if="showDenoise">
          <p class="hint dt-faint">{{ denoiseHint(denoise) }}</p>
          <div class="slider">
            <NSlider
              :value="denoise"
              :min="0.1"
              :max="1"
              :step="0.05"
              :tooltip="false"
              @update:value="emit('update:denoise', $event as number)"
            />
            <span class="val dt-mono">{{ denoise.toFixed(2) }}</span>
          </div>
          <p class="lab dt-label">重绘幅度</p>
        </template>
        <p v-else-if="note" class="hint dt-faint">{{ note }}</p>
      </div>
    </div>

    <input
      ref="input"
      type="file"
      accept="image/png,image/jpeg,image/webp,image/bmp"
      hidden
      @change="take(($event.target as HTMLInputElement).files?.[0])"
    />
    <div v-if="modelValue" class="swap">
      <NButton size="tiny" tertiary @click="input?.click()">换一张</NButton>
      <NButton
        v-if="lastFile"
        size="tiny"
        tertiary
        :loading="savingToLib"
        @click="saveToLib(lastFile)"
      >
        存进图库
      </NButton>
    </div>

    <NModal
      v-model:show="libOpen"
      preset="card"
      title="参考图库"
      style="max-width: 760px"
      :bordered="false"
    >
      <p class="ltip dt-faint">
        库里的图会一直留着，可以反复用。在图片详情页点「存为参考图」也能把生成结果放进来。
      </p>
      <div v-if="libLoading" class="lcenter dt-faint">加载中…</div>
      <p v-else-if="!refs.length" class="lcenter dt-faint">
        图库还是空的。拖一张进来之后点「存进图库」，或者去图片详情页把生成结果存进来。
      </p>
      <div v-else class="lgrid">
        <div v-for="r in refs" :key="r.id" class="lcard dt-panel">
          <button class="lthumb dt-swatch" @click="pick(r)" :title="`用「${r.name}」`">
            <img :src="`/api/refs/${r.id}/file`" :alt="r.name" loading="lazy" />
          </button>
          <p class="lname" :title="r.name">{{ r.name }}</p>
          <p class="lmeta dt-faint dt-mono">{{ r.width }}×{{ r.height }} · {{ kb(r.bytes) }}</p>
          <div class="lacts">
            <button class="lbtn" @click="renameRef(r)">改名</button>
            <button class="lbtn del" @click="removeRef(r)">删除</button>
          </div>
        </div>
      </div>
    </NModal>
  </div>
</template>

<style scoped>
.ref {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.head {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.name {
  font-size: var(--dt-fs-base);
  font-weight: 500;
}
.opt {
  font-size: var(--dt-fs-xs);
}
.clear {
  margin-left: auto;
  font: inherit;
  font-size: var(--dt-fs-xs);
  color: var(--dt-ink-faint);
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}
.clear:hover {
  color: var(--dt-danger);
}

.drop {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 20px 12px;
  border: 1px dashed var(--dt-border-strong);
  border-radius: var(--dt-radius-card);
  cursor: pointer;
  transition:
    border-color 0.2s ease,
    background 0.2s ease,
    transform 0.2s cubic-bezier(0.22, 0.8, 0.3, 1);
}
.drop:hover {
  border-color: var(--dt-accent);
  background: var(--dt-surface2);
}
.drop.on {
  border-color: var(--dt-accent);
  background: var(--dt-accent-soft);
  transform: scale(1.015);
}
.drop.busy {
  opacity: 0.6;
  pointer-events: none;
}
.d1 {
  margin: 0;
  font-size: var(--dt-fs-base);
}
.d2 {
  margin: 0;
  font-size: var(--dt-fs-xs);
}

.picked {
  display: flex;
  gap: 12px;
  animation: dt-rise 0.28s cubic-bezier(0.22, 0.8, 0.3, 1) both;
}
.thumb {
  flex: none;
  width: 76px;
  height: 76px;
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
.info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.fn {
  margin: 0;
  font-size: var(--dt-fs-xs);
  color: var(--dt-ink-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.hint {
  margin: 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.5;
}
.slider {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 2px;
}
.slider .n-slider {
  flex: 1;
}
.val {
  font-size: var(--dt-fs-xs);
  color: var(--dt-accent);
}
.lab {
  margin: 0;
  font-size: var(--dt-fs-2xs);
}
.swap {
  align-self: flex-start;
  display: flex;
  gap: 8px;
}

.lib {
  margin-left: auto;
  font: inherit;
  font-size: var(--dt-fs-xs);
  color: var(--dt-accent);
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
  gap: 5px;
}
.lib .n {
  padding: 0 5px;
  border-radius: 999px;
  background: var(--dt-accent-soft);
  font-size: var(--dt-fs-2xs);
  line-height: 1.6;
}
/* 有「图库」之后「移除」不再靠 margin-left:auto 顶到右边 */
.lib + .clear {
  margin-left: 0;
}

.ltip {
  margin: 0 0 14px;
  font-size: var(--dt-fs-sm);
  line-height: 1.7;
}
.lcenter {
  padding: 40px 0;
  text-align: center;
  font-size: var(--dt-fs-sm);
  line-height: 1.7;
}
.lgrid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
  max-height: 56vh;
  overflow: auto;
}
.lcard {
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.lthumb {
  display: block;
  aspect-ratio: 1;
  border: none;
  padding: 0;
  border-radius: var(--dt-radius);
  overflow: hidden;
  cursor: pointer;
}
.lthumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.3s cubic-bezier(0.22, 0.8, 0.3, 1);
}
.lthumb:hover img {
  transform: scale(1.05);
}
.lname {
  margin: 4px 0 0;
  font-size: var(--dt-fs-sm);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.lmeta {
  margin: 0;
  font-size: var(--dt-fs-2xs);
}
.lacts {
  display: flex;
  gap: 10px;
  margin-top: 2px;
}
.lbtn {
  font: inherit;
  font-size: var(--dt-fs-2xs);
  color: var(--dt-ink-faint);
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
}
.lbtn:hover {
  color: var(--dt-ink);
}
.lbtn.del:hover {
  color: var(--dt-danger);
}
</style>
