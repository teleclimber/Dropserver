
type Links = {
	self?: string,
	related?: string,
	// others?
}
type Relationship = {
	links: Links,
	data: RawResource
}
type RawResource = {
	type: string,
	id: string,
	attributes?: { [attr:string] : any },
	relationships?: { [rel:string] : Relationship },
	links?: Links,
	meta?: {}
}

export class Resource {
	constructor(private raw:RawResource) {
		
	}

	idString() :string {
		return this.raw.id;
	}
	idNumber() :number {
		const num = Number(this.raw.id);
		if( typeof num !== "number" ) {
			throw new Error(`Unable to cast id as number: ${this.raw.id}`);
		}
		return num;
	}
	attr(attr_name : string) :any {
		if( this.raw.attributes !== undefined ) {
			return this.raw.attributes[attr_name];
		}
	}
	attrBool(attr_name: string) :boolean {
		const a = this.attr(attr_name);
		if( typeof a != "boolean" ) {
			throw new Error(`attribute ${attr_name} is not a boolean on resource`);
		}
		return !!a;
	}
	attrNumber(attr_name: string) :number {
		const a = this.attr(attr_name);
		if( typeof a != "number" ) {
			throw new Error(`attribute ${attr_name} is not a number on resource`);
		}
		return Number(a);
	}
	attrString(attr_name:string) :string {
		const s = this.attr(attr_name);
		if( typeof s != "string" ) {
			throw new Error(`attribute ${attr_name} is not a string on resource`);
		}
		return s+'';
	}
	attrDate(attr_name:string) :Date {
		const s = this.attr(attr_name);
		if( typeof s != "string" ) {
			throw new Error(`attribute ${attr_name} is not a date string on resource`);
		}
		return new Date(s);
	}
}

export class DocumentBuilder {
	raw : RawResource;
	constructor(type:string, id:string) {
		this.raw = {
			id,
			type
		};
	}
	getJSON():string {
		const doc = {
			data: this.raw
		};
		return JSON.stringify(doc);
	}
	setAttr(name:string, val:any) {
		if( this.raw.attributes === undefined ) this.raw.attributes = {};
		this.raw.attributes[name] = val;
	}
}