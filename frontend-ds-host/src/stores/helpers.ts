import type {User} from './types';

export function userFromRaw(raw:any) :User {
	return {
		user_id: Number(raw.user_id),
		email: raw.email+"",
		has_password: !!raw.has_password,
		tsnet_identifier: raw.tsnet_identifier+'',
		tsnet_extra_name: raw.tsnet_extra_name+'',
		is_admin: !!raw.is_admin
	};
}