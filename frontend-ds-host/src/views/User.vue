<script setup lang="ts">
import { ref, reactive } from 'vue';

import { useAuthUserStore} from '@/stores/auth_user';

import {DomainNames} from '../models/domainnames';
import {DropIDs} from '../models/dropids';

import DropIDFull from '../components/DropIDFull.vue';
import ViewWrap from '../components/ViewWrap.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import DataDef from '../components/ui/DataDef.vue';
import ChangeEmail from '@/components/user/ChangeEmail.vue';

const authUserStore = useAuthUserStore();
authUserStore.fetch();

const show_change_email = ref(false);

function openChangeEmail() {
	// check if other things aren't open first...
	show_change_email.value = true; 
}

const domains = reactive( new DomainNames);
domains.fetchForOwner();

const dropids = reactive(new DropIDs);
dropids.fetchForOwner();

</script>

<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Account</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">Use this email and password to log in to this Dropserver account. Do not share these credentials.</p>
			</div>
			<div class="py-5">
				<DataDef field="Email:">
					<ChangeEmail v-if="show_change_email" @close="show_change_email = false"></ChangeEmail>
					<div v-else class="flex justify-between">
						<span>{{authUserStore.email || '...'}}</span>
						<button class="btn" @click="openChangeEmail">Change</button>
					</div>
				</DataDef>
				<DataDef field="Password:">
					<p>********</p>
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
