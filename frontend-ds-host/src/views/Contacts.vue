<script lang="ts" setup>
import { onMounted, reactive } from 'vue';

import {Contacts} from '../models/contacts';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import UnderConstruction from '@/components/ui/UnderConstruction.vue';
import ContactListItem from '../components/ContactListItem.vue';

const contacts = reactive(new Contacts);
onMounted( async () => {
	await contacts.fetchForOwner();
});
		
</script>

<template>
	<ViewWrap>
		<div class="m-4 md:m-0 md:mb-6 ">
			<router-link to="contact-new" class="btn btn-blue">New Contact</router-link>
		</div>
		<UnderConstruction head="Contacts Section Not Yet Functional" class="my-6">
			The contacts section is not functional. You can add contacts at a basic level, but can not use them anywhere in the system.
			Kinda pointless for now.
			Leaving it here to show it's coming.
		</UnderConstruction>
		<ContactListItem v-for="contact in contacts.asArray" :key="contact.contact_id" :contact="contact"></ContactListItem>
		<BigLoader v-if="!contacts.loaded"></BigLoader>
		<!-- <MessageSad v-else-if="contacts.asArray.length === 0" head="No Contacts" class="mx-4 sm:mx-0 my-6 sm:rounded-xl shadow">
			There are no contacts yet. Invite some friends!
		</MessageSad> -->
	</ViewWrap>
</template>
