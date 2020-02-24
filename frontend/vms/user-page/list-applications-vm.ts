// not sure where this fits, but is basically used to shoult be used
// ..to show, search, etc.. applications, along with derived data

import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import ApplicationsDM from '../../dms/applications-dm';
import AppspacesDM from "../../dms/appspaces-dm";

import AppUsesVM from './app-uses-vm';

type ListApplicationsVMDeps = {
	applications_dm: ApplicationsDM,
	appspaces_dm: AppspacesDM
}

export default class ListApplicationsVM {
	static injectKey = Symbol();

	constructor(private deps: ListApplicationsVMDeps) {}

	// sort methods, etc...
	get sorted_apps() {
		return this.deps.applications_dm.applications;
	}

	@computed get app_uses(): { [app_id: string]: AppUsesVM } {
		const ret : { [app_id: string]: AppUsesVM } = {};

		this.deps.applications_dm.applications.forEach( app_dm => {
			ret[app_dm.app_id] = new AppUsesVM(app_dm, {appspaces_dm:this.deps.appspaces_dm});
		});

		return ret;
	}
}