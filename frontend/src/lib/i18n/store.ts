// Locale store for TomatoTogether
import { writable } from 'svelte/store';
import { getStoredLocale, setStoredLocale, type Locale } from './locales';

// Create the locale store with stored value
function createLocaleStore() {
	const { subscribe, set } = writable<Locale>(getStoredLocale());

	return {
		subscribe,
		set: (locale: Locale) => {
			setStoredLocale(locale);
			set(locale);
		},
	};
}

export const locale = createLocaleStore();
