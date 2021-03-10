<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6">
			<h3 class="text-2xl leading-6 font-medium text-gray-900">{{app.name}}</h3>
			<p class="mt-1 max-w-2xl text-sm text-gray-500">
				Created: {{app.created_dt.toLocaleString()}} [app detail string]
			</p>
		</div>
		<div class="border-t border-gray-200">

			<div class="px-4 py-5 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6">
				<dt class="text-sm font-medium text-gray-500">
					Versions
				</dt>
				<dd class="mt-1 text-sm text-gray-900 sm:mt-0 sm:col-span-2">
					<ul class="border border-gray-200 rounded-md divide-y divide-gray-200">
						<li v-for="ver in app.versions" :key="ver.version" class="pl-3 pr-4 py-3 flex items-center justify-between text-sm">
							<div class="w-0 flex-1 flex items-center">
								<span class="ml-2 flex-1 w-0 font-bold">
									{{ver.version}}
								</span>
							</div>
							<div class="w-0 flex-1 flex items-center">
								<span class="ml-2 flex-1 w-0">
									{{ver.created_dt.toLocaleString()}}
								</span>
							</div>
							<div class="ml-4 flex-shrink-0">
								<router-link :to="{name:'new-appspace', query:{app_id:app.app_id, version:ver.version}}" class="btn flex items-center">
									<svg class="h-6 w-6 inline" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
										<path fill-rule="evenodd" d="M10 5a1 1 0 011 1v3h3a1 1 0 110 2h-3v3a1 1 0 11-2 0v-3H6a1 1 0 110-2h3V6a1 1 0 011-1z" clip-rule="evenodd" />
									</svg>Appspace
								</router-link>
							</div>
						</li>
					</ul>
				</dd>
			</div>

			<div class="flex justify-end pb-5 px-4 sm:px-6">
				<router-link :to="{name: 'manage-app', params:{id:app.app_id}}" class="btn btn-blue">Manage</router-link>
			</div>

		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';

import type {App} from '../models/apps';

export default defineComponent({
	name: 'AppListItem',
	components: {
		
	},
	props: {
		app: {
			type: Object as PropType<App>,
			required: true
		}
	}
	
});
</script>