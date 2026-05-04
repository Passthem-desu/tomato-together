<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { currentMember } from '$lib/store';
  import { getMember, isAuthenticated } from '$lib/api';
  import { locale, locales, t, type Locale } from '$lib/i18n';

  let showLangMenu = $state(false);

  onMount(() => {
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
    <div class="logo">🍅</div>
    <h1>TomatoTogether</h1>
    <p class="tagline">{t('tagline', $locale)}</p>
  </div>

  <div class="actions card">
    <button class="btn-primary btn-full" onclick={() => goto('/create')}>
      {t('create_room', $locale)}
    </button>
    <button class="btn-secondary btn-full" onclick={() => goto('/join')}>
      {t('join_room', $locale)}
    </button>
  </div>

  <div class="lang-selector">
    <button class="lang-btn" onclick={() => showLangMenu = !showLangMenu}>
      <span>{locales.find(l => l.code === $locale)?.name}</span>
      <span class="arrow" class:open={showLangMenu}>▼</span>
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
    min-height: 100dvh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 2rem;
    background: var(--color-bg-0);
  }

  .hero {
    text-align: center;
    margin-bottom: 2.5rem;
  }

  .logo {
    font-size: 4rem;
    margin-bottom: 1rem;
    animation: float 3s ease-in-out infinite;
  }

  @keyframes float {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-8px); }
  }

  .hero h1 {
    font-size: var(--text-4xl);
    font-weight: 700;
    letter-spacing: -0.02em;
    margin-bottom: 0.5rem;
  }

  .tagline {
    font-size: var(--text-lg);
    color: var(--color-fg-muted);
  }

  .actions {
    width: 100%;
    max-width: 320px;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  /* Language Selector */
  .lang-selector {
    position: fixed;
    top: 1.25rem;
    right: 1.25rem;
  }

  .lang-btn {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.875rem;
    background: var(--color-bg-1);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    font-size: var(--text-sm);
    color: var(--color-fg-1);
  }

  .lang-btn:hover {
    background: var(--color-bg-2);
    border-color: var(--color-fg-muted);
  }

  .arrow {
    font-size: 0.625rem;
    transition: transform var(--duration-fast) var(--ease-out);
  }

  .arrow.open {
    transform: rotate(180deg);
  }

  .lang-menu {
    position: absolute;
    top: calc(100% + 0.5rem);
    right: 0;
    background: var(--color-bg-1);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-lg);
    overflow: hidden;
    min-width: 120px;
    animation: fadeIn var(--duration-fast) var(--ease-out);
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(-4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .lang-option {
    width: 100%;
    padding: 0.625rem 1rem;
    text-align: left;
    font-size: var(--text-sm);
    color: var(--color-fg-1);
    border-radius: 0;
  }

  .lang-option:hover {
    background: var(--color-bg-2);
  }

  .lang-option.active {
    background: var(--color-brand-subtle);
    color: var(--color-brand);
  }

  @media (max-width: 640px) {
    .hero h1 {
      font-size: var(--text-3xl);
    }

    .logo {
      font-size: 3rem;
    }
  }
</style>