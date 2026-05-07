<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		currentMember,
		currentRoom,
		roomUsers,
		pomodoroStatus,
		isLoading,
		error,
		lastAnnouncement,
		startPomodoro,
		endPomodoro,
		unfollowPomodoro,
		pausePomodoro,
		resumePomodoro,
		skipRest,
		refreshRoomUsers,
		logout,
		connectSSE,
		sseConnected,
	} from '$lib/store';
	import { api } from '$lib/api';
	import { locale, t } from '$lib/i18n';
	import { taskStore } from '$lib/taskStore';
	import { PomodoroCountdown } from '$lib/countdown';
	import { SoundManager } from '$lib/sounds';
	import RoomHeader from '$lib/components/RoomHeader.svelte';
	import TimerCard from '$lib/components/TimerCard.svelte';
	import SettingsPanel from '$lib/components/SettingsPanel.svelte';
	import UserList from '$lib/components/UserList.svelte';
	import WipPanel from '$lib/components/WipPanel.svelte';
	import AnnouncementPanel from '$lib/components/AnnouncementPanel.svelte';

	const sound = new SoundManager();
	const countdown = new PomodoroCountdown();

	let displayTime = $state(25 * 60);
	let plannedMinutes = $state(25);
	let restMinutes = $state(5);
	let longBreakMinutes = $state(15);
	let totalSessions = $state(4);
	let sessionsBeforeLong = $state(4);
	let sessionIndex = $state(0);
	let skippingRest = $state(false); // suppress idle flash during skip→focus transition
	let showSettings = $state(false);
	let notifyEnabled = $state(loadNotifyPref());
	let soundVersion = $state(0);
	let lastTaskSync = 0;
	const TASK_SYNC_COOLDOWN = 30_000;

	function handleNotifyChange(v: boolean) {
		notifyEnabled = v;
		saveNotifyPref(v);
		if (v) requestNotificationPermission();
	}

	function handleSoundsChanged() {
		soundVersion++;
	}

	let allSounds = $derived.by(() => {
		void soundVersion;
		return sound.allSounds().map((s) => ({
			...s,
			label: s.source === 'builtin' ? t(`sound_${s.key}`, $locale) : s.label,
		}));
	});

	let estimatedFinish = $derived(computeEstimate());

	function computeEstimate(): string {
		if (totalSessions <= 0 || plannedMinutes <= 0) return '';
		const f = totalSessions * plannedMinutes * 60;
		const longs = Math.floor((totalSessions - 1) / sessionsBeforeLong);
		const shorts = totalSessions - 1 - longs;
		const totalSec = f + longs * longBreakMinutes * 60 + shorts * restMinutes * 60;
		const finish = new Date(Date.now() + totalSec * 1000);
		return finish.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	// Single source of truth: sync countdown from pomodoroStatus
	$effect(() => {
		const s = $pomodoroStatus;

		// Always stop current timer on state change
		countdown.halt();

		// Sync state from SSE (only when not idle — idle events may have undefined values)
		if (s.phase !== 'idle') {
			if (s.sessions_completed !== undefined) {
				sessionIndex = s.sessions_completed;
			}
			if (s.total_sessions !== undefined && s.total_sessions > 0) {
				totalSessions = s.total_sessions;
			}
			if (s.planned_duration !== undefined && s.planned_duration > 0) {
				plannedMinutes = Math.floor(s.planned_duration / 60);
			}
		}

		// Start timer for ticking states
		if (s.phase === 'focusing' || s.phase === 'following' || s.phase === 'rest') {
			if (s.remaining_seconds && s.remaining_seconds > 0) {
				countdown.syncFromServer(s.remaining_seconds);
			}
		}

		// Set displayTime for non-ticking states
		if (s.phase === 'paused' && s.remaining_seconds !== undefined) {
			displayTime = s.remaining_seconds;
		} else if (s.phase === 'idle' && !skippingRest) {
			displayTime = plannedMinutes * 60;
		}
		// ticking states: displayTime set by countdown.on('tick') below
	});

	// ── Countdown events ──
	const unsubs: (() => void)[] = [];

	unsubs.push(
		countdown.on('tick', (remaining: number) => {
			displayTime = remaining;
		})
	);

	unsubs.push(
		countdown.on('complete', async () => {
			const phase = $pomodoroStatus.phase;
			try {
				if (phase === 'focusing' || phase === 'following') {
					await endPomodoro(false);
					sound.play('focus_end');
					notifyPomodoroEnd();
				} else if (phase === 'rest') {
					sound.play('rest_end');
					notifyRestEnd();
					if (sessionIndex < totalSessions) {
						await startNextFocus();
					} else {
						sound.play('all_done');
						notifyAllDone();
					}
				}
				// idle: nothing (SSE will handle state)
			} catch {
				// SSE will correct state on next tick
			}
		})
	);

	async function startNextFocus(sessionIdx?: number) {
		const idx = sessionIdx ?? sessionIndex + 1;
		await startPomodoro({
			planned_duration: plannedMinutes * 60,
			rest_duration: restMinutes * 60,
			long_break_duration: longBreakMinutes * 60,
			sessions_before_long_break: sessionsBeforeLong,
			session_index: idx,
			total_sessions: totalSessions,
		});
	}

	// ── Persist settings to localStorage ──
	$effect(() => {
		void plannedMinutes;
		void restMinutes;
		void longBreakMinutes;
		void totalSessions;
		void sessionsBeforeLong;
		scheduleSave();
	});

	onMount(() => {
		if (!$currentMember) {
			goto('/');
			return;
		}
		loadSettings();
		refreshRoomUsers();
		connectSSE();
		restorePomodoroState();

		const handleVisibility = () => {
			if (document.visibilityState === 'visible') {
				if ($pomodoroStatus.phase !== 'idle') restorePomodoroState();
				const now = Date.now();
				if (now - lastTaskSync > TASK_SYNC_COOLDOWN) {
					lastTaskSync = now;
					taskStore.syncWithServer($currentRoom?.name || '');
				}
			}
		};
		document.addEventListener('visibilitychange', handleVisibility);
		unsubs.push(() => document.removeEventListener('visibilitychange', handleVisibility));
	});

	onDestroy(() => {
		countdown.destroy();
		unsubs.forEach((fn) => fn());
	});

	// ── Actions ──
	async function handleStart() {
		sound.play('focus_start');
		await startPomodoro({
			planned_duration: plannedMinutes * 60,
			rest_duration: restMinutes * 60,
			long_break_duration: longBreakMinutes * 60,
			sessions_before_long_break: sessionsBeforeLong,
			session_index: 1,
			total_sessions: totalSessions,
		});
	}

	async function handlePause() {
		sound.play('focus_pause');
		await pausePomodoro();
	}

	async function handleResume() {
		sound.play('focus_resume');
		await resumePomodoro();
	}

	async function handleSkip() {
		sound.play('rest_end');
		notifyRestEnd();
		const oldIndex = sessionIndex;
		skippingRest = true;
		const ok = await skipRest();
		if (!ok) {
			skippingRest = false;
			return;
		}
		if (oldIndex < totalSessions) {
			await startNextFocus(oldIndex + 1);
		} else {
			sound.play('all_done');
			notifyAllDone();
		}
		skippingRest = false;
	}

	async function handleStop() {
		sound.play('focus_end');
		await endPomodoro(true);
	}

	async function handleEnd() {
		sound.play('focus_end');
		await endPomodoro(false, sessionIndex);
		notifyPomodoroEnd();
	}

	async function handleLogout() {
		if ($pomodoroStatus.phase !== 'idle') {
			if (!confirm(t('confirm_leave_active', $locale))) return;
		}
		if (confirm(t('confirm_logout', $locale))) {
			logout();
			goto('/');
		}
	}

	async function restorePomodoroState() {
		try {
			const resp = await api.getPomodoroStatus();
			if (resp.data) pomodoroStatus.set(resp.data);
		} catch {
			/* ignore */
		}
	}

	function formatMinutes(sec: number) {
		const m = Math.round(sec / 60);
		if (m < 60) return m + 'm';
		return Math.floor(m / 60) + 'h ' + (m % 60) + 'm';
	}

	// ── Notifications ──
	function requestNotificationPermission() {
		if ('Notification' in window && Notification.permission === 'default')
			Notification.requestPermission();
	}

	const NOTIFY_KEY = 'tomatogether_notify';
	const SETTINGS_KEY = 'tomatogether_settings';

	function loadNotifyPref(): boolean {
		try {
			return localStorage.getItem(NOTIFY_KEY) === 'true';
		} catch {
			return false;
		}
	}
	function saveNotifyPref(v: boolean) {
		localStorage.setItem(NOTIFY_KEY, String(v));
	}

	function loadSettings() {
		try {
			const raw = localStorage.getItem(SETTINGS_KEY);
			if (!raw) return;
			const s = JSON.parse(raw);
			if (typeof s.plannedMinutes === 'number') plannedMinutes = s.plannedMinutes;
			if (typeof s.restMinutes === 'number') restMinutes = s.restMinutes;
			if (typeof s.longBreakMinutes === 'number') longBreakMinutes = s.longBreakMinutes;
			if (typeof s.totalSessions === 'number') totalSessions = s.totalSessions;
			if (typeof s.sessionsBeforeLong === 'number') sessionsBeforeLong = s.sessionsBeforeLong;
		} catch {
			/* ignore corrupt data */
		}
	}

	let _saveTimer: ReturnType<typeof setTimeout> | null = null;
	function scheduleSave() {
		if (_saveTimer) clearTimeout(_saveTimer);
		_saveTimer = setTimeout(() => {
			try {
				localStorage.setItem(
					SETTINGS_KEY,
					JSON.stringify({
						plannedMinutes,
						restMinutes,
						longBreakMinutes,
						totalSessions,
						sessionsBeforeLong,
					})
				);
			} catch {
				/* ignore */
			}
		}, 500);
	}

	function notify(title: string, body: string) {
		if (!notifyEnabled) return;
		if ('Notification' in window && Notification.permission === 'granted')
			new Notification(title, { body, icon: '/favicon.png' });
	}
	function notifyPomodoroEnd() {
		notify(t('notify_focus_end_title', $locale), t('notify_focus_end_body', $locale));
	}
	function notifyRestEnd() {
		notify(t('notify_rest_end_title', $locale), t('notify_rest_end_body', $locale));
	}
	function notifyAllDone() {
		notify(t('notify_all_done_title', $locale), t('notify_all_done_title', $locale));
	}

	function onlineCount() {
		return $roomUsers.filter((u) => u.is_online).length;
	}
</script>

<svelte:head>
	<title>{t('room_title', $locale)} - TomatoTogether</title>
</svelte:head>

<main class="room">
	<RoomHeader
		roomName={$currentRoom?.name || ''}
		onlineCount={onlineCount()}
		sseConnected={$sseConnected}
		username={$currentMember?.username || ''}
		isOwner={$currentMember?.is_owner ?? false}
		onlogout={handleLogout}
	/>

	{#if $currentRoom?.is_readonly}
		<div class="readonly-banner">{t('room_readonly_banner', $locale)}</div>
	{/if}

	<div class="room-content">
		<section class="main-panel">
			{#if showSettings}
				<SettingsPanel
					{plannedMinutes}
					{restMinutes}
					{longBreakMinutes}
					{totalSessions}
					{sessionsBeforeLong}
					{notifyEnabled}
					{estimatedFinish}
					{sound}
					{allSounds}
					isOwner={$currentMember?.is_owner ?? false}
					isReadonly={$currentRoom?.is_readonly ?? false}
					hasRoomPassword={$currentRoom?.has_password ?? false}
					isPersistent={$currentMember?.is_persistent ?? false}
					onclose={() => (showSettings = false)}
					onplannedMinutesChange={(v) => (plannedMinutes = v)}
					onrestMinutesChange={(v) => (restMinutes = v)}
					onlongBreakMinutesChange={(v) => (longBreakMinutes = v)}
					ontotalSessionsChange={(v) => (totalSessions = v)}
					onsessionsBeforeLongChange={(v) => (sessionsBeforeLong = v)}
					onnotifyEnabledChange={handleNotifyChange}
					onsoundschanged={handleSoundsChanged}
					onreset={() => refreshRoomUsers()}
				/>
			{:else}
				<TimerCard
					{displayTime}
					phase={$pomodoroStatus.phase}
					leaderUsername={$pomodoroStatus.leader_username}
					isLongBreak={$pomodoroStatus.is_long_break}
					{sessionIndex}
					{totalSessions}
					isLoading={$isLoading}
					error={$error}
					onstart={handleStart}
					onpause={handlePause}
					onresume={handleResume}
					onskip={handleSkip}
					onstop={handleStop}
					onend={handleEnd}
					onunfollow={unfollowPomodoro}
					onsettings={() => (showSettings = true)}
				/>
			{/if}

			<div class="wip-section">
				<WipPanel />
			</div>
		</section>

		<section class="users-panel">
			<UserList
				users={$roomUsers}
				currentMemberId={$currentMember?.id || ''}
				isOwner={$currentMember?.is_owner ?? false}
			/>
			<div class="ann-section">
				<AnnouncementPanel
					isOwner={$currentMember?.is_owner ?? false}
					newAnnouncement={$lastAnnouncement}
				/>
			</div>
		</section>
	</div>
</main>

<style>
	.room {
		min-height: 100dvh;
		padding: 1.5rem 1.5rem 40vh;
		background: var(--color-bg-0);
	}
	.wip-section {
		margin-top: 1rem;
	}
	.ann-section {
		margin-top: 1rem;
	}
	.room-content {
		display: flex;
		gap: 1.5rem;
		align-items: flex-start;
	}
	.main-panel {
		flex: 1;
		min-width: 0;
	}
	.users-panel {
		width: 320px;
		flex-shrink: 0;
	}

	@media (max-width: 900px) {
		.room-content {
			flex-direction: column;
		}
		.main-panel {
			width: 100%;
		}
		.users-panel {
			width: 100%;
		}
	}
	@media (max-width: 640px) {
		.room {
			padding: 1rem;
		}
	}
</style>
