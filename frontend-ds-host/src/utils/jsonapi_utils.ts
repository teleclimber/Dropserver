// type Identifier = {
// 	type: string,
// 	id: string
// }
type RawDoc = {
	data?: RawResource | RawResource[],	// or null?
	errors?: any,	// doc must include one of data, errors, oro meta.
	meta?: any,		// Will add those latter ones later.
	links?: Links,
	included?: RawResource[]
}
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
export class Document {
	constructor( private raw:RawDoc) {

	}
	getResource() :Resource {
		if( this.raw.data === undefined ) throw new Error("no data in document");
		if( Array.isArray(this.raw.data) ) throw new Error("data is a collection");
		return new Resource(this.raw.data);
	}
	getCollection() :Resource[] {
		if( this.raw.data === undefined ) throw new Error("no data in document");
		if( !Array.isArray(this.raw.data) ) throw new Error("data is not a collection");
		return this.raw.data.map(raw => {
			return new Resource(raw);
		});
	}
	getIncluded(type:string, id:string) :Resource {
		if( this.raw.included === undefined ) throw new Error("nothing included in document");
		const raw_inc = this.raw.included.find((inc) => inc.type === type && inc.id === id );
		if( raw_inc === undefined ) throw new Error("expected to find included resource.");
		return new Resource(raw_inc);
	}
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
	// might need a hasAttr, because we could be looking at just a type and id?
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

	relOne(rel_name:string) : Resource {
		if( this.raw.relationships === undefined || this.raw.relationships[rel_name] === undefined || this.raw.relationships[rel_name].data === undefined ) {
			throw new Error('Unable to reutrn relationship '+rel_name);
		}
		return new Resource(this.raw.relationships[rel_name].data);		
	}
	relMany(rel_name:string) :Resource[] {
		if( this.raw.relationships === undefined || this.raw.relationships[rel_name] === undefined || this.raw.relationships[rel_name].data === undefined ) {
			throw new Error('Unable to reutrn relationship '+rel_name);
		}
		const rel = this.raw.relationships[rel_name].data;
		if( !Array.isArray(rel) ) throw new Error("to many relationship not an array in doc "+rel_name);
		
		return rel.map(raw => {
			return new Resource(raw);
		});
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