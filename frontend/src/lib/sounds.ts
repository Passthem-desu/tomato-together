// Sound manager — built-in generators + custom URL/data sounds.
//
// Built-in sounds are Web Audio generators.
// Custom sounds can be added from URLs or uploaded as data URIs (stored in localStorage).

export type SoundEvent =
	| 'focus_start'
	| 'focus_end'
	| 'focus_pause'
	| 'focus_resume'
	| 'rest_end'
	| 'all_done';

// ── Sound definition ──

export interface SoundDef {
	key: string;
	label: string;
	source: 'builtin' | 'url' | 'data';
	/** For builtin: generator function key */
	generatorKey?: string;
	/** For url/data: the src */
	src?: string;
}

// ── Built-in sound generators ──

function beep(ctx: AudioContext, freq: number, duration: number, startDelay = 0, vol = 0.12) {
	const osc = ctx.createOscillator();
	const gain = ctx.createGain();
	osc.type = 'sine';
	osc.frequency.value = freq;
	gain.gain.setValueAtTime(vol, ctx.currentTime + startDelay);
	gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + startDelay + duration);
	osc.connect(gain);
	gain.connect(ctx.destination);
	osc.start(ctx.currentTime + startDelay);
	osc.stop(ctx.currentTime + startDelay + duration);
}

function genBeep(ctx: AudioContext) {
	beep(ctx, 800, 0.15);
}
function genDoubleBeep(ctx: AudioContext) {
	beep(ctx, 660, 0.12, 0);
	beep(ctx, 880, 0.12, 0.15);
}
function genChimeUp(ctx: AudioContext) {
	beep(ctx, 523, 0.2, 0);
	beep(ctx, 659, 0.2, 0.15);
	beep(ctx, 784, 0.3, 0.3);
}
function genChimeDown(ctx: AudioContext) {
	beep(ctx, 784, 0.2, 0);
	beep(ctx, 659, 0.2, 0.15);
	beep(ctx, 523, 0.3, 0.3);
}

/** 噔噔咚 — F# E C (descending) */
function genDengDengDong(ctx: AudioContext) {
	beep(ctx, 370, 0.25, 0);
	beep(ctx, 330, 0.25, 0.15);
	beep(ctx, 262, 0.35, 0.3);
}

/** 柴又猫 — G A# C D D# A# (arpeggio, eighth-note spacing) */
function genShibamata(ctx: AudioContext) {
	const notes = [392, 466, 523, 587, 622, 932];
	notes.forEach((f, i) => beep(ctx, f, 0.2, i * 0.15));
}

const GENERATORS: Record<string, (ctx: AudioContext) => void> = {
	beep: genBeep,
	double_beep: genDoubleBeep,
	chime_up: genChimeUp,
	chime_down: genChimeDown,
	dengdengdong: genDengDengDong,
	shibamata: genShibamata,
};

const BUILTIN_SOUNDS: SoundDef[] = [
	{ key: 'beep', label: '短蜂鸣', source: 'builtin', generatorKey: 'beep' },
	{ key: 'double_beep', label: '双蜂鸣', source: 'builtin', generatorKey: 'double_beep' },
	{ key: 'chime_up', label: '上行铃音', source: 'builtin', generatorKey: 'chime_up' },
	{ key: 'chime_down', label: '下行铃音', source: 'builtin', generatorKey: 'chime_down' },
	{ key: 'dengdengdong', label: '噔噔咚', source: 'builtin', generatorKey: 'dengdengdong' },
	{ key: 'shibamata', label: '柴又猫', source: 'builtin', generatorKey: 'shibamata' },
	{ key: 'none', label: '静音', source: 'builtin' },
];

// ── Persistent custom sounds ──

const CUSTOM_SOUNDS_KEY = 'tomatogether_custom_sounds';

function loadCustomSounds(): SoundDef[] {
	try {
		const raw = localStorage.getItem(CUSTOM_SOUNDS_KEY);
		return raw ? JSON.parse(raw) : [];
	} catch {
		return [];
	}
}

function saveCustomSounds(sounds: SoundDef[]) {
	localStorage.setItem(CUSTOM_SOUNDS_KEY, JSON.stringify(sounds));
}

// ── Manager ──

const PREFS_KEY = 'tomatogether_sounds';

type SoundPrefs = Record<SoundEvent, string>;

const DEFAULT_PREFS: SoundPrefs = {
	focus_start: 'chime_up',
	focus_end: 'chime_down',
	focus_pause: 'beep',
	focus_resume: 'beep',
	rest_end: 'double_beep',
	all_done: 'chime_up',
};

function loadPrefs(): SoundPrefs {
	try {
		const raw = localStorage.getItem(PREFS_KEY);
		if (raw) return { ...DEFAULT_PREFS, ...JSON.parse(raw) };
	} catch {
		/* ignore */
	}
	return { ...DEFAULT_PREFS };
}

function savePrefs(prefs: SoundPrefs) {
	localStorage.setItem(PREFS_KEY, JSON.stringify(prefs));
}

export class SoundManager {
	private prefs: SoundPrefs;
	private customSounds: SoundDef[];

	constructor() {
		this.prefs = loadPrefs();
		this.customSounds = loadCustomSounds();
	}

	/** Play the sound configured for an event. */
	play(event: SoundEvent) {
		const def = this.findSound(this.prefs[event]);
		if (!def) return;
		this.playDef(def);
	}

	/** Preview a sound by key. */
	preview(key: string) {
		const def = this.findSound(key);
		if (def) this.playDef(def);
	}

	/** Set which sound to use for an event. */
	setSound(event: SoundEvent, key: string) {
		this.prefs[event] = key;
		savePrefs(this.prefs);
	}

	/** Get current sound key for an event. */
	getSound(event: SoundEvent): string {
		return this.prefs[event];
	}

	/** All available sounds (built-in + custom). */
	allSounds(): SoundDef[] {
		return [...this.customSounds, ...BUILTIN_SOUNDS];
	}

	/** Add a custom sound from URL. */
	addUrlSound(label: string, url: string): SoundDef | null {
		if (this.customSounds.find((s) => s.src === url)) return null;
		const key = 'custom_' + Date.now();
		const def: SoundDef = { key, label, source: 'url', src: url };
		this.customSounds.push(def);
		saveCustomSounds(this.customSounds);
		return def;
	}

	/** Add a custom sound from data URI (file upload). */
	addDataSound(label: string, dataUri: string): SoundDef {
		const key = 'custom_' + Date.now();
		const def: SoundDef = { key, label, source: 'data', src: dataUri };
		this.customSounds.push(def);
		saveCustomSounds(this.customSounds);
		return def;
	}

	/** Remove a custom sound. */
	removeSound(key: string) {
		this.customSounds = this.customSounds.filter((s) => s.key !== key);
		saveCustomSounds(this.customSounds);
		// Reset prefs pointing to removed sound
		let changed = false;
		for (const ev of Object.keys(this.prefs) as SoundEvent[]) {
			if (this.prefs[ev] === key) {
				this.prefs[ev] = DEFAULT_PREFS[ev];
				changed = true;
			}
		}
		if (changed) savePrefs(this.prefs);
	}

	// ── private ──

	private findSound(key: string): SoundDef | undefined {
		return (
			BUILTIN_SOUNDS.find((s) => s.key === key) ||
			this.customSounds.find((s) => s.key === key)
		);
	}

	private playDef(def: SoundDef) {
		if (def.source === 'builtin' && def.generatorKey) {
			GENERATORS[def.generatorKey]?.(new AudioContext());
		} else if (def.src) {
			const a = new Audio(def.src);
			a.volume = 0.3;
			a.play().catch(() => {});
		}
	}
}
