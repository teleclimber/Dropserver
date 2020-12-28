import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router';
import Home from '../views/Home.vue';
import Appspaces from '../views/Appspaces.vue';
import Apps from '../views/Apps.vue';

const routes: Array<RouteRecordRaw> = [
	{
		path: '/',
		name: 'Home',
		component: Home
	},{
		path: '/appspace',
		name: 'Appspaces',
		component: Appspaces
	},{
		path: '/app',
		name: 'Apps',
		component: Apps
	}
];

const router = createRouter({
	history: createWebHistory(process.env.BASE_URL),
	routes
});

export default router;
