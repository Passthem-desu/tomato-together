<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { 
    currentMember, 
    currentRoom, 
    roomUsers, 
    pomodoroStatus, 
    isLoading, 
    error,
    startPomodoro,
    endPomodoro,
    unfollowPomodoro,
    leaveRoom,
    refreshRoomUsers,
    logout
  } from '$lib/store';
  import { locale, t } from '$lib/i18n';

  let displayTime = $state(25 * 60);
  let timerInterval: number | null = null;
  let pollInterval: number | null = null;

  let plannedMinutes = $state(25);
  let restMinutes = $state(5);
  let longBreakMinutes = $state(15);

  onMount(() => {
    if (!$currentMember) {
      goto('/');
      return;
    }
    
    if ($pomodoroStatus.remaining_seconds !== undefined) {
      displayTime = $pomodoroStatus.remaining_seconds;
    }
    
    timerInterval = setInterval(() => {
      tickTimer();
    }, 1000) as unknown as number;
    
    refreshRoomUsers();
    pollInterval = setInterval(() => {
      refreshRoomUsers();
    }, 5000) as unknown as number;
    
    return () => {
      if (timerInterval) clearInterval(timerInterval);
      if (pollInterval) clearInterval(pollInterval);
    };
  });

  function tickTimer() {
    if ($pomodoroStatus.is_active && displayTime > 0) {
      displayTime = Math.max(0, displayTime - 1);
    }
  }

  function formatTime(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  }

  async function handleStart() {
    await startPomodoro({
      planned_duration: plannedMinutes * 60,
      rest_duration: restMinutes * 60,
      long_break_duration: longBreakMinutes * 60,
    });
    displayTime = plannedMinutes * 60;
  }

  async function handleEnd() {
    await endPomodoro(false);
  }

  async function handleLeave() {
    const msg = $locale === 'zh-hans' 
      ? '确定要离开房间吗？' 
      : $locale === 'zh-hant' 
        ? '確定要離開房間嗎？'
        : 'Are you sure you want to leave the room?';
    if (confirm(msg)) {
      await leaveRoom();
      goto('/');
    }
  }

  async function handleLogout() {
    const msg = $locale === 'zh-hans'
      ? '确定要退出登录吗？'
      : $locale === 'zh-hant'
        ? '確定要退出登入嗎？'
        : 'Are you sure you want to logout?';
    if (confirm(msg)) {
      logout();
      goto('/');
    }
  }

  function getStatusLabel(status: string) {
    switch (status) {
      case 'idle': return t('idle', $locale);
      case 'focusing': return t('focusing', $locale);
      case 'following': return `${t('following', $locale)} ${$pomodoroStatus.leader_username}`;
      case 'rest': return t('resting', $locale);
      default: return '';
    }
  }
</script>

<svelte:head>
  <title>{t('room_title', $locale)} - TomatoTogether</title>
</svelte:head>

<main class="room">
  <header class="room-header">
    <div class="header-left">
      <div class="room-avatar">🍅</div>
      <div class="room-info">
        <h1>{$currentRoom?.name || t('room_title', $locale)}</h1>
        <div class="room-meta">
          <span class="status-dot online"></span>
          <span class="member-count">{$roomUsers.length} {t('online', $locale)}</span>
        </div>
      </div>
    </div>
    <div class="header-right">
      <div class="user-badge">
        <span class="user-name">{$currentMember?.username}</span>
        {#if $currentMember?.is_owner}
          <span class="badge">{t('owner', $locale)}</span>
        {/if}
      </div>
      <button class="btn-ghost btn-sm" onclick={handleLogout}>
        {t('logout', $locale)}
      </button>
    </div>
  </header>

  <div class="room-content">
    <section class="timer-card card">
      <div class="timer-display">
        <span class="time">{formatTime(displayTime)}</span>
        <span class="status-label" class:focusing={$pomodoroStatus.status === 'focusing'} class:resting={$pomodoroStatus.status === 'rest'}>
          {getStatusLabel($pomodoroStatus.status)}
        </span>
      </div>

      {#if !$pomodoroStatus.is_active}
        <div class="timer-settings">
          <div class="setting">
            <label for="plannedMinutes">{t('focus_time', $locale)}</label>
            <input id="plannedMinutes" type="number" bind:value={plannedMinutes} min="1" max="60" />
          </div>
          <div class="setting">
            <label for="restMinutes">{t('short_break', $locale)}</label>
            <input id="restMinutes" type="number" bind:value={restMinutes} min="1" max="30" />
          </div>
          <div class="setting">
            <label for="longBreakMinutes">{t('long_break', $locale)}</label>
            <input id="longBreakMinutes" type="number" bind:value={longBreakMinutes} min="1" max="60" />
          </div>
        </div>
        <button class="btn-primary btn-lg btn-full" onclick={handleStart} disabled={$isLoading}>
          🍅 {t('start_pomodoro', $locale)}
        </button>
      {:else}
        <div class="timer-actions">
          <button class="btn-secondary" onclick={handleEnd} disabled={$isLoading}>
            {t('end_pomodoro', $locale)}
          </button>
          {#if $pomodoroStatus.status === 'following'}
            <button class="btn-ghost" onclick={unfollowPomodoro} disabled={$isLoading}>
              {t('unfollow', $locale)}
            </button>
          {/if}
        </div>
      {/if}

      {#if $error}
        <p class="form-error">{$error}</p>
      {/if}
    </section>

    <section class="users-card card">
      <h2>{t('online_users', $locale)}</h2>
      
      {#if $roomUsers.length === 0}
        <p class="empty-state">{t('no_online_users', $locale)}</p>
      {:else}
        <div class="users-list">
          {#each $roomUsers as user}
            <div class="user-item">
              <div class="user-info">
                <div class="user-avatar">
                  {user.is_owner ? '👑' : '👤'}
                </div>
                <div class="user-details">
                  <span class="user-name">
                    {user.username}
                    {#if user.id === $currentMember?.id}
                      <span class="you-badge">{t('me', $locale)}</span>
                    {/if}
                  </span>
                  {#if user.status}
                    <span class="user-status">{user.status.emoji} {user.status.message}</span>
                  {/if}
                </div>
              </div>
              <div class="user-pomodoro">
                {#if user.pomodoro?.is_active}
                  {#if user.pomodoro.is_following}
                    <span class="following">
                      🔄 {t('following', $locale)} {user.pomodoro.leader_username}
                    </span>
                  {:else}
                    <span class="active-pomodoro">
                      🍅 {user.pomodoro.remaining_seconds ? formatTime(user.pomodoro.remaining_seconds) : ''}
                    </span>
                  {/if}
                {:else}
                  <span class="idle-status">{t('idle', $locale)}</span>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </section>

    <section class="actions-section">
      <button class="btn-ghost" onclick={handleLeave}>
        ← {t('leave_room', $locale)}
      </button>
    </section>
  </div>
</main>

<style>
  .room {
    min-height: 100dvh;
    padding: 1.5rem;
    background: var(--color-bg-0);
  }

  /* Header */
  .room-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 0;
    margin-bottom: 2rem;
    border-bottom: 1px solid var(--color-border);
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 0.875rem;
  }

  .room-avatar {
    font-size: 2.5rem;
  }

  .room-info h1 {
    font-size: var(--text-xl);
    font-weight: 600;
    margin-bottom: 0.25rem;
  }

  .room-meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--color-fg-muted);
    font-size: var(--text-sm);
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: var(--radius-full);
    background: var(--color-fg-muted);
  }

  .status-dot.online {
    background: var(--color-success);
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .user-badge {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .user-badge .user-name {
    font-weight: 500;
  }

  .badge {
    display: inline-flex;
    padding: 0.25rem 0.5rem;
    font-size: var(--text-xs);
    font-weight: 500;
    border-radius: var(--radius-full);
    background: var(--color-brand-subtle);
    color: var(--color-brand);
  }

  /* Content */
  .room-content {
    max-width: 560px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  /* Timer Card */
  .timer-card {
    text-align: center;
    padding: 2rem;
  }

  .timer-display {
    margin-bottom: 1.5rem;
  }

  .time {
    font-size: 4.5rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    letter-spacing: -0.02em;
    display: block;
    margin-bottom: 0.5rem;
    background: linear-gradient(135deg, var(--color-fg-0), var(--color-brand));
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }

  .status-label {
    color: var(--color-fg-muted);
    font-size: var(--text-lg);
  }

  .status-label.focusing {
    color: var(--color-brand);
  }

  .status-label.resting {
    color: var(--color-success);
  }

  .timer-settings {
    display: flex;
    gap: 1rem;
    justify-content: center;
    margin-bottom: 1.5rem;
    flex-wrap: wrap;
  }

  .setting {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .setting label {
    font-size: var(--text-xs);
    color: var(--color-fg-muted);
    font-weight: 500;
  }

  .setting input {
    width: 70px;
    text-align: center;
    padding: 0.5rem;
  }

  .timer-actions {
    display: flex;
    gap: 0.75rem;
    justify-content: center;
  }

  /* Users Card */
  .users-card h2 {
    font-size: var(--text-base);
    font-weight: 600;
    margin-bottom: 1rem;
    color: var(--color-fg-1);
  }

  .empty-state {
    text-align: center;
    color: var(--color-fg-muted);
    padding: 1.5rem;
  }

  .users-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .user-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem;
    background: var(--color-bg-0);
    border-radius: var(--radius-md);
    transition: background var(--duration-fast) var(--ease-out);
  }

  .user-item:hover {
    background: var(--color-bg-2);
  }

  .user-info {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .user-avatar {
    font-size: 1.25rem;
    width: 2rem;
    text-align: center;
  }

  .user-details {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .user-details .user-name {
    font-weight: 500;
    display: flex;
    align-items: center;
    gap: 0.375rem;
  }

  .you-badge {
    font-size: var(--text-xs);
    padding: 0.125rem 0.375rem;
    background: var(--color-bg-2);
    border-radius: var(--radius-sm);
    color: var(--color-fg-muted);
  }

  .user-status {
    font-size: var(--text-xs);
    color: var(--color-fg-muted);
  }

  .user-pomodoro .active-pomodoro {
    font-weight: 600;
    color: var(--color-brand);
  }

  .user-pomodoro .following {
    font-size: var(--text-sm);
    color: var(--color-fg-muted);
  }

  .user-pomodoro .idle-status {
    font-size: var(--text-sm);
    color: var(--color-fg-muted);
  }

  /* Actions */
  .actions-section {
    text-align: center;
    padding: 1rem 0;
  }

  .form-error {
    margin-top: 1rem;
    padding: 0.75rem;
    background: var(--color-error-subtle);
    color: var(--color-error);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
  }

  /* Responsive */
  @media (max-width: 640px) {
    .room {
      padding: 1rem;
    }

    .room-header {
      flex-direction: column;
      gap: 1rem;
      text-align: center;
    }

    .header-left {
      flex-direction: column;
    }

    .time {
      font-size: 3.5rem;
    }

    .timer-settings {
      flex-direction: column;
      align-items: center;
    }

    .timer-actions {
      flex-direction: column;
    }

    .user-item {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.5rem;
    }
  }
</style>