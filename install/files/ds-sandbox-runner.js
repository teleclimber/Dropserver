// container side, untrusted.

const http = require( 'http' );
const net = require( 'net' );
const path = require( 'path' );
const sock_path = '/home/cdeveloper/run_files/reverse.sock'

const user_id = 1000;
process.setuid(user_id);

//Q: do I still have the ability to setuid back to 0?

///////////////////////////////////////////
// reverse channel client
const ip = process.argv[process.argv.length -1];

const rev_stream = net.connect(45454, ip+"%eth0", () => {
	console.log( 'RUNNER rev_stream connected');
});
rev_stream.on( 'data', data => {
	const cmd = data.toString();
	//...
});
rev_stream.on( 'error', error => {
	console.log( 'RUNNER: rev_stream error', error );
})
rev_stream.on( 'end', () => {
	console.log( 'RUNNER: rev_stream got end event' );
})

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

		const script_module = require( path.join('/home/cdeveloper/run_files/app/', script) );
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

server.listen( 3030, () => {
	rev_stream.write( 'hi' );
	//console.log( 'uncomment this line');
} );

process.on( 'SIGTERM', () => {
	console.log( 'RUNNER: caught sigterm, closing things down' );
	server.close( () => {
		console.log( 'RUNNER: server closed')
	});

	rev_stream.end();
});


// the http handler inside the container.
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
// - select a container and prep it for app_space
// - forward the request to the container's HTTP endpoint
// - simultaneously send via IPC some metadata on the request?
//   ^^ or not? wouldn't it be easier to stash that data in the request headers?

// Container side:
// - server receives request
// - augment the req and res obj to express-like level. Or just use Express?
// - get route handler (script/fn) from header
// - call the handler with req and res