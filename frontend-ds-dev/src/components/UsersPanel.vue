<style scoped>
	.log-grid {
		display: grid;
		grid-template-columns: 3rem 50px max-content max-content 1fr max-content;
	}
</style>

<template>
	<div class="m-4 flex items-center h-8">
		Logged-in user:
		<div v-if="active_user" class="flex items-center bg-gray-100 mx-2 pr-2">
			<img v-if="active_user.avatar" class="h-8 w-8" :src="'avatar/appspace/'+active_user.avatar">
			<span class="font-bold ml-2">{{active_user.display_name}}</span>
			<span class="font-mono text-sm ml-2">{{active_user.proxy_id}}</span>
		</div>
		<UiButton v-if="active_user" class="mx-2 flex items-center" @click.stop.prevent="userData.setActiveUser('')">
			<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M3 3a1 1 0 00-1 1v12a1 1 0 102 0V4a1 1 0 00-1-1zm10.293 9.293a1 1 0 001.414 1.414l3-3a1 1 0 000-1.414l-3-3a1 1 0 10-1.414 1.414L14.586 9H7a1 1 0 100 2h7.586l-1.293 1.293z" clip-rule="evenodd" />
			</svg>
			Log out
		</UiButton>
		<span v-else class="italic text-gray-600 ml-2">
			No user selected. Requests are interpreted as unauthenticated.
		</span>
	</div>
	<div class="m-4">
		<div class="flex justify-between">
			<h2 class="text-2xl my-2">{{userData.users.length}} Appspace Users:</h2>
			<UiButton ref="add_btn" class="self-center mx-2 flex items-center" @click.stop.prevent="showAddUser" title="Add a new appspace user">
				<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 pr-1" viewBox="0 0 20 20" fill="currentColor">
					<path d="M8 9a3 3 0 100-6 3 3 0 000 6zM8 11a6 6 0 016 6H2a6 6 0 016-6zM16 7a1 1 0 10-2 0v1h-1a1 1 0 100 2h1v1a1 1 0 102 0v-1h1a1 1 0 100-2h-1V7z" />
				</svg>
				Add User
			</UiButton>
		</div>
		<div class=" bg-gray-100">
			<div class="log-grid items-stretch border-t border-gray-400">
				<template  v-for="user in userData.users" :key="user.proxy_id">
					<label class="bg-gray-200 border-b border-gray-400 flex justify-center items-center text-center hover:bg-yellow-200" 
						:class="{'bg-yellow-100':userData.isUser(user.proxy_id)}">
						<input type="radio" name="activeuser" :value="user.proxy_id" v-model="active_user_input" />
					</label>
					<img v-if="user.avatar" :src="'avatar/appspace/'+user.avatar">
					<div v-else class="bg-gray-300 border-b border-gray-400">&nbsp;</div>
					<span class="bg-gray-200 text-gray-700 pl-4 self-stretch flex items-center text-lg font-bold border-b border-gray-400">{{user.display_name}}</span>
					<span class="bg-gray-200 text-gray-700 pl-4 self-stretch flex items-center text-sm font-mono border-b border-gray-400">{{user.proxy_id}}</span>
					<span class="bg-gray-200 text-gray-700 pl-4 self-stretch flex items-center text-sm border-b border-gray-400">{{user.permissions.join(", ")}}</span>
					<span class="bg-gray-200 text-sm border-b border-gray-400 flex items-center justify-end">
						<UiButton class="mx-2 flex items-center" @click.stop.prevent="showEditUser(user.proxy_id)">
							<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 pr-1" viewBox="0 0 20 20" fill="currentColor">
								<path d="M17.414 2.586a2 2 0 00-2.828 0L7 10.172V13h2.828l7.586-7.586a2 2 0 000-2.828z" />
								<path fill-rule="evenodd" d="M2 6a2 2 0 012-2h4a1 1 0 010 2H4v10h10v-4a1 1 0 112 0v4a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" clip-rule="evenodd" />
							</svg>
							Edit
						</UiButton>
						<UiButton class="mx-2 flex items-center" @click.stop.prevent="delUser(user.proxy_id)">
							<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 pr-1" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
							</svg>
							Del
						</UiButton>
					</span>
				</template>
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
				<div class="flex justify-between mt-4">
					<UiButton @click="closeUserModal" class="">Cancel</UiButton>
					<UiButton type="submit" @click="saveEditUser" class="">Save</UiButton>
				</div>

				<button class="absolute top-0 right-0 p-2" @click="closeUserModal">
					<svg class="h-8 w-8 fill-current text-gray hover:text-grey-darkest" role="button" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><title>Close</title><path d="M14.348 14.849a1.2 1.2 0 0 1-1.697 0L10 11.819l-2.651 3.029a1.2 1.2 0 1 1-1.697-1.697l2.758-3.15-2.759-3.152a1.2 1.2 0 1 1 1.697-1.697L10 8.183l2.651-3.031a1.2 1.2 0 1 1 1.697 1.697l-2.758 3.152 2.758 3.15a1.2 1.2 0 0 1 0 1.698z"/></svg>
				</button>
			</div>
		</div>
	</teleport>	
</template>


<script lang="ts">
import { defineComponent, onMounted, reactive, ref, Ref, nextTick, computed, watch} from 'vue';

import appData from '../models/app-data';
import userData from '../models/user-data';

import UiButton from './ui/UiButton.vue';

// Redo:
// - list users from db
// - add a "public" user
// - each db user can be updated
// - each user (inc public) can be selected as the owner
// - create new user
// - delete user
// - need a create/update UI



export default defineComponent({
	components: {
		UiButton
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

		const active_user = computed( () => {
			return userData.getActiveUser();
		});
		const active_user_input = ref("");
		onMounted( () => {
			const u = userData.getActiveUser();
			if(u) active_user_input.value =  u.proxy_id;
		});
		watch( active_user, () => {
			active_user_input.value = active_user.value ? active_user.value.proxy_id : '';
		});
		watch( active_user_input, () => {
			userData.setActiveUser(active_user_input.value);
		});

		return { userData, appData, 
			edit_user_open, is_edit,
			display_name_input, display_name,
			permissions,
			showAddUser, showEditUser, saveEditUser, closeUserModal,
			baked_in_avatars, avatar, avatar_url, avatarChanged,
			delUser,
			active_user, active_user_input,
		};
	},
});
</script>