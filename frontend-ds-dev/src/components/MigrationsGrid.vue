<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
	migrations: {
		direction: "up" | "down";
		schema: number;
	}[]
}>();

const grid = computed( () => {
	const m = props.migrations;
	const from_schema = m.reduce( (cur, m) => Math.min(cur, m.schema), 9999999 );
	const to_schema = m.reduce( (cur, m) => Math.max(cur, m.schema), 0 );
	const ret :{schema:number, up:boolean, down:boolean}[] = [];
	for( let i=from_schema; i<=to_schema; ++i ) {
		ret.push({
			schema: i,
			up: !!m.find( g=> g.schema === i && g.direction === "up"),
			down: !!m.find( g=> g.schema === i && g.direction === "down")
		});
	}
	return ret;
});
</script>

<template>
	<table v-if="grid.length" class="border">
		<thead>
			<tr class="text-gray-800">
				<th class="px-1 font-medium">schema:</th>
				<th v-for="g in grid" class="font-medium">{{ g.schema }}</th>
			</tr>
		</thead>
		<tbody>
			<tr>
				<td class="uppercase text-sm text-gray-500 text-right px-1">up:</td>
				<td v-for="g in grid" class="px-1">
					<svg v-if="g.up" xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-arrow-big-up-filled text-green-500" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
						<path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
						<path d="M10.586 3l-6.586 6.586a2 2 0 0 0 -.434 2.18l.068 .145a2 2 0 0 0 1.78 1.089h2.586v7a2 2 0 0 0 2 2h4l.15 -.005a2 2 0 0 0 1.85 -1.995l-.001 -7h2.587a2 2 0 0 0 1.414 -3.414l-6.586 -6.586a2 2 0 0 0 -2.828 0z" stroke-width="0" fill="currentColor"></path>
					</svg>
					<svg v-else xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-x text-orange-500" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
						<path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
						<path d="M18 6l-12 12"></path>
						<path d="M6 6l12 12"></path>
					</svg>
				</td>
			</tr>
			<tr>
				<td class="uppercase text-sm text-gray-500 text-right px-1">down:</td>
				<td v-for="g in grid" class="px-1">
					<svg v-if="g.down" xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-arrow-big-down-filled text-green-500" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
						<path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
						<path d="M10 2l-.15 .005a2 2 0 0 0 -1.85 1.995v6.999l-2.586 .001a2 2 0 0 0 -1.414 3.414l6.586 6.586a2 2 0 0 0 2.828 0l6.586 -6.586a2 2 0 0 0 .434 -2.18l-.068 -.145a2 2 0 0 0 -1.78 -1.089l-2.586 -.001v-6.999a2 2 0 0 0 -2 -2h-4z" stroke-width="0" fill="currentColor"></path>
					</svg>
					<svg v-else xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-x text-orange-500" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
						<path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
						<path d="M18 6l-12 12"></path>
						<path d="M6 6l12 12"></path>
					</svg>
				</td>
			</tr>
		</tbody>
	</table>
	<div v-else class="inline italic text-gray-500">
		(no data migrations)
	</div>
</template>