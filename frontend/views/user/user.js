import Vue from 'vue';
import User from './User.vue';
import vue_model from './user-vm.js';

import '../style.css';

new Vue({
	el: '#app',
	data: function() { return vue_model; },
	render: h => h(User)
});

