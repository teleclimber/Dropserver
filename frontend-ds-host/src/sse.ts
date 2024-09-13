let eventSrc: EventSource|undefined;

export function on(name:string, fn:(raw:any)=>void) {
	startSSE();
	eventSrc!.addEventListener(name, (evt) => {
		const data = JSON.parse(evt.data);
		fn(data);
	});
}

function startSSE() {
	if( eventSrc ) return;
	
	eventSrc = new EventSource("/events/");

	eventSrc.onmessage = (event) => {
		console.log('sse event: ', event);
	};
	eventSrc.onerror = (err) => {
		console.error("EventSource failed:", err);
	};
}
