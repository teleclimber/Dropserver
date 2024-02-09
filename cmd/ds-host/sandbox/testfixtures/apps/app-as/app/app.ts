//@ts-ignore
window.DROPSERVER.appRoutes.setCallback();

console.log(await Deno.readTextFile('data.txt'));

console.log('hw');