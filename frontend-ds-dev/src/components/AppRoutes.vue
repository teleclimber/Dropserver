<script setup lang="ts">
import appRoutesService from '../models/app-routes';

import Route from './Route.vue';

</script>

<template>
	<div class="border-l-4 border-gray-800  my-8">
		<h4 class="bg-gray-800 px-2 py-1 text-white inline-block">App Routes:</h4>
		<div class="bg-gray-100">
			<div v-if="appRoutesService.error" class="">
				<p class="bg-red-200">{{appRoutesService.error}}</p>
			</div>
			<div v-else-if="appRoutesService.routes.length" class="log-grid">
				<template  v-for="route in appRoutesService.routes"  :key="route.method+route.path">
					<Route :route="route"></Route>
				</template>
			</div>
			<div v-else class="flex justify-center items-center p-4">
				No app routes.
			</div>
		</div>
	</div>
</template>

<style scoped>
	.log-grid {
		display: grid;
		grid-template-columns: 8rem 6rem 4fr 7rem 5fr;
	}
</style>
