<script lang="ts" setup>
import { Ref, ref, reactive, onMounted, watch, computed, watchEffect } from 'vue';
import { useRouter } from 'vue-router';

import { useAppsStore } from '@/stores/apps';
import { useAppGetterStore } from '@/stores/appgetter';

import { LiveLog } from '../models/log';

import ViewWrap from '../components/ViewWrap.vue';

import MessageSad from '../components/ui/MessageSad.vue';
import MessageWarn from '@/components/ui/MessageWarn.vue';
import MessageProcessing from '../components/ui/MessageProcessing.vue';
import AppCard from '@/components/app/AppCard.vue';
import Manifest from '@/components/app/Manifest.vue';
import LogViewer from '../components/ui/LogViewer.vue';

const router = useRouter();

const props = defineProps<{
	app_id?: number
	app_get_key: string
}>();

const appsStore = useAppsStore();

const appGetterStore = useAppGetterStore();

const desc_str = computed( () => props.app_id === undefined ? 'App' : 'Version' );

appGetterStore.loadKey(props.app_get_key);

const appGet = appGetterStore.loadKey(props.app_get_key);

const meta = computed( () => appGet.meta.value );
const manifest = computed( () => meta.value?.version_manifest );

const app_icon = computed( () => {
	if( !manifest.value ) return "";
	return `/api/application/in-process/${props.app_get_key}/file/app-icon`;
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
	if( !appGet.must_confirm ) return;
	committing.value = true;
	const new_app_id = await appsStore.commitNewApplication(appGet.key);
	//router.replace({name: 'manage-app', params:{id: new_app_id}});
	// The watchEffect below now does the router operation.
}

watchEffect( () => {
	if( !appGet.done || appGet.expects_input || appGet.has_error ) return;
	const app_id = appGet.meta.value?.app_id;
	if( !app_id ) console.error("Expected an app_id")
	router.replace({name: 'manage-app', params:{id: app_id}});
});

const show_details = computed( () => {
	if( appGet.version_manifest === undefined ) return false;
	return (appGet.done && appGet.has_error) || appGet.must_confirm;
});

const changelog = ref("");
watch( show_details, async () => {
	if( !show_details.value ) return;
	changelog.value = "";
	const resp = await fetch(`/api/application/in-process/${props.app_get_key}/changelog`);
	changelog.value = await resp.text();
}, { immediate: true });

const show_log = ref(false);
const live_log = reactive(new LiveLog);
watch( show_details, () => {
	if( show_details.value ) live_log.initInProcessAppLog(props.app_get_key);
});

async function startOver() {
	await appGet.cancel();
	if( props.app_id === undefined ) router.push({name: 'new-app'});
	else router.push( {name:'new-app-version', params:{id:props.app_id}});
}
async function cancel() {
	await appGet.cancel();
	if( props.app_id === undefined ) router.push({name: 'apps'});
	else router.push( {name:'manage-app', params:{id:props.app_id}});
}

</script>

<template>
	<ViewWrap>
		<MessageSad v-if="appGet.not_found.value" class="" head="Sorry, unable to find that">
			It's possible the app files have been removed because it's been too long. Please try again.
			<div class="pt-5 flex ">
				<button @click="startOver" class="btn">Start Over</button>
			</div>
		</MessageSad>

		<MessageProcessing v-else-if="!appGet.must_confirm && !appGet.done" class="" head="Processing...">
			<p v-if="appGet.last_event.value">{{appGet.last_event.value.step}}</p>
			<p v-else>Getting info...</p>
			<div class="pt-5 flex ">
				<button @click="startOver" class="btn">Cancel</button>
			</div>
		</MessageProcessing>

		<MessageSad v-if="meta && meta.errors.length" class="" head="Error">
			<p v-for="err in meta.errors" :key="'meta-errors-'+err">{{err}}</p>
			<p class="mt-2"><a href="#" @click.stop.prevent="startOver" class="btn">start over</a></p>
		</MessageSad>

		<div v-if="show_details" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">
					Review New {{ desc_str }}
				</h3>
			</div>

			<MessageWarn v-if="meta?.warnings.length" class="mx-4 sm:mx-6 my-5 rounded" head="Warning">
				<p>App can be installed but some issues were found.
					Please review the warnings below before continuing.</p>
			</MessageWarn>

			<AppCard v-if="manifest" :manifest="manifest" :icon_url="app_icon"></AppCard>

			<div class="px-4 sm:px-2 mx-auto max-w-xl font-medium mt-6">What's new:</div>
			<div class="bg-gray-100 px-4 sm:px-2 py-2 mx-auto max-w-xl max-h-48 overflow-y-scroll mb-6">
				<pre class="text-sm whitespace-pre-wrap">{{ changelog || "No changelog :(" }}</pre>
			</div>

			<Manifest v-if="manifest" :manifest="manifest" :warnings="meta?.warnings"></Manifest>

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
						:disabled="appGet.has_error || committing"
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