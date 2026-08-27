<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NButton,
  NInput,
  NInputNumber,
  NSelect,
  NSlider,
  NSpin,
  NSwitch,
  NTag,
  NAlert,
  useDialog,
  useMessage,
} from 'naive-ui'
import { RouterLink } from 'vue-router'
import SettingRow from '../components/SettingRow.vue'
import DeployPanel from '../components/DeployPanel.vue'
import PageHeader from '../components/PageHeader.vue'
import { api } from '../api/client'
import { health, loadImagen } from '../store'
import type { ConfigPatch, RuntimeConfig, Settings } from '../api/types'

const message = useMessage()
const dialog = useDialog()

const cfg = ref<RuntimeConfig | null>(null)
const settings = ref<Settings | null>(null)
const loading = ref(true)
const error = ref('')
const busy = ref('')

// 令牌与接口地址都是只写不读的，输入框永远从空开始。
const tokenInput = ref<Record<string, string>>({})
const endpointInput = ref<Record<string, string>>({})

// 这些绑到本地副本，改完点保存才提交——设置页里边打字边保存
// 会在填一半的路径上反复校验失败。
const proxy = ref('')
const flatten = ref(1)
const comfyPython = ref('')
const comfyMainPy = ref('')
const comfyBaseURL = ref('')
const comfyMode = ref<'managed' | 'attach'>('managed')
const autoRestart = ref(true)
const refineModel = ref('')
const textModels = ref<string[]>([])
const modelsError = ref('')
const loadingModels = ref(false)

// 下拉里除了服务端列出来的，还要保证当前已保存的那个在场：
// 它可能是手输的、也可能是网关这次没列出来的，不放进去的话下拉框会显示空白，
// 让人以为设置丢了。
const textModelOptions = computed(() => {
  const ids = [...textModels.value]
  const cur = refineModel.value.trim()
  if (cur && !ids.includes(cur)) ids.unshift(cur)
  return ids.map((id) => ({ label: id, value: id }))
})

async function loadTextModels() {
  loadingModels.value = true
  modelsError.value = ''
  try {
    const r = await api.textModels()
    textModels.value = r.models
    modelsError.value = r.error ?? ''
  } catch (e) {
    modelsError.value = String((e as Error).message)
  } finally {
    loadingModels.value = false
  }
}
const reserveVRAM = ref(1)

const providers = computed(() => settings.value?.token_providers ?? [])

/** 只刷服务端状态，不动输入框。 */
async function refresh() {
  error.value = ''
  try {
    const [c, s] = await Promise.all([api.runtimeConfig(), api.settings()])
    cfg.value = c
    settings.value = s
  } catch (e) {
    error.value = String((e as Error).message)
  } finally {
    loading.value = false
  }
}

/**
 * 首次进来时把输入框也按服务端的值填上。
 *
 * 保存之后只调 refresh 不调 load：保存 A 字段时把 B 字段正在编辑的内容
 * 冲掉，是设置页里最让人恼火的一类 bug。
 */
async function load() {
  await refresh()
  const c = cfg.value
  if (!c) return
  proxy.value = c.imagen.proxy
  flatten.value = c.imagen.flatten
  comfyPython.value = c.comfy.python
  comfyMainPy.value = c.comfy.main_py
  comfyBaseURL.value = c.comfy.base_url
  comfyMode.value = c.comfy.mode
  autoRestart.value = c.comfy.auto_restart
  refineModel.value = c.imagen.refine_model
  reserveVRAM.value = c.comfy.reserve_vram_gb
  // 不 await：列模型要打一趟外网，卡着整个设置页不值得。
  void loadTextModels()
}
onMounted(load)

async function save(patch: ConfigPatch, what: string) {
  busy.value = what
  try {
    const r = await api.updateConfig(patch)
    await refresh()
    // 加问号是第二道防线：后端已经保证不返回 null 了，但"保存成功却弹红字"
    // 这种错误的代价太不成比例——真出问题时宁可少提示一句，也不要谎报失败。
    if (r.need_restart?.length) {
      message.warning(`已保存。${r.need_restart.join('、')} 需要重启后端才生效`)
    } else {
      message.success('已保存')
    }
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    busy.value = ''
  }
}

async function saveToken(provider: string) {
  const v = tokenInput.value[provider] ?? ''
  try {
    const r = await api.setToken(provider, v)
    if (settings.value) {
      settings.value.tokens = r.tokens
      settings.value.token_providers = settings.value.token_providers.map((p) =>
        p.id === provider ? { ...p, set: !!r.tokens[provider] } : p,
      )
    }
    tokenInput.value[provider] = ''
    message.success(v ? '已保存' : '已清除')
  } catch (e) {
    message.error(String((e as Error).message))
  }
}

async function saveEndpoint(provider: string) {
  busy.value = 'endpoint-' + provider
  try {
    const r = await api.setEndpoint(provider, endpointInput.value[provider] ?? '')
    if (settings.value) {
      settings.value.token_providers = settings.value.token_providers.map((p) =>
        p.id === provider ? { ...p, endpoint_origin: r.origin } : p,
      )
    }
    endpointInput.value[provider] = ''
    // 后端保存时顺手探了一次连通性，把结果原样告诉用户
    if (r.message.includes('不通')) message.warning(r.message)
    else message.success(r.message)
    await loadImagen(true)
    // 换了网关，之前那份模型清单就不作数了——扩写的地址会回落到出图那个，
    // 所以两个来源改哪一个都得重列。
    void loadTextModels()
  } catch (e) {
    message.error(String((e as Error).message))
  } finally {
    busy.value = ''
  }
}

const alive = computed(() => !!health.value?.alive)

async function comfyAct(what: 'start' | 'stop' | 'restart') {
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
    await refresh()
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
    onPositiveClick: () => comfyAct('stop'),
  })
}

const MODES = [
  { label: 'managed（后端托管）', value: 'managed' },
  { label: 'attach（连接已有实例）', value: 'attach' },
]

function hostOf(u?: string): string {
  if (!u) return ''
  try {
    return new URL(u).host
  } catch {
    return u
  }
}
</script>

<template>
  <div class="dt-page dt-page-narrow">
    <PageHeader
      title="设置"
      desc="令牌、网络、ComfyUI 运行环境都在这里。改动写回配置文件，标了「需重启」的项要重启后端才生效。"
    />

    <div v-if="loading" class="center"><NSpin /></div>
    <NAlert v-else-if="error" type="error" :bordered="false">{{ error }}</NAlert>

    <template v-else-if="cfg">
      <!-- ── 访问令牌 ── -->
      <p class="dt-label sec">访问令牌</p>
      <p class="secnote dt-faint">
        只保存在本机的 <span class="dt-mono">{{ cfg.config_path.replace(/[^\\/]+$/, 'secrets.json') }}</span>，
        权限 0600 且已 gitignore。保存后界面不再显示内容，只显示是否已设置——想换就重新填一次。
      </p>
      <div class="rows">
        <template v-for="p in providers" :key="p.id">
          <SettingRow icon="key" :title="p.label" :desc="p.help" stack>
            <div class="line">
              <NInput
                v-model:value="tokenInput[p.id]"
                type="password"
                show-password-on="click"
                :placeholder="p.set ? '已设置，填新值可替换' : '粘贴令牌'"
                size="small"
              />
              <NTag size="tiny" :bordered="false" :type="p.set ? 'success' : 'default'">
                {{ p.set ? '已设置' : '未设置' }}
              </NTag>
              <NButton size="small" @click="saveToken(p.id)">
                {{ tokenInput[p.id] ? '保存' : '清除' }}
              </NButton>
            </div>
          </SettingRow>

          <SettingRow
            v-if="p.endpoint"
            icon="cloud"
            :title="`${p.label} 接口地址`"
            :desc="p.endpoint_help ?? ''"
            stack
          >
            <div class="line">
              <NInput
                v-model:value="endpointInput[p.id]"
                type="password"
                show-password-on="click"
                :placeholder="p.endpoint_default"
                size="small"
              />
              <span class="now dt-mono dt-faint">
                → {{ hostOf(p.endpoint_origin) || hostOf(p.endpoint_default) }}
              </span>
              <NButton
                size="small"
                :loading="busy === 'endpoint-' + p.id"
                @click="saveEndpoint(p.id)"
              >
                <!-- 没有官方地址可回退的项（例如"沿用上面那套"），说"清除"才对 -->
              {{
                endpointInput[p.id]
                  ? '保存'
                  : p.endpoint_default?.startsWith('http')
                    ? '恢复官方'
                    : '清除'
              }}
              </NButton>
            </div>
          </SettingRow>
        </template>
      </div>

      <!-- ── 网络 ── -->
      <p class="dt-label sec">网络</p>
      <div class="rows">
        <SettingRow
          icon="globe"
          title="代理"
          desc="用于访问云端底图接口。留空则跟随环境变量。后端若以服务方式启动，往往读不到你终端里的 HTTPS_PROXY，那时就得在这儿写死"
          stack
        >
          <div class="line">
            <NInput
              v-model:value="proxy"
              size="small"
              :placeholder="cfg.env_proxy || 'http://127.0.0.1:7890'"
              clearable
            />
            <span v-if="cfg.env_proxy" class="now dt-mono dt-faint">
              环境变量：{{ cfg.env_proxy }}
            </span>
            <NButton size="small" :loading="busy === 'proxy'" @click="save({ proxy }, 'proxy')">
              保存
            </NButton>
          </div>
        </SettingRow>
      </div>

      <!-- ── 云端底图 ── -->
      <p class="dt-label sec">云端</p>
      <div class="rows">
        <SettingRow
          icon="wand"
          title="提示词扩写模型"
          desc="生成台上「让模型扩写」用的文本模型。走的是同一个网关、同一把令牌，但通常不是出图那个模型"
          stack
        >
          <div class="line">
            <!--
              filterable + tag：清单只是帮忙，不是限制。网关不提供 /models、
              或者列出来的东西被我们的筛选漏掉了，都得能直接把名字打进去。
            -->
            <NSelect
              v-model:value="refineModel"
              :options="textModelOptions"
              :loading="loadingModels"
              :placeholder="cfg.imagen.refine_model_default"
              size="small"
              filterable
              tag
              clearable
            />
            <NButton size="small" tertiary :loading="loadingModels" @click="loadTextModels">
              刷新
            </NButton>
            <NButton
              size="small"
              :loading="busy === 'refine'"
              @click="save({ refine_model: refineModel }, 'refine')"
            >
              保存
            </NButton>
          </div>
          <p v-if="modelsError" class="modelnote dt-faint">
            列不出可用模型（{{ modelsError }}）。直接把模型名打进去也能用。
          </p>
        </SettingRow>

        <SettingRow
          icon="image"
          title="亮度场压平"
          desc="消掉云端模型的暗角。不压的话平铺时相邻瓦片的暗边会拼成可见网格。调低可保留模型原本的明暗氛围；0 为关闭"
        >
          <div class="slide">
            <NSlider v-model:value="flatten" :min="0" :max="1" :step="0.05" :tooltip="false" />
            <NInputNumber
              v-model:value="flatten"
              size="small"
              :min="0"
              :max="1"
              :step="0.05"
              class="num"
            />
            <NButton size="small" :loading="busy === 'flatten'" @click="save({ flatten }, 'flatten')">
              保存
            </NButton>
          </div>
        </SettingRow>
      </div>

      <!-- ── ComfyUI 环境 ── -->
      <p class="dt-label sec">ComfyUI 环境</p>
      <div class="rows">
        <DeployPanel @applied="load" />

        <SettingRow
          icon="chip"
          title="运行状态"
          :desc="
            cfg.comfy.mode === 'attach'
              ? 'attach 模式下后端不管理它的生命周期，启停请到它自己的窗口'
              : alive
                ? '后端托管中。停止后自动重启不会把它拉起来，直到你手动启动'
                : cfg.comfy.user_stopped
                  ? '已被手动停止'
                  : '未运行'
          "
        >
          <NTag size="small" :bordered="false" :type="alive ? 'success' : 'default'">
            {{ alive ? '运行中' : cfg.comfy.user_stopped ? '已停止' : '未连接' }}
          </NTag>
          <NButton
            v-if="alive"
            size="small"
            :loading="busy === 'stop'"
            :disabled="!!busy || cfg.comfy.mode === 'attach'"
            @click="confirmStop"
          >
            停止
          </NButton>
          <NButton
            v-else
            size="small"
            type="primary"
            :loading="busy === 'start'"
            :disabled="!!busy || cfg.comfy.mode === 'attach'"
            @click="comfyAct('start')"
          >
            启动
          </NButton>
          <NButton
            size="small"
            tertiary
            :loading="busy === 'restart'"
            :disabled="!!busy || !alive || cfg.comfy.mode === 'attach'"
            @click="comfyAct('restart')"
          >
            重启
          </NButton>
          <RouterLink to="/console"><NButton size="small" tertiary>看日志</NButton></RouterLink>
        </SettingRow>

        <SettingRow
          icon="sliders"
          title="运行模式"
          desc="managed 由后端拉起并看护 ComfyUI 子进程；attach 只连接你自己开着的实例，不管生死"
          restart
        >
          <NSelect v-model:value="comfyMode" :options="MODES" size="small" class="pick" />
          <NButton
            size="small"
            :loading="busy === 'mode'"
            @click="save({ comfy_mode: comfyMode }, 'mode')"
          >
            保存
          </NButton>
        </SettingRow>

        <SettingRow icon="globe" title="ComfyUI 地址" desc="后端连它的地址；managed 模式下也据此决定子进程监听的端口" restart stack>
          <div class="line">
            <NInput v-model:value="comfyBaseURL" size="small" placeholder="http://127.0.0.1:8188" />
            <NButton
              size="small"
              :loading="busy === 'baseurl'"
              @click="save({ comfy_base_url: comfyBaseURL }, 'baseurl')"
            >
              保存
            </NButton>
          </div>
        </SettingRow>

        <SettingRow
          icon="folder"
          title="Python 解释器"
          :desc="
            cfg.comfy.python_exists
              ? '用来跑 ComfyUI 的解释器。写相对路径的话按程序根目录解析，整个文件夹搬走也能用'
              : `路径有问题：${cfg.comfy.python_detail}`
          "
          restart
          stack
        >
          <div class="line">
            <NInput v-model:value="comfyPython" size="small" placeholder="runtime/python/python.exe" />
            <NTag size="tiny" :bordered="false" :type="cfg.comfy.python_exists ? 'success' : 'error'">
              {{ cfg.comfy.python_exists ? '存在' : '找不到' }}
            </NTag>
            <NButton
              size="small"
              :loading="busy === 'python'"
              @click="save({ comfy_python: comfyPython }, 'python')"
            >
              保存
            </NButton>
          </div>
        </SettingRow>

        <SettingRow
          icon="folder"
          title="ComfyUI 主程序"
          :desc="
            cfg.comfy.main_py_exists
              ? 'ComfyUI 的 main.py。它所在的目录会被当作工作目录'
              : `路径有问题：${cfg.comfy.main_py_detail}`
          "
          restart
          stack
        >
          <div class="line">
            <NInput v-model:value="comfyMainPy" size="small" placeholder="runtime/ComfyUI/main.py" />
            <NTag size="tiny" :bordered="false" :type="cfg.comfy.main_py_exists ? 'success' : 'error'">
              {{ cfg.comfy.main_py_exists ? '存在' : '找不到' }}
            </NTag>
            <NButton
              size="small"
              :loading="busy === 'mainpy'"
              @click="save({ comfy_main_py: comfyMainPy }, 'mainpy')"
            >
              保存
            </NButton>
          </div>
        </SettingRow>

        <SettingRow
          icon="shield"
          title="异常时自动重启"
          desc="ComfyUI 进程意外退出后自动拉起。手动停止不受影响——那是明确的意图，不会被覆盖"
          restart
        >
          <NSwitch
            v-model:value="autoRestart"
            @update:value="save({ comfy_auto_restart: autoRestart }, 'autorestart')"
          />
        </SettingRow>

        <SettingRow
          icon="gauge"
          title="显存余量"
          desc="留给别的程序的显存。同时开着虚幻编辑器时把它调到 2~3 GB：ComfyUI 只在装载模型那一刻看一眼空闲显存，之后不再复查；被吃干净之后 Windows 显卡驱动会悄悄回退到内存硬算，不报错，但慢几十倍——症状就是进度条卡住不动"
          restart
          stack
        >
          <div class="line">
            <NSlider
              v-model:value="reserveVRAM"
              :min="0"
              :max="8"
              :step="0.5"
              :marks="{ 0: '不留', 2: '2G', 4: '4G', 8: '8G' }"
              style="flex: 1"
            />
            <NButton
              size="small"
              :loading="busy === 'vram'"
              @click="save({ comfy_reserve_vram: reserveVRAM }, 'vram')"
            >
              保存
            </NButton>
          </div>
        </SettingRow>
      </div>

      <!-- ── 目录 ── -->
      <p class="dt-label sec">目录</p>
      <div class="rows">
        <SettingRow icon="folder" title="程序根目录" desc="相对路径都相对它解析" stack>
          <p class="path dt-mono dt-faint">{{ cfg.paths.root }}</p>
        </SettingRow>
        <SettingRow icon="folder" title="素材输出" desc="每套材质一个子目录，内含 manifest.json" stack>
          <p class="path dt-mono dt-faint">{{ cfg.paths.output }}</p>
        </SettingRow>
        <SettingRow icon="folder" title="工作流" desc="风格预设就是这里的一对 json：节点图 + 参数声明" stack>
          <p class="path dt-mono dt-faint">{{ cfg.paths.workflows }}</p>
        </SettingRow>
        <SettingRow icon="folder" title="配置文件" desc="设置页的改动都写回这个文件" stack>
          <p class="path dt-mono dt-faint">{{ cfg.config_path }}</p>
        </SettingRow>
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
.sec {
  margin: 26px 0 4px;
}
.secnote {
  margin: 0 0 10px;
  font-size: var(--dt-fs-sm);
  line-height: 1.65;
}
.rows {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.line {
  display: flex;
  align-items: center;
  gap: 10px;
}
.line :deep(.n-input) {
  flex: 1;
  min-width: 0;
}
.now {
  font-size: var(--dt-fs-xs);
  white-space: nowrap;
}
.slide {
  display: grid;
  grid-template-columns: 160px 92px auto;
  gap: 12px;
  align-items: center;
}
.pick {
  width: 210px;
}
.modelnote {
  margin: 6px 0 0;
  font-size: var(--dt-fs-xs);
  line-height: 1.6;
}
.path {
  margin: 0;
  font-size: var(--dt-fs-sm);
  word-break: break-all;
}
</style>
