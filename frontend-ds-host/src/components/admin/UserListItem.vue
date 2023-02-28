<script setup lang="ts">
import { defineProps } from 'vue';

import type {UserForAdmin} from '@/stores/types';

const props = defineProps<{
	user: UserForAdmin
}>();

</script>

<template>
	<div class="bg-white border-b border-b-gray-300 flex items-baseline">
		<div class="px-4 text-gray-500 text-right w-16">{{ props.user.user_id }}</div>
		<div class="flex-grow py-2">{{ props.user.email }}</div>
		<div 
			v-if="props.user.is_admin"
			class="text-xs uppercase font-bold text-orange-500 mx-4 py-1 px-3 rounded-full bg-orange-100">
			admin</div>
	</div>
</template>