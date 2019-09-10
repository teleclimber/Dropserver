import Vue from 'vue';
import User from './User.vue';
import vue_model from './user-vm.js';

import '../style.css';

new Vue({
	el: '#app',
	provide: {
		user_vm: vue_model,
		applications_vm: vue_model.applications_vm,
		app_spaces_vm: vue_model.app_spaces_vm
	},
	render: h => h(User)
});

