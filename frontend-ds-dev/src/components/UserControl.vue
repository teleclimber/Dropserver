<style scoped>
	.log-grid {
		display: grid;
		grid-template-columns: 3rem 50px 10rem 5rem 1fr 8rem;
	}
</style>

<template>
	<div class="border-l-4 border-gray-800  my-8">
		<h4 class="bg-gray-800 px-2 text-white inline-block">Users:</h4>
		<button ref="add_btn" class="bg-blue-400 inline-block px-2 mx-2 rounded" @click.stop.prevent="showAddUser">Add</button>
		<div class="overflow-y-scroll h-64 bg-gray-100" style="scroll-behavior: smooth" ref="scroll_container">
			<div class="log-grid items-stretch">
				<template  v-for="user in userData.users" :key="user.proxy_id">
					<span class="bg-gray-200 pt-1 border-b border-gray-400 text-green-500 text-center hover:bg-green-200" @click.stop.prevent="userData.setUser(user.proxy_id)">
						<svg v-if="userData.isUser(user.proxy_id)" class="inline w-8 h-8" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M6.267 3.455a3.066 3.066 0 001.745-.723 3.066 3.066 0 013.976 0 3.066 3.066 0 001.745.723 3.066 3.066 0 012.812 2.812c.051.643.304 1.254.723 1.745a3.066 3.066 0 010 3.976 3.066 3.066 0 00-.723 1.745 3.066 3.066 0 01-2.812 2.812 3.066 3.066 0 00-1.745.723 3.066 3.066 0 01-3.976 0 3.066 3.066 0 00-1.745-.723 3.066 3.066 0 01-2.812-2.812 3.066 3.066 0 00-.723-1.745 3.066 3.066 0 010-3.976 3.066 3.066 0 00.723-1.745 3.066 3.066 0 012.812-2.812zm7.44 5.252a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
						</svg>
					</span>
					<img v-if="user.avatar" :src="'avatar/appspace/'+user.avatar">
					<div v-else class="bg-gray-300 border-b border-gray-400">&nbsp;</div>
					<span class="bg-gray-200 text-gray-700 pl-2 py-2 text-lg font-bold border-b border-gray-400">{{user.display_name}}</span>
					<span class="bg-gray-200 text-gray-700 pl-2 pt-3 text-sm font-mono border-b border-gray-400">{{user.proxy_id}}</span>
					<span class="bg-gray-200 text-gray-700 pl-2 pt-3 text-sm border-b border-gray-400">{{user.permissions.join(", ")}}</span>
					<span class="bg-gray-200 pt-3 text-sm border-b border-gray-400">
						<button class="bg-blue-400 inline-block px-2 mx-2 rounded" @click.stop.prevent="showEditUser(user.proxy_id)">Edit</button>
						<button class="bg-blue-400 inline-block px-2 mx-2 rounded" @click.stop.prevent="delUser(user.proxy_id)">Del</button>
					</span>
				</template>

				<span class="bg-blue-100 pt-1 border-b border-gray-400 text-green-500 text-center hover:bg-green-200" @click.stop.prevent="userData.setUser('')">
					<svg v-if="userData.isUser('')" class="inline w-8 h-8" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M6.267 3.455a3.066 3.066 0 001.745-.723 3.066 3.066 0 013.976 0 3.066 3.066 0 001.745.723 3.066 3.066 0 012.812 2.812c.051.643.304 1.254.723 1.745a3.066 3.066 0 010 3.976 3.066 3.066 0 00-.723 1.745 3.066 3.066 0 01-2.812 2.812 3.066 3.066 0 00-1.745.723 3.066 3.066 0 01-3.976 0 3.066 3.066 0 00-1.745-.723 3.066 3.066 0 01-2.812-2.812 3.066 3.066 0 00-.723-1.745 3.066 3.066 0 010-3.976 3.066 3.066 0 00.723-1.745 3.066 3.066 0 012.812-2.812zm7.44 5.252a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
					</svg>
				</span>
				<div class="bg-blue-100 border-b border-gray-400">&nbsp;</div>
				<span style="grid-column: span 4" class="bg-blue-100 text-blue-700 pl-2 py-2 text-lg font-bold border-b border-gray-400 italic">Public</span>
					
			</div>
		</div>
	</div>

	<teleport to="body">
		<div v-if="edit_user_open" class="fixed inset-0 flex items-center justify-center bg-gray-400 bg-opacity-75">
			<div class="relative p-4 bg-white w-full max-w-lg m-auto flex-col flex">
				<h4 class="text-2xl mb-4">{{ is_edit ? "Edit User:" : "Add User:" }}</h4>

				<div class="flex flex-col my-2">
					<label for="display_name" class="">Display Name:</label>
					<input type="text" ref="display_name_input" name="display_name" id="display_name" v-model="display_name" class="border rounded p-2">
				</div>
				<div class="my-2 flex">
					<div class="flex flex-col">
						Avatar:
						<img v-if="avatar" :src="avatar_url" class="w-24 h-24">
						<div v-else class="w-24 h-24 flex-grow bg-gray-100 text-gray-500 flex justify-center italic items-center">none</div>
					</div>
					<div class="pl-4 flex flex-col">
						Select Avatar:
						<div class="flex items-start">
							<div class="w-16 h-16 bg-gray-100 text-gray-500 flex justify-center italic items-center"  @click="avatarChanged('')">none</div>
							<img v-for="a in baked_in_avatars" :key="a" class="w-16 h-16 opacity-50 hover:opacity-100" :src="'avatar/baked-in/'+a" @click="avatarChanged(a)">
						</div>
					</div>
				</div>
				<div>Permissions:</div>
				<label v-for="permission in baseData.user_permissions" :key="permission.key">
					<input type="checkbox" v-model="permissions[permission.key]" />
					{{permission.name}}
				</label>

				<div class="flex justify-between mt-4">
					<button @click="closeUserModal" class="bg-gray-700 hover:bg-gray-900 text-white py-1 px-2 rounded">Cancel</button>
					<button type="submit" @click="saveEditUser" class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 rounded">Save</button>
				</div>

				<button class="absolute top-0 right-0 p-2" @click="closeUserModal">
					<svg class="h-8 w-8 fill-current text-gray hover:text-grey-darkest" role="button" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><title>Close</title><path d="M14.348 14.849a1.2 1.2 0 0 1-1.697 0L10 11.819l-2.651 3.029a1.2 1.2 0 1 1-1.697-1.697l2.758-3.15-2.759-3.152a1.2 1.2 0 1 1 1.697-1.697L10 8.183l2.651-3.031a1.2 1.2 0 1 1 1.697 1.697l-2.758 3.152 2.758 3.15a1.2 1.2 0 0 1 0 1.698z"/></svg>
				</button>
			</div>
		</div>
	</teleport>	
</template>


<script lang="ts">
import { defineComponent, reactive, ref, Ref, nextTick} from 'vue';

import baseData from '../models/base-data';
import userData from '../models/user-data';

// Redo:
// - list users from db
// - add a "public" user
// - each db user can be updated
// - each user (inc public) can be selected as the owner
// - create new user
// - delete user
// - need a create/update UI



export default defineComponent({
	name: 'UserControl',
	components: {
	},
	setup(props, context) {
		const display_name_input :Ref<HTMLInputElement|null> = ref(null);
		const edit_user_open = ref(false);
		const is_edit = ref(false);
		const proxy_id = ref("");
		const display_name = ref("");
		const avatar = ref("");
		const avatar_url = ref("");	// either avatar/appspace/<appspace avatar file> or avatar/baked-in/<baked-in-file>

		const permissions :{[key:string]:boolean} = reactive({});

		const baked_in_avatars :Ref<string[]> = ref([]);
		fetch('avatar/baked-in').then( async (resp) => {
			if( !resp.ok ) throw new Error("fetch error for basic data");
			baked_in_avatars.value = await resp.json();
		});

		function avatarChanged(a :string) {
			console.log("change", a);
			avatar.value = a;
			avatar_url.value = "avatar/baked-in/"+a;
		}

		function showAddUser() {
			if( edit_user_open.value ) return;

			is_edit.value = false;

			proxy_id.value = "";
			display_name.value = "";
			avatar.value = "";
			avatar_url.value = "";

			for( let p in permissions ) {
				permissions[p] = false;
			}

			edit_user_open.value = true;

			focusModal();
		}
		function showEditUser(p_id:string) {
			if( edit_user_open.value ) return;

			is_edit.value = true;

			const u = userData.getUser(p_id);
			if( u === undefined ) {
				throw new Error("can't find user: "+p_id);
			}

			// get user and copy values
			proxy_id.value = p_id;
			display_name.value = u.display_name;
			avatar.value = u.avatar;
			avatar_url.value = "avatar/appspace/"+u.avatar;

			// first reset permissions:
			for( let p in permissions ) {
				permissions[p] = false;
			}
			u.permissions.forEach((p) => {
				permissions[p] = true;
			});

			edit_user_open.value = true;

			focusModal();
		}
		function focusModal() {
			nextTick( () => {
				if( !display_name_input.value ) return;
				display_name_input.value.focus();
			});
		}
		function saveEditUser() {
			if( !edit_user_open.value ) return;

			if( display_name.value == "" || display_name.value.length > 20 ) return;	//what are the validatiosn again?
			// maybe let the user model perform validations. Just wait for response?

			const ps :string[] = [];
			for( let p in permissions ) {
				if( permissions[p] ) ps.push(p);
			}

			if( is_edit.value ) {
				//this.userData.editUser();
				userData.editUser(proxy_id.value, display_name.value, avatar.value, ps)
			}
			else {
				userData.addUser(display_name.value, avatar.value, ps)
			}
			closeUserModal();
		}
		function closeUserModal() {
			// move focus back
			if( is_edit.value ) {
				// 
			} 
			else {
				//TODO later... (<HTMLInputElement>this.$refs.add_btn).focus();
			}
			edit_user_open.value = false;
		}
		
		function delUser(proxy_id: string) {
			const u = userData.getUser(proxy_id);
			if( u === undefined ) return;
			if( confirm("Delete "+u.display_name+"?") ) {
				userData.deleteUser(proxy_id);
			}
		}

		return { userData, baseData, 
			edit_user_open, is_edit,
			display_name_input, display_name,
			permissions,
			showAddUser, showEditUser, saveEditUser, closeUserModal,
			baked_in_avatars, avatar, avatar_url, avatarChanged,
			delUser,
		};
	},
});
</script>