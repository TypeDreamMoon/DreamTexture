<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import { NConfigProvider, NMessageProvider, NDialogProvider, NTooltip, darkTheme, zhCN, dateZhCN } from 'naive-ui'
import DtIcon from './components/DtIcon.vue'
import { darkOverrides, lightOverrides, applyCSSVars } from './theme'
import { bootstrap, connectEvents, health, wsConnected, activeJobs, activeDownloads } from './store'
import { migrateSidebar, persisted, readRaw } from './persist'

// 主题：没选过就跟随系统，选过就以用户的选择为准。
//
// 用 null 表示"还没表过态"，不能用 false 顶替：那样一个白天用浅色的人
// 到了晚上系统切深色，界面反而不跟着变了。
const themePref = persisted<'dark' | 'light' | null>('theme', null, (v) =>
  v === null || v === 'dark' || v === 'light',
)
const systemDark = ref(matchMedia('(prefers-color-scheme: dark)').matches)
const dark = computed({
  get: () => (themePref.value === null ? systemDark.value : themePref.value === 'dark'),
  set: (v: boolean) => (themePref.value = v ? 'dark' : 'light'),
})
watch(dark, (v) => applyCSSVars(v), { immediate: true })

const systemTheme = matchMedia('(prefers-color-scheme: dark)')
const onSystemTheme = (e: MediaQueryListEvent) => (systemDark.value = e.matches)

const route = useRoute()
let disconnect: (() => void) | null = null

// 收起状态记在本地：这是个人偏好，每次开都要重收一遍很烦。
//
// 旧版本存的是 'collapsed' / 'open' 两个裸字符串，不是合法 JSON。就地转一次，
// 否则老用户的偏好会读不出来，而 userSet 又因为"确实存过"而是 true——
// 结果是既没恢复成收起，窄窗口自动收起也不再生效，两头落空。
migrateSidebar()
const collapsed = persisted<boolean>('sidebar', false)

// 用户自己表过态没有。存过就说明表过，从此以他的选择为准。
//
// 这一行是必须的：否则 onMounted 里的 onResize 会立刻按窗口宽度把刚读回来的
// 偏好覆盖掉，"记住收起状态"就成了摆设——而且没有任何报错，只会让人觉得
// "我明明收起来了怎么每次打开又是展开的"。
const userSet = ref(readRaw('sidebar') !== null)

// 窄窗口自动收起，但只在用户还没表过态时。
const NARROW = 1100
function onResize() {
  if (userSet.value) return
  collapsed.value = window.innerWidth < NARROW
}
function toggle() {
  userSet.value = true
  collapsed.value = !collapsed.value
}

onMounted(async () => {
  onResize()
  window.addEventListener('resize', onResize)
  systemTheme.addEventListener('change', onSystemTheme)
  await bootstrap()
  disconnect = connectEvents()
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  systemTheme.removeEventListener('change', onSystemTheme)
  disconnect?.()
})

// 上面是干活的地方，下面是看和调的地方。
//
// 分成两组而不是排成一列：这五项是每天都要点的，另两项是偶尔进去看一眼、
// 调一下的。挤在一起的话，最常用的入口反而被稀释了。放到底部也顺手填掉了
// 侧栏中间那块空白。
const NAV = [
  { to: '/generate', label: '生成台', icon: 'wand' },
  { to: '/library', label: '素材库', icon: 'grid' },
  { to: '/workflows', label: '工作流', icon: 'flow' },
  { to: '/nodes', label: '节点', icon: 'chip' },
  { to: '/models', label: '模型', icon: 'layers' },
]
const NAV_FOOT = [
  { to: '/console', label: '控制台', icon: 'terminal' },
  { to: '/settings', label: '设置', icon: 'sliders' },
]

// ComfyUI 的地址由后端给（可能是 attach 模式下的远端实例），不要写死。
const comfyURL = computed(() => health.value?.base_url ?? 'http://127.0.0.1:8188')

// 角标显示各页正在进行的事情数量，切走了也能知道后台还在跑。
function badgeOf(to: string): number {
  if (to === '/generate') return activeJobs.value.length
  if (to === '/models') return activeDownloads.value.length
  return 0
}

// 状态点只有三种含义：能干活、连得上但没准备好、连不上。
const statusTone = computed(() => {
  const h = health.value
  if (!h || !h.alive) return { color: 'var(--dt-danger)', text: 'ComfyUI 未连接' }
  if (!h.ready) return { color: 'var(--dt-warn)', text: 'ComfyUI 忙' }
  return { color: 'var(--dt-ok)', text: 'ComfyUI 就绪' }
})

const statusDetail = computed(() => {
  const h = health.value
  if (!h) return ''
  if (h.reason) return h.reason
  const parts: string[] = []
  if (h.comfyui_version) parts.push(`ComfyUI ${h.comfyui_version}`)
  if (h.vram_free_mb && h.vram_total_mb) {
    parts.push(`显存 ${(h.vram_free_mb / 1024).toFixed(1)} / ${(h.vram_total_mb / 1024).toFixed(1)} GB 可用`)
  }
  if (h.queue_depth) parts.push(`队列 ${h.queue_depth}`)
  if (h.restarts) parts.push(`已重启 ${h.restarts} 次`)
  return parts.join(' · ')
})
</script>

<template>
  <NConfigProvider
    :theme="dark ? darkTheme : null"
    :theme-overrides="dark ? darkOverrides : lightOverrides"
    :locale="zhCN"
    :date-locale="dateZhCN"
  >
    <NMessageProvider :max="3">
      <NDialogProvider>
        <div class="shell" :class="{ collapsed }">
          <aside class="side">
            <RouterLink to="/generate" class="brand" :title="collapsed ? 'DreamTexture' : undefined">
              <span class="mark">DT</span>
              <span class="bname">DreamTexture</span>
            </RouterLink>

            <!-- 始终套着 tooltip，靠 disabled 开关，而不是 v-if 写两份链接：
                 两份复制品迟早会走样，而且切换收起时整个链接会被销毁重建。 -->
            <nav class="nav">
              <NTooltip
                v-for="n in NAV"
                :key="n.to"
                placement="right"
                :delay="200"
                :disabled="!collapsed"
              >
                <template #trigger>
                  <RouterLink :to="n.to" class="navlink" :class="{ on: route.path.startsWith(n.to) }">
                    <span class="nico"><DtIcon :name="n.icon" :size="17" /></span>
                    <span class="nlabel">{{ n.label }}</span>
                    <span v-if="badgeOf(n.to)" class="badge dt-mono">{{ badgeOf(n.to) }}</span>
                  </RouterLink>
                </template>
                {{ n.label }}
              </NTooltip>
            </nav>

            <div class="foot">
              <nav class="nav">
                <NTooltip
                  v-for="n in NAV_FOOT"
                  :key="n.to"
                  placement="right"
                  :delay="200"
                  :disabled="!collapsed"
                >
                  <template #trigger>
                    <RouterLink :to="n.to" class="navlink" :class="{ on: route.path.startsWith(n.to) }">
                      <span class="nico"><DtIcon :name="n.icon" :size="17" /></span>
                      <span class="nlabel">{{ n.label }}</span>
                    </RouterLink>
                  </template>
                  {{ n.label }}
                </NTooltip>
              </nav>

              <hr class="rule" />
              <!-- 状态这条即使展开也留着 tooltip：展开时显示的是简短状态，
                   悬停才给出版本、显存、队列这些细节。 -->
              <NTooltip placement="right" :delay="200">
                <template #trigger>
                  <RouterLink to="/console" class="status">
                    <i class="dot" :style="{ background: statusTone.color }" />
                    <span class="stext dt-faint">{{ statusTone.text }}</span>
                  </RouterLink>
                </template>
                {{ collapsed ? statusTone.text : '' }}{{ statusDetail ? (collapsed ? ' · ' : '') + statusDetail : '' }}
              </NTooltip>

              <p v-if="!wsConnected && !collapsed" class="offline dt-faint">事件流断开</p>

              <div class="tools">
                <a
                  :href="comfyURL"
                  target="_blank"
                  rel="noreferrer"
                  class="tool"
                  title="在新标签页打开 ComfyUI"
                >
                  <DtIcon name="globe" :size="15" />
                  <span class="tlabel">ComfyUI</span>
                </a>
                <button class="tool" :title="dark ? '切到浅色' : '切到深色'" @click="dark = !dark">
                  <DtIcon name="gauge" :size="15" />
                  <span class="tlabel">{{ dark ? '浅色' : '深色' }}</span>
                </button>
              </div>

              <button
                class="collapse"
                :title="collapsed ? '展开侧边栏' : '收起侧边栏'"
                @click="toggle"
              >
                <DtIcon name="panelLeft" :size="16" />
                <span class="tlabel">收起</span>
              </button>
            </div>
          </aside>

          <main class="body">
            <RouterView v-slot="{ Component }">
              <Transition name="page" mode="out-in">
                <component :is="Component" :key="route.path" />
              </Transition>
            </RouterView>
          </main>
        </div>
      </NDialogProvider>
    </NMessageProvider>
  </NConfigProvider>
</template>

<style scoped>
/* 左栏固定不滚，右侧内容自己滚。整页只有 .body 一个滚动容器。 */
/* 高度用 100vh 而不是 100%。
   .shell 外面套着三层 naive-ui 的 Provider，只要其中任何一层没有撑满高度，
   height:100% 就会落空——侧栏塌成内容高度，控制台那个 100vh 再一叠就把
   整页顶出屏幕。视口单位不依赖祖先，没有这个隐患。 */
.shell {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  height: 100vh;
  overflow: hidden;
}

/* 宽度过渡放在 .side 自己的 width 上，不要去过渡 grid-template-columns。
   曾经写成 `grid-template-columns: var(--side-w) ...` 再改变量，结果宽度
   总是慢一拍：类已经切了、宽度还停在上一态。自定义属性的变更与依赖它的
   属性过渡解析次序对不上，过渡两端拿到的是同一个旧值。
   直接过渡一个普通的 width 就没这问题。 */
.side {
  display: flex;
  flex-direction: column;
  width: 210px;
  height: 100%;
  min-height: 0;
  padding: 12px 10px;
  gap: 6px;
  background: var(--dt-surface);
  border-right: 1px solid var(--dt-border);
  overflow: hidden;
  transition: width 0.22s cubic-bezier(0.22, 0.8, 0.3, 1);
}
.shell.collapsed .side {
  width: 60px;
}
/* 侧边栏不用毛玻璃：backdrop-filter 要底下真有东西滑过才成立，
   而固定不动的左栏后面什么都不会经过，只会变成一块半透明色。 */

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px 12px;
  color: inherit;
  text-decoration: none;
  white-space: nowrap;
}
.mark {
  display: grid;
  place-items: center;
  flex: none;
  width: 26px;
  height: 26px;
  border-radius: 7px;
  background: var(--dt-accent);
  color: var(--dt-bg);
  font-size: var(--dt-fs-xs);
  font-weight: 700;
  letter-spacing: 0.04em;
  font-family: 'JetBrains Mono', Consolas, monospace;
  transition: transform 0.25s cubic-bezier(0.22, 0.8, 0.3, 1);
}
.brand:hover .mark {
  transform: rotate(-8deg) scale(1.06);
}
.bname {
  font-weight: 500;
  letter-spacing: 0.01em;
  font-size: var(--dt-fs-md);
}

/* 四处会被收掉的文字统一处理：宽度收到 0 时必须自己裁掉，
   否则文字照样溢出来——外层的 overflow:hidden 只是在最后一刻挡住它，
   宽度过渡的那 0.22 秒里还是会看见字压在图标上。 */
.bname,
.nlabel,
.stext,
.tlabel {
  min-width: 0;
  overflow: hidden;
  transition:
    opacity 0.16s ease,
    width 0.22s cubic-bezier(0.22, 0.8, 0.3, 1);
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-height: 0;
  overflow-y: auto;
}

.navlink {
  position: relative;
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 8px 9px;
  border-radius: var(--dt-radius);
  color: var(--dt-ink-muted);
  text-decoration: none;
  font-size: var(--dt-fs-base);
  white-space: nowrap;
  transition:
    color 0.16s ease,
    background 0.16s ease;
}
.navlink:hover {
  color: var(--dt-ink);
  background: var(--dt-surface2);
}
.navlink.on {
  color: var(--dt-accent);
  background: var(--dt-accent-soft);
  font-weight: 500;
}
.nico {
  display: grid;
  place-items: center;
  flex: none;
  width: 22px;
}
.nlabel {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}
.badge {
  flex: none;
  min-width: 18px;
  padding: 0 5px;
  text-align: center;
  font-size: var(--dt-fs-xs);
  line-height: 17px;
  border-radius: 9px;
  background: var(--dt-accent);
  color: var(--dt-bg);
  animation: dt-pulse 1.8s ease-in-out infinite;
}

/* 底部这一组：控制台/设置 用和上面完全一样的导航样式，下面一条细线，
   再往下是状态与开关。上面那组 min-height:0 + auto 会把剩余空间吃掉，
   这一组自然被顶到最底。 */
.foot {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.rule {
  margin: 6px 4px;
  border: none;
  border-top: 1px solid var(--dt-border);
}
.status {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 7px 9px;
  border-radius: var(--dt-radius);
  text-decoration: none;
  color: inherit;
  white-space: nowrap;
  transition: background 0.16s ease;
}
.status:hover {
  background: var(--dt-surface2);
}
.dot {
  flex: none;
  /* 收起时这个点就是唯一的状态指示，所以让它落在图标的位置上 */
  width: 8px;
  height: 8px;
  margin: 0 7px;
  display: block;
  border-radius: 50%;
  transition: background 0.3s ease;
}
.stext {
  font-size: var(--dt-fs-sm);
  overflow: hidden;
}
.offline {
  margin: 0 0 2px 9px;
  font-size: var(--dt-fs-xs);
  color: var(--dt-warn);
}

.tools {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.tool,
.collapse {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 7px 9px;
  border-radius: var(--dt-radius);
  font: inherit;
  font-size: var(--dt-fs-sm);
  color: var(--dt-ink-faint);
  background: none;
  border: none;
  text-decoration: none;
  cursor: pointer;
  white-space: nowrap;
  transition:
    color 0.16s ease,
    background 0.16s ease;
}
.tool:hover,
.collapse:hover {
  color: var(--dt-ink);
  background: var(--dt-surface2);
}
.tool :deep(.dt-icon),
.collapse :deep(.dt-icon) {
  /* 和导航图标同一条竖线上，收起时才不会参差不齐 */
  width: 22px;
}
.tlabel {
  overflow: hidden;
}
.collapse {
  margin-top: 2px;
}
.shell.collapsed .collapse :deep(.dt-icon) {
  transform: scaleX(-1);
}

/* 收起：只留图标。标签用 opacity+width 收掉而不是 display:none，
   这样宽度过渡时文字是淡出的，不会在中途换行抖一下。 */
.shell.collapsed .bname,
.shell.collapsed .nlabel,
.shell.collapsed .stext,
.shell.collapsed .tlabel {
  opacity: 0;
  width: 0;
  flex: none;
}
.shell.collapsed .badge {
  /* 收起时角标叠在图标右上角 */
  position: absolute;
  top: 3px;
  right: 3px;
  min-width: 15px;
  padding: 0 4px;
  line-height: 14px;
  font-size: 9.5px;
}
.body {
  min-width: 0;
  height: 100%;
  overflow: auto;
}

@media (prefers-reduced-motion: reduce) {
  .side,
  .bname,
  .nlabel,
  .stext,
  .tlabel {
    transition: none;
  }
}
</style>
