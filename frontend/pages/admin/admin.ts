import Vue from "vue";
import Admin from "../../components/admin-page/AdminPage.vue";

import AdminVM from "../../vms/admin-page/admin-root-vm";

import { runInAction } from 'mobx';

import '../style.css';

declare global {
    interface Window { A: Vue; }
}

window.A = new Vue({
	el: '#app',
	render: h => h(Admin),
	provide: runInAction( function() {
		return { 
			vm: new AdminVM()
		}
	})
});