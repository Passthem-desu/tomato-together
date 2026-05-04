<script lang="ts">
  import { localTimerStore, pomodoroStore } from '$lib/stores';
  import { pomodoro as pomodoroApi } from '$lib/api/client';

  interface Props {
    roomName: string;
    isOwner?: boolean;
    onStart?: () => void;
    onEnd?: () => void;
  }

  let { roomName, isOwner = false, onStart = () => {}, onEnd = () => {} }: Props = $props();

  let isRunning = $state(false);

  let minutes = $derived(Math.floor($localTimerStore.remaining / 60));
  let seconds = $derived($localTimerStore.remaining % 60);
  let timeDisplay = $derived(`${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`);

  let progress = $derived($localTimerStore.plannedDuration > 0 
    ? (($localTimerStore.plannedDuration - $localTimerStore.remaining) / $localTimerStore.plannedDuration) * 100 
    : 0);

  function handleStart() {
    localTimerStore.start(1500, 'focusing');
    isRunning = true;
    onStart();
  }

  function handleEnd() {
    localTimerStore.stop();
    isRunning = false;
    onEnd();
  }

  function formatTime(seconds: number): string {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  }
</script>

<div class="pomodoro-timer">
  <div class="timer-ring" style="--progress: {progress}%">
    <div class="timer-content">
      <div class="time-display" class:rest={$localTimerStore.status === 'rest'}>
        {timeDisplay}
      </div>
      <div class="status-text">
        {#if $localTimerStore.status === 'idle'}
          准备开始
        {:else if $localTimerStore.status === 'focusing'}
          专注中
        {:else if $localTimerStore.status === 'following'}
          跟随中
        {:else if $localTimerStore.status === 'rest'}
          休息中
        {/if}
      </div>
    </div>
  </div>

  <div class="timer-controls">
    {#if !isRunning}
      <button class="btn-start" onclick={handleStart} disabled={isOwner}>
        开始专注
      </button>
    {:else}
      <button class="btn-end" onclick={handleEnd}>
        结束
      </button>
    {/if}
  </div>

  <div class="timer-info">
    <span class="sessions">今日完成: {$pomodoroStore?.sessions_today || 0} 个番茄</span>
  </div>
</div>

<style>
  .pomodoro-timer {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 2rem;
  }

  .timer-ring {
    position: relative;
    width: 280px;
    height: 280px;
    border-radius: 50%;
    background: conic-gradient(
      var(--color-primary) calc(var(--progress) * 1%),
      var(--color-bg-secondary) calc(var(--progress) * 1%)
    );
    display: flex;
    align-items: center;
    justify-content: center;
    transition: --progress 1s linear;
  }

  .timer-ring::before {
    content: '';
    position: absolute;
    width: 240px;
    height: 240px;
    border-radius: 50%;
    background: var(--color-bg);
  }

  .timer-content {
    position: relative;
    z-index: 1;
    text-align: center;
  }

  .time-display {
    font-size: 4rem;
    font-weight: bold;
    color: var(--color-text);
    font-variant-numeric: tabular-nums;
  }

  .time-display.rest {
    color: var(--color-success);
  }

  .status-text {
    font-size: 1rem;
    color: var(--color-text-secondary);
    margin-top: 0.5rem;
  }

  .timer-controls {
    margin-top: 2rem;
  }

  .btn-start, .btn-end {
    padding: 0.75rem 2rem;
    font-size: 1rem;
    border: none;
    border-radius: 2rem;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-start {
    background: var(--color-primary);
    color: white;
  }

  .btn-start:hover:not(:disabled) {
    background: var(--color-primary-dark);
  }

  .btn-start:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-end {
    background: var(--color-danger);
    color: white;
  }

  .btn-end:hover {
    background: #d32f2f;
  }

  .timer-info {
    margin-top: 1.5rem;
    color: var(--color-text-secondary);
    font-size: 0.875rem;
  }

  :global(:root) {
    --color-primary: #ff6b6b;
    --color-primary-dark: #ee5a5a;
    --color-success: #4caf50;
    --color-danger: #f44336;
    --color-bg: #1a1a2e;
    --color-bg-secondary: #16213e;
    --color-text: #eaeaea;
    --color-text-secondary: #a0a0a0;
  }

  @media (prefers-color-scheme: light) {
    :global(:root) {
      --color-bg: #f5f5f5;
      --color-bg-secondary: #e0e0e0;
      --color-text: #333;
      --color-text-secondary: #666;
    }
  }
</style>