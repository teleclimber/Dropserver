import Vue from 'vue';
import Admin from './Admin.vue';

import AdminUI from "./vms/admin-root-vm";

import { runInAction } from 'mobx';

import '../style.css';

window.A = new Vue({
	el: '#app',
	data: runInAction( function() {
		return { 
			vm: new AdminUI(),
		}
	}),
	render: h => h(Admin)
});