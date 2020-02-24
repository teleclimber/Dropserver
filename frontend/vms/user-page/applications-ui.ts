import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import ApplicationsDM from '../../dms/applications-dm';
import AppspacesDM from '../../dms/appspaces-dm';

import CreateApplicationVM from './create-application-vm';
import ManageApplicationVM from './manage-application-vm';

type ApplicationsUIDeps = {
	applications_dm: ApplicationsDM,
	appspaces_dm: AppspacesDM
}

type ApplicationsUICbs = {
	//cancelCreateApplication(): void, 
	showCreateAppspace(app_id?: number, version?: string): void,
}

export default class ApplicationsUI {	// this is UI
	static injectKey = Symbol();

	parent: ApplicationsUICbs | undefined;

	@observable show_list: boolean;

	@observable create_vm: CreateApplicationVM | undefined;
	@observable manage_vm: ManageApplicationVM | undefined;

	constructor(private deps: ApplicationsUIDeps) {
		this.show_list = false;
	}

	@action
	showList() {
		this.show_list = true;
	}
	@action
	listCloseClicked() {
		// do something to close the list
		this.show_list = false;
	}

	@action
	createNew() {
		// Create new
		if( this.create_vm != undefined ) {
			console.error('create status should be undefined before creating new one');
		}

		this.show_list = false;
		this.create_vm = new CreateApplicationVM(this, {applications_dm: this.deps.applications_dm});
	}
	@action
	closeCreateClicked() {
		// call its termination function if it has one.
		this.create_vm = undefined;

		if( this.parent == undefined ) return;
		//this.parent.cancelCreateApplication();
	}

	createAppspaceClicked(app_id: number, version: string) {
		//TODO: first close your children VMS, right?
		if( this.parent == undefined ) return;	//not an error

		this.parent.showCreateAppspace(app_id, version);
	}

	@action
	showManageApplication(app_id: number) {
		this.show_list = false;
		this.manage_vm = new ManageApplicationVM({
			close: () => {
				this.manage_vm = undefined;
			}
		}, {
			applications_dm: this.deps.applications_dm,
			appspaces_dm: this.deps.appspaces_dm }, app_id);
	}
	
}


	
