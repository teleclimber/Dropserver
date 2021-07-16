
// export type ImageData = {
// 	width: number,
// 	height: number,
// 	data: Uint8Array	//I think?
// }

export async function decodeImage(f:File) :Promise<ImageBitmap> {
	return new Promise( (resolve, reject) => {
		const fr = new FileReader;
		fr.onload = async () => {
			try {
				const ab = <ArrayBuffer>fr.result;
				const b = new Blob([ab]);
				const bm = await createImageBitmap(b);	// this won't work on iOS. https://bugs.webkit.org/show_bug.cgi?id=182424
				resolve(bm);
			}
			catch(e) {
				reject(e);
			}
		}
		fr.readAsArrayBuffer(f);
	});
}