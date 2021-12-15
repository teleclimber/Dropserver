<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Account</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">Use this email and password to log in to this DropServer account. Do not share these credentials.</p>
			</div>
			<div class="py-5">
				<DataDef field="Email:">
					<div v-if="show_change_email">
						<input type="text" ref="email_input" v-model="email" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
						<div class="flex justify-between pt-2">
							<button class="btn-blue" @click="cancelChangeEmail">Cancel</button>
							<button class="btn-blue" @click="saveChangeEmail">Save</button>
						</div>
					</div>
					<div v-else class="flex justify-between">{{user.email}} <button class="btn" @click="openChangeEmail">Change</button></div>
				</DataDef>
				<DataDef field="Password:">
					<p>Not implemented...</p>
				</DataDef>
			</div>
		</div>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
				<div>
					<h3 class="text-lg leading-6 font-medium text-gray-900">Domains</h3>
					<p class="mt-1 max-w-2xl text-sm text-gray-500">Domains for your appspaces and DropIDs.</p>
				</div>
				<div>
					[Connect Domain]
				</div>
			</div>
			<div class="px-4 py-5 sm:px-6 ">
				<div v-for="domain in domains.domains" :key="'domain-'+domain.domain_name">
					{{domain.domain_name}}
				</div>
			</div>
		</div>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
				<div>
					<h3 class="text-lg leading-6 font-medium text-gray-900">DropIDs</h3>
					<p class="mt-1 max-w-2xl text-sm text-gray-500">Share your DropID with friends to join their appspace.</p>
				</div>
				<div>
					<router-link to="/dropid-new" class="btn">New DropID</router-link>
				</div>
			</div>
			<div class="px-4 py-5 sm:px-6 ">
				<DropIDFull v-for="d in dropids.dropids" :key="d.handle+'@@'+d.domain_name" :dropid="d"></DropIDFull>
				<MessageSad v-if="dropids.loaded && dropids.dropids.length === 0" head="No DropIDs">Create a DropID to interact with other people's DropServers.</MessageSad>
			</div>
		</div>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, Ref, reactive, nextTick } from 'vue';

import user from '../models/user';
import {DomainNames} from '../models/domainnames';
import {DropIDs} from '../models/dropids';

import DropIDFull from '../components/DropIDFull.vue';
import ViewWrap from '../components/ViewWrap.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import DataDef from '../components/ui/DataDef.vue';


export default defineComponent({
	name: 'User',
	components: {
		ViewWrap,
		DataDef,
		MessageSad,
		DropIDFull
	},
	setup() {

		const show_change_email = ref(false);
		const email_input :Ref<HTMLInputElement|null> = ref(null);
		const email = ref("");

		function openChangeEmail() {
			show_change_email.value = true; 
			email.value = user.email;
			nextTick( () => {
				if( email_input.value === null ) return;
				email_input.value.focus();
			});
		}
		function cancelChangeEmail() {
			show_change_email.value = false;
		}
		function saveChangeEmail() {
			// TODO
		}

		const domains = reactive( new DomainNames);
		domains.fetchForOwner();

		const dropids = reactive(new DropIDs);
		dropids.fetchForOwner();

		return {
			user,
			show_change_email,
			openChangeEmail,
			cancelChangeEmail,
			saveChangeEmail,
			email_input,
			email,
			domains,
			dropids
		}
	}
});
</script>
