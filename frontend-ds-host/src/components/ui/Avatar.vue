<template>
	<div>
		<canvas v-show="step!=='pick'" ref="canvas_elem" width="100" height="100" @pointerdown="cropMoveStart"></canvas>
		<div v-if="step==='pick'" class="h-28 p-2 border-4 border-gray-100 rounded-lg">
			<input type="file" ref="input_elem" @change="fileChosen" />
		</div>

		<div v-if="step === 'ready'">
			<button @click.stop.prevent="stepToPick" class="btn btn-blue">Replace</button>
			<button @click.stop.prevent="del" class="btn btn-blue">Remove</button>
		</div>
		<div v-else-if="step==='pick'">
			<button @click.stop.prevent="cancelPick" class="btn btn-blue">Cancel</button>
		</div>
		<div v-if="step==='crop'">
			<input ref="zoom_elem" type="range" step="any" :min="zoom_min" :max="zoom_max" v-model="zoom">
			<button @click.stop.prevent="saveCrop" class="btn btn-blue">Done</button>
		</div>
		
	</div>
</template>


<script lang="ts">
import { defineComponent, PropType, ref , Ref, watch, watchEffect} from "vue";

import { decodeImage } from '../../utils/decodeimage';

// - show existing avatar, with button for load image
// - select image
// - crop
// - back to show avatar, with button to change (and re-crop?)

export default defineComponent({
	name: "Avatar",
	components: {
		
	},
	emits:['changed'],
	props: {
		current: {	// current avatar as filename.
			type: String,
			required: true
		}
	},
	setup(props, context) {

		const canvas_w = 400;
		const canvas_h = 400;
		const crop_w = 100;
		const crop_h = 100;

		const step = ref("ready");	// ready, pick, crop

		function stepToPick() {
			step.value = "pick";
		}
		function cancelPick() {
			step.value = "ready";
		}

		let original_image :ImageBitmap|undefined = undefined;
		const cropped_image :Ref<ImageData|undefined> = ref(undefined);

		const input_elem :Ref<HTMLInputElement|null> = ref(null);
		const canvas_elem :Ref<HTMLCanvasElement|null> = ref(null);
		const zoom_elem :Ref<HTMLInputElement|null> = ref(null);

		// multiplier for cropping: px_crop = px_orig * zoom
		const zoom_min = ref(1);
		const zoom_max = ref(100);
		const zoom = ref(1);

		let y = 0;
		let x = 0;

		watchEffect( () => {
			if( canvas_elem.value === null ) return;
			const crop_ctx = canvas_elem.value.getContext("2d");
			if( crop_ctx === null ) return;

			if( props.current === "" ) {
				if( step.value !== 'ready' || cropped_image.value !== undefined ) return;
				crop_ctx.fillStyle = '#eee';
				crop_ctx.fillRect(0, 0, crop_w, crop_h);
				return;
			}
			const img = new Image();
			img.onload = function() {
				if( step.value !== 'ready' || cropped_image.value !== undefined ) return;
				crop_ctx.drawImage(img, 0, 0);
			};
			img.src = props.current;
		});

		async function fileChosen() {
			console.log("file chosen");
			if( input_elem.value === null || canvas_elem.value === null ) throw new Error("expected input and cavas elems");
			if( !input_elem.value.files ) throw new Error("expected input elem to have files");
			if( !input_elem.value.files[0] ) {
				console.log("no file selected.");
				return;
			}

			canvas_elem.value.height = canvas_h;
			canvas_elem.value.width = canvas_w;

			const crop_ctx = canvas_elem.value.getContext("2d");
			if( crop_ctx === null ) throw new Error("expected canvas ctx");

			const file = input_elem.value.files[0];

			original_image = await decodeImage(file);
			if( !original_image ) return;

			const small_dim = Math.min(original_image.width, original_image.height);
			const big_dim = Math.max(original_image.width, original_image.height);

			zoom_min.value = crop_w / small_dim;
			zoom_max.value = 1;
			zoom.value = canvas_w / big_dim;

			y = original_image.height / 2;
			x = original_image.width / 2;
			
			updateCropCanvas(true);

			step.value = 'crop';
		}

		watch( zoom, () => {
			updateCropCanvas(true);
		});

		function cropMoveStart(event:PointerEvent) {
			if( step.value !== 'crop' ) return;
			if( !original_image ) return;
			if( canvas_elem.value === null ) throw new Error("expected input and cavas elems");

			canvas_elem.value.setPointerCapture(event.pointerId);

			canvas_elem.value.addEventListener("pointermove", cropMove);
			canvas_elem.value.addEventListener("pointerup", cropMoveEnd);
		}
		function cropMove(event:PointerEvent) {
			if( !original_image ) return;

			const z = zoom.value;
			x = x - event.movementX/z;
			y = y - event.movementY/z;
			
			x = Math.max(x, crop_w/2/z);
			x = Math.min(x, original_image.width - crop_w/2/z);

			y = Math.max(y, crop_h/2/z);
			y = Math.min(y, original_image.height - crop_h/2/z);

			updateCropCanvas(true);
		}
		function cropMoveEnd(event:PointerEvent) {
			if( canvas_elem.value === null ) return;
			canvas_elem.value.removeEventListener("pointermove", cropMove);
			canvas_elem.value.removeEventListener("pointerup", cropMoveEnd);
			canvas_elem.value.releasePointerCapture(event.pointerId);
		}

		function updateCropCanvas(show_ui:boolean) {
			if( !original_image ) return;
			if( canvas_elem.value === null ) throw new Error("expected input and cavas elems");
			const crop_ctx = canvas_elem.value.getContext("2d");
			if( crop_ctx === null ) throw new Error("expected canvas ctx");

			crop_ctx.fillStyle = '#aaa';
			crop_ctx.fillRect(0, 0, canvas_w, canvas_h);

			// crop edges wrt canvas
			const crop_left = (canvas_w-crop_w)/2;
			const crop_right = (canvas_w+crop_w)/2;
			const crop_top = (canvas_h-crop_h)/2;
			const crop_bot = (canvas_h+crop_h)/2;

			const z = zoom.value;

			const w = original_image.width * z;
			const h = original_image.height * z;

			let left = canvas_w/2 - x*z;
			let top = canvas_h/2 - y*z;

			left = Math.min(left, crop_left);
			left = Math.max(left, crop_right - w);
			
			top = Math.min(top, crop_top);
			top = Math.max(top, crop_bot - h);

			crop_ctx.drawImage(original_image, left , top, w, h);

			if( !show_ui ) return;

			crop_ctx.strokeStyle = "#ddd";
			crop_ctx.strokeRect(crop_left, crop_top, crop_w, crop_h);
			crop_ctx.strokeStyle = "#444";
			crop_ctx.strokeRect(crop_left+1, crop_top+1, crop_w, crop_h);
		}

		function saveCrop() {
			if( canvas_elem.value === null ) throw new Error("expected input and cavas elems");

			const crop_ctx = canvas_elem.value.getContext("2d");
			if( crop_ctx === null ) throw new Error("expected crop context");

			updateCropCanvas(false);

			cropped_image.value = crop_ctx.getImageData(150, 150, 100, 100);

			canvas_elem.value.height = 100;
			canvas_elem.value.width = 100;

			crop_ctx?.putImageData(cropped_image.value, 0, 0);
			
			canvas_elem.value.toBlob( (b:Blob|null) => {
				console.log("camera data bloc", b);
				if( b ) {
					context.emit('changed', b);
				}
				step.value = "ready"
			}, "image/jpeg", 0.7)
		}

		function del(){
			cropped_image.value = undefined;

			context.emit('changed', undefined);

			if( canvas_elem.value === null ) throw new Error("expected input and cavas elems");

			const ctx = canvas_elem.value.getContext("2d");
			if( ctx === null ) throw new Error("expected crop context");

			ctx.fillStyle = '#eee';
			ctx.fillRect(0,0,100,100);
		}

		return {
			props,
			step, stepToPick, cancelPick,
			input_elem, canvas_elem,
			zoom_elem,
			cropMoveStart, 
			zoom, zoom_min, zoom_max,
			cropped_image, 
			fileChosen, saveCrop,
			del
		}
	}
});
</script>