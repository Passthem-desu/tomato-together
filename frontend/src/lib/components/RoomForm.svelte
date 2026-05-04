<script lang="ts">
  import { rooms } from '$lib/api/client';
  import { currentRoomStore, userStore, roomTokenStore, roomMembersStore, errorStore } from '$lib/stores';
  import { goto } from '$app/navigation';
  import { connectSSE } from '$lib/sse/client';

  let mode = $state<'join' | 'create'>('join');
  let roomName = $state('');
  let username = $state('');
  let password = $state('');
  let roomPassword = $state('');
  let isReadonly = $state(false);
  let loading = $state(false);
  let error = $state('');

  async function handleSubmit() {
    if (!roomName.trim() || !username.trim()) {
      error = '请填写房间名和用户名';
      return;
    }

    loading = true;
    error = '';

    try {
      if (mode === 'join') {
        const result = await rooms.join(roomName.trim(), username.trim(), password || undefined);
        currentRoomStore.set(result.room);
        roomTokenStore.set(result.token);
        
        connectSSE(result.room.name, result.token);
        
        goto(`/room/${encodeURIComponent(result.room.name)}`);
      } else {
        const room = await rooms.create(roomName.trim(), roomPassword || undefined, isReadonly);
        currentRoomStore.set(room);
        
        const joinResult = await rooms.join(roomName.trim(), username.trim(), roomPassword || undefined);
        currentRoomStore.set(joinResult.room);
        roomTokenStore.set(joinResult.token);
        
        connectSSE(joinResult.room.name, joinResult.token);
        
        goto(`/room/${encodeURIComponent(joinResult.room.name)}`);
      }
    } catch (e: any) {
      error = e.message || '操作失败';
      errorStore.set(error);
    } finally {
      loading = false;
    }
  }

  function toggleMode() {
    mode = mode === 'join' ? 'create' : 'join';
    error = '';
  }
</script>

<div class="room-form">
  <h2 class="title">
    {mode === 'join' ? '加入房间' : '创建房间'}
  </h2>

  {#if error}
    <div class="error">{error}</div>
  {/if}

  <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
    <div class="form-group">
      <label for="username">用户名</label>
      <input 
        type="text" 
        id="username" 
        bind:value={username}
        placeholder="你的名字"
        disabled={loading}
      />
    </div>

    <div class="form-group">
      <label for="roomName">房间名</label>
      <input 
        type="text" 
        id="roomName" 
        bind:value={roomName}
        placeholder="{mode === 'join' ? '输入房间名' : '房间名称'}"
        disabled={loading}
      />
    </div>

    {#if mode === 'join'}
      <div class="form-group">
        <label for="password">房间密码（如有）</label>
        <input 
          type="password" 
          id="password" 
          bind:value={password}
          placeholder="输入房间密码"
          disabled={loading}
        />
      </div>
    {:else}
      <div class="form-group">
        <label for="roomPassword">房间密码（可选）</label>
        <input 
          type="password" 
          id="roomPassword" 
          bind:value={roomPassword}
          placeholder="设置房间密码"
          disabled={loading}
        />
      </div>

      <div class="form-group checkbox">
        <label>
          <input type="checkbox" bind:checked={isReadonly} disabled={loading} />
          只读模式（成员只能观看，不能开始番茄）
        </label>
      </div>
    {/if}

    <button type="submit" class="btn-submit" disabled={loading}>
      {loading ? '处理中...' : (mode === 'join' ? '加入' : '创建房间')}
    </button>
  </form>

  <p class="toggle-mode">
    {mode === 'join' ? '想要创建新房间？' : '想要加入已有房间？'}
    <button onclick={toggleMode}>
      {mode === 'join' ? '创建房间' : '加入房间'}
    </button>
  </p>
</div>

<style>
  .room-form {
    background: var(--color-bg-secondary);
    border-radius: 0.5rem;
    padding: 2rem;
    min-width: 300px;
    max-width: 400px;
  }

  .title {
    text-align: center;
    margin: 0 0 1.5rem 0;
    color: var(--color-text);
  }

  .error {
    background: rgba(244, 67, 54, 0.1);
    border: 1px solid var(--color-danger);
    color: var(--color-danger);
    padding: 0.75rem;
    border-radius: 0.25rem;
    margin-bottom: 1rem;
    font-size: 0.875rem;
  }

  .form-group {
    margin-bottom: 1rem;
  }

  .form-group label {
    display: block;
    margin-bottom: 0.5rem;
    color: var(--color-text);
    font-size: 0.875rem;
  }

  .form-group input[type="text"],
  .form-group input[type="password"] {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid var(--color-border);
    border-radius: 0.25rem;
    background: var(--color-bg);
    color: var(--color-text);
    font-size: 1rem;
  }

  .form-group.checkbox label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    cursor: pointer;
  }

  .form-group input:focus {
    outline: none;
    border-color: var(--color-primary);
  }

  .btn-submit {
    width: 100%;
    padding: 0.75rem;
    background: var(--color-primary);
    color: white;
    border: none;
    border-radius: 0.25rem;
    font-size: 1rem;
    cursor: pointer;
    transition: background 0.2s;
  }

  .btn-submit:hover:not(:disabled) {
    background: var(--color-primary-dark);
  }

  .btn-submit:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .toggle-mode {
    text-align: center;
    margin-top: 1rem;
    color: var(--color-text-secondary);
    font-size: 0.875rem;
  }

  .toggle-mode button {
    background: none;
    border: none;
    color: var(--color-primary);
    cursor: pointer;
    text-decoration: underline;
  }

  :global(:root) {
    --color-primary: #ff6b6b;
    --color-primary-dark: #ee5a5a;
    --color-danger: #f44336;
    --color-bg: #1a1a2e;
    --color-bg-secondary: #16213e;
    --color-text: #eaeaea;
    --color-text-secondary: #a0a0a0;
    --color-border: #333;
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
</style>