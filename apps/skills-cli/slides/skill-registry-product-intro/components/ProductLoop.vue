<template>
  <div class="loop-shell">
    <div class="loop-center">Skill Registry</div>
    <div class="loop-item top">Web</div>
    <div class="loop-item left">CLI</div>
    <div class="loop-item right">Catalog</div>
    <div class="loop-item bottom">npm-compatible registry</div>
  </div>
</template>

<style scoped>
.loop-shell {
  display: grid;
  grid-template-areas:
    ". top ."
    "left center right"
    ". bottom .";
  grid-template-columns: 1fr 1.2fr 1fr;
  grid-template-rows: auto 1fr auto;
  gap: 1rem;
  min-height: 18rem;
  place-items: center;
}

.loop-center,
.loop-item {
  align-items: center;
  background: color-mix(in srgb, var(--sr-surface-strong) 94%, transparent);
  border: 1px solid var(--sr-border);
  border-radius: 24px;
  box-shadow: 0 18px 45px rgba(20, 33, 61, 0.08);
  color: var(--sr-ink);
  display: flex;
  font-weight: 700;
  justify-content: center;
  padding: 1rem 1.1rem;
  text-align: center;
}

.loop-center {
  background: linear-gradient(180deg, var(--sr-accent) 0%, var(--sr-accent-soft) 100%);
  grid-area: center;
}

.top {
  grid-area: top;
}

.left {
  grid-area: left;
}

.right {
  grid-area: right;
}

.bottom {
  grid-area: bottom;
}
</style>
