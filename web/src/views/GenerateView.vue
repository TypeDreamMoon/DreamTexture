<script setup lang="ts">
import { computed, h, ref, watch } from 'vue'
import {
  NButton,
  NCollapse,
  NCollapseItem,
  NRadioGroup,
  NRadioButton,
  NSelect,
  useMessage,
} from 'naive-ui'
import type { SelectOption } from 'naive-ui'
import PageHeader from '../components/PageHeader.vue'
import ParamField from '../components/ParamField.vue'
import ReferenceInput from '../components/ReferenceInput.vue'
import SelfCheck from '../components/SelfCheck.vue'
import JobCard from '../components/JobCard.vue'
import { api } from '../api/client'
import { workflows, jobs, batchJobs, upsertJob, health } from '../store'
import type { Param, WorkflowMeta } from '../api/types'

const message = useMessage()

const selectedID = ref<string>('')
const variants = ref(4)
const values = ref<Record<string, unknown>>({})
const submitting = ref(false)
const currentBatch = ref<string>('')

// 材质 / 图片是两类完全不同的产物，分成两档而不是混在一个预设列表里：
// 混着的话用户得先认出哪个预设产出什么，而两者的参数、产物、去处都不一样。
const kind = ref<'material' | 'image'>('material')
const KINDS = [
  { label: '材质', value: 'material' as const },
  { label: '图片', value: 'image' as const },
]

const kindWorkflows = computed(() => workflows.value.filter((w) => w.kind === kind.value))

const current = computed<WorkflowMeta | undefined>(() =>
  workflows.value.find((w) => w.id === selectedID.value),
)

const STYLE_LABEL: Record<string, string> = {
  realistic: '写实',
  stylized: '手绘',
  image: '出图',
}

const workflowOptions = computed(() =>
  kindWorkflows.value.map((w) => ({
    label: w.name,
    value: w.id,
    style: STYLE_LABEL[w.style] ?? w.style,
    cloud: !!w.source,
  })),
)

// 换档时选中当前档里的第一个预设。
watch(kind, () => {
  const list = kindWorkflows.value
  if (list.length && !list.some((w) => w.id === selectedID.value)) {
    selectedID.value = list[0]!.id
  }
})

// 自己渲染下拉项，好在名字后面挂上风格和"底图在哪出"两个标签。
// 预设的名字本身已经带了这些信息，但一眼扫下来时标签比读句子快，
// 而"要不要花钱"这件事值得一眼看见。
//
// 类名用全局的 dt-opt-*：下拉菜单被 teleport 到 body 之外，组件的 scoped
// 样式（哪怕 :deep）都够不着它。
function renderWorkflow(opt: SelectOption) {
  return h('div', { class: 'dt-opt' }, [
    h('span', { class: 'dt-opt-name' }, String(opt.label)),
    h('span', { class: 'dt-opt-tag' }, String(opt.style ?? '')),
    h(
      'span',
      { class: opt.cloud ? 'dt-opt-tag dt-opt-tag-accent' : 'dt-opt-tag' },
      opt.cloud ? '云端底图' : '本地',
    ),
  ])
}

// 工作流列表到位后默认选第一个；换工作流要按新的声明重置参数。
watch(
  workflows,
  (list) => {
    if (selectedID.value || !list.length) return
    // 回填指定的工作流优先，否则用第一个。
    let want = ''
    try {
      const raw = sessionStorage.getItem('dt.refill')
      if (raw) want = (JSON.parse(raw) as { workflow_id: string }).workflow_id
    } catch {
      want = ''
    }
    const wf = list.find((w) => w.id === want) ?? list[0]!
    selectedID.value = wf.id
    // 回填来自图片详情时要连档一起切过去，否则选中的预设不在当前档里，
    // 下拉框会显示一个列表里根本没有的值。
    kind.value = wf.kind === 'image' ? 'image' : 'material'
  },
  { immediate: true },
)

// immediate 是必须的：本视图是懒加载的，chunk 到位时 workflows 往往已经拉好了，
// 上面那个 watcher 会在 setup 期间同步把 selectedID 设好，于是 current 在本监听
// 注册之前就已有值——不加 immediate 就再也不会触发，表单会一直是空的。
watch(
  current,
  (wf, prev) => {
    if (!wf) return
    const next: Record<string, unknown> = {}
    for (const p of [...wf.params, ...wf.advanced]) next[p.key] = p.default ?? defaultFor(p)

    // 换预设时保留用户自己改过的值：切风格不该把已经写好的提示词冲掉。
    // 没动过的参数仍然跟随新预设的默认值——两条管线的派生参数差别很大，
    // 把它们一起带过去反而会得到糟糕的结果。
    if (prev) {
      const prevDefaults = new Map(
        [...prev.params, ...prev.advanced].map((p) => [p.key, p.default ?? defaultFor(p)]),
      )
      for (const [k, v] of Object.entries(values.value)) {
        if (!(k in next)) continue
        if (!prevDefaults.has(k) || prevDefaults.get(k) === v) continue
        if (typeof v !== typeof next[k]) continue
        next[k] = v
      }
    }

    // 从素材详情页带回来的参数覆盖默认值，实现"用这套参数再来一张"。
    Object.assign(next, takeRefill(wf.id))
    values.value = next
  },
  { immediate: true },
)

/** 取出素材详情页存下的回填参数，只用一次。 */
function takeRefill(workflowID: string): Record<string, unknown> {
  const raw = sessionStorage.getItem('dt.refill')
  if (!raw) return {}
  sessionStorage.removeItem('dt.refill')
  try {
    const r = JSON.parse(raw) as { workflow_id: string; params: Record<string, unknown> }
    if (r.workflow_id !== workflowID) return {}
    // 种子不回填：想复现同一张图会显式指定，默认还是每次换新的。
    const { seed: _seed, ...rest } = r.params
    return rest
  } catch {
    return {}
  }
}

function defaultFor(p: Param): unknown {
  switch (p.type) {
    case 'bool':
      return false
    case 'int':
    case 'float':
      return p.min ?? 0
    case 'enum':
      return p.options?.[0] ?? ''
    default:
      return ''
  }
}

const promptParam = computed(() => current.value?.params.find((p) => p.key === 'prompt'))

// 参考图有专门的拖放组件，不走通用控件；提示词单独排在最上面；
// hidden 的参数是后端自己填的（例如云端底图落盘后的文件名），不该露出来。
const basicRest = computed(
  () =>
    current.value?.params.filter(
      (p) => p.key !== 'prompt' && p.type !== 'image' && !p.hidden,
    ) ?? [],
)

// 参考图那个组件带着"重绘幅度"一起显示，所以只有工作流真的把参考图注入节点图
// 时才用它。云端底图的管线里参考图是发给云端做图生图的，本地没有 denoise 这一说。
const referenceParam = computed(() =>
  current.value?.params.find((p) => p.type === 'image' && !p.hidden),
)
const hasReference = computed(() => !!referenceParam.value)
const localDenoise = computed(() =>
  current.value?.advanced.some((p) => p.key === 'denoise') ?? false,
)

// 重绘幅度已经跟着参考图一起显示了，高级区里不再重复一遍。
const advancedRest = computed(
  () =>
    current.value?.advanced.filter(
      (p) => !p.hidden && !(hasReference.value && localDenoise.value && p.key === 'denoise'),
    ) ?? [],
)

// 云端底图来源；为空表示这条管线全在本地跑。
const sourceProvider = computed(() => current.value?.source?.provider ?? '')
const isCloud = computed(() => !!sourceProvider.value)

const reference = computed({
  get: () => String(values.value['reference'] ?? ''),
  set: (v: string) => (values.value['reference'] = v),
})
const denoise = computed({
  get: () => Number(values.value['denoise'] ?? 1),
  set: (v: number) => (values.value['denoise'] = v),
})

// 提示词的前后缀由模板固定追加（例如 CHORD 需要的平光约束），
// 这里如实展示，让用户知道真正送进去的是什么。
const fullPrompt = computed(() => {
  const p = promptParam.value
  if (!p) return ''
  return `${p.prefix ?? ''}${String(values.value[p.key] ?? '')}${p.suffix ?? ''}`
})

const canSubmit = computed(
  () => !!current.value && !submitting.value && String(values.value['prompt'] ?? '').trim() !== '',
)

async function submit() {
  if (!current.value || !canSubmit.value) return
  submitting.value = true
  try {
    const created = await api.generate({
      workflow_id: current.value.id,
      params: { ...values.value },
      variants: variants.value,
      name: String(values.value['prompt'] ?? '').slice(0, 60),
    })
    created.forEach(upsertJob)
    currentBatch.value = created[0]?.batch_id ?? ''
    message.success(`已提交 ${created.length} 个变体`)
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    submitting.value = false
  }
}

async function cancel(id: string) {
  try {
    await api.cancelJob(id)
  } catch (e) {
    message.error(String((e as Error).message))
  }
}

// 当前批次优先展示；还没提交过就展示最近的一批。
const shown = computed(() => {
  if (currentBatch.value) return batchJobs(currentBatch.value)
  const latest = jobs.value[0]
  return latest?.batch_id ? batchJobs(latest.batch_id) : jobs.value.slice(0, 4)
})

const notReady = computed(() => !health.value?.ready)

// 云端底图是按量计费的，变体数直接乘上去。这里不给具体金额——
// 实际花费按 token 结算，随提示词长短、画质档、有没有参考图变化，
// 报一个精确到分的数字只会让人误以为那是账单。真实花费在素材详情里，
// 那是服务端回报的 usage 算出来的。
const cloudCostHint = computed(() => {
  if (!isCloud.value) return ''
  const q = String(values.value['api_quality'] ?? 'medium')
  const rough: Record<string, string> = {
    low: '约 $0.01',
    medium: '约 $0.05',
    high: '约 $0.13~0.22',
  }
  const per = rough[q]
  if (!per) return `将调用 ${variants.value} 次云端接口，按 token 计费`
  return `${per}/张 × ${variants.value} 张（1024² 参考值，实际以服务端回报为准）`
})

// high 档单张要两三分钟，比整条本地管线还慢——提前说清楚，别让人以为卡死了。
const cloudSlow = computed(
  () => isCloud.value && String(values.value['api_quality'] ?? '') === 'high',
)

const tileFix = computed(() => Number(values.value['tile_fix'] ?? 0))

// ── 提示词扩写 ────────────────────────────────────────────────
//
// 扩写完不直接盖掉输入框，先摆出来让用户看一眼再决定。模型有时会自作主张
// 加东西，直接替换会让人莫名其妙地丢掉自己写的要求。
const refining = ref(false)
const refined = ref<{ before: string; after: string; model: string } | null>(null)

async function refine() {
  const p = String(values.value['prompt'] ?? '').trim()
  if (!p) {
    message.warning('先写点什么再让它扩写')
    return
  }
  refining.value = true
  try {
    // 材质要带上正交平光可平铺那套硬约束，普通出图不带——
    // 硬塞平光约束等于把画面限死。
    const r = await api.refinePrompt(p, kind.value === 'material' ? 'texture' : 'image')
    refined.value = { before: p, after: r.prompt, model: r.model }
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    refining.value = false
  }
}

function acceptRefined() {
  if (refined.value) values.value['prompt'] = refined.value.after
  refined.value = null
}
</script>

<template>
  <div class="dt-page dt-page-wide page">
    <PageHeader
      class="span"
      title="生成台"
      desc="选一个风格预设、写句提示词，出一整套 PBR 通道。一次多出几张变体再挑，比反复重来快。"
    />

    <SelfCheck class="span" />

    <!-- 左：参数 -->
    <section class="panel dt-glass">
      <div class="pad">
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

        <p class="dt-label">{{ kind === 'material' ? '风格预设' : '出图方式' }}</p>
        <NSelect
          v-model:value="selectedID"
          :options="workflowOptions"
          :render-label="renderWorkflow"
          class="styles"
        />
        <p v-if="current" class="desc dt-muted">{{ current.description }}</p>
        <p v-if="current?.license_notice && !current.license_notice.commercial" class="notice">
          {{ current.license_notice.component }} 为研究用途许可，产出不可商用
        </p>
      </div>

      <div v-if="current" class="pad fields">
        <ParamField
          v-if="promptParam"
          :param="promptParam"
          v-model="values[promptParam.key]"
        />

        <div v-if="promptParam" class="refine-row">
          <NButton size="tiny" tertiary :loading="refining" @click="refine">
            让模型扩写
          </NButton>
          <span class="tiny dt-faint">
            {{ kind === 'material' ? '会带上正交平光、可平铺这些硬约束' : '按画面来扩写，不加材质约束' }}
          </span>
        </div>

        <div v-if="refined" class="refined dt-panel">
          <p class="rtitle">
            扩写结果
            <span class="dt-faint dt-mono">{{ refined.model }}</span>
          </p>
          <p class="rtext">{{ refined.after }}</p>
          <div class="racts">
            <NButton size="tiny" type="primary" @click="acceptRefined">用它</NButton>
            <NButton size="tiny" tertiary @click="refined = null">不用</NButton>
          </div>
        </div>

        <p v-if="promptParam && (promptParam.prefix || promptParam.suffix)" class="expanded dt-mono">
          {{ fullPrompt }}
        </p>

        <ReferenceInput
          v-if="hasReference"
          v-model="reference"
          v-model:denoise="denoise"
          :show-denoise="localDenoise"
          :note="isCloud ? '发给云端做图生图；本地不参与' : ''"
        />

        <ParamField
          v-for="p in basicRest"
          :key="p.key"
          :param="p"
          :provider="sourceProvider"
          v-model="values[p.key]"
        />

        <p v-if="isCloud && !tileFix" class="warn-line">
          无缝重整为 0 时直接分解云端底图，<b>产物不保证可平铺</b>——
          三维预览里会看到接缝。好处是不经过本地 SDXL，不用下底模。
        </p>

        <div class="field">
          <div class="head"><span class="name">变体数</span></div>
          <NRadioGroup v-model:value="variants" size="small">
            <NRadioButton v-for="n in [1, 2, 4, 6]" :key="n" :value="n">{{ n }}</NRadioButton>
          </NRadioGroup>
          <p class="tiny dt-faint">一次多出几张再挑，比反复重来快</p>
        </div>
      </div>

      <div v-if="current && current.advanced.length" class="advanced">
        <NCollapse>
          <NCollapseItem title="高级参数" name="adv">
            <div class="fields">
              <ParamField
                v-for="p in advancedRest"
                :key="p.key"
                :param="p"
                :provider="sourceProvider"
                v-model="values[p.key]"
              />
            </div>
          </NCollapseItem>
        </NCollapse>
      </div>

      <div class="pad submit">
        <p v-if="cloudCostHint" class="cost dt-mono">{{ cloudCostHint }}</p>
        <NButton
          type="primary"
          block
          :disabled="!canSubmit"
          :loading="submitting"
          @click="submit"
        >
          生成
        </NButton>
        <p v-if="cloudSlow" class="tiny dt-faint">
          high 档每张要两三分钟，比本地整条管线还慢，先用 medium 试提示词更划算。
        </p>
        <p v-if="notReady" class="tiny warn">
          {{ health?.reason || 'ComfyUI 尚未就绪，任务会排队等待' }}
        </p>
      </div>
    </section>

    <!-- 右：结果 -->
    <section class="results">
      <div v-if="!shown.length" class="empty dt-faint">
        <p>还没有生成记录。</p>
        <p class="tiny">左侧填个提示词，点生成。</p>
      </div>
      <div v-else class="grid">
        <JobCard
          v-for="(j, i) in shown"
          :key="j.id"
          :job="j"
          class="dt-enter"
          :style="{ animationDelay: `${Math.min(i, 6) * 45}ms` }"
          @cancel="cancel"
        />
      </div>
    </section>
  </div>
</template>

<style scoped>
/* 左栏定宽是刻意的：参数表单的行宽一旦跟着窗口变，滑条和输入框的比例
   就会在不同屏幕上完全不一样。右栏吃掉剩余空间去排出图网格。 */
.page {
  display: grid;
  grid-template-columns: 380px minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}
/* 自检横幅横跨两列，出现时把内容整体往下推 */
.span {
  grid-column: 1 / -1;
}
/* 页头在网格里，行间距由 gap 给，它自己的下边距要撤掉，
   否则两段间距叠起来会在页头下方留出一道莫名其妙的空白。 */
.span.ph {
  margin-bottom: 0;
}

.panel {
  position: sticky;
  /* 顶栏改成左侧栏之后上方没有东西挡着了，留出页面自己的上边距即可 */
  top: 18px;
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 36px);
  overflow: auto;
}
.pad {
  padding: 16px 18px;
}
.pad + .pad,
.advanced,
.submit {
  border-top: 1px solid var(--dt-border);
}

/* 材质 / 图片切换。做成分段控件而不是下拉：只有两档，
   而且这是"我现在要做什么"的选择，值得一眼看见当前在哪一档。 */
.kinds {
  display: flex;
  gap: 4px;
  padding: 3px;
  margin-bottom: 14px;
  border-radius: var(--dt-radius);
  background: var(--dt-surface2);
}
.kind {
  flex: 1;
  font: inherit;
  font-size: var(--dt-fs-base);
  padding: 5px 0;
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

.styles {
  margin-top: 10px;
}

.refine-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: -8px;
}
.refined {
  padding: 11px 13px;
  margin-top: -4px;
  border-left: 2px solid var(--dt-accent);
}
.rtitle {
  margin: 0 0 6px;
  font-size: var(--dt-fs-xs);
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--dt-ink-faint);
  display: flex;
  gap: 9px;
  align-items: baseline;
}
.rtext {
  margin: 0 0 9px;
  font-size: var(--dt-fs-sm);
  line-height: 1.7;
  color: var(--dt-ink);
  max-height: 200px;
  overflow: auto;
}
.racts {
  display: flex;
  gap: 8px;
}
.desc {
  margin: 12px 0 0;
  font-size: var(--dt-fs-base);
  line-height: 1.65;
}
.notice {
  margin: 8px 0 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.6;
  color: var(--dt-warn);
  border-left: 2px solid var(--dt-warn);
  padding-left: 9px;
}

.fields {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.head .name {
  font-size: var(--dt-fs-base);
  font-weight: 500;
}
.expanded {
  margin: -8px 0 0;
  font-size: var(--dt-fs-xs);
  line-height: 1.6;
  color: var(--dt-ink-faint);
  background: var(--dt-surface2);
  padding: 7px 9px;
  border-left: 2px solid var(--dt-border-strong);
  word-break: break-word;
}
.tiny {
  margin: 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.5;
}
.warn {
  color: var(--dt-warn);
  margin-top: 8px;
}
.warn-line {
  margin: -4px 0 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.65;
  color: var(--dt-warn);
  border-left: 2px solid var(--dt-warn);
  padding-left: 9px;
}
.warn-line b {
  font-weight: 500;
}
.submit {
  padding: 16px 18px;
}
.cost {
  margin: 0 0 10px;
  font-size: var(--dt-fs-sm);
  color: var(--dt-ink-faint);
  text-align: center;
}

.results {
  min-height: 300px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
  gap: 14px;
}
.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  height: 320px;
  border: 1px dashed var(--dt-border);
}
.empty p {
  margin: 0;
}

@media (max-width: 980px) {
  .page {
    grid-template-columns: 1fr;
  }
  .panel {
    position: static;
  }
}
</style>
