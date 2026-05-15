export namespace model {
	
	export class AppLogItem {
	    id: number;
	    siteId: number;
	    tenantId: number;
	    tenantName: string;
	    requestId: string;
	    level: string;
	    module: string;
	    message: string;
	    contextJson: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AppLogItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.tenantId = source["tenantId"];
	        this.tenantName = source["tenantName"];
	        this.requestId = source["requestId"];
	        this.level = source["level"];
	        this.module = source["module"];
	        this.message = source["message"];
	        this.contextJson = source["contextJson"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class AppLogListInput {
	    tenantId: number;
	    keyword: string;
	    level: string;
	    module: string;
	    requestId: string;
	    dateFrom: string;
	    dateTo: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new AppLogListInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tenantId = source["tenantId"];
	        this.keyword = source["keyword"];
	        this.level = source["level"];
	        this.module = source["module"];
	        this.requestId = source["requestId"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.limit = source["limit"];
	    }
	}
	export class CurrentUser {
	    id: number;
	    name: string;
	    username: string;
	    roles: string[];
	
	    static createFrom(source: any = {}) {
	        return new CurrentUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.username = source["username"];
	        this.roles = source["roles"];
	    }
	}
	export class TenantTokenState {
	    accessToken: string;
	    refreshToken: string;
	    tokenType: string;
	    expiresAt: string;
	    refreshExpiresAt: string;
	    sessionId: number;
	    tenantId: number;
	
	    static createFrom(source: any = {}) {
	        return new TenantTokenState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accessToken = source["accessToken"];
	        this.refreshToken = source["refreshToken"];
	        this.tokenType = source["tokenType"];
	        this.expiresAt = source["expiresAt"];
	        this.refreshExpiresAt = source["refreshExpiresAt"];
	        this.sessionId = source["sessionId"];
	        this.tenantId = source["tenantId"];
	    }
	}
	export class AuthBootstrap {
	    token: TenantTokenState;
	    user: CurrentUser;
	
	    static createFrom(source: any = {}) {
	        return new AuthBootstrap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = this.convertValues(source["token"], TenantTokenState);
	        this.user = this.convertValues(source["user"], CurrentUser);
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
	export class BrowserRenderedPreviewResult {
	    requestedUrl: string;
	    finalUrl: string;
	    host: string;
	    browser: string;
	    html: string;
	    renderedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new BrowserRenderedPreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestedUrl = source["requestedUrl"];
	        this.finalUrl = source["finalUrl"];
	        this.host = source["host"];
	        this.browser = source["browser"];
	        this.html = source["html"];
	        this.renderedAt = source["renderedAt"];
	    }
	}
	export class CollectWebPageInput {
	    url: string;
	    siteId?: number;
	    tenantId?: number;
	    useBrowserRender?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CollectWebPageInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.siteId = source["siteId"];
	        this.tenantId = source["tenantId"];
	        this.useBrowserRender = source["useBrowserRender"];
	    }
	}
	export class CollectedImage {
	    url: string;
	    alt?: string;
	
	    static createFrom(source: any = {}) {
	        return new CollectedImage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.alt = source["alt"];
	    }
	}
	export class CollectedWebPageResult {
	    requestedUrl: string;
	    finalUrl: string;
	    canonicalUrl?: string;
	    officialUrl?: string;
	    host: string;
	    sourceHtml?: string;
	    browserPreviewHtml?: string;
	    siteName?: string;
	    title: string;
	    description?: string;
	    excerpt?: string;
	    iconUrl?: string;
	    thumbnailUrl?: string;
	    contentHtml: string;
	    contentText: string;
	    keywords: string[];
	    seoTitle?: string;
	    seoDescription?: string;
	    seoKeywords: string[];
	    suggestedTags: string[];
	    images: CollectedImage[];
	    publishedAt?: string;
	    fetchedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CollectedWebPageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestedUrl = source["requestedUrl"];
	        this.finalUrl = source["finalUrl"];
	        this.canonicalUrl = source["canonicalUrl"];
	        this.officialUrl = source["officialUrl"];
	        this.host = source["host"];
	        this.sourceHtml = source["sourceHtml"];
	        this.browserPreviewHtml = source["browserPreviewHtml"];
	        this.siteName = source["siteName"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.excerpt = source["excerpt"];
	        this.iconUrl = source["iconUrl"];
	        this.thumbnailUrl = source["thumbnailUrl"];
	        this.contentHtml = source["contentHtml"];
	        this.contentText = source["contentText"];
	        this.keywords = source["keywords"];
	        this.seoTitle = source["seoTitle"];
	        this.seoDescription = source["seoDescription"];
	        this.seoKeywords = source["seoKeywords"];
	        this.suggestedTags = source["suggestedTags"];
	        this.images = this.convertValues(source["images"], CollectedImage);
	        this.publishedAt = source["publishedAt"];
	        this.fetchedAt = source["fetchedAt"];
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
	export class CreateSiteInput {
	    name: string;
	    baseUrl: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateSiteInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.baseUrl = source["baseUrl"];
	        this.description = source["description"];
	    }
	}
	export class CreateTenantInput {
	    siteId: number;
	    name: string;
	    baseUrl: string;
	    apiBaseUrl: string;
	    tenantName: string;
	    tenantSlug: string;
	    lastUsername: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateTenantInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.siteId = source["siteId"];
	        this.name = source["name"];
	        this.baseUrl = source["baseUrl"];
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.tenantName = source["tenantName"];
	        this.tenantSlug = source["tenantSlug"];
	        this.lastUsername = source["lastUsername"];
	    }
	}
	
	export class DeleteSiteInput {
	    id: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteSiteInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	    }
	}
	export class DeleteTenantInput {
	    id: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteTenantInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	    }
	}
	export class TenantAuthEntry {
	    tenantId: number;
	    token: TenantTokenState;
	    user: CurrentUser;
	
	    static createFrom(source: any = {}) {
	        return new TenantAuthEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tenantId = source["tenantId"];
	        this.token = this.convertValues(source["token"], TenantTokenState);
	        this.user = this.convertValues(source["user"], CurrentUser);
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
	export class TenantLimits {
	    maxUploadMb: number;
	    maxBatchCount: number;
	
	    static createFrom(source: any = {}) {
	        return new TenantLimits(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxUploadMb = source["maxUploadMb"];
	        this.maxBatchCount = source["maxBatchCount"];
	    }
	}
	export class TenantCapabilities {
	    article: boolean;
	    tool: boolean;
	    dictionary: boolean;
	    media: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TenantCapabilities(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.article = source["article"];
	        this.tool = source["tool"];
	        this.dictionary = source["dictionary"];
	        this.media = source["media"];
	    }
	}
	export class TenantItem {
	    id: number;
	    siteId: number;
	    name: string;
	    baseUrl: string;
	    apiBaseUrl: string;
	    tenantName?: string;
	    tenantSlug?: string;
	    lastUsername?: string;
	    status: string;
	    capabilities: TenantCapabilities;
	    limits: TenantLimits;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new TenantItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.name = source["name"];
	        this.baseUrl = source["baseUrl"];
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.tenantName = source["tenantName"];
	        this.tenantSlug = source["tenantSlug"];
	        this.lastUsername = source["lastUsername"];
	        this.status = source["status"];
	        this.capabilities = this.convertValues(source["capabilities"], TenantCapabilities);
	        this.limits = this.convertValues(source["limits"], TenantLimits);
	        this.createdAt = source["createdAt"];
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
	export class SiteItem {
	    id: number;
	    name: string;
	    baseUrl: string;
	    description?: string;
	    isDefault?: boolean;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SiteItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.baseUrl = source["baseUrl"];
	        this.description = source["description"];
	        this.isDefault = source["isDefault"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class DesktopBootstrap {
	    sites: SiteItem[];
	    tenants: TenantItem[];
	    token?: TenantTokenState;
	    user?: CurrentUser;
	    tenantAuths: TenantAuthEntry[];
	
	    static createFrom(source: any = {}) {
	        return new DesktopBootstrap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sites = this.convertValues(source["sites"], SiteItem);
	        this.tenants = this.convertValues(source["tenants"], TenantItem);
	        this.token = this.convertValues(source["token"], TenantTokenState);
	        this.user = this.convertValues(source["user"], CurrentUser);
	        this.tenantAuths = this.convertValues(source["tenantAuths"], TenantAuthEntry);
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
	export class DiscoverSiteTenantsInput {
	    baseUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new DiscoverSiteTenantsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.baseUrl = source["baseUrl"];
	    }
	}
	export class DownloadRemoteMediaInput {
	    url: string;
	    referer: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadRemoteMediaInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.referer = source["referer"];
	    }
	}
	export class DownloadRemoteMediaResult {
	    fileBase64: string;
	    mimeType: string;
	    fileName: string;
	    fileSize: number;
	
	    static createFrom(source: any = {}) {
	        return new DownloadRemoteMediaResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileBase64 = source["fileBase64"];
	        this.mimeType = source["mimeType"];
	        this.fileName = source["fileName"];
	        this.fileSize = source["fileSize"];
	    }
	}
	export class LocalDraftItem {
	    id: number;
	    tenantId: number;
	    contentType: string;
	    targetId: string;
	    title?: string;
	    payloadJson: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalDraftItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tenantId = source["tenantId"];
	        this.contentType = source["contentType"];
	        this.targetId = source["targetId"];
	        this.title = source["title"];
	        this.payloadJson = source["payloadJson"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class LocalDraftListInput {
	    tenantId: number;
	    contentType?: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalDraftListInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tenantId = source["tenantId"];
	        this.contentType = source["contentType"];
	    }
	}
	export class LocalDraftQueryInput {
	    tenantId: number;
	    contentType: string;
	    targetId: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalDraftQueryInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tenantId = source["tenantId"];
	        this.contentType = source["contentType"];
	        this.targetId = source["targetId"];
	    }
	}
	export class LoginTenantInput {
	    tenantId: number;
	    apiBaseUrl: string;
	    username: string;
	    password: string;
	    tenantSlug: string;
	    deviceName: string;
	    clientVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new LoginTenantInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tenantId = source["tenantId"];
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.tenantSlug = source["tenantSlug"];
	        this.deviceName = source["deviceName"];
	        this.clientVersion = source["clientVersion"];
	    }
	}
	export class LogoutTenantInput {
	    apiBaseUrl: string;
	    accessToken: string;
	
	    static createFrom(source: any = {}) {
	        return new LogoutTenantInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.accessToken = source["accessToken"];
	    }
	}
	export class MediaTaskCacheCleanupInput {
	    taskId: number;
	
	    static createFrom(source: any = {}) {
	        return new MediaTaskCacheCleanupInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	    }
	}
	export class MediaTaskItem {
	    id: number;
	    siteId: number;
	    tenantId: number;
	    tenantName: string;
	    fileName: string;
	    originalName: string;
	    mimeType: string;
	    uploadScene: string;
	    canRetry: boolean;
	    sourceUrl: string;
	    mediaCategoryId: number;
	    draftId: number;
	    status: string;
	    requestId: string;
	    remoteMediaId: number;
	    remoteUrl: string;
	    remotePath: string;
	    disk: string;
	    sizeBytes: number;
	    width: number;
	    height: number;
	    errorMessage: string;
	    responseJson: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new MediaTaskItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.tenantId = source["tenantId"];
	        this.tenantName = source["tenantName"];
	        this.fileName = source["fileName"];
	        this.originalName = source["originalName"];
	        this.mimeType = source["mimeType"];
	        this.uploadScene = source["uploadScene"];
	        this.canRetry = source["canRetry"];
	        this.sourceUrl = source["sourceUrl"];
	        this.mediaCategoryId = source["mediaCategoryId"];
	        this.draftId = source["draftId"];
	        this.status = source["status"];
	        this.requestId = source["requestId"];
	        this.remoteMediaId = source["remoteMediaId"];
	        this.remoteUrl = source["remoteUrl"];
	        this.remotePath = source["remotePath"];
	        this.disk = source["disk"];
	        this.sizeBytes = source["sizeBytes"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.errorMessage = source["errorMessage"];
	        this.responseJson = source["responseJson"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class MediaTaskListInput {
	    tenantId: number;
	    keyword: string;
	    scene: string;
	    status: string;
	    dateFrom: string;
	    dateTo: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new MediaTaskListInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tenantId = source["tenantId"];
	        this.keyword = source["keyword"];
	        this.scene = source["scene"];
	        this.status = source["status"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.limit = source["limit"];
	    }
	}
	export class MediaTaskRetryInput {
	    taskId: number;
	
	    static createFrom(source: any = {}) {
	        return new MediaTaskRetryInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	    }
	}
	export class RefreshTenantTokenInput {
	    tenantId: number;
	    apiBaseUrl: string;
	    refreshToken: string;
	    deviceName: string;
	    clientVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new RefreshTenantTokenInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tenantId = source["tenantId"];
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.refreshToken = source["refreshToken"];
	        this.deviceName = source["deviceName"];
	        this.clientVersion = source["clientVersion"];
	    }
	}
	export class RemoteDesktopAPIRequestInput {
	    apiBaseUrl: string;
	    path: string;
	    method: string;
	    headers: Record<string, string>;
	    body: string;
	    accessToken: string;
	
	    static createFrom(source: any = {}) {
	        return new RemoteDesktopAPIRequestInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.path = source["path"];
	        this.method = source["method"];
	        this.headers = source["headers"];
	        this.body = source["body"];
	        this.accessToken = source["accessToken"];
	    }
	}
	export class RemoteDesktopAPIResponse {
	    success: boolean;
	    message: string;
	    data: any;
	    errors?: Record<string, Array<string>>;
	    error_code?: string;
	    request_id?: string;
	    status_code?: number;
	
	    static createFrom(source: any = {}) {
	        return new RemoteDesktopAPIResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.data = source["data"];
	        this.errors = source["errors"];
	        this.error_code = source["error_code"];
	        this.request_id = source["request_id"];
	        this.status_code = source["status_code"];
	    }
	}
	export class SaveLocalDraftInput {
	    tenantId: number;
	    contentType: string;
	    targetId: string;
	    title: string;
	    payloadJson: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveLocalDraftInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tenantId = source["tenantId"];
	        this.contentType = source["contentType"];
	        this.targetId = source["targetId"];
	        this.title = source["title"];
	        this.payloadJson = source["payloadJson"];
	    }
	}
	export class SaveTenantAuthInput {
	    tenantId: number;
	    accessToken: string;
	    refreshToken: string;
	    tokenType: string;
	    expiresAt: string;
	    refreshExpiresAt: string;
	    sessionId: number;
	    userId: number;
	    username: string;
	    name: string;
	    roles: string[];
	
	    static createFrom(source: any = {}) {
	        return new SaveTenantAuthInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tenantId = source["tenantId"];
	        this.accessToken = source["accessToken"];
	        this.refreshToken = source["refreshToken"];
	        this.tokenType = source["tokenType"];
	        this.expiresAt = source["expiresAt"];
	        this.refreshExpiresAt = source["refreshExpiresAt"];
	        this.sessionId = source["sessionId"];
	        this.userId = source["userId"];
	        this.username = source["username"];
	        this.name = source["name"];
	        this.roles = source["roles"];
	    }
	}
	export class ScanFilterRule {
	    field: string;
	    operator: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanFilterRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.field = source["field"];
	        this.operator = source["operator"];
	        this.value = source["value"];
	    }
	}
	export class ScanSiteLinksInput {
	    url: string;
	    maxLinks: number;
	    scanSitemap: boolean;
	    filterRules?: ScanFilterRule[];
	    siteId?: number;
	    tenantId?: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanSiteLinksInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.maxLinks = source["maxLinks"];
	        this.scanSitemap = source["scanSitemap"];
	        this.filterRules = this.convertValues(source["filterRules"], ScanFilterRule);
	        this.siteId = source["siteId"];
	        this.tenantId = source["tenantId"];
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
	export class SiteLinkItem {
	    url: string;
	    title: string;
	    source?: string;
	
	    static createFrom(source: any = {}) {
	        return new SiteLinkItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.title = source["title"];
	        this.source = source["source"];
	    }
	}
	export class ScanSiteLinksResult {
	    requestedUrl: string;
	    finalUrl: string;
	    host: string;
	    siteName?: string;
	    title: string;
	    links: SiteLinkItem[];
	    pageHtmlCount: number;
	    sitemapUrlCount: number;
	    sitemapSources?: string[];
	    scannedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanSiteLinksResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestedUrl = source["requestedUrl"];
	        this.finalUrl = source["finalUrl"];
	        this.host = source["host"];
	        this.siteName = source["siteName"];
	        this.title = source["title"];
	        this.links = this.convertValues(source["links"], SiteLinkItem);
	        this.pageHtmlCount = source["pageHtmlCount"];
	        this.sitemapUrlCount = source["sitemapUrlCount"];
	        this.sitemapSources = source["sitemapSources"];
	        this.scannedAt = source["scannedAt"];
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
	export class SiteDiscoveryInfo {
	    host: string;
	    baseUrl: string;
	    isSystemHost: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SiteDiscoveryInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.baseUrl = source["baseUrl"];
	        this.isSystemHost = source["isSystemHost"];
	    }
	}
	
	
	export class SubmitRecordItem {
	    id: number;
	    jobId: number;
	    siteId: number;
	    tenantId: number;
	    tenantName: string;
	    title: string;
	    contentType: string;
	    jobType: string;
	    status: string;
	    idempotencyKey: string;
	    remoteId: number;
	    remoteUrl: string;
	    matchType: string;
	    createdCount: number;
	    updatedCount: number;
	    failedCount: number;
	    errorMessage: string;
	    payloadJson: string;
	    resultJson: string;
	    submittedAt: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new SubmitRecordItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.jobId = source["jobId"];
	        this.siteId = source["siteId"];
	        this.tenantId = source["tenantId"];
	        this.tenantName = source["tenantName"];
	        this.title = source["title"];
	        this.contentType = source["contentType"];
	        this.jobType = source["jobType"];
	        this.status = source["status"];
	        this.idempotencyKey = source["idempotencyKey"];
	        this.remoteId = source["remoteId"];
	        this.remoteUrl = source["remoteUrl"];
	        this.matchType = source["matchType"];
	        this.createdCount = source["createdCount"];
	        this.updatedCount = source["updatedCount"];
	        this.failedCount = source["failedCount"];
	        this.errorMessage = source["errorMessage"];
	        this.payloadJson = source["payloadJson"];
	        this.resultJson = source["resultJson"];
	        this.submittedAt = source["submittedAt"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class SubmitRecordListInput {
	    tenantId: number;
	    keyword: string;
	    contentType: string;
	    status: string;
	    dateFrom: string;
	    dateTo: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new SubmitRecordListInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tenantId = source["tenantId"];
	        this.keyword = source["keyword"];
	        this.contentType = source["contentType"];
	        this.status = source["status"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.limit = source["limit"];
	    }
	}
	export class SyncSiteTenantsInput {
	    siteId: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncSiteTenantsInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.siteId = source["siteId"];
	    }
	}
	
	
	export class TenantDiscoveryLoginHint {
	    mode: string;
	    tenantSlug: string;
	
	    static createFrom(source: any = {}) {
	        return new TenantDiscoveryLoginHint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.tenantSlug = source["tenantSlug"];
	    }
	}
	export class TenantDiscoveryItem {
	    slug: string;
	    name: string;
	    hasActiveDomain: boolean;
	    recommendedBaseUrl: string;
	    apiBaseUrl: string;
	    loginHint: TenantDiscoveryLoginHint;
	
	    static createFrom(source: any = {}) {
	        return new TenantDiscoveryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slug = source["slug"];
	        this.name = source["name"];
	        this.hasActiveDomain = source["hasActiveDomain"];
	        this.recommendedBaseUrl = source["recommendedBaseUrl"];
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.loginHint = this.convertValues(source["loginHint"], TenantDiscoveryLoginHint);
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
	
	export class TenantDiscoveryPayload {
	    site: SiteDiscoveryInfo;
	    tenants: TenantDiscoveryItem[];
	
	    static createFrom(source: any = {}) {
	        return new TenantDiscoveryPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.site = this.convertValues(source["site"], SiteDiscoveryInfo);
	        this.tenants = this.convertValues(source["tenants"], TenantDiscoveryItem);
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
	
	
	
	export class UpdateSiteInput {
	    id: number;
	    name: string;
	    baseUrl: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSiteInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.baseUrl = source["baseUrl"];
	        this.description = source["description"];
	    }
	}
	export class UpdateTenantInput {
	    id: number;
	    siteId: number;
	    name: string;
	    baseUrl: string;
	    apiBaseUrl: string;
	    tenantName: string;
	    tenantSlug: string;
	    lastUsername: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateTenantInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.siteId = source["siteId"];
	        this.name = source["name"];
	        this.baseUrl = source["baseUrl"];
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.tenantName = source["tenantName"];
	        this.tenantSlug = source["tenantSlug"];
	        this.lastUsername = source["lastUsername"];
	    }
	}
	export class UploadDesktopMediaInput {
	    apiBaseUrl: string;
	    accessToken: string;
	    fileName: string;
	    mimeType: string;
	    fileBase64: string;
	    originalName: string;
	    mediaCategoryId: number;
	    sourceUrl: string;
	    draftId: number;
	    uploadScene: string;
	
	    static createFrom(source: any = {}) {
	        return new UploadDesktopMediaInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiBaseUrl = source["apiBaseUrl"];
	        this.accessToken = source["accessToken"];
	        this.fileName = source["fileName"];
	        this.mimeType = source["mimeType"];
	        this.fileBase64 = source["fileBase64"];
	        this.originalName = source["originalName"];
	        this.mediaCategoryId = source["mediaCategoryId"];
	        this.sourceUrl = source["sourceUrl"];
	        this.draftId = source["draftId"];
	        this.uploadScene = source["uploadScene"];
	    }
	}

}

