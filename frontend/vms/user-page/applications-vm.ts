import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import ApplicationsDM from '../../dms/applications-dm';

export enum EditState { start, upload, uploading, processing, error, enter_meta, finishing, finished };

type ApplicationsVMDeps = {
	applications_dm: ApplicationsDM,
}

type ApplicationsVMCbs = {
	//cancelCreateApplication(): void, 
	showCreateAppspace(app_id?: number, version?: string): void,
}

export default class ApplicationsVM {
	static injectKey = Symbol();

	parent: ApplicationsVMCbs | undefined;

	@observable show_list: boolean;

	@observable create_vm: CreateApplicationVM | undefined;
	@observable manage_vm: ManageApplicationVM | undefined;

	constructor(private deps: ApplicationsVMDeps) {
		this.show_list = false;
	}

	// list
	// showList() ...
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
		}, {applications_dm: this.deps.applications_dm}, app_id);
	}
	
}

type CreateApplicationVMDeps = {
	applications_dm: ApplicationsDM,
}

type CreateApplicationVMCbs = {
	closeCreateClicked(): void, 
	createAppspaceClicked(app_id: number, version: string): void,
}

export class CreateApplicationVM {
	@observable state: EditState = EditState.start;
	@observable error_message: string = '';
	@observable upload_data: any;
	@observable app_meta?: ApplicationMeta;
	@observable version_meta?: VersionMeta;

	constructor(private cbs: CreateApplicationVMCbs, private deps: CreateApplicationVMDeps) {
	}

	@action
	doStartOver() {
		this.state = EditState.start;
		// actually it should delete itself and recreate.
		// need to call back to parent for that. may not bother.
	}

	@action
	async doUpload() {
		this.state = EditState.uploading;
	
		const upRet = await this.deps.applications_dm.uploadNewApplication(this.upload_data);
		runInAction( () => {	//because of await
			if( upRet.error || upRet.app_meta == undefined ) {
				// I don't know what to do exactly.
				this.state = EditState.error;
			}
			else {
				// check upRet structure
				this.state = EditState.finished;
				this.app_meta = upRet.app_meta;
				this.version_meta = upRet.app_meta.versions[0];
			}
		});
	}

	createAppspaceClicked() {
		if( !this.app_meta || !this.version_meta ) return;
		this.cbs.createAppspaceClicked(this.app_meta.app_id, this.version_meta.version);
	}

	doClose() {
		this.cbs.closeCreateClicked();
	}
}

type ManageApplicationVMDeps = {
	applications_dm: ApplicationsDM
}
type ManageApplicationVMCbs = {
	close(): void,
}
export class ManageApplicationVM {
	app_id: number;
	application: ApplicationMeta;
	@observable show_version: VersionMeta | undefined;
	@observable state: EditState = EditState.start;
	@observable error_message: string = '';

	@observable upload_data: any = null;
	@observable delete_check: string = '';

	constructor(private cbs: ManageApplicationVMCbs, private deps: ManageApplicationVMDeps, app_id: number) {
		this.app_id = app_id;
		const a = this.deps.applications_dm.applications.find( (a:ApplicationMeta) => a.app_id === app_id );
		if( !a ) {
			throw new Error("application not found for app_id "+app_id);
		}
		this.application = a;	//should this not be a deep copy, or something?
	}

	@action
	showVersion(version: VersionMeta) {
		this.show_version = version;
	}

	@action 
	closeClicked() {
		if( this.show_version ) this.show_version = undefined;
		else this.cbs.close();
	}

	@action
	showVersionUpload() {
		this.state = EditState.upload;
	}

	async uploadNewVersion() {
		this.state = EditState.uploading;

		const upRet = await this.deps.applications_dm.uploadNewVersion(this.app_id, this.upload_data);

		if( upRet.error || upRet.version_meta == undefined ) {
			// I don't know what to do exactly.
			this.state = EditState.error;
		}
		else {
			// check upRet structure
			this.state = EditState.finished;
			//this.manage_status.app_meta = upRet.app_meta;
			//this.manage_status.version_meta = upRet.app_meta.versions[0];

			// I think what we have to do here is in flux, 
			// ..based on our incomplete implementation and even incomplete design of this aspect of DS

			// we could go back to manage app, which should show new version in listing?
			// ..or we need to show appsapces that could use an upgrade?
		}

	}

	async deleteVersion(version: string) {
		await this.deps.applications_dm.deleteVersion( this.application.app_id, version );
		if( this.show_version && this.show_version.version === version ) {
			runInAction( () => this.show_version = undefined );
		}
	}

	// delete 
	@computed get allow_delete() {
		return this.delete_check === this.application.app_name;
	}
	async deleteApplication() {
		await this.deps.applications_dm.deleteApplication( this.app_id );
		this.cbs.close();
	}

}
	
