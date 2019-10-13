// augmented appspace data 
// with data about application etc...
// and callbacks to whatever parent
// this is a new pattern, so good luck.

import { action, computed, observable, decorate, configure, runInAction } from "mobx";

import { ApplicationMeta, VersionMeta } from '../../generated-types/userroutes-classes';

import ApplicationsDM from '../../dms/applications-dm';

type AppspaceVMCbs = {
	showManage(appspace_id: number): void
}
type AppspaceVMDeps = {
	applications_dm: ApplicationsDM,
}
export default class AppspaceVM {
	constructor(private cbs: AppspaceVMCbs, private deps: AppspaceVMDeps, public appspace: AppspaceMeta) {
	}

	@computed
	get application(): ApplicationMeta {
		const app_id = this.appspace.app_id;
		let a = this.deps.applications_dm.applications.find( (a:ApplicationMeta) => a.app_id === app_id );
		if( a == undefined ) {
			console.log("application not found, sending in the dummy...");
			a = {
				app_id: -1,
				app_name:'...loading...',
				created_dt: new Date,
				versions:[],
			}
		}
		return a;
	}
	@computed get version(): VersionMeta {
		const version = this.appspace.app_version;
		const v = this.application.versions.find((v:VersionMeta) => v.version === version);
		if(!v) {
			throw new Error("version not found");
		}
		return v;
	}

	// map appspace data as getters:
	@computed get app_id() {
		return this.appspace.app_id;
	}
	@computed get app_version() {
		return this.appspace.app_version;
	}
	@computed get subdomain() {
		return this.appspace.subdomain;
	}
	@computed get paused() {
		return this.appspace.paused;
	}

	// actual computed values
	@computed
	get open_url() {
		const loc = window.location;
		return '//'+ this.appspace.subdomain+'.'+getBaseDomain()+':'+loc.port;
	}
	@computed
	get display_url() {
		//const loc = window.location;
		return window.location.protocol+'//'+ this.appspace.subdomain+'.'+getBaseDomain();
	}

	@computed
	get upgrade(): string | undefined {
		if( this.application && this.application.versions.length != 0 ) {
			const latest_version = this.application.versions[0];
			if( latest_version.version !== this.appspace.app_version ) {
				return latest_version.version;
			}
		}
	}

	doUpgrade() {
		//this.cbs.
	}
	manage() {
		this.cbs.showManage(this.appspace.appspace_id);
	}
}


function getBaseDomain() {	//not the right way to do this.
	const pieces = window.location.hostname.split( '.' );
	pieces.shift();
	return pieces.join( '.' );
}