import LiveDataDM, {MigrationStatus} from './live-data-dm';

// // MigrationJobResp describes a pending or ongoing appspace migration job
// type MigrationJobResp struct {
// 	JobID      domain.JobID         `json:"job_id"`
// 	OwnerID    domain.UserID        `json:"owner_id"`
// 	AppspaceID domain.AppspaceID    `json:"appspace_id"`
// 	ToVersion  domain.Version       `json:"to_version"`
// 	Created    time.Time            `json:"created"`
// 	Started    nulltypes.NullTime   `json:"started"`
// 	Finished   nulltypes.NullTime   `json:"finished"`
// 	Priority   bool                 `json:"priority"`
// 	Error      nulltypes.NullString `json:"error"`
// }

// // MigrationStatusResp reflects the current status of the migrationJob referenced
// type MigrationStatusResp struct {
// 	JobID        domain.JobID         `json:"job_id"`
// 	MigrationJob *MigrationJobResp    `json:"migration_job,omitempty"`
// 	Status       string               `json:"status"`
// 	Started      nulltypes.NullTime   `json:"started"`
// 	Finished     nulltypes.NullTime   `json:"finished"`
// 	Error        nulltypes.NullString `json:"error"`
// 	CurSchema    int                  `json:"cur_schema"`
// }

const initial_update = {
	job_id: "1",
	migration_job: {
		job_id: "1",
		owner_id: 7,
		appspace_id: 11,
		to_version: "0.0.2",
		created: new Date,
		started: new Date,
		finished: null,
		priority: false,
		error: null
	},
	status: "started",
	started: new Date,
	finished: null,
	error: null,
	cur_schema: 0
}

const running_update = {
	job_id: "1",
	status: "running",
	started: new Date,
	finished: null,
	error: null,
	cur_schema: 0
}

describe('it hydrates jobs', () => {
	const live_data_dm = new LiveDataDM

	test('it hydrates full job', () => {
		live_data_dm.hydrateJob(initial_update);
		expect( Object.keys(live_data_dm.jobs).length ).toBe(1);
		const lj = live_data_dm.jobs["1"];
		expect(lj._is_dummy).toBe(false);
		expect(lj.appspace_id).toBe(11);
		expect(lj.to_version).toBe("0.0.2");
	});

	test('it updates job on update', () => {
		live_data_dm.hydrateJob(running_update);
		expect( Object.keys(live_data_dm.jobs).length ).toBe(1);
		const lj = live_data_dm.jobs["1"];
		expect(lj._is_dummy).toBe(false);
		expect(lj.status).toBe(MigrationStatus.running);
	});
});