// sandbox side, untrusted.

const http = require( 'http' );
const path = require( 'path' );


///////////////////////////////////////////
// reverse channel client
/// const ip = process.argv[process.argv.length -1];	// actually unix socket

// so how does this work now?
// Probably need a separate function to send messages back to host.
// And Probably a lib for the appspaceAPI
const sock_path = process.argv[process.argv.length -2];
const app_path = process.argv[process.argv.length -1];

//////////////////////////////////////////////
// HTTP Server
const server = http.createServer();
server.on( 'request', (request,response) => {
	request.socket.setKeepAlive(true);

	const { method, url, headers } = request;

	if( headers['ds-warm-up'] ) {
		response.end('ok\n');
		console.log( 'RUNNER: connection warm up' );
	}
	else {
		const script = headers['app-space-script'];
		const fn = headers['app-space-fn'];
		const user_id = headers['ds-user-id'];

		console.log( 'RUNNER:', script, fn, user_id );

		const script_module = require( path.join(app_path, script) );
		script_module[fn]( request, response );

		// let body = [];
		// request.on('error', (err) => {
		// 	console.error(err);
		// }).on('data', (chunk) => {
		// 	//console.log( 'CONTAINER: chunk' );
		// 	body.push(chunk);
		// }).on('end', () => {
		// 	body = Buffer.concat(body).toString();
		// 	// At this point, we have the headers, method, url and body, and can now
		// 	// do whatever we need to in order to respond to this request.

		// 	response.setHeader('X-Powered-By', 'bacon');
		// 	response.write( 'hello from... ' );
		// 	setTimeout( () => {
		// 		response.end( 'container!\n' );
		// 	}, 500 );
			
		// });
	}

	// console.log( 'CONTAINER: request received.' );
});
const _all_sockets = [];
server.on( 'connection', socket => {
	_all_sockets.push[socket]
	socket.on('close', () => {
		const i = _all_sockets.indexOf(socket);
		_all_sockets.splice(i, 1);
		console.log('RUNNER: socket closed, num left:', _all_sockets.length);
	});
	console.log( 'RUNNER: connection' );
});
server.on( 'clientError', (err, socket) => {
	console.log( 'RUNNER clientError', err );
});

server.listen( 0, () => {	// Here port will have to be sent via cl args.. or we can let OS assign and send it back via rev channel.
	console.log( 'PORT:'+server.address().port);

	revPost('/hi', { port: server.address().port } );
} );

process.on( 'SIGTERM', () => {
	console.log( 'RUNNER: caught sigterm, closing things down' );
	server.close( () => {
		console.log( 'RUNNER: server closed')
	});
});

function revPost( statusPath, data ) {
	jsonStr = JSON.stringify(data)

	req = http.request({
		socketPath: sock_path,
		path: '/sandbox'+statusPath,
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


// the http handler inside the sandbox.
// this is presumably some sort of server.

// The goal is to call the app_space's handler
// ..with a req obj that is appropriately augmented

// This means knowing:
// - app-space
// - route handler fn
// - metadata to augment req with
// - the request itself

// I suspect it basically functions as such:
// DropServer-side:
// - outer proxy (nginx, HAProxy) recognizes a app_souce route and forwards to JS proxy
//   ..this may not be necessary? Can we do it all from nginx or HAP?
// - on dropserver-side js proxy receives the connection
// - it determines app_space
// - determines route handler
// - it checks against auth
// - forward the request to the sandbox HTTP endpoint

// Sandbox side:
// - server receives request
// - augment the req and res obj to express-like level. Or just use Express?
// - get route handler (script/fn) from header
// - call the handler with req and res