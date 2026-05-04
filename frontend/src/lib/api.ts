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
    is_active: boolean;
    is_following: boolean;
    leader_username?: string;
    started_at?: string;
    remaining_seconds?: number;
  };
  is_online: boolean;
}

export interface PomodoroStatus {
  is_active: boolean;
  status: 'idle' | 'focusing' | 'following' | 'rest';
  session_id?: string;
  started_at?: string;
  remaining_seconds?: number;
  leader_id?: string;
  leader_username?: string;
  planned_duration?: number;
  rest_duration?: number;
  long_break_duration?: number;
  sessions_before_long_break?: number;
}

// API helper
async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
  
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
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
  joinRoom: async (roomName: string, data: {
    username: string;
    password?: string;
    room_password?: string;
  }): Promise<{ success: boolean; data: RoomResponse }> => {
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
  checkUser: async (roomName: string, username: string): Promise<{ success: boolean; data: { exists: boolean; is_persistent: boolean } }> => {
    return apiRequest(`/rooms/${encodeURIComponent(roomName)}/check-user`, {
      method: 'POST',
      body: JSON.stringify({ username }),
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

  // Auth
  getMe: async (): Promise<{ success: boolean; data: Member }> => {
    return apiRequest('/auth/me');
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
    project_id?: string;
    task_id?: string;
    planned_duration?: number;
    rest_duration?: number;
    long_break_duration?: number;
    sessions_before_long_break?: number;
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

  endPomodoro: async (roomName: string, aborted = false): Promise<{ success: boolean; data: PomodoroStatus }> => {
    return apiRequest('/pomodoro/end', {
      method: 'POST',
      body: JSON.stringify({ room_name: roomName, aborted }),
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
};

// Store helpers
export function saveAuth(token: string, roomName: string, member: Member) {
  localStorage.setItem('token', token);
  localStorage.setItem('room_name', roomName);
  localStorage.setItem('member', JSON.stringify(member));
}

export function clearAuth() {
  localStorage.removeItem('token');
  localStorage.removeItem('room_name');
  localStorage.removeItem('member');
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
