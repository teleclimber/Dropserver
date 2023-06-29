<script lang="ts" setup>

defineProps<{
	license: string|undefined
}>();

</script>

<template>
	<span v-if="license" class="text-xs uppercase font-medium px-2 inline-block rounded bg-green-600 text-white">
		{{ license }} License
	</span>
	<span v-else class="italic text-xs uppercase font-medium px-2 inline-block rounded bg-orange-400 text-white">
		no license
	</span>
</template>