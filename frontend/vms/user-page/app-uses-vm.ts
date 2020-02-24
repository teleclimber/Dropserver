import { action, computed, observable, decorate, configure, runInAction } from "mobx";

import { VersionMeta } from '../../generated-types/userroutes-classes';

import ApplicationDM from '../../dms/application-dm';
import AppspacesDM from '../../dms/appspaces-dm';

export type VersionComparison = {
	upload: VersionMeta,
	previous?: VersionMeta,
	next?: VersionMeta,
	fatal: boolean,
	errors: {
		version: string,
		schema: string,
	}
}

type AppUsesVMDeps = {
	appspaces_dm: AppspacesDM
}

export default class AppUsesVM {
	constructor( private app_dm: ApplicationDM, private deps: AppUsesVMDeps) {
		
	}

	@computed get version_num_appspace() : { [version:string]: number } {
		const version_as = this.deps.appspaces_dm.app_version_appspaces[this.app_dm.app_id];
		if( !version_as ) return {};

		const ret : { [version:string]: number } = {};
		
		for( let v in version_as ) {
			ret[v] = version_as[v].length;
		}
		return ret;
	}

	@computed get num_appspace() :number {
		let sum = 0;

		for( let v in this.version_num_appspace ) {
			sum += this.version_num_appspace[v];
		}

		return sum;
	}
}
