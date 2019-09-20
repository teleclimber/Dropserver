
type User = {
	user_id: number,
	email: string,
	is_admin: boolean
}

type ApplicationMeta = {
	app_id: number,
	app_name: string,
	created_dt: Date,
	versions: VersionMeta[]
}

type VersionMeta = {
	version: string,
	schema: number,
	created_dt: Date,
}

type AppspaceMeta = {
	appspace_id: number,
	app_id: number,
	app_version: string,
	subdomain: string,
	paused: boolean,
	created_dt: Date,
}