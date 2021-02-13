<style scoped>
.animated-loader {
	display: inline-block;
	position: relative;
	width: 80px;
	height: 80px;
}
.animated-loader div {
	display: inline-block;
	position: absolute;
	left: 8px;
	width: 16px;
	background: #e4e4e4;
	animation: animated-loader 1.2s cubic-bezier(0, 0.5, 0.5, 1) infinite;
}
.animated-loader div:nth-child(1) {
	left: 8px;
	animation-delay: -0.24s;
}
.animated-loader div:nth-child(2) {
	left: 32px;
	animation-delay: -0.12s;
}
.animated-loader div:nth-child(3) {
	left: 56px;
	animation-delay: 0;
}
@keyframes animated-loader {
	0% {
		top: 8px;
		height: 64px;
	}
	50%, 100% {
		top: 24px;
		height: 32px;
	}
}
</style>
<template>
	<div class="animated-loader"><div></div><div></div><div></div></div>
</template>