<script lang="ts" setup>
import type { AppVersion } from '../stores/types';

defineProps<{
	app_version: AppVersion
}>();

</script>

<template>
	<div class="grid grid-cols-4 items-center justify-center border border-gray-200 rounded-md">
		<span class="bg-gray-200 text-center">version</span>
		<span class="bg-gray-200 text-center">schema</span>
		<span class="bg-gray-200 text-center">API</span>
		<span class="bg-gray-200">&nbsp;</span>
		<div class="px-4 py-2 font-bold text-lg text-center">{{app_version.version}}</div>
		<div class=" text-center">{{app_version.schema}}</div>
		<div class=" text-center">0...</div>
		<div>details...</div>
	</div>
</template>