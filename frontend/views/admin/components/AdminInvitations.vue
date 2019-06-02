<style scoped>

</style>

<template>
	<component 
		:is="vm.page.is"
		:vm="vm.page"
	></component>
</template>

<script>
import { observer } from "mobx-vue";

import AdminInvitationsList from './AdminInvitationsList.vue';
import AdminInviteNew from './AdminInviteNew.vue';
import AdminUpdateInvitation from './AdminUpdateInvitation.vue';

export default observer({
	name: 'AdminInvitations',
	props: ['vm'],
	components: {
		AdminInvitationsList,
		AdminInviteNew,
		AdminUpdateInvitation
	},
	methods: {
		
	}
});
</script>