<script setup lang="ts">
import {ref, watchEffect, isReactive, shallowRef, ShallowRef} from 'vue';
import {Appspace} from '@/models/appspaces';
import {RemoteAppspace} from '@/models/remote_appspaces';
import {useAppsStore} from '@/stores/apps';

import { AppspaceUsers } from '@/models/appspace_users';

const props = defineProps<{
	local_appspace?: Appspace,
	remote_appspace?: RemoteAppspace
}>();

interface User {
	proxy_id: string,
	display_name: string,
	avatar_url: string,
	is_owner: boolean
}

const is_local = ref(true);
const domain_strong = ref('');
const domain = ref('');
const app_name = ref('');
const users :ShallowRef<User[]> = shallowRef([]);
const paused = ref(false);
const enter_link = ref('');

const appsStore = useAppsStore();
appsStore.fetchForOwner();

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

	watchEffect( () => {
		if( appsStore.isLoaded ) {
			const app = appsStore.apps.get(a.app_id);
			if( app === undefined ) return;
			app_name.value = app.value.name;
		}
	});

	const a_users = new AppspaceUsers;
	a_users.fetchForAppspace(a.id).then(()=> {
		users.value = a_users.au.map( u => {
			return {
				proxy_id: u.proxy_id,
				display_name: u.display_name,
				avatar_url: u.avatarURL,
				is_owner: u.auth_id === a.dropid
			};
		});
	});

	paused.value = a.paused;
	enter_link.value = "/appspacelogin?appspace="+encodeURIComponent(a.domain_name)
}
else if( props.remote_appspace ) {
	is_local.value = false;
	const a = props.remote_appspace;
	domain_strong.value = a.domain_name;
	enter_link.value = "/appspacelogin?appspace="+encodeURIComponent(a.domain_name);
}
else throw new Error("got neither local nor remote appspace");

</script>

<template>
	<div class="mb-6 grid grid-cols-1 md:grid-cols-3 justify-items-stretch border-t-2 w-full box-border">
		<a :href="enter_link" class="p-4 pt-3 md:col-span-2 w-full"
			:class="{'hover:bg-yellow-50':!paused, 'bg-gray-50':paused, 'bg-white': !paused}">
			<h1 class="text-xl overflow-x-hidden text-ellipsis max-w-full box-border">
				<span class="font-bold">{{ domain_strong }}</span>
				<span class="text-gray-500" v-if="domain !== ''">.{{ domain }}</span>
			</h1>
			<h4 v-if="is_local" class="overflow-x-hidden text-ellipsis text-gray-600">{{ app_name }}</h4>
			<h4 v-else class="flex items-center text-gray-600">
				Remote appspace
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 ml-1 ">
					<path fill-rule="evenodd" d="M4.25 5.5a.75.75 0 00-.75.75v8.5c0 .414.336.75.75.75h8.5a.75.75 0 00.75-.75v-4a.75.75 0 011.5 0v4A2.25 2.25 0 0112.75 17h-8.5A2.25 2.25 0 012 14.75v-8.5A2.25 2.25 0 014.25 4h5a.75.75 0 010 1.5h-5z" clip-rule="evenodd" />
					<path fill-rule="evenodd" d="M6.194 12.753a.75.75 0 001.06.053L16.5 4.44v2.81a.75.75 0 001.5 0v-4.5a.75.75 0 00-.75-.75h-4.5a.75.75 0 000 1.5h2.553l-9.056 8.194a.75.75 0 00-.053 1.06z" clip-rule="evenodd" />
				</svg>
			</h4>
			<div class="flex justify-end">
				<span v-if="paused" class="mr-2 bg-pink-200 text-pink-800 px-2 text-xs font-bold uppercase">
					<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4 inline-block">
						<path d="M5.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75A.75.75 0 007.25 3h-1.5zM12.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75a.75.75 0 00-.75-.75h-1.5z" />
					</svg>
					Paused
				</span>
				<span class="btn" :class="{'text-gray-400': paused}">Enter Appspace</span>
			</div>
		</a>
		<div class="py-2 px-4 flex flex-row items-start bg-gray-50">
			<div v-for="u in users" class="flex items-center my-1 mr-3 rounded-full text-gray-700">
				<div class="w-7 h-7 rounded-full bg-gray-300  flex justify-center content-center text-gray-400">
					<img v-if="u.avatar_url" :src="u.avatar_url" class="rounded-full bg-clip-border"/>
					<svg v-else xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 self-end">
						<path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
					</svg>
				</div>
				<span class="pl-1">{{ u.display_name }}</span>
				<svg v-if="u.is_owner" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 text-yellow-500">
					<path fill-rule="evenodd" d="M8 7a5 5 0 113.61 4.804l-1.903 1.903A1 1 0 019 14H8v1a1 1 0 01-1 1H6v1a1 1 0 01-1 1H3a1 1 0 01-1-1v-2a1 1 0 01.293-.707L8.196 8.39A5.002 5.002 0 018 7zm5-3a.75.75 0 000 1.5A1.5 1.5 0 0114.5 7 .75.75 0 0016 7a3 3 0 00-3-3z" clip-rule="evenodd" />
				</svg>
			</div>
		</div>
	</div>
</template>