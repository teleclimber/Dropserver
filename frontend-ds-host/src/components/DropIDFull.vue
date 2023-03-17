<script lang="ts" setup>
import type { UserDropID } from '../stores/types';

defineProps<{
	dropid: UserDropID
}>();

</script>

<template>
	<div class="py-2 border-t px-4 sm:px-6">
		<div class="flex justify-between">
			<h3 class="md:text-lg whitespace-nowrap">
				<span class="font-medium text-blue-800">{{dropid.domain_name}}</span>
				<template v-if="dropid.handle != ''">
					<span class=""> / </span>
					<span class=" ">{{dropid.handle}}</span>
				</template>
			</h3>
			<div>
				<router-link :to="{name:'dropid', query:{domain:dropid.domain_name, handle:dropid.handle}}" class="btn">Manage</router-link>
			</div>
		</div>
		<div>
			<span class="text-gray-700">Display name:</span>
			{{dropid.display_name }}
			<span v-if="dropid.display_name ===''" class="text-gray-500 text-sm">(none)</span>
		</div>

	</div>
</template>

