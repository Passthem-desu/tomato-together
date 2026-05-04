<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { goto } from '$app/navigation';
  import {
    currentMember, currentRoom, roomUsers, pomodoroStatus, isLoading, error,
    startPomodoro, endPomodoro, unfollowPomodoro,
    refreshRoomUsers, logout, connectSSE, sseConnected,
  } from '$lib/store';
  import { api } from '$lib/api';
  import { locale, t } from '$lib/i18n';
  import { PomodoroCountdown } from '$lib/countdown';
  import { SoundManager, type SoundEvent } from '$lib/sounds';

  const sound = new SoundManager();
  const countdown = new PomodoroCountdown();

  const allSoundEvents: SoundEvent[] = ['focus_start', 'focus_end', 'focus_pause', 'focus_resume', 'rest_end', 'all_done'];
  const soundEventLabel = (ev: SoundEvent) => {
    const map: Record<SoundEvent, string> = {
      focus_start: 'focus_start', focus_end: 'focus_end',
      focus_pause: 'focus_pause', focus_resume: 'focus_resume',
      rest_end: 'rest_end', all_done: 'all_done',
    };
    return t(map[ev], $locale);
  };
  let displayTime = $state(25 * 60);
  let plannedMinutes = $state(25);
  let restMinutes = $state(5);
  let longBreakMinutes = $state(15);
  let totalSessions = $state(4);
  let sessionsBeforeLong = $state(4);
  let sessionIndex = $state(0);
  let showSettings = $state(false);
  let newSoundLabel = $state('');
  let newSoundUrl = $state('');
  let fileInput = $state<HTMLInputElement | null>(null);
  let soundVersion = $state(0);
  let showSoundDialog = $state(false);
  let allSounds = $derived((soundVersion, sound.allSounds()));

  let estimatedFinish = $derived(computeEstimate());

  function computeEstimate(): string {
    if (totalSessions <= 0 || plannedMinutes <= 0) return '';
    const f = totalSessions * plannedMinutes * 60;
    const longs = Math.floor((totalSessions - 1) / sessionsBeforeLong);
    const shorts = (totalSessions - 1) - longs;
    const totalSec = f + longs * longBreakMinutes * 60 + shorts * restMinutes * 60;
    const finish = new Date(Date.now() + totalSec * 1000);
    return finish.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  // ── Wire countdown events ──
  const unsubs: (() => void)[] = [];

  unsubs.push(countdown.on('tick', (remaining: number) => {
    displayTime = remaining;
  }));

  // Focus complete → enter rest
  unsubs.push(countdown.on('complete', async () => {
    const phase = $pomodoroStatus.phase;
    if (phase === 'focusing' || phase === 'following') {
      // Try to end on server; if offline, proceed locally
      let restSec = restMinutes * 60;
      try {
        await endPomodoro(false);
        // Use server-authoritative rest duration
        restSec = $pomodoroStatus.remaining_seconds || $pomodoroStatus.rest_duration || restSec;
      } catch { /* offline — use local */ }
      sound.play('focus_end');
      notifyPomodoroEnd();
      countdown.start(restSec);
    } else if (phase === 'rest') {
      sound.play('rest_end');
      notifyRestEnd();
      if (sessionIndex < totalSessions) {
        sessionIndex++;
        try { await startNextFocus(); } catch { countdown.start(plannedMinutes * 60); }
      } else {
        sessionIndex = 0;
        pomodoroStatus.set({ phase: 'idle' });
        displayTime = plannedMinutes * 60;
        sound.play('all_done');
        notifyAllDone();
      }
    }
  }));

  async function startNextFocus() {
    await startPomodoro({
      planned_duration: plannedMinutes * 60,
      rest_duration: restMinutes * 60,
      long_break_duration: longBreakMinutes * 60,
      sessions_before_long_break: sessionsBeforeLong,
    });
    countdown.start(plannedMinutes * 60);
  }

  // ── Sync displayTime when idle ──
  $effect(() => {
    const _ = plannedMinutes;
    if ($pomodoroStatus.phase === 'idle' && countdown.getState() === 'idle') {
      displayTime = plannedMinutes * 60;
    }
  });

  onMount(() => {
    if (!$currentMember) { goto('/'); return; }
    refreshRoomUsers();
    connectSSE();
    restorePomodoroState();

    const handleVisibility = () => {
      if (document.visibilityState === 'visible' && $pomodoroStatus.phase !== 'idle')
        restorePomodoroState();
    };
    document.addEventListener('visibilitychange', handleVisibility);
    unsubs.push(() => document.removeEventListener('visibilitychange', handleVisibility));
  });

  onDestroy(() => {
    countdown.destroy();
    unsubs.forEach(fn => fn());
  });

  // ── Actions ──
  async function handleStart() {
    requestNotificationPermission();
    sound.play('focus_start');
    sessionIndex = 1;
    await startNextFocus();
  }

  async function handlePause() {
    sound.play('focus_pause');
    const resp = await api.pausePomodoro();
    if (resp.data) { pomodoroStatus.set(resp.data); countdown.pause(); countdown.setRemaining(resp.data.remaining_seconds ?? displayTime); }
  }

  async function handleResume() {
    sound.play('focus_resume');
    const resp = await api.resumePomodoro();
    if (resp.data) { pomodoroStatus.set(resp.data); countdown.setRemaining(resp.data.remaining_seconds ?? displayTime); countdown.resume(); }
  }

  async function handleSkip() {
    sound.play('rest_end');
    await api.skipRest();
    if (sessionIndex < totalSessions) {
      sessionIndex++;
      startNextFocus();
    } else {
      sessionIndex = 0;
      pomodoroStatus.set({ phase: 'idle' });
      displayTime = plannedMinutes * 60;
    }
  }

  async function handleStop() {
    sound.play('focus_end');
    await endPomodoro(true);
    countdown.stop();
    sessionIndex = 0;
    displayTime = plannedMinutes * 60;
  }

  async function handleEnd() {
    sound.play('focus_end');
    await endPomodoro(false);
    const restSec = $pomodoroStatus.remaining_seconds || $pomodoroStatus.rest_duration || restMinutes * 60;
    countdown.stop();
    countdown.start(restSec);
    displayTime = restSec;
    notifyPomodoroEnd();
  }

  // ── Drift correction ──

  function addCustomSound() {
    const label = newSoundLabel.trim();
    const url = newSoundUrl.trim();
    if (!label) return;
    if (url) {
      sound.addUrlSound(label, url);
    }
    newSoundLabel = '';
    newSoundUrl = '';
    showSoundDialog = false;
    soundVersion++;
  }

  function handleFileUpload(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      sound.addDataSound(file.name.replace(/\.[^.]+$/, ''), reader.result as string);
      soundVersion++;
    };
    reader.readAsDataURL(file);
  }
  $effect(() => {
    const sec = $pomodoroStatus.remaining_seconds;
    const phase = $pomodoroStatus.phase;
    // Only correct during focusing/following — rest is client-driven
    if (sec !== undefined && countdown.getState() === 'running' && phase !== 'rest') {
      if (Math.abs(countdown.getRemaining() - sec) > 3) countdown.setRemaining(sec);
    }
  });

  // ── Leave ──
  async function handleLogout() {
    if (confirm(t('confirm_logout', $locale))) { logout(); goto('/'); }
  }

  // ── Helpers ──
  function formatTime(s: number) {
    return `${Math.floor(s / 60).toString().padStart(2, '0')}:${(s % 60).toString().padStart(2, '0')}`;
  }
  function phaseLabel(): string {
    const p = $pomodoroStatus.phase;
    if (p === 'focusing') return t('focusing', $locale);
    if (p === 'paused') return t('paused', $locale);
    if (p === 'following') return `${t('following', $locale)} ${$pomodoroStatus.leader_username}`;
    if (p === 'rest') return $pomodoroStatus.is_long_break ? t('rest_long', $locale) : t('rest_short', $locale);
    return t('idle', $locale);
  }
  function onlineCount() { return $roomUsers.filter(u => u.is_online).length; }

  async function restorePomodoroState() {
    try {
      const resp = await api.getPomodoroStatus();
      if (resp.data) {
        pomodoroStatus.set(resp.data);
        if (resp.data.remaining_seconds !== undefined) {
          if (resp.data.phase === 'paused') {
            countdown.setRemaining(resp.data.remaining_seconds);
          } else if (resp.data.phase !== 'idle') {
            countdown.start(resp.data.remaining_seconds);
          }
        }
      }
    } catch { /* ignore */ }
  }

  // ── Notifications ──
  function requestNotificationPermission() {
    if ('Notification' in window && Notification.permission === 'default')
      Notification.requestPermission();
  }
  function notify(title: string, body: string) {
    if ('Notification' in window && Notification.permission === 'granted')
      new Notification(title, { body, icon: '/favicon.png' });
  }
  function notifyPomodoroEnd() {
    notify(t('notify_focus_end_title', $locale), t('notify_focus_end_body', $locale));
  }
  function notifyRestEnd() {
    notify(t('notify_rest_end_title', $locale), t('notify_rest_end_body', $locale));
  }
  function notifyAllDone() {
    notify(t('notify_all_done_title', $locale), t('notify_all_done_title', $locale));
  }
</script>

<svelte:head>
  <title>{t('room_title', $locale)} - TomatoTogether</title>
</svelte:head>

<main class="room">
  <header class="room-header">
    <div class="header-left">
      <div class="room-avatar"></div>
      <div class="room-info">
        <h1>{$currentRoom?.name || t('room_title', $locale)}</h1>
        <div class="room-meta">
          <span class="status-dot" class:online={onlineCount() > 0} class:disconnected={!$sseConnected}></span>
          <span class="member-count">{onlineCount()} {t('online', $locale)}</span>
        </div>
      </div>
    </div>
    <div class="header-right">
      <div class="user-badge">
        <span class="user-name">{$currentMember?.username}</span>
        {#if $currentMember?.is_owner}<span class="badge">{t('owner', $locale)}</span>{/if}
      </div>
      <button class="btn-ghost btn-sm" onclick={handleLogout}>{t('logout', $locale)}</button>
    </div>
  </header>

  <div class="room-content">
    <section class="main-panel">
      {#if showSettings}
        <!-- ═══ Settings Panel ═══ -->
        <div class="card settings-card">
          <div class="settings-header">
            <h2 class="settings-title">{t('settings', $locale)}</h2>
            <button class="btn-ghost btn-sm" onclick={() => showSettings = false}>{t('close', $locale)}</button>
          </div>
          <div class="settings-grid">
            <div class="setting">
              <label>{t('focus_time', $locale)}</label>
              <input type="number" bind:value={plannedMinutes} min="1" max="60" />
            </div>
            <div class="setting">
              <label>{t('short_break', $locale)}</label>
              <input type="number" bind:value={restMinutes} min="1" max="30" />
            </div>
            <div class="setting">
              <label>{t('long_break', $locale)}</label>
              <input type="number" bind:value={longBreakMinutes} min="1" max="60" />
            </div>
            <div class="setting">
              <label>{t('total_sessions', $locale)}</label>
              <input type="number" bind:value={totalSessions} min="1" max="20" />
            </div>
            <div class="setting">
              <label>{t('long_break_after', $locale)}</label>
              <input type="number" bind:value={sessionsBeforeLong} min="1" max="10" />
            </div>
          </div>
          {#if estimatedFinish}
            <p class="estimate">⏱ {t('estimated_finish', $locale)} {estimatedFinish}</p>
          {/if}

          <h3 class="settings-subtitle">{t('sounds', $locale)}</h3>
          <div class="sounds-grid">
          {#each allSoundEvents as ev}
            <div class="sound-row">
              <label>{soundEventLabel(ev)}</label>
              <div class="sound-row-right">
                <button class="btn-text btn-xs" onclick={() => sound.preview(sound.getSound(ev))} title={t('preview', $locale)}>▶</button>
                <select value={sound.getSound(ev)} onchange={(e) => { const v = (e.target as HTMLSelectElement).value; sound.setSound(ev, v); sound.preview(v); }}>
                {#each allSounds as s (s.key)}
                  <option value={s.key} selected={sound.getSound(ev) === s.key}>{s.label}</option>
                {/each}
              </select>
              </div>
            </div>
          {/each}
          </div>

          <h3 class="settings-subtitle">{t('custom_sounds', $locale)}</h3>
          <p class="hint-text">{t('custom_sounds_hint', $locale)}</p>
          <div class="custom-actions">
            <button class="btn-secondary btn-sm" onclick={() => showSoundDialog = true}>{t('add_network_sound', $locale)}</button>
            <input type="file" accept="audio/*" onchange={handleFileUpload} style="display:none" bind:this={fileInput} />
            <button class="btn-secondary btn-sm" onclick={() => fileInput?.click()}>{t('upload_sound', $locale)}</button>
          </div>
          {#if allSounds.filter(s => s.source !== 'builtin').length > 0}
            <div class="custom-list-box">
              <ul class="custom-list">
                {#each allSounds.filter(s => s.source !== 'builtin') as s (s.key)}
                  <li>{s.label} <span><button class="btn-text btn-xs" onclick={() => sound.preview(s.key)}>▶</button> <button class="btn-text btn-xs" onclick={() => { sound.removeSound(s.key); soundVersion++; }}>×</button></span></li>
                {/each}
              </ul>
            </div>
          {/if}
        </div>

        {#if showSoundDialog}
          <div class="dialog-overlay" onclick={() => showSoundDialog = false}>
            <div class="dialog" onclick={(e) => e.stopPropagation()}>
              <h3>{t('add_network_sound', $locale)}</h3>
              <div class="form-group">
                <label>{t('sound_label_placeholder', $locale)}</label>
                <input type="text" bind:value={newSoundLabel} />
              </div>
              <div class="form-group">
                <label>{t('sound_url_placeholder', $locale)}</label>
                <input type="text" bind:value={newSoundUrl} />
              </div>
              <div class="dialog-actions">
                <button class="btn-text" onclick={() => sound.preview(newSoundUrl)} disabled={!newSoundUrl}>{t('preview', $locale)}</button>
                <button class="btn-secondary" onclick={addCustomSound}>{t('add', $locale)}</button>
                <button class="btn-ghost" onclick={() => showSoundDialog = false}>{t('close', $locale)}</button>
              </div>
            </div>
          </div>
        {/if}
      {:else}
        <!-- ═══ Timer Card ═══ -->
        <div class="card timer-card">
          {#if sessionIndex > 0}
            <p class="session-progress">🍅 {sessionIndex} / {totalSessions}</p>
          {/if}

          <div class="timer-display">
            <span class="time">{formatTime(displayTime)}</span>
            <span class="status-label" class:focusing={$pomodoroStatus.phase === 'focusing' || $pomodoroStatus.phase === 'following'}
                  class:resting={$pomodoroStatus.phase === 'rest'}
                  class:paused={$pomodoroStatus.phase === 'paused'}>
              {phaseLabel()}
            </span>
          </div>

          {#if $pomodoroStatus.phase === 'idle'}
            <button class="btn-primary btn-lg btn-full" onclick={handleStart} disabled={$isLoading}>
              {t('start_pomodoro', $locale)}
            </button>
            <button class="btn-secondary btn-lg btn-full" onclick={() => showSettings = true} style="margin-top:0.5rem">{t('settings', $locale)}</button>
          {:else if $pomodoroStatus.phase === 'paused'}
            <div class="timer-actions">
              <button class="btn-primary" onclick={handleResume}>▶ {t('continue', $locale)}</button>
              <button class="btn-ghost" onclick={handleStop}>⏹ {t('stop', $locale)}</button>
            </div>
          {:else if $pomodoroStatus.phase === 'rest'}
            <div class="timer-actions">
              <button class="btn-primary" onclick={handleSkip}>⏭ {t('skip', $locale)}</button>
            </div>
          {:else}
            <div class="timer-actions">
              <button class="btn-secondary" onclick={handlePause}>⏸ {t('pause', $locale)}</button>
              <button class="btn-secondary" onclick={handleEnd}>{t('end_pomodoro', $locale)}</button>
              <button class="btn-ghost" onclick={handleStop}>⏹ {t('stop', $locale)}</button>
              {#if $pomodoroStatus.phase === 'following'}
                <button class="btn-ghost" onclick={unfollowPomodoro}>{t('unfollow', $locale)}</button>
              {/if}
            </div>
          {/if}

          {#if $error}<p class="form-error">{$error}</p>{/if}
        </div>
      {/if}
    </section>

    <section class="users-panel">
      <div class="card users-card">
        <h2>{t('online_users', $locale)}</h2>
        {#if $roomUsers.filter(u => u.is_online).length === 0}
          <p class="empty-state">{t('no_online_users', $locale)}</p>
        {:else}
          <div class="users-list">
            {#each $roomUsers.filter(u => u.is_online) as user}
              <div class="user-item">
                <div class="user-info">
                <div class="user-details">
                  <span class="user-name">{user.username}
                    {#if user.id === $currentMember?.id}<span class="you-badge">{t('me', $locale)}</span>{/if}
                    {#if user.is_owner}<span class="owner-badge">{t('owner', $locale)}</span>{/if}
                  </span>
                    {#if user.status}
                      <span class="user-status">{user.status.emoji} {user.status.message}</span>
                    {/if}
                  </div>
                </div>
                <div class="user-pomodoro">
                  {#if user.pomodoro?.phase && user.pomodoro.phase !== 'idle'}
                    {#if user.pomodoro.is_following}
                      <span class="following">🔄 {t('following', $locale)} {user.pomodoro.leader_username}</span>
                    {:else if user.pomodoro.phase === 'paused'}
                      <span class="paused-status">⏸ {t('paused', $locale)}</span>
                    {:else if user.pomodoro.phase === 'rest'}
                      <span class="rest-status">{t('resting', $locale)} {user.pomodoro.remaining_seconds ? formatTime(user.pomodoro.remaining_seconds) : ''}</span>
                    {:else}
                      <span class="active-pomodoro">{user.pomodoro.remaining_seconds ? formatTime(user.pomodoro.remaining_seconds) : ''}</span>
                    {/if}
                  {:else}
                    <span class="idle-status">{t('idle', $locale)}</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </section>
  </div>
</main>

<style>
  .room { min-height: 100dvh; padding: 1.5rem; background: var(--color-bg-0); }
  .room-header { display: flex; justify-content: space-between; align-items: center; padding: 1rem 0; margin-bottom: 2rem; border-bottom: 1px solid var(--color-border); }
  .header-left { display: flex; align-items: center; gap: 0.875rem; }
  .room-avatar { font-size: 2.5rem; }
  .room-info h1 { font-size: var(--text-xl); font-weight: 600; margin-bottom: 0.25rem; }
  .room-meta { display: flex; align-items: center; gap: 0.5rem; color: var(--color-fg-muted); font-size: var(--text-sm); }
  .status-dot { width: 8px; height: 8px; border-radius: var(--radius-full); background: var(--color-fg-muted); }
  .status-dot.online { background: var(--color-success); }
  .status-dot.disconnected { background: var(--color-warning, #f59e0b); }
  .header-right { display: flex; align-items: center; gap: 1rem; }
  .user-badge { display: flex; align-items: center; gap: 0.5rem; }
  .user-badge .user-name { font-weight: 500; }
  .badge { display: inline-flex; padding: 0.25rem 0.5rem; font-size: var(--text-xs); font-weight: 500; border-radius: var(--radius-full); background: var(--color-brand-subtle); color: var(--color-brand); }
  .room-content { display: flex; gap: 1.5rem; align-items: flex-start; }
  .main-panel { flex: 1; min-width: 0; }
  .users-panel { width: 320px; flex-shrink: 0; }
  .timer-card { text-align: center; padding: 2rem; }
  .settings-card { padding: 1.5rem; }
  .settings-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.25rem; }
  .settings-title { font-size: var(--text-lg); font-weight: 600; }
  .settings-grid { display: flex; gap: 1rem; flex-wrap: wrap; }
  .settings-grid .setting { flex: 0 0 auto; }
  .session-progress { font-size: var(--text-sm); color: var(--color-fg-muted); margin-bottom: 0.5rem; }
  .timer-display { margin-bottom: 1.5rem; }
  .time { font-size: 4.5rem; font-weight: 700; font-variant-numeric: tabular-nums; letter-spacing: -0.02em; display: block; margin-bottom: 0.5rem; background: linear-gradient(135deg, var(--color-fg-0), var(--color-brand)); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }
  .status-label { color: var(--color-fg-muted); font-size: var(--text-lg); }
  .status-label.focusing { color: var(--color-brand); }
  .status-label.resting { color: var(--color-success); }
  .status-label.paused { color: var(--color-warning, #f59e0b); }
  .estimate { font-size: var(--text-sm); color: var(--color-fg-muted); margin-bottom: 1rem; margin-top: 0.5rem; }
  .settings-subtitle { font-size: var(--text-base); font-weight: 600; margin: 1.5rem 0 0.5rem; }
  .hint-text { font-size: var(--text-xs); color: var(--color-fg-muted); margin-bottom: 0.5rem; }
  .custom-actions { display: flex; gap: 0.5rem; margin-bottom: 0.5rem; }
  .custom-list-box { border: 1px solid var(--color-border); border-radius: var(--radius-md); padding: 0.5rem; margin-top: 0.5rem; }
  .custom-list { list-style: none; padding: 0; margin: 0; }
  .custom-list li { display: flex; justify-content: space-between; align-items: center; padding: 0.25rem 0; font-size: var(--text-sm); }
  .dialog-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; z-index: 100; }
  .dialog { background: var(--color-bg-1); border-radius: var(--radius-lg); padding: 1.5rem; width: 90%; max-width: 400px; box-shadow: var(--shadow-lg); }
  .dialog h3 { margin-bottom: 1rem; }
  .dialog .form-group { margin-bottom: 0.75rem; }
  .dialog .form-group label { display: block; font-size: var(--text-sm); margin-bottom: 0.25rem; }
  .dialog .form-group input { width: 100%; padding: 0.5rem; }
  .dialog-actions { display: flex; gap: 0.5rem; justify-content: flex-end; margin-top: 1rem; }
  .sound-row { display: flex; align-items: baseline; padding: 0.375rem 0; gap: 0.5rem; }
  .sound-row-right { display: flex; align-items: center; gap: 0.25rem; }
  .sound-row label { font-size: var(--text-sm); white-space: nowrap; }
  .sound-row select { font-size: var(--text-sm); padding: 0.25rem 0.5rem; min-width: 110px; }
  .sounds-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0 2rem; }
  .timer-settings { display: flex; gap: 1rem; justify-content: center; margin-bottom: 1rem; flex-wrap: wrap; }
  .setting { display: flex; flex-direction: column; gap: 0.25rem; }
  .setting label { font-size: var(--text-xs); color: var(--color-fg-muted); font-weight: 500; }
  .setting input { width: 70px; text-align: center; padding: 0.5rem; }
  .timer-actions { display: flex; gap: 0.75rem; justify-content: center; flex-wrap: wrap; }
  .users-card h2 { font-size: var(--text-base); font-weight: 600; margin-bottom: 1rem; color: var(--color-fg-1); }
  .empty-state { text-align: center; color: var(--color-fg-muted); padding: 1.5rem; }
  .users-list { display: flex; flex-direction: column; gap: 0.5rem; }
  .user-item { display: flex; justify-content: space-between; align-items: center; padding: 0.75rem; background: var(--color-bg-0); border-radius: var(--radius-md); }
  .user-item:hover { background: var(--color-bg-2); }
  .user-info { display: flex; align-items: center; gap: 0.75rem; }
  .user-avatar { font-size: 1.25rem; width: 2rem; text-align: center; }
  .user-details { display: flex; flex-direction: column; gap: 0.125rem; }
  .user-details .user-name { font-weight: 500; display: flex; align-items: center; gap: 0.375rem; }
  .you-badge { font-size: var(--text-xs); padding: 0.125rem 0.375rem; background: var(--color-bg-2); border-radius: var(--radius-sm); color: var(--color-fg-muted); }
  .owner-badge { font-size: var(--text-xs); padding: 0.125rem 0.375rem; background: var(--color-brand-subtle); border-radius: var(--radius-sm); color: var(--color-brand); }
  .user-status { font-size: var(--text-xs); color: var(--color-fg-muted); }
  .user-pomodoro .active-pomodoro { font-weight: 600; color: var(--color-brand); }
  .user-pomodoro .following { font-size: var(--text-sm); color: var(--color-fg-muted); }
  .user-pomodoro .paused-status { font-size: var(--text-sm); color: var(--color-warning, #f59e0b); }
  .user-pomodoro .rest-status { font-size: var(--text-sm); color: var(--color-success); }
  .user-pomodoro .idle-status { font-size: var(--text-sm); color: var(--color-fg-muted); }
  .actions-section { text-align: center; padding: 1rem 0; }
  .form-error { margin-top: 1rem; padding: 0.75rem; background: var(--color-error-subtle); color: var(--color-error); border-radius: var(--radius-md); font-size: var(--text-sm); }
  @media (max-width: 640px) {
    .room { padding: 1rem; } .room-header { flex-direction: column; gap: 1rem; text-align: center; }
    .header-left { flex-direction: column; } .time { font-size: 3.5rem; }
    .timer-actions { flex-direction: column; } .user-item { flex-direction: column; align-items: flex-start; gap: 0.5rem; }
    .sounds-grid { grid-template-columns: 1fr; }
  }
  @media (max-width: 900px) {
    .room-content { flex-direction: column; }
    .main-panel { width: 100%; }
    .users-panel { width: 100%; }
  }
</style>