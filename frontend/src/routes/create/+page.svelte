<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { currentMember, createRoom, isLoading, error } from '$lib/store';
  import { getMember, isAuthenticated } from '$lib/api';
  import { locale, t, getErrorMessage } from '$lib/i18n';

  // Step 1: room info, Step 2: user info
  let step = $state(1);
  let roomName = $state('');
  let roomPassword = $state('');
  let username = $state('');
  let userPassword = $state('');

  onMount(() => {
    // Clear any previous error
    error.set(null);

    // Check if already logged in
    if (isAuthenticated()) {
      const member = getMember();
      if (member) {
        currentMember.set(member);
        goto('/room');
      }
    }
    // Focus roomName input on mount
    document.getElementById('roomName')?.focus();
  });

  // Focus input when step changes
  $effect(() => {
    const _ = step; // track step changes
    setTimeout(() => {
      if (step === 1) {
        document.getElementById('roomName')?.focus();
      } else {
        document.getElementById('username')?.focus();
      }
    }, 50);
  });

  function handleStep1() {
    if (!roomName.trim()) return;
    step = 2;
  }

  function handleStep2() {
    if (!username.trim() || !userPassword.trim()) {
      if (!userPassword) {
        error.set(t('error_password_required', $locale));
      }
      return;
    }
    
    isLoading.set(true);
    error.set(null);
    
    createRoom(roomName, username, userPassword, roomPassword || undefined)
      .then(success => {
        if (success) {
          goto('/room');
        }
      })
      .catch((e: any) => {
        error.set(getErrorMessage(e, $locale));
      })
      .finally(() => {
        isLoading.set(false);
      });
  }

  function goBack() {
    step = 1;
  }
</script>

<svelte:head>
  <title>{t('create_room', $locale)} - TomatoTogether</title>
</svelte:head>

<main class="create-room">
  <div class="hero">
    <h1>🍅 TomatoTogether</h1>
    <p>{t('create_room_title', $locale)}</p>
  </div>

  <div class="card">
    {#if step === 1}
      <h2>{t('room_info', $locale)}</h2>
      <p class="hint">{t('room_name', $locale)}</p>
      
      <form onsubmit={(e) => { e.preventDefault(); handleStep1(); }}>
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
        
        <div class="form-group">
          <label for="roomPassword">{t('room_password', $locale)} <span class="optional">{t('room_password_optional', $locale)}</span></label>
          <input
            id="roomPassword"
            type="password"
            bind:value={roomPassword}
            placeholder={t('room_password', $locale)}
          />
          <span class="hint-small">{t('room_password_hint', $locale)}</span>
        </div>
        
        {#if $error}
          <p class="error">{$error}</p>
        {/if}
        
        <button class="primary" type="submit" disabled={$isLoading}>
          {t('next', $locale)}
        </button>
      </form>
      
    {:else}
      <h2>{t('account_info', $locale)}</h2>
      <p class="hint">{t('owner_notice', $locale)}</p>
      
      <form onsubmit={(e) => { e.preventDefault(); handleStep2(); }}>
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
        
        <div class="form-group">
          <label for="userPassword">{t('set_password', $locale)}</label>
          <input
            id="userPassword"
            type="password"
            bind:value={userPassword}
            placeholder={t('password_placeholder', $locale)}
            required
          />
          <span class="hint-small">{t('password_hint', $locale)}</span>
        </div>
        
        {#if $error}
          <p class="error">{$error}</p>
        {/if}
        
        <button class="primary" type="submit" disabled={$isLoading}>
          {$isLoading ? t('loading', $locale) : t('create', $locale)}
        </button>
      </form>
      
      <button class="link" onclick={goBack}>
        ← {t('back', $locale)}
      </button>
    {/if}
    
    <button class="link cancel" onclick={() => goto('/')}>
      {t('cancel', $locale)}
    </button>
  </div>
</main>

<style>
  .create-room {
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

  .optional {
    color: var(--color-text-secondary);
    font-size: 0.875rem;
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
    margin-top: 1rem;
    text-align: center;
    width: 100%;
    display: block;
  }

  .link:hover {
    color: var(--color-text);
  }

  .link.cancel {
    margin-top: 0.5rem;
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

  button[type="submit"] {
    width: 100%;
    margin-top: 0.5rem;
  }
</style>
