<script setup lang="ts">
defineProps<{ body: string; generatedAt: string; evidenceCount: number }>()
const emit = defineEmits<{ evidence: [] }>()
</script>

<template>
  <section
    class="ai-summary"
    aria-label="AI 每日摘要"
  >
    <AIResultBadge />
    <div class="ai-summary-title">
      今日速览
    </div>
    <div class="ai-summary-body">
      {{ body }}
    </div>
    <div class="ai-meta">
      <span>生成于 {{ generatedAt.slice(11, 16) }}</span>
      <span>基于 {{ evidenceCount }} 条记录</span>
      <button
        class="ai-evidence-link"
        @click="emit('evidence')"
      >
        查看依据
      </button>
    </div>
    <div class="ai-disclaimer">
      以上由 AI 整理，不构成医疗诊断。如有异常请咨询兽医。
    </div>
  </section>
</template>

<style scoped>
.ai-summary {
  background: var(--color-ai-bg);
  border: 1px solid var(--color-ai-border);
  border-radius: var(--radius-md);
  padding: var(--space-12);
  margin-bottom: var(--space-16);
}
.ai-summary-title { font-size: var(--font-size-body-lg); font-weight: 600; margin-top: 8px; color: var(--color-ai-text); }
.ai-summary-body { font-size: var(--font-size-body); margin-top: 8px; color: var(--color-text-secondary); line-height: 1.6; }
.ai-meta { display: flex; flex-wrap: wrap; gap: var(--space-8); margin-top: 8px; font-size: var(--font-size-caption); color: var(--color-text-tertiary); }
.ai-evidence-link { color: var(--color-ai-accent); cursor: pointer; text-decoration: underline; }
.ai-disclaimer { margin-top: 8px; font-size: var(--font-size-caption); color: var(--color-text-tertiary); line-height: 1.5; }
</style>
