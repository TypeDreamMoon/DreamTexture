<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { NButton, NInput, NSelect, NSwitch, NTag, NAlert, useDialog, useMessage } from 'naive-ui'
import { RouterLink } from 'vue-router'
import DtIcon from './DtIcon.vue'
import { api } from '../api/client'
import type { DeployInfo, DeployOptions, DeployStatus } from '../api/types'

const message = useMessage()
const dialog = useDialog()
const emit = defineEmits<{ applied: [] }>()

const info = ref<DeployInfo | null>(null)
const status = ref<DeployStatus | null>(null)
const opt = ref<DeployOptions>({
  dir: '',
  py_version: '3.13',
  torch: 'cu130',
  mirror: true,
  model_base_path: '',
})
const open = ref(false)
const busy = ref('')

const TORCH = [
  { label: 'CUDA 13.0（RTX 40/50 系）', value: 'cu130' },
  { label: 'CUDA 12.8（较老的驱动）', value: 'cu128' },
  { label: 'CPU（没有独显时）', value: 'cpu' },
]

const running = computed(() => !!status.value?.running)
const done = computed(() => !!status.value && !status.value.running && !status.value.error && !!status.value.python)
const failed = computed(() => !!status.value?.error)

async function load() {
  try {
    const r = await api.deployInfo()
    info.value = r
    status.value = r.status
    // 只在用户还没动过表单时套用默认值，免得轮询把正在填的框冲掉。
    if (!opt.value.dir) opt.value = { ...r.defaults }
  } catch {
    /* 后端重启途中会断 */
  }
}

let timer: number | undefined
onMounted(async () => {
  await load()
  // 部署要跑十几分钟，进度靠轮询；不跑的时候降频，别白占。
  timer = window.setInterval(() => {
    if (running.value || !status.value) load()
  }, 1500)
})
onBeforeUnmount(() => clearInterval(timer))

function start() {
  dialog.info({
    title: '开始部署',
    content: () =>
      `将在 ${opt.value.dir} 下装一套独立的 ComfyUI。` +
      (opt.value.torch === 'cpu' ? '' : 'PyTorch 约需下载 2~3GB，') +
      '整个过程可能十几分钟。现有的 ComfyUI 不受影响，装完由你决定切不切过去。',
    positiveText: '开始',
    negativeText: '再想想',
    onPositiveClick: async () => {
      busy.value = 'start'
      try {
        const r = await api.deployStart(opt.value)
        status.value = r.status
        message.success('已开始，详细输出在控制台')
      } catch (e) {
        message.error(String((e as Error).message))
      } finally {
        busy.value = ''
      }
    },
  })
}

async function cancel() {
  try {
    await api.deployCancel()
    message.info('已请求中断；已经下好的东西会留着，重跑时自动跳过')
  } catch (e) {
    message.error(String((e as Error).message))
  }
}

async function apply() {
  busy.value = 'apply'
  try {
    const r = await api.deployApply()
    message.success(
      r.need_restart.length
        ? `已切到新环境。${r.need_restart.join('、')} 需要重启后端才生效`
        : '已切到新环境',
    )
    emit('applied')
    await load()
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    busy.value = ''
  }
}

const STATE_LABEL: Record<string, string> = {
  pending: '等待',
  running: '进行中',
  done: '完成',
  skipped: '跳过',
  failed: '失败',
}
const STATE_TYPE: Record<string, 'default' | 'info' | 'success' | 'warning' | 'error'> = {
  pending: 'default',
  running: 'info',
  done: 'success',
  skipped: 'default',
  failed: 'error',
}

function secs(ms?: number) {
  if (!ms) return ''
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`
}

// 已经切到某套环境时，把它认出来，免得用户重复部署。
const alreadyUsing = computed(
  () =>
    !!status.value?.python &&
    !!info.value?.current.python &&
    status.value.python === info.value.current.python,
)
</script>

<template>
  <div class="deploy dt-panel">
    <button class="head" @click="open = !open">
      <span class="ico"><DtIcon name="download" /></span>
      <div class="text">
        <p class="title">
          一键部署独立环境
          <NTag v-if="running" size="tiny" :bordered="false" type="info">进行中</NTag>
          <NTag v-else-if="failed" size="tiny" :bordered="false" type="error">上次失败</NTag>
          <NTag v-else-if="done" size="tiny" :bordered="false" type="success">已就绪</NTag>
        </p>
        <p class="desc dt-faint">
          在 DreamTexture 目录下装一套自带 Python 与 PyTorch 的 ComfyUI，
          从此不依赖外部整合包。已有的模型库直接复用，不用重下。
        </p>
      </div>
      <span class="chev" :class="{ open }">›</span>
    </button>

    <div v-if="open" class="body">
      <NAlert type="info" :bordered="false" class="tip">
        现有的 ComfyUI <b>不会被改动</b>。装完之后由你决定切不切过去，
        随时能在上面的路径设置里切回来。
      </NAlert>

      <div class="form">
        <label class="f">
          <span class="lb">目标目录</span>
          <NInput v-model:value="opt.dir" size="small" :disabled="running" />
        </label>
        <label class="f">
          <span class="lb">PyTorch</span>
          <NSelect v-model:value="opt.torch" :options="TORCH" size="small" :disabled="running" />
        </label>
        <label class="f">
          <span class="lb">复用模型库</span>
          <NInput
            v-model:value="opt.model_base_path"
            size="small"
            placeholder="留空则新环境用自己的 models/"
            :disabled="running"
          />
        </label>
        <label class="f row">
          <span class="lb">国内镜像</span>
          <NSwitch v-model:value="opt.mirror" :disabled="running" />
          <span class="hint dt-faint">PyPI 走清华、ComfyUI 源走 jihulab。PyTorch 只有官方源，不受影响</span>
        </label>
      </div>

      <div class="acts">
        <NButton
          v-if="!running"
          type="primary"
          size="small"
          :loading="busy === 'start'"
          :disabled="!opt.dir"
          @click="start"
        >
          {{ failed ? '重试' : done ? '重新部署' : '开始部署' }}
        </NButton>
        <NButton v-else size="small" @click="cancel">中断</NButton>

        <NButton
          v-if="done && !alreadyUsing"
          size="small"
          type="primary"
          ghost
          :loading="busy === 'apply'"
          @click="apply"
        >
          应用到配置
        </NButton>
        <NTag v-else-if="alreadyUsing" size="small" :bordered="false" type="success">
          当前就在用这套环境
        </NTag>

        <RouterLink to="/console" class="lnk">看详细输出 →</RouterLink>
      </div>

      <NAlert v-if="failed" type="error" :bordered="false" class="tip">
        {{ status!.error }}
      </NAlert>

      <ol v-if="status?.steps?.length" class="steps">
        <li v-for="s in status.steps" :key="s.key" class="step" :class="s.state">
          <NTag size="tiny" :bordered="false" :type="STATE_TYPE[s.state]">
            {{ STATE_LABEL[s.state] }}
          </NTag>
          <span class="st">{{ s.title }}</span>
          <span v-if="s.detail" class="sd dt-faint">{{ s.detail }}</span>
          <span class="sm dt-mono dt-faint">{{ secs(s.ms) }}</span>
        </li>
      </ol>
    </div>
  </div>
</template>

<style scoped>
.deploy {
  overflow: hidden;
}
.head {
  display: flex;
  align-items: center;
  gap: 13px;
  width: 100%;
  padding: 13px 16px;
  background: none;
  border: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.head:hover {
  background: var(--dt-surface2);
}
.ico {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--dt-accent-soft);
  color: var(--dt-accent);
  flex: none;
}
.text {
  min-width: 0;
  flex: 1;
}
.title {
  margin: 0;
  font-size: var(--dt-fs-md);
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 8px;
}
.desc {
  margin: 3px 0 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.6;
}
.chev {
  flex: none;
  font-size: 18px;
  color: var(--dt-ink-faint);
  transition: transform 0.2s ease;
}
.chev.open {
  transform: rotate(90deg);
}

.body {
  padding: 4px 16px 16px;
  border-top: 1px solid var(--dt-border);
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tip {
  font-size: var(--dt-fs-sm);
  line-height: 1.7;
}
.tip b {
  font-weight: 500;
}
.form {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 12px;
}
.f {
  display: grid;
  grid-template-columns: 92px 1fr;
  gap: 12px;
  align-items: center;
}
.f.row {
  grid-template-columns: 92px auto 1fr;
}
.lb {
  font-size: var(--dt-fs-base);
  color: var(--dt-ink-muted);
}
.hint {
  font-size: var(--dt-fs-xs);
}
.acts {
  display: flex;
  align-items: center;
  gap: 10px;
}
.lnk {
  margin-left: auto;
  font-size: var(--dt-fs-sm);
  color: var(--dt-accent);
  text-decoration: none;
}
.lnk:hover {
  text-decoration: underline;
}

.steps {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 5px;
  border-top: 1px solid var(--dt-border);
  padding-top: 12px;
}
.step {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: var(--dt-fs-base);
}
.step.pending {
  opacity: 0.5;
}
.step.running .st {
  color: var(--dt-accent);
}
.st {
  flex: none;
}
.sd {
  font-size: var(--dt-fs-sm);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sm {
  margin-left: auto;
  font-size: var(--dt-fs-xs);
  flex: none;
}
</style>
