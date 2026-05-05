const CACHE = 'tomatogether-v2';

self.addEventListener('install', () => {
	self.skipWaiting();
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		caches.keys().then((keys) =>
			Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))
		)
	);
	self.clients.claim();
});

self.addEventListener('fetch', (event) => {
	if (event.request.method !== 'GET') return;
	const url = new URL(event.request.url);
	if (url.pathname.startsWith('/api/')) return;

	event.respondWith(
		caches.match(event.request).then((cached) => {
			const fetched = fetch(event.request).then((response) => {
				if (response.ok && (url.pathname.startsWith('/_app/') || url.pathname === '/' || url.pathname.endsWith('.png') || url.pathname.endsWith('.svg'))) {
					const clone = response.clone();
					caches.open(CACHE).then((cache) => cache.put(event.request, clone));
				}
				return response;
			}).catch(() => cached);
			return cached || fetched;
		})
	);
});
