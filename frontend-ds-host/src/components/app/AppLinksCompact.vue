<script setup lang="ts">
import { computed } from 'vue';
import { AppVersionUI } from '@/stores/types';

interface Links {
	website: string,
	code: string,
	funding: string
}
const props = defineProps<{
	ver_data: Links|undefined
}>();

const no_links = computed( () => {
	return !props.ver_data || (!props.ver_data.website && !props.ver_data.code && !props.ver_data.funding)
});

const keys = ['website']
const links = computed( () => {
	const ret :{link:string, name:string}[] = [];
	if( !props.ver_data ) return ret;
	const d = props.ver_data;
	if( d.website)	ret.push({link: d.website,	name: 'website'});
	if( d.code) 	ret.push({link: d.code,		name: 'code'});
	if( d.funding)	ret.push({link: d.funding,	name: 'funding'});
	return ret;
})

</script>

<template>
	<div class="flex" v-if="links.length">
		<a v-for="l in links" :href="l.link" class="mr-6 uppercase text-blue-500 hover:underline hover:text-blue-700">{{ l.name }}</a>
	</div>
</template>