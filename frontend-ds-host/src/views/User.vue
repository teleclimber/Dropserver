<template>
	<ViewWrap>
		<h2>Email:</h2>
		<p v-if="show_change_email">
			Email: <input type="text" ref="email_input" v-model="email">
			<button @click="cancelChangeEmail">Cancel</button>
			<button @click="saveChangeEmail">Save</button>
		</p>
		<p v-else>Email: {{user.email}} <button @click="openChangeEmail">Change</button></p>

		<h2>Password:</h2>
		<p>Not implemented...</p>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, Ref, reactive, nextTick } from 'vue';

import user from '../models/user';

import ViewWrap from '../components/ViewWrap.vue';


export default defineComponent({
	name: 'User',
	components: {
		ViewWrap,
	},
	setup() {

		const show_change_email = ref(false);
		const email_input :Ref<HTMLInputElement|null> = ref(null);
		const email = ref("");

		function openChangeEmail() {
			show_change_email.value = true; 
			email.value = user.email;
			nextTick( () => {
				if( email_input.value === null ) return;
				email_input.value.focus();
			});
		}
		function cancelChangeEmail() {
			show_change_email.value = false;
		}
		function saveChangeEmail() {
			// TODO
		}

		return {
			user,
			show_change_email,
			openChangeEmail,
			cancelChangeEmail,
			saveChangeEmail,
			email_input,
			email,

		}
	}
});
</script>
