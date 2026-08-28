import type {
  CatalogEntry,
  CheckResult,
  ComfyHealth,
  ConfigPatch,
  DeployInfo,
  DeployOptions,
  ComfyVersion,
  ComfyVersionStatus,
  DeployStatus,
  FlagOption,
  Download,
  ImagenProvider,
  ManagerCapability,
  NodePack,
  NodeQueue,
  Inventory,
  Job,
  LogLine,
  Manifest,
  MaterialIndex,
  Picture,
  PictureMeta,
  RefItem,
  Refined,
  RuntimeConfig,
  Settings,
  WorkflowMeta,
} from './types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) },
  })
  const text = await resp.text()
  let body: unknown = null
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      // 后端出错时可能返回纯文本，保留原样以便展示。
      body = { error: text }
    }
  }
  if (!resp.ok) {
    const msg = (body as { error?: string })?.error ?? `HTTP ${resp.status}`
    throw new Error(msg)
  }
  return body as T
}

export interface GenerateRequest {
  workflow_id: string
  params: Record<string, unknown>
  variants: number
  name?: string
}

export const api = {
  checks: () => request<CheckResult>('/api/checks'),

  workflows: () =>
    request<{ workflows: WorkflowMeta[] }>('/api/workflows').then((r) => r.workflows),

  reloadWorkflows: () =>
    request<{ workflows: string[] }>('/api/workflows/reload', { method: 'POST' }),

  openInComfy: (id: string) =>
    request<{ file: string; comfy_url: string; hint: string }>(
      `/api/workflows/${id}/open-in-comfy`,
      { method: 'POST' },
    ),

  importWorkflow: (body: {
    id: string
    name?: string
    style?: string
    graph: unknown
    override?: boolean
  }) =>
    request<{ id: string; template: string; params: string }>('/api/workflows/import', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  generate: (req: GenerateRequest) =>
    request<{ jobs: Job[] }>('/api/generate', {
      method: 'POST',
      body: JSON.stringify(req),
    }).then((r) => r.jobs),

  jobs: (limit = 50) =>
    request<{ jobs: Job[] }>(`/api/jobs?limit=${limit}`).then((r) => r.jobs),

  job: (id: string) => request<Job>(`/api/jobs/${id}`),

  cancelJob: (id: string) =>
    request<{ ok: boolean }>(`/api/jobs/${id}/cancel`, { method: 'POST' }),

  materials: (q: { q?: string; style?: string; fav?: boolean; limit?: number } = {}) => {
    const p = new URLSearchParams()
    if (q.q) p.set('q', q.q)
    if (q.style) p.set('style', q.style)
    if (q.fav) p.set('fav', '1')
    if (q.limit) p.set('limit', String(q.limit))
    return request<{ materials: MaterialIndex[]; fts: boolean }>(
      `/api/materials?${p.toString()}`,
    )
  },

  material: (id: string) =>
    request<{ manifest: Manifest; index: MaterialIndex | null }>(`/api/materials/${id}`),

  setFavorite: (id: string, favorite: boolean) =>
    request<{ ok: boolean }>(`/api/materials/${id}/favorite`, {
      method: 'POST',
      body: JSON.stringify({ favorite }),
    }),

  // ---- 图片 ----
  pictures: (q: { q?: string; fav?: boolean; limit?: number } = {}) => {
    const p = new URLSearchParams()
    if (q.q) p.set('q', q.q)
    if (q.fav) p.set('fav', '1')
    if (q.limit) p.set('limit', String(q.limit))
    return request<{ pictures: Picture[] }>(`/api/pictures?${p}`)
  },
  picture: (id: string) => request<{ picture: Picture; meta: PictureMeta }>(`/api/pictures/${id}`),
  favoritePicture: (id: string, favorite: boolean) =>
    request<{ ok: boolean }>(`/api/pictures/${id}/favorite`, {
      method: 'POST',
      body: JSON.stringify({ favorite }),
    }),
  deletePicture: (id: string) =>
    request<{ ok: boolean }>(`/api/pictures/${id}`, { method: 'DELETE' }),

  // ---- ComfyUI 版本 ----
  // 仓库位置由后端从配置里的 main_py 推出来，不由这边指定——那个值决定了
  // 要去哪个目录跑 git checkout。
  comfyVersions: (kind: 'stable' | 'dev') =>
    request<{
      kind: string
      versions: ComfyVersion[]
      status: ComfyVersionStatus
      error?: string
    }>(`/api/comfy/versions?kind=${kind}`),
  fetchComfyVersions: () =>
    request<{ ok: boolean; status: ComfyVersionStatus }>('/api/comfy/versions/fetch', {
      method: 'POST',
    }),
  switchComfyVersion: (ref: string, mirror = true) =>
    request<{ ok: boolean }>('/api/comfy/versions/switch', {
      method: 'POST',
      body: JSON.stringify({ ref, mirror }),
    }),

  // ---- ComfyUI 启动参数 ----
  // 合成放在后端：Build 与 Parse 必须是同一份逻辑的两半，分家之后
  // "存进去的和读出来的不一样"这种 bug 会非常难查。
  comfyFlags: () =>
    request<{
      catalog: FlagOption[]
      values: Record<string, string>
      extra: string
      managed: string[]
      raw: string
    }>('/api/comfy/flags'),
  setComfyFlags: (values: Record<string, string>, extra: string) =>
    request<{ ok: boolean; need_restart: string[]; args: string }>('/api/comfy/flags', {
      method: 'POST',
      body: JSON.stringify({ values, extra }),
    }),

  // ---- 提示词扩写 ----
  // 列不出来不算错，所以 error 是可选字段而不是抛异常——网关不提供 /models、
  // 或者令牌只有调用权限没有列举权限都很常见，这时手输模型名照样能用。
  textModels: () =>
    request<{ models: string[]; default: string; error?: string }>('/api/prompts/models'),
  // ---- 提示词扩写 ----
  // 只返回结果，不替用户改——扩写完让他自己看一眼再决定用不用。
  refinePrompt: (prompt: string, purpose: 'texture' | 'image') =>
    request<Refined>('/api/prompts/refine', {
      method: 'POST',
      body: JSON.stringify({ prompt, purpose }),
    }),

  // ---- 参考图库 ----
  refs: () => request<{ refs: RefItem[] }>('/api/refs'),
  refFromPicture: (pictureID: string, name?: string) =>
    request<{ ref: RefItem }>('/api/refs/from-picture', {
      method: 'POST',
      body: JSON.stringify({ picture_id: pictureID, name }),
    }),
  renameRef: (id: string, name: string) =>
    request<{ ok: boolean }>(`/api/refs/${id}/rename`, {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  deleteRef: (id: string) => request<{ ok: boolean }>(`/api/refs/${id}`, { method: 'DELETE' }),
  // 每次用都重新上传进 ComfyUI：它可能被清空或换了实例，
  // 记着上次的文件名会在提交工作流时才炸，报错还看不出跟参考图库有关。
  useRef: (id: string) =>
    request<{ name: string; ref: RefItem }>(`/api/refs/${id}/use`, { method: 'POST' }),

  models: (refresh = false) =>
    request<{ inventory: Inventory; missing: number; downloads: Download[] }>(
      `/api/models${refresh ? '?refresh=1' : ''}`,
    ),

  downloadModel: (file: string, dir: string) =>
    request<Download>('/api/models/downloads', {
      method: 'POST',
      body: JSON.stringify({ file, dir }),
    }),

  cancelDownload: (id: string) =>
    request<{ ok: boolean }>(`/api/models/downloads/${id}/cancel`, { method: 'POST' }),

  settings: () => request<Settings>('/api/settings'),

  // 令牌只写不读：后端从不回传内容，界面上只显示"已设置 / 未设置"。
  setToken: (provider: string, token: string) =>
    request<{ tokens: Record<string, boolean> }>('/api/settings/tokens', {
      method: 'POST',
      body: JSON.stringify({ provider, token }),
    }),

  // 自定义接口地址。同样只回传 origin，不回传完整地址。
  // 后端保存后会立刻探一次连通性，message 里带结果。
  setEndpoint: (provider: string, baseURL: string) =>
    request<{ origin: string; message: string }>('/api/settings/endpoint', {
      method: 'POST',
      body: JSON.stringify({ provider, base_url: baseURL }),
    }),

  // ---- 云端底图来源 ----
  // 模型清单是现问服务端的，所以这个接口可能慢到一秒左右，不要放进热路径。
  imagenProviders: () => request<{ providers: ImagenProvider[] }>('/api/imagen/providers'),

  // ---- 节点管理（代理 ComfyUI-Manager）----
  managerInfo: () => request<ManagerCapability>('/api/nodes/manager'),

  nodes: (q: { q?: string; state?: string; limit?: number; offset?: number } = {}) => {
    const p = new URLSearchParams()
    if (q.q) p.set('q', q.q)
    if (q.state) p.set('state', q.state)
    if (q.limit) p.set('limit', String(q.limit))
    if (q.offset) p.set('offset', String(q.offset))
    return request<{ packs: NodePack[]; total: number }>(`/api/nodes?${p}`)
  },

  nodeAction: (id: string, action: string, version?: string) =>
    request<{ ok: boolean; hint: string }>('/api/nodes/action', {
      method: 'POST',
      body: JSON.stringify({ id, action, version }),
    }),

  nodeQueue: () => request<NodeQueue>('/api/nodes/queue'),

  // ---- 模型库 ----
  catalog: (q: { q?: string; kind?: string; source?: string; limit?: number } = {}) => {
    const p = new URLSearchParams()
    if (q.q) p.set('q', q.q)
    if (q.kind) p.set('kind', q.kind)
    if (q.source) p.set('source', q.source)
    if (q.limit) p.set('limit', String(q.limit))
    return request<{ entries: CatalogEntry[]; warnings?: string[] }>(`/api/catalog/models?${p}`)
  },

  modelDirs: () => request<{ dirs: string[] }>('/api/catalog/dirs').then((r) => r.dirs),

  catalogDownload: (body: {
    source: string
    id: string
    query?: string
    kind?: string
    dir?: string
  }) =>
    request<Download>('/api/catalog/download', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  comfyStatus: () => request<ComfyHealth>('/api/comfy/status'),

  comfyRestart: () => request<ComfyHealth>('/api/comfy/restart', { method: 'POST' }),

  // 启停可能要等好几分钟（ComfyUI 冷启动 + 等启动期后台任务安定），
  // 调用方要按长操作处理，别配超时。
  comfyStart: () =>
    request<{ ok: boolean; health: ComfyHealth }>('/api/comfy/start', { method: 'POST' }),
  comfyStop: () =>
    request<{ ok: boolean; health: ComfyHealth }>('/api/comfy/stop', { method: 'POST' }),

  // 增量拉日志。since 传上次的 last，服务端只回新增的。
  logs: (since = 0, limit = 500) =>
    request<{ lines: LogLine[]; last: number }>(`/api/logs?since=${since}&limit=${limit}`),

  // ---- 一键部署 ----
  // 开始之后立刻返回，进度靠轮询 deployStatus，详细输出去控制台看。
  deployInfo: () => request<DeployInfo>('/api/deploy'),
  deployStart: (opt: DeployOptions) =>
    request<{ ok: boolean; status: DeployStatus }>('/api/deploy', {
      method: 'POST',
      body: JSON.stringify(opt),
    }),
  deployCancel: () => request<{ ok: boolean }>('/api/deploy/cancel', { method: 'POST' }),
  deployApply: () =>
    request<{ ok: boolean; need_restart: string[]; python: string; main_py: string }>(
      '/api/deploy/apply',
      { method: 'POST' },
    ),

  // ---- 运行时配置 ----
  runtimeConfig: () => request<RuntimeConfig>('/api/config'),

  // 只发改动的字段：没带的项服务端不动，这样两处同时改也不会互相覆盖。
  updateConfig: (patch: ConfigPatch) =>
    request<{ ok: boolean; need_restart: string[] }>('/api/config', {
      method: 'POST',
      body: JSON.stringify(patch),
    }),
}

/** 材质套装内某个文件的 URL。 */
export function fileURL(materialID: string, file: string): string {
  return `/api/materials/${materialID}/files/${file}`
}
