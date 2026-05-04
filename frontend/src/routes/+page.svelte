<script lang="ts">
  import { userStore } from '$lib/stores';
  import RoomForm from '$lib/components/RoomForm.svelte';
  import AuthForm from '$lib/components/AuthForm.svelte';
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';
  import { onMount } from 'svelte';

  let showAuth = $state(false);

  onMount(() => {
    if (browser && $userStore) {
      const currentRoom = localStorage.getItem('current_room');
      if (currentRoom) {
        const room = JSON.parse(currentRoom);
        goto(`/room/${encodeURIComponent(room.name)}`);
      }
    }
  });
</script>

<div class="home">
  <div class="hero">
    <h1 class="title">🍅 TomatoTogether</h1>
    <p class="subtitle">和朋友一起专注，享受陪伴的番茄时光</p>
  </div>

  <div class="content">
    {#if showAuth}
      <AuthForm />
      <p class="switch-mode">
        已经有账号？
        <button onclick={() => showAuth = false}>返回</button>
      </p>
    {:else}
      <RoomForm />
      <p class="switch-mode">
        还没有账号？
        <button onclick={() => showAuth = true}>注册账号</button>
      </p>
    {/if}
  </div>

  <div class="features">
    <div class="feature">
      <span class="icon">👥</span>
      <h3>多人同步</h3>
      <p>和朋友一起开始番茄，相互陪伴</p>
    </div>
    <div class="feature">
      <span class="icon">🔔</span>
      <h3>实时通知</h3>
      <p>番茄结束、公告发布，第一时间知晓</p>
    </div>
    <div class="feature">
      <span class="icon">📊</span>
      <h3>专注统计</h3>
      <p>记录你的番茄历史，了解专注趋势</p>
    </div>
  </div>
</div>

<style>
  .home {
    text-align: center;
    max-width: 800px;
  }

  .hero {
    margin-bottom: 3rem;
  }

  .title {
    font-size: 3rem;
    color: var(--color-primary);
    margin: 0;
  }

  .subtitle {
    font-size: 1.25rem;
    color: var(--color-text-secondary);
    margin: 1rem 0 0 0;
  }

  .content {
    display: flex;
    flex-direction: column;
    align-items: center;
    margin-bottom: 3rem;
  }

  .switch-mode {
    margin-top: 1rem;
    color: var(--color-text-secondary);
  }

  .switch-mode button {
    background: none;
    border: none;
    color: var(--color-primary);
    cursor: pointer;
    text-decoration: underline;
  }

  .features {
    display: flex;
    justify-content: center;
    gap: 2rem;
    flex-wrap: wrap;
  }

  .feature {
    flex: 1;
    min-width: 200px;
    max-width: 250px;
    padding: 1.5rem;
    background: var(--color-bg-secondary);
    border-radius: 0.5rem;
  }

  .feature .icon {
    font-size: 2rem;
  }

  .feature h3 {
    margin: 0.5rem 0;
    color: var(--color-text);
  }

  .feature p {
    margin: 0;
    font-size: 0.875rem;
    color: var(--color-text-secondary);
  }

  :global(:root) {
    --color-primary: #ff6b6b;
    --color-bg: #1a1a2e;
    --color-bg-secondary: #16213e;
    --color-text: #eaeaea;
    --color-text-secondary: #a0a0a0;
  }

  @media (prefers-color-scheme: light) {
    :global(:root) {
      --color-bg: #f5f5f5;
      --color-bg-secondary: #ffffff;
      --color-text: #333;
      --color-text-secondary: #666;
    }
  }
</style>