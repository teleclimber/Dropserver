import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import ApplicationsDM from '../../dms/applications-dm';
import ApplicationDM from '../../dms/application-dm';// is that really a DM?
import { VersionComparison } from './app-uses-vm';

import SelectFilesVM from '../ui/select-app-files-vm';

export enum EditState { start, upload, uploading, processing, error, enter_meta, finishing, finished };

type CreateApplicationVMDeps = {
	applications_dm: ApplicationsDM,
}

type CreateApplicationVMCbs = {
	closeCreateClicked(): void, 
	createAppspaceClicked(app_id: number, version: string): void,
}

export default class CreateApplicationVM {
	@observable state: EditState = EditState.start;
	@observable application?: ApplicationDM;
	@observable version_meta?: {
		app_name: string,
		version: string,
		schema: number
	};

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
			if( upRet.error || upRet.application == undefined ) {
				// I don't know what to do exactly.
				this.state = EditState.error;
			}
			else {
				// check upRet structure
				this.state = EditState.finished;
				this.application = upRet.application;
				this.version_meta = upRet.application.sorted_versions[0];
			}
		});
	}

	createAppspaceClicked() {
		if( !this.application || !this.version_meta ) return;
		this.cbs.createAppspaceClicked(this.application.app_id, this.version_meta.version);
	}

	doClose() {
		this.cbs.closeCreateClicked();
	}
}