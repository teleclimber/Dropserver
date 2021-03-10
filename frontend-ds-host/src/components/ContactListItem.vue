<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 flex justify-between">
			<div>
				<h3 class="text-2xl leading-6 font-medium text-gray-900">{{contact.name}}</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">
					Created: {{contact.created_dt.toLocaleString()}}
				</p>
			</div>
			<div>
				<router-link :to="{name:'manage-contact', params:{contact_id:contact.contact_id}}" class="btn-blue">Edit</router-link>
			</div>
		</div>
		<div class="border-t border-gray-200">

		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';

import type {Contact} from '../models/contacts';

export default defineComponent({
	name: 'ContactListItem',
	components: {
		
	},
	props: {
		contact: {
			type: Object as PropType<Contact>,
			required: true
		}
	}
	
});
</script>