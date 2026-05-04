import { writable, derived, get } from 'svelte/store';
import type { Member, Room, UserInfo, PomodoroStatus } from './api';
import { api, saveAuth, clearAuth } from './api';
import { getErrorMessage, locale } from './i18n';

// App state
export const currentMember = writable<Member | null>(null);
export const currentRoom = writable<Room | null>(null);
export const roomUsers = writable<UserInfo[]>([]);
export const pomodoroStatus = writable<PomodoroStatus>({ is_active: false, status: 'idle' });
export const isLoading = writable(false);
export const error = writable<string | null>(null);

// Derived stores
export const isAuthenticated = derived(currentMember, ($member) => !!$member);
export const isOwner = derived(currentMember, ($member) => $member?.is_owner ?? false);

// Actions
export async function createRoom(roomName: string, username: string, password: string, roomPassword?: string) {
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
    saveAuth(token, room.name, member);
    
    currentRoom.set(room);
    currentMember.set(member);
    roomUsers.set([{
      id: member.id,
      username: member.username,
      is_owner: member.is_owner,
      is_persistent: member.is_persistent,
      is_online: true,
    }]);
    
    return true;
  } catch (e: any) {
    error.set(getErrorMessage(e, get(locale)));
    return false;
  } finally {
    isLoading.set(false);
  }
}

export async function joinRoom(roomName: string, username: string, roomPassword?: string, password?: string) {
  isLoading.set(true);
  error.set(null);
  
  try {
    const response = await api.joinRoom(roomName, {
      username,
      room_password: roomPassword,
      password,
    });
    
    const { room, member, token } = response.data;
    saveAuth(token, room.name, member);
    
    currentRoom.set(room);
    currentMember.set(member);
    
    // Refresh users after join
    await refreshRoomUsers();
    
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
  clearAuth();
  currentMember.set(null);
  currentRoom.set(null);
  roomUsers.set([]);
  pomodoroStatus.set({ is_active: false, status: 'idle' });
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
    await refreshRoomUsers();
    
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
    await refreshRoomUsers();
    
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
    pomodoroStatus.set({ is_active: false, status: 'idle' });
    await refreshRoomUsers();
    
    return true;
  } catch (e: any) {
    error.set(getErrorMessage(e, get(locale)));
    return false;
  } finally {
    isLoading.set(false);
  }
}

export async function endPomodoro(aborted = false) {
  isLoading.set(true);
  error.set(null);
  
  try {
    const roomName = get(currentRoom)?.name;
    if (!roomName) throw new Error('Not in a room');
    
    const response = await api.endPomodoro(roomName, aborted);
    pomodoroStatus.set(response.data);
    await refreshRoomUsers();
    
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
    
    await refreshRoomUsers();
    
    return true;
  } catch (e: any) {
    error.set(getErrorMessage(e, get(locale)));
    return false;
  } finally {
    isLoading.set(false);
  }
}
