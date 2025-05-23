<script lang="ts" setup>
import { getAvatarUrl } from '@/stores/appspace_users';
import type { AppspaceUser } from '../stores/types';

const props = defineProps<{
	user: AppspaceUser
}>();

const avatar_url = getAvatarUrl(props.user);

</script>

<template>
	<div class="flex">
		<img v-if="avatar_url" :src="avatar_url" class="w-12 h-12 flex-shrink-0 rounded-full bg-clip-border">
		<div v-else class="w-12 h-12 flex-shrink-0 rounded-full bg-clip-border bg-gray-200">&nbsp;</div>
		<div class="flex-grow flex-shrink pl-4 overflow-hidden">
			<div class="flex flex-col sm:flex-row items-baseline">
				<span class="pr-2 font-bold text-l">{{user.display_name}}</span>
			</div>
			<ul>
				<li v-for="auth in user.auths" class="text-gray-600">
					<template v-if="auth.type === 'tsnetid'">
						<span>Tailnet ID: </span>
						<span>{{ auth.extra_name }} ({{ auth.identifier.split("@")[1] }})</span>
					</template>
					<template v-else>
						<span class="">{{ auth.type === 'dropid' ? 'DropID' : auth.type }}: </span>
						<span>{{ auth.identifier }}</span>
					</template>
				</li>
			</ul>
		</div>
		<router-link class="btn self-start flex-shrink-0" :to="{name:'appspace-user', params:{appspace_id: user.appspace_id, proxy_id:user.proxy_id}}">Edit</router-link>
	</div>
</template>

