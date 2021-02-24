<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<DataDef field="Contact Name:">
				<input type="text" v-model="contact_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
			</DataDef>
			<DataDef field="Display Name:">
				<input type="text" v-model="display_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
			</DataDef>
			<div class="flex justify-end px-4 py-5 sm:px-6 border-t border-gray-200">
				<button @click="save" class="btn-blue">Create Contact</button>
			</div>
		</div>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, onMounted } from 'vue';
import type { Ref } from 'vue';

import router from '../router';

import {createContact} from '../models/contacts';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';

export default defineComponent({
	name: 'NewContact',
	components: {
		ViewWrap,
		DataDef
	},
	setup() {
		const contact_name = ref("");
		const display_name = ref("");

		async function save() {
			contact_name.value = contact_name.value.trim();
			display_name.value = display_name.value.trim();

			if( contact_name.value.length === 0 || contact_name.value.length > 100 || 
				display_name.value.length === 0 || display_name.value.length > 100 ) {
					alert("Names should not be blank or longer than 100 characters.");
					return;
			}

			const contact = await createContact(contact_name.value, display_name.value);
			router.push({name: 'manage-contact', params:{contact_id:contact.contact_id}});
		}

		return {
			contact_name,
			display_name,
			save
		}
	}
});
</script>