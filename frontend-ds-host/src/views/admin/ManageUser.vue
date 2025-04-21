<script setup lang="ts" >
import { ref, Ref, computed, onMounted, nextTick } from 'vue';

import { useAdminAllUsersStore } from '@/stores/admin/all_users';
import { useAdminTSNetStore } from '@/stores/admin/tsnet';
import type { TSNetPeerUser } from '@/stores/types';

import BigLoader from '@/components/ui/BigLoader.vue';
import DataDef from '@/components/ui/DataDef.vue';
import ViewWrap from '../../components/ViewWrap.vue';

const props = defineProps<{
	user_id: number
}>();

const adminTSNetStore = useAdminTSNetStore();
const adminUsersStore = useAdminAllUsersStore();

onMounted( () => {
	adminTSNetStore.loadTSNetStatus();
	adminTSNetStore.loadTSNetPeerUsers();
	adminUsersStore.fetch();
});

const user = computed( () => {
	const u = adminUsersStore.users.get(props.user_id);
	if( u ) return u.value;
	return undefined;
});

// need to create a list of available peer users to offer up for association.
const tsnet_peer_unmatched_users = computed( () => {
	return adminTSNetStore.peer_users.filter( (pu) => {
		return !adminUsersStore.users.values().find( u => u.value.tsnet_identifier === pu.full_id );
	});
});

const tsnet_input_elem :Ref<HTMLInputElement|undefined> = ref();
const tsnet_input_value = ref("");
const show_change_tsnet = ref(false);
function showChangeTSNet() {
	tsnet_input_value.value = "";
	show_change_tsnet.value = true;
	nextTick( () => {
		tsnet_input_elem.value?.focus();
	});
}
async function saveTSNet() {
	if( tsnet_input_elem.value === undefined ) throw new Error("no input element for tsnet id");
	if( tsnet_input_value.value === "" ) await adminUsersStore.deleteTSNet(props.user_id);
	else await adminUsersStore.updateTSNet(props.user_id, tsnet_input_value.value);
	show_change_tsnet.value = false;
}

</script>

<template>
	<ViewWrap>
		<div v-if="user" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Email & Password</h3>
			</div>
			<div class="p-4 sm:px-6">
				<DataDef field="Email:">
					<span v-if="user.email">{{ user.email }}</span>
					<span v-else class="text-gray-500 italic">No email set</span>
				</DataDef>
				<DataDef field="Password:">
					<span v-if="user.has_password">**********</span>
					<span v-else class="text-gray-500 italic">No password set</span>
				</DataDef>
			</div>
		</div>
		<div v-if="user" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Tailscale</h3>
			</div>
			<div class="p-4 sm:px-6">
				<template  v-if="show_change_tsnet && adminTSNetStore.tsnet_status.usable">
					<form @submit.prevent="saveTSNet" @keyup.esc="show_change_tsnet = false">
						<DataDef field="TS Network User:">
							<select ref="tsnet_input_elem" v-model="tsnet_input_value">
								<option value="">No tsnet user</option>
								<option v-for="pu in tsnet_peer_unmatched_users" :value="pu.id">{{ pu.display_name }} ({{ pu.login_name }})</option>
							</select>
						</DataDef>
						<div class="flex justify-between">
							<input type="button" class="btn" @click="show_change_tsnet = false" value="Cancel" />
							<input
								type="submit"
								class="btn-blue"
								value="Save" />
						</div>
					</form>
				</template>
				<template v-else-if="show_change_tsnet">
					tsnet node not connected. Connect it to change.
				</template>
				<template class="p-4 sm:px-6" v-else-if="user.tsnet_identifier">
					<DataDef field="Tsnet ID:">
						{{ user.tsnet_identifier }}
						<button class="btn" @click.stop.prevent="showChangeTSNet">change</button>
					</DataDef>
					<DataDef field="Login Name:">{{ user.tsnet_extra_name }}</DataDef>
				</template>
				<template v-else-if="adminTSNetStore.tsnet_status.usable">
					No tsnet id set for this user
					<button class="btn" @click.stop.prevent="showChangeTSNet">change</button>
				</template>
				<template v-else>
					tsnet node not create or not connected. Connect ds-host to a tailsclae network.
				</template>
			</div>
		</div>
		<BigLoader v-if="!user"></BigLoader>
	</ViewWrap>
</template>