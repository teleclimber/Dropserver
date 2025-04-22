<script lang="ts" setup>
import { shallowRef, ref, Ref, reactive, watch, onMounted, ComputedRef, computed, isReactive } from 'vue';
import { useRouter } from 'vue-router';

import { useAppspacesStore } from '@/stores/appspaces';
import { useAppsStore } from '@/stores/apps';
import { useDropIDsStore } from '@/stores/dropids';
import { App } from '@/stores/types';

import { DomainNames, checkAppspaceDomain } from '../models/domainnames';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '@/components/ui/DataDef.vue';
import MessageSad from '../components/ui/MessageSad.vue';

const props = defineProps<{
	app_id?: number,
	version?: string
}>();

const router = useRouter();

const appspacesStore = useAppspacesStore();

const appsStore = useAppsStore();
appsStore.loadData();

const dropIDsStore = useDropIDsStore();
dropIDsStore.loadData();

const app_pick_elem :Ref<HTMLInputElement|undefined> = ref();
const domain_pick_elem :Ref<HTMLInputElement|undefined> = ref(); 

const app_pick :Ref<number|undefined> = ref();
function setInitialAppID() {
	if( props.app_id !== undefined ) app_pick.value = props.app_id;
}

const picked_app :ComputedRef<App|undefined> = computed( () => {
	if( app_pick.value === undefined || !appsStore.is_loaded ) return;
	return appsStore.getApp(app_pick.value)?.value;
});

const version_options :Ref<string[]> = shallowRef([]);
watch( picked_app, async () => {
	if( picked_app.value === undefined ) return;
	await appsStore.loadAppVersions(picked_app.value.app_id);
	const versions = appsStore.mustGetAppVersions(picked_app.value.app_id);
	version_options.value = versions.map( v => v.version ).reverse();
});

const version_pick :Ref<string> = ref("");
watch( version_options, () => {
	if( !picked_app.value ) {
		version_pick.value = '';
		return;
	}
	if( props.version && props.app_id === picked_app.value.app_id ) {
		version_pick.value = props.version;
		return;
	}
	if( picked_app.value.cur_ver !== undefined ) {
		version_pick.value = picked_app.value.cur_ver;
		return;
	}
	if( version_options.value?.length ) {
		version_pick.value = version_options.value[0];
	}
});

onMounted( () => {
	setInitialAppID();
	if( props.app_id === undefined ) {
		if( app_pick_elem.value ) app_pick_elem.value.focus(); 
	}
	else if( domain_pick_elem.value ) domain_pick_elem.value.focus();
});

const domain_names = reactive(new DomainNames);
domain_names.fetchForOwner();

const domain_name = ref("");
const subdomain = ref("");

watch( domain_names, () => {
	if( domain_names.loaded && domain_names.for_appspace.length !== 0 && domain_name.value === '') {
		domain_name.value = domain_names.for_appspace[0].domain_name;
	}
});

const full_domain = computed( () => {
	let ret = domain_name.value;
	if( subdomain.value != "" ) ret = subdomain.value + "."+ domain_name.value;
	return ret;
});

const domain_valid = ref("");
const domain_valid_classes = ref([""]);

const classes_bad = ["bg-red-100", "text-red-800"];
const classes_neutral = ["bg-gray-100", "text-gray-700"]
watch( [domain_name, subdomain], async () => {
	subdomain.value = subdomain.value.trim();

	if( domain_name.value === '' ) {
		domain_valid.value = '...';
		domain_valid_classes.value = classes_neutral;
		return;
	}

	const domain_data = domain_names.for_appspace.find( d => d.domain_name === domain_name.value );
	if( domain_data === undefined ) return;

	if( subdomain.value === "" && domain_data.appspace_subdomain_required ) {
		domain_valid.value = 'subdomain required';
		domain_valid_classes.value = classes_bad;
		return;
	}
	if( subdomain.value.length > 62 ) {
		domain_valid.value = 'long';
		domain_valid_classes.value = classes_bad;
		return;
	}
	// check for bad chars
	
	// Here we query the server to see if the id already exists.
	// Note this is a pretty poor way to do this.
	domain_valid.value = 'checking';
	domain_valid_classes.value = classes_neutral;
	const check = await checkAppspaceDomain(domain_name.value, subdomain.value)
	if( !check.valid ) {
		domain_valid.value = "Invalid: "+check.message;
		domain_valid_classes.value = classes_bad;
	}
	else if( !check.available ) {
		domain_valid.value = "Unavailable: "+check.message;
		domain_valid_classes.value = classes_bad;
	}
	else {
		domain_valid.value = "";
		domain_valid_classes.value = ["bg-green-100", "text-green-700"];
	}
});

const dropid = ref('');
if( dropIDsStore.is_loaded ) setInitialDropID();
else watch( () => dropIDsStore.dropids, setInitialDropID );
function setInitialDropID() {
	if( dropIDsStore.dropids.size !== 0 ) {
		dropid.value = dropIDsStore.dropids.entries().next().value![1].value.compound_id;
	}
}

// TODO if tehre are no dropids send user to create one
// Maybe even at top of page. so they see it right wawy.

const ok_to_create = computed( () => {
	if( !picked_app.value || !version_pick.value ) return false;
	if( domain_valid.value !== "" ) return false;
	if( dropid.value === "" ) return false;
	return true;
});

async function create() {
	if( !ok_to_create.value ) return;
	const new_data = await appspacesStore.createAppspace({
		app_id: picked_app.value!.app_id,
		app_version: version_pick.value,
		domain_name: domain_name.value,
		subdomain: subdomain.value,
		dropid: dropid.value
	});

	router.replace({
		name: 'migrate-appspace',
		params:{appspace_id: new_data.appspace_id+''},
		query:{job_id:new_data.job_id, migrate_only: 'true'}
	});
}

function cancel() {
	router.back();
}

</script>

<template>
	<ViewWrap>
		<MessageSad v-if="dropIDsStore.is_loaded && dropIDsStore.dropids.size === 0" head="Create A DropID First">
			A DropID is necessary to create an Appspace.
			Create one <router-link to="/dropid-new" class="text-blue-700 underline">here</router-link>.
		</MessageSad>
		<form @submit.prevent="create" @keyup.esc="cancel">
			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Application:</h3>
					<div>
						<router-link :to="{name: 'new-app'}" class="btn">Get New Application</router-link>
					</div>
				</div>
				<div class="py-5 ">
					<MessageSad head="No Apps" v-if="appsStore.apps.size === 0" class="mb-4 md:mx-4 md:rounded-lg">
						You do not have any apps. Click "Get new application" above.
					</MessageSad>
					<DataDef field="Choose Application:">
						<select ref="app_pick_elem" v-model="app_pick" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
							<option :value="undefined">Pick Application</option>
							<option v-for="[_, a] in appsStore.apps" :key="'app-pick-'+a.value.app_id" :value="a.value.app_id">{{a.value.ver_data?.name}}</option>
						</select>
					</DataDef>
					<DataDef field="Choose Version:">
						<select v-model="version_pick" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
							<option value="">Pick Version</option>
							<option v-for="v in version_options" :key="'version-pick-'+v" :value="v">{{v}}</option>
						</select>
					</DataDef>
				</div>
			</div>

			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Appspace Settings:</h3>
				</div>

				<div class="my-5">
					<DataDef field="Base Domain:">
						<select ref="domain_pick_elem" v-model="domain_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
							<option value="">Pick Domain Name</option>
							<option v-for="d in domain_names.for_appspace" :key="d.domain_name" :value="d.domain_name">{{d.domain_name}}</option>
						</select>
					</DataDef>
					<DataDef field="Subdomain:">
						<input type="text" v-model="subdomain" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
						<p class="mt-2 py-1 px-3 rounded-xl flex flex-col sm:flex-row sm:justify-between items-baseline" :class="domain_valid_classes">
							<span class="font-medium">{{full_domain}}</span>
							<span>{{domain_valid || "OK"}}</span>
						</p>
					</DataDef>

				</div>

				<div class="my-5">
					<DataDef field="DropID:">
						<select v-model="dropid" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
							<option value="">Pick DropID</option>
							<option v-for="[_, d] in dropIDsStore.dropids" :key="d.value.compound_id" :value="d.value.compound_id">{{d.value.compound_id}}</option>
						</select>
					</DataDef>
				</div>
				<div class="px-4 py-5 sm:px-6 border-t border-gray-200 flex justify-between items-center">
					<input type="button" class="btn" @click="cancel" value="Cancel" />
					<input
						type="submit"
						class="btn-blue"
						:disabled="!ok_to_create"
						value="Create" />
				</div>
			</div>
		</form>
	</ViewWrap>
</template>
