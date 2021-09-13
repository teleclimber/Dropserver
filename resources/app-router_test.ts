import { assertEquals, assert } from "https://deno.land/std@0.106.0/testing/asserts.ts";
import AppRouter, {AuthAllow, Context, normalizeMethod} from './app-router.ts';


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
		})
	}
});

Deno.test({
	name: "add then get route",
	fn: () => {
		const router = new AppRouter;
		const routeID = router.add("get", "/abc/:def", {allow:AuthAllow.public}, (_:Context) => {})
		const route = router.getRouteWithMatch(routeID);
		if( route === undefined ) throw new Error("route should have been returned");
		if( route.match === undefined ) throw new Error("route should have match function");
		const match = route.match("/abc/foo");
		assert(match);
		const params = <{def:string}>match.params;
		assertEquals(params.def, "foo");
	}
});


// need more tests...