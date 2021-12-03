<template>
	<ViewWrap>

		<MessageSad v-if="appGetter.not_found" class="" head="Sorry, unable to find that">
			It's possible the app files have been removed because it's been too long. Please try again.

			<div class="pt-5 flex ">
				<button @click="startOver" class="btn">Start Over</button>
			</div>
		</MessageSad>
		<MessageProcessing v-else-if="!appGetter.done" class="" head="Processing...">
			<p v-if="appGetter.last_event">{{appGetter.last_event.step}}</p>
			<p v-else>Getting info...</p>

			<div class="pt-5 flex ">
				<button @click="startOver" class="btn">Cancel</button>
			</div>
		</MessageProcessing>

		<div v-if="appGetter.done" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Review</h3>
			</div>
			<MessageSad v-if="appGetter.meta && appGetter.meta.errors.length" class="mx-4 sm:mx-6 my-5 rounded" head="Error">
				<p v-for="err in appGetter.meta.errors" :key="'meta-errors-'+err">{{err}}</p>
			</MessageSad>
			<MessageHappy v-else class="mx-4 sm:mx-6 my-5 rounded " head="Looks good!">
				App version checked and no errors were found.
			</MessageHappy>

			<div class="px-4 py-5 sm:px-6" v-if="appGetter.meta && appGetter.meta.version_metadata">
				<dl class="border border-gray-200 rounded divide-y divide-gray-200">
					<DataDef field="App Name">{{appGetter.meta.version_metadata.name}}</DataDef>
					<DataDef field="Version">{{appGetter.meta.version_metadata.version}}</DataDef>
					<DataDef field="App Schema">{{appGetter.meta.version_metadata.schema}}</DataDef>
					<DataDef field="DropServer API">{{appGetter.meta.version_metadata.api_version}}</DataDef>
				</dl>
			</div>

			<div class="md:mb-6 my-6 overflow-hidden ">
				<div class="px-4 py-2 sm:px-6">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Existing Versions:</h3>
				</div>

				<ul class="border-t border-b border-gray-200 divide-y divide-gray-200">
					<li v-for="ver in versions" :key="ver.version" class="px-4 py-2 flex items-center justify-between text-sm" :class="{'bg-yellow-100':ver.is_uploaded}">
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
						<div v-if="ver.is_uploaded" class="w-0 flex-1 flex items-center">
							<span class="ml-2 flex-1 w-0">
								UPLOADED
							</span>
						</div>
						<div v-else class="w-0 flex-1 flex items-center">
							<span class="ml-2 flex-1 w-0">
								{{ver.created_dt.toLocaleString()}}
							</span>
						</div>
					</li>
				</ul>
			</div>

			<div class="my-6">
				<div class="px-4 py-2 sm:px-6">
					<h3 class="text-lg leading-6 font-medium text-gray-900">App Log:</h3>
				</div>
				<div class="mx-6 border border-gray-200">
					<LogViewer :live_log="live_log"></LogViewer>
				</div>
			</div>
				
			<div class="px-4 py-5 sm:px-6 flex justify-between">
				<button @click="startOver" class="btn">Start Over</button>
				<button v-if="!committing" @click="doCommit" class="btn btn-blue" :disabled="!appGetter.canCommit">Create App Version</button>
				<button v-else class="btn btn-blue" disabled="true">Creating App Version...</button>
			</div>
		</div>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue';
import router from '../router';
import {useRoute} from 'vue-router';

import { App, AppGetter } from '../models/apps';
import {LiveLog} from '../models/log';

import {setTitle} from '../controllers/nav';

import ViewWrap from '../components/ViewWrap.vue';
import SelectFiles from '../components/ui/SelectFiles.vue';
import DataDef from '../components/ui/DataDef.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import MessageHappy from '../components/ui/MessageHappy.vue';
import MessageProcessing from '../components/ui/MessageProcessing.vue';
import LogViewer from '../components/ui/LogViewer.vue';

type displayVer = {
	version:string,
	schema: number,
	api_version: number,
	is_uploaded: boolean,
	created_dt: Date
}

export default defineComponent({
	name: 'NewAppVersionInProcess',
	components: {
		ViewWrap,
		SelectFiles,
		DataDef,
		MessageSad,
		MessageHappy,
		MessageProcessing,
		LogViewer
	},
	props: {
		app_get_key: {
			type: String,
			required: true
		}
	},
	setup(props) {
		const route = useRoute();
		const app = reactive(new App);
		onMounted( async () => {
			await app.fetch(Number(route.params.id));
			setTitle(app.name);
		});
		onUnmounted( () => {
			setTitle("");
		});

		const appGetter = reactive(new AppGetter);
		appGetter.updateKey(props.app_get_key);

		const live_log = reactive(new LiveLog);
		watch( () => appGetter.done, () => {
			if( appGetter.done ) live_log.initInProcessAppLog(props.app_get_key);
		});

		const committing = ref(false);

		async function doCommit() {
			if( !appGetter.canCommit ) return;
			committing.value = true;
			const resp = await appGetter.commit();
			router.push({name: 'manage-app', params:{id: resp.app_id}});
		}

		async function startOver() {
			await appGetter.cancel();
			router.push({name: 'new-app-version'});
		}

		const versions = computed( () => {
			const ret :displayVer[] = app.versions.map( v => {
				return {
					version: v.version,
					schema:v.schema,
					api_version: v.api_version,
					is_uploaded: false,
					created_dt: v.created_dt };
			});
			if( appGetter.meta?.version_metadata !== undefined ) {
				const m = appGetter.meta?.version_metadata;
				const resp = appGetter.meta;
				const uv = {
					version: m.version,
					schema:m.schema,
					api_version: m.api_version,
					is_uploaded: true,
					created_dt: new Date
				};
				if( resp.prev_version ) {
					const index = ret.findIndex( v => v.version === resp.prev_version );
					if( index !== -1 ) ret.splice(index, 0, uv );
				}
				else if( resp.next_version ) {
					ret.push(uv);
				}
			}
			return ret;
		});
		
		return {
			appGetter,
			committing,
			doCommit,
			startOver,
			versions,
			live_log
		};
	},
});

</script>