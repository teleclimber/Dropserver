<style scoped>
	.dark {
		color: rgb(202, 202, 202);
	}
	.dropserver-logo .drop-o {
		text-shadow:
			0px -0.3em 0px hsla( 0, 0%, 50%, 0.8 ) ,
			0px -0.6em 0px hsla( 0, 0%, 50%, 0.5 ) ,
			0px -1.0em 0px hsla( 0, 0%, 50%, 0.2 ) ;
	}
	.dropserver-logo .s {
		color: rgba(253,29,29,1);
	}
	@supports( -webkit-background-clip: text ) {
		.dropserver-logo .s {
			background: linear-gradient(-90deg, rgb(57, 78, 172) 0%, rgb(235, 54, 54) 50%, rgba(252,176,69,1) 100%);	/*rgba(131,58,180,1)*/
			-webkit-background-clip: text;
			background-clip: text;
			color: transparent;
		}
	}
</style>

<template>
	<h1 class="dropserver-logo font-bold" :class="{dark:!!dark}">
		dr<span class="drop-o">o</span>p<span class="s">server</span>
	</h1>
</template>

<script lang="ts">
import { defineComponent } from 'vue';

export default defineComponent({
	name: 'DropServerLogo',
	props: ['dark']
});
</script>