//@ts-ignore
globalThis.DROPSERVER.appRoutes.setCallback();

console.log(await Deno.readTextFile('data.txt'));

console.log('hw');