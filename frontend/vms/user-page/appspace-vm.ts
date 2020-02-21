// augmented appspace data 
// with data about application etc...
// and callbacks to whatever parent
// this is a new pattern, so good luck.

import { action, computed, observable, decorate, configure, runInAction } from "mobx";

import { ApplicationMeta, VersionMeta } from '../../generated-types/userroutes-classes';

import ApplicationsDM from '../../dms/applications-dm';
import ApplicationDM from '../../dms/application-dm';
import AppspaceDM from '../../dms/appspace-dm';
import LiveDataDM from '../../dms/live-data-dm';

type AppspaceVMCbs = {
	showManage(appspace_id: number): void,
	showUpgrade(appspace_id: number): void
}
type AppspaceVMDeps = {
	applications_dm: ApplicationsDM,
	live_data_dm:LiveDataDM
}
export default class AppspaceVM {
	constructor(private cbs: AppspaceVMCbs, private deps: AppspaceVMDeps, public appspace: AppspaceDM) {
	}

	@computed
	get application(): ApplicationDM {
		let a : ApplicationDM;
		if( this.deps.applications_dm.fetched ) {
			a = this.deps.applications_dm.getApplication(this.appspace.app_id);
		}
		else {
			console.log("application not found, sending in the dummy...");
			a = new ApplicationDM({
				app_id: -1,
				app_name:'...loading...',
				created_dt: new Date,
				versions:[],
			});
		}
		return a;
	}
	@computed get version(): VersionMeta {
		return this.application.getVersion(this.appspace.app_version);
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
			const latest_version = this.application.sorted_versions[0];
			if( latest_version.version !== this.appspace.app_version ) {
				return latest_version.version;
			}
		}
	}

	// I think these might be misplaced?
	showUpgrade() {
		this.cbs.showUpgrade(this.appspace.appspace_id);
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