<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex items-baseline justify-between">
				<h3 class="text-lg leading-6 font-medium text-gray-900">
					{{ proxy_id ? "Manage Appspace User" : "New Appspace User" }}
				</h3>
				<div class="flex items-stretch">
					<router-link class="btn" :to="{name:'manage-appspace', params:{id:appspace.id}}">back to appspace</router-link>
				</div>
			</div>
			<div v-if="proxy_id" class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
				<div>{{user.auth_id}}</div>
				<div>[add to contacts / see in contacts]</div>
				<!-- Need: delete, block, change auth -->
			</div>
			<div v-else class="py-5 border-b border-gray-200">
				<DataDef field="Add Using:">
					<select v-model="add_using" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
						<option value="contact">Pick From Contacts</option>
						<option value="dropid">Enter DropID</option>
						<option value="email">Enter Email</option>
					</select>
				</DataDef>
				<DataDef v-if="add_using === 'contact'" field="Contact:">
					<select v-model="contact_id" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
						<option value="contact">Pick From Contacts</option>
					</select>
				</DataDef>
				<DataDef v-if="add_using === 'dropid'" field="DropID:">
					<input type="text" v-model="drop_id" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
				</DataDef>
				<DataDef v-if="add_using === 'email'" field="Email:">
					<input type="text" v-model="email" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
				</DataDef>
			</div>
			<div class="py-5 border-b border-gray-200">
				<DataDef field="Display Name:">
					<input type="text" v-model="display_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
				</DataDef>
				<DataDef field="Permissions:">
					[Permisssions to be implemented]
				</DataDef>
			</div>
			<div class="py-5 px-4 sm:px-6 flex items-baseline justify-between">
				<router-link class="btn" :to="{name:'manage-appspace', params:{id:appspace.id}}">back to appspace</router-link>
				<button class="btn-blue" @click="save">Save</button>
			</div>
		</div>
	</ViewWrap>
</template>

<script lang="ts">
import {useRoute} from 'vue-router';
import router from '../router/index';
import { defineComponent, ref, Ref, reactive, computed, onMounted, onUnmounted, PropType } from 'vue';

import {setTitle} from '../controllers/nav';

import {Appspace} from '../models/appspaces';
import {AppspaceUser, saveNewUser, updateUserMeta} from '../models/appspace_users';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';

export default defineComponent({
	name: 'ManageAppspaceUser',
	components: {
		ViewWrap,
		DataDef
	},
	setup() {
		const route = useRoute();
		const appspace = reactive( new Appspace );
		const user = reactive( new AppspaceUser );

		const proxy_id :Ref<string|undefined> = ref("");

		const add_using = ref("contact");
		const contact_id = ref(0);
		const drop_id = ref("");
		const email = ref("");

		const display_name = ref("");

		onMounted( async () => {
			const appspace_id = Number(route.params.id);
			await appspace.fetch(appspace_id);
			setTitle(appspace.domain_name);
			
			const route_proxy_id = route.params.proxy_id;
			if( Array.isArray(route_proxy_id) ) return;
			if( !route_proxy_id ) return;
			proxy_id.value = route_proxy_id;

			await user.fetch(appspace_id, proxy_id.value);
			// fill in variables
			display_name.value = user.display_name;
			// permissions...
		});
		onUnmounted( async () => {
			setTitle("");
		});

		async function save() {
			if( proxy_id.value ) {
				// update
				await updateUserMeta(appspace.id, proxy_id.value, {
					display_name: display_name.value,
					permissions: []
				});
			}
			else {
				let auth_type = "";
				let auth_id = "";
				if( add_using.value === 'contact' ) {
					// handle taht
				}
				else if( add_using.value === 'email' ) {
					auth_type = 'email';
					auth_id = email.value;
				}
				else if( add_using.value === 'dropid' ) {
					auth_type = 'dropid';
					auth_id = drop_id.value;
				}
				else throw new Error("what is this add using? "+add_using.value);
				
				await saveNewUser(appspace.id, {
					auth_type,
					auth_id,
					display_name: display_name.value,
					permissions: []
				});
			}
			router.push({name: 'manage-appspace', params:{id: appspace.id}});
		}

		return {
			appspace,
			proxy_id,
			user,
			add_using,
			contact_id,
			drop_id,
			email,
			display_name,
			save,
		}
	}

});
</script>