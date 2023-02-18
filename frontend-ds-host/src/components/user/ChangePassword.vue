<script setup lang="ts">
import { ref, Ref, watch, onMounted, computed } from 'vue';
import { useAuthUserStore} from '@/stores/auth_user';

const emit = defineEmits<{
  (e: 'close'): void
}>()

const authUserStore = useAuthUserStore();
authUserStore.fetch();

const old_pw_input :Ref<HTMLInputElement|null> = ref(null);
const new_pw_input :Ref<HTMLInputElement|null> = ref(null);
const old_pw = ref("");
const new_pw = ref("");

onMounted( () => {
	if( old_pw_input.value === null ) return;
	old_pw_input.value.focus();
});

const invalid = computed( () => {
	let e = new_pw.value;
	if( e.length < 10 ) return 'Password should be at least 10 characters';
	e = old_pw.value;
	if( e.length < 1 ) return 'Please enter your old password too';
	return ''; 	
});

const save_rejected = ref('');
watch( new_pw, () => save_rejected.value = '' );

const saving = ref(false);
async function saveClicked() {
	if( invalid.value || saving.value ) return;
	saving.value = true;
	save_rejected.value = await authUserStore.changePassword(old_pw.value, new_pw.value);
	saving.value = false;
	if( save_rejected.value === '' ) emit('close');
}
</script>

<template>
<div class="rounded border border-yellow-200 p-3 bg-yellow-100">
	<form @submit.prevent="saveClicked" @keyup.esc="$emit('close')">
		<div class="grid grid-cols-labin gap-2 items-center">
			<label class="text-right whitespace-nowrap">Old Password:</label>
			<input 
				type="password"
				name="old_pw"
				ref="old_pw_input"
				v-model="old_pw"
				class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md" />
			<label class="text-right whitespace-nowrap">New Password:</label>
			<input
				type="password"
				name="new_pw"
				ref="new_pw_input"
				v-model="new_pw"
				class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md" />
		</div>
		<div class="bg-yellow-50 rounded px-2 mt-2">
			<p v-if="invalid" class="text-yellow-800 font-medium">{{ invalid }}</p>
			<p v-else-if="save_rejected" class="text-yellow-800 font-medium">{{ save_rejected }}</p>
			<p v-else>&nbsp;</p>
		</div>
		<div class="flex justify-between pt-2">
			<input type="button" class="btn" @click="$emit('close')" value="Cancel" />
			<input
				type="submit"
				class="btn-blue"
				:disabled="!!invalid || saving || !!save_rejected"
				value="Save" />
		</div>
	</form>
</div>
</template>