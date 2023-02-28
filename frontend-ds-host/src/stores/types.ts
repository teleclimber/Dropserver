// types are placed in this file instead of in store to prevent circular dependency.

export enum LoadState {
	NotLoaded = 0,
	Loading = 1,
	Loaded = 2
}

export interface AdminInvite {
	email: string
}

export interface UserForAdmin {
	user_id: number,
	email: string,
	is_admin: boolean
}