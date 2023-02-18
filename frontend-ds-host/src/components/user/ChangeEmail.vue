<script setup lang="ts">
import { ref, Ref, watchEffect, watch, onMounted, computed } from 'vue';
import { useAuthUserStore} from '@/stores/auth_user';

const emit = defineEmits<{
  (e: 'close'): void
}>()

const authUserStore = useAuthUserStore();
authUserStore.fetch();

const email_input :Ref<HTMLInputElement|null> = ref(null);
const email = ref("");
watchEffect( () => {
	if( authUserStore.is_loaded ) email.value = authUserStore.email;
});

onMounted( () => {
	if( email_input.value === null ) return;
	email_input.value.focus();
});

const invalid = computed( () => {
	const e = email.value;
	if( e.length < 3 || e.indexOf('@') <0 ) return 'Please enter a valid email';
	return ''; 	
});

const save_rejected = ref('');
watch( email, () => save_rejected.value = '' );

const saving = ref(false);
async function saveClicked() {
	if( invalid.value || saving.value ) return;
	saving.value = true;
	save_rejected.value = await authUserStore.changeEmail(email.value);
	saving.value = false;
	if( save_rejected.value === '' ) emit('close');
}
</script>

<template>
<div class="rounded border border-yellow-200 p-3 bg-yellow-100">
	<form @submit.prevent="saveClicked" @keyup.esc="$emit('close')">
		<input 
			type="text"
			ref="email_input"
			name="email"
			v-model="email"
			class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
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