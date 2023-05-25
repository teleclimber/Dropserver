<script lang="ts" setup>
import { Ref, ref, reactive, onMounted, onUnmounted, watch, computed } from 'vue';
import { useRouter } from 'vue-router';

import { useAppsStore, AppGetter } from '@/stores/apps';
import { LiveLog } from '../models/log';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import MessageWarn from '@/components/ui/MessageWarn.vue';
import MessageProcessing from '../components/ui/MessageProcessing.vue';
import LogViewer from '../components/ui/LogViewer.vue';

const router = useRouter();

const props = defineProps<{
	app_id?: number
	app_get_key: string
}>();

const appsStore = useAppsStore();
const app = computed( () => {
	if( props.app_id === undefined ) return undefined;
	appsStore.loadData();
	if( !appsStore.is_loaded ) return;
	const a = appsStore.getApp(props.app_id);
	if( a === undefined ) return;
	return a.value;
});

const desc_str = computed( () => props.app_id === undefined ? 'App' : 'Version' );

const appGetter = new AppGetter;
appGetter.updateKey(props.app_get_key);

const meta = computed( () => appGetter.meta.value );
const manifest = computed( () => meta.value?.version_manifest );

const show_log = ref(false);
const live_log = reactive(new LiveLog);
watch( () => appGetter.done, () => {
	if( appGetter.done ) live_log.initInProcessAppLog(props.app_get_key);
});

type displayVer = {
	version:string,
	schema: number,
	is_uploaded: boolean,
	created_dt: Date
}
const versions = computed( () => {
	if( app.value === undefined ) return;
	const ret :displayVer[] = app.value.versions.map( v => {
		return {
			version: v.version,
			schema:v.schema,
			is_uploaded: false,
			created_dt: v.created_dt };
	});
	if( meta.value !== undefined && meta.value.version_manifest !== undefined ) {
		const m = meta.value.version_manifest;
		const uv = {
			version: m.version,
			schema: m.schema,
			is_uploaded: true,
			created_dt: new Date
		};
		if( meta.value.prev_version ) {
			const p = meta.value.prev_version;
			const index = ret.findIndex( v => v.version === p );
			if( index !== -1 ) ret.splice(index, 0, uv );
		}
		else if( meta.value.next_version ) {
			ret.push(uv);
		}
	}
	return ret;
});

const create_button :Ref<HTMLInputElement|undefined> = ref();
onMounted( () => {
	if( create_button.value ) create_button.value.focus();
});
watch( create_button, () => {
	if( create_button.value ) create_button.value.focus();
});
const committing = ref(false);

async function doCommit() {
	if( !appGetter.canCommit ) return;
	committing.value = true;
	const new_app_id = await appsStore.commitNewApplication(appGetter.key.value);
	router.replace({name: 'manage-app', params:{id: new_app_id}});
}

async function startOver() {
	await appGetter.cancel();
	if( props.app_id === undefined ) router.push({name: 'new-app'});
	else router.push( {name:'new-app-version', params:{id:props.app_id}});
}
async function cancel() {
	await appGetter.cancel();
	if( props.app_id === undefined ) router.push({name: 'apps'});
	else router.push( {name:'manage-app', params:{id:props.app_id}});
}

onUnmounted( () => {
	appGetter.unsubscribeKey();
});

</script>

<template>
	<ViewWrap>

		<MessageSad v-if="appGetter.not_found.value" class="" head="Sorry, unable to find that">
			It's possible the app files have been removed because it's been too long. Please try again.

			<div class="pt-5 flex ">
				<button @click="startOver" class="btn">Start Over</button>
			</div>
		</MessageSad>
		<MessageProcessing v-else-if="!appGetter.done" class="" head="Processing...">
			<p v-if="appGetter.last_event.value">{{appGetter.last_event.value.step}}</p>
			<p v-else>Getting info...</p>

			<div class="pt-5 flex ">
				<button @click="startOver" class="btn">Cancel</button>
			</div>
		</MessageProcessing>

		<div v-if="appGetter.done" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">
					Review New {{ desc_str }}
				</h3>
			</div>
			<MessageSad v-if="meta && meta.errors.length" class="mx-4 sm:mx-6 my-5 rounded" head="Error">
				<p v-for="err in meta.errors" :key="'meta-errors-'+err">{{err}}</p>
			</MessageSad>
			<MessageWarn v-else-if="meta && Object.keys(meta.warnings).length" class="mx-4 sm:mx-6 my-5 rounded" head="Warning">
				<p>App can be installed but some issues were found.
					Please review the warnings below before continuing.</p>
			</MessageWarn>

			<div class="my-5" v-if="manifest">
				<div class="px-4 sm:px-6">
					<h2 class="text-2xl font-medium">{{manifest.name}}</h2>
					<p v-if="manifest.short_description" class="text-gray-600 italic">{{ manifest.short_description }}</p>
					<p>
						Version {{manifest.version}}
						<template v-if="manifest.release_date">(released on {{ manifest.release_date.toLocaleDateString() }})</template>
					</p>
				</div>
				<!-- here present app a bit better: BIG name, icon on left, short desc below (if provided), 
					author, version and date, all the clearly user-interesting stuff should be here, nicely presented. -->
				<!-- However some of this will be very different for new version of existing app! -->
				<DataDef field="App Name:">{{manifest.name}}</DataDef>
				<DataDef field="Version:">{{manifest.version}}</DataDef>
				<DataDef field="Data Schema:">{{manifest.schema}}</DataDef>
				<DataDef field="Icon:">
					{{ manifest.icon }}
					<p v-if="meta?.warnings.icon" class="text-orange-500">
						<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 inline">
							<path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
						</svg>
						{{ meta?.warnings.icon }}
					</p>
				</DataDef>
			</div>

			<div v-if="versions" class=" md:mx-6 my-6 overflow-hidden ">
				<div class="mx-4 md:mx-0 py-2 ">
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
						<div v-if="ver.is_uploaded" class="w-0 flex-1 flex items-center">
							<span class="ml-2 flex-1 w-0 italic text-gray-500">
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

			<div class="my-6 px-4 sm:px-6">
				<div class="py-2 flex justify-between items-baseline">
					<h3 class="text-lg leading-6 font-medium text-gray-900">App Log:</h3>
					<button v-if="show_log" class="btn" @click="show_log = !show_log">hide</button>
					<button v-else class="btn" @click="show_log = !show_log">show</button>
				</div>

				<div v-if="show_log" class="border border-gray-200">
					<LogViewer :live_log="live_log"></LogViewer>
				</div>
				<div v-else class="border border-gray-200 bg-gray-50 text-sm italic text-gray-500 p-2 rounded" @click="show_log = !show_log">Click to show app log...</div>
			</div>
			
			<form @submit.prevent="doCommit" @keyup.esc="cancel">
				<div class="px-4 py-5 sm:px-6 flex justify-between">
					<input type="button" class="btn" @click="cancel" value="Cancel" />
					<input
						ref="create_button"
						type="submit"
						class="btn-blue"
						:disabled="!appGetter.canCommit || committing"
						value="Finish" />
						<!-- TODO tweak submit messge for version-->
				</div>
			</form>
		</div>
	</ViewWrap>
</template>
