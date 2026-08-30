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
import { workflows, segments, jobs, batchJobs, upsertJob, health } from '../store'
import { persisted, persistedEnum } from '../persist'
import type { Param, WorkflowMeta } from '../api/types'

const message = useMessage()

// 选中的预设与变体数都记住：回来还得重选一遍是很烦人的。
// selectedID 存下来的值可能指向一个已经被删掉的工作流，所以下面装载列表时
// 还要再验一次——这里没法验，那会儿列表还没到。
const selectedID = persisted<string>('gen.workflow', '')
const variants = persisted<number>('gen.variants', 4, (v) => [1, 2, 4, 6].includes(v as number))
const values = ref<Record<string, unknown>>({})
const submitting = ref(false)
const currentBatch = ref<string>('')

// 材质 / 图片是两类完全不同的产物，分成两档而不是混在一个预设列表里：
// 混着的话用户得先认出哪个预设产出什么，而两者的参数、产物、去处都不一样。
const kind = persistedEnum<'material' | 'image'>('gen.kind', 'material', ['material', 'image'])
const KINDS = [
  { label: '材质', value: 'material' as const },
  { label: '图片', value: 'image' as const },
]

const STYLE_LABEL: Record<string, string> = {
  realistic: '写实',
  stylized: '手绘',
  image: '出图',
}

const kindWorkflows = computed(() => workflows.value.filter((w) => w.kind === kind.value))

const current = computed<WorkflowMeta | undefined>(() =>
  workflows.value.find((w) => w.id === selectedID.value),
)

// 材质这一档下面还有两种选法：
//
//   组合   —— 出图模型 × 分解模型，两个下拉各选各的。这才是常态。
//   单工作流 —— 直接选一份完整模板，给自己导入的工作流用。
//
// 分成两种是因为组合出来的管线不该和自定义模板混在一个列表里：混着的话，八条
// 组合会把列表撑满，而它们本质上只是两个维度的乘积，用两个下拉表达才对得上
// 用户心里的模型。
const matMode = persistedEnum<'combo' | 'custom'>('gen.matmode', 'combo', ['combo', 'custom'])
const srcID = persisted<string>('gen.src', '')
const decID = persisted<string>('gen.dec', '')

const sourceSegs = computed(() => segments.value.filter((s) => s.segment === 'source'))
const decomposeSegs = computed(() => segments.value.filter((s) => s.segment === 'decompose'))
// 自定义模板 = 材质档里不是拼出来的那些。
const customWorkflows = computed(() => kindWorkflows.value.filter((w) => !w.source_segment))
// 组合模式只在真的有段可选时才成立，否则退回单工作流——不然界面上是两个空下拉。
const canCombine = computed(() => sourceSegs.value.length > 0 && decomposeSegs.value.length > 0)
const comboMode = computed(() => kind.value === 'material' && matMode.value === 'combo' && canCombine.value)

const segOptions = (list: WorkflowMeta[]) =>
  list.map((w) => ({
    label: w.name,
    value: w.id,
    style: STYLE_LABEL[w.domain ?? ''] ?? '',
    cloud: !!w.source,
  }))
const sourceOptions = computed(() => segOptions(sourceSegs.value))
const decomposeOptions = computed(() =>
  // 分解段没有"画风"也没有"底图在哪出"，两个标签都传空——挂个"本地"上去
  // 是句不知所云的话。
  decomposeSegs.value.map((w) => ({ label: w.name, value: w.id, style: '', cloud: null })),
)

// 两个下拉推出选中的组合。
//
// 用事件而不是 watch(srcID, decID)：watch 会在列表加载、切档这些时刻也触发，
// 拿着**上次残留的** srcID/decID 去覆盖刚刚恢复出来的 selectedID——表现是每次
// 进页面都跳回上一次的组合，而不是你真正选的那个。改成只在用户动下拉时才推。
//
// 找不到对应组合就不动 selectedID：那说明两段拼不起来，保持原样比把界面清空好。
function applyCombo() {
  if (!comboMode.value || !srcID.value || !decID.value) return
  const id = `${srcID.value}.${decID.value}`
  if (workflows.value.some((w) => w.id === id)) selectedID.value = id
}
function pickSource(id: string) {
  srcID.value = id
  applyCombo()
}
function pickDecompose(id: string) {
  decID.value = id
  applyCombo()
}
function pickMode(m: 'combo' | 'custom') {
  matMode.value = m
  applyCombo()
}

// 反过来：选中的是组合时，把它拆回两个下拉。恢复上次的选择、以及详情页
// 「再来一张」带过来的那个 workflow_id，都要靠这一步才能在界面上显示对。
watch(
  [current, segments],
  () => {
    const wf = current.value
    if (wf?.source_segment && wf.decompose_segment) {
      srcID.value = wf.source_segment
      decID.value = wf.decompose_segment
      return
    }
    if (!srcID.value) srcID.value = sourceSegs.value[0]?.id ?? ''
    if (!decID.value) decID.value = decomposeSegs.value[0]?.id ?? ''
  },
  { immediate: true },
)

// 单工作流模式下只列自定义模板——组合出来的那八条由上面两个下拉表达，
// 混进来只会让人以为有两套一样的东西。
const singleList = computed(() =>
  kind.value === 'material' ? customWorkflows.value : kindWorkflows.value,
)

// 许可要看并集：云端底图 + CHORD 的 research-only 标记来自分解段，
// 只看一边会把它漏掉。
const nonCommercial = computed(() =>
  (current.value?.licenses ?? []).filter((l) => !l.commercial),
)

const workflowOptions = computed(() =>
  singleList.value.map((w) => ({
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
  const tags = []
  if (opt.style) tags.push(h('span', { class: 'dt-opt-tag' }, String(opt.style)))
  // null 表示"这一项没有底图来源这个维度"（分解段就是），与 false（本地出图）
  // 是两回事，所以判 null 而不是判真假。
  if (opt.cloud !== null && opt.cloud !== undefined) {
    tags.push(
      h(
        'span',
        { class: opt.cloud ? 'dt-opt-tag dt-opt-tag-accent' : 'dt-opt-tag' },
        opt.cloud ? '云端底图' : '本地',
      ),
    )
  }
  return h('div', { class: 'dt-opt' }, [
    h('span', { class: 'dt-opt-name' }, String(opt.label)),
    ...tags,
  ])
}

// 工作流列表到位后决定选谁。优先级从高到低：
//
//   1. 详情页点「再来一张」指定的那个——那是用户刚做出的明确动作
//   2. 上次离开时选的那个，**且它还在列表里**
//   3. 上次那个档里的第一个
//   4. 列表里的第一个
//
// 第 2 条必须验在不在：工作流会被删被改名，存下来的 id 指向一个不存在的东西时
// 下拉框会显示空白且怎么点都不对，还不给任何报错。
watch(
  workflows,
  (list) => {
    if (!list.length) return

    let want = ''
    try {
      const raw = sessionStorage.getItem('dt.refill')
      if (raw) want = (JSON.parse(raw) as { workflow_id: string }).workflow_id
    } catch {
      want = ''
    }

    const wf =
      list.find((w) => w.id === want) ??
      list.find((w) => w.id === selectedID.value) ??
      list.find((w) => w.kind === kind.value) ??
      list[0]!
    selectedID.value = wf.id
    // 档要跟着选中的走，否则选中的预设不在当前档里，
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

        <!-- 材质档下面还分组合 / 单工作流。只有真存在自定义模板时才把这个
             切换露出来：没有的话它是个点了什么也没有的死按钮。 -->
        <div v-if="kind === 'material' && canCombine && customWorkflows.length" class="modes">
          <button class="mode" :class="{ on: matMode === 'combo' }" @click="pickMode('combo')">
            组合
          </button>
          <button class="mode" :class="{ on: matMode === 'custom' }" @click="pickMode('custom')">
            单工作流
          </button>
        </div>

        <template v-if="comboMode">
          <p class="dt-label">出图模型</p>
          <NSelect
            :value="srcID"
            :options="sourceOptions"
            @update:value="pickSource"
            :render-label="renderWorkflow"
            class="styles"
          />
          <p class="dt-label sp">分解模型</p>
          <NSelect
            :value="decID"
            :options="decomposeOptions"
            @update:value="pickDecompose"
            :render-label="renderWorkflow"
            class="styles"
          />
        </template>
        <template v-else>
          <p class="dt-label">{{ kind === 'material' ? '工作流' : '出图方式' }}</p>
          <NSelect
            v-model:value="selectedID"
            :options="workflowOptions"
            :render-label="renderWorkflow"
            class="styles"
          />
        </template>

        <p v-if="current" class="desc dt-muted">{{ current.description }}</p>
        <!-- 两段搭不上只提醒不拦：手绘图对 CHORD 是分布外输入，结果会明显变差，
             但要不要试是用户的事。 -->
        <p v-if="current?.mismatch" class="warn">{{ current.mismatch }}</p>
        <p v-for="lic in nonCommercial" :key="lic.component" class="notice">
          {{ lic.component }} 为研究用途许可，产出不可商用
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

/* 组合 / 单工作流。比档位切换（.kinds）低一级，所以做得更轻：
   没有底板，只有一条选中下划线，免得两排一样的胶囊按钮叠在一起分不出主次。 */
.modes {
  display: flex;
  gap: 14px;
  margin-bottom: 12px;
}
.mode {
  font: inherit;
  font-size: var(--dt-fs-sm);
  padding: 2px 0 5px;
  border: none;
  border-bottom: 2px solid transparent;
  background: none;
  color: var(--dt-ink-muted);
  cursor: pointer;
  transition:
    color 0.16s ease,
    border-color 0.16s ease;
}
.mode:hover {
  color: var(--dt-ink);
}
.mode.on {
  color: var(--dt-accent);
  border-bottom-color: var(--dt-accent);
}

.styles {
  margin-top: 10px;
}
/* 第二个下拉的标签要跟上面那个拉开，不然两组控件糊成一片。 */
.sp {
  margin-top: 14px;
}

/* 两段搭不上的提醒。与 .notice（许可）区分开：那条是硬约束，这条是"可以试，
   但心里有数"，所以用 muted 的说明体而不是同一种警告色块。 */
.warn {
  margin: 8px 0 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.6;
  color: var(--dt-ink-muted);
  border-left: 2px solid var(--dt-border-strong);
  padding-left: 9px;
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
