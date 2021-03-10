<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Identity</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">You will share this domain and handle with others.</p>
			</div>
			<div class="py-5">
				<DataDef field="Domain Name:">
					<select v-model="domain_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
						<option value="">Pick Domain Name</option>
						<option v-for="d in domain_names.for_dropid" :key="d.domain_name" :value="d.domain_name">{{d.domain_name}}</option>
					</select>
				</DataDef>
				<DataDef field="Handle:">
					<input type="text" v-model="handle" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					<p>valid: {{handle_valid}}</p>
				</DataDef>
			</div>

			<div class="py-5 border-t border-gray-200">
				<DataDef field="Display Name:">
					<input type="text" v-model="display_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
				</DataDef>
			</div>	
			<div class="flex justify-between px-4 py-5 sm:px-6 border-t border-gray-200">
				<router-link to="user" class="btn py-2">Cancel</router-link>
				<button @click="save" class="btn-blue">Create DropID</button>
			</div>
		</div>
	</ViewWrap>
</template>


<script lang="ts">
import { defineComponent, ref, reactive, watch, onMounted, onUnmounted } from 'vue';
import type { Ref } from 'vue';

import router from '../router';

import {setTitle, unsetTitle} from '../controllers/nav';

import {DomainNames} from '../models/domainnames';
import {checkHandle, createDropID} from '../models/dropids';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';

export default defineComponent({
	name: 'NewDropID',
	components: {
		ViewWrap,
		DataDef
	},
	setup() {
		// domain... How do we get the list of domains that are usable for this?
		const domain_names = reactive( new DomainNames );
		domain_names.fetchForOwner();

		const domain_name = ref("")
		const handle = ref("");
		const display_name = ref("");

		// handle_valid values:
		// '': not entered yet
		// required: blank but must be specified
		// checking: talking to server
		// unavailable: not available for that domain
		// long: too many characters
		// invalid: bad characters (don't know which yet?)
		// ok: check and is valid
		const handle_valid = ref('');

		onMounted( () => {
			setTitle("Add DropID");
		});
		onUnmounted( () => {
			unsetTitle();
		});

		watch( [domain_name, handle], async () => {
			if( domain_name.value === '' ) {
				handle_valid.value = '';
				return;
			}
			if( handle.value === "" ) {
				handle_valid.value = '';
				return;
			}
			if( handle.value.length > 100 ) {
				handle_valid.value = 'long';
				return;
			}
			// check for bad chars, whatever they are.

			// Here we query the server to see if the id already exists.
			// Note this is a pretty poor way to do this.
			if( await checkHandle(handle.value, domain_name.value) ) {
				handle_valid.value = 'ok';
			}
			else {
				handle_valid.value = 'unavailable';
			}

		});

		async function save() {
			handle.value = handle.value.trim();
			display_name.value = display_name.value.trim();

			if( domain_name.value == '' ) {
				alert("Please pick a domain name");
				return;
			}

			if( handle.value.length === 0 || handle.value.length > 100 ||
				display_name.value.length > 100 ) {
					alert("Names should not be blank or longer than 100 characters.");
					return;
			}

			const dropid = await createDropID(handle.value, domain_name.value, display_name.value);
			// ^^ How are actually going to handle handles being already taken?
			router.push({name: 'user'});
		}

		return {
			domain_names,
			domain_name,
			handle, handle_valid,
			display_name,
			save
		}
	}
});
</script>