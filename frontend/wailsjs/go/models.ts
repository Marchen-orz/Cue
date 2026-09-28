export namespace app {
	
	export class InterviewStatus {
	    running: boolean;
	    listening: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InterviewStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.listening = source["listening"];
	    }
	}

}

export namespace config {
	
	export class Config {
	    apiKey?: string;
	    baseURL?: string;
	    model?: string;
	    prompt?: string;
	    domainId?: string;
	    opacity?: number;
	    noCompression?: boolean;
	    compressionQuality?: number;
	    sharpening?: number;
	    grayscale?: boolean;
	    keepContext?: boolean;
	    interruptThinking?: boolean;
	    screenshotMode?: string;
	    resumePath?: string;
	    resumeContent?: string;
	    shortcuts?: Record<string, shortcut.KeyBinding>;
	    assistantModel?: string;
	    windowWidth?: number;
	    windowHeight?: number;
	    screenIndex?: number;
	    regionX?: number;
	    regionY?: number;
	    regionW?: number;
	    regionH?: number;
	    toolbarCollapsed?: boolean;
	    toolbarX?: number;
	    toolbarY?: number;
	    hoverTriggerMs?: number;
	    shieldEnabled: boolean;
	    reviewEnabled: boolean;
	    reviewDir?: string;
	    theme?: string;
	    interviewAudioSource?: string;
	    interviewMode?: string;
	    interviewModel?: string;
	    asrApiKey?: string;
	    asrModel?: string;
	    interviewSilenceMs?: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	        this.baseURL = source["baseURL"];
	        this.model = source["model"];
	        this.prompt = source["prompt"];
	        this.domainId = source["domainId"];
	        this.opacity = source["opacity"];
	        this.noCompression = source["noCompression"];
	        this.compressionQuality = source["compressionQuality"];
	        this.sharpening = source["sharpening"];
	        this.grayscale = source["grayscale"];
	        this.keepContext = source["keepContext"];
	        this.interruptThinking = source["interruptThinking"];
	        this.screenshotMode = source["screenshotMode"];
	        this.resumePath = source["resumePath"];
	        this.resumeContent = source["resumeContent"];
	        this.shortcuts = this.convertValues(source["shortcuts"], shortcut.KeyBinding, true);
	        this.assistantModel = source["assistantModel"];
	        this.windowWidth = source["windowWidth"];
	        this.windowHeight = source["windowHeight"];
	        this.screenIndex = source["screenIndex"];
	        this.regionX = source["regionX"];
	        this.regionY = source["regionY"];
	        this.regionW = source["regionW"];
	        this.regionH = source["regionH"];
	        this.toolbarCollapsed = source["toolbarCollapsed"];
	        this.toolbarX = source["toolbarX"];
	        this.toolbarY = source["toolbarY"];
	        this.hoverTriggerMs = source["hoverTriggerMs"];
	        this.shieldEnabled = source["shieldEnabled"];
	        this.reviewEnabled = source["reviewEnabled"];
	        this.reviewDir = source["reviewDir"];
	        this.theme = source["theme"];
	        this.interviewAudioSource = source["interviewAudioSource"];
	        this.interviewMode = source["interviewMode"];
	        this.interviewModel = source["interviewModel"];
	        this.asrApiKey = source["asrApiKey"];
	        this.asrModel = source["asrModel"];
	        this.interviewSilenceMs = source["interviewSilenceMs"];
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

export namespace domain {
	
	export class DomainItem {
	    id: string;
	    label: string;
	    icon: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new DomainItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.icon = source["icon"];
	        this.description = source["description"];
	    }
	}
	export class Category {
	    id: string;
	    label: string;
	    items: DomainItem[];
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.items = this.convertValues(source["items"], DomainItem);
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

export namespace screen {
	
	export class DisplayInfo {
	    index: number;
	    x: number;
	    y: number;
	    w: number;
	    h: number;
	    primary: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DisplayInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.x = source["x"];
	        this.y = source["y"];
	        this.w = source["w"];
	        this.h = source["h"];
	        this.primary = source["primary"];
	    }
	}
	export class PreviewResult {
	    imgBytes: number[];
	    base64: string;
	    size: string;
	
	    static createFrom(source: any = {}) {
	        return new PreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imgBytes = source["imgBytes"];
	        this.base64 = source["base64"];
	        this.size = source["size"];
	    }
	}
	export class Region {
	    X: number;
	    Y: number;
	    W: number;
	    H: number;
	
	    static createFrom(source: any = {}) {
	        return new Region(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.X = source["X"];
	        this.Y = source["Y"];
	        this.W = source["W"];
	        this.H = source["H"];
	    }
	}

}

export namespace shortcut {
	
	export class KeyBinding {
	    vkCode: string;
	    keyName: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.vkCode = source["vkCode"];
	        this.keyName = source["keyName"];
	    }
	}

}

