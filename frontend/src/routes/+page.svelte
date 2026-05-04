<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { currentMember } from '$lib/store';
  import { getMember, isAuthenticated } from '$lib/api';
  import { locale, locales, t, type Locale } from '$lib/i18n';

  let showLangMenu = $state(false);

  onMount(() => {
    // Check if already logged in
    if (isAuthenticated()) {
      const member = getMember();
      if (member) {
        currentMember.set(member);
        goto('/room');
      }
    }
  });

  function selectLocale(code: Locale) {
    locale.set(code);
    showLangMenu = false;
  }

  function toggleLangMenu() {
    showLangMenu = !showLangMenu;
  }

  // Close menu when clicking outside
  function handleClickOutside(event: MouseEvent) {
    const target = event.target as HTMLElement;
    if (!target.closest('.lang-selector')) {
      showLangMenu = false;
    }
  }
</script>

<svelte:head>
  <title>TomatoTogether</title>
</svelte:head>

<svelte:window onclick={handleClickOutside} />

<main class="landing">
  <div class="hero">
    <h1>🍅 TomatoTogether</h1>
    <p>{t('tagline', $locale)}</p>
  </div>

  <div class="card actions">
    <button class="primary" onclick={() => goto('/create')}>
      {t('create_room', $locale)}
    </button>
    <button class="secondary" onclick={() => goto('/join')}>
      {t('join_room', $locale)}
    </button>
  </div>

  <div class="lang-selector">
    <button class="lang-btn" onclick={toggleLangMenu}>
      <span class="lang-name">{locales.find(l => l.code === $locale)?.name}</span>
      <span class="arrow">{showLangMenu ? '▲' : '▼'}</span>
    </button>
    
    {#if showLangMenu}
      <div class="lang-menu">
        {#each locales as loc}
          <button 
            class="lang-option" 
            class:active={$locale === loc.code}
            onclick={() => selectLocale(loc.code)}
          >
            {loc.name}
          </button>
        {/each}
      </div>
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

  .actions {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .actions button {
    width: 100%;
  }

  .lang-selector {
    position: fixed;
    top: 1rem;
    right: 1rem;
    z-index: 100;
  }

  .lang-btn {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 0.5rem;
    cursor: pointer;
    font-size: 0.875rem;
  }

  .lang-btn:hover {
    background: var(--color-surface-hover);
  }

  .lang-name {
    min-width: 60px;
  }

  .arrow {
    font-size: 0.625rem;
    color: var(--color-text-secondary);
  }

  .lang-menu {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 0.25rem;
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: 0.5rem;
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
    overflow: hidden;
    min-width: 140px;
  }

  .lang-option {
    width: 100%;
    padding: 0.75rem 1rem;
    border: none;
    background: none;
    cursor: pointer;
    text-align: left;
    font-size: 0.875rem;
  }

  .lang-option:hover {
    background: var(--color-surface-hover);
  }

  .lang-option.active {
    background: var(--color-primary-light);
  }
</style>
