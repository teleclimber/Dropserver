<style scoped>
.modal-container {
	position: fixed;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	background-color: rgba(122, 122, 122, 0.651);
	display: flex;
	align-items: center;
	justify-content: center;
}
.contents {
	background-color: #eee;
	box-sizing: border-box;
	width: calc( 100vw - 20px );
	padding: 10px;
}
@media (min-width: 620px) {
	.contents {
		width: 600px;
	}
}
</style>

<template>
	<div class="modal-container" @click="close">
		<div class="contents">
			<slot></slot>
		</div>
	</div>

</template>

<script lang="ts">
import { Vue, Component, Prop, Inject } from "vue-property-decorator";

@Component
export default class DsModal extends Vue {
	close(): void {
		this.$emit( 'close' );
	}
}
</script>