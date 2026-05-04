<script lang="ts">
	import { locale, t } from '$lib/i18n';
	import type { UserInfo } from '$lib/api';

	interface Props {
		users: UserInfo[];
		currentMemberId: string;
	}

	let { users, currentMemberId }: Props = $props();

	function formatTime(s: number) {
		return `${Math.floor(s / 60)
			.toString()
			.padStart(2, '0')}:${(s % 60).toString().padStart(2, '0')}`;
	}
</script>

<div class="card users-card">
	<h2>{t('online_users', $locale)}</h2>
	{#if users.filter((u) => u.is_online).length === 0}
		<p class="empty-state">{t('no_online_users', $locale)}</p>
	{:else}
		<div class="users-list">
			{#each users.filter((u) => u.is_online) as user}
				<div class="user-item">
					<div class="user-info">
						<div class="user-details">
							<span class="user-name"
								>{user.username}
								{#if user.id === currentMemberId}<span class="you-badge"
										>{t('me', $locale)}</span
									>{/if}
								{#if user.is_owner}<span class="owner-badge"
										>{t('owner', $locale)}</span
									>{/if}
							</span>
							{#if user.status}
								<span class="user-status"
									>{user.status.emoji} {user.status.message}</span
								>
							{/if}
						</div>
					</div>
					<div class="user-pomodoro">
						{#if user.pomodoro?.phase && user.pomodoro.phase !== 'idle'}
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
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.users-card h2 {
		font-size: var(--text-base);
		font-weight: 600;
		margin-bottom: 1rem;
		color: var(--color-fg-1);
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
		color: var(--color-warning, #f59e0b);
	}
	.user-pomodoro .rest-status {
		font-size: var(--text-sm);
		color: var(--color-success);
	}
	.user-pomodoro .idle-status {
		font-size: var(--text-sm);
		color: var(--color-fg-muted);
	}
	@media (max-width: 640px) {
		.user-item {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.5rem;
		}
	}
</style>
