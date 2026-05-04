<script lang="ts">
	import { locale, t } from '$lib/i18n';
	import LocaleSwitcher from './LocaleSwitcher.svelte';

	interface Props {
		roomName: string;
		onlineCount: number;
		sseConnected: boolean;
		username: string;
		isOwner: boolean;
		onlogout: () => void;
	}

	let { roomName, onlineCount, sseConnected, username, isOwner, onlogout }: Props = $props();
</script>

<header class="room-header">
	<div class="header-left">
		<div class="room-avatar"></div>
		<div class="room-info">
			<h1>{roomName || t('room_title', $locale)}</h1>
			<div class="room-meta">
				<span
					class="status-dot"
					class:online={onlineCount > 0}
					class:disconnected={!sseConnected}
				></span>
				<span class="member-count">{onlineCount} {t('online', $locale)}</span>
			</div>
		</div>
	</div>
	<div class="header-right">
		<LocaleSwitcher />
		<div class="user-badge">
			<span class="user-name">{username}</span>
			{#if isOwner}<span class="badge">{t('owner', $locale)}</span>{/if}
		</div>
		<button class="btn-ghost btn-sm" onclick={onlogout}>{t('logout', $locale)}</button>
	</div>
</header>

<style>
	.room-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem 0;
		margin-bottom: 2rem;
		border-bottom: 1px solid var(--color-border);
	}
	.header-left {
		display: flex;
		align-items: center;
		gap: 0.875rem;
		min-width: 0;
	}
	.btn-ghost {
		white-space: nowrap;
	}
	.room-avatar {
		font-size: 2.5rem;
	}
	.room-info h1 {
		font-size: var(--text-xl);
		font-weight: 600;
		margin-bottom: 0.25rem;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.room-meta {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: var(--color-fg-muted);
		font-size: var(--text-sm);
	}
	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: var(--radius-full);
		background: var(--color-fg-muted);
	}
	.status-dot.online {
		background: var(--color-success);
	}
	.status-dot.disconnected {
		background: var(--color-warning, #f59e0b);
	}
	.header-right {
		display: flex;
		align-items: center;
		gap: 1rem;
		flex-shrink: 0;
		white-space: nowrap;
	}
	.user-badge {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		white-space: nowrap;
	}
	.user-badge .user-name {
		font-weight: 500;
	}
	.badge {
		display: inline-flex;
		padding: 0.25rem 0.5rem;
		font-size: var(--text-xs);
		font-weight: 500;
		border-radius: var(--radius-full);
		background: var(--color-brand-subtle);
		color: var(--color-brand);
	}

	@media (max-width: 640px) {
		.room-header {
			flex-direction: column;
			gap: 1rem;
			text-align: center;
		}
		.header-left {
			flex-direction: column;
		}
	}
</style>
