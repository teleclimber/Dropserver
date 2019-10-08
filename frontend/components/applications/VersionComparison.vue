<style scoped>
	table {
		table-layout: fixed ;
		width: 100%;
		border: 1px solid rgb(66, 66, 66);
	}
	thead {
		background: rgba(0,0,0,0.2);
	}
	.selected {
		background: rgba(255, 255, 0, 0.205);
	}
	td {
		padding: 0.2em 1em;
	}
	td:first-child {
		text-align: right;
		padding-right: 1em;
	}
</style>

<template>
	<table>
		<thead>
			<tr>
				<th></th>
				<th>Previous</th>
				<th class="selected">Selected</th>
				<th>Next</th>
			</tr>
		</thead>
		<tbody>
			<tr>
				<td>Name:</td>
				<td>{{cmp.previous ? cmp.previous.app_name : 'n/a'}}</td>
				<td class="selected">{{cmp.upload.app_name}}</td>
				<td>{{cmp.next ? cmp.next.app_name : 'n/a'}}</td>
			</tr>
			<tr>
				<td>Version:</td>
				<td>{{cmp.previous ? cmp.previous.version : 'n/a'}}</td>
				<td class="selected">{{cmp.upload.version}}</td>
				<td>{{cmp.next ? cmp.next.version : 'n/a'}}</td>
			</tr>
			<tr>
				<td>Schema:</td>
				<td>{{cmp.previous ? cmp.previous.schema : 'n/a'}}</td>
				<td class="selected">{{cmp.upload.schema}}</td>
				<td>{{cmp.next ? cmp.next.schema : 'n/a'}}</td>
			</tr>
		</tbody>
	</table>


</template>


<script lang="ts">
import { Vue, Component, Prop, Inject, Ref } from "vue-property-decorator";
import { Observer } from "mobx-vue";

import { VersionComparison as Cmp } from '../../vms/user-page/applications-vm';


@Observer
@Component
export default class VersionComparison extends Vue{
	@Prop({required:true}) readonly cmp!: Cmp;
}
</script>
