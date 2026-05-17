export namespace main {
	
	export class ImportOptions {
	    importSubscriptions: boolean;
	    importPlaylists: boolean;
	    importLikes: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ImportOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.importSubscriptions = source["importSubscriptions"];
	        this.importPlaylists = source["importPlaylists"];
	        this.importLikes = source["importLikes"];
	    }
	}

}

export namespace youtube {
	
	export class ExportInfo {
	    subscriptionCount: number;
	    playlistCount: number;
	    videoCount: number;
	    likedVideoCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subscriptionCount = source["subscriptionCount"];
	        this.playlistCount = source["playlistCount"];
	        this.videoCount = source["videoCount"];
	        this.likedVideoCount = source["likedVideoCount"];
	    }
	}

}

