<script lang="ts">
  import { page } from '$app/stores';
  import { onMount, onDestroy } from 'svelte';
  import { browser } from '$app/environment';
  
  import PomodoroTimer from '$lib/components/PomodoroTimer.svelte';
  import UserList from '$lib/components/UserList.svelte';
  import TaskPanel from '$lib/components/TaskPanel.svelte';
  import NotificationToast from '$lib/components/NotificationToast.svelte';
  
  import { 
    currentRoomStore, 
    roomMembersStore, 
    userStore, 
    localTimerStore,
    pomodoroStore,
    sseConnectedStore
  } from '$lib/stores';
  import { rooms, pomodoro as pomodoroApi } from '$lib/api/client';
  import { connectSSE, disconnectSSE, subscribeEvent } from '$lib/sse/client';
  import { goto } from '$app/navigation';
  import type { RoomMember } from '$lib/types';

  let roomName = $derived(decodeURIComponent($page.params.name));
  
  let loading = $state(true);
  let isOwner = $state(false);
  let showTaskPanel = $state(false);
  
  // 事件订阅清理函数
  let unsubscribeJoined: (() => void) | null = null;
  let unsubscribeLeft: (() => void) | null = null;
  let unsubscribePomodoroStarted: (() => void) | null = null;
  let unsubscribePomodoroEnded: (() => void) | null = null;
  let unsubscribeTick: (() => void) | null = null;

  onMount(async () => {
    if (!browser) return;

    const currentRoom = localStorage.getItem('current_room');
    if (!currentRoom) {
      goto('/');
      return;
    }

    try {
      // 获取房间信息
      const room = await rooms.get(roomName);
      currentRoomStore.set(room);

      // 获取房间成员
      const users = await rooms.getUsers(roomName);
      roomMembersStore.set(users);

      // 检查是否房主
      const token = localStorage.getItem('tomatogether_token');
      if (token && $userStore) {
        const member = users.find(u => u.username === $userStore?.username);
        isOwner = member?.is_owner || false;
      }

      // 连接 SSE
      const roomToken = localStorage.getItem('tomatogether_room_token');
      connectSSE(roomName, roomToken || undefined);

      // 订阅 SSE 事件
      unsubscribeJoined = subscribeEvent('user_joined', handleUserJoined);
      unsubscribeLeft = subscribeEvent('user_left', handleUserLeft);
      unsubscribePomodoroStarted = subscribeEvent('pomodoro_started', handlePomodoroStarted);
      unsubscribePomodoroEnded = subscribeEvent('pomodoro_ended', handlePomodoroEnded);
      unsubscribeTick = subscribeEvent('tick', handleTick);

    } catch (e) {
      console.error('Failed to load room:', e);
      goto('/');
    } finally {
      loading = false;
    }
  });

  onDestroy(() => {
    if (browser) {
      disconnectSSE();
      unsubscribeJoined?.();
      unsubscribeLeft?.();
      unsubscribePomodoroStarted?.();
      unsubscribePomodoroEnded?.();
      unsubscribeTick?.();
    }
  });

  function handleUserJoined(data: any) {
    rooms.getUsers(roomName).then(users => {
      roomMembersStore.set(users);
    });
  }

  function handleUserLeft(data: any) {
    roomMembersStore.update(members => 
      members.filter(m => m.id !== data.user_id)
    );
  }

  function handlePomodoroStarted(data: any) {
    // 如果是自己开始的，直接开始计时
    if (data.user_id === $userStore?.id) {
      // 已通过 API 开始
    }
  }

  function handlePomodoroEnded(data: any) {
    if (data.user_id === $userStore?.id) {
      // 已通过 API 结束
    }
  }

  function handleTick(data: any) {
    // 更新在线状态
    if (data.users) {
      roomMembersStore.update(members => 
        members.map(m => {
          const userData = data.users.find((u: any) => u.id === m.id);
          return {
            ...m,
            is_online: userData ? true : m.is_online,
            pomodoro: userData?.status ? {
              ...m.pomodoro,
              remaining_seconds: userData.remaining_seconds,
              status: userData.status
            } : m.pomodoro
          };
        })
      );
    }
  }

  async function handleStartPomodoro() {
    try {
      const session = await pomodoroApi.start(roomName);
      pomodoroStore.set(session);
      localTimerStore.start(session.planned_duration, 'focusing');
    } catch (e) {
      console.error('Failed to start pomodoro:', e);
    }
  }

  async function handleEndPomodoro() {
    try {
      await pomodoroApi.end(roomName);
      pomodoroStore.set(null);
      localTimerStore.stop();
    } catch (e) {
      console.error('Failed to end pomodoro:', e);
    }
  }

  async function handleFollow(userId: string) {
    try {
      const session = await pomodoroApi.follow(roomName, userId);
      pomodoroStore.set(session);
      
      // 从房间成员获取开始时间
      const member = $roomMembersStore.find(m => m.id === userId);
      if (member?.pomodoro?.started_at) {
        const remaining = member.pomodoro.remaining_seconds || session.planned_duration;
        localTimerStore.start(remaining, 'following');
      }
    } catch (e) {
      console.error('Failed to follow:', e);
    }
  }

  async function handleLeave() {
    try {
      await rooms.leave(roomName);
      currentRoomStore.set(null);
      goto('/');
    } catch (e) {
      console.error('Failed to leave room:', e);
    }
  }

  function toggleTaskPanel() {
    showTaskPanel = !showTaskPanel;
  }
</script>

{#if loading}
  <div class="loading">加载中...</div>
{:else if $currentRoomStore}
  <div class="room-page">
    <header class="room-header">
      <div class="room-info">
        <h1 class="room-name">🍅 {$currentRoomStore.name}</h1>
        {#if isOwner}
          <span class="owner-badge">👑 房主</span>
        {/if}
      </div>
      <div class="room-actions">
        <button class="btn-toggle-tasks" onclick={toggleTaskPanel}>
          {showTaskPanel ? '隐藏任务' : '显示任务'}
        </button>
        <button class="btn-leave" onclick={handleLeave}>离开房间</button>
      </div>
    </header>

    <div class="room-content">
      <div class="main-area">
        <PomodoroTimer 
          {roomName}
          {isOwner}
          onStart={handleStartPomodoro}
          onEnd={handleEndPomodoro}
        />
      </div>

      <aside class="sidebar" class:collapsed={!showTaskPanel}>
        {#if showTaskPanel}
          <TaskPanel />
        {/if}
        <UserList onFollow={handleFollow} />
      </aside>
    </div>

    <div class="connection-status" class:connected={$sseConnectedStore}>
      {$sseConnectedStore ? '🟢 已连接' : '🔴 未连接'}
    </div>
  </div>
{:else}
  <div class="error">房间不存在或已离开</div>
  <button onclick={() => goto('/')}>返回首页</button>
{/if}

<NotificationToast />

<style>
  .room-page {
    display: flex;
    flex-direction: column;
    height: 100vh;
    padding: 1rem;
  }

  .loading, .error {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100vh;
    color: var(--color-text-secondary);
  }

  .error button {
    margin-top: 1rem;
    padding: 0.5rem 1rem;
    background: var(--color-primary);
    color: white;
    border: none;
    border-radius: 0.25rem;
    cursor: pointer;
  }

  .room-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem;
    background: var(--color-bg-secondary);
    border-radius: 0.5rem;
    margin-bottom: 1rem;
  }

  .room-info {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .room-name {
    margin: 0;
    font-size: 1.5rem;
    color: var(--color-text);
  }

  .owner-badge {
    font-size: 0.875rem;
    background: var(--color-primary);
    color: white;
    padding: 0.25rem 0.5rem;
    border-radius: 1rem;
  }

  .room-actions {
    display: flex;
    gap: 0.5rem;
  }

  .btn-toggle-tasks, .btn-leave {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 0.25rem;
    cursor: pointer;
    font-size: 0.875rem;
  }

  .btn-toggle-tasks {
    background: var(--color-bg);
    color: var(--color-text);
  }

  .btn-leave {
    background: var(--color-danger);
    color: white;
  }

  .room-content {
    flex: 1;
    display: flex;
    gap: 1rem;
    overflow: hidden;
  }

  .main-area {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
  }

  .sidebar {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    width: 300px;
    overflow-y: auto;
  }

  .sidebar.collapsed {
    width: auto;
  }

  .connection-status {
    position: fixed;
    bottom: 1rem;
    right: 1rem;
    padding: 0.5rem 1rem;
    background: var(--color-bg-secondary);
    border-radius: 0.25rem;
    font-size: 0.75rem;
    color: var(--color-text-secondary);
  }

  .connection-status.connected {
    color: var(--color-success);
  }

  :global(:root) {
    --color-primary: #ff6b6b;
    --color-bg: #1a1a2e;
    --color-bg-secondary: #16213e;
    --color-text: #eaeaea;
    --color-text-secondary: #a0a0a0;
    --color-danger: #f44336;
    --color-success: #4caf50;
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