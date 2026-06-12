<script setup lang="ts">
const steps = [
  'Create',
  'Inspect',
  'Publish',
  'Release',
  'Search',
  'Install',
  'Update',
  'Govern',
];
</script>

<template>
  <div class="lifecycle-flow">
    <template v-for="(step, index) in steps" :key="step">
      <div class="lifecycle-step">
        <span>{{ step }}</span>
      </div>
      <span v-if="index < steps.length - 1" class="lifecycle-arrow" aria-hidden="true">
        →
      </span>
    </template>
  </div>
</template>

<style scoped>
.lifecycle-flow {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.lifecycle-step {
  align-items: center;
  background: linear-gradient(
    180deg,
    var(--sr-surface-strong) 0%,
    var(--sr-accent-soft) 100%
  );
  border: 1px solid color-mix(in srgb, var(--sr-accent) 28%, transparent);
  border-radius: 999px;
  color: var(--sr-ink);
  display: flex;
  font-size: 0.92rem;
  font-weight: 700;
  justify-content: center;
  min-height: 3.25rem;
  min-width: 7.5rem;
  padding: 0 1rem;
}

.lifecycle-arrow {
  color: color-mix(in srgb, var(--sr-ink) 35%, transparent);
  font-size: 1.1rem;
  font-weight: 700;
}
</style>
