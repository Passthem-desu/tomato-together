import { writable, derived, get } from 'svelte/store';
import type { Member, Room, UserInfo, PomodoroStatus } from './api';
import { api, saveAuth, saveJWT, clearAuth } from './api';
import { SSEClient, type TickData } from './sse/client';
import { getErrorMessage, locale, t } from './i18n';

// App state
export const currentMember = writable<Member | null>(null);
export const currentRoom = writable<Room | null>(null);
export const roomUsers = writable<UserInfo[]>([]);
export const pomodoroStatus = writable<PomodoroStatus>({ phase: 'idle' });
export const isLoading = writable(false);
export const error = writable<string | null>(null);
export const sseConnected = writable(true);

// Last SSE announcement (emitted to trigger UI updates)
export const lastAnnouncement = writable<{ title: string; body: string } | null>(null);

// Derived stores
export const isAuthenticated = derived(currentMember, ($member) => !!$member);
export const isOwner = derived(currentMember, ($member) => $member?.is_owner ?? false);

// SSE client
let sseClient: SSEClient | null = null;
let sseUnsubscribers: (() => void)[] = [];

/**
 * Connect SSE for real-time room events.
 * Must be called after joinRoom/createRoom when token is available.
 */
export function connectSSE() {
	const token = localStorage.getItem('access_token') || localStorage.getItem('token') || '';
	const roomName = localStorage.getItem('room_name') || '';

	if (!roomName) return;

	// Disconnect existing SSE
	disconnectSSE();

	sseClient = new SSEClient(roomName, token, (connected) => {
		sseConnected.set(connected);
	});
	sseClient.connect();

	// Start predictive countdown for other users' tomato timers
	startPredictiveCountdown();

	// Handle tick events to keep users list updated
	const unsubTick = sseClient.on('tick', (data: TickData) => {
		handleTick(data);
	});

	// Handle user events
	const unsubJoined = sseClient.on('user_joined', (data: any) => {
		handleUserJoined(data);
	});

	const unsubLeft = sseClient.on('user_left', (data: any) => {
		handleUserLeft(data);
	});

	// Handle pomodoro events
	const unsubStarted = sseClient.on('pomodoro_started', () => {
		refreshRoomUsers();
	});

	const unsubEnded = sseClient.on('pomodoro_ended', () => {
		refreshRoomUsers();
	});

	const unsubFollowed = sseClient.on('pomodoro_followed', () => {
		refreshRoomUsers();
	});

	const unsubUnfollowed = sseClient.on('pomodoro_unfollowed', () => {
		refreshRoomUsers();
	});

	// Handle status updates
	const unsubStatus = sseClient.on('status_updated', () => {
		refreshRoomUsers();
	});

	// Handle phase changes (pause/resume/skip)
	const unsubPhase = sseClient.on('phase_changed', () => {
		refreshRoomUsers();
	});

	// Handle token expiry
	const unsubTokenExpired = sseClient.on('token_expired', () => {
		error.set(t('token_expired', get(locale)));
		logout();
	});

	// Handle kicked from room
	const unsubKicked = sseClient.on('kicked', (data: any) => {
		const currentId = get(currentMember)?.id;
		if (data.user_id === currentId) {
			error.set(t('kicked_from_room', get(locale)));
			logout();
		}
	});

	// Handle announcements
	const unsubAnnouncement = sseClient.on('announcement', (data: any) => {
		lastAnnouncement.set({ title: data.title || '', body: data.body || '' });
	});

	sseUnsubscribers = [
		unsubTick,
		unsubJoined,
		unsubLeft,
		unsubStarted,
		unsubEnded,
		unsubFollowed,
		unsubUnfollowed,
		unsubStatus,
		unsubPhase,
		unsubTokenExpired,
		unsubKicked,
		unsubAnnouncement,
	];
}

export function disconnectSSE() {
	stopPredictiveCountdown();
	for (const unsub of sseUnsubscribers) {
		unsub();
	}
	sseUnsubscribers = [];

	if (sseClient) {
		sseClient.disconnect();
		sseClient = null;
	}
}

// SSE event handlers

// Store the last tick data for interpolation
let lastTickData: TickData | null = null;

function handleTick(data: TickData) {
	lastTickData = data;
	const member = get(currentMember);

	// Sync current user's pomodoro status from server (corrects tab-throttling drift)
	for (const tickUser of data.users) {
		if (member && tickUser.id === member.id) {
			pomodoroStatus.update((prev) => ({
				...prev,
				remaining_seconds: tickUser.remaining_seconds,
			}));
		}
	}

	// Create a lookup map for O(1) access
	const tickMap = new Map(data.users.map((u) => [u.id, u]));

	roomUsers.update((users) => {
		let changed = false;
		const updated = users.map((user) => {
			const tickUser = tickMap.get(user.id);
			if (!tickUser) return user;

			changed = true;
			const existingPomodoro = user.pomodoro || {
				is_active: false,
				is_following: false,
			};
			return {
				...user,
				pomodoro: {
					...existingPomodoro,
					remaining_seconds: tickUser.remaining_seconds,
					phase: tickUser.phase || 'idle',
				},
			};
		});
		return changed ? updated : users;
	});
}

/**
 * Start predictive countdown: decrement users' remaining_seconds every second
 * between tick events, like a game engine's interpolation.
 */
let predictInterval: number | null = null;

export function startPredictiveCountdown() {
	if (predictInterval) return;
	predictInterval = window.setInterval(() => {
		roomUsers.update((users) => {
			let changed = false;
			const updated = users.map((user) => {
				if (
					user.pomodoro?.phase &&
					user.pomodoro.phase !== 'idle' &&
					user.pomodoro.phase !== 'paused' &&
					user.pomodoro.remaining_seconds !== undefined &&
					user.pomodoro.remaining_seconds > 0
				) {
					changed = true;
					return {
						...user,
						pomodoro: {
							...user.pomodoro,
							remaining_seconds: Math.max(0, user.pomodoro.remaining_seconds - 1),
						},
					};
				}
				return user;
			});
			return changed ? updated : users;
		});
	}, 1000);
}

export function stopPredictiveCountdown() {
	if (predictInterval) {
		clearInterval(predictInterval);
		predictInterval = null;
	}
}

function handleUserJoined(data: { user: { id: string; username: string; is_online: boolean } }) {
	// Refresh full user list to get complete user info
	refreshRoomUsers();
}

function handleUserLeft(data: { user_id: string }) {
	const users = get(roomUsers);
	roomUsers.set(users.filter((u) => u.id !== data.user_id));
}

// Actions

export async function createRoom(
	roomName: string,
	username: string,
	password: string,
	roomPassword?: string
) {
	isLoading.set(true);
	error.set(null);

	try {
		const response = await api.createRoom({
			room_name: roomName,
			username,
			password,
			room_password: roomPassword,
		});

		const { room, member, token, access_token, refresh_token } = response.data;
		saveAuth(token, room.name, member, room);
		if (access_token && refresh_token) {
			saveJWT(access_token, refresh_token);
		}

		currentRoom.set(room);
		currentMember.set(member);
		roomUsers.set([
			{
				id: member.id,
				username: member.username,
				is_owner: member.is_owner,
				is_persistent: member.is_persistent,
				is_online: true,
			},
		]);

		// Connect SSE for real-time updates
		connectSSE();

		return true;
	} catch (e: any) {
		error.set(getErrorMessage(e, get(locale)));
		return false;
	} finally {
		isLoading.set(false);
	}
}

export async function joinRoom(
	roomName: string,
	username: string,
	roomPassword?: string,
	password?: string
) {
	isLoading.set(true);
	error.set(null);

	try {
		const response = await api.joinRoom(roomName, {
			username,
			room_password: roomPassword,
			password,
		});

		const { room, member, token, access_token, refresh_token } = response.data;
		saveAuth(token, room.name, member, room);
		if (access_token && refresh_token) {
			saveJWT(access_token, refresh_token);
		}

		currentRoom.set(room);
		currentMember.set(member);

		// Initial refresh
		await refreshRoomUsers();

		// Connect SSE for real-time updates
		connectSSE();

		return true;
	} catch (e: any) {
		error.set(getErrorMessage(e, get(locale)));
		return false;
	} finally {
		isLoading.set(false);
	}
}

export async function leaveRoom() {
	isLoading.set(true);
	error.set(null);

	try {
		await api.leaveRoom();
		logout();
		return true;
	} catch (e: any) {
		error.set(getErrorMessage(e, get(locale)));
		return false;
	} finally {
		isLoading.set(false);
	}
}

export function logout() {
	disconnectSSE();
	clearAuth();
	currentMember.set(null);
	currentRoom.set(null);
	roomUsers.set([]);
	pomodoroStatus.set({ phase: 'idle' });
}

export async function refreshRoomUsers() {
	try {
		const response = await api.getRoomUsers();
		roomUsers.set(response.data.users);
	} catch (e: any) {
		error.set(getErrorMessage(e, get(locale)));
	}
}

export async function startPomodoro(options?: {
	planned_duration?: number;
	rest_duration?: number;
	long_break_duration?: number;
	sessions_before_long_break?: number;
	session_index?: number;
	tag_id?: string;
	task_id?: string;
}) {
	isLoading.set(true);
	error.set(null);

	try {
		const roomName = get(currentRoom)?.name;
		if (!roomName) throw new Error('Not in a room');

		const response = await api.startPomodoro({
			room_name: roomName,
			...options,
		});

		pomodoroStatus.set(response.data);

		return true;
	} catch (e: any) {
		error.set(getErrorMessage(e, get(locale)));
		return false;
	} finally {
		isLoading.set(false);
	}
}

export async function followPomodoro(leaderId: string) {
	isLoading.set(true);
	error.set(null);

	try {
		const roomName = get(currentRoom)?.name;
		if (!roomName) throw new Error('Not in a room');

		const response = await api.followPomodoro({
			room_name: roomName,
			leader_id: leaderId,
		});

		pomodoroStatus.set(response.data);

		return true;
	} catch (e: any) {
		error.set(getErrorMessage(e, get(locale)));
		return false;
	} finally {
		isLoading.set(false);
	}
}

export async function unfollowPomodoro() {
	isLoading.set(true);
	error.set(null);

	try {
		const roomName = get(currentRoom)?.name;
		if (!roomName) throw new Error('Not in a room');

		await api.unfollowPomodoro(roomName);
		pomodoroStatus.set({ phase: 'idle' });

		return true;
	} catch (e: any) {
		error.set(getErrorMessage(e, get(locale)));
		return false;
	} finally {
		isLoading.set(false);
	}
}

export async function endPomodoro(aborted = false, sessionIndex = 0) {
	isLoading.set(true);
	error.set(null);

	try {
		const roomName = get(currentRoom)?.name;
		if (!roomName) throw new Error('Not in a room');

		const response = await api.endPomodoro(roomName, aborted, sessionIndex);
		pomodoroStatus.set(response.data);

		return true;
	} catch (e: any) {
		error.set(getErrorMessage(e, get(locale)));
		return false;
	} finally {
		isLoading.set(false);
	}
}

export async function updateStatus(emoji: string, message: string) {
	isLoading.set(true);
	error.set(null);

	try {
		const roomName = get(currentRoom)?.name;
		if (!roomName) throw new Error('Not in a room');

		await api.updateStatus({
			room_name: roomName,
			emoji,
			message,
		});

		return true;
	} catch (e: any) {
		error.set(getErrorMessage(e, get(locale)));
		return false;
	} finally {
		isLoading.set(false);
	}
}
