import ApplicationDM from "./application-dm";


test('it sorts versions', () => {
	const app = getTestAppThreeVersions();
	const sorted = app.sorted_versions;
	expect(sorted.length).toBe(3);
	expect(sorted[0].version).toBe('0.0.5');
	expect(sorted[2].version).toBe('0.0.3');
});


describe('it gets Next Version, with one version', () => {
	const app = getTestAppOneVersion();
	const cases = [
		{ v:'0.0.1', e:'0.0.2' },
		{ v:'0.0.2', e:undefined },
		{ v:'0.0.3', e:undefined },
	];

	cases.forEach( c => {
		test( `${c.v} -> ${c.e}`, () => {
			const ex = expect( app.getNextVersion(c.v) );
			if( c.e === undefined ) ex.toBeUndefined();
			else ex.toHaveProperty('version', c.e);
		});
	});
});

describe('it gets Next Version, with three versions', () => {
	const app = getTestAppThreeVersions();

	const cases = [
		{ v:'0.0.1', e:'0.0.3' },
		{ v:'0.0.2', e:'0.0.3' },
		{ v:'0.0.3', e:'0.0.4' },
		{ v:'0.0.4', e:'0.0.5' },
		{ v:'0.0.5', e:undefined },
		{ v:'0.0.6', e:undefined },
	];

	cases.forEach( c => {
		test( `${c.v} -> ${c.e}`, () => {
			const ex = expect( app.getNextVersion(c.v) );
			if( c.e === undefined ) ex.toBeUndefined();
			else ex.toHaveProperty('version', c.e);
		});
	});
});


describe('it gets Prev Version, single version', () => {
	const app = getTestAppOneVersion();

	const cases = [
		{ v:'0.0.1', e:undefined },
		{ v:'0.0.2', e:undefined },
		{ v:'0.0.3', e:'0.0.2' },
		{ v:'0.0.4', e:'0.0.2' },
	];

	cases.forEach( c => {
		test( `${c.v} -> ${c.e}`, () => {
			const ex = expect( app.getPrevVersion(c.v) );
			if( c.e === undefined ) ex.toBeUndefined();
			else ex.toHaveProperty('version', c.e);
		});
	});
});

describe('it gets Prev Version, three versions', () => {
	const app = getTestAppThreeVersions();

	const cases = [
		{ v:'0.0.2', e:undefined },
		{ v:'0.0.3', e:undefined },
		{ v:'0.0.4', e:'0.0.3' },
		{ v:'0.0.5', e:'0.0.4' },
		{ v:'0.0.6', e:'0.0.5' },
		{ v:'0.0.7', e:'0.0.5' },
	];

	cases.forEach( c => {
		test( `${c.v} -> ${c.e}`, () => {
			const ex = expect( app.getPrevVersion(c.v) );
			if( c.e === undefined ) ex.toBeUndefined();
			else ex.toHaveProperty('version', c.e);
		});
	});
});

function getTestAppOneVersion() {
	return new ApplicationDM({
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
	});
}

function getTestAppThreeVersions() {
	return new ApplicationDM({
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
	});
}