import ds_axios from '../../ds-axios-helper.js'

import user_vm from './user-vm.js';

function validateData( data ) {
	let v = { valid: true };
	if( !data.old_pw ) v.valid = false;
	if( !data.new_pw ) v.valid = false;
	if( !data.repeat_pw ) v.valid = false;
	
	if( data.new_pw && data.new_pw.length < 8 ) {
		v.valid = false;
		v.new_pw = 'too short';
	}
	if( data.repeat_pw && data.repeat_pw !== data.new_pw ) {
		v.valid = false;
		v.repeat_pw = 'does not match';
	}
	change_pw_vm.validations = v;
}
function inputChanged( data ) {
	validateData( data );
}
async function doSave( data ) {
	validateData( data );
	if( change_pw_vm.validations.valid ) {
		// go ahead and ship it.
		// return data may indicate old pw is wrong
		change_pw_vm.saving = true;

		const resp = await ds_axios.patch( '/api/user/password', {
			old: data.old_pw,
			new: data.new_pw
		}, {
			validateStatus: status => status == 200 || status == 401
		} );

		if( resp ) {
			change_pw_vm.saving = false;
			if( resp.status == 200 ) {
				user_vm.closeChangePassword();
			}
			else {
				change_pw_vm.validations = { 
					valid: false,
					old_pw: 'incorrect password'
				};
			}
		}
	}
}

const change_pw_vm = {
	validations: {},
	saving: false,

	inputChanged,
	doSave
};

export default change_pw_vm;