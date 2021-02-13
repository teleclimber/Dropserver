<template>
	<ViewWrap>
		<template v-if="app.loaded">
			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Application</h3>
				</div>
				<div class="px-4 py-5 sm:px-6">
					<p>Application created {{app.created_dt.toLocaleString()}}</p>
					<p>[Description...]</p>
					<p>[Usage...]</p>
				</div>
			</div>

			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 flex justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Versions</h3>
					<div >
						<router-link :to="{name: 'new-app-version', params:{id:app.app_id}}" class="btn btn-blue">Upload New Version</router-link>
					</div>
				</div>

				<ul class="border-t border-b border-gray-200 divide-y divide-gray-200">
					<li v-for="ver in app.versions" :key="ver.version" class="pl-3 pr-4 py-3 flex items-center justify-between text-sm">
						<div class="w-0 flex-1 flex items-center">
							<span class="ml-2 flex-1 w-0 font-bold">
								{{ver.version}}
							</span>
						</div>
						<div class="w-0 flex-1 flex items-center">
							<span class="ml-2 flex-1 w-0 ">
								{{ver.schema}}
							</span>
						</div>
						<div class="w-0 flex-1 flex items-center">
							<span class="ml-2 flex-1 w-0 ">
								{{ver.api_version}}
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

				
			</div>
		</template>
		<BigLoader v-else></BigLoader> 
	</ViewWrap>
</template>


<script lang="ts">
import {useRoute} from 'vue-router';
import { defineComponent, ref, reactive, onMounted, onUnmounted } from 'vue';

import {setTitle} from '../controllers/nav';
import { App } from '../models/apps';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';

export default defineComponent({
	name: 'ManageApp',
	components: {
		ViewWrap,
		BigLoader
	},
	setup() {
		const route = useRoute();
		const app = reactive(new App);
		onMounted( async () => {
			await app.fetch(Number(route.params.id));
			setTitle(app.name);
		});
		onUnmounted( () => {
			setTitle("");
		});

		return {
			app,
		};
	}
});

</script>