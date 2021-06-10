import { assertEquals } from "https://deno.land/std@0.97.0/testing/asserts.ts";
import { stub, Stub } from "https://raw.githubusercontent.com/udibo/mock/v0.8.0/stub.ts";
import Twine from "./twine/twine.ts";
import DsServices from "./ds-services.ts";
//import Routes from "./appspace-routes-db.ts";

Deno.test({
	name: "create route",
	//ignore: true,
	fn: async () => {
		const t = new Twine("", false);
		//@ts-ignore
		DsServices.twine = t;
		const stubbed_sendBlock: Stub<Twine> = stub(t, "sendBlock");
		stubbed_sendBlock.returns = [{ok:true}];

		const routes_module = await import("./appspace-routes-db.ts");	// import after stubbing
		const Routes = routes_module.default;
		await Routes.createRoute(["get", "post"], "/abc/def", {allow:"owner"}, {file:"file.ts", function:"handleRoute", type:"function"});
		
		const calls = stubbed_sendBlock.calls;
		assertEquals(calls.length, 1);

		const payload = <Uint8Array>calls[0].args[2];
		const data = JSON.parse(new TextDecoder().decode(payload));
		assertEquals(data["route-path"], "/abc/def");

		stubbed_sendBlock.restore();
	}
});