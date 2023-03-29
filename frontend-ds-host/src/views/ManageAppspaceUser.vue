<script lang="ts" setup>
import { ref, Ref, reactive, computed, onMounted, onUnmounted, watch, watchEffect } from 'vue';
import { useRouter } from 'vue-router';

import { setTitle } from '../controllers/nav';

import { useAppspacesStore } from '@/stores/appspaces';
import { useAppspaceUsersStore, AvatarState, getAvatarUrl } from '@/stores/appspace_users';

// import {Appspace} from '../models/appspaces';
// import {AppspaceUser, AvatarState, saveNewUser, updateUserMeta} from '../models/appspace_users';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';
import Avatar from '../components/ui/Avatar.vue';

const props = defineProps<{
	appspace_id: number,
	proxy_id?: string
}>();

const router = useRouter();

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();
const appspace = computed( () => {
	if( appspacesStore.is_loaded ) return appspacesStore.mustGetAppspace(props.appspace_id).value;
});
watchEffect( () => {
	if( appspace.value ) setTitle(appspace.value.domain_name);
});

const appspaceUsersStore = useAppspaceUsersStore();
appspaceUsersStore.loadData(props.appspace_id);

const user = computed( () => {
	if( props.proxy_id === undefined || !appspaceUsersStore.isLoaded(props.appspace_id) ) return;
	const u = appspaceUsersStore.getUser(props.appspace_id, props.proxy_id );
	if( u ) return u.value;
});

const drop_id_input :Ref<HTMLInputElement|undefined> = ref();
onMounted( () => {
	if( drop_id_input.value === undefined ) return;
	drop_id_input.value.focus();
});

const drop_id = ref("");
const display_name = ref("");

watchEffect( () => {
	if( user.value === undefined ) return;
	drop_id.value = user.value.auth_id;	// assumes authid is dropid
	display_name.value = user.value.display_name;
	// avatar?
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

const invalid = computed( () => {
	if( display_name.value.trim() === "" ) return "display name can not be empty";
	if( display_name.value.length > 29 ) return "display name is too long";
	if( !props.proxy_id ) {
		if( drop_id.value.trim().length < 3 ) return "dropID is too short";
		if( drop_id.value.trim().length > 200 ) return "dropID is too long";
		// TODO check for dupes can'thave dupe dropids as appspace users
	}
	return "";
});

async function save() {
	if( invalid.value !== "" ) return;

	if( props.proxy_id ) {
		let auth_id = ""; 
		if( drop_id.value !== user.value?.auth_id ) auth_id = drop_id.value;
		await appspaceUsersStore.updateUserMeta(props.appspace_id, props.proxy_id, {
			auth_type: "dropid",
			auth_id,
			display_name: display_name.value,
			permissions: [],
			avatar: avatar_state
		}, avatar);
	}
	else {
		await appspaceUsersStore.addNewUser(props.appspace_id, {
			auth_type: "dropid",
			auth_id: drop_id.value,
			display_name: display_name.value,
			permissions: [],
			avatar: avatar_state
		}, avatar);
	}
	router.push({name: 'manage-appspace', params:{appspace_id: props.appspace_id}});
}

function cancel() {
	router.back();
}

onUnmounted( async () => {
	setTitle("");
});

</script>
<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<form @submit.prevent="save" @keyup.esc="cancel">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex items-baseline justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">
						{{ proxy_id ? "Manage Appspace User" : "New Appspace User" }}
					</h3>
				</div>
				<div class="border-b border-gray-200 py-6">
					<DataDef field="DropID:">
						<input ref="drop_id_input" type="text" v-model="drop_id" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					</DataDef>
					<DataDef field="Display Name:">
						<input type="text" v-model="display_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					</DataDef>
					<DataDef field="Avatar:">
						<Avatar :current="user ? getAvatarUrl(user) : ''" @changed="avatarChanged"></Avatar>
					</DataDef>
				</div>
				<div class="py-5 px-4 sm:px-6 flex items-baseline justify-between">
					<input type="button" class="btn" @click="cancel" value="Cancel" />
					<input
						type="submit"
						class="btn-blue"
						:disabled="!!invalid "
						value="Save" />
				</div>
			</form>
		</div>
	</ViewWrap>
</template>
