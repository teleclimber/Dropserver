import DsServices from "./ds-services.ts";
import DsRouteServer from "./ds-route-server.ts";

async function run() {
	DsServices.initTwine();
	DsRouteServer.startServer();
}

run();
