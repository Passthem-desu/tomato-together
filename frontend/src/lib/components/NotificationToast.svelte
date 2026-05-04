<script lang="ts">
  import { notificationsStore } from '$lib/stores';

  let visible = $state(false);
  let currentNotification = $state<{ type: string; title: string; body: string } | null>(null);

  $effect(() => {
    if ($notificationsStore.length > 0 && !visible) {
      showNext();
    }
  });

  function showNext() {
    if ($notificationsStore.length === 0) {
      visible = false;
      currentNotification = null;
      return;
    }

    currentNotification = $notificationsStore[0];
    visible = true;

    setTimeout(() => {
      notificationsStore.update(n => n.slice(1));
      visible = false;
      currentNotification = null;
      
      setTimeout(() => showNext(), 300);
    }, 4000);
  }

  function dismiss() {
    visible = false;
    notificationsStore.update(n => n.slice(1));
    currentNotification = null;
  }
</script>

{#if visible && currentNotification}
  <div class="notification" class:visible role="alert">
    <div class="notification-icon">
      {#if currentNotification.type === 'announcement'}
        📢
      {:else if currentNotification.type === 'leader_aborted'}
        ⚠️
      {:else if currentNotification.type === 'session_end'}
        🍅
      {:else}
        🔔
      {/if}
    </div>
    <div class="notification-content">
      <div class="notification-title">{currentNotification.title}</div>
      <div class="notification-body">{currentNotification.body}</div>
    </div>
    <button class="notification-close" onclick={dismiss} aria-label="关闭">×</button>
  </div>
{/if}

<style>
  .notification {
    position: fixed;
    bottom: 2rem;
    right: 2rem;
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    padding: 1rem;
    background: var(--color-bg-secondary);
    border-radius: 0.5rem;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    max-width: 320px;
    transform: translateY(120%);
    opacity: 0;
    transition: all 0.3s ease-out;
    z-index: 1000;
  }

  .notification.visible {
    transform: translateY(0);
    opacity: 1;
  }

  .notification-icon {
    font-size: 1.5rem;
    flex-shrink: 0;
  }

  .notification-content {
    flex: 1;
    min-width: 0;
  }

  .notification-title {
    font-weight: 600;
    color: var(--color-text);
    margin-bottom: 0.25rem;
  }

  .notification-body {
    font-size: 0.875rem;
    color: var(--color-text-secondary);
  }

  .notification-close {
    background: none;
    border: none;
    font-size: 1.25rem;
    color: var(--color-text-secondary);
    cursor: pointer;
    padding: 0;
    line-height: 1;
  }

  .notification-close:hover {
    color: var(--color-text);
  }

  :global(:root) {
    --color-bg-secondary: #16213e;
    --color-text: #eaeaea;
    --color-text-secondary: #a0a0a0;
  }

  @media (prefers-color-scheme: light) {
    :global(:root) {
      --color-bg-secondary: #ffffff;
      --color-text: #333;
      --color-text-secondary: #666;
    }
  }
</style>