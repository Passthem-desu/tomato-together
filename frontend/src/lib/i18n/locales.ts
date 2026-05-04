// Locale configuration
export type Locale = 'zh-hans' | 'zh-hant' | 'en' | 'ja';

export interface LocaleOption {
	code: Locale;
	name: string;
}

export const locales: LocaleOption[] = [
	{ code: 'zh-hans', name: '简体中文' },
	{ code: 'zh-hant', name: '繁體中文' },
	{ code: 'en', name: 'English' },
	{ code: 'ja', name: '日本語' },
];

// Get stored locale or default
export function getStoredLocale(): Locale {
	if (typeof window !== 'undefined') {
		const stored = localStorage.getItem('locale');
		if (stored && locales.some((l) => l.code === stored)) {
			return stored as Locale;
		}
	}
	return 'zh-hans';
}

// Store locale
export function setStoredLocale(locale: Locale): void {
	if (typeof window !== 'undefined') {
		localStorage.setItem('locale', locale);
	}
}
