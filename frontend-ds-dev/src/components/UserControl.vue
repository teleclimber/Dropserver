<template>
	<div class="flex my-4 items-stretch">
		<select v-model="type">
			<option>owner</option>
			<option>user</option>
			<option>public</option>
		</select>
			
		<button @click="setUser()">Set User</button>
	</div>
			
</template>


<script lang="ts">
import { defineComponent, reactive } from 'vue';

import userData from '../models/user-data';

// basically radio-ish buttons to select:
// - public
// - authorized (set display name?)
// - permissions (based on apps' declared permissions eventually)
// - owner



export default defineComponent({
	name: 'UserControl',
	data() {
		return {
			type: 'owner',
			display_name: 'name',
			proxy_id: 'abc'
		};
	},
	components: {
	},
	setup(props, context) {
		return { userData };
	},
	methods: {
		setUser() {
			this.userData.setUser({
				type: this.type,
				permissions: [],
				display_name: this.display_name,
				proxy_id: this.proxy_id
			});
		}
	}
});
</script>