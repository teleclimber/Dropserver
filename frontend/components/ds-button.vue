<style scoped>
button {
	align-items: center;
    background: var(--main-color);
    border-radius: .125rem;
	border: none;
	color: white;
	cursor: pointer;
    display: inline-flex;
	font-weight: 500;
	height: 2rem;
    justify-content: center;
    letter-spacing: 0.02em;
    line-height: 1;
    outline: none;
    overflow: hidden;
    padding: 0;
    padding-left: 1rem;
    padding-right: 1rem;
    position: relative;
    text-transform: uppercase;
}
button.cancel {
	background: transparent;
	color: rgb(179, 73, 73);
	border: 1px solid rgb(192, 131, 131);
}
button.close {
	background: transparent;
	color: #666;
	border: 1px solid #888;
}
button.disabled {
	background-color: #bbb;
	color: #ddd;
	cursor: not-allowed;
}
button:hover {

}
</style>

<template>
	<button
		:disabled="disabled"
		@click="clicked"
		:class="classes"
	>
		<slot></slot>
	</button>
</template>

<script lang="ts">
import { Vue, Component, Prop, Inject } from "vue-property-decorator";

@Component
export default class DsButton extends Vue {
	@Prop(String) readonly type!: String;
	@Prop(String) readonly disabled!: String;	// todo: both these could be bttter types

	get classes() {
		return {
			cancel: this.type === 'cancel',
			close: this.type === 'close',
			disabled: this.disabled
		}
	}

	clicked() {
		if( !this.disabled ) this.$emit( 'click' );
	}
}
</script>