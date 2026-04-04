export namespace duplicates {
	
	export class FileMetadata {
	    path: string;
	    size: number;
	    modTime: number;
	    resolution: string;
	    durationMins: number;
	
	    static createFrom(source: any = {}) {
	        return new FileMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	        this.resolution = source["resolution"];
	        this.durationMins = source["durationMins"];
	    }
	}
	export class DuplicateGroup {
	    hash: string;
	    files: FileMetadata[];
	    selected: boolean[];
	    deletedFiles: FileMetadata[];
	
	    static createFrom(source: any = {}) {
	        return new DuplicateGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hash = source["hash"];
	        this.files = this.convertValues(source["files"], FileMetadata);
	        this.selected = source["selected"];
	        this.deletedFiles = this.convertValues(source["deletedFiles"], FileMetadata);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class AppState {
	    groups: duplicates.DuplicateGroup[];
	    totalSelectedCount: number;
	    groupsWithSelections: Record<number, number>;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groups = this.convertValues(source["groups"], duplicates.DuplicateGroup);
	        this.totalSelectedCount = source["totalSelectedCount"];
	        this.groupsWithSelections = source["groupsWithSelections"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Preview {
	    type: string;
	    data: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Preview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.data = source["data"];
	        this.error = source["error"];
	    }
	}

}

