<script lang="ts" setup>
import { computed } from 'vue';
import { AppAuthor } from '@/stores/types';

const props = defineProps<{
	authors: AppAuthor[]|undefined
}>();

const author = computed( () => {
	if( !props.authors?.length ) return undefined;
	return props.authors[0]
});
</script>

<template>
	<span>
		<span v-if="!author" class="italic text-gray-500">author unknown</span>
		<a v-else-if="author.url" :href="author.url" class="text-blue-500 underline hover:text-blue-700">
			{{ author.name||author.url }}
		</a>
		<a v-else-if="author.email" :href="'mailto:'+author.email" class="text-blue-500 underline hover:text-blue-700">
			{{ author.name||author.email }}
		</a>
		<span v-else>{{ author.name }}</span>
		<span v-if="authors && authors.length > 1">and {{ authors.length -1 }} more</span>
	</span>
</template>