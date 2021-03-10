<template>
	<ViewWrap>
		<p v-if="!admin_settings.loaded">Loading...</p>
		<p v-else>New user registration: {{ admin_settings.registration_open ? 'open to public' : 'invitation only' }} <button @click="changeRegOpen">Change</button></p>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, reactive } from 'vue';

import {AdminSettings} from '../../models/admin_settings';

import ViewWrap from '../../components/ViewWrap.vue';

export default defineComponent({
	name: 'AdminSettings',
	components: {
		ViewWrap
	},
	setup() {
		const admin_settings = reactive(new AdminSettings);
		admin_settings.fetch();

		async function changeRegOpen() {
			await admin_settings.setRegistrationOpen(!admin_settings.registration_open);
		}

		return {
			admin_settings,
			changeRegOpen
		}
	}
});
</script>
