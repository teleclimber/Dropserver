// dummy up migration for properly working up migration

module.exports = function() {
	return new Promise( (resolve, reject) => {
		setTimeout( () => {
			resolve();
		}, 50 );
	});
}