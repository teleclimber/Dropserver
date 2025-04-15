<script setup lang="ts">
import { ref, Ref, nextTick } from 'vue';

import { useInstanceSettingsStore } from '@/stores/admin/instance_settings';

import ViewWrap from '../../components/ViewWrap.vue';
import BigLoader from '@/components/ui/BigLoader.vue';
import ManageUserTSNet from '@/components/admin/ManageUserTSNet.vue';

const settings_store = useInstanceSettingsStore();
settings_store.loadData();

const open_radio_elem :Ref<HTMLInputElement|undefined> = ref();
const close_radio_elem :Ref<HTMLInputElement|undefined> = ref();
const reg_open_input = ref("");
const show_change = ref(false);
const saving = ref(false);

function showChange() {
	reg_open_input.value = settings_store.registration_open ? "open": "closed";
	show_change.value = true;
	nextTick( () => {
		if( reg_open_input.value === "open" ) open_radio_elem.value?.focus();
		else close_radio_elem.value?.focus();
		console.log(open_radio_elem.value);
	});
}

async function formSubmitted() {
	saving.value = true;
	await settings_store.setRegistrationOpen(reg_open_input.value === "open" ? true : false);
	saving.value = false;
	show_change.value = false;
}
</script>

<template>
	<ViewWrap>
		<ManageUserTSNet></ManageUserTSNet>
		
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">New User Registration:</h3>
			</div>
			<div v-if="show_change" class="p-4 sm:px-6">
				<form @submit.prevent="formSubmitted" @keyup.esc="show_change = false">
					<label 
						class="flex items-center px-4 py-2 rounded " 
						:class="{'bg-red-100': reg_open_input === 'open'}">
						<input type="radio" name="reg_open" value="open" v-model="reg_open_input" ref="open_radio_elem" />
						<p class="pl-3">
							<span class="font-bold">Open</span>:
							Anybody can register for an account on your instance.	
						</p>
					</label>
					<label 
						class="flex items-center px-4 py-2 rounded "
						:class="{'bg-green-100': reg_open_input === 'closed'}">
						<input type="radio" name="reg_open" value="closed" v-model="reg_open_input" ref="close_radio_elem" />
						<p class="pl-3">
							<span class="font-bold">Closed</span>:
							A new user must be invited to register.
						</p>
					</label>
					<div class="flex justify-between mt-4">
						<input type="button" class="btn" @click="show_change = false" value="Cancel" />
						<input
							type="submit"
							class="btn-blue"
							:disabled="saving"
							value="Save" />
					</div>
				</form>
			</div>
			<div v-else-if="!settings_store.is_loaded">
				<BigLoader></BigLoader>
			</div>
			<div v-else class="p-4 sm:px-6" :class="[settings_store.registration_open ? 'bg-red-100' : 'bg-green-100']">
				<template v-if="settings_store.registration_open" >
					<h4 class="text-lg font-medium text-red-700">
						New user registration is 
						<span class="bg-red-200 text-red-800 px-1 rounded uppercase text-sm font-bold">open</span>.
					</h4>
					<p>Anybody can create an account on this instance.</p>
				</template>
				<template v-else>
					<h4 class="text-lg font-medium text-green-700">
						New user registration is 
						<span class="bg-green-200 text-green-800 px-1 rounded uppercase text-sm font-bold">closed</span>.
					</h4>
					<p>
						A user must be invited by the admin to register. 
						Invite user <router-link to="/admin/users" class="text-blue-600 underline">here</router-link>.
					</p>
				</template>
				<div class="flex justify-end">
					<button class="btn" @click="showChange()">change</button>
				</div>
			</div>
		</div>
	</ViewWrap>
</template>
