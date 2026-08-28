// 与后端 internal/workflow、internal/store、internal/material 的 JSON 一一对应。

// image 类型不走通用控件，由 ReferenceInput 单独承载（拖放 + 预览 + 重绘幅度）。
export type ParamType = 'string' | 'int' | 'float' | 'bool' | 'enum' | 'image'

export interface Param {
  key: string
  label: string
  type: ParamType
  multiline?: boolean
  options?: (string | number)[]
  prefix?: string
  suffix?: string
  default?: unknown
  min?: number
  max?: number
  note?: string
  /** 由后端填写，界面上不显示（例如云端底图落盘后的文件名）。 */
  hidden?: boolean
  /** 选项要运行时才知道的参数用它指定控件，留空则按 type 渲染。 */
  widget?: 'imagen-model' | string
}

export interface OutputSpec {
  node: string
  colorspace: string
  y?: string
  packing?: string
  role?: string
  note?: string
}

export interface LicenseNotice {
  component: string
  license: string
  commercial: boolean
  replaceable_segment: string
  alternatives: string[]
}

/** 底图来自外部服务时的声明。缺省表示由本地 ComfyUI 自己采样。 */
export interface SourceSpec {
  kind: 'api'
  provider: string
  image_param: string
  roles: Record<string, string>
  /** true = 云端拿回来的就是成品，整条链路不经过 ComfyUI，也就没有节点图。 */
  direct_output?: boolean
}

/** material = 一整套 PBR 通道；image = 单张图片。 */
export type WorkflowKind = 'material' | 'image'

export interface WorkflowMeta {
  id: string
  kind: WorkflowKind
  version: number
  name: string
  style: 'realistic' | 'stylized' | string
  description: string
  resolution: number
  tileable: boolean
  /** 该参数大于 0 时产物才是无缝的（云端底图管线用）。 */
  tileable_when_positive?: string
  source?: SourceSpec
  license_notice?: LicenseNotice
  node_packs: string[]
  params: Param[]
  advanced: Param[]
  outputs: Record<string, OutputSpec>
}

export interface ImagenModel {
  id: string
  label: string
  provider: string
  sizes?: string[]
  qualities?: string[]
  edits: boolean
  note?: string
  /** false 表示是从服务端发现的新模型，参数表只能靠推定。 */
  known: boolean
}

export interface ImagenProvider {
  id: string
  label: string
  configured: boolean
  models: ImagenModel[]
  error?: string
}

export type JobStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled'

export interface Job {
  id: string
  material_id: string
  workflow_id: string
  batch_id?: string
  params: Record<string, unknown>
  status: JobStatus
  prompt_id?: string
  progress: number
  stage?: string
  error?: string
  created_at: string
  started_at?: string
  finished_at?: string
}

export interface MaterialMap {
  file: string
  colorspace: string
  y?: string
  packing?: string
  role?: string
  width: number
  height: number
  bytes: number
  sha256: string
}

export interface Manifest {
  schema: string
  schema_version: number
  id: string
  name: string
  style: string
  workflow: { id: string; version: number }
  prompt: string
  negative?: string
  seed: number
  resolution: number
  tileable: boolean
  maps: Record<string, MaterialMap>
  preview?: string
  created_at: string
  generator: {
    comfyui?: string
    checkpoint?: string
    pbr_estimator?: string
    node_packs?: string[]
  }
  license_flags?: { commercial_use: boolean; reason?: string }
  params?: Record<string, unknown>
  /**
   * 底图来自云端时的出处。这个字段一旦存在就说明这份材质**复现不了**：
   * 云端图像接口不支持种子，manifest 里的 seed 只作用于本地那一半。
   */
  source?: {
    provider: string
    model: string
    prompt: string
    size?: string
    quality?: string
    revised_prompt?: string
    input_tokens?: number
    output_tokens?: number
    cost_usd?: number
    elapsed_ms?: number
    flattened: boolean
    falloff_before?: number
    falloff_after?: number
  }
}

export interface MaterialIndex {
  id: string
  name: string
  style: string
  workflow_id: string
  prompt: string
  negative?: string
  seed: number
  resolution: number
  favorite: boolean
  tags?: string[]
  created_at: string
}

export interface ComfyHealth {
  mode: string
  alive: boolean
  ready: boolean
  reason?: string
  base_url: string
  pid?: number
  comfyui_version?: string
  device?: string
  vram_total_mb?: number
  vram_free_mb?: number
  queue_depth: number
  restarts: number
  /** 子进程起来了但 ComfyUI 还没应答。冷启动几十秒到几分钟，不是"未连接"。 */
  starting: boolean
  /** 已经等了多久（秒）。 */
  starting_secs?: number
  last_checked_at: string
  /** true = 用户主动停的，不是它自己挂了。别拿 reason 里的中文去判断。 */
  user_stopped: boolean
}

export interface BusEvent {
  type:
    | 'job.queued'
    | 'job.progress'
    | 'job.done'
    | 'job.failed'
    | 'comfy.status'
    | 'model.download'
    // 心跳。只用来让前端知道连接还活着，没有负载（见 store.ts 的 STALE_AFTER）。
    | 'hb'
  job?: Job
  data?: unknown
}

export interface ModelFile {
  name: string
  dir: string
  path: string
  size: number
  modified: string
  used_by?: string[]
}

export interface ModelFolder {
  name: string
  paths: string[]
  count: number
  total_bytes: number
}

export interface ModelRequirement {
  kind: string
  file: string
  dir: string
  size_bytes: number
  source: string
  download_url?: string
  auth: string
  note?: string
  workflow_ids: string[]
  present: boolean
  actual_bytes?: number
  target?: string
}

export interface Inventory {
  folders: ModelFolder[]
  files: ModelFile[]
  requirements: ModelRequirement[]
  total_bytes: number
  scanned_at: string
}

export type DownloadState = 'queued' | 'running' | 'done' | 'failed' | 'canceled'

export interface Download {
  id: string
  file: string
  dir: string
  target: string
  source: string
  state: DownloadState
  received: number
  total: number
  error?: string
  started_at: string
  ended_at?: string
}

export interface NodePack {
  id: string
  title: string
  author: string
  description: string
  repository: string
  reference: string
  state: 'enabled' | 'disabled' | 'not-installed' | string
  version: string
  cnr_latest: string
  stars: number
  last_update: string
  trust: boolean
}

export interface ManagerCapability {
  available: boolean
  version?: string
  api?: string
  has_node_list: boolean
  reason?: string
}

export interface NodeQueue {
  total_count: number
  done_count: number
  in_progress_count: number
  is_processing: boolean
}

export interface CatalogEntry {
  source: 'curated' | 'civitai' | string
  id: string
  name: string
  description?: string
  author?: string
  kind: string
  dir: string
  base?: string
  filename: string
  url: string
  size_bytes?: number
  downloads?: number
  trained_words?: string[]
  preview?: string
  page?: string
  auth: string
  installed: boolean
}

export interface Check {
  key: string
  label: string
  status: 'ok' | 'warn' | 'fail'
  detail: string
  fix?: string
  items?: string[]
}

export interface CheckResult {
  status: 'ok' | 'warn' | 'fail'
  checks: Check[]
}

export interface RuntimeConfig {
  config_path: string
  imagen: {
    proxy: string
    openai_base_url: string
    flatten: number
    refine_model: string
    refine_model_default: string
  }
  comfy: {
    mode: 'managed' | 'attach'
    base_url: string
    python: string
    main_py: string
    extra_args: string[] | null
    auto_restart: boolean
    reserve_vram_gb: number
    python_exists: boolean
    python_detail: string
    main_py_exists: boolean
    main_py_detail: string
    alive: boolean
    user_stopped: boolean
  }
  paths: { output: string; data: string; workflows: string; root: string }
  /** 环境变量里的代理；配置留空时走的就是它。 */
  env_proxy: string
  os: string
}

export interface ConfigPatch {
  proxy?: string
  openai_base_url?: string
  flatten?: number
  refine_model?: string
  comfy_mode?: string
  comfy_base_url?: string
  comfy_python?: string
  comfy_main_py?: string
  comfy_extra_args?: string[]
  comfy_auto_restart?: boolean
  comfy_reserve_vram?: number
}

/** 一张生成出来的图片（不是材质套装）。 */
export interface Picture {
  id: string
  name: string
  workflow_id: string
  prompt: string
  negative?: string
  seed: number
  width: number
  height: number
  provider?: string
  model?: string
  cost_usd?: number
  favorite: boolean
  tags?: string[]
  created_at: string
}

/** 与图片并排落盘的那份元信息。source 存在即表示复现不了。 */
export interface PictureMeta {
  schema: string
  schema_version: number
  id: string
  name: string
  file: string
  workflow_id: string
  prompt: string
  negative?: string
  seed: number
  width: number
  height: number
  reference?: string
  source?: {
    provider: string
    model: string
    size?: string
    quality?: string
    revised_prompt?: string
    input_tokens?: number
    output_tokens?: number
    cost_usd?: number
    elapsed_ms?: number
  }
  params?: Record<string, unknown>
  created_at: string
}

/** 参考图库里的一条。 */
export interface RefItem {
  id: string
  name: string
  file: string
  comfy_name?: string
  width: number
  height: number
  bytes: number
  origin?: string
  created_at: string
}

export interface Refined {
  prompt: string
  model: string
  usage: { input_tokens?: number; output_tokens?: number }
  elapsed_ms: number
}

/** ComfyUI 启动参数目录里的一项。 */
export interface FlagChoice {
  value: string
  label: string
  note?: string
  /** 这个选项展开成的参数；仅界面预览用，实际合成在后端做。 */
  args?: string[]
}

export interface FlagOption {
  key: string
  label: string
  help: string
  kind: 'choice' | 'bool'
  icon?: string
  choices?: FlagChoice[]
  flag?: string
  /** true = 开关打开时**不加**这个参数（ComfyUI 的开关多是 --disable-xxx）。 */
  invert?: boolean
  advanced?: boolean
}

/** ComfyUI 的一个可切换版本。 */
export interface ComfyVersion {
  ref: string
  short: string
  name: string
  date: string
  kind: 'stable' | 'dev'
  current: boolean
}

export interface ComfyVersionStatus {
  available: boolean
  reason?: string
  dir?: string
  remote?: string
  branch?: string
  ref?: string
  short?: string
  name?: string
  date?: string
  /** 浅克隆：历史不全，一个版本都列不出来，得先补一次。 */
  shallow: boolean
  /** 工作区有改动，切版本会冲掉，所以会被拦下。 */
  dirty: boolean
  dirty_files?: string[]
}

export interface DeployStep {
  key: string
  title: string
  state: 'pending' | 'running' | 'done' | 'skipped' | 'failed'
  detail?: string
  ms?: number
}

export interface DeployStatus {
  running: boolean
  steps: DeployStep[]
  error?: string
  started_at?: string
  finished_at?: string
  /** 装完之后应当写进配置的两个路径。 */
  python?: string
  main_py?: string
}

export interface DeployOptions {
  dir: string
  py_version: string
  torch: string
  mirror: boolean
  model_base_path: string
  skip_torch?: boolean
}

export interface DeployInfo {
  status: DeployStatus
  defaults: DeployOptions
  current: { python: string; main_py: string }
}

export interface LogLine {
  /** 单调递增，客户端据此增量拉取。 */
  seq: number
  at: string
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR' | string
  /** backend = 后端自己；comfyui = ComfyUI 的标准输出。 */
  source: 'backend' | 'comfyui' | string
  text: string
}

export interface TokenProvider {
  id: string
  label: string
  help: string
  set: boolean
  /** 该来源是否允许自定义接口地址（兼容网关 / 自建中转）。 */
  endpoint: boolean
  /** 当前自定义地址的协议+主机；空表示用官方地址。完整地址不回传——网关地址里可能带密钥。 */
  endpoint_origin?: string
  endpoint_default?: string
  endpoint_help?: string
}

export interface Settings {
  tokens: Record<string, boolean>
  token_providers: TokenProvider[]
  comfy: ComfyHealth
  output_dir: string
}

// 通道的展示顺序与中文名。后端不关心这些，纯展示层的事。
export const CHANNEL_ORDER = [
  'basecolor',
  'normal',
  'roughness',
  'metallic',
  'ao',
  'height',
  'orm',
  'source',
] as const

export const CHANNEL_LABEL: Record<string, string> = {
  basecolor: '基础色',
  normal: '法线',
  roughness: '粗糙度',
  metallic: '金属度',
  ao: '环境光遮蔽',
  height: '高度',
  orm: 'ORM 打包',
  source: '原始生成图',
}
