// jest-expo stubs expo-crypto with fixed bytes, which makes every ULID of one millisecond collide.
// Use the platform CSPRNG so tests see what a device sees.
jest.mock("expo-crypto", () => ({
	getRandomValues: (a: Uint8Array<ArrayBuffer>): Uint8Array<ArrayBuffer> => crypto.getRandomValues(a),
}));
