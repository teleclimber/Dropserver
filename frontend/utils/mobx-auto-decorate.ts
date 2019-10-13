import { observable, decorate } from "mobx";

export default function autoDecorate( cl:any ) {
	const dec : any = {};
	Object.getOwnPropertyNames(new cl).forEach( p => {
		dec[p] = observable;
	});
	decorate( cl, dec );
}