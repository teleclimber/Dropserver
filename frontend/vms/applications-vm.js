import Vue from 'vue';
import debounce from 'debounce';
import ds_axios from '../ds-axios-helper.js'

import user_vm from '../views/user/user-vm.js';

function loadApplications() {
	return new Promise( (resolve, reject) => {
		ds_axios.get( '/api/application' )
		.then( resp => {
			Vue.set( application_vm, 'applications', resp.data.apps );
			resolve();
		});
	});
}

function createNew() {
	// wipe data from create_status
	application_vm.create_status = {
		state: null,	// null, uploading, processing, error, enter-meta, finishing, finished
		upload_progress: 0,
		error_message: '',
		app_meta: {},	// id, name, ... of created application
		version_meta: {},	// app id, version, ... of created version
		cur_name: '',	// hmm not doing this anymoer
		name_available: {}	// or this.
	};
}
function createUpload( form_data ) {
	application_vm.create_status.state = 'uploading';

	ds_axios.post( '/api/application/', form_data, {	
		headers: {
			'Content-Type': 'multipart/form-data'
		},
		validateStatus: status => status == 200 || status == 422
	}).then( resp => {
		if( resp.status == 422 ) {
			application_vm.create_status.state = 'error';
			application_vm.create_status.error_message = resp.data.error;
		}
		else {
			console.log( 'returned 200');
			// that means everything is peachy as far as the server is concerned.
			// go to finishing screen?
			application_vm.create_status.state = 'finished';
		
			application_vm.create_status.app_meta = resp.data.app_meta;
			application_vm.create_status.version_meta = resp.data.app_meta.versions[0];

			application_vm.applications.push( resp.data.app_meta );	///eeeepp check data format

			//application_vm.create_status.temp_key = resp.data.temp_key;	// no longer needed?

			//application_vm.create_status.cur_name = resp.data.app_meta.name; //?
			//checkAppName_( resp.data.app_meta.name );
		}
	})
	.catch( () => {
		console.log('FAILURE!!');
	});
}
function openCreateAppSpace() {
	user_vm.closeAllModals();
	const app_meta = application_vm.create_status.app_meta
	user_vm.showCreateAppSpace( {
		app_name: app_meta.name,// app_name I think. app_id would be better!
		app_version: app_meta.version
	} );
}

function appNameChanged( app_name ) {
	application_vm.create_status.cur_name = app_name;
	checkAppName( app_name );
}
function checkAppName_( app_name ) {
	ds_axios.get( '/api/logged-in-user/application/', {
		params: {
			name: app_name,
			'exists-only': true
		},
		validateStatus: status => status == 200 || status == 404
	}).then( resp => {
		let avail;
		if( resp.status == 404 ) avail = true;
		else avail = false;
		Vue.set( application_vm.create_status.name_available, app_name, avail );
	});
}
const checkAppName = debounce( checkAppName_, 300 );

function showManageApplication( app_id ) {
	// need to hide manage applications modal...
	if( user_vm.closeAllModals() ) {
		application_vm.manage_status = {
			app_id: app_id,
			state: null,
			error_message: '',
			temp_key: null
		};
	}
}
function showVersionUpload() {
	application_vm.manage_status.state = 'upload';
}
function closeManageApplication() {
	application_vm.manage_status.app_id = null;
}

function uploadNewVersion( app_id, form_data ) {
	application_vm.manage_status.state = 'uploading';

	ds_axios.post( '/api/application/'+encodeURIComponent(app_id), form_data, {	
		headers: {
			'Content-Type': 'multipart/form-data'
		},
		validateStatus: status => status == 200 || status == 422
	}).then( resp => {
		if( resp.status == 422 ) {
			application_vm.manage_status.state = 'error';
			application_vm.manage_status.error_message = resp.data.error;
		}
		else {
			console.log( 'returned 200');

			// for new version, if everything looks good the server will "install it" right away
			// If not (like mismatched author names w/ prev version) then there may be an interim confirmation screen
			// other catches: version higher but migration_level lower

			// So the response data should include data like confirm:
			// ..and include meta data from prev version and what was read in temp dir.
			// Also include temp key. 


			if( resp.data.confirm ) {
				application_vm.create_status.state = 'enter-meta';
			
				application_vm.create_status.app_meta = resp.data.app_meta;
				application_vm.create_status.temp_key = resp.data.temp_key;
			}
			else {
				// went straght through.
				// finish upload and return to list of versions
				// Though probably need to reload that application's versions first
				loadApplications().then( () => {
					application_vm.manage_status.state = null;
					application_vm.manage_status.error_message = '';
					application_vm.manage_status.temp_key = null;
				});
			}
		}
	})
	.catch( () => {
		console.log('FAILURE!!');
	});
}

function deleteVersion( app_name, ver_name ) {
	return new Promise( (resolve, reject) => {
		ds_axios.delete( '/api/logged-in-user/application/'
			+encodeURIComponent(app_name)+'/'+encodeURIComponent(ver_name) )
		.then( () => {
			const application = application_vm.applications.find( a => a.name === app_name );
			const versions = application.versions;
			const i = versions.findIndex( v => v.name === ver_name );
			versions.splice( i, 1 );
			resolve();
		})
	});
}
function deleteApplication( app_name ) {
	return new Promise( (resolve, reject) => {
		ds_axios.delete( '/api/logged-in-user/application/'
			+encodeURIComponent(app_name) )
		.then( () => {
			application_vm.manage_status = {};
			const index = application_vm.applications.findIndex( a => a.name === app_name );
			application_vm.applications.splice( index, 1 );
			resolve();
		})
	});
}

function getVersionMeta( app_name, ver_name ) {
	const app = application_vm.applications.find( a => a.name === app_name );
	if( !app ) return;
	const ver_data = app.versions.find( v => v.name === ver_name );
	if( !ver_data ) return;
	if( !ver_data.meta ) {
		ds_axios.get( '/api/logged-in-user/application/'
			+encodeURIComponent(app_name)+'/'+encodeURIComponent(ver_name) )
		.then( resp => {
			Vue.set( ver_data, 'meta', resp.data );
		});
	}
}

const application_vm = {
	applications: [],	//this contains the applications.

	create_status: {},

	createNew,
	createUpload,
	openCreateAppSpace,

	appNameChanged,

	// manage application
	manage_status: {},

	showManageApplication,
	showVersionUpload,
	closeManageApplication,

	uploadNewVersion,
	deleteVersion,
	deleteApplication,

	getVersionMeta
};

loadApplications();
createNew();

export default application_vm;