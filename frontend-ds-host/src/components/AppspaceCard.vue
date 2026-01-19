<script setup lang="ts">
import {ref, computed} from 'vue';

import type { Appspace, RemoteAppspace } from '@/stores/types';
import { getAvatarUrl } from '@/stores/appspace_users';
import { useAuthUserStore } from '@/stores/auth_user';

const props = defineProps<{
	local_appspace?: Appspace,
	remote_appspace?: RemoteAppspace
}>();

const authUserStore = useAuthUserStore();
authUserStore.fetch();

const is_local = ref(true);
const domain_strong = ref('');
const domain = ref('');
const paused = computed( () => {
	return !!props.local_appspace?.status.paused;
});
const enter_link = ref('');

if( props.local_appspace ) {
	const a = props.local_appspace;
	const pieces = a.domain_name.trim().split('.');
	if( pieces.length > 0 ){
		domain_strong.value = pieces.shift() as string;
		domain.value = pieces.join('.');
	}
	else {
		domain_strong.value = a.domain_name
	}
	enter_link.value = "/appspacelogin?appspace_id="+encodeURIComponent(a.appspace_id)
}
else if( props.remote_appspace ) {
	is_local.value = false;
	const a = props.remote_appspace;
	domain_strong.value = a.domain_name;
	enter_link.value = "/remoteappspacelogin?appspace="+encodeURIComponent(a.domain_name);
}
else throw new Error("got neither local nor remote appspace");

const app_icon_error = ref(false);
const app_icon = computed( () => {
	if( app_icon_error.value || !props.local_appspace ) return "";
	return `/api/appspace/${props.local_appspace.appspace_id}/app-icon`;
});

const users = computed( () => {
	if( !props.local_appspace ) return;
	return props.local_appspace.users.map( u => {
		const appspace = props.local_appspace!;
		return {
			proxy_id: u.proxy_id,
			display_name: u.display_name,
			avatar_url: getAvatarUrl(appspace.appspace_id, u.avatar),
		};
	});
});

const user_proxy_id = computed( () => {
	return props.local_appspace?.auth_user_id_conflicts?.proxy_id;
});
const user_conflict = computed( () => {
	return props.local_appspace?.auth_user_id_conflicts?.conflict;
});

</script>

<template>
	<div class="mb-6 grid grid-cols-1 md:grid-cols-3 justify-items-stretch border-t-4 w-full box-border" 
		:style="'border-color:'+(local_appspace?.ver_data?.color || 'rgb(135, 151, 164)')" >
		<a :href="enter_link" class="p-2 md:p-3 pt-3 md:col-span-2 w-full"
			:class="{'hover:bg-yellow-50':!paused, 'bg-gray-50':paused, 'bg-white': !paused}">
			<div class="flex items-center">
				<img v-if="app_icon" :src="app_icon" @error="app_icon_error = true" class="w-12 h-12" />
				<div v-else class="w-12 h-12 text-gray-300 __border flex justify-center items-center">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-12 h-12">
						<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" />
					</svg>
				</div>
				<div class="ml-2 md:ml-3">
					<h1 class="text-xl overflow-x-hidden text-ellipsis max-w-full box-border">
						<span class="font-bold">{{ domain_strong }}</span>
						<span class="text-gray-500" v-if="domain !== ''">.{{ domain }}</span>
					</h1>
					<h4 v-if="is_local" class="overflow-x-hidden text-ellipsis text-gray-600">{{ local_appspace?.ver_data?.name }}</h4>
					<h4 v-else class="flex items-center text-gray-600">
						Remote appspace
						<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 ml-1 ">
							<path fill-rule="evenodd" d="M4.25 5.5a.75.75 0 00-.75.75v8.5c0 .414.336.75.75.75h8.5a.75.75 0 00.75-.75v-4a.75.75 0 011.5 0v4A2.25 2.25 0 0112.75 17h-8.5A2.25 2.25 0 012 14.75v-8.5A2.25 2.25 0 014.25 4h5a.75.75 0 010 1.5h-5z" clip-rule="evenodd" />
							<path fill-rule="evenodd" d="M6.194 12.753a.75.75 0 001.06.053L16.5 4.44v2.81a.75.75 0 001.5 0v-4.5a.75.75 0 00-.75-.75h-4.5a.75.75 0 000 1.5h2.553l-9.056 8.194a.75.75 0 00-.053 1.06z" clip-rule="evenodd" />
						</svg>
					</h4>
				</div>
			</div>
			<div class="flex justify-end mt-2">
				<span v-if="paused" class="mr-2 bg-pink-200 text-pink-800 px-2 text-xs font-bold uppercase">
					<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4 inline-block">
						<path d="M5.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75A.75.75 0 007.25 3h-1.5zM12.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75a.75.75 0 00-.75-.75h-1.5z" />
					</svg>
					Paused
				</span>
				<span class="btn" :class="{'text-gray-400': paused}">Enter Appspace</span>
			</div>
		</a>
		<div class="py-2 px-4  bg-gray-50 flex flex-col justify-between">
			<div class=" flex flex-row items-start">
				<div v-for="u in users" class="flex items-center my-1 mr-1 pr-2 rounded-full text-gray-700"
					:class="[u.proxy_id === user_proxy_id ? 'bg-gray-200' : '']">
					<div class="w-7 h-7 rounded-full bg-gray-300  flex justify-center content-center text-gray-400">
						<img v-if="u.avatar_url" :src="u.avatar_url" class="rounded-full bg-clip-border"/>
						<svg v-else xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 self-end">
							<path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
						</svg>
					</div>
					<span class="pl-1">{{ u.display_name }}</span>
				</div>
			</div>
			<span v-if="user_conflict" class="text-orange-500 flex items-center">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="size-5">
					<path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495ZM10 5a.75.75 0 0 1 .75.75v3.5a.75.75 0 0 1-1.5 0v-3.5A.75.75 0 0 1 10 5Zm0 9a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z" clip-rule="evenodd" />
				</svg>
				conflicts
			</span>
		</div>
	</div>
</template>
