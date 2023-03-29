<script lang="ts" setup>
import { computed, } from 'vue';

const props = defineProps<{
	name: string,
	val: number,
	unit: string
}>();

const val_h = computed( () => humanValues(props.val, props.unit) );

function humanValues(val:number, unit:string) :{val:string, unit:string} {
	if( unit === 'bytes' ) {
		if( val >= 1024 ) {
			val = val/1024;
			unit = 'Kb';
			if( val >= 1024 ) {
				val = val/1024;
				unit = 'Mb';
			}
		}
	}
	else if( unit === 'byte-sec' ) {
		if( val >= 1024 ) {
			val = val/1024;
			unit = 'Mb-sec';
			if( val >= 3600 ) {
				val = val/3600;
				unit = 'Mb-hour';
			}
		}
	}
	else if( unit == 'usec' ) {
		if( val > 1000 ) {
			val = val /1000;
			unit = 'ms';
			if( val > 1000 ) {
				val = val /1000;
				unit = 'sec';
			}	
		}
	}
	// ms?

	return {val: new Intl.NumberFormat(undefined, {maximumSignificantDigits: 3}).format(val), unit};
}
</script>

<template>
	<div class="p-2">
		<h4 class="uppercase text-sm">{{name}}:</h4>
		<div>
			<span class="font-bold">{{val_h.val}}</span> {{val_h.unit}}
		</div>
	</div>
</template>
