import type { GlobalThemeOverrides } from 'naive-ui'

// 浅色是暖纸调，深色是中性近黑 —— 这是刻意的不对称。
//
// 最初深色也用暖色打底（褐黑 #1A1611 配褐灰），结果整片发浑发脏：低饱和的暖色
// 铺满大面积时读起来像蒙了层灰。改成中性近黑之后，暖琥珀反而成了画面里唯一的
// 暖点，identity 更清楚，也干净得多。暖色留给强调色，不铺满底。

const palette = {
  light: {
    bg: '#F6F2E9',
    surface: '#FFFDF8',
    surface2: '#EFE9DC',
    glass: 'rgba(255, 253, 248, 0.72)',
    ink: '#24201B',
    inkMuted: '#6A6154',
    inkFaint: '#988C7C',
    border: '#E5DDCC',
    borderStrong: '#D3C8B2',
    accent: '#A96F0E',
    accentHover: '#C2811A',
    accentPressed: '#8D5B08',
    accentSoft: '#F5E9CF',
    ok: '#4E7030',
    warn: '#96601A',
    danger: '#A63E2B',
    shadow: 'rgba(60, 48, 30, 0.10)',
  },
  dark: {
    bg: '#0E0E11',
    surface: '#16161A',
    surface2: '#1E1E24',
    glass: 'rgba(20, 20, 24, 0.70)',
    ink: '#ECECF1',
    inkMuted: '#9E9EAC',
    inkFaint: '#6C6C7B',
    border: '#26262E',
    borderStrong: '#36363F',
    accent: '#F2B04A',
    accentHover: '#F7C169',
    accentPressed: '#D89838',
    accentSoft: '#2C2418',
    ok: '#7DC48D',
    warn: '#E8B45C',
    danger: '#F08A7A',
    shadow: 'rgba(0, 0, 0, 0.45)',
  },
}

type Palette = typeof palette.light

const FONT =
  '"Noto Sans SC", "PingFang SC", "Microsoft YaHei", -apple-system, BlinkMacSystemFont, sans-serif'
const MONO = '"JetBrains Mono", "Cascadia Mono", Consolas, monospace'

// 小圆角：控件 6px，卡片 10px。够软化边缘，又不至于变成气泡。
const R = { control: '6px', small: '4px', card: '10px' }

// 字号只留六档。
//
// 之前是散着写的，全局数下来有 13 种，其中 11 / 11.5 / 12 / 12.5 / 13 / 13.5
// 挤成一团——差半个像素的两档读起来不是层级，是没定过标准。收成六档之后每一步
// 都看得出来，写新界面时也不必再"大概比它小一点"地猜。
const FS = {
  '2xs': '10.5px', // 等宽微标签：标签块、参数键名
  xs: '11px', //     说明、元信息
  sm: '12px', //     次要正文、设置项描述
  base: '13px', //   正文、列表项
  md: '14px', //     强调、行标题
  lg: '20px', //     页标题
}

// 版面宽度也收成三档，并且一律居中。
//
// 原先各页各写各的（1100 / 1040 / 1000 / 900，还有两个干脆没写），而且都没
// 居中——宽屏上内容全贴在左边，右侧留一大片空白，看着就像没做完。
const W = {
  narrow: '900px', //  表单、设置：行宽超过这个就不好读了
  mid: '1120px', //    列表页
  wide: '1440px', //   网格、两栏
}

function build(c: Palette): GlobalThemeOverrides {
  return {
    common: {
      fontFamily: FONT,
      fontFamilyMono: MONO,
      borderRadius: R.control,
      borderRadiusSmall: R.small,
      primaryColor: c.accent,
      primaryColorHover: c.accentHover,
      primaryColorPressed: c.accentPressed,
      primaryColorSuppl: c.accentHover,
      successColor: c.ok,
      warningColor: c.warn,
      errorColor: c.danger,
      infoColor: c.accent,
      textColorBase: c.ink,
      textColor1: c.ink,
      textColor2: c.ink,
      textColor3: c.inkFaint,
      baseColor: c.surface,
      bodyColor: c.bg,
      cardColor: c.surface,
      modalColor: c.surface,
      popoverColor: c.surface,
      tableColor: c.surface,
      inputColor: c.surface,
      actionColor: c.surface2,
      hoverColor: c.surface2,
      borderColor: c.border,
      dividerColor: c.border,
      scrollbarColor: c.borderStrong,
      scrollbarColorHover: c.inkFaint,
    },
    Card: { borderColor: c.border, borderRadius: R.card, paddingMedium: '18px 20px' },
    Button: { borderRadiusMedium: R.control, fontWeight: '500' },
    Input: { borderRadius: R.control },
    Tag: { borderRadius: R.small },
    Progress: { railHeight: '6px' },
    Slider: { handleSize: '15px' },
    Layout: { siderColor: c.surface, headerColor: c.surface },
  }
}

export const lightOverrides = build(palette.light)
export const darkOverrides = build(palette.dark)

/** 把同一套色板同时给到 CSS 变量，自写样式与 Naive UI 组件才不会脱节。 */
export function applyCSSVars(dark: boolean) {
  const c = dark ? palette.dark : palette.light
  const root = document.documentElement
  for (const [k, v] of Object.entries(c)) {
    root.style.setProperty(`--dt-${k.replace(/[A-Z]/g, (m) => '-' + m.toLowerCase())}`, v)
  }
  root.style.setProperty('--dt-radius', R.control)
  root.style.setProperty('--dt-radius-card', R.card)
  for (const [k, v] of Object.entries(FS)) root.style.setProperty(`--dt-fs-${k}`, v)
  for (const [k, v] of Object.entries(W)) root.style.setProperty(`--dt-w-${k}`, v)
  root.style.colorScheme = dark ? 'dark' : 'light'
}
