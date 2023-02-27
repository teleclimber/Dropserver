<template>
	<div class="absolute top-0 left-0 w-full z-10 h-screen" @click="closeUserMenu">
		<div class="origin-top-right absolute right-0 mt-20 mr-4 w-56 rounded-md shadow-lg bg-white ring-1 ring-black ring-opacity-5 divide-y divide-gray-100">
			<router-link 
				:to="{name:'user'}" v-if="authUserStore.is_loaded"
				class="block px-4 py-4 text-gray-700 hover:bg-gray-100 hover:text-gray-900"
				>{{authUserStore.email}}</router-link>
			<router-link
				to="/admin"
				v-if="authUserStore.is_admin && !in_admin"
				class="block px-4 py-4 text-gray-700 hover:bg-gray-100 hover:text-gray-900"
				>Instance Administation</router-link>
			<router-link
				to="/" v-if="in_admin"
				class="block px-4 py-4 text-gray-700 hover:bg-gray-100 hover:text-gray-900"
				>User Home</router-link>
			<a href="/logout" class="block px-4 py-4 text-gray-700 hover:bg-gray-100 hover:text-gray-900">Log Out</a>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, computed } from 'vue';
import {useRoute} from 'vue-router';

import { useAuthUserStore } from '@/stores/auth_user';

import { closeUserMenu } from '../controllers/nav';

export default defineComponent({
	name: 'UserDropDownMenu',
	components: {
		
	},
	setup() {
		const authUserStore = useAuthUserStore();
		authUserStore.fetch();

		const route = useRoute();
		const in_admin = computed( () => {	// this is duplicated from NavMain, but useRoute only works inside setup.
			return route.path.startsWith('/admin');
		});

		return {
			authUserStore,
			closeUserMenu,
			in_admin,
		}
	}

});
</script>