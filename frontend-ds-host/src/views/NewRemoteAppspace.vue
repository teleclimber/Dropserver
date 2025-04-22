<script lang="ts" setup>
import { ref, Ref, watchEffect, onMounted } from 'vue';
import { useRouter } from 'vue-router';

import { useRemoteAppspacesStore } from '@/stores/remote_appspaces';
import type { RemoteAppspacePostResp } from '@/stores/remote_appspaces';
import { useDropIDsStore } from '@/stores/dropids';
import type { UserDropID } from '@/stores/types';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';

const router = useRouter();

const remoteAppspacesStore = useRemoteAppspacesStore();
onMounted( () => {
	remoteAppspacesStore.loadData();
});

const dropIDsStore = useDropIDsStore();
dropIDsStore.loadData();

const dropid :Ref<UserDropID|undefined> = ref();

watchEffect( () => {
	if( dropIDsStore.dropids.size !== 0 ) {
		dropid.value = dropIDsStore.dropids.entries().next().value![1].value;
	}
});

const domain_name = ref("");

const post_resp:Ref<RemoteAppspacePostResp|undefined> = ref()

async function create() {
	if( dropid.value === undefined ) return;

	let dom = domain_name.value.trim();
	if( dom === "" ) return;

	post_resp.value = await remoteAppspacesStore.create( domain_name.value, dropid.value.compound_id );
	if( post_resp.value.inputs_valid ) {
		router.push({name: 'manage-remote-appspace', params:{domain: domain_name.value}});
	}
}

</script>

<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Access Someone Else's Appspace</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">Enter the address of the appspace and the DropID you'd like to use to identify yourself.</p>
			</div>
			<!-- TODO this needs to be a form -->
			<div class="py-5">
				<DataDef field="Appspace Address:">
					<input type="text" v-model="domain_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					<p v-if="post_resp && post_resp.domain_message">{{post_resp.domain_message}}</p>
				</DataDef>
				<DataDef field="Your DropID:">
					<select v-model="dropid" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
						<option value="">Pick DropID</option>
						<option v-for="[_,d] in dropIDsStore.dropids" :key="d.value.compound_id" :value="d.value">{{d.value.compound_id}}</option>
					</select>
				</DataDef>
				<!-- later on also override dropid's default handle and avatar -->
			</div>
			<!-- later on probably have a "check access" or some such thing -->
			<!-- Or maybe it's a mandatory step that also reveals the drop id of the appspace owner for verficiation
			    .. and maybe other things. Then you can save (or asave anyways if you're sure) -->
			<div class="px-4 py-5 sm:px-6 border-t border-gray-200 flex justify-between items-center">
				<router-link class="btn" to="/appspace">cancel</router-link>
				<button @click="create" class="btn-blue">Connect</button>
			</div>
		</div>
	</ViewWrap>
</template>
