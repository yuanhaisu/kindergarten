module.exports = (function() {
var __MODS__ = {};
var __DEFINE__ = function(modId, func, req) { var m = { exports: {}, _tempexports: {} }; __MODS__[modId] = { status: 0, func: func, req: req, m: m }; };
var __REQUIRE__ = function(modId, source) { if(!__MODS__[modId]) return require(source); if(!__MODS__[modId].status) { var m = __MODS__[modId].m; m._exports = m._tempexports; var desp = Object.getOwnPropertyDescriptor(m, "exports"); if (desp && desp.configurable) Object.defineProperty(m, "exports", { set: function (val) { if(typeof val === "object" && val !== m._exports) { m._exports.__proto__ = val.__proto__; Object.keys(val).forEach(function (k) { m._exports[k] = val[k]; }); } m._tempexports = val }, get: function () { return m._tempexports; } }); __MODS__[modId].status = 1; __MODS__[modId].func(__MODS__[modId].req, m, m.exports); } return __MODS__[modId].m.exports; };
var __REQUIRE_WILDCARD__ = function(obj) { if(obj && obj.__esModule) { return obj; } else { var newObj = {}; if(obj != null) { for(var k in obj) { if (Object.prototype.hasOwnProperty.call(obj, k)) newObj[k] = obj[k]; } } newObj.default = obj; return newObj; } };
var __REQUIRE_DEFAULT__ = function(obj) { return obj && obj.__esModule ? obj.default : obj; };
__DEFINE__(1781530615240, function(require, module, exports) {

var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const path_1 = __importDefault(require("path"));
const plugin_basic_ssl_1 = __importDefault(require("@vitejs/plugin-basic-ssl"));
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const vueCompilerDom = __importStar(require("@vue/compiler-dom"));
const uniCliShared = __importStar(require("@dcloudio/uni-cli-shared"));
const plugin_1 = require("./plugin");
const config_1 = require("./plugin/config");
const css_1 = require("./plugins/css");
const easycom_1 = require("./plugins/easycom");
const inject_1 = require("./plugins/inject");
const mainJs_1 = require("./plugins/mainJs");
const manifestJson_1 = require("./plugins/manifestJson");
const pagesJson_1 = require("./plugins/pagesJson");
const postVue_1 = require("./plugins/postVue");
const renderjs_1 = require("./plugins/renderjs");
const resolveId_1 = require("./plugins/resolveId");
const setup_1 = require("./plugins/setup");
const ssr_1 = require("./plugins/ssr");
const sourcemap_1 = require("./plugins/sourcemap");
const customElement_1 = require("./plugins/customElement");
const api_1 = require("./plugins/api");
const polyfill_1 = require("./utils/polyfill");
if (process.env.UNI_APP_STYLE_ISOLATION_VERSION === '2' &&
    process.env.UNI_APP_X === 'true') {
    (0, polyfill_1.rewriteCompilerSfcParse)();
}
exports.default = () => {
    const isNewStyleIsolation = process.env.UNI_APP_STYLE_ISOLATION_VERSION === '2';
    // 从 manifest.json 的 h5.devServer 中解析 HTTPS 扩展配置，按需注入 basic-ssl 插件。
    const h5BasicSslPlugin = resolveH5BasicSslPlugin();
    return [
        ...(process.env.UNI_APP_X === 'true' && (0, uni_cli_shared_1.isNormalCompileTarget)()
            ? [(0, uni_cli_shared_1.uniWorkersPlugin)(), (0, uni_cli_shared_1.uniJavaScriptWorkersPlugin)()]
            : []),
        ...((0, uni_cli_shared_1.isEnableConsole)() ? [(0, uni_cli_shared_1.uniHBuilderXConsolePlugin)('uni.__f__')] : []),
        ...(process.env.UNI_APP_X === 'true'
            ? [
                (0, uni_cli_shared_1.uniDecryptUniModulesPlugin)(),
                (0, uni_cli_shared_1.uniUTSUVueJavaScriptPlugin)(),
                (0, uni_cli_shared_1.resolveUTSCompiler)().uts2js({
                    platform: 'web',
                    inputDir: process.env.UNI_INPUT_DIR,
                    version: process.env.UNI_COMPILER_VERSION,
                    sourceMap: (0, uni_cli_shared_1.enableSourceMap)(),
                    cacheRoot: path_1.default.resolve(process.env.UNI_APP_X_CACHE_DIR, '.uts2js/cache'),
                    modules: {
                        vueCompilerDom,
                        uniCliShared,
                    },
                    workers: {
                        extname: '.js',
                        resolve: () => {
                            return (0, uni_cli_shared_1.getWorkers)();
                        },
                    },
                }),
            ]
            : []),
        (0, easycom_1.uniEasycomPlugin)({ exclude: uni_cli_shared_1.UNI_EASYCOM_EXCLUDE }),
        (0, uni_cli_shared_1.uniCssScopedPlugin)({
            filter: (id) => {
                // Vapor 模式下，App.vue 也需要处理
                if (isNewStyleIsolation) {
                    return (0, uni_cli_shared_1.isVueSfcFile)(id);
                }
                return (0, uni_cli_shared_1.isVueSfcFile)(id) && !(0, uni_cli_shared_1.isAppVue)(id);
            },
        }),
        (0, resolveId_1.uniResolveIdPlugin)(),
        ...(process.env.UNI_COMPILE_TARGET === 'uni_modules'
            ? []
            : [(0, mainJs_1.uniMainJsPlugin)(), (0, manifestJson_1.uniManifestJsonPlugin)(), (0, pagesJson_1.uniPagesJsonPlugin)()]),
        (0, inject_1.uniInjectPlugin)(),
        (0, css_1.uniCssPlugin)(),
        (0, ssr_1.uniSSRPlugin)(),
        (0, setup_1.uniSetupPlugin)(),
        (0, renderjs_1.uniRenderjsPlugin)(),
        ...(h5BasicSslPlugin ? [h5BasicSslPlugin] : []),
        (0, plugin_1.uniH5Plugin)(),
        ...(process.env.UNI_COMPILE_TARGET === 'uni_modules'
            ? [(0, uni_cli_shared_1.uniEncryptUniModulesAssetsPlugin)(), (0, uni_cli_shared_1.uniEncryptUniModulesPlugin)()]
            : []),
        (0, postVue_1.uniPostVuePlugin)(),
        (0, sourcemap_1.uniPostSourceMapPlugin)(),
        (0, customElement_1.uniCustomElementPlugin)(),
        (0, api_1.uniApiPlugin)(),
    ];
};
function resolveH5BasicSslPlugin() {
    const inputDir = process.env.UNI_INPUT_DIR;
    if (!inputDir) {
        return;
    }
    // 这里只关心是否启用自动证书，以及透传给 basic-ssl 的那部分参数。
    const { enableBasicSsl, basicSslOptions } = (0, config_1.resolveManifestServerOptions)(inputDir);
    if (!enableBasicSsl) {
        return;
    }
    // 需要在真正的 Vite 插件数组中注册，才能参与 configResolved 并注入证书。
    return Object.assign((0, plugin_basic_ssl_1.default)(basicSslOptions), {
        apply: 'serve',
    });
}

}, function(modId) {var map = {"./plugin":1781530615241,"./plugin/config":1781530615258,"./plugins/css":1781530615260,"./plugins/easycom":1781530615261,"./plugins/inject":1781530615262,"./plugins/mainJs":1781530615263,"./plugins/manifestJson":1781530615264,"./plugins/pagesJson":1781530615265,"./plugins/postVue":1781530615266,"./plugins/renderjs":1781530615267,"./plugins/resolveId":1781530615268,"./plugins/setup":1781530615269,"./plugins/ssr":1781530615270,"./plugins/customElement":1781530615272,"./plugins/api":1781530615273,"./utils/polyfill":1781530615274}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615241, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.uniH5Plugin = void 0;
const handleHotUpdate_1 = require("./handleHotUpdate");
const transformIndexHtml_1 = require("./transformIndexHtml");
const configureServer_1 = require("./configureServer");
const uni_1 = require("./uni");
const config_1 = require("./config");
function uniH5Plugin() {
    const configOptions = {
        resolvedConfig: null,
    };
    return {
        name: 'uni:h5',
        uni: (0, uni_1.createUni)(),
        config: (0, config_1.createConfig)(configOptions),
        configResolved(config) {
            configOptions.resolvedConfig = config;
        },
        configureServer: (0, configureServer_1.createConfigureServer)(),
        handleHotUpdate: (0, handleHotUpdate_1.createHandleHotUpdate)(),
        transformIndexHtml: (0, transformIndexHtml_1.createTransformIndexHtml)(),
    };
}
exports.uniH5Plugin = uniH5Plugin;

}, function(modId) { var map = {"./handleHotUpdate":1781530615242,"./configureServer":1781530615250,"./uni":1781530615255,"./config":1781530615258}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615242, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.createHandleHotUpdate = void 0;
const path_1 = __importDefault(require("path"));
const debug_1 = __importDefault(require("debug"));
const shared_1 = require("@vue/shared");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const utils_1 = require("../../utils");
const debugHmr = (0, debug_1.default)('uni:hmr');
async function invalidate(file, moduleGraph) {
    const mods = await moduleGraph.getModulesByFile((0, uni_cli_shared_1.normalizePath)(file));
    if (mods && mods.size) {
        ;
        [...mods].forEach((mod) => {
            debugHmr('invalidate', mod.id);
            moduleGraph.invalidateModule(mod);
        });
    }
}
let invalidateFiles;
function createHandleHotUpdate() {
    return async function ({ file, server }) {
        const inputDir = process.env.UNI_INPUT_DIR;
        const platform = process.env.UNI_PLATFORM;
        if (!invalidateFiles) {
            invalidateFiles = [
                path_1.default.resolve(inputDir, uni_cli_shared_1.PAGES_JSON_JS),
                path_1.default.resolve(inputDir, uni_cli_shared_1.MANIFEST_JSON_JS),
                (0, uni_cli_shared_1.resolveBuiltIn)('@dcloudio/uni-h5/' + (0, utils_1.resolveFrameworkDistDir)() + '/uni-h5.es.js'),
            ];
            try {
                invalidateFiles.push((0, uni_cli_shared_1.resolveBuiltIn)('vite/dist/client/env.mjs'));
            }
            catch (e) { }
        }
        // TODO 目前简单处理，当pages.json,manifest.json发生变化，就直接刷新，理想情况下，应该区分变化的内容，仅必要时做整页面刷新
        const isPagesJson = file.endsWith('pages.json');
        const isManifestJson = file.endsWith('manifest.json');
        if (!isPagesJson && !isManifestJson) {
            return;
        }
        debugHmr(file);
        const pagesJson = (0, uni_cli_shared_1.parsePagesJson)(inputDir, platform);
        // 更新define
        const { define, server: { middlewareMode }, } = server.config;
        (0, shared_1.extend)(define, (0, uni_cli_shared_1.initFeatures)({
            inputDir,
            command: 'serve',
            platform,
            pagesJson,
            manifestJson: (0, uni_cli_shared_1.parseManifestJson)(inputDir),
            ssr: !!middlewareMode,
        }));
        debugHmr('define', define);
        if (isPagesJson) {
            const easycom = pagesJson.easycom || {};
            const { easyComOptions, refresh } = (0, uni_cli_shared_1.initEasycomsOnce)(inputDir, {
                dirs: (0, uni_cli_shared_1.resolveComponentsLibDirs)(),
                platform,
                isX: process.env.UNI_APP_X === 'true',
            });
            if (!equal({ autoscan: easycom.autoscan, custom: easycom.custom }, {
                autoscan: easyComOptions.autoscan,
                custom: easyComOptions.custom,
            })) {
                refresh();
            }
        }
        // 当pages.json,manifest.json发生变化时，作废pages.json.js缓存
        for (const file of invalidateFiles) {
            await invalidate(file, server.moduleGraph);
        }
        server.ws.send({
            type: 'full-reload',
            path: '*',
        });
        return [];
    };
}
exports.createHandleHotUpdate = createHandleHotUpdate;
function equal(obj1, obj2) {
    return JSON.stringify(obj1) === JSON.stringify(obj2);
}

}, function(modId) { var map = {"../../utils":1781530615243}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615243, function(require, module, exports) {

var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
Object.defineProperty(exports, "__esModule", { value: true });
__exportStar(require("./ssr"), exports);
__exportStar(require("./features"), exports);
__exportStar(require("./constants"), exports);
__exportStar(require("./utils"), exports);

}, function(modId) { var map = {"./features":1781530615245,"./constants":1781530615247,"./utils":1781530615248}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615245, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.createDefine = void 0;
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const ssr_1 = require("./ssr");
function createDefine(command, config) {
    const platform = process.env.UNI_PLATFORM;
    const inputDir = process.env.UNI_INPUT_DIR;
    return (0, uni_cli_shared_1.initFeatures)({
        inputDir,
        command,
        platform,
        pagesJson: (0, uni_cli_shared_1.parsePagesJsonOnce)(inputDir, platform),
        manifestJson: (0, uni_cli_shared_1.parseManifestJsonOnce)(inputDir),
        ssr: (0, uni_cli_shared_1.isSsr)(command, config) || (0, ssr_1.isSsrManifest)(command, config),
    });
}
exports.createDefine = createDefine;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615247, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.AliYunCloudAuthWebSDK = exports.ownerModuleName = void 0;
exports.ownerModuleName = '@dcloudio/uni-h5';
exports.AliYunCloudAuthWebSDK = 'https://cn-shanghai-aliyun-cloudauth.oss-cn-shanghai.aliyuncs.com/web_sdk_js/jsvm_all.js';

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615248, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.resolveDistDir = exports.resolveVueDistDir = exports.resolveFrameworkDistDir = void 0;
function resolveFrameworkDistDir() {
    return resolveDistDir();
}
exports.resolveFrameworkDistDir = resolveFrameworkDistDir;
function resolveVueDistDir() {
    return resolveDistDir();
}
exports.resolveVueDistDir = resolveVueDistDir;
function resolveDistDir() {
    // // 重要：目前只要manifest.json中配置了vapor:true，就认为是vapor版本（虽然还没有支持）
    // process.env.UNI_APP_X_VAPOR === 'true'
    //   ? 'dist-x-vapor'
    //   :
    return process.env.UNI_APP_X === 'true' ? 'dist-x' : 'dist';
}
exports.resolveDistDir = resolveDistDir;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615250, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.createConfigureServer = void 0;
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const timestamp_1 = require("./middlewares/timestamp");
const ssr_1 = require("./ssr");
const static_1 = require("./static");
function createConfigureServer() {
    return function (server) {
        (0, ssr_1.initSSR)(server);
        const routerOptions = (0, uni_cli_shared_1.getRouterOptions)((0, uni_cli_shared_1.parseManifestJsonOnce)(process.env.UNI_INPUT_DIR));
        if (routerOptions.mode === 'history') {
            server.middlewares.use((0, timestamp_1.uniTimestampMiddleware)(server));
        }
        return () => {
            (0, static_1.initStatic)(server);
        };
    };
}
exports.createConfigureServer = createConfigureServer;

}, function(modId) { var map = {"./middlewares/timestamp":1781530615251,"./ssr":1781530615252,"./static":1781530615253}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615251, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniTimestampMiddleware = void 0;
const url_1 = require("url");
const path_1 = __importDefault(require("path"));
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
function uniTimestampMiddleware(server) {
    return async function timestampMiddleware(req, _, next) {
        // 当页面被作为组件引用时，会导致history刷新该页面直接显示js代码，因为该页面已被缓存为了module，
        // https://github.com/vitejs/vite/blob/702d50315535c189151c67d33e4a22124f926bed/packages/vite/src/node/server/transformRequest.ts#L52
        // /pages/tabBar/API/API
        let { url } = req;
        if (url) {
            const base = server.config.base;
            const parsed = (0, url_1.parse)(url);
            let newUrl = url;
            if ((parsed.pathname || '/').startsWith(base)) {
                newUrl = newUrl.replace(base, '/');
            }
            if (!path_1.default.extname(newUrl) &&
                !newUrl.endsWith('/') &&
                !newUrl.includes('?')) {
                const module = await server.moduleGraph.getModuleByUrl(newUrl);
                if (module && module.file && uni_cli_shared_1.EXTNAME_VUE_RE.test(module.file)) {
                    // /pages/tabBar/API/API => /pages/tabBar/API/API?__t__=time
                    req.url = url + '?__t__=' + Date.now();
                }
            }
        }
        next();
    };
}
exports.uniTimestampMiddleware = uniTimestampMiddleware;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615252, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.initSSR = exports.external = void 0;
exports.external = [
    '@dcloudio/uni-app',
    '@dcloudio/uni-app-plus',
    '@dcloudio/uni-cloud',
    '@dcloudio/uni-components',
    '@dcloudio/uni-h5',
    '@dcloudio/uni-h5-vue',
    '@dcloudio/uni-i18n',
    '@dcloudio/uni-mp-alipay',
    '@dcloudio/uni-mp-baidu',
    '@dcloudio/uni-mp-kuaishou',
    '@dcloudio/uni-mp-lark',
    '@dcloudio/uni-mp-qq',
    '@dcloudio/uni-mp-toutiao',
    '@dcloudio/uni-mp-weixin',
    '@dcloudio/uni-quickapp-webview',
    '@dcloudio/uni-shared',
    '@dcloudio/uni-stat',
    '@dcloudio/uni-stacktracey',
    '@vue/shared',
    'vue',
    'vue-i18n',
    'vue-router',
    'vuex',
    // dev
    '@dcloudio/types',
    '@dcloudio/uni-automator',
    '@dcloudio/uni-cli-shared',
    '@dcloudio/vite-plugin-uni',
    'autoprefixer',
    'typescript',
    'vite',
];
function initSSR(server) {
    const { ssrLoadModule } = server;
    let added = false;
    server.ssrLoadModule = (url) => {
        const res = ssrLoadModule(url);
        if (!added) {
            // HBuilderX项目，根目录可能没有package.json，导致 ssrExternals 不生效
            added = true;
            if (server._ssrExternals) {
                const { _ssrExternals } = server;
                exports.external.forEach((module) => {
                    if (!_ssrExternals.includes(module)) {
                        _ssrExternals.push(module);
                    }
                });
            }
        }
        return res;
    };
}
exports.initSSR = initSSR;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615253, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.createPublicFileFilter = exports.initStatic = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const debug_1 = __importDefault(require("debug"));
const pluginutils_1 = require("@rollup/pluginutils");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const static_1 = require("./middlewares/static");
const debugStatic = (0, debug_1.default)('uni:static');
/**
 * devServer时提供static等目录的静态资源服务
 * @param server
 * @param param
 */
const initStatic = (server) => {
    const filter = createPublicFileFilter();
    const serve = (0, static_1.uniStaticMiddleware)({
        etag: true,
        resolve(pathname) {
            if (!filter(pathname)) {
                return;
            }
            const filename = path_1.default.join(process.env.UNI_INPUT_DIR, pathname);
            if (fs_1.default.existsSync(filename)) {
                debugStatic(filename, 'success');
                return filename;
            }
            else {
                debugStatic(filename, 'fail');
            }
        },
    });
    const viteServePublicMiddlewareIndex = server.middlewares.stack.findIndex((middleware) => {
        return (middleware.handle.name === 'viteServePublicMiddleware');
    });
    // 替换 vite 自带的 public middleware
    if (viteServePublicMiddlewareIndex > -1) {
        server.middlewares.stack.splice(viteServePublicMiddlewareIndex, 1, {
            route: '',
            handle: ((req, res, next) => {
                if ((0, uni_cli_shared_1.isImportRequest)(req.url) || (0, uni_cli_shared_1.isInternalRequest)(req.url)) {
                    return next();
                }
                return serve(req, res, next);
            }),
        });
    }
};
exports.initStatic = initStatic;
function createPublicFileFilter(base = '/') {
    const publicDir = (0, uni_cli_shared_1.normalizePath)(path_1.default.join(base, uni_cli_shared_1.PUBLIC_DIR + '/**/*'));
    const uniModulesDir = (0, uni_cli_shared_1.normalizePath)(path_1.default.join(base, 'uni_modules/*/' + uni_cli_shared_1.PUBLIC_DIR + '/**/*'));
    return (0, pluginutils_1.createFilter)([publicDir, uniModulesDir]);
}
exports.createPublicFileFilter = createPublicFileFilter;

}, function(modId) { var map = {"./middlewares/static":1781530615254}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615254, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniStaticMiddleware = void 0;
const fs_1 = __importDefault(require("fs"));
const url_1 = __importDefault(require("url"));
const lite_1 = __importDefault(require("mime/lite"));
function normalizeFile(filename, isEtag) {
    const stats = fs_1.default.statSync(filename);
    return {
        stats,
        headers: normalizeHeaders(filename, stats, isEtag),
    };
}
function normalizeHeaders(filename, stats, isEtag) {
    const headers = {
        'Content-Length': stats.size,
        'Content-Type': lite_1.default.getType(filename) || '',
        'Last-Modified': stats.mtime.toUTCString(),
    };
    if (isEtag) {
        headers['ETag'] = `W/"${stats.size}-${stats.mtime.getTime()}"`;
    }
    return headers;
}
function send(req, res, filename, stats, headers) {
    let code = 200;
    headers = { ...headers };
    const opts = {};
    for (const key in headers) {
        const value = res.getHeader(key);
        if (value) {
            headers[key] = value;
        }
    }
    if (res.getHeader('content-type')) {
        headers['Content-Type'] = res.getHeader('content-type');
    }
    if (req.headers.range) {
        code = 206;
        const [x, y] = req.headers.range.replace('bytes=', '').split('-');
        const end = (opts.end = parseInt(y, 10) || stats.size - 1);
        const start = (opts.start = parseInt(x, 10) || 0);
        if (start >= stats.size || end >= stats.size) {
            res.setHeader('Content-Range', `bytes */${stats.size}`);
            res.statusCode = 416;
            return res.end();
        }
        headers['Content-Range'] = `bytes ${start}-${end}/${stats.size}`;
        headers['Content-Length'] = end - start + 1;
        headers['Accept-Ranges'] = 'bytes';
    }
    res.writeHead(code, headers);
    fs_1.default.createReadStream(filename, opts).pipe(res);
}
function uniStaticMiddleware(opts) {
    const isEtag = !!opts.etag;
    return function staticMiddleware(req, res, next) {
        const pathname = url_1.default.parse(req.url).pathname;
        if (!pathname) {
            return next();
        }
        const filename = opts.resolve(pathname);
        if (!filename) {
            return next();
        }
        const data = normalizeFile(filename, isEtag);
        if (!data) {
            return next();
        }
        if (isEtag && req.headers['if-none-match'] === data.headers['ETag']) {
            res.writeHead(304);
            return res.end();
        }
        return send(req, res, filename, data.stats, data.headers);
    };
}
exports.uniStaticMiddleware = uniStaticMiddleware;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615255, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.createUni = exports.compilerOptions = void 0;
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const uni_shared_1 = require("@dcloudio/uni-shared");
const transformCustomElement_1 = require("./transforms/transformCustomElement");
const transformAttributePart_1 = require("./transforms/transformAttributePart");
function realIsH5CustomElement(tag) {
    // TODO isH5CustomElement目前被多个平台引用，重构比较麻烦
    if (process.env.UNI_APP_X === 'true' &&
        uni_shared_1.UVUE_WEB_BUILT_IN_CUSTOM_ELEMENTS.includes(tag)) {
        return true;
    }
    return (0, uni_shared_1.isH5CustomElement)(tag, process.env.UNI_APP_X === 'true');
}
const nodeTransforms = [
    uni_cli_shared_1.transformRefresherSlot,
    uni_cli_shared_1.transformH5BuiltInComponents,
    uni_cli_shared_1.transformTapToClick,
    uni_cli_shared_1.transformMatchMedia,
    uni_cli_shared_1.transformPageHead,
];
if (process.env.UNI_APP_X === 'true') {
    nodeTransforms.splice(nodeTransforms.indexOf(uni_cli_shared_1.transformMatchMedia), 1);
    nodeTransforms.push(transformCustomElement_1.transformCustomElement);
    if (process.env.UNI_UTS_PLATFORM === 'web') {
        nodeTransforms.push(transformAttributePart_1.transformAttributePart);
    }
}
exports.compilerOptions = {
    isNativeTag: uni_shared_1.isH5NativeTag,
    isCustomElement: realIsH5CustomElement,
    nodeTransforms,
};
function createUni() {
    return {
        copyOptions: {
            assets: ['hybrid/html/**/*', 'uni_modules/*/hybrid/html/**/*'],
        },
        compilerOptions: exports.compilerOptions,
        jsxOptions: {
            babelPlugins: [uni_cli_shared_1.transformUniH5Jsx],
        },
    };
}
exports.createUni = createUni;

}, function(modId) { var map = {"./transforms/transformCustomElement":1781530615256}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615256, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.transformCustomElement = void 0;
const compiler_core_1 = require("@vue/compiler-core");
const uni_shared_1 = require("@dcloudio/uni-shared");
const transformCustomElement = (node, context) => {
    if (!!node &&
        node.type === compiler_core_1.NodeTypes.ELEMENT &&
        uni_shared_1.UVUE_WEB_BUILT_IN_CUSTOM_ELEMENTS.includes(node.tag)) {
        node.tag = `$UniCustomElement$${node.tag}`;
    }
};
exports.transformCustomElement = transformCustomElement;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615258, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.resolveManifestServerOptions = exports.createConfig = void 0;
const os_1 = __importDefault(require("os"));
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const shared_1 = require("@vue/shared");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const utils_1 = require("../utils");
const esbuildPrePlugin_1 = require("./esbuild/esbuildPrePlugin");
const ssr_1 = require("./configureServer/ssr");
function createConfig(options) {
    return function config(config, env) {
        const inputDir = process.env.UNI_INPUT_DIR;
        if ((0, uni_cli_shared_1.isInHBuilderX)()) {
            if (!fs_1.default.existsSync(path_1.default.resolve(inputDir, 'index.html'))) {
                console.error(`请确认您的项目模板是否支持vue3：根目录缺少 index.html`);
                process.exit();
            }
        }
        // 先对 manifest 中的 devServer 做一次清洗，避免把自定义字段直接透传给 Vite。
        const { server: manifestServer } = resolveManifestServerOptions(inputDir);
        const server = {
            hmr: process.env.UNI_AUTOMATOR_WS_ENDPOINT
                ? false
                : {
                    // mac 内置浏览器版本较低不支持 globalThis，而 overlay 使用了 globalThis
                    overlay: os_1.default.platform() !== 'win32'
                        ? process.env.UNI_H5_BROWSER !== 'builtin'
                        : true,
                },
            fs: { strict: false },
            watch: {
                ignored: [
                    '**/uniCloud-aliyun/**',
                    '**/uniCloud-tcb/**',
                    '**/uniCloud-alipay/**',
                    '**/uniCloud-dcloud/**',
                    '**/uni_modules/uniCloud/**',
                    '**/__snapshots__/**',
                    (0, uni_cli_shared_1.normalizePath)(path_1.default.join(inputDir, 'unpackage/**')),
                    (0, uni_cli_shared_1.normalizePath)(path_1.default.join(inputDir, 'dist/**')),
                ],
            },
            ...manifestServer,
        };
        if ((0, uni_cli_shared_1.runByHBuilderX)()) {
            // 仅在 HBuilderX 中运行时，将 host 设置为 true，cli 项目命令行运行需要自行开启 --host 参数
            server.host = true;
        }
        if (server.port === '') {
            delete server.port;
        }
        const { server: userServer } = config;
        if (userServer) {
            if ((0, shared_1.hasOwn)(userServer, 'host')) {
                server.host = userServer.host;
            }
            if ((0, shared_1.hasOwn)(userServer, 'fs')) {
                (0, shared_1.extend)(server.fs, userServer.fs);
            }
            if ((0, shared_1.hasOwn)(userServer, 'watch')) {
                (0, shared_1.extend)(server.watch, userServer.watch);
            }
        }
        let sourcemapPathTransform = undefined;
        if (
        // 仅在 uni-app-x 模式下，且非开发模式，且需要 sourcemap 时，才进行 sourcemap 路径转换
        process.env.UNI_APP_X === 'true' &&
            process.env.NODE_ENV !== 'development' &&
            (0, uni_cli_shared_1.withSourcemap)(config)) {
            sourcemapPathTransform = transformSourcemapPath;
        }
        return {
            legacy: {
                // 目前先使用旧模式
                proxySsrExternalModules: true,
            },
            css: {
                postcss: {
                    plugins: (0, uni_cli_shared_1.initPostcssPlugin)({
                        uniApp: (0, uni_cli_shared_1.parseRpx2UnitOnce)(inputDir, process.env.UNI_PLATFORM),
                    }),
                },
            },
            optimizeDeps: {
                entries: (0, uni_cli_shared_1.resolveMainPathOnce)(inputDir),
                exclude: ssr_1.external,
                esbuildOptions: {
                    plugins: [(0, esbuildPrePlugin_1.esbuildPrePlugin)()],
                },
            },
            define: (0, utils_1.createDefine)(env.command, config),
            server,
            ssr: {
                external: ssr_1.external,
            },
            build: {
                target: process.env.UNI_APP_X === 'true'
                    ? ['es2015', 'edge79', 'firefox62', 'chrome64', 'safari11.1']
                    : undefined,
                rollupOptions: {
                    // resolveSSRExternal 会判定package.json，hbx 工程可能没有，通过 rollup 来配置
                    external: (0, uni_cli_shared_1.isSsr)(env.command, config) ? ssr_1.external : [],
                    output: {
                        sourcemapPathTransform,
                        chunkFileNames(chunkInfo) {
                            const hash = 
                            // 为了测试额外加的逻辑，避免因为环境不一致导致hash有变化
                            process.env.UNI_WEB_DISABLE_CHUNK_HASH === 'true'
                                ? ''
                                : '.[hash]';
                            const { assetsDir } = options.resolvedConfig.build;
                            if (chunkInfo.facadeModuleId) {
                                const dirname = (0, uni_cli_shared_1.normalizePath)(path_1.default.relative(inputDir, path_1.default.dirname(chunkInfo.facadeModuleId)));
                                if (dirname) {
                                    // 保留workers的目录结构，目前不支持不同的workers引入同一个uts文件，因为目前不能很好的分别打包进各自的chunk中
                                    const workersRootDir = (0, uni_cli_shared_1.getWorkersRootDirs)();
                                    if (workersRootDir.length) {
                                        if (workersRootDir.some((workersRootDir) => dirname.startsWith(workersRootDir))) {
                                            return `${dirname}/[name].js`;
                                        }
                                    }
                                    return path_1.default.posix.join(assetsDir, dirname.replace(/\//g, '-') + `-[name]${hash}.js`);
                                }
                            }
                            return path_1.default.posix.join(assetsDir, `[name]${hash}.js`);
                        },
                    },
                },
            },
        };
    };
}
exports.createConfig = createConfig;
function resolveManifestServerOptions(inputDir) {
    const server = (0, uni_cli_shared_1.getDevServerOptions)((0, uni_cli_shared_1.parseManifestJsonOnce)(inputDir));
    // 约定 https: true 时启用 basic-ssl，由插件补齐实际证书内容。
    const enableBasicSsl = server.https === true;
    const basicSslOptions = enableBasicSsl
        ? normalizeManifestBasicSslOptions(server.basicSsl)
        : undefined;
    // basicSsl 仅作为内置插件的扩展配置使用，不能直接透传给 Vite 的 server 配置。
    if (enableBasicSsl && (0, shared_1.hasOwn)(server, 'basicSsl')) {
        delete server.basicSsl;
    }
    return {
        server,
        basicSslOptions,
        enableBasicSsl,
    };
}
exports.resolveManifestServerOptions = resolveManifestServerOptions;
function normalizeManifestBasicSslOptions(basicSsl) {
    if (!(0, shared_1.isPlainObject)(basicSsl)) {
        return;
    }
    // 仅保留 basic-ssl 已支持的字段，避免把无效配置带入插件。
    const options = {};
    if (typeof basicSsl.name === 'string') {
        options.name = basicSsl.name;
    }
    if (Array.isArray(basicSsl.domains)) {
        options.domains = basicSsl.domains.filter((domain) => typeof domain === 'string');
    }
    if (typeof basicSsl.certDir === 'string') {
        options.certDir = basicSsl.certDir;
    }
    return Object.keys(options).length ? options : undefined;
}
function transformSourcemapPath(relativeSourcePath, sourcemapPath) {
    const sourcePath = (0, uni_cli_shared_1.normalizePath)(path_1.default.relative(process.env.UNI_INPUT_DIR, path_1.default.resolve(path_1.default.dirname(sourcemapPath), relativeSourcePath)));
    if (sourcePath.startsWith('..')) {
        return '';
    }
    return sourcePath;
}

}, function(modId) { var map = {"../utils":1781530615243,"./esbuild/esbuildPrePlugin":1781530615259,"./configureServer/ssr":1781530615252}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615259, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.esbuildPrePlugin = exports.JS_TYPES_RE = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
exports.JS_TYPES_RE = /\.(?:j|t)sx?$|\.mjs$/;
function esbuildPrePlugin() {
    return {
        name: 'uni:dep-scan',
        setup(build) {
            build.onLoad({ filter: exports.JS_TYPES_RE }, ({ path: id }) => {
                let ext = path_1.default.extname(id).slice(1);
                if (ext === 'mjs')
                    ext = 'js';
                if (fs_1.default.existsSync(id)) {
                    let contents = fs_1.default.readFileSync(id, 'utf-8');
                    if (contents.includes('#endif')) {
                        contents = (0, uni_cli_shared_1.preJs)(contents, id);
                    }
                    return {
                        loader: ext,
                        contents,
                    };
                }
            });
        },
    };
}
exports.esbuildPrePlugin = esbuildPrePlugin;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615260, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.assetFileNamesToFileName = exports.uniCssPlugin = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const vite_1 = require("vite");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const shared_1 = require("@vue/shared");
function isCombineBuiltInCss(config) {
    return config.command === 'build' && config.build.cssCodeSplit;
}
function uniCssPlugin() {
    let resolvedConfig;
    let file = '';
    let fileName = '';
    let isInternational = false;
    return {
        name: 'uni:h5-css',
        apply: 'build',
        enforce: 'pre',
        configResolved(config) {
            resolvedConfig = config;
            file = path_1.default.join(process.env.UNI_INPUT_DIR, 'uni.css');
            isInternational = !!(process.env.UNI_APP_ID && process.env.UNI_APP_ID.startsWith('__UNI__G'));
            if (process.env.UNI_COMPILE_TARGET === 'uni_modules') {
                (0, uni_cli_shared_1.injectCssPlugin)(config, {
                    createUrlReplacer: uni_cli_shared_1.createEncryptCssUrlReplacer,
                });
                (0, uni_cli_shared_1.injectCssPostPlugin)(config, (0, uni_cli_shared_1.cssPostPlugin)(config, {
                    platform: process.env.UNI_PLATFORM,
                    preserveModules: true,
                    chunkCssFilename(id) {
                        if ((0, uni_cli_shared_1.isVueSfcFile)(id)) {
                            return ((0, uni_cli_shared_1.removeExt)((0, vite_1.normalizePath)(path_1.default.relative(process.env.UNI_INPUT_DIR, id))) + '.css');
                        }
                    },
                    chunkCssCode(_filename, cssCode) {
                        return cssCode;
                    },
                }));
                (0, uni_cli_shared_1.injectAssetPlugin)(config);
            }
        },
        transform(code, id) {
            id = (0, vite_1.normalizePath)(id);
            if (id.endsWith(uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'shadow.css')) {
                const url = (0, uni_cli_shared_1.createShadowImageUrl)(0, 'grey', isInternational);
                return {
                    code: code +
                        `
@keyframes shadow-preload {
  0% {
    background-image: url(${url});
  }
  100% {
    background-image: url(${url});
  }
}
`,
                    map: { mappings: '' },
                };
            }
            else if (id.endsWith(uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'pageHead.css')) {
                return {
                    code: code +
                        `
.uni-page-head-shadow-grey::after {
  background-image: url('${(0, uni_cli_shared_1.createShadowImageUrl)(0, 'grey', isInternational)}');
}

.uni-page-head-shadow-blue::after {
  background-image: url('${(0, uni_cli_shared_1.createShadowImageUrl)(0, 'blue', isInternational)}');
}

.uni-page-head-shadow-green::after {
  background-image: url('${(0, uni_cli_shared_1.createShadowImageUrl)(0, 'green', isInternational)}');
}

.uni-page-head-shadow-orange::after {
  background-image: url('${(0, uni_cli_shared_1.createShadowImageUrl)(0, 'orange', isInternational)}');
}

.uni-page-head-shadow-red::after {
  background-image: url('${(0, uni_cli_shared_1.createShadowImageUrl)(0, 'red', isInternational)}');
}

.uni-page-head-shadow-yellow::after {
  background-image: url('${(0, uni_cli_shared_1.createShadowImageUrl)(0, 'yellow', isInternational)}');
}
            
`,
                    map: { mappings: '' },
                };
            }
        },
        async generateBundle() {
            if (!isCombineBuiltInCss(resolvedConfig) || !uni_cli_shared_1.buildInCssSet.size) {
                return;
            }
            // 生成框架css(需要排序，避免生成不一样的内容)
            const content = await (0, uni_cli_shared_1.minifyCSS)(generateBuiltInCssCode([...uni_cli_shared_1.buildInCssSet].sort()), resolvedConfig);
            // 'Buffer' only refers to a type, but is being used as a value here
            const contentHash = (0, uni_cli_shared_1.getAssetHash)(Buffer.from(content, 'utf-8'));
            const assetFileNames = path_1.default.posix.join(resolvedConfig.build.assetsDir, '[name].[hash][extname]');
            fileName = assetFileNamesToFileName(assetFileNames, file, contentHash, content);
            const name = (0, vite_1.normalizePath)(path_1.default.relative(resolvedConfig.root, file));
            this.emitFile({
                name,
                fileName,
                type: 'asset',
                source: content,
            });
        },
        transformIndexHtml: {
            order: 'post',
            handler() {
                if (!fileName) {
                    return;
                }
                // 追加框架css
                return [
                    {
                        tag: 'link',
                        attrs: {
                            rel: 'stylesheet',
                            href: toPublicPath(fileName, resolvedConfig),
                        },
                        injectTo: 'head-prepend',
                    },
                ];
            },
        },
    };
}
exports.uniCssPlugin = uniCssPlugin;
function toPublicPath(filename, config) {
    return (0, uni_cli_shared_1.isExternalUrl)(filename) ? filename : config.base + filename;
}
function generateBuiltInCssCode(cssImports) {
    return cssImports
        .map((cssImport) => fs_1.default.readFileSync((0, uni_cli_shared_1.resolveBuiltIn)(cssImport), 'utf8'))
        .join('\n');
}
/**
 * converts the source filepath of the asset to the output filename based on the assetFileNames option. \
 * this function imitates the behavior of rollup.js. \
 * https://rollupjs.org/guide/en/#outputassetfilenames
 *
 * @example
 * ```ts
 * const content = Buffer.from('text');
 * const fileName = assetFileNamesToFileName(
 *   'assets/[name].[hash][extname]',
 *   '/path/to/file.txt',
 *   getAssetHash(content),
 *   content
 * )
 * // fileName: 'assets/file.982d9e3e.txt'
 * ```
 *
 * @param assetFileNames filename pattern. e.g. `'assets/[name].[hash][extname]'`
 * @param file filepath of the asset
 * @param contentHash hash of the asset. used for `'[hash]'` placeholder
 * @param content content of the asset. passed to `assetFileNames` if `assetFileNames` is a function
 * @returns output filename
 */
function assetFileNamesToFileName(assetFileNames, file, contentHash, content) {
    const basename = path_1.default.basename(file);
    // placeholders for `assetFileNames`
    // `hash` is slightly different from the rollup's one
    const extname = path_1.default.extname(basename);
    const ext = extname.slice(1);
    const name = basename.slice(0, -extname.length);
    const hash = contentHash;
    if ((0, shared_1.isFunction)(assetFileNames)) {
        assetFileNames = assetFileNames({
            name: file,
            originalFileName: file,
            source: content,
            type: 'asset',
        });
        if (!(0, shared_1.isString)(assetFileNames)) {
            throw new TypeError('assetFileNames must return a string');
        }
    }
    else if (!(0, shared_1.isString)(assetFileNames)) {
        throw new TypeError('assetFileNames must be a string or a function');
    }
    const fileName = assetFileNames.replace(/\[\w+\]/g, (placeholder) => {
        switch (placeholder) {
            case '[ext]':
                return ext;
            case '[extname]':
                return extname;
            case '[hash]':
                return hash;
            case '[name]':
                return name;
        }
        throw new Error(`invalid placeholder ${placeholder} in assetFileNames "${assetFileNames}"`);
    });
    return fileName;
}
exports.assetFileNamesToFileName = assetFileNamesToFileName;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615261, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniEasycomPlugin = void 0;
const fs_extra_1 = __importDefault(require("fs-extra"));
const path_1 = __importDefault(require("path"));
const pluginutils_1 = require("@rollup/pluginutils");
const shared_1 = require("@vue/shared");
const uni_shared_1 = require("@dcloudio/uni-shared");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const H5_COMPONENTS_PATH = '@dcloudio/uni-h5';
const xBaseComponents = ['slider', 'switch', 'loading', 'page-container'];
const baseComponents = [
    'audio',
    'button',
    'canvas',
    'checkbox',
    'checkbox-group',
    'editor',
    'form',
    'icon',
    'image',
    'input',
    'label',
    'movable-area',
    'movable-view',
    'navigator',
    'picker-view',
    'picker-view-column',
    'progress',
    'radio',
    'radio-group',
    'resize-sensor',
    'refresher',
    'rich-text',
    'scroll-view',
    'slider',
    'swiper',
    'swiper-item',
    'switch',
    'text',
    'textarea',
    'view',
    'list-view',
    'list-item',
    'sticky-section',
    'sticky-header',
];
let componentDepsCss;
function uniEasycomPlugin(options) {
    const filter = (0, pluginutils_1.createFilter)(options.include, options.exclude);
    let needCombineBuiltInCss = false;
    componentDepsCss = (0, uni_cli_shared_1.COMPONENT_DEPS_CSS)(process.env.UNI_APP_X === 'true');
    const isDevX = process.env.UNI_HX_VERSION_DEV === 'true' &&
        process.env.UNI_APP_X === 'true';
    return {
        name: 'uni:h5-easycom',
        configResolved(config) {
            needCombineBuiltInCss = (0, uni_cli_shared_1.isCombineBuiltInCss)(config);
        },
        transform(code, id) {
            if (!filter(id)) {
                return;
            }
            const { filename } = (0, uni_cli_shared_1.parseVueRequest)(id);
            if (!uni_cli_shared_1.EXTNAME_VUE_TEMPLATE.includes(path_1.default.extname(filename))) {
                return;
            }
            if (!code.includes('_resolveComponent')) {
                return;
            }
            let i = 0;
            const importDeclarations = [];
            code = code.replace(/_resolveComponent\("(.+?)"(, true)?\)/g, (str, name) => {
                if (name && !name.startsWith('_')) {
                    // 为了兼容性，仅处理dev模式
                    if (!isDevX) {
                        const result = buildInComponent();
                        if (result) {
                            return result;
                        }
                    }
                    if (isDevX && name.startsWith('v-uni-')) {
                        name = name.replace('v-uni-', '');
                    }
                    const source = (0, uni_cli_shared_1.matchEasycom)(name);
                    if (source) {
                        const isHelpers = source.includes('?uni_helpers');
                        if (isHelpers) {
                            const cssFilename = path_1.default.join(process.env.UNI_MODULES_ENCRYPT_CACHE_DIR, path_1.default.relative(process.env.UNI_INPUT_DIR, source.replace('?uni_helpers', '/components/' + name + '/' + name + '.css')));
                            if (fs_extra_1.default.existsSync(cssFilename)) {
                                importDeclarations.push(`import "${(0, uni_cli_shared_1.normalizePath)(cssFilename)}";`);
                            }
                        }
                        // 处理easycom组件优先级
                        return (0, uni_cli_shared_1.genResolveEasycomCode)(importDeclarations, str, (0, uni_cli_shared_1.addImportDeclaration)(importDeclarations, `__easycom_${i++}`, source, isHelpers ? (0, shared_1.capitalize)((0, shared_1.camelize)(name)) : ''));
                    }
                    if (isDevX) {
                        const result = buildInComponent();
                        if (result) {
                            return result;
                        }
                    }
                    function buildInComponent() {
                        if ((0, uni_shared_1.isBuiltInComponent)(name)) {
                            name = name.replace(uni_shared_1.COMPONENT_PREFIX, '');
                            const local = `__syscom_${i++}`;
                            if (needCombineBuiltInCss) {
                                // 发行模式下，应该将内置组件css输出到入口css中
                                resolveBuiltInCssImport(name).forEach((cssImport) => uni_cli_shared_1.buildInCssSet.add(cssImport));
                                return (0, uni_cli_shared_1.addImportDeclaration)(importDeclarations, local, H5_COMPONENTS_PATH, (0, shared_1.capitalize)((0, shared_1.camelize)(name)));
                            }
                            return addBuiltInImportDeclaration(importDeclarations, local, name);
                        }
                    }
                }
                return str;
            });
            if (importDeclarations.length) {
                code = importDeclarations.join('') + code;
            }
            return {
                code,
                map: null,
            };
        },
    };
}
exports.uniEasycomPlugin = uniEasycomPlugin;
function resolveBuiltInCssImport(name) {
    const cssImports = [];
    const isX = process.env.UNI_APP_X === 'true';
    if (isX && xBaseComponents.includes(name)) {
        cssImports.push(uni_cli_shared_1.X_BASE_COMPONENTS_STYLE_PATH + name + '.css');
    }
    else if (baseComponents.includes(name)) {
        cssImports.push(uni_cli_shared_1.BASE_COMPONENTS_STYLE_PATH + name + '.css');
    }
    else {
        cssImports.push(uni_cli_shared_1.H5_COMPONENTS_STYLE_PATH + name + '.css');
    }
    const deps = componentDepsCss[name];
    deps && deps.forEach((dep) => cssImports.push(dep));
    return cssImports;
}
function addBuiltInImportDeclaration(importDeclarations, local, name) {
    resolveBuiltInCssImport(name).forEach((cssImport) => importDeclarations.push(`import '${cssImport}';`));
    return (0, uni_cli_shared_1.addImportDeclaration)(importDeclarations, local, H5_COMPONENTS_PATH, (0, shared_1.capitalize)((0, shared_1.camelize)(name)));
}

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615262, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniInjectPlugin = void 0;
const path_1 = __importDefault(require("path"));
const shared_1 = require("@vue/shared");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const apiJson = require(path_1.default.resolve(__dirname, process.env.UNI_APP_X === 'true'
    ? '../../lib/api.x.json'
    : '../../lib/api.json'));
const uniInjectPluginOptions = {
    exclude: [...uni_cli_shared_1.COMMON_EXCLUDE],
    'uni.': [
        '@dcloudio/uni-h5',
        ((method) => apiJson.includes(method)),
    ],
    // 兼容 wx 对象
    'wx.': [
        '@dcloudio/uni-h5',
        ((method) => apiJson.includes(method)),
    ],
    getApp: ['@dcloudio/uni-h5', 'getApp'],
    getCurrentPages: ['@dcloudio/uni-h5', 'getCurrentPages'],
    UniServiceJSBridge: ['@dcloudio/uni-h5', 'UniServiceJSBridge'],
    UniViewJSBridge: ['@dcloudio/uni-h5', 'UniViewJSBridge'],
};
function uniInjectPlugin() {
    let resolvedConfig;
    const apiDepsCss = (0, uni_cli_shared_1.API_DEPS_CSS)(process.env.UNI_APP_X === 'true');
    const callback = function (imports, mod) {
        const styles = mod[0] === '@dcloudio/uni-h5' &&
            apiDepsCss[mod[1]];
        if (!styles) {
            return;
        }
        styles.forEach((style) => {
            if ((0, uni_cli_shared_1.isCombineBuiltInCss)(resolvedConfig)) {
                uni_cli_shared_1.buildInCssSet.add(style);
            }
            else {
                if (!imports.has(style)) {
                    imports.set(style, `import '${style}';`);
                }
            }
        });
    };
    let injectPlugin;
    return {
        name: 'uni:h5-inject',
        apply: 'build',
        enforce: 'post',
        configResolved(config) {
            resolvedConfig = config;
            const enableTreeShaking = (0, uni_cli_shared_1.isEnableTreeShaking)((0, uni_cli_shared_1.parseManifestJsonOnce)(process.env.UNI_INPUT_DIR));
            if (!enableTreeShaking) {
                // 不启用摇树优化，移除 wx、uni 等 API 配置
                delete uniInjectPluginOptions['wx.'];
                delete uniInjectPluginOptions['uni.'];
            }
            injectPlugin = (0, uni_cli_shared_1.uniViteInjectPlugin)('uni:h5-inject', (0, shared_1.extend)(uniInjectPluginOptions, {
                callback,
            }));
        },
        transform(code, id) {
            return injectPlugin.transform.call(this, code, id);
        },
    };
}
exports.uniInjectPlugin = uniInjectPlugin;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615263, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.uniMainJsPlugin = void 0;
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const utils_1 = require("../utils");
function uniMainJsPlugin() {
    return (0, uni_cli_shared_1.defineUniMainJsPlugin)((opts) => {
        let runSSR = false;
        return {
            name: 'uni:h5-main-js',
            enforce: 'pre',
            configResolved(config) {
                runSSR =
                    (0, uni_cli_shared_1.isSsr)(config.command, config) || (0, utils_1.isSsrManifest)(config.command, config);
            },
            transform(code, id, options) {
                if (opts.filter(id)) {
                    if (!runSSR) {
                        code = code.includes('createSSRApp')
                            ? createApp(code)
                            : createLegacyApp(code);
                    }
                    else {
                        code = (0, utils_1.isSSR)(options)
                            ? createSSRServerApp(code)
                            : createSSRClientApp(code);
                    }
                    code = `import './${uni_cli_shared_1.PAGES_JSON_JS}';${code}`;
                    return {
                        code,
                        map: this.getCombinedSourcemap(),
                    };
                }
            },
        };
    });
}
exports.uniMainJsPlugin = uniMainJsPlugin;
function createApp(code) {
    return `import { plugin as __plugin } from '@dcloudio/uni-h5';${code.replace('createSSRApp', 'createVueApp as createSSRApp')};\ncreateApp().app.use(__plugin).mount("#app");`;
}
function createLegacyApp(code) {
    return `import { plugin as __plugin } from '@dcloudio/uni-h5';function createApp(rootComponent,rootProps){return createVueApp(rootComponent, rootProps).use(__plugin)};${code.replace('createApp', 'createVueApp')}`;
}
function createSSRClientApp(code) {
    return `import { plugin as __plugin } from '@dcloudio/uni-h5';import { UNI_SSR, UNI_SSR_STORE } from '@dcloudio/uni-shared';${code};\n// @ts-ignore\nconst { app: __app, store: __store } = createApp();__app.use(__plugin);__store && window[UNI_SSR] && window[UNI_SSR][UNI_SSR_STORE] && __store.replaceState(window[UNI_SSR][UNI_SSR_STORE]);__app.router.isReady().then(() => __app.mount("#app"));`;
}
function createSSRServerApp(code) {
    return code;
}

}, function(modId) { var map = {"../utils":1781530615243}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615264, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.uniManifestJsonPlugin = void 0;
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const defaultRouter = {
    mode: 'hash',
    base: '/',
    assets: 'assets',
    routerBase: '/',
};
const defaultAsync = {
    loading: 'AsyncLoading',
    error: 'AsyncError',
    delay: 200,
    timeout: 60000,
    suspensible: true,
};
function uniManifestJsonPlugin() {
    return (0, uni_cli_shared_1.defineUniManifestJsonPlugin)((opts) => {
        let resolvedConfig;
        return {
            name: 'uni:h5-manifest-json',
            enforce: 'pre',
            configResolved(config) {
                defaultRouter.assets = config.build.assetsDir;
                resolvedConfig = config;
            },
            transform(code, id) {
                if (!opts.filter(id)) {
                    return;
                }
                const manifest = (0, uni_cli_shared_1.parseJson)(code, false, id);
                const { debug } = manifest;
                const h5 = (0, uni_cli_shared_1.getPlatformManifestJson)(manifest, 'h5');
                const router = {
                    ...defaultRouter,
                    ...{ base: resolvedConfig.base },
                    ...((h5 && h5.router) || {}),
                };
                if (!router.base) {
                    router.base = '/';
                }
                /**
                 * ssr时base和访问域名不一致导致跳到错误链接，其实应该区分server和client的部署路径，后续有需求可以加上
                 */
                router.routerBase = new URL(router.base, 'http://localhost').pathname;
                const async = { ...defaultAsync, ...((h5 && h5.async) || {}) };
                const networkTimeout = (0, uni_cli_shared_1.normalizeNetworkTimeout)(manifest.networkTimeout);
                const sdkConfigs = (h5 && h5.sdkConfigs) || {};
                const tempTencentMapKey = sdkConfigs.maps &&
                    sdkConfigs.maps.tencent &&
                    sdkConfigs.maps.tencent.key;
                const tempQQMapKey = sdkConfigs.maps && sdkConfigs.maps.qqmap && sdkConfigs.maps.qqmap.key;
                const qqMapKey = tempTencentMapKey || tempQQMapKey;
                const bMapKey = sdkConfigs.maps && sdkConfigs.maps.bmap && sdkConfigs.maps.bmap.key;
                const googleMapKey = sdkConfigs.maps &&
                    sdkConfigs.maps.google &&
                    sdkConfigs.maps.google.key;
                const aMapKey = sdkConfigs.maps && sdkConfigs.maps.amap && sdkConfigs.maps.amap.key;
                const aMapSecurityJsCode = sdkConfigs.maps &&
                    sdkConfigs.maps.amap &&
                    sdkConfigs.maps.amap.securityJsCode;
                const aMapServiceHost = sdkConfigs.maps &&
                    sdkConfigs.maps.amap &&
                    sdkConfigs.maps.amap.serviceHost;
                let locale = manifest.locale;
                locale = locale && locale.toUpperCase() !== 'AUTO' ? locale : '';
                const i18nOptions = (0, uni_cli_shared_1.initI18nOptions)(process.env.UNI_PLATFORM, process.env.UNI_INPUT_DIR, false, false);
                const fallbackLocale = (i18nOptions && i18nOptions.locale) || '';
                const vueType = process.env.UNI_APP_X === 'true' ? 'uvue' : 'nvue';
                const flexDirection = (process.env.UNI_APP_X === 'true'
                    ? manifest['uni-app-x'] && manifest['uni-app-x']['flex-direction']
                    : manifest['app'] &&
                        manifest['app'].nvue &&
                        manifest['app'].nvue['flex-direction']) || 'column';
                return {
                    code: `export const appId = ${JSON.stringify(manifest.appid || '')}
  export const appName = ${JSON.stringify(manifest.name || '')}
  export const appVersion = ${JSON.stringify(manifest.versionName || '')}
  export const appVersionCode = ${JSON.stringify(manifest.versionCode || '')}

  export const debug = ${!!debug}
  export const ${vueType} = ${JSON.stringify({
                        'flex-direction': flexDirection,
                    })}
  export const networkTimeout = ${JSON.stringify(networkTimeout)}
  // h5
  export const router = ${JSON.stringify(router)}
  export const async = ${JSON.stringify(async)}
  export const qqMapKey = ${JSON.stringify(qqMapKey)}
  export const bMapKey = ${JSON.stringify(bMapKey)}
  export const googleMapKey = ${JSON.stringify(googleMapKey)}
  export const aMapKey = ${JSON.stringify(aMapKey)}
  export const aMapSecurityJsCode = ${JSON.stringify(aMapSecurityJsCode)}
  export const aMapServiceHost = ${JSON.stringify(aMapServiceHost)}
  export const sdkConfigs = ${JSON.stringify(sdkConfigs)}
  export const locale = '${locale}'
  export const fallbackLocale = '${fallbackLocale}'
  export const darkmode = ${h5.darkmode || 'false'}
  export const themeConfig = ${JSON.stringify((0, uni_cli_shared_1.normalizeThemeConfigOnce)(h5))}
  `,
                    map: { mappings: '' },
                };
            },
        };
    });
}
exports.uniManifestJsonPlugin = uniManifestJsonPlugin;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615265, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.uniPagesJsonPlugin = void 0;
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const utils_1 = require("../utils");
function uniPagesJsonPlugin() {
    return (0, uni_cli_shared_1.defineUniPagesJsonPlugin)((opts) => {
        return {
            name: 'uni:h5-pages-json',
            enforce: 'pre',
            transform(code, id, opt) {
                if (opts.filter(id)) {
                    const { resolvedConfig } = opts;
                    const ssr = (0, utils_1.isSSR)(opt);
                    if (process.env.UNI_APP_X === 'true') {
                        // 调整换行符，确保 parseTree 的loc正确
                        const jsonCode = code.replace(/\r\n/g, '\n');
                        (0, uni_cli_shared_1.checkPagesJson)((0, uni_cli_shared_1.preUVueJson)(jsonCode, 'pages.json'), process.env.UNI_INPUT_DIR);
                    }
                    return {
                        code: registerGlobalCode(resolvedConfig, ssr) +
                            generatePagesJsonCode(ssr, code, resolvedConfig),
                        map: { mappings: '' },
                    };
                }
            },
        };
    });
}
exports.uniPagesJsonPlugin = uniPagesJsonPlugin;
function generatePagesJsonCode(ssr, jsonStr, config) {
    const globalName = getGlobal(ssr);
    const pagesJson = (0, uni_cli_shared_1.normalizePagesJson)(jsonStr, process.env.UNI_PLATFORM);
    const { importLayoutComponentsCode, defineLayoutComponentsCode } = generateLayoutComponentsCode(globalName, pagesJson);
    const definePagesCode = generatePagesDefineCode(pagesJson, config);
    const uniRoutesCode = generateRoutes(globalName, pagesJson, config);
    const uniConfigCode = generateConfig(globalName, pagesJson, config);
    const cssCode = generateCssCode(config);
    const vueType = process.env.UNI_APP_X === 'true' ? 'uvue' : 'nvue';
    let workersCode = '';
    if (config.command === 'build' && process.env.UNI_APP_X === 'true') {
        const workers = (0, uni_cli_shared_1.getWorkers)();
        const workerCode = Object.keys(workers).map((key) => {
            return `import('@/${key}?worker')`;
        });
        if (workerCode.length) {
            // 仅用于激活动态chunk，运行时不能执行
            workersCode = `if(!Math){\n${workerCode.join('\n')}\n}`;
        }
    }
    return `
import { defineAsyncComponent, resolveComponent, createVNode, withCtx, openBlock, createBlock } from 'vue'
import { PageComponent, useI18n, setupWindow, setupPage } from '@dcloudio/uni-h5'
import { appId, appName, appVersion, appVersionCode, debug, networkTimeout, router, async, sdkConfigs, qqMapKey, googleMapKey, aMapKey, bMapKey, aMapSecurityJsCode, aMapServiceHost, ${vueType}, locale, fallbackLocale, darkmode, themeConfig } from './${uni_cli_shared_1.MANIFEST_JSON_JS}'
const locales = import.meta.glob('./locale/*.json', { eager: true })
${importLayoutComponentsCode}
const extend = Object.assign
${cssCode}
${uniConfigCode}
${defineLayoutComponentsCode}
${definePagesCode}
${uniRoutesCode}
${workersCode}
${config.command === 'serve' ? hmrCode : ''}
export {}
`;
}
const hmrCode = `if(import.meta.hot){
  import.meta.hot.on('invalidate', (data) => {
      import.meta.hot.invalidate()
  })
}`;
function getGlobal(ssr) {
    return ssr ? 'global' : 'window';
}
// 兼容 wx 对象
function registerGlobalCode(config, ssr) {
    const name = getGlobal(ssr);
    const enableTreeShaking = (0, uni_cli_shared_1.isEnableTreeShaking)((0, uni_cli_shared_1.parseManifestJsonOnce)(process.env.UNI_INPUT_DIR));
    if (enableTreeShaking && config.command === 'build' && !ssr) {
        // 非 SSR 的发行模式，补充全局 uni 对象
        return `import { upx2px, getApp } from '@dcloudio/uni-h5';${name}.uni = {};${name}.wx = {};${name}.rpx2px = upx2px`;
    }
    return `
import {uni,upx2px,getCurrentPages,getApp,UniServiceJSBridge,UniViewJSBridge} from '@dcloudio/uni-h5'
${name}.getApp = getApp
${name}.getCurrentPages = getCurrentPages
${name}.wx = uni
${name}.uni = uni
${name}.UniViewJSBridge = UniViewJSBridge
${name}.UniServiceJSBridge = UniServiceJSBridge
${name}.rpx2px = upx2px
${name}.__setupPage = (com)=>setupPage(com)
`;
}
function generateCssCode(config) {
    const define = config.define;
    const cssFiles = [uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'base.css'];
    if (config.isProduction) {
        cssFiles.push(uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'shadow.css');
    }
    // if (define.__UNI_FEATURE_PAGES__) {
    cssFiles.push(uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'async.css');
    // }
    if (define.__UNI_FEATURE_RESPONSIVE__) {
        cssFiles.push(uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'layout.css');
    }
    if (define.__UNI_FEATURE_NAVIGATIONBAR__) {
        cssFiles.push(uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'pageHead.css');
    }
    if (define.__UNI_FEATURE_TABBAR__) {
        cssFiles.push(uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'tabBar.css');
    }
    // x 项目直接集成 uvue.css
    if (process.env.UNI_APP_X === 'true') {
        cssFiles.push(uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'uvue.css');
    }
    else {
        if (define.__UNI_FEATURE_NVUE__) {
            cssFiles.push(uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'nvue.css');
        }
    }
    if (define.__UNI_FEATURE_PULL_DOWN_REFRESH__) {
        cssFiles.push(uni_cli_shared_1.H5_FRAMEWORK_STYLE_PATH + 'pageRefresh.css');
    }
    if (define.__UNI_FEATURE_NAVIGATIONBAR_SEARCHINPUT__) {
        cssFiles.push(uni_cli_shared_1.BASE_COMPONENTS_STYLE_PATH + 'input.css');
    }
    const enableTreeShaking = (0, uni_cli_shared_1.isEnableTreeShaking)((0, uni_cli_shared_1.parseManifestJsonOnce)(process.env.UNI_INPUT_DIR));
    if (config.command === 'serve' || !enableTreeShaking) {
        const apiDepsCss = (0, uni_cli_shared_1.API_DEPS_CSS)(process.env.UNI_APP_X === 'true');
        // 开发模式或禁用摇树优化，自动添加所有API相关css
        Object.keys(apiDepsCss).forEach((name) => {
            const styles = apiDepsCss[name];
            styles.forEach((style) => {
                if (!cssFiles.includes(style)) {
                    cssFiles.push(style);
                }
            });
        });
    }
    return cssFiles.map((file) => `import '${file}'`).join('\n');
}
function generateLayoutComponentsCode(globalName, pagesJson) {
    const windowNames = {
        topWindow: -1,
        leftWindow: -2,
        rightWindow: -3,
    };
    let importLayoutComponentsCode = '';
    let defineLayoutComponentsCode = `${globalName}.__uniLayout = ${globalName}.__uniLayout || {}\n`;
    Object.keys(windowNames).forEach((name) => {
        const windowConfig = pagesJson[name];
        if (windowConfig && windowConfig.path) {
            importLayoutComponentsCode += `const ${name} = defineAsyncComponent(()=>import('./${windowConfig.path}').then(com=>setupWindow(com.default || com,${windowNames[name]})))\n`;
            defineLayoutComponentsCode += `${globalName}.__uniConfig.${name}.component = ${name}\n`;
        }
    });
    return {
        importLayoutComponentsCode,
        defineLayoutComponentsCode,
    };
}
function generatePageDefineCode(pageOptions) {
    let pagePathWithExtname = (0, uni_cli_shared_1.normalizePagePath)(pageOptions.path, 'h5');
    if (!pagePathWithExtname) {
        // 不存在时，仍引用，此时编译会报错文件不存在
        pagePathWithExtname = pageOptions.path + '.vue';
    }
    const pageIdent = (0, uni_cli_shared_1.normalizeIdentifier)(pageOptions.path);
    return `const ${pageIdent}Loader = ()=>import('./${pagePathWithExtname}').then(com => setupPage(com.default || com, '${pagePathWithExtname}'))
const ${pageIdent} = defineAsyncComponent(extend({loader:${pageIdent}Loader},AsyncComponentOptions))`;
}
function generatePagesDefineCode(pagesJson, _config) {
    const { pages } = pagesJson;
    return (`const AsyncComponentOptions = {
      delay: async.delay,
      timeout: async.timeout,
      suspensible: async.suspensible
    }
    if(async.loading){
      AsyncComponentOptions.loadingComponent = {
        name:'SystemAsyncLoading',
        render(){
          return createVNode(resolveComponent(async.loading))
        }
      }
    }
    if(async.error){
      AsyncComponentOptions.errorComponent = {
        name:'SystemAsyncError',
        props:['error'],
        render(){
          return createVNode(resolveComponent(async.error), { error: this.error })
        }
      }
    }
  ` + pages.map((pageOptions) => generatePageDefineCode(pageOptions)).join('\n'));
}
function generatePageRoute({ path, meta }, _config) {
    const { isEntry } = meta;
    const alias = isEntry ? `\n  alias:'/${path}',` : '';
    // 目前单页面未处理 query=>props
    const queryCode = process.env.UNI_APP_X === 'true'
        ? 'app && app.vm && app.vm.$route && app.vm.$route.query || {};'
        : 'app && app.$route && app.$route.query || {};';
    return `{
  path:'/${isEntry ? '' : path}',${alias}
  component:{setup(){ const app = getApp(); const query = ${queryCode} return ()=>renderPage(${(0, uni_cli_shared_1.normalizeIdentifier)(path)},query)}},
  loader: ${(0, uni_cli_shared_1.normalizeIdentifier)(path)}Loader,
  meta: ${JSON.stringify(meta)}
}`;
}
function generatePagesRoute(pagesRouteOptions, config) {
    return pagesRouteOptions.map((pageOptions) => generatePageRoute(pageOptions, config));
}
function generateRoutes(globalName, pagesJson, config) {
    return `
function renderPage(component,props){
  return (openBlock(), createBlock(PageComponent, null, {page: withCtx(() => [createVNode(component, extend({},props,{ref: "page"}), null, 512 /* NEED_PATCH */)]), _: 1 /* STABLE */}))
}
${globalName}.__uniRoutes=[${[
        ...generatePagesRoute((0, uni_cli_shared_1.normalizePagesRoute)(pagesJson), config),
    ].join(',')}].map(uniRoute=>(uniRoute.meta.route = (uniRoute.alias || uniRoute.path).slice(1),uniRoute))`;
}
function generateConfig(globalName, pagesJson, config) {
    const isX = process.env.UNI_APP_X === 'true';
    delete pagesJson.pages;
    delete pagesJson.subPackages;
    delete pagesJson.subpackages;
    pagesJson.compilerVersion = process.env.UNI_COMPILER_VERSION;
    const vueType = isX ? 'uvue' : 'nvue';
    let tabBarCode = '';
    if (isX) {
        const tabBar = pagesJson.tabBar;
        delete pagesJson.tabBar;
        tabBarCode = `${globalName}.__uniConfig.getTabBarConfig = () => {return ${tabBar ? JSON.stringify(tabBar) : 'undefined'}};
    ${globalName}.__uniConfig.tabBar = __uniConfig.getTabBarConfig();`;
    }
    return `${isX ? `${globalName}.__uniX = true` : ''}
  ${globalName}.__uniConfig=extend(${JSON.stringify(pagesJson)},{
  appId,
  appName,
  appVersion,
  appVersionCode,
  async,
  debug,
  networkTimeout,
  sdkConfigs,
  qqMapKey,
  bMapKey,
  googleMapKey,
  aMapKey,
  aMapSecurityJsCode,
  aMapServiceHost,
  ${vueType},
  locale,
  fallbackLocale,
  locales:Object.keys(locales).reduce((res,name)=>{const locale=name.replace(/\\.\\/locale\\/(uni-app.)?(.*).json/,'$2');extend(res[locale]||(res[locale]={}),locales[name].default);return res},{}),
  router,
  darkmode,
  themeConfig,
})
${tabBarCode}
`;
}

}, function(modId) { var map = {"../utils":1781530615243}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615266, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniPostVuePlugin = void 0;
const path_1 = __importDefault(require("path"));
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const WXS_RE = /vue&type=(wxs|renderjs)/;
function uniPostVuePlugin() {
    return {
        name: 'uni:post-vue',
        apply: 'serve',
        enforce: 'post',
        async transform(code, id) {
            const { filename, query } = (0, uni_cli_shared_1.parseVueRequest)(id);
            if (query.vue) {
                return;
            }
            if (!uni_cli_shared_1.EXTNAME_VUE.includes(path_1.default.extname(filename))) {
                return;
            }
            if (!WXS_RE.test(code)) {
                return;
            }
            const hmrId = parseHmrId(code);
            if (!hmrId) {
                return;
            }
            // TODO 内部解决 @vitejs/plugin-vue 自定义块外链热刷的问题
            // https://github.com/vitejs/vite/blob/main/packages/plugin-vue/src/main.ts#L387
            // 没有增加 src=descriptor.id
            // 包含外链 wxs,renderjs
            code = code.replace(/vue&type=(wxs|renderjs)&index=([0-9]+)&src&/gi, (_, type, index) => {
                return `vue&type=${type}&index=${index}&src=${hmrId}&`;
            });
            return {
                code: code, // 暂不提供sourcemap,意义不大
                map: null,
            };
        },
    };
}
exports.uniPostVuePlugin = uniPostVuePlugin;
function parseHmrId(code) {
    const matches = code.match(/_sfc_main.__hmrId = "(.*)"/);
    return matches && matches[1];
}

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615267, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniRenderjsPlugin = void 0;
const debug_1 = __importDefault(require("debug"));
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const debugRenderjs = (0, debug_1.default)('uni:h5-renderjs');
function uniRenderjsPlugin() {
    return {
        name: 'uni:h5-renderjs',
        transform(code, id) {
            const { type, name } = (0, uni_cli_shared_1.parseRenderjs)(id);
            if (!type) {
                return;
            }
            debugRenderjs(id);
            if (!name) {
                this.error((0, uni_cli_shared_1.missingModuleName)(type, code));
            }
            return {
                code: `${require('@vue/compiler-sfc').rewriteDefault(code.replace(/module\.exports\s*=/, 'export default '), '_sfc_' + type)}
${type === 'renderjs' ? genRenderjsCode(name) : genWxsCode(name)}`,
                map: { mappings: '' },
            };
        },
    };
}
exports.uniRenderjsPlugin = uniRenderjsPlugin;
function genRenderjsCode(name) {
    return `export default Comp => {
  if(!Comp.$renderjs){Comp.$renderjs = []}
  Comp.$renderjs.push('${name}')
  if(!Comp.mixins){Comp.mixins = []}
  Comp.mixins.push({beforeCreate(){ this['${name}'] = this },mounted(){ this.$ownerInstance = this.$gcd(this, true) }})
  Comp.mixins.push(_sfc_renderjs)
}`;
}
function genWxsCode(name) {
    return `export default Comp => {
  if(!Comp.$wxs){Comp.$wxs = []} 
  Comp.$wxs.push('${name}')
  if(!Comp.mixins){Comp.mixins = []}
  Comp.mixins.push({beforeCreate(){ this['${name}'] = _sfc_wxs }})
}`;
}

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615268, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniResolveIdPlugin = void 0;
const path_1 = __importDefault(require("path"));
const debug_1 = __importDefault(require("debug"));
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const utils_1 = require("../utils");
const debugResolve = (0, debug_1.default)('uni:resolve');
function uniResolveIdPlugin() {
    const resolveCache = {};
    const mainPath = (0, uni_cli_shared_1.resolveMainPathOnce)(process.env.UNI_INPUT_DIR);
    return {
        name: 'uni:h5-resolve-id',
        enforce: 'pre',
        configResolved(config) {
            resolveCache[utils_1.ownerModuleName] = (0, uni_cli_shared_1.resolveBuiltIn)(path_1.default.join(utils_1.ownerModuleName, (0, utils_1.resolveFrameworkDistDir)() + '/uni-h5.es.js'));
            resolveCache['@dcloudio/uni-h5-vue'] = (0, uni_cli_shared_1.resolveBuiltIn)(path_1.default.join('@dcloudio/uni-h5-vue', (0, utils_1.resolveVueDistDir)() +
                `/vue.runtime.${process.env.VITEST ? 'cjs' : 'esm'}.js`));
        },
        resolveId(id, importer, options) {
            if (id === '/main' && importer && importer.endsWith('index.html')) {
                return mainPath;
            }
            if (id === 'vue') {
                id = '@dcloudio/uni-h5-vue';
            }
            if ((0, utils_1.isSSR)(options)) {
                if (id === '@dcloudio/uni-h5-vue') {
                    return (0, uni_cli_shared_1.resolveBuiltIn)(path_1.default.join('@dcloudio/uni-h5-vue', (0, utils_1.resolveVueDistDir)() + `/vue.runtime.cjs.js`));
                }
            }
            const cache = resolveCache[id];
            if (cache) {
                debugResolve('cache', id, cache);
                return cache;
            }
            if (id.startsWith('@dcloudio/uni-h5/style')) {
                return (resolveCache[id] = (0, uni_cli_shared_1.resolveBuiltIn)(id));
            }
            if (id.startsWith('@dcloudio/uni-components/style')) {
                return (resolveCache[id] = (0, uni_cli_shared_1.resolveBuiltIn)(id));
            }
        },
    };
}
exports.uniResolveIdPlugin = uniResolveIdPlugin;

}, function(modId) { var map = {"../utils":1781530615243}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615269, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniSetupPlugin = void 0;
const debug_1 = __importDefault(require("debug"));
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const debugSetup = (0, debug_1.default)('uni:setup');
function uniSetupPlugin() {
    let appVuePath;
    let resolvedConfig;
    return {
        name: 'uni:setup',
        configResolved(config) {
            resolvedConfig = config;
            appVuePath = (0, uni_cli_shared_1.resolveAppVue)(process.env.UNI_INPUT_DIR);
        },
        transform(code, id) {
            const { filename, query } = (0, uni_cli_shared_1.parseVueRequest)(id);
            if (filename === appVuePath && !query.vue) {
                debugSetup(filename);
                return {
                    code: code +
                        `;import { setupApp } from '@dcloudio/uni-h5';setupApp(_sfc_main);`,
                    map: (0, uni_cli_shared_1.withSourcemap)(resolvedConfig)
                        ? this.getCombinedSourcemap()
                        : null,
                };
            }
        },
    };
}
exports.uniSetupPlugin = uniSetupPlugin;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615270, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniSSRPlugin = void 0;
const path_1 = __importDefault(require("path"));
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const utils_1 = require("../utils");
const ENTRY_SERVER_JS = 'entry-server.js';
function uniSSRPlugin() {
    let entryServerJs;
    let resolvedConfig;
    const entryServerJsCode = (0, utils_1.generateSsrEntryServerCode)();
    return {
        name: 'uni:h5-ssr',
        config(userConfig, env) {
            if ((0, uni_cli_shared_1.isSsr)(env.command, userConfig)) {
                (0, utils_1.initSsrAliasOnce)();
                (0, utils_1.rewriteSsrVue)();
                (0, utils_1.rewriteSsrNativeTag)();
                (0, utils_1.rewriteSsrRenderStyle)(process.env.UNI_INPUT_DIR);
                const alias = [
                    {
                        find: 'vue/server-renderer',
                        replacement: path_1.default.dirname((0, uni_cli_shared_1.resolveBuiltIn)('@vue/server-renderer')),
                    },
                ];
                try {
                    const replacement = path_1.default.dirname((0, uni_cli_shared_1.resolveBuiltIn)('vuex/package.json'));
                    alias.push({
                        find: 'vuex',
                        replacement,
                    });
                }
                catch (error) { }
                return {
                    resolve: {
                        alias,
                    },
                };
            }
        },
        configResolved(config) {
            resolvedConfig = config;
            entryServerJs = path_1.default.join(process.env.UNI_INPUT_DIR, ENTRY_SERVER_JS);
            if ((0, uni_cli_shared_1.isSsr)(resolvedConfig.command, resolvedConfig)) {
                (0, utils_1.initSsrDefine)(resolvedConfig);
            }
        },
        resolveId(id) {
            if (id.endsWith(ENTRY_SERVER_JS)) {
                return entryServerJs;
            }
        },
        load(id) {
            if (id.endsWith(ENTRY_SERVER_JS)) {
                return entryServerJsCode;
            }
        },
        generateBundle(_options, bundle) {
            const chunk = bundle['entry-server.js'];
            if (chunk) {
                chunk.code =
                    (0, utils_1.generateSsrDefineCode)(resolvedConfig, (0, uni_cli_shared_1.parseRpx2UnitOnce)(process.env.UNI_INPUT_DIR, process.env.UNI_PLATFORM)) +
                        '\n' +
                        chunk.code;
            }
        },
    };
}
exports.uniSSRPlugin = uniSSRPlugin;

}, function(modId) { var map = {"../utils":1781530615243}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615272, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.uniCustomElementPlugin = void 0;
const shared_1 = require("@vue/shared");
const uni_shared_1 = require("@dcloudio/uni-shared");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const H5_COMPONENTS_PATH = '@dcloudio/uni-h5';
function uniCustomElementPlugin() {
    return {
        name: 'uni:h5-custom-element',
        transform(code, id) {
            if (!(0, uni_cli_shared_1.isVueSfcFile)(id)) {
                return;
            }
            if (!code.includes('$UniCustomElement$')) {
                return;
            }
            const importSpecifiers = [];
            code = code.replace(/['|"]\$UniCustomElement\$([\w|-]+)['|"]/g, (_, name) => {
                if (!uni_shared_1.UVUE_WEB_BUILT_IN_CUSTOM_ELEMENTS.includes(name)) {
                    return _;
                }
                const elementName = (0, shared_1.capitalize)((0, shared_1.camelize)(name));
                const localName = '_' + elementName + 'Element';
                importSpecifiers.push(`${elementName} as ${localName}`);
                return localName;
            });
            if (importSpecifiers.length) {
                code =
                    `import {${importSpecifiers.join(',')}} from "${H5_COMPONENTS_PATH}";` + code;
            }
            return {
                code,
                map: null,
            };
        },
    };
}
exports.uniCustomElementPlugin = uniCustomElementPlugin;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615273, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.uniApiPlugin = void 0;
const utils_1 = require("../utils");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
let enableFacialRecognition = false;
function isEnableFacialRecognition() {
    return enableFacialRecognition;
}
function setEnableFacialRecognition(enable) {
    enableFacialRecognition = enable;
}
function checkGetMetaInfo(code) {
    return code.includes('window.getMetaInfo');
}
function checkFacialRecognition(code) {
    return code.includes('getFacialRecognitionMetaInfo');
}
function uniApiPlugin() {
    let viteServer = undefined;
    const inputDir = (0, uni_cli_shared_1.normalizePath)(process.env.UNI_INPUT_DIR);
    return {
        name: 'uni:h5-api',
        enforce: 'pre',
        configureServer(server) {
            viteServer = server;
        },
        transform(code, id) {
            if (!viteServer)
                return;
            // 通过transform阶段识别，仅判断inputDir内部的文件，避免框架文件影响
            if (!isEnableFacialRecognition() &&
                (0, uni_cli_shared_1.normalizePath)(id).startsWith(inputDir)) {
                if (checkFacialRecognition(code)) {
                    setEnableFacialRecognition(true);
                    // 开发模式触发重新刷新
                    viteServer.hot.send({
                        type: 'full-reload',
                        path: '*',
                    });
                }
            }
        },
        generateBundle(_options, bundle) {
            if (viteServer)
                return;
            if (!isEnableFacialRecognition()) {
                const filesNames = Object.keys(bundle);
                for (const fileName of filesNames) {
                    const chunk = bundle[fileName];
                    if (chunk &&
                        chunk.type === 'chunk' &&
                        chunk.code &&
                        !isEnableFacialRecognition()) {
                        setEnableFacialRecognition(checkFacialRecognition(chunk.code) || checkGetMetaInfo(chunk.code));
                    }
                }
            }
        },
        transformIndexHtml: {
            order: 'post',
            handler() {
                if (!isEnableFacialRecognition()) {
                    return;
                }
                return [
                    {
                        tag: 'script',
                        attrs: {
                            src: utils_1.AliYunCloudAuthWebSDK,
                        },
                        injectTo: 'head',
                    },
                ];
            },
        },
    };
}
exports.uniApiPlugin = uniApiPlugin;

}, function(modId) { var map = {"../utils":1781530615243}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615274, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.rewriteCompilerSfcParse = void 0;
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
/**
 * 重写 @vue/compiler-sfc 的 parse 函数
 * web 平台下，如果 SFC 没有 style，注入一个默认的空 style
 */
function rewriteCompilerSfcParse() {
    const compilerSfc = require((0, uni_cli_shared_1.resolveBuiltIn)('@vue/compiler-sfc'));
    const { parse } = compilerSfc;
    compilerSfc.parse = (source, options) => {
        const result = parse(source, options);
        // 如果没有 style，注入一个默认的空 style
        if (result.descriptor.styles.length === 0) {
            result.descriptor.styles = [(0, uni_cli_shared_1.createDefaultSFCStyleBlock)(source)];
        }
        return result;
    };
}
exports.rewriteCompilerSfcParse = rewriteCompilerSfcParse;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
return __REQUIRE__(1781530615240);
})()
//miniprogram-npm-outsideDeps=["path","@vitejs/plugin-basic-ssl","@dcloudio/uni-cli-shared","@vue/compiler-dom","./plugins/sourcemap","./transformIndexHtml","debug","@vue/shared","./ssr","url","fs","@rollup/pluginutils","mime/lite","@dcloudio/uni-shared","./transforms/transformAttributePart","@vue/compiler-core","os","vite","fs-extra","@vue/compiler-sfc"]
//# sourceMappingURL=index.js.map