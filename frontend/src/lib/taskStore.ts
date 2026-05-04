// Local-first task store with optional server sync
import { api, type Task } from './api';

const STORAGE_KEY = 'tomatogether_tasks';
const TOMBSTONE_KEY = 'tomatogether_tombstones';

export interface LocalTask {
	client_id: string;
	title: string;
	status: 'TODO' | 'WIP' | 'DONE';
	tag_id?: string;
	created_at: string;
	updated_at: string;
	server_id?: string;
}

function load(): LocalTask[] {
	try {
		return JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]');
	} catch {
		return [];
	}
}

function save(tasks: LocalTask[]) {
	localStorage.setItem(STORAGE_KEY, JSON.stringify(tasks));
}

function getTombstones(): string[] {
	try {
		return JSON.parse(localStorage.getItem(TOMBSTONE_KEY) || '[]');
	} catch {
		return [];
	}
}

function addTombstone(server_id: string) {
	const list = getTombstones();
	if (!list.includes(server_id)) list.push(server_id);
	localStorage.setItem(TOMBSTONE_KEY, JSON.stringify(list));
}

function clearTombstones() {
	localStorage.removeItem(TOMBSTONE_KEY);
}

export const taskStore = {
	getAll(): LocalTask[] {
		return load();
	},

	add(title: string, tag_id?: string): LocalTask {
		const tasks = load();
		const now = new Date().toISOString();
		const task: LocalTask = {
			client_id: crypto.randomUUID(),
			title,
			status: 'TODO',
			tag_id: tag_id || undefined,
			created_at: now,
			updated_at: now,
		};
		tasks.push(task);
		save(tasks);
		return task;
	},

	update(client_id: string, patch: Partial<Pick<LocalTask, 'title' | 'status' | 'tag_id'>>) {
		const tasks = load();
		const idx = tasks.findIndex((t) => t.client_id === client_id);
		if (idx === -1) return;
		Object.assign(tasks[idx], patch, { updated_at: new Date().toISOString() });
		save(tasks);
	},

	remove(client_id: string) {
		const tasks = load();
		const task = tasks.find((t) => t.client_id === client_id);
		if (task?.server_id) addTombstone(task.server_id);
		save(tasks.filter((t) => t.client_id !== client_id));
	},

	reorder(fromIdx: number, toIdx: number) {
		const tasks = load();
		if (fromIdx < 0 || fromIdx >= tasks.length || toIdx < 0 || toIdx >= tasks.length) return;
		const [item] = tasks.splice(fromIdx, 1);
		tasks.splice(toIdx, 0, item);
		save(tasks);
	},

	async syncWithServer(roomName: string): Promise<LocalTask[]> {
		const local = load();

		// 1. Push unsynced local tasks
		const unsynced = local.filter((t) => !t.server_id);
		if (unsynced.length > 0) {
			try {
				const resp = await api.syncTasks(
					roomName,
					unsynced.map((t) => ({
						client_id: t.client_id,
						title: t.title,
						status: t.status,
						tag_id: t.tag_id || '',
						created_at: t.created_at,
					}))
				);
				for (const r of resp.data.tasks) {
					const lt = local.find((t) => t.client_id === r.client_id);
					if (lt) lt.server_id = r.server_id;
				}
			} catch {
				/* offline */
			}
		}

		// 2. Delete server tasks removed locally
		const tombstones = getTombstones();
		for (const sid of tombstones) {
			try {
				await api.deleteTask(sid);
			} catch {
				/* ignore */
			}
		}
		clearTombstones();

		// 3. Pull server tasks and merge
		try {
			const resp = await api.getTasks();
			const serverTasks: Task[] = resp.data?.tasks || [];
			for (const st of serverTasks) {
				const existing = local.find(
					(t) => t.client_id === st.client_id || t.server_id === st.id
				);
				if (!existing) {
					local.push({
						client_id: st.client_id || crypto.randomUUID(),
						server_id: st.id,
						title: st.title,
						status: st.status as LocalTask['status'],
						tag_id: st.tag_id || undefined,
						created_at: st.created_at,
						updated_at: st.updated_at,
					});
				} else {
					if (!existing.server_id) existing.server_id = st.id;
					existing.status = st.status as LocalTask['status'];
					existing.tag_id = st.tag_id || undefined;
				}
			}
		} catch {
			/* offline */
		}

		save(local);
		return local;
	},
};
