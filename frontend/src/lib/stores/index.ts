import { writable, derived } from 'svelte/store';
import type { User, Room, RoomMember, PomodoroSession, Task, Project } from '$lib/types';
import { browser } from '$app/environment';

// 用户 Store
function createUserStore() {
  const stored = browser ? localStorage.getItem('tomatogether_user') : null;
  const initial = stored ? JSON.parse(stored) : null;
  
  const { subscribe, set, update } = writable<User | null>(initial);

  return {
    subscribe,
    set: (user: User | null) => {
      if (browser) {
        if (user) {
          localStorage.setItem('tomatogether_user', JSON.stringify(user));
        } else {
          localStorage.removeItem('tomatogether_user');
        }
      }
      set(user);
    },
    update
  };
}

export const userStore = createUserStore();
export const isLoggedIn = derived(userStore, $user => $user !== null);

// Token Store
function createTokenStore() {
  const stored = browser ? localStorage.getItem('tomatogether_token') : null;
  const { subscribe, set } = writable<string | null>(stored);

  return {
    subscribe,
    set: (token: string | null) => {
      if (browser) {
        if (token) {
          localStorage.setItem('tomatogether_token', token);
        } else {
          localStorage.removeItem('tomatogether_token');
        }
      }
      set(token);
    }
  };
}

export const tokenStore = createTokenStore();

// 房间 Token Store
function createRoomTokenStore() {
  const stored = browser ? localStorage.getItem('tomatogether_room_token') : null;
  const { subscribe, set } = writable<string | null>(stored);

  return {
    subscribe,
    set: (token: string | null) => {
      if (browser) {
        if (token) {
          localStorage.setItem('tomatogether_room_token', token);
        } else {
          localStorage.removeItem('tomatogether_room_token');
        }
      }
      set(token);
    }
  };
}

export const roomTokenStore = createRoomTokenStore();

// 当前房间 Store
function createCurrentRoomStore() {
  const stored = browser ? localStorage.getItem('current_room') : null;
  const initial = stored ? JSON.parse(stored) : null;
  
  const { subscribe, set, update } = writable<Room | null>(initial);

  return {
    subscribe,
    set: (room: Room | null) => {
      if (browser) {
        if (room) {
          localStorage.setItem('current_room', JSON.stringify(room));
        } else {
          localStorage.removeItem('current_room');
        }
      }
      set(room);
    },
    update
  };
}

export const currentRoomStore = createCurrentRoomStore();

// 房间成员列表
export const roomMembersStore = writable<RoomMember[]>([]);

// 番茄状态
export const pomodoroStore = writable<PomodoroSession | null>(null);

// 本地番茄计时器（客户端计算）
function createLocalTimerStore() {
  const initial = {
    startedAt: null as Date | null,
    plannedDuration: 1500,
    remaining: 1500,
    status: 'idle' as 'idle' | 'focusing' | 'following' | 'rest'
  };
  
  const { subscribe, set, update } = writable(initial);

  let interval: ReturnType<typeof setInterval> | null = null;

  return {
    subscribe,
    start: (plannedDuration: number, status: 'focusing' | 'following') => {
      if (interval) clearInterval(interval);
      const now = new Date();
      set({
        startedAt: now,
        plannedDuration,
        remaining: plannedDuration,
        status
      });
      
      interval = setInterval(() => {
        update(t => {
          const elapsed = Math.floor((Date.now() - (t.startedAt?.getTime() || Date.now())) / 1000);
          const remaining = Math.max(0, t.plannedDuration - elapsed);
          return { ...t, remaining };
        });
      }, 1000);
    },
    stop: () => {
      if (interval) clearInterval(interval);
      interval = null;
      set({ ...initial, startedAt: null });
    },
    rest: (duration: number) => {
      if (interval) clearInterval(interval);
      set({
        startedAt: new Date(),
        plannedDuration: duration,
        remaining: duration,
        status: 'rest'
      });
      
      interval = setInterval(() => {
        update(t => {
          const elapsed = Math.floor((Date.now() - (t.startedAt?.getTime() || Date.now())) / 1000);
          const remaining = Math.max(0, t.plannedDuration - elapsed);
          return { ...t, remaining };
        });
      }, 1000);
    }
  };
}

export const localTimerStore = createLocalTimerStore();

// WIP 任务列表
export const tasksStore = writable<Task[]>([]);

// 项目列表
export const projectsStore = writable<Project[]>([]);

// SSE 连接状态
export const sseConnectedStore = writable(false);

// 通知列表
export const notificationsStore = writable<{ type: string; title: string; body: string }[]>([]);

// 错误状态
export const errorStore = writable<string | null>(null);

// 清除错误
export function clearError() {
  errorStore.set(null);
}