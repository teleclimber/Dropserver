import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';
import Home from '../views/Home.vue';
import Appspaces from '../views/Appspaces.vue';
import ManageAppspace from '../views/ManageAppspace.vue';
import Apps from '../views/Apps.vue';

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
	}
];

const router = createRouter({
	history: createWebHistory(process.env.BASE_URL),
	routes
});

export default router;
