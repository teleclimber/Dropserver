import { LoadState } from "./types";

// experimental...

const LoadStateSymbol = Symbol();

export type Loadable<T> = T & {
	[LoadStateSymbol]: LoadState
}

export function attachLoadState<T extends object>(r :T, l: LoadState) : Loadable<T> {
	Object.defineProperty(r, LoadStateSymbol, {
		configurable: false,
		enumerable: false,
		writable: true,
		value: l	// This might need to be ref(l)? At least in some cases?
	});
	
	return r as Loadable<T>;
}

export function getLoadState<T>(r :Loadable<T>) :LoadState {
	return r[LoadStateSymbol];
}

export function setLoadState<T>(r :Loadable<T>, l:LoadState) {
	return r[LoadStateSymbol] = l;
}
