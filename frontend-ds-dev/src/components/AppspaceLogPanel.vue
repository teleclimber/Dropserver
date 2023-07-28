<script setup lang="ts">
import { reactive } from 'vue';

import LiveLog from '../models/appspace-log-data';

import Log from './Log.vue';

const appspaceLog = reactive(new LiveLog) as LiveLog;
appspaceLog.subscribeAppspaceLog(15);	// 15 is designated hard-coded appspace id in ds-dev.

</script>
<template>
	<div class="border-t-4 border-black px-4  text-sm uppercase font-bold">
		Appspace Log:
		<span v-if="!appspaceLog.log_open" class="ml-2 px-2 rounded-sm inline-block bg-yellow-700 text-white text-sm font-bold">Log Closed</span>
	</div>
	<div class="h-32">
		<Log title="Appspace" :live_log="appspaceLog"></Log>
	</div>
</template>