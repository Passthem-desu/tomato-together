<script lang="ts">
	import { onMount } from 'svelte';
	import { locale, t } from '$lib/i18n';
	import { api, type Announcement } from '$lib/api';

	let {
		isOwner = false,
		newAnnouncement = null as { title: string; body: string } | null,
	}: {
		isOwner: boolean;
		newAnnouncement: { title: string; body: string } | null;
	} = $props();

	let announcements = $state<Announcement[]>([]);
	let loading = $state(true);
	let showForm = $state(false);
	let title = $state('');
	let body = $state('');
	let error = $state('');
	let sending = $state(false);

	function getRoomName() {
		return localStorage.getItem('room_name') || '';
	}

	async function loadAnnouncements() {
		loading = true;
		try {
			const resp = await api.getAnnouncements(getRoomName());
			announcements = resp.data?.announcements || [];
		} catch {
			/* offline */
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (newAnnouncement) {
			announcements = [
				{
					id: crypto.randomUUID(),
					title: newAnnouncement.title,
					body: newAnnouncement.body,
					created_at: new Date().toISOString(),
				},
				...announcements,
			];
		}
	});

	async function handleSend() {
		const t = title.trim();
		const b = body.trim();
		if (!t || !b) return;
		sending = true;
		error = '';
		try {
			await api.createAnnouncement(getRoomName(), { title: t, body: b });
			title = '';
			body = '';
			showForm = false;
			await loadAnnouncements();
		} catch (e: any) {
			error = e.message;
		} finally {
			sending = false;
		}
	}

	async function handleDelete(id: string) {
		if (!confirm(t('delete', $locale) + '?')) return;
		try {
			await api.deleteAnnouncement(getRoomName(), id);
			announcements = announcements.filter((a) => a.id !== id);
		} catch (e: any) {
			error = e.message;
		}
	}

	function formatTime(iso: string) {
		return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	onMount(() => {
		loadAnnouncements();
	});
</script>

<div class="announcement-panel">
	<div class="ann-header">
		<h3>{t('announcements', $locale)}</h3>
		{#if isOwner}
			<button class="btn-send-ann" onclick={() => (showForm = !showForm)}>
				{showForm ? '×' : '+ ' + t('send_announcement', $locale)}
			</button>
		{/if}
	</div>

	{#if showForm}
		<div class="ann-form">
			<input
				type="text"
				bind:value={title}
				placeholder={t('announcement_title_placeholder', $locale)}
				maxlength={100}
			/>
			<textarea
				bind:value={body}
				placeholder={t('announcement_body_placeholder', $locale)}
				maxlength={500}
				rows={3}
			></textarea>
			<div class="ann-form-actions">
				<button class="btn-submit" onclick={handleSend} disabled={sending || !title.trim() || !body.trim()}>
					{sending ? '...' : t('send', $locale)}
				</button>
				<button class="btn-cancel" onclick={() => { showForm = false; title = ''; body = ''; }}>
					{t('cancel', $locale)}
				</button>
			</div>
			{#if error}
				<p class="ann-error">{error}</p>
			{/if}
		</div>
	{/if}

	{#if loading}
		<p class="ann-empty">{t('loading', $locale)}</p>
	{:else if announcements.length === 0}
		<p class="ann-empty">{t('no_announcements', $locale)}</p>
	{:else}
		<ul class="ann-list">
			{#each announcements as a (a.id)}
				<li class="ann-item">
					<div class="ann-meta">
						<span class="ann-title">{a.title}</span>
						<div class="ann-meta-right">
							<span class="ann-time">{formatTime(a.created_at)}</span>
							{#if isOwner}
								<button class="btn-circle-del" onclick={() => handleDelete(a.id)} title={t('delete', $locale)}>×</button>
							{/if}
						</div>
					</div>
					<p class="ann-body">{a.body}</p>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.announcement-panel {
		background: var(--color-bg-1);
		border-radius: 0.75rem;
		padding: 1rem;
		border: 1px solid var(--color-border);
	}
	.ann-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.5rem;
	}
	.ann-header h3 {
		margin: 0;
		font-size: 0.95rem;
		font-weight: 700;
	}
	.btn-send-ann {
		padding: 0.2rem 0.6rem;
		font-size: 0.75rem;
		border-radius: 999px;
		border: 1.5px dashed var(--color-brand);
		background: none;
		color: var(--color-brand);
		cursor: pointer;
		transition: all 0.15s;
		font-weight: 500;
	}
	.btn-send-ann:hover {
		background: var(--color-brand-subtle);
	}

	.ann-form {
		margin-bottom: 0.75rem;
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
	}
	.ann-form input,
	.ann-form textarea {
		padding: 0.4rem 0.6rem;
		font-size: 0.85rem;
		border: 1px solid var(--color-border);
		border-radius: 0.5rem;
		background: var(--color-bg-0);
		color: var(--color-fg-0);
		outline: none;
		font-family: inherit;
		resize: vertical;
	}
	.ann-form input:focus,
	.ann-form textarea:focus {
		border-color: var(--color-brand);
	}
	.ann-form-actions {
		display: flex;
		gap: 0.4rem;
	}
	.btn-submit {
		padding: 0.35rem 0.9rem;
		border-radius: 0.5rem;
		border: none;
		background: var(--color-brand);
		color: #fff;
		font-size: 0.8rem;
		cursor: pointer;
		font-weight: 500;
	}
	.btn-submit:hover:not(:disabled) {
		background: var(--color-brand-hover);
	}
	.btn-submit:disabled {
		opacity: 0.4;
		cursor: default;
	}
	.btn-cancel {
		padding: 0.35rem 0.9rem;
		border-radius: 0.5rem;
		border: 1px solid var(--color-border);
		background: var(--color-bg-0);
		color: var(--color-fg-muted);
		font-size: 0.8rem;
		cursor: pointer;
	}
	.ann-error {
		color: var(--color-error);
		font-size: 0.75rem;
		margin: 0;
	}

	.ann-empty {
		text-align: center;
		color: var(--color-fg-muted);
		font-size: 0.8rem;
		padding: 1rem 0;
	}
	.ann-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.ann-item {
		padding: 0.5rem;
		border-radius: 0.5rem;
		background: var(--color-bg-0);
		border: 1px solid var(--color-border);
	}
	.ann-meta {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.2rem;
	}
	.ann-meta-right {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}
	.ann-title {
		font-weight: 600;
		font-size: 0.85rem;
	}
	.ann-time {
		font-size: 0.7rem;
		color: var(--color-fg-muted);
	}
	.btn-circle-del {
		width: 20px;
		height: 20px;
		padding: 0;
		border-radius: 50%;
		border: 1px solid var(--color-border);
		background: var(--color-bg-0);
		color: var(--color-fg-muted);
		font-size: 0.7rem;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.15s;
	}
	.btn-circle-del:hover {
		color: var(--color-error);
		border-color: var(--color-error);
		background: var(--color-error-subtle);
	}
	.ann-body {
		font-size: 0.8rem;
		color: var(--color-fg-muted);
		margin: 0;
		line-height: 1.4;
		white-space: pre-wrap;
		word-break: break-word;
	}
</style>
