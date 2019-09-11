import ds_axios from '../../ds-axios-helper.js'
import user_vm from './user-vm.js';
import application_vm from './applications-vm.js';
import Vue from 'vue';

function loadAppSpaces() {
	return new Promise( (resolve, reject) => {
		ds_axios.get( '/api/appspace' )
		.then(function (response) {
			console.log('got app-space data', response);
			app_spaces_vm.app_spaces = response.data.appspaces;
			resolve();
		});	
	});
}

function showCreateAppSpace( data ) {
	Vue.nextTick( () => {
		app_spaces_vm.create_data = {
			app_name: data && data.app_name ? data.app_name : ''
		}
	});
}
function createAppSpace( data ) {	// app-spaces should have their own vm
	app_spaces_vm.action_pending = 'Creating...';

	ds_axios.post( '/api/appspace', data )
	.then( function(response) {
		console.log( 'create app space resp', response );
		app_spaces_vm.app_spaces.push( response.data.appspace );
		app_spaces_vm.managed_app_space = findAppSpace( response.data.appspace.appspace_id );

		app_spaces_vm.action_pending = null;
		app_spaces_vm.state = 'created';
	});
}

function manageAppSpace( app_space ) {
	app_spaces_vm.managed_app_space = app_space;
}

function pauseAppSpace( app_space, pause_on ) {
	app_spaces_vm.action_pending = pause_on ? 'Pausing...' : 'Unpausing...';
	ds_axios.patch( '/api/logged-in-user/appspaces/'+encodeURIComponent(app_space.id), {
		pause: !!pause_on
	} ).then( () => {
		app_space.paused = !!pause_on;
		app_spaces_vm.action_pending = null;
	});
}

function deleteAppSpace( app_space ) {
	// how does this work?
	// send request.
	// put UI in "pending..." state I suppose?  -> other processes will have that too: upgrade, backup/export, ...
	// then have to remove the app_space from list.

	app_spaces_vm.action_pending = 'Deleting...';

	ds_axios.delete( '/api/logged-in-user/appspaces/'+encodeURIComponent(app_space.id) )
	.then( response => {
		return loadAppSpaces();
	})
	.then( () => {
		app_spaces_vm.action_pending = null;
		app_spaces_vm.managed_app_space = null;
		// somehow cancel the UI side of things.
		user_vm.closeManageAppSpace();
	});
}

////
function showPickVersion() {
	app_spaces_vm.state = 'pick-version';
}
function closePickVersion() {
	app_spaces_vm.state = null;
}
function showUpgradeVersion( version ) {
	app_spaces_vm.state = 'show-upgrade';
	app_spaces_vm.upgrade_version = version;

	//application_vm.getVersionMeta( app_spaces_vm.managed_app_space.app_id, version );
}
function closeUpgradeVersion() {
	app_spaces_vm.upgrade_version = null;
	showPickVersion();
}
function doUpgradeVersion() {
	app_spaces_vm.action_pending = 'Upgrading...';

	const app_space = app_spaces_vm.managed_app_space;

	ds_axios.post( '/api/appspace/'+encodeURIComponent(app_space.id)+'/version', {
		version: app_spaces_vm.upgrade_version
	} ).then( resp => {
		const index = app_spaces_vm.app_spaces.findIndex( a => a.id === app_space.id );
		app_spaces_vm.app_spaces.splice( index, 1, resp.data );
		app_spaces_vm.managed_app_space = resp.data;

		app_spaces_vm.upgrade_version = null;
		app_spaces_vm.state = null;
		app_spaces_vm.action_pending = null;
	})
}

//// util
function findAppSpace( id ) {
	return app_spaces_vm.app_spaces.find( a => a.appspace_id === id );
}
function getBaseDomain() {	//not the right way to do this.
	const pieces = window.location.hostname.split( '.' );
	pieces.shift();
	return pieces.join( '.' );
}
function getOpenUrl( app_space ) {
	const loc = window.location;
	return '//'+ app_space.id+'.'+getBaseDomain()+':'+loc.port;
}
function getDisplayUrl( app_space ) {
	//const loc = window.location;
	return window.location.protocol+'//'+ app_space.subdomain+'.'+getBaseDomain();
}
// could also have displayUrlParts if you wanted.

const app_spaces_vm = {
	app_spaces: [],	// if this is raw appspace metadata, it is missing joined fields like application name, version, etc...
	create_data: {},
	managed_app_space: null,
	action_pending: null,
	state: null,
	upgrade_version: null,

	showCreateAppSpace,
	createAppSpace,
	manageAppSpace,
	pauseAppSpace,
	deleteAppSpace,

	showPickVersion,
	closePickVersion,
	showUpgradeVersion,
	closeUpgradeVersion,

	doUpgradeVersion,

	//util:
	getOpenUrl,
	getDisplayUrl

};

loadAppSpaces();

export default app_spaces_vm;