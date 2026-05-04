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

  let displayTime = $state(25 * 60); // 25 minutes in seconds
  let timerInterval: number | null = null;
  let pollInterval: number | null = null;

  // Timer settings (in minutes for UI)
  let plannedMinutes = $state(25);
  let restMinutes = $state(5);
  let longBreakMinutes = $state(15);

  onMount(() => {
    if (!$currentMember) {
      goto('/');
      return;
    }
    
    // Set initial display time from pomodoro status or default
    if ($pomodoroStatus.remaining_seconds !== undefined) {
      displayTime = $pomodoroStatus.remaining_seconds;
    }
    
    // Timer: tick every second
    timerInterval = setInterval(() => {
      tickTimer();
    }, 1000) as unknown as number;
    
    // Polling: refresh users every 5 seconds
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
    // Set initial display time
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
</script>

<svelte:head>
  <title>{t('room_title', $locale)} - TomatoTogether</title>
</svelte:head>

<main class="room">
  <header>
    <div class="header-left">
      <h1>{$currentRoom?.name || t('room_title', $locale)}</h1>
      <span class="member-count">{$roomUsers.length} {t('online', $locale)}</span>
    </div>
    <div class="header-right">
      <span class="username">{$currentMember?.username}</span>
      {#if $currentMember?.is_owner}
        <span class="badge owner">{t('owner', $locale)}</span>
      {/if}
      <button class="secondary" onclick={handleLogout}>{t('logout', $locale)}</button>
    </div>
  </header>

  <div class="content">
    <section class="pomodoro-section card">
      <div class="timer-display">
        <span class="time">{formatTime(displayTime)}</span>
        <span class="status">
          {#if $pomodoroStatus.status === 'idle'}
            {t('idle', $locale)}
          {:else if $pomodoroStatus.status === 'focusing'}
            {t('focusing', $locale)} 🎯
          {:else if $pomodoroStatus.status === 'following'}
            {t('following', $locale)} {$pomodoroStatus.leader_username}
          {:else if $pomodoroStatus.status === 'rest'}
            {t('resting', $locale)} ☕
          {/if}
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
        <button class="primary start-btn" onclick={handleStart} disabled={$isLoading}>
          {t('start_pomodoro', $locale)}
        </button>
      {:else}
        <button class="secondary end-btn" onclick={handleEnd} disabled={$isLoading}>
          {t('end_pomodoro', $locale)}
        </button>
        {#if $pomodoroStatus.status === 'following'}
          <button class="secondary unfollow-btn" onclick={unfollowPomodoro} disabled={$isLoading}>
            {t('unfollow', $locale)}
          </button>
        {/if}
      {/if}

      {#if $error}
        <p class="error">{$error}</p>
      {/if}
    </section>

    <section class="users-section card">
      <h2>{t('online_users', $locale)}</h2>
      {#if $roomUsers.length === 0}
        <p class="empty">{t('no_online_users', $locale)}</p>
      {:else}
        <div class="users-list">
          {#each $roomUsers as user}
            <div class="user-item">
              <div class="user-info">
                <span class="user-name">
                  {user.username}
                  {#if user.id === $currentMember?.id}
                    ({t('me', $locale)})
                  {/if}
                </span>
                {#if user.is_owner}
                  <span class="badge owner">{t('owner', $locale)}</span>
                {/if}
                {#if user.status}
                  <span class="user-status">
                    {user.status.emoji} {user.status.message}
                  </span>
                {/if}
              </div>
              <div class="user-pomodoro">
                {#if user.pomodoro?.is_active}
                  {#if user.pomodoro.is_following}
                    <span class="following">{t('following', $locale)} {user.pomodoro.leader_username}</span>
                  {:else}
                    <span class="active">🍅 {user.pomodoro.remaining_seconds ? formatTime(user.pomodoro.remaining_seconds) : ''}</span>
                  {/if}
                {:else}
                  <span class="idle">{t('idle', $locale)}</span>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </section>

    <section class="actions-section">
      <button class="secondary" onclick={handleLeave}>
        {t('leave_room', $locale)}
      </button>
    </section>
  </div>
</main>

<style>
  .room {
    min-height: 100vh;
    padding: 1rem;
  }

  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--color-border);
  }

  .header-left h1 {
    font-size: 1.5rem;
    margin-bottom: 0.25rem;
  }

  .member-count {
    color: var(--color-text-secondary);
    font-size: 0.875rem;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .username {
    font-weight: 500;
  }

  .badge {
    padding: 0.25rem 0.5rem;
    border-radius: 0.25rem;
    font-size: 0.75rem;
    font-weight: 500;
  }

  .badge.owner {
    background: var(--color-primary-light);
    color: var(--color-primary);
  }

  .content {
    max-width: 600px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 0.75rem;
    padding: 1.5rem;
  }

  .pomodoro-section {
    text-align: center;
  }

  .timer-display {
    margin-bottom: 1.5rem;
  }

  .timer-display .time {
    font-size: 4rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    display: block;
    margin-bottom: 0.5rem;
  }

  .timer-display .status {
    color: var(--color-text-secondary);
    font-size: 1.125rem;
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
    font-size: 0.75rem;
    color: var(--color-text-secondary);
  }

  .setting input {
    width: 80px;
    text-align: center;
  }

  .start-btn, .end-btn, .unfollow-btn {
    min-width: 150px;
  }

  .users-section h2 {
    margin-bottom: 1rem;
    font-size: 1.125rem;
  }

  .users-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .user-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem;
    background: var(--color-surface-hover);
    border-radius: 0.5rem;
  }

  .user-info {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .user-name {
    font-weight: 500;
  }

  .user-status {
    font-size: 0.875rem;
    color: var(--color-text-secondary);
  }

  .user-pomodoro .active {
    color: var(--color-primary);
    font-weight: 500;
  }

  .user-pomodoro .idle {
    color: var(--color-text-secondary);
    font-size: 0.875rem;
  }

  .user-pomodoro .following {
    color: var(--color-secondary);
    font-size: 0.875rem;
  }

  .empty {
    color: var(--color-text-secondary);
    text-align: center;
    padding: 1rem;
  }

  .actions-section {
    text-align: center;
  }

  .error {
    color: var(--color-error);
    background: var(--color-error-light);
    padding: 0.75rem;
    border-radius: 0.5rem;
    margin-top: 1rem;
  }

  button {
    padding: 0.5rem 1rem;
    border-radius: 0.5rem;
    font-size: 0.875rem;
    cursor: pointer;
    border: none;
    transition: background-color 0.2s;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .primary {
    background: var(--color-primary);
    color: white;
  }

  .primary:hover:not(:disabled) {
    background: var(--color-primary-dark);
  }

  .secondary {
    background: var(--color-surface-hover);
    color: var(--color-text);
  }

  .secondary:hover:not(:disabled) {
    background: var(--color-border);
  }

  input {
    padding: 0.5rem;
    border: 1px solid var(--color-border);
    border-radius: 0.5rem;
    font-size: 1rem;
    background: var(--color-background);
  }

  input:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  @media (max-width: 640px) {
    .timer-display .time {
      font-size: 3rem;
    }

    .timer-settings {
      flex-direction: column;
      align-items: center;
    }

    .user-item {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.5rem;
    }
  }
</style>
