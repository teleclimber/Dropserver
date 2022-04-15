<template>
	<div class="p-2">
		<h4 class="uppercase text-sm">{{name}}:</h4>
		<div>
			<span class="font-bold">{{val_h.val}}</span> {{val_h.unit}}
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, computed, } from 'vue';

export default defineComponent({
	props: {
		name: {
			type: String,
			required: true
		},
		val: {
			type: Number,
			required: true
		},
		unit: {
			type: String,
			required: true
		}
	},
	setup(props) {
		const val_h = computed( () => humanValues(props.val, props.unit) );
		return {
			val_h
		}
	}
});

function humanValues(val:number, unit:string) :{val:string, unit:string} {
	if( unit === 'byte-ms' ) {
		if( val >= 1024 ) {
			val = val/1024;
			unit = 'Mb-ms';
			if( val >= 1000 ) {
				val = val/1000;
				unit = 'Mb-sec';
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