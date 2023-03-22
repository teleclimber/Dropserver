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

// UserDropID is a dropid of a local user
export interface UserDropID {
	user_id: number,
	handle: string,
	domain_name: string,
	compound_id: string,
	display_name: string,
	created_dt: Date
}

export interface AppVersion {
	app_id: number,
	version: string,
	app_name: string,	// unused?
	api_version: number,	
	schema: number,
	created_dt: Date
}

export interface App {
	app_id: number,
	name: string,
	created_dt: Date,
	versions: AppVersion[]
}

export type SelectedFile = {
	file: File,
	rel_path: string
}

export interface Appspace {
	appspace_id: number,
	domain_name: string,
	no_tls: boolean,
	port_string: string,
	dropid: string,
	created_dt: Date,
	paused: boolean,
	app_id: number,
	app_version: string,
	upgrade_version: string|undefined
}

export interface RemoteAppspace {
	domain_name: string,
	owner_dropid: string,
	user_dropid: string,
	no_tls: boolean,
	port_string: string,
	created_dt: Date
}
