<style scoped>
	.log-grid {
		display: grid;
		grid-template-columns: 3rem 10rem 5rem 1fr 8rem;
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
import { defineComponent, reactive } from 'vue';

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
	data() {
		return {
			edit_user_open: false,
			is_edit: false,
			display_name: "",
			permissions: <{[permission:string]:boolean}>{},		// improve that
			proxy_id: ""
		}
	},
	components: {
	},
	setup(props, context) {
		return { userData, baseData };
	},
	methods: {
		showAddUser() {
			if( this.edit_user_open ) return;

			this.is_edit = false;

			this.proxy_id = "";
			this.display_name = "";
			this.permissions = {};

			this.edit_user_open = true;

			this.focusModal();
		},
		showEditUser(proxy_id:string) {
			if( this.edit_user_open ) return;

			this.is_edit = true;

			const u = this.userData.getUser(proxy_id);
			if( u === undefined ) {
				throw new Error("can't find user: "+proxy_id);
			}

			// get user and copy values
			this.proxy_id = proxy_id;
			this.display_name = u.display_name;
			this.permissions = {};
			u.permissions.forEach((p) => {
				this.permissions[p] = true;
			});

			this.edit_user_open = true;

			this.focusModal();
		},
		focusModal() {
			this.$nextTick( () => {
				const input = <HTMLInputElement>this.$refs.display_name_input;
				input.focus();
			});
		},
		saveEditUser() {
			if( !this.edit_user_open ) return;

			if( this.display_name == "" || this.display_name.length > 20 ) return;	//what are the validatiosn again?
			// maybe let the user model perform validations. Just wait for response?

			const permissions :string[] = [];
			for( let p in this.permissions ) {
				if(this.permissions[p])	permissions.push(p);
			};

			if( this.is_edit ) {
				//this.userData.editUser();
				this.userData.editUser(this.proxy_id, this.display_name, permissions)
			}
			else {
				this.userData.addUser(this.display_name, permissions)
			}
			this.closeUserModal();
		},
		closeUserModal() {
			// move focus back
			if( this.is_edit ) {
				// 
			} 
			else {
				(<HTMLInputElement>this.$refs.add_btn).focus();
			}
			this.edit_user_open = false;
		},
		delUser(proxy_id: string) {
			const u = this.userData.getUser(proxy_id);
			if( u === undefined ) return;
			if( confirm("Delete "+u.display_name+"?") ) {
				this.userData.deleteUser(proxy_id);
			}
		},
	}
});
</script>