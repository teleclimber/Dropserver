<script setup lang="ts">
import {onMounted, ref, Ref, watch} from 'vue';

const props = defineProps<{
	initial_value?: string
}>();

export type ValidatedURL = {
	url: string,
	valid: boolean,
	message: string
}

const emit = defineEmits<{
	(e: 'changed', d: ValidatedURL): void
}>();

const input_url = ref("");

const input_elem :Ref<HTMLInputElement|undefined> = ref();
onMounted( () => {
	input_elem.value?.focus();
});

watch( () => props.initial_value, () => {
	if( props.initial_value ) input_url.value = props.initial_value;
}, {immediate:true});

watch( input_url, () => {
	const u = input_url.value.trim().toLowerCase();
	const normalized = normalize(u);
	const valid = validate(normalized);
	let msg = "";
	if( u === "" ) msg = "Please enter a link";
	else if( valid !== "" ) msg = valid;
	else if( u != normalized ) msg = "OK: "+normalized;
	else msg = "OK";
	emit('changed', {
		url:normalized,
		valid: valid === '',
		message: msg
	});
}, {immediate:true});

function normalize(u :string) {
	u = u.trim().toLowerCase();
	if( u === "" ) return "";
	if( !u.startsWith("http://") && !u.startsWith("https://") ) u = "https://"+u;
	return u;
};

function validate(url :string) {
	if( url === "" ) return "";
	let u :URL|undefined;
	try {
		u = new URL(url);
	}
	catch {
		return "Please check the link, it appears to be invalid.";
	}
	if( u.protocol !== "https:" ) {
		return "Please use a secure https:// URL.";
	}
	return "";
}

</script>

<template>
	<input type="text" v-model="input_url" ref="input_elem" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
</template>