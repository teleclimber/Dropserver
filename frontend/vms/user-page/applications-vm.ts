import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import ApplicationsDM from '../../dms/applications-dm';
import SelectFilesVM from '../ui/select-app-files-vm';

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

export class CreateApplicationVM {
	@observable state: EditState = EditState.start;
	@observable app_meta?: ApplicationMeta;
	@observable version_meta?: VersionMeta;

	@observable select_files_vm: SelectFilesVM | undefined;

	constructor(private cbs: CreateApplicationVMCbs, private deps: CreateApplicationVMDeps) {
		this.select_files_vm = new SelectFilesVM;
	}

	@action
	doStartOver() {
		this.state = EditState.start;
		this.select_files_vm = new SelectFilesVM;
		// actually it should delete itself and recreate.
		// need to call back to parent for that. may not bother.
	}

	@computed get app_files_error(): string {	// ooff we already have an "error message"
		if( !this.select_files_vm ) {
			return '';
		}
		if( this.select_files_vm.error ) {
			return this.select_files_vm.error;
		}

		if( !this.select_files_vm.app_files || !this.select_files_vm.metadata ) {
			return '';
		}

		return '';
	}

	@computed get version_comparison(): VersionComparison | undefined {	// temporary
		if( !this.select_files_vm || !this.select_files_vm.app_files || !this.select_files_vm.metadata || this.app_files_error ) {
			return undefined;
		}

		const ret:VersionComparison = {
			upload: this.select_files_vm.metadata,
			previous: undefined,
			next: undefined,
			fatal: false,
			errors: {
				schema: '',
				version: ''
			}
		};

		return ret;
	}

	@computed get enable_upload() {
		return this.select_files_vm 
			&& this.select_files_vm.app_files 
			&& this.select_files_vm.metadata
			&& !this.select_files_vm.error;
	}

	@action
	async doUpload() {
		if( !this.enable_upload )return;
		if( !this.select_files_vm  || !this.select_files_vm.app_files ) return;	//friggin typescript

		this.state = EditState.uploading;
	
		const upRet = await this.deps.applications_dm.uploadNewApplication(this.select_files_vm.app_files);
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

	@observable delete_check: string = '';

	@observable select_files_vm: SelectFilesVM | undefined;

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
		this.select_files_vm = new SelectFilesVM;
		this.state = EditState.upload;
	}

	@computed get app_files_error(): string {
		if( !this.select_files_vm ) {
			return '';
		}
		if( this.select_files_vm.error ) {
			return this.select_files_vm.error;
		}

		if( !this.select_files_vm.app_files || !this.select_files_vm.metadata ) {
			return '';
		}

		const upload_version = this.select_files_vm.metadata.version;
		if( this.deps.applications_dm.versionExists(this.app_id, upload_version) ) {
			return 'Version '+upload_version+' already exists';
		}

		return '';
	}

	@computed get version_comparison(): VersionComparison | undefined {
		if( !this.select_files_vm || !this.select_files_vm.app_files || !this.select_files_vm.metadata || this.app_files_error ) {
			return undefined;
		}

		const upload_version = this.select_files_vm.metadata.version;

		const ret:VersionComparison = {
			upload: this.select_files_vm.metadata,
			previous: this.deps.applications_dm.getPrevVersion(this.app_id, upload_version),
			next: this.deps.applications_dm.getNextVersion(this.app_id, upload_version),
			fatal: false,
			errors: {
				schema: '',
				version: ''
			}
		};

		if( (ret.previous && ret.previous.schema > ret.upload.schema)
			|| (ret.next && ret.next.schema < ret.upload.schema) ) {
			ret.fatal = true;
			ret.errors.schema = 'Schema is out of whack';
			return ret;
		}

		return ret;
	}

	@computed get enable_upload() {
		return this.select_files_vm 
		&& this.select_files_vm.app_files
		&& this.select_files_vm.metadata
		&& !this.select_files_vm.error
		&& this.version_comparison
		&& !this.version_comparison.fatal;
	}

	async uploadNewVersion() {
		if( !this.enable_upload ) return;
		if( !this.select_files_vm || !this.select_files_vm.app_files ) {	// ugh typescript
			return;
		}

		this.state = EditState.uploading;

		const upRet = await this.deps.applications_dm.uploadNewVersion(this.app_id, this.select_files_vm.app_files);

		if( upRet.error || upRet.version_meta == undefined ) {
			// I don't know what to do exactly.
			this.state = EditState.error;
		}
		else {
			// check upRet structure
			this.state = EditState.finished;

			// TODO: still somewhat unresolved what happens here
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
	
