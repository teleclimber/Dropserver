<script setup lang="ts">
import { computed } from 'vue';
import { AppManifest, Warning } from '@/stores/types';

import DataDef from '@/components/ui/DataDef.vue';
import AppLicense from '@/components/app/AppLicense.vue';
import SmallMessage from '@/components/ui/SmallMessage.vue';
import MigrationsGrid from '@/components/appspace/MigrationsGrid.vue';

const props = defineProps<{
	manifest: AppManifest,
	warnings: Warning[] | undefined,

}>();

const release_date = computed( () => {
	if( !props.manifest.release_date ) return;
	return new Date(props.manifest.release_date).toLocaleDateString(undefined, {
		dateStyle:'medium'
	});
});

const field_warns = computed( () => {
	const ret :Record<string,Warning[]> = {};
	props.warnings?.forEach(w => {
		const f = w.field;
		if( !ret[f] ) ret[f] = [];
		ret[f].push(w);
	});
	return ret;
});
const bad_values = computed( () => {
	const ret :Record<string,string> = {};
	props.warnings?.forEach(w => {
		if( w.bad_value ) ret[w.field] = w.bad_value;
	});
	return ret;
});

const small_msg_classes = ['inline-block', 'mt-1'];
const link_classes = ['text-blue-500', 'hover:underline', 'hover:text-blue-600' ];

</script>

<template>
	<div>
		<DataDef field="App name:">
			<p class="font-medium text-lg">{{ manifest.name }}</p>
			<SmallMessage v-for="w in field_warns['name']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
		</DataDef>

		<DataDef field="Version:">
			<p class="">
				<span class="font-medium text-lg bg-gray-200 text-gray-700 px-1 rounded-md">{{ manifest.version }}</span>
				<span v-if="release_date"> released {{ release_date || '' }}</span>
				<SmallMessage v-for="w in field_warns['version-sequence']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
			</p>
		</DataDef>

		<DataDef field="Data schema:">
			<p class="font-medium text-lg">{{ manifest.schema }}</p>
			<SmallMessage v-for="w in field_warns['schema']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
		</DataDef>

		<DataDef field="Migrations:">
			<MigrationsGrid :migrations="manifest.migrations"></MigrationsGrid>
			<SmallMessage v-for="w in field_warns['migrations']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
		</DataDef>

		<DataDef field="License:">
			<p><AppLicense :license="manifest.license" ></AppLicense></p>
			<SmallMessage v-for="w in field_warns['license']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
			<SmallMessage v-for="w in field_warns['license-file']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
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
			<SmallMessage v-for="w in field_warns['authors']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
			<p v-if="!manifest.authors || manifest.authors.length === 0" class="text-gray-500 italic">No authors listed</p>
		</DataDef>

		<DataDef field="Website:">
			<a v-if="manifest.website" :href="manifest.website" :class="link_classes">{{ manifest.website }}</a>
			<span v-else-if="bad_values['website']">{{ bad_values['website'] }}</span>
			<p v-else class="text-gray-500 italic">No website listed</p>
			<SmallMessage v-for="w in field_warns['website']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
		</DataDef>

		<DataDef field="Code repository:">
			<a v-if="manifest.code" :href="manifest.code" :class="link_classes">{{ manifest.code }}</a>
			<span v-else-if="bad_values['code']">{{ bad_values['code'] }}</span>
			<p v-else class="text-gray-500 italic">No code repository listed</p>
			<SmallMessage v-for="w in field_warns['code']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
		</DataDef>

		<DataDef field="Funding:">
			<a v-if="manifest.funding" :href="manifest.funding" :class="link_classes">{{ manifest.funding }}</a>
			<span v-else-if="bad_values['funding']">{{ bad_values['funding'] }}</span>
			<p v-else class="text-gray-500 italic">No funding website listed</p>
			<SmallMessage v-for="w in field_warns['funding']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
		</DataDef>

		<DataDef v-if="field_warns['icon']" field="Icon:">
			<SmallMessage v-for="w in field_warns['icon']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
		</DataDef>
		<DataDef v-if="field_warns['accent-color']" field="Accent color:">
			<SmallMessage v-for="w in field_warns['accent-color']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
		</DataDef>
		<DataDef v-if="field_warns['short-description']" field="Short description:">
			<p>“{{ manifest.short_description }}”</p>
			<SmallMessage v-for="w in field_warns['short-description']" mood="warn" :class="small_msg_classes">{{ w.message }}</SmallMessage>
		</DataDef>
	</div>
</template>