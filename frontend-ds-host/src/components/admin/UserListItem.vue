<script setup lang="ts">
import type {User} from '@/stores/types';

const props = defineProps<{
	user: User
}>();

</script>

<template>
	<div class="bg-white border-b border-b-gray-300 flex items-baseline">
		<div class="px-4 text-gray-500 text-right w-16">{{ user.user_id }}</div>
		<div class="flex-grow py-2">
			<div v-if="user.email" class="">{{ user.email }}</div>
			<div v-else class="italic text-gray-500">No email set</div>
			<div v-if="user.tsnet_identifier">{{ user.tsnet_identifier }} ({{ user.tsnet_extra_name }})</div>
			<div v-else class="italic text-gray-500">No tailscale identifier</div>
		</div>
		<div class="self-center flex flex-row items-baseline">
			<span 
				v-if="user.is_admin"
				class="text-xs uppercase font-bold text-orange-500 mx-4 py-1 px-3 rounded-full bg-orange-100">
				admin</span>
			<router-link class="btn mr-4" :to="{name:'admin-user', params:{user_id:user.user_id}}">Edit</router-link>
		</div>
		
	</div>
</template>