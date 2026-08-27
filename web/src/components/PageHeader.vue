<script setup lang="ts">
// 页头：标题 + 一句说明 + 右侧操作。
//
// 说明那一句不是凑数的。原先「节点」「模型」「工作流」几页都只有一个光秃秃的
// 标题，第一次进来的人得靠点开试试才知道这页是干什么的——尤其"工作流"和"节点"
// 在 ComfyUI 的语境里都容易望文生义。
withDefaults(defineProps<{ title: string; desc?: string }>(), { desc: '' })
</script>

<template>
  <header class="ph">
    <div class="txt">
      <h1>{{ title }}</h1>
      <p v-if="desc" class="desc dt-muted">{{ desc }}</p>
    </div>
    <div v-if="$slots.actions" class="acts"><slot name="actions" /></div>
  </header>
</template>

<style scoped>
.ph {
  display: flex;
  align-items: flex-start;
  gap: 18px;
  margin-bottom: 18px;
  padding-bottom: 14px;
  /* 一条细线把页头和内容分开。比单纯留白更能立住层级，
     又比整块底色轻。 */
  border-bottom: 1px solid var(--dt-border);
}
.txt {
  min-width: 0;
  flex: 1;
}
h1 {
  margin: 0;
  font-size: var(--dt-fs-lg);
  font-weight: 500;
  letter-spacing: 0.01em;
  line-height: 1.3;
}
.desc {
  margin: 5px 0 0;
  font-size: var(--dt-fs-sm);
  line-height: 1.65;
  max-width: 62ch;
}
.acts {
  flex: none;
  display: flex;
  align-items: center;
  gap: 9px;
  /* 和标题基线对齐，而不是和整个文本块居中——有说明文字时居中会让按钮
     莫名其妙地掉到下面去。 */
  padding-top: 2px;
}
</style>
