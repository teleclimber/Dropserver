import {migrationFunction, GetMigrationsCallback} from 'https://deno.land/x/dropserver_lib_support@v0.1.0/mod.ts';

// MigrationMeta is what is sent to the host to identify and make use of migration
// Basically should include everything except the actual function
export type MigrationMeta = {
	direction: "up"|"down",
	schema: number,
}

export default class Migrations {
	#migrationsLoaded = false;
	up: Map<number, migrationFunction> = new Map;
	down: Map<number, migrationFunction> = new Map;

	cb :GetMigrationsCallback;
	setCallback(cb:GetMigrationsCallback) :void {
		if( this.cb !== undefined ) throw new Error("migration callback already set.");
		this.cb = cb;
	}
	async loadMigrations() {
		if( this.#migrationsLoaded ) return;
		this.#migrationsLoaded = true;
		if( this.cb === undefined ) return;
		const migrations = await Promise.resolve(this.cb());	// don't like this really need to partition out instances sent to app code.
		migrations.forEach( m => {
			const map = m.direction === "up" ? this.up : this.down;
			const schema = Math.round(m.schema);
			if( map.has(schema) ) throw new Error("migration function already exists: "+schema);
			map.set(schema, m.func);
		});
	}
	getMigrations() :MigrationMeta[] {
		if( !this.#migrationsLoaded ) throw new Error("migrations not loaded!");
		const ret: MigrationMeta[] = [];
		this.up.forEach( (_, schema) => {
			ret.push({
				direction: "up",
				schema
			});
		});
		this.down.forEach( (_, schema) => {
			ret.push({
				direction: "down",
				schema
			});
		});
		return ret;
	}

	// getFunc returns the migration function for the desired schema and direction
	getFunc(up:boolean, schema:number) :migrationFunction {
		if( !this.#migrationsLoaded ) throw new Error("migrations not loaded!");
		const m = up ? this.up : this.down;
		schema = Math.round(schema);
		const func = m.get(schema);
		if( func === undefined ) throw new Error(`trying to get migration function that does not exist: up:${up} ${schema}`);
		return func;
	}
}