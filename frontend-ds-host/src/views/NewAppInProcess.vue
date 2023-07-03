<script lang="ts" setup>
import { Ref, ref, reactive, onMounted, onUnmounted, watch, computed, ComputedRef } from 'vue';
import { useRouter } from 'vue-router';

import { useAppsStore, AppGetter } from '@/stores/apps';
import { LiveLog } from '../models/log';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import MessageWarn from '@/components/ui/MessageWarn.vue';
import MessageProcessing from '../components/ui/MessageProcessing.vue';
import LogViewer from '../components/ui/LogViewer.vue';
import AppLicense from '@/components/app/AppLicense.vue';
import SmallMessage from '@/components/ui/SmallMessage.vue';

import { getLoadState } from '@/stores/loadable';
import { LoadState, AppVersionUI } from '@/stores/types';

const router = useRouter();

const props = defineProps<{
	app_id?: number
	app_get_key: string
}>();

const appsStore = useAppsStore();

const desc_str = computed( () => props.app_id === undefined ? 'App' : 'Version' );

const appGetter = new AppGetter;
appGetter.updateKey(props.app_get_key);

const meta = computed( () => appGetter.meta.value );
const manifest = computed( () => meta.value?.version_manifest );
const warnings :ComputedRef<Record<string, string>> = computed( () => {
	if( !meta.value || !meta.value.warnings ) return {};
	else return meta.value.warnings;
});

const app_icon_error = ref(false);
const app_icon = computed( () => {
	if( app_icon_error.value || !manifest.value ) return "";
	return `/api/application/in-process/${props.app_get_key}/file/app-icon`;
});

const accent_color = computed( () => {
	if( manifest.value && manifest.value.accent_color ) return manifest.value.accent_color;
	return 'rgb(135, 151, 164)';
});

const release_date = computed( () => {
	if( !manifest.value?.release_date ) return;
	return new Date(manifest.value.release_date).toLocaleDateString(undefined, {
		dateStyle:'medium'
	});
});

const show_log = ref(false);
const live_log = reactive(new LiveLog);
watch( () => appGetter.done, () => {
	if( appGetter.done ) live_log.initInProcessAppLog(props.app_get_key);
});

// Get next and prev versions, if any according to app getter meta.
const app_versions = computed( () => {
	if( props.app_id === undefined ) return undefined;
	appsStore.loadAppVersions(props.app_id);
	const av = appsStore.mustGetAppVersions(props.app_id);
	return av;
});
const sibling_versions :ComputedRef<{prev?: AppVersionUI, next?: AppVersionUI}> = computed( () => {
	if( !app_versions.value ) return {};
	if( getLoadState(app_versions.value) !== LoadState.Loaded ) return {};
	return {
		prev: app_versions.value.find( v => v.version === meta.value?.prev_version ),
		next: app_versions.value.find( v => v.version === meta.value?.next_version )
	}
});
const prev = computed( () => !!meta.value?.prev_version );
const next = computed( () => !!meta.value?.next_version );

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

const small_msg_classes = ['inline-block', 'mt-1'];
const link_classes = ['text-blue-500', 'hover:underline', 'hover:text-blue-600' ];

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
			<MessageWarn v-else-if="meta && Object.keys(warnings).length" class="mx-4 sm:mx-6 my-5 rounded" head="Warning">
				<p>App can be installed but some issues were found.
					Please review the warnings below before continuing.</p>
			</MessageWarn>

			<!-- app version card -->
			<div class="my-8 px-4 sm:px-6" v-if="manifest">
				<div class=" mx-auto max-w-xl pb-4 bg-white shadow overflow-hidden border-2" style="border-top-width:1rem" :style="'border-color:'+accent_color" >
					<div class="grid app-grid gap-x-2 gap-y-2 px-2 py-2 ">
						<img v-if="app_icon" :src="app_icon" @error="app_icon_error = true" class="w-20 h-20" />
						<div v-else class="w-20 h-20 text-gray-300 flex justify-center items-center">
							<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-14 h-14">
								<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" />
							</svg>
						</div>
						<div class="self-center">
							<h3 class="text-2xl leading-6 font-medium text-gray-900">{{manifest.name}}</h3>
							<p class="italic" v-if="manifest.short_description">“{{manifest.short_description}}”</p>
						</div>
						<div class="col-span-3 sm:col-start-2">
							<p class="text-lg font-medium">
								Version 
								<span class="bg-gray-200 text-gray-700 px-1 rounded-md">{{manifest.version}}</span>
								<span v-if="release_date"> released {{ release_date || '' }}</span>
							</p>
						</div>
					</div>
				</div>
			</div>

			<div v-if="manifest">
				<DataDef field="App name:">
					<p class="font-medium text-lg">{{ manifest.name }}</p>
					<SmallMessage v-if="prev && sibling_versions.prev?.name !== manifest.name" mood="warn" :class="small_msg_classes">
						Name changed since {{ sibling_versions.prev?.version }}! 
						Was: “<span class="font-medium">{{ sibling_versions.prev?.name }}</span>”
					</SmallMessage>
					<SmallMessage v-else-if="next && manifest.name !== sibling_versions.next?.name" mood="warn" :class="small_msg_classes">
						Name changed! App is called “<span class="font-medium">{{ sibling_versions.next?.name  }}</span>”
						in {{  sibling_versions.next?.version }}
					</SmallMessage>
					<SmallMessage v-if="warnings['name']" mood="warn" :class="small_msg_classes">{{ warnings['name'] }}</SmallMessage>
				</DataDef>

				<DataDef field="Version:">
					<p class="">
						<span class="font-medium text-lg bg-gray-200 text-gray-700 px-1 rounded-md">{{ manifest.version }}</span>
						<span v-if="release_date"> released {{ release_date || '' }}</span>
					</p>
					<SmallMessage v-if="prev && next" mood="info" :class="small_msg_classes">
						This version comes between previously uploaded versions {{ sibling_versions.prev?.version }}
						and {{ sibling_versions.next?.version }}.
					</SmallMessage>
					<!-- <SmallMessage v-else-if="prev" mood="info" :class="small_msg_classes">
						Previous uploaded version: {{ sibling_versions.prev?.version }}.
					</SmallMessage> -->
					<SmallMessage v-else-if="next" mood="info" :class="small_msg_classes">
						This version comes before previously uploaded version {{ sibling_versions.next?.version }}.
					</SmallMessage>
				</DataDef>

				<DataDef field="Data schema:">
					<p class="font-medium text-lg">{{ manifest.schema }}</p>
					<SmallMessage v-if="sibling_versions.prev && sibling_versions.prev.schema < manifest.schema" mood="info" :class="small_msg_classes">
						The previous version ({{ sibling_versions.prev?.version }}) has a schema of 
						{{  sibling_versions.prev.schema }}. 
					</SmallMessage>
					<SmallMessage v-if="sibling_versions.prev && sibling_versions.prev.schema > manifest.schema"  mood="warn" :class="small_msg_classes">
						Error: The previous version ({{ sibling_versions.prev?.version }}) has a schema of 
						{{  sibling_versions.prev.schema }}. 
					</SmallMessage>
					<SmallMessage v-if="sibling_versions.next && sibling_versions.next.schema < manifest.schema" mood="warn" :class="small_msg_classes">
						Error: The next version ({{ sibling_versions.next.version }}) has a schema of 
						{{  sibling_versions.next.schema }}. 
					</SmallMessage>
				</DataDef>

				<DataDef field="License:">
					<p><AppLicense :license="manifest.license" ></AppLicense></p>
					<SmallMessage v-if="prev && sibling_versions.prev?.license !== manifest.license"  mood="warn" :class="small_msg_classes">
						License changed since {{ sibling_versions.prev?.version }}!
						Was: <AppLicense :license="sibling_versions.prev?.license"></AppLicense>
					</SmallMessage>
					<SmallMessage v-else-if="next && manifest.license !== sibling_versions.next?.license"  mood="warn" :class="small_msg_classes">
						License change: license is <AppLicense :license="sibling_versions.next?.license"></AppLicense>
						in {{ sibling_versions.next?.version }}
					</SmallMessage>
					<SmallMessage v-if="warnings['license']" mood="warn" :class="small_msg_classes">{{ warnings['license'] }}</SmallMessage>
				</DataDef>

				<DataDef field="Authors:">
					<ul>
						<li v-for="a in manifest.authors" class="">
							<span class="mr-2">{{ a.name }} </span>
							<a v-if="a.url" :href="a.url" class="mr-2 uppercase" :class="link_classes">
								<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 inline-block align-text-bottom">
									<path d="M16.555 5.412a8.028 8.028 0 00-3.503-2.81 14.899 14.899 0 011.663 4.472 8.547 8.547 0 001.84-1.662zM13.326 7.825a13.43 13.43 0 00-2.413-5.773 8.087 8.087 0 00-1.826 0 13.43 13.43 0 00-2.413 5.773A8.473 8.473 0 0010 8.5c1.18 0 2.304-.24 3.326-.675zM6.514 9.376A9.98 9.98 0 0010 10c1.226 0 2.4-.22 3.486-.624a13.54 13.54 0 01-.351 3.759A13.54 13.54 0 0110 13.5c-1.079 0-2.128-.127-3.134-.366a13.538 13.538 0 01-.352-3.758zM5.285 7.074a14.9 14.9 0 011.663-4.471 8.028 8.028 0 00-3.503 2.81c.529.638 1.149 1.199 1.84 1.66zM17.334 6.798a7.973 7.973 0 01.614 4.115 13.47 13.47 0 01-3.178 1.72 15.093 15.093 0 00.174-3.939 10.043 10.043 0 002.39-1.896zM2.666 6.798a10.042 10.042 0 002.39 1.896 15.196 15.196 0 00.174 3.94 13.472 13.472 0 01-3.178-1.72 7.973 7.973 0 01.615-4.115zM10 15c.898 0 1.778-.079 2.633-.23a13.473 13.473 0 01-1.72 3.178 8.099 8.099 0 01-1.826 0 13.47 13.47 0 01-1.72-3.178c.855.151 1.735.23 2.633.23zM14.357 14.357a14.912 14.912 0 01-1.305 3.04 8.027 8.027 0 004.345-4.345c-.953.542-1.971.981-3.04 1.305zM6.948 17.397a8.027 8.027 0 01-4.345-4.345c.953.542 1.971.981 3.04 1.305a14.912 14.912 0 001.305 3.04z" />
								</svg>
								web
							</a>
							<a v-if="a.email" :href="'mailto:'+a.email" class="uppercase" :class="link_classes">
								<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 inline-block align-text-bottom">
									<path d="M3 4a2 2 0 00-2 2v1.161l8.441 4.221a1.25 1.25 0 001.118 0L19 7.162V6a2 2 0 00-2-2H3z" />
									<path d="M19 8.839l-7.77 3.885a2.75 2.75 0 01-2.46 0L1 8.839V14a2 2 0 002 2h14a2 2 0 002-2V8.839z" />
								</svg>
								email
							</a>
						</li>
					</ul>
					<SmallMessage v-if="warnings['authors']" mood="warn" :class="small_msg_classes">{{ warnings['authors'] }}</SmallMessage>
					<p v-if="!manifest.authors || manifest.authors.length === 0" class="text-gray-400 italic">No authors listed</p>
				</DataDef>

				<DataDef field="Website:">
					<a v-if="manifest.website" :href="manifest.website" :class="link_classes">{{ manifest.website }}</a>
					<p v-else class="text-gray-400 italic">No website listed</p>
					<SmallMessage v-if="warnings['website']" mood="warn" :class="small_msg_classes">{{ warnings['website'] }}</SmallMessage>
				</DataDef>

				<DataDef field="Code repository:">
					<a v-if="manifest.code" :href="manifest.code" :class="link_classes">{{ manifest.code }}</a>
					<p v-else class="text-gray-400 italic">No code repository listed</p>
					<SmallMessage v-if="warnings['code']" mood="warn" :class="small_msg_classes">{{ warnings['code'] }}</SmallMessage>
				</DataDef>

				<DataDef field="Funding:">
					<a v-if="manifest.funding" :href="manifest.funding" :class="link_classes">{{ manifest.funding }}</a>
					<p v-else class="text-gray-400 italic">No funding website listed</p>
					<SmallMessage v-if="warnings['funding']" mood="warn" :class="small_msg_classes">{{ warnings['funding'] }}</SmallMessage>
				</DataDef>

				<DataDef v-if="warnings['icon']" field="Icon:">
					<SmallMessage mood="warn" :class="small_msg_classes">{{ warnings['icon'] }}</SmallMessage>
				</DataDef>
				<DataDef v-if="warnings['accent-color']" field="Accent color:">
					<SmallMessage mood="warn" :class="small_msg_classes">{{ warnings['accent-color'] }}</SmallMessage>
				</DataDef>
				<DataDef v-if="warnings['short-description']" field="Short description:">
					<p>“{{ manifest.short_description }}”</p>
					<SmallMessage mood="warn" :class="small_msg_classes">{{ warnings['short-description'] }}</SmallMessage>
				</DataDef>
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

<style scoped>
.app-grid {
	grid-template-columns: 5rem 1fr max-content;
}
</style>