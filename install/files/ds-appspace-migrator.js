// sandbox for migrating appspaces, untrusted.

const http = require( 'http' );//no?
const path = require( 'path' );

///////////////////////////////////////////
// reverse channel client
/// const ip = process.argv[process.argv.length -1];	// actually unix socket

// so how does this work now?
// Probably need a separate function to send messages back to host.
// And Probably a lib for the appspaceAPI
const sock_path = process.argv[process.argv.length -5];
const app_path = process.argv[process.argv.length -4];
const appspace_path = process.argv[process.argv.length -3];

const from_schema = Number(process.argv[process.argv.length -2]);
const to_schema = Number(process.argv[process.argv.length -1]);



(async function() {
	console.log("aoubt to start migration!");
	try {
		await migrate();
	}
	catch(e) {
		console.error(e);
		process.exit(22);
	}
	
	console.log( "Apparently migrated without errors");
})()


//process.exit(22);

// Here we just run the migration code that we had

// We should have an app path and a data dir path
// iterate over app_path/migrations from  - to
// 


// function migrateAppVersion( username, app_space_id, to_ver ) {
// 	return new Promise( (resolve, reject) => {


// 			const from_ml = app_space.migration_level;
// 			Application.getMigrationLevel( username, app_space.app_name, to_ver )
// 			.then( to_ml => {
// 				console.log( `from / to: ${from_ml} ${to_ml}` );
// 				if( from_ml !== to_ml ) {

// 					takeSnapshot( username, app_space_id )
// 					.then( sk => {
// 						snapshot_key = sk;
// 						const code_ver = semver.gt(app_space.app_version, to_ver) ? app_space.app_version : to_ver;
// 						return migrateAppSpace( app_space, code_ver, from_ml, to_ml );
// 					}).then( () => {
// 						setAppVersion( username, app_space_id, to_ver );
// 						setMigrationLevel( username, app_space_id, to_ml );
// 						resolve();
// 					}).catch( err => {
// 						// do rollback!!
// 						reject( err );
// 					});
// 				}
// 				else {
// 					setAppVersion( username, app_space_id, to_ver );
// 					resolve();
// 				}
// 			}).catch( err => {
// 				reject( err );
// 			});
// 		}
// 	});
// }

async function migrate() {
	if( from_schema === to_schema ) return;

	console.log("migrating from", from_schema, to_schema);

	if( from_schema < to_schema ) {
		for( let i=from_schema+1; i<=to_schema; ++i ) {
			console.log( 'running up migration for '+i );
			try {
				await runStep( i, true );
			}
			catch( err ) {
				console.error( err ) ;
				process.exit(22);
			}
		}
	}
	else {
		// contrary to up, going down means running down.js at current level, and stopping short of desired level
		for( let i=from_schema; i>to_schema; --i ) {
			console.log( 'running down migration for '+i );
			try {
				await runStep( i, false );
			}
			catch( err ) {
				console.error( err ) ;
				process.exit(22);
			}
		}
	}
}

async function runStep(num, up) {
	return new Promise( (resolve, reject) => {
		const script_path = path.join( app_path, "migrations", num+'', up?"up.js":"down.js" );

		let script_module;
		try {
			script_module = require( script_path );	// straight up node module for now
			// ^^ could also use import, and I think we will have to,
			// .. but not clear how we deal with async migration code (which it ofetn will be)
			// Also all this naming convention is a little offputting.
		}
		catch(e) {
			// failed to require the migration module
			console.error( e );
			reject( err );
		}
		
		let ret;
		try {
			ret = script_module();
		}
		catch( e ) {
			console.error( e );
			// should still close sandbox
			reject( err );
		}

		if( ret && ret.then ) {	// typeof ret is Promise
			ret.then( () => {
				resolve();
			});
			ret.catch( err => {
				console.error( err );
				// close sandbox, yes?
				reject( err );
			});
		}
		else {
			resolve();
		}
	});

}



process.on( 'SIGTERM', () => {
	console.log( 'RUNNER: caught sigterm, closing things down (not implemented)' );
});

function revPost( statusPath, data ) {
	jsonStr = JSON.stringify(data)

	req = http.request({
		socketPath: sock_path,
		path: '/status'+statusPath,
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			'Content-Length': jsonStr.length
		}
	}, res => {
		console.log(`${statusPath} statusCode: ${res.statusCode}`)
	});

	req.on("error", (error) => {
		console.error("error send post request", error)
	});

	req.write(jsonStr);

	req.end();
}
