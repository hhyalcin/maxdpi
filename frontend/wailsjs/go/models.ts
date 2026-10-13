export namespace main {
	
	export class Status {
	    connected: boolean;
	    profile: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.profile = source["profile"];
	    }
	}

}

