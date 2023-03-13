<script setup lang="ts">
import { ref } from 'vue';
import type {AdminInvite} from '@/stores/types';
import { useAdminUserInvitesStore } from '@/stores/admin/user_invitations';

const props = defineProps<{
	invite: AdminInvite
}>();

const deleting = ref(false);

async function delClicked() {
	if( confirm("Delete invitation for "+props.invite.email+"?") ) {
		const {deleteInvitation} = useAdminUserInvitesStore();
		deleting.value = true;
		await deleteInvitation(props.invite.email);
	}
}

</script>

<template>
	<div
		class="bg-white border-b border-b-gray-300 flex justify-between items-baseline px-4">
		<span class="py-2">{{invite.email}}</span>
		<span v-if="deleting" class="text-xs text-gray-500">deleting...</span>
		<button v-else class="btn" @click="delClicked">delete</button>
	</div>
</template>