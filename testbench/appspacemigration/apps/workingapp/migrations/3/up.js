// dummy up migration for failing up migration

module.exports = function() {
	return new Promise( (resolve, reject) => {
		setTimeout( () => {
			//resolve();
			reject("argh I don gum this up");
		}, 50 );
	});
}