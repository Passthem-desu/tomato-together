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

	@media (max-width: 640px) {
		.hero h1 {
			font-size: var(--text-3xl);
		}

		.logo {
			font-size: 3rem;
		}
	}
</style>
