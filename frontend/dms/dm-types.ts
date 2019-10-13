
// type User = {
// 	user_id: number,
// 	email_zzz: string,
// 	is_admin_zzz: boolean
// }

// type ApplicationMeta = {
// 	app_id: number,
// 	app_name: string,
// 	created_dt: Date,
// 	versions: VersionMeta[]
// }

// type VersionMeta = {
// 	app_name: string,
// 	version: string,
// 	schema: number,
// 	created_dt: Date,
// }

type AppspaceMeta = {
	appspace_id: number,
	app_id: number,
	app_version: string,
	subdomain: string,
	paused: boolean,
	created_dt: Date,
}

type SelectedFile = {
	file: File,
	rel_path: string
}