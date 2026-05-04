<script lang="ts">
	import { locale, t } from '$lib/i18n';
	import { api } from '$lib/api';
	import type { UserInfo } from '$lib/api';

	interface Props {
		users: UserInfo[];
		currentMemberId: string;
		isOwner: boolean;
	}

	let { users, currentMemberId, isOwner }: Props = $props();

	let myMessage = $state('');
	let statusSaving = $state(false);

	// Status bubble animation: memberId -> { message, timeoutId }
	let statusBubbles = $state<Record<string, { message: string }>>({});
	let prevStatuses: Record<string, string> = {};
	let showOwnerActions = $state(false);

	// Detect status changes and show bubble
	$effect(() => {
		for (const u of users) {
			if (!u.id) continue;
			const prev = prevStatuses[u.id];
			const curr = u.status?.message || '';
			if (curr && curr !== prev) {
				// Show bubble, clear after 3s
				statusBubbles = { ...statusBubbles, [u.id]: { message: curr } };
				setTimeout(() => {
					statusBubbles = Object.fromEntries(
						Object.entries(statusBubbles).filter(([k]) => k !== u.id)
					);
				}, 3000);
			}
			prevStatuses = { ...prevStatuses, [u.id]: curr };
		}
	});

	function formatTime(s: number) {
		return `${Math.floor(s / 60)
			.toString()
			.padStart(2, '0')}:${(s % 60).toString().padStart(2, '0')}`;
	}

	function formatMinutes(sec: number) {
		if (!sec) return '';
		const m = Math.round(sec / 60);
		if (m < 60) return m + 'm';
		return Math.floor(m / 60) + 'h ' + (m % 60) + 'm';
	}

	async function handleSetStatus() {
		if (!myMessage.trim()) return;
		statusSaving = true;
		try {
			const roomName = localStorage.getItem('room_name') || '';
			await api.updateStatus({ room_name: roomName, emoji: '', message: myMessage.trim() });
			myMessage = '';
		} catch {
			/* ignore */
		}
		statusSaving = false;
	}

	async function handleClearStatus() {
		statusSaving = true;
		try {
			const roomName = localStorage.getItem('room_name') || '';
			await api.deleteStatus(roomName);
		} catch {
			/* ignore */
		}
		statusSaving = false;
	}

	async function handleKick(memberId: string) {
		if (!confirm(t('confirm_kick', $locale))) return;
		try {
			await api.kickMember(memberId);
		} catch {
			/* ignore */
		}
	}

	async function handleTransferOwner(memberId: string) {
		if (!confirm(t('confirm_transfer_owner', $locale))) return;
		try {
			await api.setOwner(memberId);
		} catch {
			/* ignore */
		}
	}
</script>

<div class="card users-card">
	<div class="users-header">
		<h2>{t('online_users', $locale)}</h2>
		{#if isOwner}
			<button
				class="btn-ghost btn-xs btn-toggle-actions"
				class:active={showOwnerActions}
				onclick={() => (showOwnerActions = !showOwnerActions)}
			>
				⚙
			</button>
		{/if}
	</div>

	<!-- Status input for current user -->
	<div class="status-input-row">
		<input
			type="text"
			class="status-msg-input"
			maxlength="200"
			placeholder={t('status_message_placeholder', $locale)}
			bind:value={myMessage}
			onkeydown={(e) => {
				if (e.key === 'Enter') handleSetStatus();
			}}
		/>
		<button class="btn-ghost btn-xs" onclick={handleSetStatus} disabled={statusSaving}>
			{t('save_settings', $locale)}
		</button>
		<button
			class="btn-ghost btn-xs btn-clear-status"
			onclick={handleClearStatus}
			disabled={statusSaving}
			title={t('clear_status', $locale)}
		>
			×
		</button>
	</div>
	{#if users.filter((u) => u.is_online).length === 0}
		<p class="empty-state">{t('no_online_users', $locale)}</p>
	{:else}
		<div class="users-list">
			{#each users.filter((u) => u.is_online) as user}
				<div class="user-item">
					<div class="user-info">
						<div class="user-details">
							<span class="user-name">
								{user.username}
								{#if user.id === currentMemberId}<span class="you-badge"
										>{t('me', $locale)}</span
									>{/if}
								{#if user.is_owner}<span class="owner-badge"
										>{t('owner', $locale)}</span
									>{/if}
							</span>
							{#if user.total_pomodoros !== undefined && user.total_pomodoros > 0}
								<span class="user-stats"
									>{user.total_pomodoros}
									{t('pomodoros_short', $locale)} / {formatMinutes(
										user.total_duration || 0
									)}
									{t('focus_short', $locale)}</span
								>
							{/if}
							{#if user.status?.message}
								<span class="user-status">{user.status.message}</span>
							{/if}
						</div>
					</div>
					<div class="user-pomodoro">
						{#if showOwnerActions && isOwner && user.id !== currentMemberId}
							{#if user.is_persistent}
								<button
									class="btn-transfer btn-xs"
									onclick={() => handleTransferOwner(user.id)}
									>{t('transfer_owner', $locale)}</button
								>
							{/if}
							<button class="btn-kick btn-xs" onclick={() => handleKick(user.id)}
								>{t('kick', $locale)}</button
							>
						{:else if user.pomodoro?.phase && user.pomodoro.phase !== 'idle'}
							{#if user.pomodoro.is_following}
								<span class="following"
									>{t('following', $locale)} {user.pomodoro.leader_username}</span
								>
							{:else if user.pomodoro.phase === 'paused'}
								<span class="paused-status">{t('paused', $locale)}</span>
							{:else if user.pomodoro.phase === 'rest'}
								<span class="rest-status"
									>{t('resting', $locale)}
									{user.pomodoro.remaining_seconds
										? formatTime(user.pomodoro.remaining_seconds)
										: ''}</span
								>
							{:else}
								<span class="active-pomodoro"
									>{user.pomodoro.remaining_seconds
										? formatTime(user.pomodoro.remaining_seconds)
										: ''}</span
								>
							{/if}
						{:else}
							<span class="idle-status">{t('idle', $locale)}</span>
						{/if}
					</div>
					{#if statusBubbles[user.id]}
						<div class="status-bubble">{statusBubbles[user.id].message}</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.users-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.75rem;
	}
	.users-header h2 {
		font-size: var(--text-base);
		font-weight: 600;
		color: var(--color-fg-1);
		margin: 0;
		line-height: 1;
	}
	.btn-toggle-actions {
		font-size: 0.8rem;
		opacity: 0.4;
		padding: 0.15rem 0.3rem;
		line-height: 1;
		transition: opacity 0.15s;
	}
	.btn-toggle-actions:hover,
	.btn-toggle-actions.active {
		opacity: 1;
	}
	.users-card h2 {
		font-size: var(--text-base);
		font-weight: 600;
		color: var(--color-fg-1);
	}
	.status-input-row {
		display: flex;
		gap: 0.375rem;
		margin-bottom: 0.75rem;
		padding-bottom: 0.75rem;
		border-bottom: 1px solid var(--color-border);
	}
	.status-msg-input {
		flex: 1;
		padding: 0.3rem 0.5rem;
		font-size: var(--text-sm);
	}
	.empty-state {
		text-align: center;
		color: var(--color-fg-muted);
		padding: 1.5rem;
	}
	.users-list {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.user-item {
		position: relative;
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem;
		background: var(--color-bg-0);
		border-radius: var(--radius-md);
	}
	.user-item:hover {
		background: var(--color-bg-2);
	}
	.user-info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}
	.user-details {
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
	}
	.user-details .user-name {
		font-weight: 500;
		display: flex;
		align-items: center;
		gap: 0.375rem;
	}
	.you-badge {
		font-size: var(--text-xs);
		padding: 0.125rem 0.375rem;
		background: var(--color-bg-2);
		border-radius: var(--radius-sm);
		color: var(--color-fg-muted);
	}
	.owner-badge {
		font-size: var(--text-xs);
		padding: 0.125rem 0.375rem;
		background: var(--color-brand-subtle);
		border-radius: var(--radius-sm);
		color: var(--color-brand);
	}
	.user-status {
		font-size: var(--text-xs);
		color: var(--color-fg-muted);
	}
	.user-stats {
		font-size: 0.7rem;
		color: var(--color-fg-muted);
	}
	.user-pomodoro .active-pomodoro {
		font-weight: 600;
		color: var(--color-brand);
	}
	.user-pomodoro .following {
		font-size: var(--text-sm);
		color: var(--color-fg-muted);
	}
	.user-pomodoro .paused-status {
		font-size: var(--text-sm);
		color: var(--color-warning);
	}
	.user-pomodoro .rest-status {
		font-size: var(--text-sm);
		color: var(--color-success);
	}
	.user-pomodoro .idle-status {
		font-size: var(--text-sm);
		color: var(--color-fg-muted);
	}
	.btn-kick {
		padding: 0.15rem 0.4rem;
		font-size: 0.65rem;
		border: 1px solid var(--color-error);
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--color-error);
		cursor: pointer;
		transition: all 0.15s;
	}
	.btn-kick:hover {
		background: var(--color-error-subtle);
	}
	.btn-transfer {
		padding: 0.15rem 0.4rem;
		font-size: 0.65rem;
		border: 1px solid var(--color-brand);
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--color-brand);
		cursor: pointer;
		transition: all 0.15s;
		margin-right: 0.25rem;
	}
	.btn-transfer:hover {
		background: var(--color-brand-subtle);
	}
	.status-bubble {
		position: absolute;
		left: -0.75rem;
		top: -0.5rem;
		background: var(--color-brand);
		color: white;
		font-size: var(--text-xs);
		padding: 0.25rem 0.6rem;
		border-radius: var(--radius-md);
		white-space: nowrap;
		animation:
			bubbleIn 0.3s ease-out,
			bubbleOut 0.5s ease-in 2.5s forwards;
		z-index: 10;
		pointer-events: none;
	}
	@keyframes bubbleIn {
		from {
			opacity: 0;
			transform: translateY(0.5rem);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}
	@keyframes bubbleOut {
		to {
			opacity: 0;
			transform: translateY(-0.5rem);
		}
	}
	.btn-clear-status {
		color: var(--color-fg-muted);
		font-weight: bold;
		padding: 0.3rem 0.5rem;
	}
	.btn-clear-status:hover {
		color: var(--color-error);
	}
	@media (max-width: 640px) {
		.user-item {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.5rem;
		}
	}
</style>
