import { createRouter, createWebHistory } from 'vue-router';
import type { RouteLocationNormalized, RouteRecordRaw} from 'vue-router';

import Home from '../views/Home.vue';
import User from '../views/User.vue';

import Appspaces from '../views/Appspaces.vue';
import ManageAppspace from '../views/ManageAppspace.vue';
import MigrateAppspace from '../views/MigrateAppspace.vue';
import RestoreAppspace from '../views/RestoreAppspace.vue';
import ManageAppspaceUser from '../views/ManageAppspaceUser.vue';
import NewAppspace from '../views/NewAppspace.vue';

import NewRemoteAppspace from '../views/NewRemoteAppspace.vue';
import ManageRemoteAppspace from '../views/ManageRemoteAppspace.vue';

import Apps from '../views/Apps.vue';
import ManageApp from '../views/ManageApp.vue';
import NewApp from '../views/NewApp.vue';
import NewAppInProcess from '../views/NewAppInProcess.vue';

import Contacts from '../views/Contacts.vue';
import ManageContact from '../views/ManageContact.vue';
import NewContact from '../views/NewContact.vue';

import NewDropID from '../views/NewDropID.vue';
import ManageDropID from '../views/ManageDropID.vue';

import AdminHome from '../views/admin/AdminHome.vue';
import Users from '../views/admin/Users.vue';
import AdminSettings from '../views/admin/AdminSettings.vue';

const routes: Array<RouteRecordRaw> = [
	{
		path: '/',
		name: 'home',
		component: Home,
		meta: {
			title: "Home"
		}
	},{
		path: '/user',
		name: 'user',
		component: User,
		meta: {
			title: "User"
		}
	},{
		path: '/appspace',
		name: 'appspaces',
		component: Appspaces,
		meta: {
			title: "Appspaces"
		}
	},{
		path: '/appspace/:appspace_id',
		name: 'manage-appspace',
		component: ManageAppspace,
		props: route => {
			return {
				appspace_id: appspaceIdParam(route)
			}
		},
		meta: {
			title: "Manage Appspace"
		}
	},{
		path: '/appspace/:appspace_id/migrate',
		name: 'migrate-appspace',
		component: MigrateAppspace,
		props: route => {
			let v = '';
			if( route.query['to_version'] && !Array.isArray(route.query['to_version']) ) v = route.query['to_version'];
			let j = undefined;
			if( route.query['job_id'] && !Array.isArray(route.query['job_id']) ) j = Number(route.query['job_id']);
			return {
				appspace_id: appspaceIdParam(route),
				to_version: v,
				migrate_only: !!route.query['migrate_only'],
				job_id: j
			}
		},
		meta: {
			title: "Migrate Appspace"
		}
	},{
		path: '/appspace/:appspace_id/restore',
		name: 'restore-appspace',
		component: RestoreAppspace,
		props: route => {
			return {
				appspace_id: appspaceIdParam(route)
			}
		},
		meta: {
			title: "Restore Appspace"
		}
	},{
		path: '/appspace/:appspace_id/new-user',
		name: 'appspace-new-user',
		component: ManageAppspaceUser,
		props: route => {
			return {
				appspace_id: appspaceIdParam(route)
			}
		},
		meta: {
			title: "Add Appspace User"
		}
	},{
		path: '/appspace/:appspace_id/user/:proxy_id',
		name: 'appspace-user',
		component: ManageAppspaceUser,
		props: route => {
			return {
				appspace_id: appspaceIdParam(route),
				proxy_id: proxyIdParam(route)
			}
		},
		meta: {
			title: "Edit Appspace User"
		}
	},{
		path: '/new-appspace/',
		name: 'new-appspace',
		component: NewAppspace,
		props: route => {
			let app_id;
			const a = route.query['app_id'];
			if( Array.isArray(a) ) throw new Error("app_id can not be an array");
			if( a ) app_id = parseInt(a as string);
			let v = '';
			if( route.query['version'] && !Array.isArray(route.query['version']) ) v = route.query['version'];
			return {
				app_id,
				version: v
			};
		},
		meta: {
			title: "New Appspace"
		}
	},{
		path: '/remote-appspace/:domain',
		name: 'manage-remote-appspace',
		component: ManageRemoteAppspace,
		props: true,
		meta: {
			title: "Manage Remote Appspace"
		}
	},{
		path: '/new-remote-appspace/',
		name: 'new-remote-appspace',
		component: NewRemoteAppspace,
		meta: {
			title: "New Remote Appspace"
		}
	},{
		path: '/app',
		name: 'apps',
		component: Apps,
		meta: {
			title: "Apps"
		}
	},{
		path: '/app/:id',
		name: 'manage-app',
		component: ManageApp,
		props: route => {
			return {
				app_id: appIdParam(route)
			}
		},
		meta: {
			title: "Manage App"
		}
	},{
		path: '/app/:id/new-version',
		name: 'new-app-version',
		component: NewApp,
		props: route => {
			return {
				app_id: appIdParam(route)
			}
		},
		meta: {
			title: "New App Version"
		}
	},{
		path: '/app/:id/new-version/:app_get_key',
		name: 'new-app-version-in-process',
		component: NewAppInProcess,
		props: route => {
			return {
				app_id: appIdParam(route),
				app_get_key: route.params.app_get_key
			}
		},
		meta: {
			title: "Processing App Version"
		}
	},{
		path: '/new-app',
		name: 'new-app',
		component: NewApp,
		meta: {
			title: "New App"
		}
	},{
		path: '/new-app/:app_get_key',
		name: 'new-app-in-process',
		component: NewAppInProcess,
		props: true,
		meta: {
			title: "Processing New App"
		}
	},{
		path: '/contact',
		name: 'contacts',
		component: Contacts,
		meta: {
			title: "Contacts"
		}
	},{
		path: '/contact/:contact_id',
		name: 'manage-contact',
		component: ManageContact,
		meta: {
			title: "Edit Contact"
		}
	},{
		path: '/contact-new',
		name: 'new-contact',
		component: NewContact,
		meta: {
			title: "New Contact"
		}
	},{
		path: '/dropid-new',
		name: 'new-dropid',
		component: NewDropID
	},{
		path: '/dropid',
		name: 'dropid',
		component: ManageDropID,
		meta: {
			title: "Manage DropID"
		}
	},{
		path: '/admin',
		name: 'admin',
		component: AdminHome,
		meta: {
			title: "Admin - Home"
		}
	},{
		path: '/admin/users',
		name: 'admin-users',
		component: Users,
		meta: {
			title: "Admin - Users"
		}
	},{
		path: '/admin/settings',
		name: 'admin-settings',
		component: AdminSettings,
		meta: {
			title: "Admin - Settings"
		}
	}
];

export function appspaceIdParam(route:RouteLocationNormalized) :number {
	const p = route.params.appspace_id;
	if( Array.isArray(p) ) throw new Error("id can not be an array");
	return parseInt(p as string);
}
export function appIdParam(route:RouteLocationNormalized) :number {
	const p = route.params.id;
	if( Array.isArray(p) ) throw new Error("id can not be an array");
	return parseInt(p as string);
}
function proxyIdParam(route:RouteLocationNormalized) :string {
	const p = route.params.proxy_id;
	if( Array.isArray(p) ) throw new Error("proxy id can not be an array");
	return p+'';
}

const router = createRouter({
	history: createWebHistory(import.meta.env.BASE_URL),
	routes
});

router.beforeEach((to, _, next) => {
	let title = '';
	const rt = to.meta.title
	if( typeof rt === 'string' ) {
		title = rt + ' - ';
	}
	title += 'Dropserver';
	document.title = title;
	next();
});

export default router;
