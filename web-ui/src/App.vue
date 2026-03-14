<template>
  <router-view />
</template>

<script setup lang="ts">
</script>

<style>
/* ── Base reset ─────────────────────────────────────────────────────────── */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

#app {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  -webkit-font-smoothing: antialiased;
  height: 100vh;
}

/* ── Clash-Verge-Rev inspired dark palette ──────────────────────────────── */
:root {
  --cv-bg:          #0f1117;
  --cv-surface:     #181c2a;
  --cv-surface2:    #1e2235;
  --cv-sidebar:     #13161f;
  --cv-border:      rgba(255, 255, 255, 0.07);
  --cv-text:        #e2e8f0;
  --cv-text-muted:  #64748b;
  --cv-accent:      #5865f2;
  --cv-accent-soft: rgba(88, 101, 242, 0.15);
  --cv-radius:      10px;
  --cv-radius-sm:   6px;
}

/* ── Override Element Plus dark vars ────────────────────────────────────── */
html.dark {
  --el-bg-color:           var(--cv-surface);
  --el-bg-color-page:      var(--cv-bg);
  --el-bg-color-overlay:   var(--cv-surface2);
  --el-border-color:       var(--cv-border);
  --el-border-color-light: var(--cv-border);
  --el-text-color-primary: var(--cv-text);
  --el-text-color-regular: #94a3b8;
  --el-text-color-secondary: var(--cv-text-muted);
  --el-color-primary:      var(--cv-accent);
  --el-fill-color:         var(--cv-surface2);
  --el-fill-color-light:   rgba(255,255,255,0.04);
  --el-fill-color-blank:   var(--cv-surface);
  --el-menu-bg-color:      transparent;
  --el-menu-hover-bg-color: var(--cv-accent-soft);
  --el-menu-active-color:  var(--cv-accent);
  --el-menu-text-color:    #94a3b8;
  --el-card-border-color:  var(--cv-border);
}

/* ── Card global style ──────────────────────────────────────────────────── */
html.dark .el-card {
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
  box-shadow: 0 2px 12px rgba(0,0,0,0.3);
}

html.dark .el-card__header {
  border-bottom: 1px solid var(--cv-border);
  padding: 14px 18px;
  font-weight: 600;
  font-size: 14px;
  color: var(--cv-text);
}

/* ── Table ──────────────────────────────────────────────────────────────── */
html.dark .el-table {
  background: var(--cv-surface);
  --el-table-bg-color: var(--cv-surface);
  --el-table-tr-bg-color: var(--cv-surface);
  --el-table-header-bg-color: var(--cv-surface2);
  --el-table-border-color: var(--cv-border);
}

html.dark .el-table__row:hover > td {
  background: var(--cv-surface2) !important;
}

/* ── Dialog ─────────────────────────────────────────────────────────────── */
html.dark .el-dialog {
  background: var(--cv-surface);
  border: 1px solid var(--cv-border);
  border-radius: var(--cv-radius);
}

html.dark .el-dialog__header {
  border-bottom: 1px solid var(--cv-border);
}

/* ── Input / Select ─────────────────────────────────────────────────────── */
html.dark .el-input__wrapper,
html.dark .el-textarea__inner {
  background: var(--cv-surface2);
  box-shadow: 0 0 0 1px var(--cv-border) inset;
}

html.dark .el-select__wrapper {
  background: var(--cv-surface2);
  box-shadow: 0 0 0 1px var(--cv-border) inset;
}

/* ── Scrollbar ──────────────────────────────────────────────────────────── */
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.2); }

/* ── Tabs ────────────────────────────────────────────────────────────────── */
html.dark .el-tabs__nav-wrap::after { background: var(--cv-border); }
html.dark .el-tabs__item { color: var(--cv-text-muted); }
html.dark .el-tabs__item.is-active { color: var(--cv-accent); }
html.dark .el-tabs__active-bar { background: var(--cv-accent); }
</style>
