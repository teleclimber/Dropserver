
import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

class AdminSettingsVM {	// UI / VM !!
	constructor( dm, close_cb ) {
		this.dm = dm;	//instance settings dm
		this.close_cb = close_cb;

	}
	get is() {
		return 'AdminSettings';
	}
	get orig_data() {
		return this.dm.data;	// should we not deep-copy??
	}

	inputChanged( input_data ) {
		console.log( 'input event', input_data );
	}

	doSave( input_data ) {
		console.log( 'savin' );

		this.dm.commitData( input_data );

		setTimeout( () => {
			if( this.close_cb ) this.close_cb();
		}, 100 );
	}
	close() {
		if( this.close_cb ) this.close_cb();
	}
}
decorate( AdminSettingsVM, {
	disable_save_btn: observable,

	
});

export default AdminSettingsVM;