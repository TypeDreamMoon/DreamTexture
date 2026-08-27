<script setup lang="ts">
import DtIcon from './DtIcon.vue'

// 设置页的一行：左边图标 + 标题 + 说明，右边控件。
//
// 说明文字是必填的。设置项最常见的失败不是控件难用，而是用户看着标题
// 猜不出改了会怎样，于是不敢动——一句话解释比什么控件设计都管用。
withDefaults(
  defineProps<{
    icon: string
    title: string
    desc: string
    /** 需要重启后端才生效时标出来，别让人改完以为已经生效了。 */
    restart?: boolean
    /** 纵向堆叠：控件太宽时（长路径输入框）横排会挤成一条缝。 */
    stack?: boolean
  }>(),
  { restart: false, stack: false },
)
</script>

<template>
  <div class="row dt-panel" :class="{ stack }">
    <div class="left">
      <span class="ico"><DtIcon :name="icon" /></span>
      <div class="text">
        <p class="title">
          {{ title }}
          <span v-if="restart" class="badge">需重启</span>
        </p>
        <p class="desc dt-faint">{{ desc }}</p>
      </div>
    </div>
    <div class="ctl"><slot /></div>
  </div>
</template>

<style scoped>
.row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 13px 16px;
}
.row.stack {
  flex-direction: column;
  align-items: stretch;
  gap: 10px;
}
.left {
  display: flex;
  align-items: center;
  gap: 13px;
  min-width: 0;
  flex: 1;
}
.ico {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--dt-surface2);
  color: var(--dt-ink-muted);
  flex: none;
}
.text {
  min-width: 0;
}
.title {
  margin: 0;
  font-size: var(--dt-fs-md);
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 8px;
}
.badge {
  font-size: var(--dt-fs-2xs);
  line-height: 1.7;
  padding: 0 6px;
  border-radius: 999px;
  color: var(--dt-warn);
  border: 1px solid var(--dt-warn);
  font-weight: 400;
}
.desc {
  margin: 3px 0 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.6;
}
.ctl {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
}
.row.stack .ctl {
  flex: 1;
  width: 100%;
}
</style>
