<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { NButton, NSelect, NInput, NTag, useDialog, useMessage } from 'naive-ui'
import { api } from '../api/client'
import { persisted } from '../persist'
import { health } from '../store'
import type { LogLine } from '../api/types'

const message = useMessage()
const dialog = useDialog()

const lines = ref<LogLine[]>([])
const last = ref(0)
const source = persisted<string>('console.source', '')
const filter = persisted<string>('console.filter', '')
const busy = ref('')
const paused = ref(false)
const box = ref<HTMLElement | null>(null)

// 缓冲上限要和后端的环形缓冲对齐：后端只留 4000 行，前端留更多也没有来源。
const MAX_LINES = 4000

const SOURCES = [
  { label: '全部', value: '' },
  { label: '后端', value: 'backend' },
  { label: 'ComfyUI', value: 'comfyui' },
]

const shown = computed(() => {
  const q = filter.value.trim().toLowerCase()
  return lines.value.filter(
    (l) =>
      (!source.value || l.source === source.value) &&
      (!q || l.text.toLowerCase().includes(q)),
  )
})

// 自动跟随只在用户已经贴着底部时生效——正翻历史的时候被拽到底部很烦人。
const follow = ref(true)
function onScroll() {
  const el = box.value
  if (!el) return
  follow.value = el.scrollHeight - el.scrollTop - el.clientHeight < 40
}
function toBottom() {
  const el = box.value
  if (el) el.scrollTop = el.scrollHeight
  follow.value = true
}

async function pull() {
  if (paused.value) return
  try {
    const r = await api.logs(last.value, 1000)
    if (r.lines.length) {
      // 后端重启会让 seq 从头开始，这时旧的要全丢掉，否则新旧混在一起、
      // 时间顺序也乱了。
      if (r.lines[0]!.seq <= last.value) lines.value = []
      lines.value.push(...r.lines)
      if (lines.value.length > MAX_LINES) {
        lines.value = lines.value.slice(-MAX_LINES)
      }
      last.value = r.last
      if (follow.value) nextTick(toBottom)
    } else if (r.last < last.value) {
      // 服务端的序号倒退了：后端重启过，重新全量拉一次。
      lines.value = []
      last.value = 0
    }
  } catch {
    /* 后端重启途中会断，下一轮自然恢复 */
  }
}

let timer: number | undefined
onMounted(async () => {
  await pull()
  nextTick(toBottom)
  timer = window.setInterval(pull, 1000)
})
onBeforeUnmount(() => clearInterval(timer))

watch(paused, (v) => {
  if (!v) pull()
})

const alive = computed(() => !!health.value?.alive)
const stopped = computed(() => !!health.value?.user_stopped)

async function act(what: 'start' | 'stop' | 'restart') {
  busy.value = what
  try {
    if (what === 'stop') {
      await api.comfyStop()
      message.success('ComfyUI 已停止')
    } else if (what === 'start') {
      message.info('正在启动，冷启动通常要一两分钟')
      await api.comfyStart()
      message.success('ComfyUI 已就绪')
    } else {
      message.info('正在重启…')
      await api.comfyRestart()
      message.success('ComfyUI 已重启')
    }
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    busy.value = ''
  }
}

function confirmStop() {
  dialog.warning({
    title: '停止 ComfyUI',
    content: '正在跑的任务会被打断，排队中的任务会一直等到它回来。确定停止？',
    positiveText: '停止',
    negativeText: '取消',
    onPositiveClick: () => act('stop'),
  })
}

function copyAll() {
  const text = shown.value.map((l) => `${time(l.at)} ${l.source} ${l.text}`).join('\n')
  // 沙箱里 clipboard 常被拦，失败就退回选中，让用户自己复制。
  navigator.clipboard?.writeText(text).then(
    () => message.success(`已复制 ${shown.value.length} 行`),
    () => {
      const el = box.value
      if (!el) return
      const r = document.createRange()
      r.selectNodeContents(el)
      const sel = window.getSelection()
      sel?.removeAllRanges()
      sel?.addRange(r)
      message.warning('浏览器不允许直接写剪贴板，已帮你全选，按 Ctrl+C 复制')
    },
  )
}

function time(at: string) {
  const d = new Date(at)
  return d.toLocaleTimeString('zh-CN', { hour12: false })
}
</script>

<template>
  <div class="dt-page dt-page-full page">
    <!-- 这一条同时充当页头。控制台的日志窗口要占满高度，再叠一个标准页头
         就要吃掉 60 多像素，而状态和启停按钮本来就在这儿，识别度够了。 -->
    <header class="bar dt-glass">
      <h1>控制台</h1>
      <span class="state">
        <i class="dot" :class="{ on: alive }" />
        <span>{{ alive ? '运行中' : stopped ? '已停止' : '未连接' }}</span>
      </span>
      <NTag v-if="health?.pid" size="tiny" :bordered="false" class="dt-mono">
        PID {{ health.pid }}
      </NTag>
      <span v-if="health?.comfyui_version" class="dt-faint dt-mono ver">
        ComfyUI {{ health.comfyui_version }}
      </span>

      <div class="acts">
        <NButton
          v-if="alive"
          size="small"
          :loading="busy === 'stop'"
          :disabled="!!busy"
          @click="confirmStop"
        >
          停止
        </NButton>
        <NButton
          v-else
          size="small"
          type="primary"
          :loading="busy === 'start'"
          :disabled="!!busy"
          @click="act('start')"
        >
          启动
        </NButton>
        <NButton
          size="small"
          tertiary
          :loading="busy === 'restart'"
          :disabled="!!busy || !alive"
          @click="act('restart')"
        >
          重启
        </NButton>
      </div>
    </header>

    <div class="tools">
      <NSelect v-model:value="source" :options="SOURCES" size="small" class="pick" />
      <NInput v-model:value="filter" size="small" placeholder="过滤（子串匹配）" clearable class="find" />
      <span class="count dt-faint dt-mono">{{ shown.length }} / {{ lines.length }} 行</span>
      <NButton size="tiny" tertiary @click="paused = !paused">
        {{ paused ? '继续' : '暂停' }}
      </NButton>
      <NButton size="tiny" tertiary @click="copyAll">复制</NButton>
      <NButton size="tiny" tertiary :disabled="follow" @click="toBottom">回到底部</NButton>
    </div>

    <div ref="box" class="log dt-mono" @scroll="onScroll">
      <p v-if="!shown.length" class="empty dt-faint">
        {{ lines.length ? '没有匹配的行' : '暂无输出' }}
      </p>
      <p
        v-for="l in shown"
        :key="l.seq"
        class="line"
        :class="[l.level.toLowerCase(), l.source]"
      >
        <span class="t">{{ time(l.at) }}</span>
        <span class="src">{{ l.source === 'comfyui' ? 'CU' : 'DT' }}</span>
        <span class="msg">{{ l.text }}</span>
      </p>
    </div>
  </div>
</template>

<style scoped>
/* 控制台不定宽：日志行又长又密，越宽越少折行。
   高度占满整个视口——顶栏改成左侧栏之后，上方不再有东西占高度。 */
.page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  gap: 10px;
}

.bar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 9px 14px;
}
h1 {
  margin: 0;
  font-size: var(--dt-fs-md);
  font-weight: 500;
  padding-right: 14px;
  border-right: 1px solid var(--dt-border);
}
.state {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--dt-fs-base);
}
.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--dt-danger);
  transition: background 0.3s ease;
}
.dot.on {
  background: var(--dt-ok);
  animation: dt-pulse 2.4s ease-in-out infinite;
}
.ver {
  font-size: var(--dt-fs-xs);
}
.acts {
  margin-left: auto;
  display: flex;
  gap: 8px;
}

.tools {
  display: flex;
  align-items: center;
  gap: 10px;
}
.pick {
  width: 110px;
}
.find {
  max-width: 280px;
}
.count {
  margin-left: auto;
  font-size: var(--dt-fs-sm);
}

.log {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 10px 12px;
  background: var(--dt-surface2);
  border: 1px solid var(--dt-border);
  border-radius: var(--dt-radius-card);
  font-size: var(--dt-fs-sm);
  line-height: 1.65;
  /* 长路径和堆栈不折行反而更好读，横向滚就行 */
  white-space: pre;
  overflow-x: auto;
}
.empty {
  margin: 0;
  padding: 40px 0;
  text-align: center;
}
.line {
  margin: 0;
  display: flex;
  gap: 10px;
}
.t {
  color: var(--dt-ink-faint);
  flex: none;
}
.src {
  flex: none;
  width: 20px;
  color: var(--dt-ink-faint);
  font-size: var(--dt-fs-2xs);
}
.line.comfyui .src {
  color: var(--dt-accent);
}
.msg {
  color: var(--dt-ink-muted);
}
.line.warn .msg {
  color: var(--dt-warn);
}
.line.error .msg {
  color: var(--dt-danger);
}
.line.debug .msg {
  color: var(--dt-ink-faint);
}
</style>
