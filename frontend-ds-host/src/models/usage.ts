import {get} from '../controllers/userapi';

// Generic usage model, can be used for appspace usage, user usage, and any other usage.
// Same data form, different source.

export type SandboxSums = {
	tied_up_ms: number,
    cpu_usec: number,
    memory_byte_ms: number
}

export async function fetchAppspaceSummary(appspace_id: number) :Promise<SandboxSums> {
	return <SandboxSums>await get('/appspace/'+appspace_id+'/usage');
}