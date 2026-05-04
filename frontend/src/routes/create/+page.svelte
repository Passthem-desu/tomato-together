<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { currentMember, createRoom, isLoading, error } from '$lib/store';
	import { api, getMember, isAuthenticated } from '$lib/api';
	import { locale, t, getErrorMessage } from '$lib/i18n';
	import LocaleSwitcher from '$lib/components/LocaleSwitcher.svelte';

	let step = $state(1);
	let roomName = $state('');
	let roomPassword = $state('');
	let username = $state('');
	let userPassword = $state('');

	onMount(() => {
		error.set(null);

		if (isAuthenticated()) {
			const member = getMember();
			if (member) {
				currentMember.set(member);
				goto('/room');
			}
		}

		const input = document.getElementById('roomName');
		input?.focus();
	});

	$effect(() => {
		const _ = step;
		setTimeout(() => {
			const id = step === 1 ? 'roomName' : 'username';
			document.getElementById(id)?.focus();
		}, 50);
	});

	function handleStep1() {
		if (!roomName.trim()) return;

		isLoading.set(true);
		error.set(null);

		api.getRoomInfo(roomName)
			.then(() => {
				// Room exists — name taken
				error.set(t('error_room_name_taken', $locale));
			})
			.catch((e) => {
				// 404 means available
				if (e?.message === 'room_not_found') {
					step = 2;
				} else {
					error.set(getErrorMessage(e, $locale));
				}
			})
			.finally(() => isLoading.set(false));
	}

	function handleStep2() {
		if (!username.trim() || !userPassword.trim()) return;

		isLoading.set(true);
		error.set(null);

		createRoom(roomName, username, userPassword, roomPassword || undefined)
			.then((success) => {
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
		<div class="step-indicator">
			<span class="step" class:active={step === 1} class:completed={step > 1}>1</span>
			<span class="step-line"></span>
			<span class="step" class:active={step === 2}>2</span>
		</div>
		<h1>{step === 1 ? t('room_info', $locale) : t('account_info', $locale)}</h1>
		<p class="subtitle">
			{step === 1 ? t('create_room_title', $locale) : t('owner_notice', $locale)}
		</p>
	</div>

	<div class="form-card card">
		{#if step === 1}
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleStep1();
				}}
			>
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
					<label for="roomPassword">
						{t('room_password', $locale)}
						<span class="optional">({t('optional', $locale)})</span>
					</label>
					<input
						id="roomPassword"
						type="password"
						bind:value={roomPassword}
						placeholder={t('room_password_placeholder', $locale)}
					/>
					<span class="form-hint">{t('room_password_hint', $locale)}</span>
				</div>

				{#if $error}
					<p class="form-error">{$error}</p>
				{/if}

				<button class="btn-primary btn-full" type="submit">
					{t('next', $locale)} →
				</button>
			</form>
		{:else}
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleStep2();
				}}
			>
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
					<span class="form-hint">{t('password_hint', $locale)}</span>
				</div>

				{#if $error}
					<p class="form-error">{$error}</p>
				{/if}

				<button class="btn-primary btn-full" type="submit" disabled={$isLoading}>
					{#if $isLoading}
						<span class="spinner"></span>
						{t('loading', $locale)}
					{:else}
						{t('create_room', $locale)}
					{/if}
				</button>
			</form>

			<button class="btn-text btn-full" onclick={goBack}>
				← {t('back', $locale)}
			</button>
		{/if}

		<button class="btn-text cancel" onclick={() => goto('/')}>
			{t('cancel', $locale)}
		</button>
	</div>

	<LocaleSwitcher fixed />
</main>

<style>
	.create-room {
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
		margin-bottom: 2rem;
	}

	.step-indicator {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.step {
		width: 2rem;
		height: 2rem;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: var(--radius-full);
		background: var(--color-bg-2);
		color: var(--color-fg-muted);
		font-size: var(--text-sm);
		font-weight: 600;
		transition: all var(--duration-fast) var(--ease-out);
	}

	.step.active {
		background: var(--color-brand);
		color: white;
	}

	.step.completed {
		background: var(--color-success);
		color: white;
	}

	.step-line {
		width: 3rem;
		height: 2px;
		background: var(--color-border);
	}

	.hero h1 {
		font-size: var(--text-2xl);
		margin-bottom: 0.5rem;
	}

	.subtitle {
		color: var(--color-fg-muted);
		font-size: var(--text-sm);
	}

	.form-card {
		width: 100%;
		max-width: 400px;
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.form-card form {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.optional {
		color: var(--color-fg-muted);
		font-weight: 400;
	}

	.cancel {
		color: var(--color-fg-muted);
		font-size: var(--text-sm);
		margin-top: 0.5rem;
	}

	.spinner {
		width: 1rem;
		height: 1rem;
		border: 2px solid transparent;
		border-top-color: currentColor;
		border-radius: var(--radius-full);
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.form-card button[type='submit'] {
		margin-top: 0.5rem;
	}

	@media (max-width: 640px) {
		.form-card {
			padding: 1.25rem;
		}
	}
</style>
