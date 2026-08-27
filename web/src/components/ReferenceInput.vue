<script setup lang="ts">
import { ref } from 'vue'
import { NButton, NSlider, useMessage } from 'naive-ui'

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

async function take(file: File | undefined) {
  if (!file) return
  uploading.value = true
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
  emit('update:modelValue', '')
  emit('update:denoise', 1)
}

function onDrop(e: DragEvent) {
  dragging.value = false
  take(e.dataTransfer?.files?.[0])
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
    <NButton v-if="modelValue" size="tiny" tertiary class="swap" @click="input?.click()">
      换一张
    </NButton>
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
}
</style>
