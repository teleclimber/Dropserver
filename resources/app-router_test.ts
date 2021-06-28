import { assertEquals, assert } from "https://deno.land/std@0.97.0/testing/asserts.ts";
import {normalizeMethod} from './app-router.ts';


Deno.test({
	name: "normalize methods",
	fn: async () => {
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
		})
	}
});


// need more tests...