<template>
	<ViewWrap>
		<MessageSad v-if="not_found" head="Not Found" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			This DropID does not exist
		</MessageSad>
		<div v-else-if="loaded" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Identity</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">This can not be changed. Create a new Dropid instead.</p>
			</div>
			<div class="py-5">
				<DataDef field="Domain Name:">
					{{dropid.domain_name}}
				</DataDef>
				<DataDef field="Handle:">
					{{dropid.handle}}
				</DataDef>
			</div>

			<div class="py-5 border-t border-gray-200">
				<DataDef field="Display Name:">
					{{dropid.display_name}}
					<!-- TODO make it so display name can be edited -->
					<input type="text" v-model="display_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
				</DataDef>
			</div>	
			
		</div>
		<BigLoader v-else></BigLoader>
	</ViewWrap>
</template>


<script lang="ts">
import { defineComponent, ref, reactive, watchEffect, onMounted, onUnmounted } from 'vue';
import type { Ref } from 'vue';

import {useRoute} from 'vue-router';
import router from '../router';

import {setTitle, unsetTitle} from '../controllers/nav';

import {DomainNames} from '../models/domainnames';
import {DropID, DropIDs} from '../models/dropids';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import DataDef from '../components/ui/DataDef.vue';

export default defineComponent({
	name: 'ManageDropID',
	components: {
		ViewWrap,
		MessageSad,
		BigLoader,
		DataDef
	},
	setup() {
		const route = useRoute();
		const not_found = ref(false);
		const loaded = ref(false);
		
		const dropid = ref(new DropID);
		const dropids = reactive( new DropIDs);
		dropids.fetchForOwner();

		const handle = ref('');
		const domain_name = ref('');

		watchEffect( () => {
			if(domain_name.value == '' || !dropids.loaded) return;
			const d = dropids.dropids.find( (d) => d.handle == handle.value && d.domain_name == domain_name.value );
			if( d === undefined ) {
				not_found.value = true;
			}
			else {
				loaded.value = true;
				dropid.value = d;
			}
		});

		// domain... How do we get the list of domains that are usable for this?
		const domain_names = reactive( new DomainNames );
		domain_names.fetchForOwner();

		const display_name = ref("");

		onMounted( () => {
			handle.value = typeof route.query.handle === 'string' ? route.query.handle : '';
			domain_name.value = typeof route.query.domain === 'string' ? route.query.domain : '';

			if( domain_name.value === '' ) {
				loaded.value = true;
				not_found.value = true;
				setTitle("Manage DropID");
			}
			else {
				setTitle(handle.value + ' at ' + domain_name.value);
			}
		});
		onUnmounted( () => {
			unsetTitle();
		});

		return {
			loaded, not_found,
			dropid,
			domain_names: domain_names.domains,
			display_name,
		}
	}
});
</script>