<script lang="ts">
	import { locale, t } from '$lib/i18n';

	interface Props {
		displayTime: number;
		phase: string;
		leaderUsername?: string;
		isLongBreak?: boolean;
		sessionIndex: number;
		totalSessions: number;
		isLoading: boolean;
		error: string | null;
		onstart: () => void;
		onpause: () => void;
		onresume: () => void;
		onskip: () => void;
		onstop: () => void;
		onend: () => void;
		onunfollow: () => void;
		onsettings: () => void;
	}

	let {
		displayTime,
		phase,
		leaderUsername = undefined,
		isLongBreak = false,
		sessionIndex,
		totalSessions,
		isLoading,
		error,
		onstart,
		onpause,
		onresume,
		onskip,
		onstop,
		onend,
		onunfollow,
		onsettings,
	}: Props = $props();

	function formatTime(s: number) {
		return `${Math.floor(s / 60)
			.toString()
			.padStart(2, '0')}:${(s % 60).toString().padStart(2, '0')}`;
	}

	function phaseLabel(): string {
		if (phase === 'focusing') return t('focusing', $locale);
		if (phase === 'paused') return t('paused', $locale);
		if (phase === 'following') return `${t('following', $locale)} ${leaderUsername}`;
		if (phase === 'rest')
			return isLongBreak ? t('rest_long', $locale) : t('rest_short', $locale);
		return t('idle', $locale);
	}
</script>

<div class="card timer-card">
	{#if phase !== 'idle' && sessionIndex > 0}
		<p class="session-progress">{sessionIndex} / {totalSessions}</p>
	{/if}

	<div class="timer-display">
		<span class="time">{formatTime(displayTime)}</span>
		<span
			class="status-label"
			class:focusing={phase === 'focusing' || phase === 'following'}
			class:resting={phase === 'rest'}
			class:paused={phase === 'paused'}
		>
			{phaseLabel()}
		</span>
	</div>

	{#if phase === 'idle'}
		<button class="btn-primary btn-lg btn-full" onclick={onstart} disabled={isLoading}>
			{t('start_pomodoro', $locale)}
		</button>
		<button class="btn-secondary btn-lg btn-full" onclick={onsettings} style="margin-top:0.5rem"
			>{t('settings', $locale)}</button
		>
	{:else if phase === 'paused'}
		<div class="timer-actions">
			<button class="btn-primary" onclick={onresume}>{t('continue', $locale)}</button>
			<button class="btn-ghost" onclick={onstop}>{t('stop', $locale)}</button>
		</div>
	{:else if phase === 'rest'}
		<div class="timer-actions">
			<button class="btn-primary" onclick={onskip}>{t('skip', $locale)}</button>
		</div>
	{:else}
		<div class="timer-actions">
			<button class="btn-secondary" onclick={onpause}>{t('pause', $locale)}</button>
			<button class="btn-secondary" onclick={onend}>{t('end_pomodoro', $locale)}</button>
			<button class="btn-ghost" onclick={onstop}>{t('stop', $locale)}</button>
			{#if phase === 'following'}
				<button class="btn-ghost" onclick={onunfollow}>{t('unfollow', $locale)}</button>
			{/if}
		</div>
	{/if}

	{#if error}<p class="form-error">{error}</p>{/if}
</div>

<style>
	.timer-card {
		text-align: center;
		padding: 2rem;
	}
	.session-progress {
		font-size: var(--text-sm);
		color: var(--color-fg-muted);
		margin-bottom: 0.5rem;
	}
	.timer-display {
		margin-bottom: 1.5rem;
	}
	.time {
		font-size: 4.5rem;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
		letter-spacing: -0.02em;
		display: block;
		margin-bottom: 0.5rem;
		background: linear-gradient(135deg, var(--color-fg-0), var(--color-brand));
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}
	.status-label {
		color: var(--color-fg-muted);
		font-size: var(--text-lg);
	}
	.status-label.focusing {
		color: var(--color-brand);
	}
	.status-label.resting {
		color: var(--color-success);
	}
	.status-label.paused {
		color: var(--color-warning, #f59e0b);
	}
	.timer-actions {
		display: flex;
		gap: 0.75rem;
		justify-content: center;
		flex-wrap: wrap;
	}
	.form-error {
		margin-top: 1rem;
		padding: 0.75rem;
		background: var(--color-error-subtle);
		color: var(--color-error);
		border-radius: var(--radius-md);
		font-size: var(--text-sm);
	}
	@media (max-width: 640px) {
		.time {
			font-size: 3.5rem;
		}
		.timer-actions {
			flex-direction: column;
		}
	}
</style>
