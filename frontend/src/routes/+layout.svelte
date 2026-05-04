<script lang="ts">
  import { browser } from '$app/environment';
  import { userStore, currentRoomStore, errorStore, clearError } from '$lib/stores';
  import { auth } from '$lib/api/client';
  import { onMount } from 'svelte';
  import type { Snippet } from 'svelte';

  interface Props {
    children: Snippet;
  }

  let { children }: Props = $props();

  let loading = $state(true);

  onMount(async () => {
    if (!browser) return;
    
    const token = localStorage.getItem('tomatogether_token');
    if (token) {
      try {
        const user = await auth.me();
        userStore.set(user);
      } catch (e) {
        localStorage.removeItem('tomatogether_token');
        userStore.set(null);
      }
    }
    loading = false;
  });

  function logout() {
    auth.logout();
    userStore.set(null);
    currentRoomStore.set(null);
  }

  function dismissError() {
    errorStore.set(null);
  }
</script>

<svelte:head>
  <title>TomatoTogether - 陪伴式番茄钟</title>
</svelte:head>

<div class="app">
  <header class="header">
    <div class="logo">
      🍅 TomatoTogether
    </div>
    
    <nav class="nav">
      {#if $userStore}
        <span class="user">{$userStore.username}</span>
        <button class="btn-link" onclick={logout}>退出</button>
      {:else}
        <a href="/login">登录</a>
        <a href="/register">注册</a>
      {/if}
    </nav>
  </header>

  <main class="main">
    {@render children()}
  </main>

  {#if $errorStore}
    <div class="error-toast" onclick={dismissError} onkeydown={(e) => e.key === 'Enter' && dismissError()} role="button" tabindex="0">
      {$errorStore}
    </div>
  {/if}
</div>

<style>
  .app {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 2rem;
    background: var(--color-bg-secondary);
    border-bottom: 1px solid var(--color-border);
  }

  .logo {
    font-size: 1.5rem;
    font-weight: bold;
    color: var(--color-primary);
  }

  .nav {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .user {
    color: var(--color-text);
  }

  .nav a, .nav button {
    color: var(--color-text-secondary);
    text-decoration: none;
    background: none;
    border: none;
    cursor: pointer;
    font-size: 1rem;
  }

  .nav a:hover, .nav button:hover {
    color: var(--color-primary);
  }

  .btn-link {
    padding: 0;
  }

  .main {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 2rem;
  }

  .error-toast {
    position: fixed;
    bottom: 2rem;
    left: 50%;
    transform: translateX(-50%);
    background: var(--color-danger);
    color: white;
    padding: 1rem 2rem;
    border-radius: 0.5rem;
    cursor: pointer;
    z-index: 1000;
  }

  :global(:root) {
    --color-primary: #ff6b6b;
    --color-bg: #1a1a2e;
    --color-bg-secondary: #16213e;
    --color-text: #eaeaea;
    --color-text-secondary: #a0a0a0;
    --color-border: #333;
    --color-danger: #f44336;
  }

  @media (prefers-color-scheme: light) {
    :global(:root) {
      --color-bg: #f5f5f5;
      --color-bg-secondary: #ffffff;
      --color-text: #333;
      --color-text-secondary: #666;
      --color-border: #ddd;
    }
  }

  :global(body) {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: var(--color-bg);
    color: var(--color-text);
  }
</style>