<script lang="ts" setup>
import { Ref, ref, reactive, watch, computed, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';

import {setTitle, unsetTitle} from '../controllers/nav';

import {DomainNames} from '../models/domainnames';
import {useDropIDsStore} from '../stores/dropids';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';

const router = useRouter();

const dropIDsStore = useDropIDsStore();

// domain... How do we get the list of domains that are usable for this?
const domain_names = reactive( new DomainNames );
domain_names.fetchForOwner();

const domain_input : Ref<HTMLInputElement|undefined> = ref();
watch( domain_names, () => {
	if( domain_names.loaded && domain_names.for_dropid.length !== 0 && domain_name.value === '') {
		domain_name.value = domain_names.for_dropid[0].domain_name;
	}
});

const domain_name = ref("")
const handle = ref("");
const display_name = ref("");

const neutral_classes = ['bg-gray-50', 'text-gray-600'];
const ok_classes = ['bg-green-50', 'text-green-700'];
const bad_classes = ['bg-red-50', 'text-red-700'];

onMounted( () => {
	setTitle("Create DropID");
	if( domain_input.value ) domain_input.value.focus();
});
onUnmounted( () => {
	unsetTitle();
});

const checked = reactive({
	compound: '',
	ok: false
});
watch( [domain_name, handle], async () => {
	if( !validity.value.valid ) return;
	const d = domain_name.value.trim();
	const h = handle.value.trim();
	const compound = d + '/' + h;
	const ok = await dropIDsStore.checkHandle(h, d);
	if( compound !== getCurCompound() ) return;
	checked.compound = compound;
	checked.ok = ok;
});

const alpha_num_re = new RegExp(/^[a-z0-9]+$/i);

const validity = computed( () => {
	const d = domain_name.value.trim();
	const h = handle.value.trim();
	if( d === '' ) {
		return {
			valid: false,
			message: "Please select a domain",
			classes: bad_classes
		};
	}
	if( h.length > 30 ) {
		return {
			valid: false,
			message: "Handle is too long",
			classes: bad_classes
		};
	}
	if( h.length > 0 && !alpha_num_re.test(h) ) {
		return {
			valid: false,
			message: "Handle must be alpha-numeric",
			classes: bad_classes
		};
	}

	const compound = d + '/' + h;
	if( checked.compound === compound ) {
		if( checked.ok ) {
			return {
				valid: true,
				message: 'OK',
				classes: ok_classes
			}
		}
		else {
			return {
				valid: true,
				message: "Handle is unavailable",
				classes: bad_classes
			};
		}
	}
	else {
		return {
			valid: true,
			message: "Checking availability...",
			classes: neutral_classes
		};
	}
});

function getCurCompound() {
	return domain_name.value.trim() + '/' + handle.value.trim();
}

const save_ok = computed( () => {
	if( !validity.value.valid ) return false;
	if( checked.compound !== getCurCompound() || !checked.ok ) return false;
	return true;
})

async function save() {
	if( !save_ok.value ) return;
	if( display_name.value.length > 29 ) {
		alert("Display name too long");
		return;
	}

	const d = domain_name.value.trim();
	const h = handle.value.trim();

	await dropIDsStore.createDropID(h, d, display_name.value);
	
	router.back();
}

function cancel() {
	router.back();
}
</script>

<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">DropID</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">Your DropID is used to join an Appspace.</p>
			</div>
			<form @submit.prevent="save" @keyup.esc="cancel">
				<div class="py-5">
					<DataDef field="Domain Name:">
						<select v-model="domain_name" ref="domain_input" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
							<option value="">Pick Domain Name</option>
							<option v-for="d in domain_names.for_dropid" :key="d.domain_name" :value="d.domain_name">{{d.domain_name}}</option>
						</select>
					</DataDef>
					<DataDef field="Handle:">
						<input type="text" v-model="handle" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
						<p class="mt-2 px-3 rounded-lg" :class="validity.classes">{{ validity.message }}</p>
					</DataDef>
				</div>

				<div class="py-5 border-t border-gray-200">
					<DataDef field="Display Name:">
						<input type="text" v-model="display_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					</DataDef>
				</div>	
				<div class="flex justify-between px-4 py-5 sm:px-6 border-t border-gray-200">
					<input type="button" class="btn py-2" @click="cancel" value="Cancel" />
					<input
						type="submit"
						class="btn-blue"
						:disabled="!save_ok"
						value="Create DropID" />
				</div>
			</form>
		</div>
	</ViewWrap>
</template>
