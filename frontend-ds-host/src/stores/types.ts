export enum LoadState {
	NotLoaded = 0,
	Loading = 1,
	Loaded = 2
}

export interface UserForAdmin {
	user_id: number,
	email: string,
	is_admin: boolean
}