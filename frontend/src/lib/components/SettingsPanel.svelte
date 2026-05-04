<script lang="ts">
	import { locale, t } from '$lib/i18n';
	import { api } from '$lib/api';
	import { SoundManager, type SoundEvent, type SoundDef } from '$lib/sounds';

	interface Props {
		plannedMinutes: number;
		restMinutes: number;
		longBreakMinutes: number;
		totalSessions: number;
		sessionsBeforeLong: number;
		notifyEnabled: boolean;
		estimatedFinish: string;
		sound: SoundManager;
		allSounds: SoundDef[];
		isOwner: boolean;
		isReadonly: boolean;
		hasRoomPassword: boolean;
		isPersistent: boolean;
		onclose: () => void;
		onplannedMinutesChange: (v: number) => void;
		onrestMinutesChange: (v: number) => void;
		onlongBreakMinutesChange: (v: number) => void;
		ontotalSessionsChange: (v: number) => void;
		onsessionsBeforeLongChange: (v: number) => void;
		onnotifyEnabledChange: (v: boolean) => void;
		onsoundschanged?: () => void;
		onreset?: () => void;
	}

	let {
		plannedMinutes,
		restMinutes,
		longBreakMinutes,
		totalSessions,
		sessionsBeforeLong,
		notifyEnabled,
		estimatedFinish,
		sound,
		allSounds,
		isOwner,
		isReadonly,
		hasRoomPassword,
		isPersistent,
		onclose,
		onplannedMinutesChange,
		onrestMinutesChange,
		onlongBreakMinutesChange,
		ontotalSessionsChange,
		onsessionsBeforeLongChange,
		onnotifyEnabledChange,
		onsoundschanged,
		onreset = () => {},
	}: Props = $props();

	const allSoundEvents: SoundEvent[] = [
		'focus_start',
		'focus_end',
		'focus_pause',
		'focus_resume',
		'rest_end',
		'all_done',
	];

	const soundEventLabel = (ev: SoundEvent) => {
		const map: Record<SoundEvent, string> = {
			focus_start: 'focus_start',
			focus_end: 'focus_end',
			focus_pause: 'focus_pause',
			focus_resume: 'focus_resume',
			rest_end: 'rest_end',
			all_done: 'all_done',
		};
		return t(map[ev], $locale);
	};

	let newSoundLabel = $state('');
	let newSoundUrl = $state('');
	let fileInput = $state<HTMLInputElement | null>(null);
	let showSoundDialog = $state(false);
	let resetting = $state(false);
	let roomPassword = $state('');
	let roomReadonly = $state<boolean>(isReadonly);
	let roomSettingsSaving = $state(false);
	let oldPassword = $state('');
	let newPassword = $state('');
	let newPasswordConfirm = $state('');
	let passwordChanging = $state(false);

	async function handleChangePassword() {
		if (newPassword !== newPasswordConfirm) return;
		passwordChanging = true;
		try {
			await api.changePassword({ old_password: oldPassword, new_password: newPassword });
			oldPassword = '';
			newPassword = '';
			newPasswordConfirm = '';
		} catch {
			/* ignore */
		}
		passwordChanging = false;
	}

	async function handleSaveRoomSettings() {
		roomSettingsSaving = true;
		try {
			await api.updateRoomSettings({
				room_password: roomPassword || undefined,
				is_readonly: roomReadonly,
			});
			roomPassword = '';
		} catch {
			/* ignore */
		}
		roomSettingsSaving = false;
	}

	async function handleReset() {
		if (!confirm(t('reset_confirm', $locale))) return;
		resetting = true;
		try {
			await api.resetMyStats();
			onreset();
		} catch {
			/* ignore */
		}
		resetting = false;
	}

	function addCustomSound() {
		const label = newSoundLabel.trim();
		const url = newSoundUrl.trim();
		if (!label) return;
		if (url) {
			sound.addUrlSound(label, url);
			onsoundschanged?.();
		}
		newSoundLabel = '';
		newSoundUrl = '';
		showSoundDialog = false;
	}

	function handleFileUpload(e: Event) {
		const file = (e.target as HTMLInputElement).files?.[0];
		if (!file) return;
		const reader = new FileReader();
		reader.onload = () => {
			sound.addDataSound(file.name.replace(/\.[^.]+$/, ''), reader.result as string);
			onsoundschanged?.();
		};
		reader.readAsDataURL(file);
	}
</script>

<div class="card settings-card">
	<div class="settings-header">
		<h2 class="settings-title">{t('settings', $locale)}</h2>
		<button class="btn-ghost btn-sm" onclick={onclose}>{t('close', $locale)}</button>
	</div>

	<div class="settings-grid">
		<div class="setting">
			<label>{t('focus_time', $locale)}</label>
			<input
				type="number"
				value={plannedMinutes}
				oninput={(e) =>
					onplannedMinutesChange(Number((e.target as HTMLInputElement).value))}
				min="1"
				max="60"
			/>
		</div>
		<div class="setting">
			<label>{t('short_break', $locale)}</label>
			<input
				type="number"
				value={restMinutes}
				oninput={(e) => onrestMinutesChange(Number((e.target as HTMLInputElement).value))}
				min="1"
				max="30"
			/>
		</div>
		<div class="setting">
			<label>{t('long_break', $locale)}</label>
			<input
				type="number"
				value={longBreakMinutes}
				oninput={(e) =>
					onlongBreakMinutesChange(Number((e.target as HTMLInputElement).value))}
				min="1"
				max="60"
			/>
		</div>
		<div class="setting">
			<label>{t('total_sessions', $locale)}</label>
			<input
				type="number"
				value={totalSessions}
				oninput={(e) => ontotalSessionsChange(Number((e.target as HTMLInputElement).value))}
				min="1"
				max="20"
			/>
		</div>
		<div class="setting">
			<label>{t('long_break_after', $locale)}</label>
			<input
				type="number"
				value={sessionsBeforeLong}
				oninput={(e) =>
					onsessionsBeforeLongChange(Number((e.target as HTMLInputElement).value))}
				min="1"
				max="10"
			/>
		</div>
	</div>
	{#if estimatedFinish}
		<p class="estimate">{t('estimated_finish', $locale)} {estimatedFinish}</p>
	{/if}

	<div class="setting-row">
		<label class="toggle-label">
			<input
				type="checkbox"
				checked={notifyEnabled}
				onchange={(e) => onnotifyEnabledChange((e.target as HTMLInputElement).checked)}
			/>
			<span>{t('notifications_enabled', $locale)}</span>
		</label>
	</div>

	<h3 class="settings-subtitle">{t('sounds', $locale)}</h3>
	<div class="sounds-grid">
		{#each allSoundEvents as ev}
			<div class="sound-row">
				<label>{soundEventLabel(ev)}</label>
				<div class="sound-row-right">
					<button
						class="btn-text btn-xs"
						onclick={() => sound.preview(sound.getSound(ev))}
						title={t('preview', $locale)}>▶</button
					>
					<select
						value={sound.getSound(ev)}
						onchange={(e) => {
							const v = (e.target as HTMLSelectElement).value;
							sound.setSound(ev, v);
							sound.preview(v);
						}}
					>
						{#each allSounds as s (s.key)}
							<option value={s.key} selected={sound.getSound(ev) === s.key}
								>{s.label}</option
							>
						{/each}
					</select>
				</div>
			</div>
		{/each}
	</div>

	<h3 class="settings-subtitle">{t('custom_sounds', $locale)}</h3>
	<p class="hint-text">{t('custom_sounds_hint', $locale)}</p>
	<div class="custom-actions">
		<button class="btn-secondary btn-sm" onclick={() => (showSoundDialog = true)}
			>{t('add_network_sound', $locale)}</button
		>
		<input
			type="file"
			accept="audio/*"
			onchange={handleFileUpload}
			style="display:none"
			bind:this={fileInput}
		/>
		<button class="btn-secondary btn-sm" onclick={() => fileInput?.click()}
			>{t('upload_sound', $locale)}</button
		>
	</div>
	{#if allSounds.filter((s) => s.source !== 'builtin').length > 0}
		<div class="custom-list-box">
			<ul class="custom-list">
				{#each allSounds.filter((s) => s.source !== 'builtin') as s (s.key)}
					<li>
						{s.label}
						<span>
							<button class="btn-text btn-xs" onclick={() => sound.preview(s.key)}
								>▶</button
							>
							<button
								class="btn-text btn-xs"
								onclick={() => {
									sound.removeSound(s.key);
									onsoundschanged?.();
								}}
							>
								×
							</button>
						</span>
					</li>
				{/each}
			</ul>
		</div>
	{/if}

	<div class="reset-section">
		<button class="btn-reset" onclick={handleReset}>{t('reset_stats', $locale)}</button>
	</div>

	{#if isOwner}
		<h3 class="settings-subtitle">{t('room_settings_title', $locale)}</h3>
		<div class="room-settings">
			<div class="setting room-setting">
				<label>{t('room_password_label', $locale)}</label>
				<input
					type="password"
					placeholder={t('new_password_placeholder', $locale)}
					bind:value={roomPassword}
				/>
			</div>
			<div class="setting-row">
				<label class="toggle-label">
					<input
						type="checkbox"
						checked={roomReadonly}
						onchange={(e) => (roomReadonly = (e.target as HTMLInputElement).checked)}
					/>
					<span>{t('readonly_mode', $locale)}</span>
				</label>
			</div>
			<button
				class="btn-secondary btn-sm"
				onclick={handleSaveRoomSettings}
				disabled={roomSettingsSaving}
			>
				{t('save_settings', $locale)}
			</button>
		</div>
	{/if}

	{#if isPersistent}
		<h3 class="settings-subtitle">{t('change_password_title', $locale)}</h3>
		<div class="room-settings">
			<div class="setting room-setting">
				<label>{t('change_old_password', $locale)}</label>
				<input type="password" bind:value={oldPassword} />
			</div>
			<div class="setting room-setting">
				<label>{t('change_new_password', $locale)}</label>
				<input type="password" bind:value={newPassword} />
			</div>
			<div class="setting room-setting">
				<label>{t('change_confirm_password', $locale)}</label>
				<input type="password" bind:value={newPasswordConfirm} />
			</div>
			<button
				class="btn-secondary btn-sm"
				onclick={handleChangePassword}
				disabled={passwordChanging ||
					!oldPassword ||
					!newPassword ||
					newPassword !== newPasswordConfirm}
			>
				{t('change_password_btn', $locale)}
			</button>
		</div>
	{/if}
</div>

{#if showSoundDialog}
	<div class="dialog-overlay" onclick={() => (showSoundDialog = false)}>
		<div class="dialog" onclick={(e) => e.stopPropagation()}>
			<h3>{t('add_network_sound', $locale)}</h3>
			<div class="form-group">
				<label>{t('sound_label_placeholder', $locale)}</label>
				<input type="text" bind:value={newSoundLabel} />
			</div>
			<div class="form-group">
				<label>{t('sound_url_placeholder', $locale)}</label>
				<input type="text" bind:value={newSoundUrl} />
			</div>
			<div class="dialog-actions">
				<button
					class="btn-text"
					onclick={() => sound.preview(newSoundUrl)}
					disabled={!newSoundUrl}>{t('preview', $locale)}</button
				>
				<button class="btn-secondary" onclick={addCustomSound}>{t('add', $locale)}</button>
				<button class="btn-ghost" onclick={() => (showSoundDialog = false)}
					>{t('close', $locale)}</button
				>
			</div>
		</div>
	</div>
{/if}

<style>
	.settings-card {
		padding: 1.5rem;
	}
	.settings-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1.25rem;
	}
	.settings-title {
		font-size: var(--text-lg);
		font-weight: 600;
	}
	.settings-grid {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
	}
	.settings-grid .setting {
		flex: 0 0 auto;
	}
	.setting {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}
	.setting label {
		font-size: var(--text-xs);
		color: var(--color-fg-muted);
		font-weight: 500;
	}
	.setting input {
		width: 70px;
		text-align: center;
		padding: 0.5rem;
	}
	.estimate {
		font-size: var(--text-sm);
		color: var(--color-fg-muted);
		margin-bottom: 1rem;
		margin-top: 0.5rem;
	}
	.setting-row {
		margin-bottom: 1rem;
	}
	.toggle-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: var(--text-sm);
		cursor: pointer;
		user-select: none;
	}
	.toggle-label input[type='checkbox'] {
		width: 1rem;
		height: 1rem;
		accent-color: var(--color-brand);
		cursor: pointer;
	}
	.settings-subtitle {
		font-size: var(--text-base);
		font-weight: 600;
		margin: 1.5rem 0 0.5rem;
	}
	.hint-text {
		font-size: var(--text-xs);
		color: var(--color-fg-muted);
		margin-bottom: 0.5rem;
	}
	.custom-actions {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 0.5rem;
	}
	.custom-list-box {
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: 0.5rem;
		margin-top: 0.5rem;
	}
	.custom-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.custom-list li {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.25rem 0;
		font-size: var(--text-sm);
	}
	.sound-row {
		display: flex;
		align-items: baseline;
		padding: 0.375rem 0;
		gap: 0.5rem;
	}
	.sound-row-right {
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}
	.sound-row label {
		font-size: var(--text-sm);
		white-space: nowrap;
	}
	.sound-row select {
		font-size: var(--text-sm);
		padding: 0.25rem 0.5rem;
		min-width: 110px;
	}
	.sounds-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0 2rem;
	}
	.dialog-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.4);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
	}
	.dialog {
		background: var(--color-bg-1);
		border-radius: var(--radius-lg);
		padding: 1.5rem;
		width: 90%;
		max-width: 400px;
		box-shadow: var(--shadow-lg);
	}
	.dialog h3 {
		margin-bottom: 1rem;
	}
	.dialog .form-group {
		margin-bottom: 0.75rem;
	}
	.dialog .form-group label {
		display: block;
		font-size: var(--text-sm);
		margin-bottom: 0.25rem;
	}
	.dialog .form-group input {
		width: 100%;
		padding: 0.5rem;
	}
	.dialog-actions {
		display: flex;
		gap: 0.5rem;
		justify-content: flex-end;
		margin-top: 1rem;
	}

	.reset-section {
		margin-top: 1.25rem;
		padding-top: 1rem;
		border-top: 1px solid var(--color-border);
	}
	.room-settings {
		margin-top: 0.5rem;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}
	.room-setting input {
		width: 200px !important;
		text-align: left !important;
	}
	.btn-reset {
		padding: 0.4rem 0.9rem;
		font-size: 0.78rem;
		border: 1px solid var(--color-error);
		border-radius: var(--radius-md);
		background: transparent;
		color: var(--color-error);
		cursor: pointer;
		transition: all 0.15s;
	}
	.btn-reset:hover {
		background: var(--color-error-subtle);
	}
	@media (max-width: 640px) {
		.sounds-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
