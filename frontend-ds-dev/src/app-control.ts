export default class DsDevAppControl {
	cur_tab: string = 'app';

	setTab(tab_name:string) {
		this.cur_tab = tab_name;
	}
	get tab():string {
		return this.cur_tab;
	}
}