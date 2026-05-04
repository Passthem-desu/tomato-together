// API 配置
// 开发环境下使用相对路径，通过 Vite 代理转发到后端
// 生产环境请根据实际情况配置或使用环境变量
export const API_BASE_URL = '';

// 认证相关
export const TOKEN_KEY = 'tomatogether_token';
export const USER_KEY = 'tomatogether_user';
export const ROOM_TOKEN_KEY = 'tomatogether_room_token';
export const ROOM_KEY = 'tomatogether_room';

// API 响应类型
export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

// 用户类型
export interface User {
  id: string;
  username: string;
  has_password?: boolean;
  created_at?: string;
}

// 房间类型
export interface Room {
  id: string;
  name: string;
  is_readonly: boolean;
  has_password: boolean;
  member_count?: number;
  owner?: User;
  created_at?: string;
}

// 房间成员
export interface RoomMember {
  id: string;
  username: string;
  is_owner: boolean;
  is_online: boolean;
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
}

// 番茄会话
export interface PomodoroSession {
  session_id: string;
  started_at: string;
  planned_duration: number;
  rest_duration?: number;
  long_break_duration?: number;
  status: 'focusing' | 'following' | 'rest' | 'idle';
  sessions_today?: number;
}

// 项目
export interface Project {
  id: string;
  name: string;
  created_at: string;
}

// WIP 任务
export interface Task {
  id: string;
  client_id?: string;
  title: string;
  status: 'TODO' | 'WIP' | 'DONE';
  project_id?: string;
  project_name?: string;
  created_at: string;
  completed_at?: string;
}

// 公告
export interface Announcement {
  id: string;
  title: string;
  body: string;
  sender: User;
  created_at: string;
}

// 房间统计
export interface RoomStats {
  date: string;
  total_pomodoros: number;
  total_duration: number;
  active_users: number;
}

// SSE 事件类型
export interface SSETickData {
  timestamp: string;
  users: {
    id: string;
    username: string;
    remaining_seconds?: number;
    status?: string;
  }[];
}