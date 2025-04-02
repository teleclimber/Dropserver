import { TSNetStatus, TSNetWarning, TSNetPeerUser, TSNetUserDevice, TSNetData } from '../types';

export function tsnetPeerUsersFromRaw(raw:any) :TSNetPeerUser[] {
	if( !Array.isArray(raw) ) return [];
	return raw.map( (r:any) => {
		let devices :TSNetUserDevice[] = [];
		if( Array.isArray(r.devices) ) devices = r.devices.map( (d:any) => {
			return {
				id: d.id+'',
				name: d.name +'',
				online: d.online,
				last_seen: typeof d.last_seen === 'undefined' ? undefined : new Date(d.last_seen),
				os: d.os+'',
				device_model: d.device_model+'',
				app: d.app+'',
			}
		});
		return {
			id: r.id+'',
			full_id: r.full_id+'',
			control_url: r.control_url+'',
			login_name: r.login_name+'',
			display_name: r.display_name+'',
			sharee: !!r.sharee,
			devices: devices
		}
	});
}

export function tsnetStatusFromRaw(raw:any) :TSNetStatus {
	const warnings : TSNetWarning[] = [];
	if( Array.isArray(raw.warnings) ) {
		raw.warnings.forEach((w:any) => {
			warnings.push({
				title: raw.title+'',
				text: raw.text+'',
				severity: raw.severuty+'',
				impacts_connectivity: !! raw.impacts_connectivity
			})
		});
	}
	return {
		control_url: strFromRaw(raw.control_url),
		url: strFromRaw(raw.url),
		ip4: strFromRaw(raw.ip4),
		ip6: strFromRaw(raw.ip6),
		listening_tls: !!raw.listening_tls,
		tailnet: strFromRaw(raw?.tailnet),
		key_expiry: raw.key_expiry ? new Date(raw.key_expiry) : undefined,
		name: strFromRaw(raw.name),
		https_available: !!raw.https_available,
		magic_dns_enabled: !!raw.magic_dns_enabled,
		tags: raw.tags|| [],
		err_message: strFromRaw(raw?.err_message),
		state: strFromRaw(raw?.state),
		usable: !!raw?.usable,
		browse_to_url: strFromRaw(raw?.browse_to_url),
		login_finished: !!raw?.login_finished,
		warnings,
		transitory: strFromRaw(raw.transitory)
	}
}


export function tsnetDataFromRaw(raw:any) :TSNetData|undefined {
	if( !raw ) return undefined;
	if( !raw.hostname ) return undefined;
	return {
		control_url: strFromRaw(raw.control_url),
		hostname: strFromRaw(raw.hostname),
		connect: !!raw.connect
	}
}

function strFromRaw(raw:any) :string {
	if( !raw ) return '';
	return raw+'';
}
