import { browser } from '$app/environment';
import { sseConnectedStore, roomMembersStore, localTimerStore, notificationsStore } from '$lib/stores';
import { getSSEUrl } from '$lib/api/client';
import type { RoomMember } from '$lib/types';

let eventSource: EventSource | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let pingInterval: ReturnType<typeof setInterval> | null = null;

export type SSEEventHandler = (data: any) => void;

const eventHandlers: Map<string, Set<SSEEventHandler>> = new Map();

export function subscribeEvent(event: string, handler: SSEEventHandler) {
  if (!eventHandlers.has(event)) {
    eventHandlers.set(event, new Set());
  }
  eventHandlers.get(event)!.add(handler);

  return () => {
    eventHandlers.get(event)?.delete(handler);
  };
}

function emitEvent(event: string, data: any) {
  const handlers = eventHandlers.get(event);
  if (handlers) {
    handlers.forEach(handler => handler(data));
  }
}

export function connectSSE(roomName: string, token?: string) {
  if (!browser) return;

  disconnectSSE();

  const url = getSSEUrl(roomName, token);
  
  try {
    eventSource = new EventSource(url);

    eventSource.onopen = () => {
      sseConnectedStore.set(true);
      emitEvent('connected', { room: roomName });
      
      // 每 30 秒发送 ping
      pingInterval = setInterval(() => {
        sendPing();
      }, 30000);
    };

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        handleEvent(data);
      } catch (e) {
        console.error('Failed to parse SSE message:', e);
      }
    };

    eventSource.onerror = () => {
      sseConnectedStore.set(false);
      emitEvent('error', {});
      
      // 尝试重连
      if (reconnectTimer) clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(() => {
        if (browser) {
          connectSSE(roomName, token);
        }
      }, 5000);
    };

    // 处理不同的事件
    eventSource.addEventListener('connected', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('connected', data);
    });

    eventSource.addEventListener('user_joined', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('user_joined', data);
    });

    eventSource.addEventListener('user_left', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('user_left', data);
    });

    eventSource.addEventListener('user_offline', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('user_offline', data);
    });

    eventSource.addEventListener('pomodoro_started', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('pomodoro_started', data);
    });

    eventSource.addEventListener('pomodoro_ended', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('pomodoro_ended', data);
    });

    eventSource.addEventListener('pomodoro_followed', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('pomodoro_followed', data);
    });

    eventSource.addEventListener('pomodoro_unfollowed', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('pomodoro_unfollowed', data);
    });

    eventSource.addEventListener('leader_aborted', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('leader_aborted', data);
      
      // 显示通知
      notificationsStore.update(n => [...n, {
        type: 'leader_aborted',
        title: '主导者已结束',
        body: `${data.leader_username} 提前结束了番茄`
      }]);
    });

    eventSource.addEventListener('status_updated', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('status_updated', data);
    });

    eventSource.addEventListener('tick', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('tick', data);
      
      // 更新房间成员状态
      if (data.users) {
        roomMembersStore.update(members => {
          return members.map(m => {
            const userData = data.users.find((u: any) => u.id === m.id);
            if (userData) {
              return {
                ...m,
                pomodoro: {
                  ...m.pomodoro,
                  remaining_seconds: userData.remaining_seconds,
                  status: userData.status
                }
              };
            }
            return m;
          });
        });
      }
    });

    eventSource.addEventListener('announcement', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('announcement', data);
      
      // 显示通知（hold 到休息阶段）
      notificationsStore.update(n => [...n, {
        type: 'announcement',
        title: data.title,
        body: data.body
      }]);
    });

    eventSource.addEventListener('room_settings_changed', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('room_settings_changed', data);
    });

    eventSource.addEventListener('token_expired', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('token_expired', data);
      
      // 提示用户重新加入
      notificationsStore.update(n => [...n, {
        type: 'token_expired',
        title: 'Token 已过期',
        body: '请重新加入房间'
      }]);
    });

    eventSource.addEventListener('pong', (event) => {
      const data = JSON.parse(event.data);
      emitEvent('pong', data);
    });

  } catch (e) {
    console.error('Failed to connect SSE:', e);
    sseConnectedStore.set(false);
  }
}

function handleEvent(data: any) {
  // 通用的事件处理
  console.log('SSE event:', data);
}

export function sendPing() {
  // SSE 通过 EventSource 无法主动发送消息，这里仅作为心跳记录
  // 实际的心跳由后端的 tick 机制处理
}

export function disconnectSSE() {
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
  
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  
  if (pingInterval) {
    clearInterval(pingInterval);
    pingInterval = null;
  }
  
  sseConnectedStore.set(false);
  eventHandlers.clear();
}

export function isSSEConnected(): boolean {
  return eventSource !== null && eventSource.readyState === EventSource.OPEN;
}

// 更新本地计时器状态
export function syncTimerFromTick(tickData: any) {
  // 从 tick 数据同步本地计时器
  // 容忍 ±2 秒偏差
}

export default {
  connect: connectSSE,
  disconnect: disconnectSSE,
  subscribe: subscribeEvent,
  isConnected: isSSEConnected
};