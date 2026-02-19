<script lang="ts" setup>
import { getAvatarUrl } from '@/stores/appspace_users';
import type { AppspaceUser , AppspaceUserAuth, UserIDProxyIDConflicts, UserIDProxyIDMatches} from '../stores/types';
import InlineMessage from './ui/InlineMessage.vue';

const props = defineProps<{
	user: AppspaceUser,
	conflicts: UserIDProxyIDConflicts | undefined
}>();

const avatar_url = getAvatarUrl(props.user.appspace_id, props.user.avatar);

function getMultiProxyMatch(auth: AppspaceUserAuth) {
	const ret :Set<Number> = new Set;
	if( !props.conflicts?.conflict ) return ret;
	props.conflicts.user_id_matches.forEach( (m, u) => {
		if( m.auths.find( a => a.identifier === auth.identifier && a.type === auth.type) ) {
			ret.add(u);
		}
	});
	return ret;
}

</script>

<template>
	<div class="flex">
		<img v-if="avatar_url" :src="avatar_url" class="w-12 h-12 flex-shrink-0 rounded-full bg-clip-border">
		<div v-else class="w-12 h-12 flex-shrink-0 rounded-full bg-clip-border bg-gray-200">&nbsp;</div>
		<div class="flex-grow flex-shrink pl-4 overflow-hidden">
			<div class="flex flex-col sm:flex-row items-baseline">
				<span class="pr-2 font-bold text-l">{{user.display_name}}</span>
				<InlineMessage mood="warn" v-if="!conflicts">This appspace user is not associated with anybody on this instance</InlineMessage>
				<span v-else-if="!conflicts.conflict">inst us: {{ conflicts.user_id }}</span>
				<InlineMessage v-else-if="conflicts.conflict && conflicts.user_id_matches.size > 1" mood="warn" class="block">
					Multiple users of this Dropserver instance match this appspace user.
				</InlineMessage>
			</div>
			
			<ul>
				<li v-for="auth in user.auths" class="text-gray-600">
					<template v-if="auth.type === 'tsnetid'">
						<span>Tailnet ID: </span>
						<span>{{ auth.extra_name }} ({{ auth.identifier.split("@")[1] }})</span>
					</template>
					<template v-else>
						<span class="">{{ auth.type === 'dropid' ? 'DropID' : auth.type }}: </span>
						<span>{{ auth.identifier }}</span>
					</template>
					<InlineMessage mood="warn" v-if="getMultiProxyMatch(auth).size" class="inline-block">
						Matches
						<span v-for="user_id in getMultiProxyMatch(auth)">User {{ user_id }}</span> 
					</InlineMessage>
				</li>
			</ul>
		</div>
		<router-link class="btn self-start flex-shrink-0" :to="{name:'appspace-user', params:{appspace_id: user.appspace_id, proxy_id:user.proxy_id}}">Edit</router-link>
	</div>
</template>

