<script lang="ts">
	import { locale, t } from '$lib/i18n';
	import { api, type Tag } from '$lib/api';

	let {
		tags = [] as Tag[],
		selectedTagId = '',
		onselect = (tagId: string) => {},
		ontagschange = () => {},
	}: {
		tags: Tag[];
		selectedTagId: string;
		onselect: (tagId: string) => void;
		ontagschange: () => void;
	} = $props();

	let showAdd = $state(false);
	let newTagName = $state('');
	let error = $state('');

	async function handleAdd() {
		const name = newTagName.trim();
		if (!name) return;
		error = '';
		try {
			await api.createTag({ room_name: localStorage.getItem('room_name') || '', name });
			newTagName = '';
			showAdd = false;
			ontagschange();
		} catch (e: any) {
			error = e.message;
		}
	}

	async function handleDelete(tagId: string) {
		if (!confirm(t('delete_tag_confirm', $locale))) return;
		try {
			await api.deleteTag(tagId);
			if (selectedTagId === tagId) onselect('');
			ontagschange();
		} catch {
			/* ignore */
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') handleAdd();
		if (e.key === 'Escape') {
			showAdd = false;
			newTagName = '';
			error = '';
		}
	}
</script>

<div class="tag-selector">
	<div class="tag-list">
		<button class="tag-chip {!selectedTagId ? 'active' : ''}" onclick={() => onselect('')}>
			{t('all', $locale)}
		</button>
		{#each tags as tag (tag.id)}
			<div class="tag-row">
				<button
					class="tag-chip {selectedTagId === tag.id ? 'active' : ''}"
					onclick={() => onselect(tag.id)}
				>
					{tag.name}
				</button>
				<button
					class="tag-delete"
					onclick={() => handleDelete(tag.id)}
					title={t('delete', $locale)}
				>
					×
				</button>
			</div>
		{/each}
		{#if showAdd}
			<div class="tag-add-form">
				<input
					type="text"
					bind:value={newTagName}
					placeholder={t('new_tag_placeholder', $locale)}
					onkeydown={handleKeydown}
					maxlength={30}
					autofocus
				/>
				<button class="btn-add" onclick={handleAdd} disabled={!newTagName.trim()}>+</button>
				<button
					class="btn-cancel"
					onclick={() => {
						showAdd = false;
						newTagName = '';
						error = '';
					}}>×</button
				>
			</div>
		{:else}
			<button class="tag-chip tag-add-btn" onclick={() => (showAdd = true)}>
				+ {t('add_tag', $locale)}
			</button>
		{/if}
	</div>
	{#if error}
		<span class="tag-error">{error}</span>
	{/if}
</div>

<style>
	.tag-selector {
		margin-bottom: 0.75rem;
	}
	.tag-list {
		display: flex;
		flex-wrap: wrap;
		gap: 0.35rem;
		align-items: center;
	}
	.tag-row {
		position: relative;
		display: inline-flex;
	}
	.tag-chip {
		position: relative;
		display: inline-flex;
		align-items: center;
		padding: 0.2rem 0.6rem;
		border-radius: 999px;
		font-size: 0.8rem;
		border: 1.5px solid var(--color-border);
		background: var(--color-bg-1);
		color: var(--color-text);
		cursor: pointer;
		transition: all 0.15s;
		white-space: nowrap;
	}
	.tag-chip:hover {
		border-color: var(--color-accent);
	}
	.tag-chip.active {
		background: var(--color-accent);
		color: #fff;
		border-color: var(--color-accent);
	}
	.tag-delete {
		position: absolute;
		top: -4px;
		right: -4px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		height: 16px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		font-size: 0.9rem;
		cursor: pointer;
		border-radius: 50%;
		opacity: 0;
		transition: opacity 0.15s;
		padding: 0;
		z-index: 1;
	}
	.tag-row:hover .tag-delete {
		opacity: 1;
	}
	.tag-delete:hover {
		color: var(--color-danger);
		background: var(--color-danger-bg);
	}
	.tag-add-btn {
		border-style: dashed;
		color: var(--color-text-muted);
	}
	.tag-add-form {
		display: flex;
		gap: 0.25rem;
		align-items: center;
	}
	.tag-add-form input {
		width: 100px;
		padding: 0.2rem 0.5rem;
		font-size: 0.8rem;
		border: 1.5px solid var(--color-accent);
		border-radius: 999px;
		background: var(--color-bg-0);
		color: var(--color-text);
		outline: none;
	}
	.btn-add,
	.btn-cancel {
		width: 22px;
		height: 22px;
		padding: 0;
		border-radius: 50%;
		border: 1.5px solid var(--color-border);
		background: var(--color-bg-1);
		font-size: 0.9rem;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--color-fg-0);
	}
	.btn-add:not(:disabled) {
		color: var(--color-brand);
		border-color: var(--color-brand);
	}
	.btn-add:disabled {
		color: var(--color-fg-muted);
		border-color: var(--color-border);
		cursor: default;
	}
	.btn-add:not(:disabled):hover {
		background: var(--color-brand-subtle);
	}
	.btn-cancel {
		color: var(--color-text-muted);
	}
	.btn-cancel:hover {
		background: var(--color-bg-2);
		color: var(--color-fg-0);
	}
	.tag-error {
		font-size: 0.7rem;
		color: var(--color-danger);
	}
</style>
