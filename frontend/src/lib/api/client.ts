import { API_BASE_URL, TOKEN_KEY, ROOM_TOKEN_KEY } from '$lib/types';
import type { ApiResponse, User, Room, RoomMember, PomodoroSession, Project, Task, Announcement, RoomStats } from '$lib/types';

async function fetchApi<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem(TOKEN_KEY) : null;
  const roomToken = typeof localStorage !== 'undefined' ? localStorage.getItem(ROOM_TOKEN_KEY) : null;

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {})
  };

  if (token && !endpoint.includes('/rooms/')) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  // 使用代理：相对路径会被 Vite 转发到后端
  let url = `/api${endpoint}`;
  
  // 对于 L2 认证的接口，优先使用 room_token
  if (roomToken && (endpoint.includes('/pomodoro') || endpoint.includes('/status') || endpoint.includes('/leave') || endpoint.includes('/announcements'))) {
    url += (url.includes('?') ? '&' : '?') + `token=${roomToken}`;
  }

  const response = await fetch(url, {
    ...options,
    headers
  });

  const data = await response.json();
  
  if (!response.ok) {
    throw new Error(data.error || 'API request failed');
  }

  return data as T;
}

// 认证 API
export const auth = {
  async register(username: string, password: string): Promise<{ token: string; user: User }> {
    const res = await fetchApi<ApiResponse<{ token: string; user: User }>>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, password })
    });
    if (res.success && res.data) {
      localStorage.setItem(TOKEN_KEY, res.data.token);
      localStorage.setItem(USER_KEY, JSON.stringify(res.data.user));
      return res.data;
    }
    throw new Error('Registration failed');
  },

  async login(username: string, password: string): Promise<{ token: string; user: User }> {
    const res = await fetchApi<ApiResponse<{ token: string; user: User }>>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password })
    });
    if (res.success && res.data) {
      localStorage.setItem(TOKEN_KEY, res.data.token);
      localStorage.setItem(USER_KEY, JSON.stringify(res.data.user));
      return res.data;
    }
    throw new Error('Login failed');
  },

  async me(): Promise<User> {
    const res = await fetchApi<ApiResponse<User>>('/auth/me');
    if (res.success && res.data) {
      return res.data;
    }
    throw new Error('Get user info failed');
  },

  async upgrade(password: string): Promise<User> {
    const res = await fetchApi<ApiResponse<{ message: string; user: User }>>('/auth/upgrade', {
      method: 'POST',
      body: JSON.stringify({ password })
    });
    if (res.success && res.data && 'user' in res.data) {
      return (res.data as { message: string; user: User }).user;
    }
    throw new Error('Upgrade failed');
  },

  logout() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
  }
};

// 房间 API
export const rooms = {
  async create(name: string, roomPassword?: string, isReadonly?: boolean): Promise<Room> {
    const res = await fetchApi<ApiResponse<Room>>('/rooms', {
      method: 'POST',
      body: JSON.stringify({
        name,
        room_password: roomPassword || '',
        is_readonly: isReadonly || false
      })
    });
    if (res.success && res.data) return res.data;
    throw new Error('Create room failed');
  },

  async get(name: string): Promise<Room> {
    const res = await fetchApi<ApiResponse<Room>>(`/rooms/${encodeURIComponent(name)}`);
    if (res.success && res.data) return res.data;
    throw new Error('Get room failed');
  },

  async join(name: string, username: string, password?: string): Promise<{ room: Room; user: User; token: string }> {
    const res = await fetchApi<ApiResponse<{ room: Room; user: User; token: string }>>(
      `/rooms/${encodeURIComponent(name)}/join`,
      {
        method: 'POST',
        body: JSON.stringify({ username, password })
      }
    );
    if (res.success && res.data) {
      localStorage.setItem(ROOM_TOKEN_KEY, res.data.token);
      return res.data;
    }
    throw new Error('Join room failed');
  },

  async leave(name: string): Promise<void> {
    await fetchApi<ApiResponse<{ message: string }>>(
      `/rooms/${encodeURIComponent(name)}/leave`,
      { method: 'POST' }
    );
    localStorage.removeItem(ROOM_TOKEN_KEY);
    localStorage.removeItem('current_room');
  },

  async getUsers(name: string): Promise<RoomMember[]> {
    const res = await fetchApi<ApiResponse<{ users: RoomMember[] }>>(
      `/rooms/${encodeURIComponent(name)}/users`
    );
    if (res.success && res.data && 'users' in res.data) {
      return (res.data as { users: RoomMember[] }).users;
    }
    throw new Error('Get users failed');
  },

  async getStats(name: string): Promise<RoomStats> {
    const res = await fetchApi<ApiResponse<RoomStats>>(
      `/rooms/${encodeURIComponent(name)}/stats`
    );
    if (res.success && res.data) return res.data;
    throw new Error('Get stats failed');
  },

  async createAnnouncement(name: string, title: string, body: string): Promise<string> {
    const res = await fetchApi<ApiResponse<{ message: string; announcement_id: string }>>(
      `/rooms/${encodeURIComponent(name)}/announcement`,
      {
        method: 'POST',
        body: JSON.stringify({ title, body })
      }
    );
    if (res.success && res.data && 'announcement_id' in res.data) {
      return (res.data as { message: string; announcement_id: string }).announcement_id;
    }
    throw new Error('Create announcement failed');
  },

  async getAnnouncements(name: string, limit = 20): Promise<Announcement[]> {
    const res = await fetchApi<ApiResponse<{ announcements: Announcement[] }>>(
      `/rooms/${encodeURIComponent(name)}/announcements?limit=${limit}`
    );
    if (res.success && res.data && 'announcements' in res.data) {
      return (res.data as { announcements: Announcement[] }).announcements;
    }
    throw new Error('Get announcements failed');
  }
};

// 番茄 API
export const pomodoro = {
  async start(
    roomName: string,
    plannedDuration = 1500,
    restDuration = 300,
    longBreakDuration = 900,
    sessionsBeforeLongBreak = 4,
    projectId?: string,
    taskId?: string
  ): Promise<PomodoroSession> {
    const res = await fetchApi<ApiResponse<PomodoroSession>>('/pomodoro/start', {
      method: 'POST',
      body: JSON.stringify({
        room_name: roomName,
        planned_duration: plannedDuration,
        rest_duration: restDuration,
        long_break_duration: longBreakDuration,
        sessions_before_long_break: sessionsBeforeLongBreak,
        project_id: projectId,
        task_id: taskId
      })
    });
    if (res.success && res.data) return res.data;
    throw new Error('Start pomodoro failed');
  },

  async follow(roomName: string, leaderId: string): Promise<PomodoroSession> {
    const res = await fetchApi<ApiResponse<PomodoroSession>>('/pomodoro/follow', {
      method: 'POST',
      body: JSON.stringify({ room_name: roomName, leader_id: leaderId })
    });
    if (res.success && res.data) return res.data;
    throw new Error('Follow pomodoro failed');
  },

  async unfollow(roomName: string): Promise<void> {
    await fetchApi<ApiResponse<{ message: string; status: string }>>('/pomodoro/unfollow', {
      method: 'POST',
      body: JSON.stringify({ room_name: roomName })
    });
  },

  async end(roomName: string, aborted = false): Promise<PomodoroSession> {
    const res = await fetchApi<ApiResponse<PomodoroSession>>('/pomodoro/end', {
      method: 'POST',
      body: JSON.stringify({ room_name: roomName, aborted })
    });
    if (res.success && res.data) return res.data;
    throw new Error('End pomodoro failed');
  },

  async getStatus(): Promise<{ is_active: boolean; status: string; session_id?: string; remaining_seconds?: number; leader_id?: string; leader_username?: string }> {
    const res = await fetchApi<ApiResponse<any>>('/pomodoro/status');
    if (res.success && res.data) return res.data;
    return { is_active: false, status: 'idle' };
  }
};

// 状态 API
export const status = {
  async update(roomName: string, emoji: string, message: string): Promise<void> {
    await fetchApi<ApiResponse<{ emoji: string; message: string; updated_at: string }>>('/status', {
      method: 'PUT',
      body: JSON.stringify({ room_name: roomName, emoji, message })
    });
  },

  async clear(roomName: string): Promise<void> {
    await fetchApi<ApiResponse<{ message: string }>>('/status', {
      method: 'DELETE',
      body: JSON.stringify({ room_name: roomName })
    });
  }
};

// 项目 API
export const projects = {
  async list(): Promise<Project[]> {
    const res = await fetchApi<ApiResponse<{ projects: Project[] }>>('/projects');
    if (res.success && res.data && 'projects' in res.data) {
      return (res.data as { projects: Project[] }).projects;
    }
    return [];
  },

  async create(name: string): Promise<Project> {
    const res = await fetchApi<ApiResponse<Project>>('/projects', {
      method: 'POST',
      body: JSON.stringify({ name })
    });
    if (res.success && res.data) return res.data;
    throw new Error('Create project failed');
  },

  async update(id: string, name: string): Promise<void> {
    await fetchApi<ApiResponse<{ id: string; name: string; updated_at: string }>>(
      `/projects/${id}`,
      { method: 'PUT', body: JSON.stringify({ name }) }
    );
  },

  async delete(id: string): Promise<void> {
    await fetchApi<ApiResponse<{ message: string }>>(`/projects/${id}`, { method: 'DELETE' });
  }
};

// 任务 API
export const tasks = {
  async list(status?: string, projectId?: string): Promise<Task[]> {
    let url = '/tasks';
    const params = [];
    if (status) params.push(`status=${status}`);
    if (projectId) params.push(`project_id=${projectId}`);
    if (params.length) url += '?' + params.join('&');

    const res = await fetchApi<ApiResponse<{ tasks: Task[] }>>(url);
    if (res.success && res.data && 'tasks' in res.data) {
      return (res.data as { tasks: Task[] }).tasks;
    }
    return [];
  },

  async create(clientId: string, title: string, projectId?: string): Promise<Task> {
    const res = await fetchApi<ApiResponse<Task>>('/tasks', {
      method: 'POST',
      body: JSON.stringify({ client_id: clientId, title, project_id: projectId })
    });
    if (res.success && res.data) return res.data;
    throw new Error('Create task failed');
  },

  async update(id: string, data: { title?: string; status?: string; project_id?: string }): Promise<Task> {
    const res = await fetchApi<ApiResponse<Task>>(`/tasks/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data)
    });
    if (res.success && res.data) return res.data;
    throw new Error('Update task failed');
  },

  async delete(id: string): Promise<void> {
    await fetchApi<ApiResponse<{ message: string }>>(`/tasks/${id}`, { method: 'DELETE' });
  },

  async sync(clientTasks: { client_id: string; title: string; status: string; project_id?: string }[]): Promise<{ client_id: string; server_id: string }[]> {
    const res = await fetchApi<ApiResponse<{ synced: number; tasks: { client_id: string; server_id: string }[] }>>(
      '/tasks/sync',
      {
        method: 'POST',
        body: JSON.stringify({ tasks: clientTasks })
      }
    );
    if (res.success && res.data && 'tasks' in res.data) {
      return (res.data as { synced: number; tasks: { client_id: string; server_id: string }[] }).tasks;
    }
    return [];
  }
};

// SSE URL 构造
export function getSSEUrl(roomName: string, token?: string): string {
  const roomToken = token || (typeof localStorage !== 'undefined' ? localStorage.getItem(ROOM_TOKEN_KEY) : null);
  const encodedName = encodeURIComponent(roomName);
  return roomToken
    ? `/api/rooms/${encodedName}/sse?token=${roomToken}`
    : `/api/rooms/${encodedName}/sse`;
}