<script lang="ts">
	import { onMount } from 'svelte';
	import { locale } from '$lib/i18n';
	import { locales, type Locale } from '$lib/i18n/locales';

	interface Props {
		/** If true, use fixed positioning (top-right). Otherwise relative (inline). */
		fixed?: boolean;
	}

	let { fixed = false }: Props = $props();

	let showMenu = $state(false);

	function selectLocale(code: Locale) {
		locale.set(code);
		showMenu = false;
	}

	function handleClickOutside(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (!target.closest('.lang-selector')) {
			showMenu = false;
		}
	}

	onMount(() => {
		document.addEventListener('click', handleClickOutside);
		return () => document.removeEventListener('click', handleClickOutside);
	});
</script>

<div class="lang-selector" class:fixed>
	<button class="lang-btn" onclick={() => (showMenu = !showMenu)}>
		<span>{locales.find((l) => l.code === $locale)?.name}</span>
		<span class="arrow" class:open={showMenu}>▼</span>
	</button>

	{#if showMenu}
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

<style>
	.lang-selector {
		position: relative;
		overflow: visible;
	}
	.lang-selector.fixed {
		position: fixed;
		top: 1.25rem;
		right: 1.25rem;
		z-index: 10;
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
		white-space: nowrap;
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
		display: flex;
		flex-direction: column;
		background: var(--color-bg-1);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-lg);
		overflow: hidden;
		min-width: 120px;
		z-index: 50;
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
</style>
