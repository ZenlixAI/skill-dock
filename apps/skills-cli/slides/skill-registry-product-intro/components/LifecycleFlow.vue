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
    <div v-for="step in steps" :key="step" class="lifecycle-step">
      <span>{{ step }}</span>
    </div>
  </div>
</template>

<style scoped>
.lifecycle-flow {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.lifecycle-step {
  align-items: center;
  background: linear-gradient(180deg, #ffffff 0%, #fff5dc 100%);
  border: 1px solid rgba(252, 163, 17, 0.28);
  border-radius: 999px;
  color: #14213d;
  display: flex;
  font-size: 0.92rem;
  font-weight: 700;
  justify-content: center;
  min-height: 3.25rem;
  position: relative;
}

.lifecycle-step::after {
  color: rgba(20, 33, 61, 0.3);
  content: '→';
  position: absolute;
  right: -0.52rem;
  top: 50%;
  transform: translateY(-50%);
}

.lifecycle-step:nth-child(4)::after,
.lifecycle-step:last-child::after {
  content: '';
}
</style>
