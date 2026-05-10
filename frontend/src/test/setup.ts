import "@testing-library/jest-dom/vitest";

// api.ts reads localStorage at module load time; ensure it exists in jsdom
Object.defineProperty(globalThis, "localStorage", {
	value: {
		getItem: () => null,
		setItem: () => {},
		removeItem: () => {},
		clear: () => {},
	},
	writable: true,
});
