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
				</div>
				<div class="px-4 py-5 sm:px-6">
					<h4>Used by Appspaces:</h4>
					<ul v-if="appspaces.as.size !== 0">
						<li v-for="appspace in appspaces.asArray" :key="'appspace-'+appspace.id">
							{{appspace.domain_name}}
							({{appspace.app_version}})
							<router-link :to="{name: 'manage-appspace', params:{id:appspace.id}}" class="btn">Manage</router-link>
						</li>
					</ul>
					<p v-else>[not used by any appspace]</p>
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
					<li v-for="ver in versions" :key="ver.version" class="pl-3 pr-4 py-3 flex items-center justify-between text-sm">
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
						<div class="">
							<span v-if="ver.deleting">deleting</span>
							<span v-else-if="ver.appspaces.length">in use</span>
							<button v-else @click.stop.prevent="deleteVersion(ver.version)" class="btn text-red-700">
								<svg xmlns="http://www.w3.org/2000/svg" class="inline align-bottom h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
								</svg>
								<span class="hidden sm:inline-block">delete</span>
							</button>
						</div>

					</li>
				</ul>

			</div>

			<div class="md:mb-6 my-6 bg-yellow-100 shadow overflow-hidden sm:rounded-lg flex justify-between">
				<div class="px-4 py-5 sm:px-6 ">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Delete App</h3>
					<p class="mt-1 max-w-2xl text-sm text-gray-700">
						<template v-if="delete_app_ok">
							Delete the app and all its versions.
						</template>
						<template v-else>
							Unable to delete: app is used by appspaces.
						</template>
					</p>
				</div>
				<div class="px-4 sm:px-6 flex justify-end">
					<button v-if="!deleting_app" @click.stop.prevent="delApp" class="btn btn-blue self-center" :disabled="!delete_app_ok">delete</button>
					<span v-else>Deleting...</span>
				</div>
			</div>
		</template>
		<BigLoader v-else></BigLoader> 
	</ViewWrap>
</template>


<script lang="ts">
import { defineComponent, ref, reactive, onMounted, onUnmounted, computed } from 'vue';
import router from '../router';
import {setTitle} from '../controllers/nav';
import { App, deleteAppVersion, deleteApp } from '../models/apps';
import type {AppVersion} from '../models/app_versions';
import { Appspaces } from '../models/appspaces'
import type { Appspace } from '../models/appspaces'

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';

interface VersionView extends AppVersion {
	appspaces: Appspace[],
	deleting: boolean
}

export default defineComponent({
	name: 'ManageApp',
	components: {
		ViewWrap,
		BigLoader
	},
	props: {
		id: {
				type: String,
				required: true
		}
	},
	setup(props) {
		const app_id = Number(props.id);
		const app = reactive(new App);

		app.fetch(app_id).then( () => {
			setTitle(app.name);
		});

		const appspaces = reactive(new Appspaces);
		appspaces.fetchForApp(app_id);

		const versions = computed( () => {
			return app.versions.map( v => {
				(v as VersionView).appspaces = appspaces.asArray.filter( a => a.app_version === v.version );
				(v as VersionView).deleting = false;
				return v as VersionView;
			});
		});

		onUnmounted( () => {
			setTitle("");
		});

		async function deleteVersion(version:string) {
			const v_index = versions.value.findIndex( v => v.version === version );
			const v = versions.value[v_index];
			if( v === undefined ) throw new Error("did not find the version");
			if( v.appspaces.length ) {
				alert("Can't delete an app version that is used by appspaces")
				return;
			}

			v.deleting = true;
			await deleteAppVersion(app_id, version);

			app.versions.splice(v_index, 1);
		}

		const delete_app_ok = computed( () => {
			return !appspaces.as.size
		});

		const deleting_app = ref(false);

		async function delApp() {
			deleting_app.value = true;
			await deleteApp(app_id);
			router.push({name: 'apps'});
		}

		return {
			appspaces,
			versions,
			app,
			deleteVersion,
			delete_app_ok, deleting_app, delApp,
		};
	}
});

</script>