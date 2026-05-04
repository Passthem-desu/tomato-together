// Decoupled countdown module — channel-style event emitter.
// Handles start / pause / resume / stop / setDuration internally.
// Room page just subscribes to ticks and sends commands.

type CountdownState = 'idle' | 'running' | 'paused';

type EventHandler = (...args: any[]) => void;

export class PomodoroCountdown {
	private remaining = 0;
	private duration = 0;
	private state: CountdownState = 'idle';
	private timer: ReturnType<typeof setInterval> | null = null;
	private listeners = new Map<string, Set<EventHandler>>();

	/** Subscribe to an event. Returns unsubscribe function. */
	on(event: string, fn: EventHandler): () => void {
		if (!this.listeners.has(event)) this.listeners.set(event, new Set());
		this.listeners.get(event)!.add(fn);
		return () => this.listeners.get(event)?.delete(fn);
	}

	private emit(event: string, ...args: any[]) {
		this.listeners.get(event)?.forEach((fn) => fn(...args));
	}

	/** Start (or restart) countdown with a new duration in seconds. */
	start(durationSec: number) {
		this.stop();
		this.duration = durationSec;
		this.remaining = durationSec;
		this.state = 'running';
		this.emit('tick', this.remaining);
		this.emit('stateChange', this.state);
		this.startTimer();
	}

	/** Pause the running countdown. */
	pause() {
		if (this.state !== 'running') return;
		this.state = 'paused';
		this.stopTimer();
		this.emit('stateChange', this.state);
	}

	/** Resume from paused. */
	resume() {
		if (this.state !== 'paused') return;
		this.state = 'running';
		this.emit('stateChange', this.state);
		this.startTimer();
	}

	/** Stop completely, reset to idle. */
	stop() {
		this.state = 'idle';
		this.remaining = this.duration;
		this.stopTimer();
		this.emit('stateChange', this.state);
	}

	/** Override remaining seconds (e.g. drift correction from server). */
	setRemaining(sec: number) {
		if (sec !== this.remaining) {
			this.remaining = Math.max(0, sec);
			this.emit('tick', this.remaining);
		}
	}

	/** Get current state. */
	getState(): CountdownState {
		return this.state;
	}
	getRemaining(): number {
		return this.remaining;
	}

	/** Clean up. */
	destroy() {
		this.stopTimer();
		this.listeners.clear();
	}

	// ── private ──

	private startTimer() {
		this.stopTimer();
		this.timer = setInterval(() => this.tick(), 1000);
	}

	private stopTimer() {
		if (this.timer) {
			clearInterval(this.timer);
			this.timer = null;
		}
	}

	private tick() {
		if (this.state !== 'running') return;
		if (this.remaining > 0) {
			this.remaining = Math.max(0, this.remaining - 1);
			this.emit('tick', this.remaining);
		}
		if (this.remaining <= 0) {
			this.stopTimer();
			this.state = 'idle';
			this.emit('complete');
			this.emit('stateChange', this.state);
		}
	}
}
