import Vue from 'vue';
import UserPage from '../../components/user-page/UserPage.vue';

import CurrentUserDM from '../../dms/current-user-dm';
import ApplicationsDM from '../../dms/applications-dm';
import AppspacesDM from '../../dms/appspaces-dm';
import LiveDataDM from '../../dms/live-data-dm';

import UserPageUI from '../../vms/user-page/user-page-ui';
import ApplicationsUI from '../../vms/user-page/applications-ui';

import '../style.css';
import AppspacesUI from '../../vms/user-page/appspaces-ui';
import ListApplicationsVM from '../../vms/user-page/list-applications-vm';

const current_user_dm = new CurrentUserDM;

const applications_dm = new ApplicationsDM;
applications_dm.fetchAll();

const appspaces_dm = new AppspacesDM;
appspaces_dm.fetch();

const live_data_dm = new LiveDataDM;

const list_apps_vm = new ListApplicationsVM({
	applications_dm,
	appspaces_dm
});


const applications_ui = new ApplicationsUI({
	applications_dm,
	appspaces_dm
});

const appspaces_ui = new AppspacesUI({
	applications_dm: applications_dm,
	appspaces_dm: appspaces_dm,
	live_data_dm: live_data_dm
});

const user_page_vm = new UserPageUI({
	current_user_dm,	
	applications_dm,

	applications_ui,
	appspaces_ui,
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
		[UserPageUI.injectKey]: user_page_vm,
		[ListApplicationsVM.injectKey]: list_apps_vm,
		[ApplicationsUI.injectKey]: applications_ui,
		[AppspacesUI.injectKey]: appspaces_ui,

	},
	render: h => h(UserPage)
});

