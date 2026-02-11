<script setup lang="ts">
import { ref, Ref, computed } from 'vue';
import { RouterLink } from 'vue-router';
import type { Appspace, App } from '@/stores/types';
import { useAppsStore } from '@/stores/apps';
import { useAuthUserStore } from '@/stores/auth_user';
import { useAppspaceUserConflictsStore } from '@/stores/appspace_user_conflicts';
import { getAvatarUrl } from '@/stores/appspace_users';

import MinimalAppUrlData from './appspace/MinimalAppUrlData.vue';
import InlineMessage from './ui/InlineMessage.vue';

const props = defineProps<{
	appspace: Appspace
}>();

const authUserStore = useAuthUserStore();
authUserStore.fetch();

const users = computed( () => {
	return props.appspace.users.map( u => {
		return {
			proxy_id: u.proxy_id,
			display_name: u.display_name,
			avatar_url: getAvatarUrl(props.appspace.appspace_id, u.avatar),
		};
	});
});

const userConflicts = useAppspaceUserConflictsStore();
const has_user_conflicts = computed( () => {
	const conflicts = userConflicts.getForAppspace(props.appspace.appspace_id).value;
	return [...conflicts].some( c => c[1].conflict );
});

const owner_proxy_id = computed( () => {
	const conflicts = userConflicts.getForAppspace(props.appspace.appspace_id).value;
	const u_c = [...conflicts].find( c => {
		return !c[1].conflict && c[1].user_id === authUserStore.user_id;
	});
	return u_c ? u_c[1].proxy_id : "";
});

const appsStore = useAppsStore();
const app :Ref<App|undefined> = computed( () => {
	const a = appsStore.getApp(props.appspace.app_id);
	if( a !== undefined ) return a.value;
	return undefined;
});

const enter_link = ref("/appspacelogin?appspace_id="+encodeURIComponent(props.appspace.appspace_id));

const app_icon_error = ref(false);
const app_icon = computed( () => {
	if( app_icon_error.value || !props.appspace ) return "";
	return `/api/appspace/${props.appspace.appspace_id}/app-icon`;
});

</script>

<template>
	<div class="bg-white overflow-hidden border-b border-l-4 border-b-gray-300 px-4 py-4 mb-4" :style="'border-left-color:'+(appspace?.ver_data?.color || 'rgb(135, 151, 164)')">
		<a :href="enter_link" class="block text-xl md:text-2xl mb-2 font-medium text-gray-900  hover:text-blue-500 overflow-hidden text-ellipsis ">
			{{appspace.domain_name}}
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 inline-block">
				<path fill-rule="evenodd" d="M3 4.25A2.25 2.25 0 015.25 2h5.5A2.25 2.25 0 0113 4.25v2a.75.75 0 01-1.5 0v-2a.75.75 0 00-.75-.75h-5.5a.75.75 0 00-.75.75v11.5c0 .414.336.75.75.75h5.5a.75.75 0 00.75-.75v-2a.75.75 0 011.5 0v2A2.25 2.25 0 0110.75 18h-5.5A2.25 2.25 0 013 15.75V4.25z" clip-rule="evenodd" />
				<path fill-rule="evenodd" d="M6 10a.75.75 0 01.75-.75h9.546l-1.048-.943a.75.75 0 111.004-1.114l2.5 2.25a.75.75 0 010 1.114l-2.5 2.25a.75.75 0 11-1.004-1.114l1.048-.943H6.75A.75.75 0 016 10z" clip-rule="evenodd" />
			</svg>
		</a>
		<div class=" flex">
			<div class="my-2 pr-2 flex items-center" :style="'border-color:'+(appspace?.ver_data?.color || 'rgb(135, 151, 164)')" >
				<img v-if="app_icon" :src="app_icon" @error="app_icon_error = true" class="w-10 h-10 self-start" />
				<div v-else class="w-10 h-10 text-gray-300 __border flex justify-center items-center">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-12 h-12">
						<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" />
					</svg>
				</div>
				<div class="flex flex-col md:flex-row items-baseline">
					<p class="ml-1">
						<span class="font-medium text-lg">
							{{appspace.ver_data?.name}}
						</span>
						<span class="bg-gray-200 text-gray-600 px-1 rounded-md ml-1">{{appspace.app_version}}</span>
					</p>
					<MinimalAppUrlData v-if="app?.url_data" :cur_ver="appspace.app_version" :url_data="app.url_data" class="self-baseline"></MinimalAppUrlData>
					<span v-else-if="appspace.upgrade_version" class="ml-1">
						Upgrade available: <span class="bg-gray-200 text-gray-600 px-1 rounded-md">{{appspace.upgrade_version}}</span>
					</span>
					<router-link :to="{name: 'manage-app', params:{id:appspace.app_id}}" class="ml-1 btn">
						manage
					</router-link>
				</div>
			</div>
		</div>
		<ul class="flex items-center">
			<li v-for="u in users" class="flex items-center my-1 mr-1 pr-2 rounded-full text-gray-700"
				:class="[u.proxy_id === owner_proxy_id ? 'bg-gray-200' : '']">
				<div class="w-7 h-7 rounded-full bg-gray-300  flex justify-center content-center text-gray-400">
					<img v-if="u.avatar_url" :src="u.avatar_url" class="rounded-full bg-clip-border"/>
					<svg v-else xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 self-end">
						<path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
					</svg>
				</div>
				<span class="pl-1">{{ u.display_name }}</span>
			</li>
			<li v-if="has_user_conflicts">
				<InlineMessage mood="warn">Conflicts</InlineMessage>
			</li>
		</ul>
		<div class="pt-4 flex justify-end items-baseline">
			<span v-if="appspace.paused" class="mr-2 bg-pink-200 text-pink-800 px-2 text-xs font-bold uppercase">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4 inline-block">
					<path d="M5.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75A.75.75 0 007.25 3h-1.5zM12.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75a.75.75 0 00-.75-.75h-1.5z" />
				</svg>

				Paused
			</span>
			<router-link :to="{name: 'manage-appspace', params:{appspace_id:appspace.appspace_id}}" class="btn">Manage appspace</router-link>
		</div>
	</div>
</template>