
import ds_axios from '../../ds-axios-helper.js'
import applications_vm from '../../vms/applications-vm.js';
import app_spaces_vm from '../../vms/app-spaces-vm.js';
import change_pw_vm from '../../vms/change-pw-vm.js';

ds_axios.get( '/api/logged-in-user/user-data' )
	.then(function (response) {
		console.log('got user data', response);
		vm.user = response.data;
	})
	.catch(function (error) {
		//console.log('something wrong getting user data', error);
		if( error.response.status == 401 ) window.location.href = '/login';
		else throw new Error( error );
	});

function closeAllModals() {
	vm.ui.show_create_appspace = false;
	vm.ui.show_manage_appspace = false;
	vm.ui.show_manage_applications = false;
	vm.ui.show_create_application = false;

	return true;
}

function showCreateAppSpace( app_data ) {
	if( !vm.ui.show_create_appspace && !vm.ui.show_manage_appspace ) {
		vm.ui.show_create_appspace = true;
		app_spaces_vm.showCreateAppSpace( app_data );
	}
}
function cancelCreateAppSpace() {
	vm.ui.show_create_appspace = false;
}

function showManageAppSpace( app_space, shortcut ) {
	vm.ui.show_manage_appspace = true;
	app_spaces_vm.manageAppSpace( app_space );
	if( shortcut ) {
		if( shortcut.page === 'upgrade' ) app_spaces_vm.showUpgradeVersion( shortcut.version );
	}
}
function closeManageAppSpace() {
	vm.ui.show_manage_appspace = false;
}


// manage applications:
function showManageApplications() {
	vm.ui.show_manage_applications = true;	//make that a bit smarter please
}
function closeManageApplications() {
	vm.ui.show_manage_applications = false;
}

// create application
function showCreateApplication() {
	closeManageApplications();
	vm.ui.show_create_application = true;
}
function cancelCreateApplication() {
	// cancel at VM?
	applications_vm.createNew();
	vm.ui.show_create_application = false;
}

//password
function showChangePassword() {
	vm.ui.show_change_pw = true;
}
function closeChangePassword() {
	vm.ui.show_change_pw = false;
}

const vm = {
	ui: {
		show_create_appspace: false,
		show_manage_appspace: false,
		show_manage_applications: false,
		show_create_application: false,
		show_change_pw: false
	},
	user: {},

	applications_vm,
	app_spaces_vm,
	change_pw_vm,

	closeAllModals,

	showCreateAppSpace,
	cancelCreateAppSpace,

	showManageAppSpace,
	closeManageAppSpace,

	showManageApplications,
	closeManageApplications,

	showCreateApplication,
	cancelCreateApplication,
	
	showChangePassword,
	closeChangePassword
	
};

export default vm;