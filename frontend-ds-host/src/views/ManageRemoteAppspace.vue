<script lang="ts" setup>
import { ref, onMounted, computed } from 'vue';

import { useRemoteAppspacesStore } from '@/stores/remote_appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import DeleteRemoteAppspace from '../components/appspace/DeleteRemoteAppspace.vue';
import BigLoader from '@/components/ui/BigLoader.vue';

const props = defineProps<{
	domain: string
}>();

const remoteAppspacesStore = useRemoteAppspacesStore();
onMounted( () => {
	remoteAppspacesStore.loadData();
});

const enter_link = ref("/appspacelogin?appspace="+encodeURIComponent(props.domain));

const remote_appspace = computed( () => {
	if( !remoteAppspacesStore.is_loaded ) return
	const r = remoteAppspacesStore.get(props.domain);
	if( r === undefined ) return;
	return r.value;
});

const display_link = computed( () => {
	if( remote_appspace.value === undefined ) return 'loading...';
	const protocol = remote_appspace.value.no_tls ? 'http' : 'https';
	return protocol+'://'+remote_appspace.value.domain_name+remote_appspace.value.port_string;
});
	
</script>

<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 flex flex-col sm:flex-row sm:justify-between">
				<div>
					<h3 class="text-2xl leading-6 font-medium text-gray-900">
						Manage Remote Appspace
					</h3>
				</div>
			</div>
			<div v-if="remote_appspace" class="px-4 py-5 sm:px-6 border-t border-gray-200">
				<p class="">Remote Appspace provided by [owner dropid].</p>
				<p>{{remote_appspace.domain_name}}</p>
				<a :href="enter_link" class="text-blue-700 text-lg underline hover:text-blue-500">{{display_link}}</a>

				<!-- What are we going to do here?
					- change user drop id?
					- change display name and avatar (depends whether it's allowed by remote appspace? Or it'll be subject to approval)
					- Leave remote appspace (maybe "disconnect") and archive? or something...
					- Handle things that are incoming from remote, like change of owner, change of address, etc...
					- ...
				-->
			</div>
			<BigLoader v-else></BigLoader>
		</div>

		<DeleteRemoteAppspace v-if="remote_appspace" :domain="remote_appspace.domain_name"></DeleteRemoteAppspace>
	</ViewWrap>
</template>
