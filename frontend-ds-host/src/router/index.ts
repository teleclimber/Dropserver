import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';
import Home from '../views/Home.vue';
import Appspaces from '../views/Appspaces.vue';
import ManageAppspace from '../views/ManageAppspace.vue';
import Apps from '../views/Apps.vue';
import ManageApp from '../views/ManageApp.vue';
import NewAppVersion from '../views/NewAppVersion.vue';
import NewApp from '../views/NewApp.vue';

const routes: Array<RouteRecordRaw> = [
	{
		path: '/',
		name: 'home',
		component: Home
	},{
		path: '/appspace',
		name: 'appspaces',
		component: Appspaces
	},{
		path: '/appspace/:id',
		name: 'manage-appspace',
		component: ManageAppspace
	},{
		path: '/app',
		name: 'apps',
		component: Apps
	},{
		path: '/app/:id',
		name: 'manage-app',
		component: ManageApp
	},{
		path: '/app/:id/new-version',
		name: 'new-app-version',
		component: NewAppVersion
	},{
		path: '/new-app',
		name: 'new-app',
		component: NewApp
	}
];

const router = createRouter({
	history: createWebHistory(process.env.BASE_URL),
	routes
});

export default router;
