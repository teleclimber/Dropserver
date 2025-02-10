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

export type AppMigrationStep = {
	direction: "up"|"down"
	schema: number
}
export type AppManifest = {
	name :string,
	short_description: string,
	version :string,
	release_date: Date|undefined,
	main: string,	// do we care here?
	schema: number,
	migrations: AppMigrationStep[],
	lib_version: string,	//semver
	signature: string,	//later
	code_state: string,	 // ? later
	icon: string,	// how to reference icon? app version should have  adefault path so no need to reference it here? Except to know if there is one or not
	accent_color: string,

	authors: AppAuthor[],
	description: string,	// actually a reference to a long description. Later.
	release_notes: string,	// ref to a file or something...
	code: string,	// URL to code repo. OK.
	website: string,	//URL to home page for app
	help: string,	// URL to help
	license: string,	// SPDX format of license
	license_file: string,	// maybe this is like icon, lets us know it exists and can use the link to the file.
	funding: string,	// URL for now, but later maybe array of objects? Or...?

	size: number	// bytes of what? compressed package? 
}

export interface AppVersion {
	app_id: number,
	version: string,
	schema: number,
	created_dt: Date,
}

export interface AppAuthor {
	name: string,
	email: string,
	url: string
}

export interface AppVersionUI {
	app_id: number,
	version: string,
	schema: number,
	created_dt: Date,
	name: string,
	short_desc: string,
	//icon: boolean,	// implement later
	color: string | undefined,
	authors: AppAuthor[],
	website: string,
	code: string,
	funding: string,
	release_date: string,
	license: string
}

export interface App {
	app_id: number,
	created_dt: Date,
	cur_ver: string | undefined,
	ver_data: AppVersionUI | undefined
	url_data: AppUrlData | undefined
}

export type AppGetMeta = {
	key: string,
	prev_version: string,
	next_version: string,
	errors: string[],
	warnings: Warning[],
	version_manifest?: AppManifest,
	app_id: number
}

export type Warning = {
	field: string,
	problem: string,
	bad_value: string,
	message: string
}

export type AppUrlData = {
	app_id: number,
	url: string,
	automatic: boolean,
	last_dt: Date,
	last_result: string,
	new_url: string,
	new_url_dt: Date|undefined,
	listing_dt: Date
	latest_version: string
}

export type SelectedFile = {
	file: File,
	rel_path: string
}

export interface AppspaceStatus {
	owner_id: number,
	appspace_id: number, 
	paused: boolean,
	temp_paused: boolean,
	temp_pause_reason: string,
	appspace_schema: number,
	app_version_schema: number,
	problem: boolean
}

export interface TSNetWarning {
	title: string,
	text: string,
	severity: string,
	impacts_connectivity: boolean
}
export interface AppspaceTSNetStatus {
	control_url: string,
	url: string,
	ip4: string,
	ip6: string,
	listening_tls: boolean,
	tailnet: string,
	name: string,
	https_available: boolean,
	magic_dns_enabled: boolean,
	tags: string[],
	err_message: string,
	state: string,
	browse_to_url: string,
	login_finished: boolean,
	warnings: TSNetWarning[],
	transitory: string
}

export interface AppspaceTSNetData {
	control_url: string,
	hostname: string,
	connect: boolean
}

export interface TSNetUserDevice {
	id: string,
	name: string,
	online: boolean | undefined,
	last_seen: Date |undefined,
	os: string,
	device_model: string,
	app: string
}
export interface TSNetPeerUser {
	id: string,
	control_url: string,
	login_name: string,
	display_name: string,
	sharee: boolean,
	devices: TSNetUserDevice[],
	full_id: string
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
	upgrade_version: string|undefined,
	status: AppspaceStatus,
	tsnet_status: AppspaceTSNetStatus,
	ver_data: AppVersionUI | undefined,
	tsnet_data: AppspaceTSNetData | undefined
}

export interface RemoteAppspace {
	domain_name: string,
	owner_dropid: string,
	user_dropid: string,
	no_tls: boolean,
	port_string: string,
	created_dt: Date
}

export interface AppspaceUserAuth {
	type: string,
	identifier: string,
	created: Date,
}

export interface AppspaceUser {
	appspace_id: number,
	proxy_id: string,
	auths: AppspaceUserAuth[],
	display_name: string,
	avatar: string,
	//permissions = raw.permissions;
	created_dt: Date,
}

export interface AppspaceMigrationJob {
	job_id: number,
	owner_id: number,
	appspace_id: number,
	to_version: string,
	created: Date,
	started: null | Date,
	finished: null | Date,
	priority: boolean,
	error: string | null
}