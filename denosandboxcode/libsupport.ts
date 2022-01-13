import Metadata from './metadata.ts';
import DsServices from './ds-services.ts';
import Migrations from './migrations.ts';
import AppRoutes from './app-router.ts';
import Users from './appspace-users.ts';

export default class LibSupport {
	_migrations: Migrations|undefined;
	_appRoutes: AppRoutes|undefined;
	users:Users;
	constructor(private _metadata:Metadata, public services:DsServices ){
		this.users = new Users(services);	// maybe move this to index to follow pattern?
	}
	setMigrations(migrations:Migrations) {
		this._migrations = migrations;
	}
	get migrations() :Migrations {
		if( this._migrations === undefined ) throw new Error("migrations undefined in libSupport");
		return this._migrations;
	}
	setAppRoutes(appRoutes:AppRoutes) {
		this._appRoutes = appRoutes;
	}
	get appRoutes() :AppRoutes {
		if( this._appRoutes === undefined ) throw new Error("appRoutes undefined in libSupport");
		return this._appRoutes;
	}
	setMetadata(metadata:Metadata) {
		this._metadata = metadata;
	}
	get Metadata() :Metadata {
		if( this._metadata === undefined ) throw new Error("metadata undefined in libSupport");
		return this._metadata;
	}
	get appPath() {
		return this.Metadata.app_path;
	}
	get appspacePath() {
		return this.Metadata.appspace_path;
	}
	get avatarsPath() {
		return this.Metadata.avatars_path;
	}
}
