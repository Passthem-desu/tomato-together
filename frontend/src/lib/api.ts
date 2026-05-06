// API configuration - use /api prefix for Vite proxy
const API_BASE = '/api';

export interface Room {
	id: string;
	name: string;
	is_readonly: boolean;
	has_password: boolean;
	member_count?: number;
	created_at?: string;
	owner?: {
		username: string;
	};
}

export interface Member {
	id: string;
	username: string;
	is_owner: boolean;
	is_persistent: boolean;
	joined_at?: string;
}

export interface RoomResponse {
	room: Room;
	member: Member;
	token: string;
	access_token?: string;
	refresh_token?: string;
	expires_in?: number;
}

export interface UserInfo {
	id: string;
	username: string;
	is_owner: boolean;
	is_persistent: boolean;
	status?: {
		emoji: string;
		message: string;
	};
	pomodoro?: {
		phase: string;
		is_following: boolean;
		leader_username?: string;
		started_at?: string;
		remaining_seconds?: number;
	};
	is_online: boolean;
	total_pomodoros?: number;
	total_duration?: number;
}

export interface Tag {
	id: string;
	member_id: string;
	room_id: string;
	name: string;
	created_at: string;
}

export interface Task {
	id: string;
	client_id?: string;
	member_id: string;
	room_id: string;
	tag_id?: string;
	title: string;
	status: 'TODO' | 'WIP' | 'DONE';
	created_at: string;
	updated_at: string;
	completed_at?: string;
	sort_order?: number;
}

export interface Announcement {
	id: string;
	title: string;
	body: string;
	sender?: {
		id: string;
		username: string;
	};
	created_at: string;
}

export interface PomodoroStatus {
	phase: 'idle' | 'focusing' | 'paused' | 'following' | 'rest';
	session_id?: string;
	started_at?: string;
	paused_at?: string;
	remaining_seconds?: number;
	leader_id?: string;
	leader_username?: string;
	planned_duration?: number;
	rest_duration?: number;
	long_break_duration?: number;
	is_long_break?: boolean;
	sessions_before_long_break?: number;
	duration?: number;
	sessions_completed?: number;
}

// API helper with JWT auto-refresh

let refreshPromise: Promise<boolean> | null = null;

function getAccessToken(): string | null {
	return localStorage.getItem('access_token') || localStorage.getItem('token');
}

async function tryRefresh(): Promise<boolean> {
	const refreshToken = localStorage.getItem('refresh_token_' + getDeviceId());
	if (!refreshToken) return false;

	if (refreshPromise) return refreshPromise;

	refreshPromise = (async () => {
		try {
			const response = await fetch(`${API_BASE}/auth/refresh`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ refresh_token: refreshToken }),
			});
			if (!response.ok) return false;

			const data = await response.json();
			if (data.success) {
				localStorage.setItem('access_token', data.data.access_token);
				localStorage.setItem('refresh_token_' + getDeviceId(), data.data.refresh_token);
				return true;
			}
			return false;
		} catch {
			return false;
		} finally {
			refreshPromise = null;
		}
	})();

	return refreshPromise;
}

async function apiRequest<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
	const token = typeof window !== 'undefined' ? getAccessToken() : null;

	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...((options.headers as Record<string, string>) || {}),
	};

	if (token) {
		headers['Authorization'] = `Bearer ${token}`;
	}

	const response = await fetch(`${API_BASE}${endpoint}`, {
		...options,
		headers,
	});

	const data = await response.json();

	if (!response.ok) {
		if (response.status === 401 && token) {
			const refreshed = await tryRefresh();
			if (refreshed) {
				return apiRequest(endpoint, options);
			}
			clearAuth();
			if (typeof window !== 'undefined') window.location.href = '/';
		}
		throw new Error(data.error || 'Request failed');
	}

	return data;
}

// Room API
export const api = {
	// Create room (requires password for persistent user)
	createRoom: async (data: {
		room_name: string;
		room_password?: string;
		username: string;
		password: string;
		is_readonly?: boolean;
	}): Promise<{ success: boolean; data: RoomResponse }> => {
		return apiRequest('/rooms', {
			method: 'POST',
			body: JSON.stringify(data),
		});
	},

	// Join room
	joinRoom: async (
		roomName: string,
		data: {
			username: string;
			password?: string;
			room_password?: string;
		}
	): Promise<{ success: boolean; data: RoomResponse }> => {
		return apiRequest(`/rooms/${encodeURIComponent(roomName)}/join`, {
			method: 'POST',
			body: JSON.stringify(data),
		});
	},

	// Get room info
	getRoomInfo: async (roomName: string): Promise<{ success: boolean; data: Room }> => {
		return apiRequest(`/rooms/${encodeURIComponent(roomName)}`);
	},

	// Check if username exists in room
	checkUser: async (
		roomName: string,
		username: string
	): Promise<{ success: boolean; data: { exists: boolean; is_persistent: boolean } }> => {
		return apiRequest(`/rooms/${encodeURIComponent(roomName)}/check-user`, {
			method: 'POST',
			body: JSON.stringify({ username }),
		});
	},

	// Check room password validity
	checkRoomPassword: async (
		roomName: string,
		roomPassword: string
	): Promise<{ success: boolean; data: { valid: boolean } }> => {
		return apiRequest(`/rooms/${encodeURIComponent(roomName)}/check-password`, {
			method: 'POST',
			body: JSON.stringify({ room_password: roomPassword }),
		});
	},

	// Leave room
	leaveRoom: async (): Promise<{ success: boolean }> => {
		const roomName = localStorage.getItem('room_name');
		return apiRequest(`/rooms/${encodeURIComponent(roomName!)}/leave`, {
			method: 'POST',
		});
	},

	// Get room users
	getRoomUsers: async (): Promise<{ success: boolean; data: { users: UserInfo[] } }> => {
		const roomName = localStorage.getItem('room_name');
		return apiRequest(`/rooms/${encodeURIComponent(roomName!)}/users`);
	},

	// Update room settings (owner only)
	updateRoomSettings: async (data: {
		room_password?: string;
		is_readonly?: boolean;
	}): Promise<{ success: boolean }> => {
		const roomName = localStorage.getItem('room_name');
		return apiRequest(`/rooms/${encodeURIComponent(roomName!)}/settings`, {
			method: 'PUT',
			body: JSON.stringify(data),
		});
	},

	// Kick a member from the room (owner only)
	kickMember: async (memberId: string): Promise<{ success: boolean }> => {
		const roomName = localStorage.getItem('room_name');
		return apiRequest(`/rooms/${encodeURIComponent(roomName!)}/members/${memberId}`, {
			method: 'DELETE',
		});
	},

	// Transfer room ownership (owner only)
	setOwner: async (memberId: string): Promise<{ success: boolean }> => {
		const roomName = localStorage.getItem('room_name');
		return apiRequest(`/rooms/${encodeURIComponent(roomName!)}/owners/${memberId}`, {
			method: 'PUT',
			body: JSON.stringify({ is_owner: true }),
		});
	},

	// Auth
	getMe: async (): Promise<{ success: boolean; data: Member }> => {
		return apiRequest('/auth/me');
	},

	changePassword: async (data: {
		old_password: string;
		new_password: string;
	}): Promise<{ success: boolean }> => {
		return apiRequest('/auth/password', {
			method: 'PUT',
			body: JSON.stringify(data),
		});
	},

	login: async (data: {
		room_name: string;
		username: string;
		password: string;
		room_password?: string;
	}): Promise<{ success: boolean; data: RoomResponse }> => {
		return apiRequest('/auth/login', {
			method: 'POST',
			body: JSON.stringify(data),
		});
	},

	// Pomodoro
	startPomodoro: async (data: {
		room_name: string;
		tag_id?: string;
		task_id?: string;
		planned_duration?: number;
		rest_duration?: number;
		long_break_duration?: number;
		sessions_before_long_break?: number;
		session_index?: number;
	}): Promise<{ success: boolean; data: PomodoroStatus }> => {
		return apiRequest('/pomodoro/start', {
			method: 'POST',
			body: JSON.stringify(data),
		});
	},

	followPomodoro: async (data: {
		room_name: string;
		leader_id: string;
	}): Promise<{ success: boolean; data: PomodoroStatus }> => {
		return apiRequest('/pomodoro/follow', {
			method: 'POST',
			body: JSON.stringify(data),
		});
	},

	unfollowPomodoro: async (roomName: string): Promise<{ success: boolean }> => {
		return apiRequest('/pomodoro/unfollow', {
			method: 'POST',
			body: JSON.stringify({ room_name: roomName }),
		});
	},

	endPomodoro: async (
		roomName: string,
		aborted = false,
		sessionIndex = 0
	): Promise<{ success: boolean; data: PomodoroStatus }> => {
		return apiRequest('/pomodoro/end', {
			method: 'POST',
			body: JSON.stringify({ room_name: roomName, aborted, session_index: sessionIndex }),
		});
	},

	getPomodoroStatus: async (): Promise<{ success: boolean; data: PomodoroStatus }> => {
		return apiRequest('/pomodoro/status');
	},

	// Status
	updateStatus: async (data: {
		room_name: string;
		emoji: string;
		message: string;
	}): Promise<{ success: boolean }> => {
		return apiRequest('/status', {
			method: 'PUT',
			body: JSON.stringify(data),
		});
	},

	deleteStatus: async (roomName: string): Promise<{ success: boolean }> => {
		return apiRequest('/status', {
			method: 'DELETE',
			body: JSON.stringify({ room_name: roomName }),
		});
	},

	// Tags
	getTags: async (): Promise<{ success: boolean; data: { tags: Tag[] } }> => {
		return apiRequest('/tags');
	},

	createTag: async (data: {
		room_name: string;
		name: string;
	}): Promise<{ success: boolean; data: Tag }> => {
		return apiRequest('/tags', { method: 'POST', body: JSON.stringify(data) });
	},

	updateTag: async (id: string, data: { name: string }): Promise<{ success: boolean }> => {
		return apiRequest(`/tags/${id}`, { method: 'PUT', body: JSON.stringify(data) });
	},

	deleteTag: async (id: string): Promise<{ success: boolean }> => {
		return apiRequest(`/tags/${id}`, { method: 'DELETE' });
	},

	// Tasks (WIP)
	getTasks: async (status?: string): Promise<{ success: boolean; data: { tasks: Task[] } }> => {
		const qs = status ? `?status=${status}` : '';
		return apiRequest(`/tasks${qs}`);
	},

	createTask: async (data: {
		room_name: string;
		client_id: string;
		title: string;
		tag_id?: string;
	}): Promise<{ success: boolean; data: Task }> => {
		return apiRequest('/tasks', { method: 'POST', body: JSON.stringify(data) });
	},

	updateTask: async (
		id: string,
		data: {
			title?: string;
			status?: string;
			tag_id?: string;
		}
	): Promise<{ success: boolean }> => {
		return apiRequest(`/tasks/${id}`, { method: 'PUT', body: JSON.stringify(data) });
	},

	deleteTask: async (id: string): Promise<{ success: boolean }> => {
		return apiRequest(`/tasks/${id}`, { method: 'DELETE' });
	},

	deleteTasksBatch: async (
		taskIds: string[]
	): Promise<{ success: boolean; data?: { deleted: number } }> => {
		return apiRequest('/tasks/batch', {
			method: 'DELETE',
			body: JSON.stringify({ task_ids: taskIds }),
		});
	},

	syncTasks: async (
		roomName: string,
		tasks: Array<{
			client_id: string;
			title: string;
			status: string;
			tag_id?: string;
			created_at: string;
			updated_at: string;
			sort_order?: number;
		}>
	): Promise<{
		success: boolean;
		data: {
			synced: number;
			tasks: Array<{ client_id: string; server_id: string }>;
			conflicts?: Array<{ client_id: string; server_id: string; server_task: Task }>;
		};
	}> => {
		return apiRequest('/tasks/sync', {
			method: 'POST',
			body: JSON.stringify({ room_name: roomName, tasks }),
		});
	},

	// Announcements
	getAnnouncements: async (
		roomName: string
	): Promise<{ success: boolean; data: { announcements: Announcement[] } }> => {
		return apiRequest(`/rooms/${encodeURIComponent(roomName)}/announcements`);
	},

	createAnnouncement: async (
		roomName: string,
		data: { title: string; body: string }
	): Promise<{ success: boolean }> => {
		return apiRequest(`/rooms/${encodeURIComponent(roomName)}/announcements`, {
			method: 'POST',
			body: JSON.stringify(data),
		});
	},

	deleteAnnouncement: async (roomName: string, id: string): Promise<{ success: boolean }> => {
		return apiRequest(`/rooms/${encodeURIComponent(roomName)}/announcements/${id}`, {
			method: 'DELETE',
		});
	},

	getMyStats: async (): Promise<{
		success: boolean;
		data: { total_pomodoros: number; total_duration: number };
	}> => {
		return apiRequest('/stats/me');
	},

	resetMyStats: async (): Promise<{ success: boolean }> => {
		return apiRequest('/stats/me', { method: 'DELETE' });
	},

	// Pause pomodoro
	pausePomodoro: async (): Promise<{ success: boolean; data: PomodoroStatus }> => {
		return apiRequest('/pomodoro/pause', { method: 'POST' });
	},

	// Resume pomodoro
	resumePomodoro: async (): Promise<{ success: boolean; data: PomodoroStatus }> => {
		return apiRequest('/pomodoro/resume', { method: 'POST' });
	},

	// Skip rest
	skipRest: async (): Promise<{ success: boolean }> => {
		return apiRequest('/pomodoro/skip-rest', { method: 'POST' });
	},

	// JWT token management
	refreshToken: async (
		refreshToken: string
	): Promise<{
		success: boolean;
		data: { access_token: string; refresh_token: string; expires_in: number };
	}> => {
		return apiRequest('/auth/refresh', {
			method: 'POST',
			body: JSON.stringify({ refresh_token: refreshToken }),
		});
	},

	logout: async (refreshToken?: string): Promise<{ success: boolean }> => {
		return apiRequest('/auth/logout', {
			method: 'POST',
			body: JSON.stringify(refreshToken ? { refresh_token: refreshToken } : {}),
		});
	},

	logoutAll: async (): Promise<{ success: boolean; data: { revoked_count: number } }> => {
		return apiRequest('/auth/tokens', { method: 'DELETE' });
	},
};

// Device ID for per-device refresh token storage
function getDeviceId(): string {
	if (typeof window === 'undefined') return '';
	const key = 'device_id';
	let deviceId = localStorage.getItem(key);
	if (!deviceId) {
		deviceId = crypto.randomUUID();
		localStorage.setItem(key, deviceId);
	}
	return deviceId;
}

// Store helpers
export function saveAuth(token: string, roomName: string, member: Member, room?: Room) {
	localStorage.setItem('token', token);
	localStorage.setItem('room_name', roomName);
	localStorage.setItem('member', JSON.stringify(member));
	if (room) {
		localStorage.setItem('room', JSON.stringify(room));
	}
}

export function saveJWT(accessToken: string, refreshToken: string) {
	localStorage.setItem('access_token', accessToken);
	localStorage.setItem('refresh_token_' + getDeviceId(), refreshToken);
}

export function clearAuth() {
	['token', 'access_token', 'room_name', 'member', 'room'].forEach((k) =>
		localStorage.removeItem(k)
	);
	for (let i = localStorage.length - 1; i >= 0; i--) {
		const key = localStorage.key(i);
		if (key?.startsWith('refresh_token_')) {
			localStorage.removeItem(key);
		}
	}
}

export function getRoom(): Room | null {
	const roomStr = localStorage.getItem('room');
	if (!roomStr) return null;
	try {
		return JSON.parse(roomStr);
	} catch {
		return null;
	}
}

export function getMember(): Member | null {
	const memberStr = localStorage.getItem('member');
	if (!memberStr) return null;
	try {
		return JSON.parse(memberStr);
	} catch {
		return null;
	}
}

export function isAuthenticated(): boolean {
	return !!localStorage.getItem('token');
}
