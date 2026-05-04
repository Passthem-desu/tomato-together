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
		refreshRoomUsers,
		logout,
		connectSSE,
		sseConnected,
	} from '$lib/store';
	import { api } from '$lib/api';
	import { locale, t } from '$lib/i18n';
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
	let showSettings = $state(false);
	let notifyEnabled = $state(loadNotifyPref());
	let soundVersion = $state(0);

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

	// When idle, force displayTime BEFORE render (prevent tick artifacts)
	$effect.pre(() => {
		if ($pomodoroStatus.phase === 'idle') {
			displayTime = plannedMinutes * 60;
		}
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
			if (phase === 'focusing' || phase === 'following') {
				let restSec = restMinutes * 60;
				try {
					await endPomodoro(false, sessionIndex);
					restSec =
						$pomodoroStatus.remaining_seconds ||
						$pomodoroStatus.rest_duration ||
						restSec;
				} catch {
					/* offline */
				}
				sound.play('focus_end');
				notifyPomodoroEnd();
				countdown.start(restSec);
			} else if (phase === 'rest') {
				sound.play('rest_end');
				notifyRestEnd();
				if (sessionIndex < totalSessions) {
					sessionIndex++;
					try {
						await startNextFocus();
					} catch {
						countdown.start(plannedMinutes * 60);
					}
				} else {
					sessionIndex = 0;
					pomodoroStatus.set({ phase: 'idle' });
					displayTime = plannedMinutes * 60;
					sound.play('all_done');
					notifyAllDone();
				}
			}
		})
	);

	async function startNextFocus() {
		await startPomodoro({
			planned_duration: plannedMinutes * 60,
			rest_duration: restMinutes * 60,
			long_break_duration: longBreakMinutes * 60,
			sessions_before_long_break: sessionsBeforeLong,
			session_index: sessionIndex,
		});
		countdown.start(plannedMinutes * 60);
	}

	// ── Drift correction ──
	$effect(() => {
		const sec = $pomodoroStatus.remaining_seconds;
		const phase = $pomodoroStatus.phase;
		if (sec !== undefined && countdown.getState() === 'running' && phase !== 'rest') {
			if (Math.abs(countdown.getRemaining() - sec) > 3) countdown.setRemaining(sec);
		}
	});

	onMount(() => {
		if (!$currentMember) {
			goto('/');
			return;
		}
		refreshRoomUsers();
		connectSSE();
		restorePomodoroState();

		const handleVisibility = () => {
			if (document.visibilityState === 'visible' && $pomodoroStatus.phase !== 'idle')
				restorePomodoroState();
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
		sessionIndex = 1;
		await startNextFocus();
	}

	async function handlePause() {
		sound.play('focus_pause');
		const resp = await api.pausePomodoro();
		if (resp.data) {
			pomodoroStatus.set(resp.data);
			countdown.pause();
			countdown.setRemaining(resp.data.remaining_seconds ?? displayTime);
		}
	}

	async function handleResume() {
		sound.play('focus_resume');
		const resp = await api.resumePomodoro();
		if (resp.data) {
			pomodoroStatus.set(resp.data);
			countdown.setRemaining(resp.data.remaining_seconds ?? displayTime);
			countdown.resume();
		}
	}

	async function handleSkip() {
		countdown.stop();
		sound.play('rest_end');
		notifyRestEnd();
		await api.skipRest();
		if (sessionIndex < totalSessions) {
			sessionIndex++;
			startNextFocus();
		} else {
			sessionIndex = 0;
			pomodoroStatus.set({ phase: 'idle' });
			displayTime = plannedMinutes * 60;
			sound.play('all_done');
			notifyAllDone();
		}
	}

	async function handleStop() {
		countdown.stop();
		sound.play('focus_end');
		await endPomodoro(true);
		countdown.stop();
		sessionIndex = 0;
		displayTime = plannedMinutes * 60;
	}

	async function handleEnd() {
		sound.play('focus_end');
		await endPomodoro(false, sessionIndex);
		const restSec =
			$pomodoroStatus.remaining_seconds || $pomodoroStatus.rest_duration || restMinutes * 60;
		countdown.stop();
		countdown.start(restSec);
		displayTime = restSec;
		notifyPomodoroEnd();
	}

	async function handleLogout() {
		if (confirm(t('confirm_logout', $locale))) {
			logout();
			goto('/');
		}
	}

	async function restorePomodoroState() {
		try {
			const resp = await api.getPomodoroStatus();
			if (resp.data) {
				pomodoroStatus.set(resp.data);
				if (resp.data.remaining_seconds !== undefined) {
					if (resp.data.phase === 'paused') {
						countdown.setRemaining(resp.data.remaining_seconds);
					} else if (resp.data.phase !== 'idle') {
						countdown.start(resp.data.remaining_seconds);
					}
				}
			}
		} catch {
			/* ignore */
		}
	}

	// ── Notifications ──
	function requestNotificationPermission() {
		if ('Notification' in window && Notification.permission === 'default')
			Notification.requestPermission();
	}

	const NOTIFY_KEY = 'tomatogether_notify';
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
					onclose={() => (showSettings = false)}
					onplannedMinutesChange={(v) => (plannedMinutes = v)}
					onrestMinutesChange={(v) => (restMinutes = v)}
					onlongBreakMinutesChange={(v) => (longBreakMinutes = v)}
					ontotalSessionsChange={(v) => (totalSessions = v)}
					onsessionsBeforeLongChange={(v) => (sessionsBeforeLong = v)}
					onnotifyEnabledChange={handleNotifyChange}
					onsoundschanged={handleSoundsChanged}
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
			<UserList users={$roomUsers} currentMemberId={$currentMember?.id || ''} />
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
