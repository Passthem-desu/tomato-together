<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { currentMember, joinRoom, isLoading, error } from '$lib/store';
	import { getMember, isAuthenticated, api } from '$lib/api';
	import { locale, t, getErrorMessage } from '$lib/i18n';
	import LocaleSwitcher from '$lib/components/LocaleSwitcher.svelte';

	let step = $state('room');
	let roomName = $state('');
	let roomPassword = $state('');
	let username = $state('');
	let userPassword = $state('');
	let hasRoomPassword = $state(false);
	let userExists = $state(false);
	let userIsPersistent = $state(false);
	let checkingUser = $state(false);
	let showUserPassword = $state(false);
	let userPasswordConfirm = $state('');

	$effect(() => {
		const _step = step;
		setTimeout(() => {
			const input = document.querySelector(
				'input:not([type="hidden"]):not([disabled])'
			) as HTMLInputElement;
			input?.focus();
		}, 50);
	});

	onMount(() => {
		error.set(null);

		const urlRoom = $page.url.searchParams.get('room');
		if (urlRoom) {
			roomName = urlRoom;
		}

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
			const result = await api.checkRoomPassword(roomName, roomPassword);
			if (result.success && result.data?.valid) {
				step = 'username';
			} else {
				error.set(t('error_invalid_room_password', $locale));
			}
		} catch (e: any) {
			error.set(getErrorMessage(e, $locale));
		} finally {
			isLoading.set(false);
		}
	}

	async function handleUsernameSubmit() {
		if (!username.trim()) return;

		checkingUser = true;
		error.set(null);

		try {
			const result = await api.checkUser(roomName, username);
			if (result.success && result.data) {
				userExists = result.data.exists;
				userIsPersistent = result.data.is_persistent;

				if (userExists && userIsPersistent) {
					step = 'user_password';
				} else if (userExists && !userIsPersistent) {
					// Anonymous user exists - allow re-join (set password to upgrade or join as anonymous)
					step = 'create_user';
				} else {
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
			const success = await joinRoom(
				roomName,
				username,
				roomPassword || undefined,
				userPassword
			);
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
			const success = await joinRoom(
				roomName,
				username,
				roomPassword || undefined,
				userPassword
			);
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

<main class="join-room">
	<button class="btn-text back-btn" onclick={() => goto('/')}>
		← {t('back_home', $locale)}
	</button>

	<div class="hero">
		<h1>{t('join_room_title', $locale)}</h1>
		<p class="step-hint">
			{#if step === 'room'}
				{t('enter_room_name', $locale)}
			{:else if step === 'room_password'}
				{t('room_requires_password', $locale)}
			{:else if step === 'username'}
				{t('choose_username', $locale)}
			{:else if step === 'user_password'}
				{t('login_existing', $locale)}
			{:else if step === 'create_user'}
				{t('create_account', $locale)}
			{/if}
		</p>
	</div>

	<div class="form-card card">
		{#if step === 'room'}
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleRoomSubmit();
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

				{#if $error}
					<p class="form-error">{$error}</p>
				{/if}

				<button class="btn-primary btn-full" type="submit" disabled={$isLoading}>
					{#if $isLoading}
						<span class="spinner"></span>
					{:else}
						{t('next', $locale)} →
					{/if}
				</button>
			</form>
		{:else if step === 'room_password'}
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleRoomPasswordSubmit();
				}}
			>
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

				{#if $error}
					<p class="form-error">{$error}</p>
				{/if}

				<button class="btn-primary btn-full" type="submit" disabled={$isLoading}>
					{t('next', $locale)} →
				</button>
			</form>
		{:else if step === 'username'}
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleUsernameSubmit();
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

				{#if $error}
					<p class="form-error">{$error}</p>
				{/if}

				<button
					class="btn-primary btn-full"
					type="submit"
					disabled={$isLoading || checkingUser}
				>
					{#if checkingUser}
						<span class="spinner"></span>
						{t('checking', $locale)}
					{:else}
						{t('next', $locale)} →
					{/if}
				</button>
			</form>
		{:else if step === 'user_password'}
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleLoginPersistentUser();
				}}
			>
				<div class="user-info">
					<span class="user-name">{username}</span>
				</div>

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
					<p class="form-error">{$error}</p>
				{/if}

				<button class="btn-primary btn-full" type="submit" disabled={$isLoading}>
					{#if $isLoading}
						<span class="spinner"></span>
					{:else}
						{t('join', $locale)}
					{/if}
				</button>
			</form>
		{:else if step === 'create_user'}
			<form
				onsubmit={(e) => {
					e.preventDefault();
					handleCreateUser();
				}}
			>
				<!-- Anonymous join button first (above password) -->
				<button
					class="btn-secondary btn-full"
					type="button"
					onclick={handleAnonymousJoin}
					disabled={$isLoading}
				>
					{t('join_anonymous', $locale)}
				</button>

				<div class="divider"><span>{t('or_set_password', $locale)}</span></div>

				<div class="form-group">
					<label for="userPassword">{t('set_password', $locale)}</label>
					<div class="password-wrapper">
						<input
							id="userPassword"
							type={showUserPassword ? 'text' : 'password'}
							bind:value={userPassword}
							placeholder={t('password_placeholder', $locale)}
						/>
						<button
							type="button"
							class="btn-eye"
							onclick={() => (showUserPassword = !showUserPassword)}
						>
							{showUserPassword ? '🙈' : '👁'}
						</button>
					</div>
					{#if userPassword}
						<div class="form-group" style="margin-top:0.5rem">
							<label for="userPasswordConfirm">{t('confirm_password', $locale)}</label
							>
							<input
								id="userPasswordConfirm"
								type="password"
								bind:value={userPasswordConfirm}
								placeholder={t('confirm_password_placeholder', $locale)}
							/>
						</div>
					{/if}
					<span class="form-hint">{t('persistent_hint', $locale)}</span>
				</div>

				{#if $error}
					<p class="form-error">{$error}</p>
				{/if}

				{#if userPassword}
					<button
						class="btn-primary btn-full"
						type="submit"
						disabled={$isLoading || userPassword !== userPasswordConfirm}
					>
						{#if $isLoading}
							<span class="spinner"></span>
						{:else}
							{t('join_room', $locale)}
						{/if}
					</button>
				{/if}
			</form>
		{/if}

		{#if step !== 'room'}
			<button class="btn-text back-link" onclick={goBack}>
				← {t('back', $locale)}
			</button>
		{/if}
	</div>

	<LocaleSwitcher fixed />
</main>

<style>
	.join-room {
		min-height: 100dvh;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 2rem;
		background: var(--color-bg-0);
	}

	.back-btn {
		position: fixed;
		top: 1.25rem;
		left: 1.25rem;
		color: var(--color-fg-muted);
	}

	.back-btn:hover {
		color: var(--color-fg-0);
	}

	.hero {
		text-align: center;
		margin-bottom: 2rem;
	}

	.hero h1 {
		font-size: var(--text-2xl);
		margin-bottom: 0.5rem;
	}

	.step-hint {
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

	.user-info {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		padding: 0.75rem;
		background: var(--color-bg-2);
		border-radius: var(--radius-md);
		margin-bottom: 0.5rem;
	}

	.user-avatar {
		font-size: 1.5rem;
	}

	.user-name {
		font-weight: 600;
		color: var(--color-fg-0);
	}

	.new-user-notice {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		padding: 0.75rem;
		background: var(--color-brand-subtle);
		border-radius: var(--radius-md);
		color: var(--color-brand);
		margin-bottom: 0.5rem;
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

	.password-wrapper {
		position: relative;
	}
	.password-wrapper input {
		padding-right: 2.5rem;
	}
	.btn-eye {
		position: absolute;
		right: 0.25rem;
		top: 50%;
		transform: translateY(-50%);
		background: none;
		border: none;
		cursor: pointer;
		padding: 0.5rem;
		font-size: 1rem;
	}
	.divider {
		display: flex;
		align-items: center;
		margin: 1rem 0;
		color: var(--color-fg-muted);
		font-size: var(--text-xs);
	}
	.divider::before,
	.divider::after {
		content: '';
		flex: 1;
		border-bottom: 1px solid var(--color-border);
	}
	.divider span {
		padding: 0 0.75rem;
	}

	@media (max-width: 640px) {
		.form-card {
			padding: 1.25rem;
		}
	}
</style>
