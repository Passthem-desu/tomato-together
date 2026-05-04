<script lang="ts">
  import { auth } from '$lib/api/client';
  import { userStore, errorStore } from '$lib/stores';

  let mode = $state<'login' | 'register'>('login');
  let username = $state('');
  let password = $state('');
  let loading = $state(false);
  let error = $state('');

  async function handleSubmit() {
    if (!username.trim() || !password.trim()) {
      error = '请填写用户名和密码';
      return;
    }

    loading = true;
    error = '';

    try {
      if (mode === 'register') {
        await auth.register(username, password);
      } else {
        await auth.login(username, password);
      }
      
      const user = await auth.me();
      userStore.set(user);
    } catch (e: any) {
      error = e.message || '操作失败';
      errorStore.set(error);
    } finally {
      loading = false;
    }
  }

  function toggleMode() {
    mode = mode === 'login' ? 'register' : 'login';
    error = '';
  }
</script>

<div class="auth-form">
  <h2 class="title">
    {mode === 'login' ? '登录' : '注册'}
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
        placeholder="输入用户名"
        disabled={loading}
      />
    </div>

    <div class="form-group">
      <label for="password">密码</label>
      <input 
        type="password" 
        id="password" 
        bind:value={password}
        placeholder="输入密码"
        disabled={loading}
      />
    </div>

    <button type="submit" class="btn-submit" disabled={loading}>
      {loading ? '处理中...' : (mode === 'login' ? '登录' : '注册')}
    </button>
  </form>

  <p class="toggle-mode">
    {mode === 'login' ? '还没有账号？' : '已有账号？'}
    <button onclick={toggleMode}>
      {mode === 'login' ? '立即注册' : '立即登录'}
    </button>
  </p>
</div>

<style>
  .auth-form {
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

  .form-group input {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid var(--color-border);
    border-radius: 0.25rem;
    background: var(--color-bg);
    color: var(--color-text);
    font-size: 1rem;
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