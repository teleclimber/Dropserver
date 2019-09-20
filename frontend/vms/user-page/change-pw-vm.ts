import { action, computed, observable, decorate, configure, runInAction, flow } from "mobx";

import CurrentUserDM from '../../dms/current-user-dm';

type ChangePwVMDeps = {
	current_user_dm: CurrentUserDM
};
type ChangePwVMCbs = {
	closeChangePassword(): void
}

export type DataValidations = {
	valid: boolean,
	old_pw: string,
	new_pw: string,
	repeat_pw: string,
}

export default class ChangePwVM {
	@observable saving: boolean = false;

	@observable old_pw = '';
	@observable new_pw = '';
	@observable repeat_pw = '';

	constructor(private cbs: ChangePwVMCbs, private deps: ChangePwVMDeps) {	}

	@computed get validations() : DataValidations {
		const v = {
			valid: true,
			old_pw: '',
			new_pw: '',
			repeat_pw: '',
		};
		if( !this.old_pw || !this.new_pw || !this.repeat_pw ) v.valid = false;
		
		if( this.new_pw && this.new_pw.length < 8 ) {
			v.valid = false;
			v.new_pw = 'too short';
		}
		if( this.repeat_pw && this.repeat_pw !== this.new_pw ) {
			v.valid = false;
			v.repeat_pw = 'does not match';
		}

		return v;
	}

	@action
	async doSave() {
		if( this.validations.valid ) {
			this.saving = true;

			const ok = await this.deps.current_user_dm.changePassword(this.old_pw, this.new_pw);

			runInAction( () => {
				this.saving = false;
				if( ok ) {
					this.cbs.closeChangePassword();
				}
				else {
					alert('old pw incorrect');	// need to do something better here.
				}
			});
		}
	}

	cancel() {
		this.cbs.closeChangePassword();
	}
}
