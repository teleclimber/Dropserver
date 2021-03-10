<template>
	<ViewWrap>
		<router-link to="contact-new" class="btn btn-blue">New Contact</router-link>
		<ContactListItem v-for="contact in contacts.asArray" :key="contact.contact_id" :contact="contact"></ContactListItem>
		<BigLoader v-if="!contacts.loaded"></BigLoader>
		<MessageSad v-else-if="contacts.asArray.length === 0" head="No Contacts" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			There are no contacts yet. Invite some friends!
		</MessageSad>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent , onMounted, reactive } from 'vue';

import {Contacts} from '../models/contacts';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import ContactListItem from '../components/ContactListItem.vue';

export default defineComponent({
	name: 'Contacts',
	components: {
		ViewWrap,
		BigLoader,
		MessageSad,
		ContactListItem
	},
	setup() {
		const contacts = reactive(new Contacts);
		onMounted( async () => {
			await contacts.fetchForOwner();
		});
		
		return {contacts}
	}
});
</script>