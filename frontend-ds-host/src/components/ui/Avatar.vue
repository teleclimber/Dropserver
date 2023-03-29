<script lang="ts" setup>
import { ref , Ref, watch, watchEffect, computed } from "vue";

import { decodeImage } from '../../utils/decodeimage';

// - show existing avatar, with button for load image
// - select image
// - crop
// - back to show avatar, with button to change (and re-crop?)

const props = defineProps<{
	current: string
}>();

const emit = defineEmits<{
  (e: 'changed', d: Blob|undefined): void
}>();

const canvas_dim = Math.min(400, screen.width-40);
const canvas_w = canvas_dim;
const canvas_h = canvas_dim;
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

	const corner = Math.round(canvas_dim/2 - 50);
	cropped_image.value = crop_ctx.getImageData(corner, corner, 100, 100);

	canvas_elem.value.height = 100;
	canvas_elem.value.width = 100;

	crop_ctx?.putImageData(cropped_image.value, 0, 0);
	
	canvas_elem.value.toBlob( (b:Blob|null) => {
		console.log("camera data bloc", b);
		if( b ) {
			emit('changed', b);
		}
		step.value = "ready"
	}, "image/jpeg", 0.7)
}

function del(){
	cropped_image.value = undefined;

	emit('changed', undefined);

	if( canvas_elem.value === null ) throw new Error("expected input and cavas elems");

	const ctx = canvas_elem.value.getContext("2d");
	if( ctx === null ) throw new Error("expected crop context");

	ctx.fillStyle = '#eee';
	ctx.fillRect(0,0,100,100);
}

const main_flex = computed( () => {
	if( step.value === 'ready' ) return ['flex-row'];
	else if( step.value === 'pick' ) return ['flex-col', 'md:flex-row'];
	else return ['flex-col'];
});
</script>

<template>
	<div class="flex items-start" :class="main_flex">
		<span>&nbsp;</span> 	<!-- needed to make baseline the top of the component, so that DataDef label aligns nicely -->
		<canvas v-show="step!=='pick'" ref="canvas_elem" width="100" height="100" @pointerdown="cropMoveStart" class="block"></canvas>
		<div v-if="step==='pick'" class="h-28 p-2 border-4 border-gray-100 rounded-lg">
			<input type="file" ref="input_elem" @change="fileChosen" />
		</div>

		<div v-if="step === 'ready'" class="flex self-start">
			<button @click.stop.prevent="stepToPick" class="mx-4 btn flex items-center">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5">
					<path fill-rule="evenodd" d="M13.2 2.24a.75.75 0 00.04 1.06l2.1 1.95H6.75a.75.75 0 000 1.5h8.59l-2.1 1.95a.75.75 0 101.02 1.1l3.5-3.25a.75.75 0 000-1.1l-3.5-3.25a.75.75 0 00-1.06.04zm-6.4 8a.75.75 0 00-1.06-.04l-3.5 3.25a.75.75 0 000 1.1l3.5 3.25a.75.75 0 101.02-1.1l-2.1-1.95h8.59a.75.75 0 000-1.5H4.66l2.1-1.95a.75.75 0 00.04-1.06z" clip-rule="evenodd" />
				</svg>
				Replace
			</button>
			<button @click.stop.prevent="del" class="btn flex items-center">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5">
					<path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
				</svg>
				Remove
			</button>
		</div>
		<div v-else-if="step==='pick'">
			<button @click.stop.prevent="cancelPick" class="md:mx-2 btn flex items-center">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5">
					<path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z" />
				</svg>
				cancel
			</button>
		</div>
		<div v-if="step==='crop'" class="mt-2 flex" :style="'width:'+canvas_dim+'px'">
			<input class="flex-grow" ref="zoom_elem" type="range" step="any" :min="zoom_min" :max="zoom_max" v-model="zoom">
			<button @click.stop.prevent="saveCrop" class="ml-2 btn btn-blue">Done</button>
		</div>
	</div>
</template>
