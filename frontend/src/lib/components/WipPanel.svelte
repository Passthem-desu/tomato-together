<script lang="ts">
	import { onMount } from 'svelte';
	import { locale, t } from '$lib/i18n';
	import { api, type Task, type Tag } from '$lib/api';
	import TagSelector from './TagSelector.svelte';

	let tasks = $state<Task[]>([]);
	let tags = $state<Tag[]>([]);
	let loading = $state(true);
	let initial = $state(true);
	let error = $state('');
	let newTitle = $state('');
	let filterTagId = $state('');
	let filterStatus = $state('');

	let draggedIdx = $state<number | null>(null);

	function getRoomName() {
		return localStorage.getItem('room_name') || '';
	}

	async function loadData() {
		// Only show full loading spinner on first load; refreshes keep current data visible
		if (initial) loading = true;
		error = '';
		try {
			const [tagsResp, tasksResp] = await Promise.all([
				api.getTags(),
				api.getTasks(filterStatus || undefined),
			]);
			tags = tagsResp.data?.tags || [];
			tasks = tasksResp.data?.tasks || [];
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
			initial = false;
		}
	}

	let displayTasks = $derived(
		tasks.filter((t) => {
			if (filterTagId && t.tag_id !== filterTagId) return false;
			return true;
		})
	);

	async function handleAdd() {
		const title = newTitle.trim();
		if (!title) return;
		try {
			const clientId = crypto.randomUUID();
			await api.createTask({
				room_name: getRoomName(),
				client_id: clientId,
				title,
				tag_id: filterTagId || undefined,
			});
			newTitle = '';
			await loadData();
		} catch (e: any) {
			error = e.message;
		}
	}

	async function handleStatusCycle(task: Task) {
		const next: Record<string, string> = { TODO: 'WIP', WIP: 'DONE', DONE: 'TODO' };
		const newStatus = next[task.status] || 'TODO';
		try {
			await api.updateTask(task.id, { status: newStatus });
			await loadData();
		} catch (e: any) {
			error = e.message;
		}
	}

	async function handleTitleChange(taskId: string, title: string) {
		try {
			await api.updateTask(taskId, { title });
		} catch (e: any) {
			error = e.message;
		}
	}

	async function handleTagChange(taskId: string, tagId: string) {
		try {
			await api.updateTask(taskId, { tag_id: tagId || '' });
			await loadData();
		} catch (e: any) {
			error = e.message;
		}
	}

	async function handleDelete(taskId: string) {
		try {
			await api.deleteTask(taskId);
			await loadData();
		} catch (e: any) {
			error = e.message;
		}
	}

	// ── drag-and-drop reordering ──
	function handleDragStart(e: DragEvent, idx: number) {
		draggedIdx = idx;
		e.dataTransfer!.effectAllowed = 'move';
		(e.currentTarget as HTMLElement).closest('.wip-item')?.classList.add('dragging');
	}

	function handleDragEnd(e: DragEvent) {
		(e.currentTarget as HTMLElement).closest('.wip-item')?.classList.remove('dragging');
		draggedIdx = null;
		document.querySelectorAll('.wip-item').forEach((el) => el.classList.remove('drop-above', 'drop-below'));
	}

	function handleDragOver(e: DragEvent) {
		e.preventDefault();
		e.dataTransfer!.dropEffect = 'move';
		const el = e.currentTarget as HTMLElement;
		const rect = el.getBoundingClientRect();
		const mid = rect.top + rect.height / 2;
		el.classList.remove('drop-above', 'drop-below');
		el.classList.add(e.clientY < mid ? 'drop-above' : 'drop-below');
	}

	function handleDragLeave(e: DragEvent) {
		(e.currentTarget as HTMLElement).classList.remove('drop-above', 'drop-below');
	}

	async function handleDrop(e: DragEvent, targetIdx: number) {
		e.preventDefault();
		const el = e.currentTarget as HTMLElement;
		const rect = el.getBoundingClientRect();
		const mid = rect.top + rect.height / 2;
		let insertIdx = e.clientY < mid ? targetIdx : targetIdx + 1;
		el.classList.remove('drop-above', 'drop-below');
		if (draggedIdx === null) return;
		if (draggedIdx < insertIdx) insertIdx--;
		if (insertIdx === draggedIdx) return;

		const src = displayTasks[draggedIdx];
		const reordered = [...displayTasks];
		reordered.splice(draggedIdx, 1);
		reordered.splice(insertIdx, 0, src);

		const updates: Promise<any>[] = [];
		for (let i = 0; i < reordered.length; i++) {
			if (reordered[i].id !== displayTasks[i].id) {
				updates.push(api.updateTask(reordered[i].id, { title: displayTasks[i].title }));
			}
		}
		try {
			await Promise.all(updates);
			await loadData();
		} catch (e: any) {
			error = e.message;
		}
		draggedIdx = null;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') handleAdd();
	}

	function statusIcon(status: string) {
		if (status === 'TODO') return '○';
		if (status === 'WIP') return '◉';
		return '●';
	}

	function statusLabel(status: string) {
		if (status === 'TODO') return t('todo', $locale);
		if (status === 'WIP') return t('wip_status', $locale);
		return t('done', $locale);
	}

	function statusClass(status: string) {
		return 'st-' + status.toLowerCase();
	}

	onMount(() => {
		loadData();
	});
</script>

<div class="wip-panel">
	<div class="wip-header">
		<h3>{t('wip', $locale)}</h3>
		<div class="status-filter">
			<button class="filter-btn {!filterStatus ? 'active' : ''}" onclick={() => { filterStatus = ''; loadData(); }}>
				{t('all', $locale)}
			</button>
			<button class="filter-btn {filterStatus === 'TODO' ? 'active' : ''}" onclick={() => { filterStatus = 'TODO'; loadData(); }}>
				{t('todo', $locale)}
			</button>
			<button class="filter-btn {filterStatus === 'WIP' ? 'active' : ''}" onclick={() => { filterStatus = 'WIP'; loadData(); }}>
				{t('wip_status', $locale)}
			</button>
			<button class="filter-btn {filterStatus === 'DONE' ? 'active' : ''}" onclick={() => { filterStatus = 'DONE'; loadData(); }}>
				{t('done', $locale)}
			</button>
		</div>
	</div>

	<TagSelector {tags} selectedTagId={filterTagId} onselect={(id) => (filterTagId = id)} ontagschange={loadData} />

	<div class="wip-add">
		<input
			type="text"
			bind:value={newTitle}
			placeholder={t('new_wip_placeholder', $locale)}
			onkeydown={handleKeydown}
			maxlength={200}
		/>
		<button class="btn-circle btn-add-wip" onclick={handleAdd} disabled={!newTitle.trim()}>+</button>
	</div>

	{#if error}
		<p class="wip-error">{error}</p>
	{/if}

	{#if loading}
		<p class="wip-empty">{t('loading', $locale)}</p>
	{:else if displayTasks.length === 0}
		<p class="wip-empty">{t('no_wip', $locale)}</p>
	{:else}
		<ul class="wip-list">
			{#each displayTasks as task, idx (task.id)}
				<li
					class="wip-item {task.status === 'DONE' ? 'done' : ''}"
					draggable="true"
					ondragstart={(e) => handleDragStart(e, idx)}
					ondragend={handleDragEnd}
					ondragover={handleDragOver}
					ondragleave={handleDragLeave}
					ondrop={(e) => handleDrop(e, idx)}
				>
					<span class="drag-handle" title={t('drag_to_reorder', $locale)}>⋮⋮</span>
					<button
						class="status-btn {statusClass(task.status)}"
						onclick={() => handleStatusCycle(task)}
					>
						<span class="status-icon">{statusIcon(task.status)}</span>
						<span class="status-label">{statusLabel(task.status)}</span>
					</button>

					<input
						class="task-input"
						type="text"
						value={task.title}
						onchange={(e) => handleTitleChange(task.id, (e.target as HTMLInputElement).value)}
						maxlength={200}
					/>

					<select
						class="task-tag"
						value={task.tag_id || ''}
						onchange={(e) => handleTagChange(task.id, (e.target as HTMLSelectElement).value)}
					>
						<option value="">{t('no_tag', $locale)}</option>
						{#each tags as tag (tag.id)}
							<option value={tag.id}>{tag.name}</option>
						{/each}
					</select>

					<div class="task-actions">
						<button class="btn-del-text" onclick={() => handleDelete(task.id)} title={t('delete', $locale)}>×</button>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.wip-panel {
		background: var(--color-bg-1);
		border-radius: 0.75rem;
		padding: 1rem;
		border: 1px solid var(--color-border);
	}
	.wip-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.75rem;
		flex-wrap: wrap;
		gap: 0.4rem;
	}
	.wip-header h3 {
		margin: 0;
		font-size: 0.95rem;
		font-weight: 700;
	}

	/* ── filter buttons ── */
	.status-filter {
		display: flex;
		gap: 0.25rem;
	}
	.filter-btn {
		padding: 0.2rem 0.55rem;
		font-size: 0.7rem;
		border-radius: 999px;
		border: 1px solid var(--color-border);
		background: var(--color-bg-0);
		color: var(--color-fg-muted);
		cursor: pointer;
		transition: all 0.15s;
		font-weight: 500;
	}
	.filter-btn:hover {
		border-color: var(--color-brand);
		color: var(--color-brand);
	}
	.filter-btn.active {
		background: var(--color-brand);
		border-color: var(--color-brand);
		color: #fff;
	}

	/* ── add bar ── */
	.wip-add {
		display: flex;
		gap: 0.4rem;
		margin-bottom: 0.5rem;
	}
	.wip-add input {
		flex: 1;
		padding: 0.4rem 0.7rem;
		font-size: 0.85rem;
		border: 1px solid var(--color-border);
		border-radius: 0.5rem;
		background: var(--color-bg-0);
		color: var(--color-fg-0);
		outline: none;
	}
	.wip-add input:focus {
		border-color: var(--color-brand);
	}

	/* ── circle buttons ── */
	.btn-circle {
		width: 28px;
		height: 28px;
		padding: 0;
		border-radius: 50%;
		border: 1px solid var(--color-border);
		background: var(--color-bg-0);
		font-size: 0.85rem;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		transition: all 0.15s;
		color: var(--color-fg-muted);
	}
	.btn-circle:hover:not(:disabled) {
		border-color: var(--color-brand);
		color: var(--color-brand);
		background: var(--color-brand-subtle);
	}
	.btn-circle:disabled {
		opacity: 0.25;
		cursor: default;
	}
	.btn-add-wip {
		color: var(--color-brand);
		border-color: var(--color-brand);
		font-size: 1rem;
	}
	.btn-add-wip:disabled {
		color: var(--color-fg-muted);
		border-color: var(--color-border);
		background: var(--color-bg-0);
	}

	/* ── messages ── */
	.wip-error {
		color: var(--color-error);
		font-size: 0.75rem;
		margin: 0.25rem 0;
	}
	.wip-empty {
		text-align: center;
		color: var(--color-fg-muted);
		font-size: 0.8rem;
		padding: 1.5rem 0;
	}

	/* ── drag handle ── */
	.drag-handle {
		cursor: grab;
		color: var(--color-fg-muted);
		font-size: 0.7rem;
		letter-spacing: -0.15em;
		padding: 0 0.1rem;
		flex-shrink: 0;
		user-select: none;
		line-height: 1;
	}
	.drag-handle:active {
		cursor: grabbing;
	}
	.wip-item.dragging {
		opacity: 0.35;
	}
	.wip-item.drop-above::before,
	.wip-item.drop-below::after {
		content: '';
		position: absolute;
		left: 0.25rem;
		right: 0.25rem;
		height: 2px;
		background: var(--color-brand);
		border-radius: 1px;
		pointer-events: none;
	}
	.wip-item.drop-above::before { top: -1px; }
	.wip-item.drop-below::after { bottom: -1px; }

	/* ── list ── */
	.wip-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
	}
	.wip-item {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		padding: 0.35rem 0.25rem;
		border-radius: 0.4rem;
		transition: background 0.15s;
		position: relative;
	}
	.wip-item:hover {
		background: var(--color-bg-0);
	}
	.wip-item.done {
		opacity: 0.55;
	}
	.wip-item.done .task-input {
		text-decoration: line-through;
	}

	/* ── status cycle button ── */
	.status-btn {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.15rem 0.5rem;
		border-radius: 999px;
		border: 1.5px solid var(--color-border);
		background: var(--color-bg-0);
		cursor: pointer;
		flex-shrink: 0;
		font-size: 0.75rem;
		transition: all 0.15s;
		color: var(--color-fg-muted);
	}
	.status-btn:hover {
		border-color: var(--color-brand);
	}
	.status-btn.st-todo {
		color: var(--color-fg-muted);
	}
	.status-btn.st-wip {
		color: var(--color-brand);
		border-color: var(--color-brand);
	}
	.status-btn.st-done {
		color: var(--color-success);
		border-color: var(--color-success);
	}
	.status-icon {
		font-size: 0.7rem;
	}
	.status-label {
		font-weight: 500;
	}

	/* ── inline title input ── */
	.task-input {
		flex: 1;
		min-width: 0;
		padding: 0.25rem 0.4rem;
		font-size: 0.85rem;
		border: 1px solid transparent;
		border-radius: 0.3rem;
		background: transparent;
		color: var(--color-fg-0);
		outline: none;
		transition: border-color 0.15s;
	}
	.task-input:hover {
		border-color: var(--color-border);
	}
	.task-input:focus {
		border-color: var(--color-brand);
		background: var(--color-bg-1);
	}

	/* ── tag selector ── */
	.task-tag {
		font-size: 0.65rem;
		padding: 0.2rem 0.5rem;
		border-radius: 999px;
		border: 1px solid var(--color-border);
		background: var(--color-bg-0);
		color: var(--color-fg-muted);
		cursor: pointer;
		outline: none;
		max-width: 80px;
		flex-shrink: 0;
	}

	.btn-del-text {
		width: 24px;
		height: 24px;
		padding: 0;
		border: none;
		background: none;
		color: var(--color-fg-muted);
		font-size: 1rem;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		border-radius: 0.3rem;
		transition: all 0.15s;
	}
	.btn-del-text:hover {
		color: var(--color-error);
		background: var(--color-error-subtle);
	}

	/* ── action buttons ── */
	.task-actions {
		display: flex;
		gap: 0.15rem;
		flex-shrink: 0;
	}
</style>
