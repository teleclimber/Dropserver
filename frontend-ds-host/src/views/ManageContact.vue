<template>
	<ViewWrap>
		<template v-if="contact.loaded">
			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Contact</h3>
				</div>
				<div class="py-5">
					<DataDef field="Contact Name:">
						{{contact.name}}
					</DataDef>
					<DataDef field="Display Name:">
						{{contact.display_name}}
					</DataDef>
				</div>
			</div>
			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Authentication</h3>
				</div>
				<div class="py-5">

				</div>
			</div>

			<!-- list contact's status in appspaces, and list contact's appspaces user participate in -->
		</template>
		<BigLoader v-else></BigLoader>
	</ViewWrap>
</template>


<script lang="ts">
import {useRoute} from 'vue-router';
import { defineComponent , onMounted, reactive } from 'vue';

import {Contact} from '../models/contacts';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import DataDef from '../components/ui/DataDef.vue';
import ContactListItem from '../components/ContactListItem.vue';

export default defineComponent({
	name: 'ManageContact',
	components: {
		ViewWrap,
		BigLoader,
		DataDef,
		ContactListItem
	},
	setup() {
		const route = useRoute();
		const contact = reactive(new Contact);
		onMounted( async () => {
			const contact_id = Number(route.params.contact_id);
			await contact.fetch(contact_id);
		});
		return {contact}
	}
});
</script>