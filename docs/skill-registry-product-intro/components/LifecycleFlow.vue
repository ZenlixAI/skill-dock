<script setup lang="ts">
const steps = [
  'Create',
  'Inspect',
  'Publish',
  'Release',
  'Search',
  'Install',
  'Update',
  'Remove',
  'Govern',
];
</script>

<template>
  <ol class="lifecycle-flow">
    <li
      v-for="(step, index) in steps"
      :key="step"
      class="lifecycle-step"
      :data-step="index + 1"
    >
      <span>{{ step }}</span>
    </li>
  </ol>
</template>

<style scoped>
.lifecycle-flow {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  list-style: none;
  margin: 0;
  padding: 0;
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
  gap: 0.75rem;
  justify-content: flex-start;
  min-height: 3.25rem;
  padding: 0.85rem 1rem;
  position: relative;
}

.lifecycle-step::before {
  align-items: center;
  background: rgba(20, 33, 61, 0.08);
  border-radius: 999px;
  content: attr(data-step);
  display: inline-flex;
  flex: 0 0 auto;
  height: 1.85rem;
  justify-content: center;
  width: 1.85rem;
}

.lifecycle-step:not(:last-child)::after {
  color: color-mix(in srgb, var(--sr-ink) 35%, transparent);
  content: '→';
  font-size: 1.1rem;
  font-weight: 700;
  position: absolute;
  right: 0.85rem;
  top: 50%;
  transform: translateY(-50%);
}

.lifecycle-step:nth-child(3n):not(:last-child) {
  justify-content: center;
  padding-bottom: 1.55rem;
}

.lifecycle-step:nth-child(3n):not(:last-child)::after {
  bottom: 0.45rem;
  content: '↓';
  left: 50%;
  right: auto;
  top: auto;
  transform: translateX(-50%);
}
</style>
