import { assertEquals } from "https://deno.land/std@0.106.0/testing/asserts.ts";
import {normalizeMethod} from './approutes.ts';


Deno.test({
	name: "normalize methods",
	fn: () => {
		const cases :{m:string, norm:string, throws:boolean}[] = [
			{m:"get", norm:"get", throws:false},
			{m:"gEt", norm:"get", throws:false},
			{m:"GET", norm:"get", throws:false},
			{m:"blurf", norm:"", throws:true},
		];

		cases.forEach( c=> {
			let result = "";
			try {
				result = normalizeMethod(c.m);
			}catch(e) {
				if( !c.throws ) {
					throw e;
				}
			}
			assertEquals( result, c.norm );
		});
	}
});