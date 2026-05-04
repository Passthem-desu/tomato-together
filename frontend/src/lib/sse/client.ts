// SSE client — manual reconnect + watchdog for proxy/routing scenarios

export type SSEEventType =
  | 'tick' | 'ping' | 'connected'
  | 'user_joined' | 'user_left'
  | 'pomodoro_started' | 'pomodoro_ended'
  | 'pomodoro_followed' | 'pomodoro_unfollowed'
  | 'leader_aborted' | 'status_updated'
  | 'announcement' | 'token_expired' | 'phase_changed';

export interface TickUser {
  id: string;
  username: string;
  remaining_seconds: number;
  phase: 'idle' | 'focusing' | 'paused' | 'following' | 'rest';
}
export interface TickData { timestamp: string; users: TickUser[]; }

type EventHandler = (data: any) => void;

const EVENT_TYPES: SSEEventType[] = [
  'tick', 'ping', 'connected', 'user_joined', 'user_left',
  'pomodoro_started', 'pomodoro_ended', 'pomodoro_followed', 'pomodoro_unfollowed',
  'leader_aborted', 'status_updated', 'announcement', 'token_expired', 'phase_changed',
];

const WATCHDOG_INTERVAL = 3000;   // check every 3s
const WATCHDOG_TIMEOUT  = 10000;  // no event for 10s → dead

export class SSEClient {
  private source: EventSource | null = null;
  private handlers = new Map<SSEEventType, Set<EventHandler>>();
  private roomName: string;
  private token: string;
  private closed = false;
  private reconnectDelay = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private watchdogTimer: ReturnType<typeof setInterval> | null = null;
  private lastEventTime = 0;
  private onStatusChange?: (connected: boolean) => void;
  private reconnecting = false;

  constructor(roomName: string, token: string, onStatusChange?: (connected: boolean) => void) {
    this.roomName = roomName;
    this.token = token;
    this.onStatusChange = onStatusChange;
  }

  connect(): void {
    this.closed = false;
    this.reconnectDelay = 0;
    this.openConnection();
    this.startWatchdog();
  }

  disconnect(): void {
    this.closed = true;
    this.stopWatchdog();
    clearTimeout(this.reconnectTimer!);
    this.reconnectTimer = null;
    if (this.source) { this.source.close(); this.source = null; }
    this.onStatusChange?.(false);
  }

  on(event: SSEEventType, handler: EventHandler): () => void {
    if (!this.handlers.has(event)) this.handlers.set(event, new Set());
    this.handlers.get(event)!.add(handler);
    return () => this.handlers.get(event)?.delete(handler);
  }

  // ── private ──

  private openConnection(): void {
    if (this.closed) return;

    const url = this.token
      ? `/api/rooms/${encodeURIComponent(this.roomName)}/sse?token=${encodeURIComponent(this.token)}`
      : `/api/rooms/${encodeURIComponent(this.roomName)}/sse`;

    const es = new EventSource(url);
    this.source = es;

    es.onopen = () => {
      this.reconnectDelay = 0;
      this.lastEventTime = Date.now();
      this.onStatusChange?.(true);
    };

    es.onerror = () => {
      es.close();
      this.source = null;
      this.onStatusChange?.(false);
      if (!this.closed) this.scheduleReconnect();
    };

    const bump = () => { this.lastEventTime = Date.now(); };

    for (const type of EVENT_TYPES) {
      es.addEventListener(type, (e: MessageEvent) => {
        bump();
        try { this.dispatch(type, JSON.parse(e.data)); } catch { /* skip */ }
      });
    }
  }

  private startWatchdog(): void {
    this.stopWatchdog();
    this.lastEventTime = Date.now();
    this.watchdogTimer = setInterval(() => {
      if (this.closed) return;
      if (Date.now() - this.lastEventTime > WATCHDOG_TIMEOUT) {
        console.warn('[SSE] watchdog timeout');
        this.onStatusChange?.(false);
        if (this.source) { this.source.close(); this.source = null; }
        this.scheduleReconnect();
      }
    }, WATCHDOG_INTERVAL);
  }

  private stopWatchdog(): void {
    if (this.watchdogTimer) { clearInterval(this.watchdogTimer); this.watchdogTimer = null; }
  }

  private scheduleReconnect(): void {
    if (this.reconnecting) return;
    this.reconnecting = true;
    clearTimeout(this.reconnectTimer!);
    const delay = this.reconnectDelay === 0 ? 3000 : Math.min(this.reconnectDelay * 2, 30000);
    this.reconnectDelay = delay;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.reconnecting = false;
      this.openConnection();
    }, delay);
  }

  private dispatch(event: SSEEventType, data: any): void {
    this.handlers.get(event)?.forEach(fn => { try { fn(data); } catch { /* skip */ } });
  }
}
