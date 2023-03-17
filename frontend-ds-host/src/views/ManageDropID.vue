<script lang="ts" setup>
import { ShallowRef, Ref, ref, watchEffect, onMounted, onUnmounted, computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { setTitle, unsetTitle } from '../controllers/nav';

import { useDropIDsStore } from '@/stores/dropids';
import { UserDropID } from '@/stores/types';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import DataDef from '../components/ui/DataDef.vue';

const route = useRoute();
const not_found = ref(false);
const loaded = ref(false);

const router = useRouter();

const dropIDsStore = useDropIDsStore();
dropIDsStore.loadData();

const dropid :ShallowRef<UserDropID|undefined> = ref();

const display_name = ref("");
const display_name_input :Ref<HTMLInputElement|undefined> = ref();
watch( display_name_input, () => {
	if( display_name_input.value !== undefined ) display_name_input.value.focus();
});

watchEffect( () => {
	if( !dropIDsStore.is_loaded ) return;
	const handle = typeof route.query.handle === 'string' ? route.query.handle : '';
	const domain_name = typeof route.query.domain === 'string' ? route.query.domain : '';
	if( domain_name == '' ) return;
	const d = dropIDsStore.getDropID(domain_name, handle);
	if( d === undefined ) {
		not_found.value = true;
	}
	else {
		loaded.value = true;
		dropid.value = d.value;
		display_name.value = d.value.display_name;
	}
});

onMounted( () => {
	setTitle("Manage DropID");
});
onUnmounted( () => {
	unsetTitle();
});

async function save() {
	if( dropid.value === undefined ) return;
	if( display_name.value.length > 29 ) {
		alert("Display name is too long");
		return;
	}
	const dn = display_name.value.trim();
	if( dn === dropid.value.display_name ) {	// don't save if unchnaged. Server side will interpret as "not found" bc no rows will change
		cancel();
		return;
	}

	await dropIDsStore.updateDropID(dropid.value.handle, dropid.value.domain_name, dn);

	router.back();
}

function cancel() {
	router.back();
}

</script>

<template>
	<ViewWrap>
		<MessageSad v-if="not_found" head="Not Found" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			This DropID does not exist
		</MessageSad>
		<div v-else-if="loaded" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">{{ dropid?.compound_id }}</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">The domain and handle can not be changed.</p>
			</div>
			<form @submit.prevent="save" @keyup.esc="cancel">
				<div class="py-5 border-t border-gray-200">
					<DataDef field="Display Name:">
						<input type="text" v-model="display_name" ref="display_name_input" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					</DataDef>
				</div>	
				<div class="flex justify-between px-4 py-5 sm:px-6 border-t border-gray-200">
					<input type="button" class="btn py-2" @click="cancel" value="Cancel" />
					<input
						type="submit"
						class="btn-blue"
						value="Save" />
				</div>
			</form>
		</div>
		<!-- Maybe list of where this drop id is used? -->
		<!-- There should also be a way to delete the dropid if it is not being used anywhere -->
		
		<BigLoader v-if="!loaded && !not_found"></BigLoader>
	</ViewWrap>
</template>

