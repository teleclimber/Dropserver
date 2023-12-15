<script lang="ts" setup>
import { ref, Ref, watch } from "vue";
import { useRouter } from 'vue-router';
import { useAppsStore, AppGetMeta } from '@/stores/apps';
import ViewWrap from '@/components/ViewWrap.vue';
import AppCard from "./AppCard.vue";
import Manifest from "./Manifest.vue";

const props = defineProps<{
	app_id: number,
	version?: string
}>();

const router = useRouter();

const appsStore = useAppsStore();

const getMeta :Ref<AppGetMeta|undefined> = ref();
watch( props, async () => {
	getMeta.value = await appsStore.fetchVersionManifest(props.app_id, props.version);
}, {immediate: true});

const has_error = ref(false);	// TODO: true if there are any fatal errors in validation

const submitting = ref(false);
async function doInstall() {
	let version = props.version || getMeta.value?.version_manifest?.version;
	if( !version ) return;
	submitting.value = true;
	const app_get_key = await appsStore.getNewVersionFromURL(props.app_id, version);
	submitting.value = false;
	router.push({name: 'new-app-in-process', params:{app_get_key}});
}

async function cancel() {
	router.push( {name:'manage-app', params:{id:props.app_id}});
}

</script>

<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">
					Details for version {{ version }}
				</h3>
			</div>

			<!-- fatal errors that prevent installation should be shown here. -->

			<AppCard v-if="getMeta?.version_manifest" :manifest="getMeta.version_manifest" :icon_url="''"></AppCard>
			<Manifest v-if="getMeta?.version_manifest" :manifest="getMeta.version_manifest" :warnings="getMeta.warnings"></Manifest>

			<form @submit.prevent="doInstall" @keyup.esc="cancel">
				<div class="px-4 py-5 sm:px-6 flex justify-between">
					<input type="button" class="btn" @click="cancel" value="Cancel" />
					<input
						ref="create_button"
						type="submit"
						class="btn-blue"
						:disabled="has_error || submitting"
						value="Install" />
				</div>
			</form>

		</div>
	</ViewWrap>
</template>

