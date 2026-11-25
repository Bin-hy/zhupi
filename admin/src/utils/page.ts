export function isHttpOrigin(value: string): boolean {
	return /^https?:\/\//i.test(value);
}

export function preferredSiteOrigin(origins: string[]): string {
	const valid = origins.filter(isHttpOrigin);
	if (valid.length === 0) return '';
	const isLoopback = (value: string): boolean => {
		try {
			const host = new URL(value).hostname.toLowerCase();
			return host === 'localhost' || host === '127.0.0.1' || host === '::1' || host === '[::1]';
		} catch {
			return false;
		}
	};
	return (
		valid.find((origin) => /^https:\/\//i.test(origin) && !isLoopback(origin)) ||
		valid.find((origin) => !isLoopback(origin)) ||
		valid.find((origin) => /^https:\/\//i.test(origin)) ||
		valid[0] ||
		''
	);
}

export function buildPageLink(pageKey: string, origins: string[]): string {
	const key = pageKey.trim();
	if (!key) return '';
	if (isHttpOrigin(key)) return key;
	const origin = preferredSiteOrigin(origins);
	if (!origin) return '';
	const path = key.startsWith('/') ? key : `/${key}`;
	return origin.replace(/\/+$/, '') + encodeURI(path);
}
