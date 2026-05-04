<script lang="ts">
  import '../app.css';
  let { children } = $props();
</script>

<div class="app">
  {@render children()}
</div>

<style>
  .app {
    min-height: 100dvh;
    background: var(--color-bg-0);
  }
</style>