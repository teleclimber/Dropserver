import ApplicationsDM from './applications-dm';
import ds_axios from '../ds-axios-helper-ts';

import { ApplicationMeta } from '../generated-types/userroutes-classes';

jest.mock('../ds-axios-helper-ts');
const mockAxios = ds_axios as jest.Mocked<typeof ds_axios>;

test('it loads applications', async () => {
	const app_dm = new ApplicationsDM;

	const getResp = {	// should use response type for this.
		data: {
			apps: getTestApps()
		}
	};

	mockAxios.get.mockResolvedValue(getResp);
	await app_dm.fetchAll();
	
	const a = app_dm.getApplication(3);

	// the app is loaded
	expect(a.app_id).toBe(3);

	// versions got sorted
	// latest version is at index 0
	expect(a.versions[0].version).toBe('0.0.5');
	expect(a.versions[2].version).toBe('0.0.3');
});


describe('it gets Next Version', () => {
	let app_dm : ApplicationsDM;
	beforeAll( async () => {
		app_dm = new ApplicationsDM;

		const getResp = {	// should use response type for this.
			data: {
				apps: getTestApps()
			}
		};
	
		mockAxios.get.mockResolvedValue(getResp);
		await app_dm.fetchAll();
	});


	const cases = [
		{ a:1, v:'0.0.1', e:'0.0.2' },
		{ a:1, v:'0.0.2', e:undefined },
		{ a:1, v:'0.0.3', e:undefined },

		// three versions:
		{ a:3, v:'0.0.1', e:'0.0.3' },
		{ a:3, v:'0.0.2', e:'0.0.3' },
		{ a:3, v:'0.0.3', e:'0.0.4' },
		{ a:3, v:'0.0.4', e:'0.0.5' },
		{ a:3, v:'0.0.5', e:undefined },
		{ a:3, v:'0.0.6', e:undefined },
	];

	cases.forEach( c => {
		test( `${c.a} ${c.v} -> ${c.e}`, () => {
			const ex = expect( app_dm.getNextVersion(c.a, c.v) );
			if( c.e === undefined ) ex.toBeUndefined();
			else ex.toHaveProperty('version', c.e);
		});
	});
});

describe('it gets Prev Version', () => {
	let app_dm : ApplicationsDM;
	beforeAll( async () => {
		app_dm = new ApplicationsDM;

		const getResp = {	// should use response type for this.
			data: {
				apps: getTestApps()
			}
		};
	
		mockAxios.get.mockResolvedValue(getResp);
		await app_dm.fetchAll();
	});

	const cases = [
		{ a:1, v:'0.0.1', e:undefined },
		{ a:1, v:'0.0.2', e:undefined },
		{ a:1, v:'0.0.3', e:'0.0.2' },
		{ a:1, v:'0.0.4', e:'0.0.2' },

		// three versions:
		{ a:3, v:'0.0.2', e:undefined },
		{ a:3, v:'0.0.3', e:undefined },
		{ a:3, v:'0.0.4', e:'0.0.3' },
		{ a:3, v:'0.0.5', e:'0.0.4' },
		{ a:3, v:'0.0.6', e:'0.0.5' },
		{ a:3, v:'0.0.7', e:'0.0.5' },
	];

	cases.forEach( c => {
		test( `${c.a} ${c.v} -> ${c.e}`, () => {
			const ex = expect( app_dm.getPrevVersion(c.a, c.v) );
			if( c.e === undefined ) ex.toBeUndefined();
			else ex.toHaveProperty('version', c.e);
		});
	});
});


function getTestApps() : ApplicationMeta[] {

	const app1 : ApplicationMeta = {
		app_id: 1,
		app_name: 'one',
		created_dt: new Date,
		versions:[
			{
				app_name: 'abc',
				version: '0.0.2',
				created_dt: new Date,
				schema: 0
			}
		]
	};

	const app2 : ApplicationMeta = {
		app_id: 2,
		app_name: 'two',
		created_dt: new Date,
		versions:[
			{
				app_name: 'abc',
				version: '0.0.2',
				created_dt: new Date,
				schema: 0
			}, {
				app_name: 'abe',
				version: '0.0.5',
				created_dt: new Date,
				schema: 0
			}
		]
	};

	const app3 : ApplicationMeta = {
		app_id: 3,
		app_name: 'three',
		created_dt: new Date,
		versions:[
			{
				app_name: 'abc',
				version: '0.0.4',
				created_dt: new Date,
				schema: 0
			}, {
				app_name: 'abd',
				version: '0.0.3',
				created_dt: new Date,
				schema: 0
			}, {
				app_name: 'abe',
				version: '0.0.5',
				created_dt: new Date,
				schema: 0
			}
		]
	};

	return [app1, app2, app3];
}