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
			<div v-if="proxy_id" class="px-4 py-5 sm:px-6 border-b border-gray-200 ">
				<div class="flex justify-between">
					<div>{{user.auth_id}}</div>
					<div>[add to contacts / see in contacts]</div>
				</div>
				<div>
					<p>[Change Auth?]</p>
					<p>Show display name and avatar inherited from auth data (drop id or contact)</p>
				</div>
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
				<h3 class="px-4 sm:px-6 font-bold text-gray-900">Set or Override User Display:</h3>
				<DataDef field="Display Name:">
					<input type="text" v-model="display_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
				</DataDef>
				<DataDef field="Avatar:">
					<Avatar :current="user.avatarURL" @changed="avatarChanged"></Avatar>
				</DataDef>
			</div>
			<div class="py-5 border-b border-gray-200">
				<DataDef field="Permissions:">
					[Permisssions to be implemented]
				</DataDef>
			</div>
			<div class="py-5 px-4 sm:px-6 flex items-baseline justify-between">
				<router-link class="btn" :to="{name:'manage-appspace', params:{id:appspace.id}}">back to appspace</router-link>
				<button class="btn-blue" @click="save">Save</button>
			</div>
		</div>

		<div class="md:mb-6 my-6 bg-yellow-100 shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-yellow-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Delete or Block User</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-700">
					Delete to completely eliminate the user, Block to prevent further access.
				</p>
			</div>
			<div class="px-4 py-5 sm:px-6">
				<p>Not implemented </p>
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
import {AppspaceUser, AvatarState, saveNewUser, updateUserMeta} from '../models/appspace_users';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';
import Avatar from '../components/ui/Avatar.vue';

export default defineComponent({
	name: 'ManageAppspaceUser',
	components: {
		ViewWrap,
		DataDef,
		Avatar
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

		let avatar_state = AvatarState.Preserve;
		let avatar :Blob|null = null;

		async function avatarChanged(ev:any) {
			if( ev ) {
				avatar = ev;
				avatar_state = AvatarState.Replace;
			}
			else {
				avatar = null;
				avatar_state = AvatarState.Delete;
			}
		}

		async function save() {
			if( display_name.value.trim() === "" ) {
				alert("please enter a display name");
				return;
			}
			if( proxy_id.value ) {
				// update
				await updateUserMeta(appspace.id, proxy_id.value, {
					display_name: display_name.value,
					permissions: [],
					avatar: avatar_state
				}, avatar);
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
					permissions: [],
					avatar: avatar_state
				}, avatar);
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
			avatarChanged, save,
		}
	}

});
</script>