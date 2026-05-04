<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { currentMember, joinRoom, isLoading, error } from '$lib/store';
  import { getMember, isAuthenticated, api } from '$lib/api';
  import { locale, t, getErrorMessage } from '$lib/i18n';

  // Steps: room -> room_password -> username -> (optional user_password)
  let step = $state('room');
  let roomName = $state('');
  let roomPassword = $state('');
  let username = $state('');
  let userPassword = $state('');
  let hasRoomPassword = $state(false);
  let userExists = $state(false);
  let userIsPersistent = $state(false);
  let checkingUser = $state(false);

  // Focus input when step changes
  $effect(() => {
    const _step = step;
    setTimeout(() => {
      const input = document.querySelector('input:not([type="hidden"]):not([disabled])') as HTMLInputElement;
      input?.focus();
    }, 50);
  });

  onMount(() => {
    // Clear any previous error
    error.set(null);

    // Check URL params for pre-filled room name
    const urlRoom = $page.url.searchParams.get('room');
    if (urlRoom) {
      roomName = urlRoom;
    }

    // Check if already logged in
    if (isAuthenticated()) {
      const member = getMember();
      if (member) {
        currentMember.set(member);
        goto('/room');
      }
    }
  });

  async function handleRoomSubmit() {
    if (!roomName.trim()) return;
    
    isLoading.set(true);
    error.set(null);
    
    try {
      // Check if room exists and if it has password
      const roomInfo = await api.getRoomInfo(roomName);
      if (roomInfo.success && roomInfo.data) {
        if (roomInfo.data.has_password) {
          hasRoomPassword = true;
          step = 'room_password';
        } else {
          hasRoomPassword = false;
          step = 'username';
        }
      }
    } catch (e: any) {
      error.set(getErrorMessage(e, $locale));
    } finally {
      isLoading.set(false);
    }
  }

  async function handleRoomPasswordSubmit() {
    if (!roomPassword.trim()) return;
    
    isLoading.set(true);
    error.set(null);
    
    try {
      // Verify room password by attempting to join (will fail if password wrong)
      // Actually, we just proceed to username step, password will be verified on join
      step = 'username';
    } finally {
      isLoading.set(false);
    }
  }

  async function handleUsernameSubmit() {
    if (!username.trim()) return;
    
    checkingUser = true;
    error.set(null);
    
    try {
      // Check if username exists
      const result = await api.checkUser(roomName, username);
      if (result.success && result.data) {
        userExists = result.data.exists;
        userIsPersistent = result.data.is_persistent;
        
        if (userExists && userIsPersistent) {
          // Need password for existing persistent user
          step = 'user_password';
        } else if (userExists && !userIsPersistent) {
          // Username taken by anonymous user - cannot join
          error.set(t('username_taken_anonymous', $locale));
          userExists = false;
        } else {
          // New user - can create account
          step = 'create_user';
        }
      }
    } catch (e: any) {
      error.set(getErrorMessage(e, $locale));
    } finally {
      checkingUser = false;
    }
  }

  async function handleCreateUser() {
    if (!userPassword.trim()) {
      error.set(t('password_for_persistent', $locale));
      return;
    }
    
    isLoading.set(true);
    error.set(null);
    
    try {
      const success = await joinRoom(roomName, username, roomPassword || undefined, userPassword);
      if (success) {
        goto('/room');
      }
    } catch (e: any) {
      error.set(getErrorMessage(e, $locale));
    } finally {
      isLoading.set(false);
    }
  }

  async function handleLoginPersistentUser() {
    if (!userPassword.trim()) {
      error.set(t('enter_password', $locale));
      return;
    }
    
    isLoading.set(true);
    error.set(null);
    
    try {
      const success = await joinRoom(roomName, username, roomPassword || undefined, userPassword);
      if (success) {
        goto('/room');
      }
    } catch (e: any) {
      error.set(getErrorMessage(e, $locale));
    } finally {
      isLoading.set(false);
    }
  }

  async function handleAnonymousJoin() {
    isLoading.set(true);
    error.set(null);
    
    try {
      const success = await joinRoom(roomName, username, roomPassword || undefined);
      if (success) {
        goto('/room');
      }
    } finally {
      isLoading.set(false);
    }
  }

  function goBack() {
    if (step === 'room_password') {
      step = 'room';
      roomPassword = '';
    } else if (step === 'username') {
      step = hasRoomPassword ? 'room_password' : 'room';
      hasRoomPassword = false;
    } else if (step === 'user_password' || step === 'create_user') {
      step = 'username';
      userPassword = '';
    }
  }
</script>

<svelte:head>
  <title>{t('join_room', $locale)} - TomatoTogether</title>
</svelte:head>

<main class="landing">
  <div class="hero">
    <h1>🍅 TomatoTogether</h1>
    <p>{t('tagline', $locale)}</p>
  </div>

  <div class="card">
    <button class="link" onclick={() => goto('/')}>
      ← {t('back_home', $locale)}
    </button>
    
    {#if step === 'room'}
      <h2>{t('join_room_title', $locale)}</h2>
      <p class="hint">{t('enter_room_name', $locale)}</p>
      <form onsubmit={(e) => { e.preventDefault(); handleRoomSubmit(); }}>
        <div class="form-group">
          <label for="roomName">{t('room_name', $locale)}</label>
          <input
            id="roomName"
            type="text"
            bind:value={roomName}
            placeholder={t('room_name_placeholder', $locale)}
            required
          />
        </div>
        
        {#if $error}
          <p class="error">{$error}</p>
        {/if}
        
        <button class="primary" type="submit" disabled={$isLoading}>
          {$isLoading ? t('loading', $locale) : t('next', $locale)}
        </button>
      </form>
      
    {:else if step === 'room_password'}
      <h2>{t('room_password', $locale)}</h2>
      <p class="hint">{t('room_requires_password', $locale)}</p>
      <form onsubmit={(e) => { e.preventDefault(); handleRoomPasswordSubmit(); }}>
        <div class="form-group">
          <label for="roomPassword">{t('room_password', $locale)}</label>
          <input
            id="roomPassword"
            type="password"
            bind:value={roomPassword}
            placeholder={t('enter_room_password', $locale)}
            required
          />
        </div>
        
        <button class="primary" type="submit" disabled={$isLoading}>
          {t('next', $locale)}
        </button>
      </form>
      
    {:else if step === 'username'}
      <h2>{t('set_username', $locale)}</h2>
      <p class="hint">{t('choose_username', $locale)}</p>
      <form onsubmit={(e) => { e.preventDefault(); handleUsernameSubmit(); }}>
        <div class="form-group">
          <label for="username">{t('username', $locale)}</label>
          <input
            id="username"
            type="text"
            bind:value={username}
            placeholder={t('username_placeholder', $locale)}
            required
          />
        </div>
        
        {#if $error}
          <p class="error">{$error}</p>
        {/if}
        
        <button class="primary" type="submit" disabled={$isLoading || checkingUser}>
          {checkingUser ? t('loading', $locale) : t('next', $locale)}
        </button>
      </form>
      
    {:else if step === 'user_password'}
      <h2>{t('login_existing', $locale)}</h2>
      <p class="hint">{t('existing_user_notice', $locale)}</p>
      <form onsubmit={(e) => { e.preventDefault(); handleLoginPersistentUser(); }}>
        <div class="form-group">
          <label for="userPassword">{t('password', $locale)}</label>
          <input
            id="userPassword"
            type="password"
            bind:value={userPassword}
            placeholder={t('password_placeholder', $locale)}
            required
          />
        </div>
        
        {#if $error}
          <p class="error">{$error}</p>
        {/if}
        
        <button class="primary" type="submit" disabled={$isLoading}>
          {$isLoading ? t('loading', $locale) : t('join', $locale)}
        </button>
      </form>
      
    {:else if step === 'create_user'}
      <h2>{t('create_account', $locale)}</h2>
      <p class="hint">{t('password_for_persistent', $locale)}</p>
      <form onsubmit={(e) => { e.preventDefault(); handleCreateUser(); }}>
        <div class="form-group">
          <label for="userPassword">{t('set_password', $locale)}</label>
          <input
            id="userPassword"
            type="password"
            bind:value={userPassword}
            placeholder={t('password_placeholder', $locale)}
          />
          <span class="hint-small">{t('persistent_hint', $locale)}</span>
        </div>
        
        {#if $error}
          <p class="error">{$error}</p>
        {/if}
        
        <button class="primary" type="submit" disabled={$isLoading}>
          {$isLoading ? t('loading', $locale) : t('join_room', $locale)}
        </button>
        
        {#if !userPassword}
          <button class="secondary" type="button" onclick={handleAnonymousJoin} disabled={$isLoading}>
            {t('join_anonymous', $locale)}
          </button>
        {/if}
      </form>
    {/if}
    
    {#if step !== 'room'}
      <button class="link" onclick={goBack}>
        ← {t('back', $locale)}
      </button>
    {/if}
  </div>
</main>

<style>
  .landing {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 2rem;
  }

  .hero {
    text-align: center;
    margin-bottom: 2rem;
  }

  .hero h1 {
    font-size: 2.5rem;
    margin-bottom: 0.5rem;
  }

  .hero p {
    color: var(--color-text-secondary);
    font-size: 1.125rem;
  }

  .card {
    width: 100%;
    max-width: 400px;
  }

  .card h2 {
    margin-bottom: 0.5rem;
    text-align: center;
  }

  .hint {
    color: var(--color-text-secondary);
    font-size: 0.875rem;
    text-align: center;
    margin-bottom: 1.5rem;
  }

  .hint-small {
    color: var(--color-text-secondary);
    font-size: 0.75rem;
    display: block;
    margin-top: 0.25rem;
  }

  .link {
    background: none;
    padding: 0;
    color: var(--color-text-secondary);
    margin-bottom: 1rem;
    text-align: left;
    width: 100%;
  }

  .link:hover {
    color: var(--color-text);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .form-group {
    display: flex;
    flex-direction: column;
  }

  button[type="submit"],
  button[type="button"] {
    width: 100%;
  }

  .secondary {
    margin-top: 0.5rem;
  }
</style>
