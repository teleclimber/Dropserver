import libSupportIface from 'https://deno.land/x/dropserver_lib_support@v0.1.0/mod.ts';

import Metadata from './metadata.ts';
import DsServices from './services/services.ts';
import Migrations from './migrations.ts';
import MigrationService from './services/migrateservice.ts';
import AppRoutes from './approutes.ts';
import DsAppService from './services/appservice.ts';
import DsRouteServer from './services/routeserver.ts';
import LibSupport from './libsupport.ts';

const metadata = new Metadata;
const services = new DsServices;

const w = <{["DROPSERVER"]?:libSupportIface}>window;
const libSupport = new LibSupport(metadata, services);
w["DROPSERVER"] = libSupport;

const migrations = new Migrations;
libSupport.setMigrations(migrations);

const migrationService = new MigrationService(migrations);
services.setMigrationService(migrationService);

const appRoutes = new AppRoutes(services);
libSupport.setAppRoutes(appRoutes);

const appService = new DsAppService(appRoutes);
services.setAppService(appService);

const server = new DsRouteServer(services, libSupport.appRoutes);
services.setServer(server);

services.initTwine(metadata.rev_sock_path);	// this results in host getting "Ready", which is weird because we haven't even loaded the app code yet?
server.startServer(metadata.sock_path);	// only start it if we need to, but deal with later.
