// Decoupled countdown module — channel-style event emitter.
// Handles start / pause / resume / stop / setDuration internally.
// Room page just subscribes to ticks and sends commands.
//
// Uses absolute timestamps (endTime) instead of decrementing remaining.
// This ensures accurate timing even when the page is backgrounded
// and setInterval is throttled by the browser.

type CountdownState = 'idle' | 'running' | 'paused';

type EventHandler = (...args: any[]) => void;

export class PomodoroCountdown {
	private remaining = 0;
	private state: CountdownState = 'idle';
	private timer: ReturnType<typeof setInterval> | null = null;
	private listeners = new Map<string, Set<EventHandler>>();
	private endTime = 0;
	private pausedRemaining = 0;

	getRemaining(): number {
		if (this.state === 'paused') return this.pausedRemaining;
		if (this.state !== 'running' || this.endTime === 0) return this.remaining;
		return Math.max(0, Math.ceil((this.endTime - Date.now()) / 1000));
	}

	getState(): CountdownState {
		return this.state;
	}

	on(event: string, fn: EventHandler): () => void {
		if (!this.listeners.has(event)) this.listeners.set(event, new Set());
		this.listeners.get(event)!.add(fn);
		return () => this.listeners.get(event)?.delete(fn);
	}

	private emit(event: string, ...args: any[]) {
		this.listeners.get(event)?.forEach((fn) => fn(...args));
	}

	start(durationSec: number) {
		this.stop();
		this.remaining = durationSec;
		this.endTime = Date.now() + durationSec * 1000;
		this.state = 'running';
		this.emit('tick', this.getRemaining());
		this.emit('stateChange', this.state);
		this.startTimer();
	}

	pause() {
		if (this.state !== 'running') return;
		this.pausedRemaining = this.getRemaining();
		this.state = 'paused';
		this.stopTimer();
		this.emit('stateChange', this.state);
	}

	resume() {
		if (this.state !== 'paused') return;
		this.endTime = Date.now() + this.pausedRemaining * 1000;
		this.state = 'running';
		this.emit('stateChange', this.state);
		this.startTimer();
	}

	stop() {
		this.state = 'idle';
		this.stopTimer();
		this.emit('stateChange', this.state);
	}

	setRemaining(sec: number) {
		const newRemaining = Math.max(0, sec);
		if (newRemaining !== this.getRemaining()) {
			if (this.state === 'paused') {
				this.pausedRemaining = newRemaining;
			} else if (this.state === 'running') {
				this.endTime = Date.now() + newRemaining * 1000;
			}
			this.emit('tick', newRemaining);
		}
	}

	destroy() {
		this.stopTimer();
		this.listeners.clear();
	}

	private startTimer() {
		this.stopTimer();
		this.timer = setInterval(() => this.tick(), 100);
	}

	private stopTimer() {
		if (this.timer) {
			clearInterval(this.timer);
			this.timer = null;
		}
	}

	private tick() {
		if (this.state !== 'running') return;
		const remaining = this.getRemaining();
		if (remaining > 0) {
			this.emit('tick', remaining);
		} else {
			this.stopTimer();
			this.state = 'idle';
			this.emit('complete');
			this.emit('stateChange', this.state);
		}
	}
}
