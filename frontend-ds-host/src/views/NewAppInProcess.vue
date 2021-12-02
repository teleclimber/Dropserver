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
			<MessageSad v-if="appGetter.meta && appGetter.meta.errors.length" class="my-5" head="Error">
				<p v-for="err in appGetter.meta.errors" :key="'meta-errors-'+err">{{err}}</p>
			</MessageSad>
			<MessageHappy v-else class="mx-4 my-5 rounded " head="Looks good!">
				App checked and no errors were found.
			</MessageHappy>

			<div class="px-4 py-5 sm:px-6" v-if="appGetter.meta && appGetter.meta.version_metadata">
				<dl class="border border-gray-200 rounded divide-y divide-gray-200">
					<DataDef field="App Name">{{appGetter.meta.version_metadata.name}}</DataDef>
					<DataDef field="Version">{{appGetter.meta.version_metadata.version}}</DataDef>
					<DataDef field="App Schema">{{appGetter.meta.version_metadata.schema}}</DataDef>
					<DataDef field="DropServer API">{{appGetter.meta.version_metadata.api_version}}</DataDef>
				</dl>
			</div>
				
			<div class="px-4 py-5 sm:px-6 flex justify-between">
				<button @click="startOver" class="btn">Start Over</button>
				<button v-if="!committing" @click="doCommit" class="btn btn-blue" :disabled="!appGetter.canCommit">Create Application</button>
				<button v-else class="btn btn-blue" disabled="true">Creating Application...</button>
			</div>
		</div>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, onUnmounted } from 'vue';
import router from '../router';

import { App, AppGetter } from '../models/apps';

import ViewWrap from '../components/ViewWrap.vue';
import SelectFiles from '../components/ui/SelectFiles.vue';
import DataDef from '../components/ui/DataDef.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import MessageHappy from '../components/ui/MessageHappy.vue';
import MessageProcessing from '../components/ui/MessageProcessing.vue';

export default defineComponent({
	name: 'NewAppInProcess',
	components: {
		ViewWrap,
		SelectFiles,
		DataDef,
		MessageSad,
		MessageHappy,
		MessageProcessing
	},
	props: {
		app_get_key: {
			type: String,
			required: true
		}
	},
	setup(props) {
		const appGetter = reactive(new AppGetter);
		appGetter.updateKey(props.app_get_key);

		const committing = ref(false);

		async function doCommit() {
			if( !appGetter.canCommit ) return;
			committing.value = true;
			const resp = await appGetter.commit();
			router.replace({name: 'manage-app', params:{id: resp.app_id}});
		}

		async function startOver() {
			await appGetter.cancel();
			router.push({name: 'new-app'});
		}

		onUnmounted( () => {
			appGetter.unsubscribeKey();
		});
		
		return {
			appGetter,
			committing,
			doCommit,
			startOver
		};
	},
});

</script>