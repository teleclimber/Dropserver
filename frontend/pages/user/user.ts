import Vue from 'vue';
import UserPage from '../../components/user-page/UserPage.vue';

import CurrentUserDM from '../../dms/current-user-dm';
import ApplicationsDM from '../../dms/applications-dm';
import AppspacesDM from '../../dms/appspaces-dm';

import UserPageVM from '../../vms/user-page/user-page-vm';
import ApplicationsVM from '../../vms/user-page/applications-vm';

import '../style.css';
import AppspacesVM from '../../vms/user-page/appspaces-vm';

// OK, let's rethink this.
// there is probably at least one vm: user-page-vm.ts
// Then we need applications-dm, appspaces-dm, ...?
// It's possible that we also need applications-vm.ts, and appspaces-vm.ts
// .. which would handle the ui of the management of these.

// the dms and vms are almost certainly "provided"
// Not clear where the distinction is between the different vms though.

// Each vm and dm must be a mobx store
// .. and each must be initialized here
// .. and passed down either through injection or because it is created from a parent.


const current_user_dm = new CurrentUserDM;

const applications_dm = new ApplicationsDM;
applications_dm.fetchAll();

const appspaces_dm = new AppspacesDM;


const applications_vm = new ApplicationsVM({
	applications_dm
});

const appspaces_vm = new AppspacesVM({
	applications_dm: applications_dm,
	appspaces_dm: appspaces_dm,
});

const user_page_vm = new UserPageVM({
	current_user_dm,	
	applications_dm,

	applications_vm,
	appspaces_vm,
});

// The other option here is to have a single vm, 
// .. that in turn creates its child vms
// .. and inject where needed as needed by referencing the child from the local vm.

new Vue({
	el: '#app',
	provide: {
		// dms:
		[CurrentUserDM.injectKey]: current_user_dm,
		[ApplicationsDM.injectKey]: applications_dm,
		[AppspacesDM.injectKey]: appspaces_dm,
		// vms:
		[UserPageVM.injectKey]: user_page_vm,
		[ApplicationsVM.injectKey]: applications_vm,
		[AppspacesVM.injectKey]: appspaces_vm,

	},
	render: h => h(UserPage)
});

