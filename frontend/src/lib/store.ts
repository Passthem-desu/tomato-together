import { writable, derived, get } from 'svelte/store';
import type { Member, Room, UserInfo, PomodoroStatus } from './api';
import { api, saveAuth, clearAuth } from './api';
import { SSEClient, type TickData } from './sse/client';
import { getErrorMessage, locale } from './i18n';

// App state
export const currentMember = writable<Member | null>(null);
export const currentRoom = writable<Room | null>(null);
export const roomUsers = writable<UserInfo[]>([]);
export const pomodoroStatus = writable<PomodoroStatus>({ phase: 'idle' });
export const isLoading = writable(false);
export const error = writable<string | null>(null);
export const sseConnected = writable(true);

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
	const token = localStorage.getItem('token') || '';
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
		error.set('Token 已过期，请重新加入房间');
		logout();
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
	const users = get(roomUsers);
	const member = get(currentMember);

	// Update remaining seconds and status from tick
	for (const tickUser of data.users) {
		const existing = users.find((u) => u.id === tickUser.id);
		if (existing) {
			// Preserve existing fields (status, is_persistent, etc.) and update pomodoro
			if (!existing.pomodoro) {
				existing.pomodoro = { is_active: false, is_following: false };
			}
			existing.pomodoro.remaining_seconds = tickUser.remaining_seconds;
			existing.pomodoro.phase = tickUser.phase || 'idle';
		}

		// Sync current user's pomodoro status from server (corrects tab-throttling drift)
		if (member && tickUser.id === member.id) {
			pomodoroStatus.update((prev) => ({
				...prev,
				remaining_seconds: tickUser.remaining_seconds,
			}));
		}
	}

	roomUsers.set(users);
}

/**
 * Start predictive countdown: decrement users' remaining_seconds every second
 * between tick events, like a game engine's interpolation.
 */
let predictInterval: number | null = null;

export function startPredictiveCountdown() {
	if (predictInterval) return;
	predictInterval = window.setInterval(() => {
		const users = get(roomUsers);
		let changed = false;
		const now = Date.now();

		for (const user of users) {
			if (
				user.pomodoro?.phase &&
				user.pomodoro.phase !== 'idle' &&
				user.pomodoro.phase !== 'paused' &&
				user.pomodoro.remaining_seconds !== undefined &&
				user.pomodoro.remaining_seconds > 0
			) {
				user.pomodoro.remaining_seconds = Math.max(0, user.pomodoro.remaining_seconds - 1);
				changed = true;
			}
		}

		if (changed) {
			roomUsers.set(users);
		}
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

		const { room, member, token } = response.data;
		saveAuth(token, room.name, member, room);

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

		const { room, member, token } = response.data;
		saveAuth(token, room.name, member, room);

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
