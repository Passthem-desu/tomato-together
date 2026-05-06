// Pure display-layer countdown — server is the single source of truth.
// syncFromServer(remainingSec) is the only way to update the timer.
// Uses absolute timestamps (endTime) for accurate background-tab timing.

type EventHandler = (...args: any[]) => void;

export class PomodoroCountdown {
	private endTime = 0;
	private remaining = 0;
	private timer: ReturnType<typeof setInterval> | null = null;
	private listeners = new Map<string, Set<EventHandler>>();

	getRemaining(): number {
		if (this.endTime === 0) return this.remaining;
		return Math.max(0, Math.ceil((this.endTime - Date.now()) / 1000));
	}

	on(event: string, fn: EventHandler): () => void {
		if (!this.listeners.has(event)) this.listeners.set(event, new Set());
		this.listeners.get(event)!.add(fn);
		return () => this.listeners.get(event)?.delete(fn);
	}

	// Single entry point — server tells us the remaining time.
	// If >0: start/continue decrement timer. If 0: stop and emit 'complete'.
	syncFromServer(remainingSec: number) {
		const newRemaining = Math.max(0, remainingSec);
		if (newRemaining <= 0) {
			this.endTime = 0;
			this.remaining = 0;
			this.halt();
			this.emit('complete');
			return;
		}
		this.endTime = Date.now() + newRemaining * 1000;
		this.remaining = newRemaining;
		this.startDecrement();
		this.emit('tick', this.getRemaining());
	}

	// Stop the decrement timer without changing remaining (for paused state)
	halt(): void {
		if (this.timer) {
			clearInterval(this.timer);
			this.timer = null;
		}
	}

	destroy() {
		this.halt();
		this.listeners.clear();
	}

	private startDecrement() {
		if (this.timer) return;
		this.timer = setInterval(() => this.tick(), 1000);
	}

	private tick() {
		const r = this.getRemaining();
		if (r > 0) {
			this.emit('tick', r);
		} else {
			this.halt();
			this.endTime = 0;
			this.emit('complete');
		}
	}

	private emit(event: string, ...args: any[]) {
		this.listeners.get(event)?.forEach((fn) => fn(...args));
	}
}
