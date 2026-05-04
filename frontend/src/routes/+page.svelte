<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { currentMember, currentRoom } from '$lib/store';
	import { getMember, getRoom, isAuthenticated } from '$lib/api';
	import { locale, t } from '$lib/i18n';
	import LocaleSwitcher from '$lib/components/LocaleSwitcher.svelte';

	onMount(() => {
		if (isAuthenticated()) {
			const member = getMember();
			if (member) {
				currentMember.set(member);
				const room = getRoom();
				if (room) {
					currentRoom.set(room);
				}
				goto('/room');
			}
		}
	});
</script>

<svelte:head>
	<title>TomatoTogether</title>
</svelte:head>

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

	<a
		class="github-link"
		href="https://github.com/Passthem-desu/tomato-together"
		target="_blank"
		rel="noopener noreferrer"
	>
		<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
			<path
				d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"
			/>
		</svg>
		GitHub
	</a>

	<LocaleSwitcher fixed />
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
		0%,
		100% {
			transform: translateY(0);
		}
		50% {
			transform: translateY(-8px);
		}
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

	.github-link {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		margin-top: 1.5rem;
		font-size: var(--text-sm);
		color: var(--color-fg-muted);
		text-decoration: none;
		transition: color 0.2s;
	}

	.github-link:hover {
		color: var(--color-fg);
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
