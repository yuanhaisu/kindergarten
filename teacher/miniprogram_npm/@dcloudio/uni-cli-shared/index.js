module.exports = (function() {
var __MODS__ = {};
var __DEFINE__ = function(modId, func, req) { var m = { exports: {}, _tempexports: {} }; __MODS__[modId] = { status: 0, func: func, req: req, m: m }; };
var __REQUIRE__ = function(modId, source) { if(!__MODS__[modId]) return require(source); if(!__MODS__[modId].status) { var m = __MODS__[modId].m; m._exports = m._tempexports; var desp = Object.getOwnPropertyDescriptor(m, "exports"); if (desp && desp.configurable) Object.defineProperty(m, "exports", { set: function (val) { if(typeof val === "object" && val !== m._exports) { m._exports.__proto__ = val.__proto__; Object.keys(val).forEach(function (k) { m._exports[k] = val[k]; }); } m._tempexports = val }, get: function () { return m._tempexports; } }); __MODS__[modId].status = 1; __MODS__[modId].func(__MODS__[modId].req, m, m.exports); } return __MODS__[modId].m.exports; };
var __REQUIRE_WILDCARD__ = function(obj) { if(obj && obj.__esModule) { return obj; } else { var newObj = {}; if(obj != null) { for(var k in obj) { if (Object.prototype.hasOwnProperty.call(obj, k)) newObj[k] = obj[k]; } } newObj.default = obj; return newObj; } };
var __REQUIRE_DEFAULT__ = function(obj) { return obj && obj.__esModule ? obj.default : obj; };
__DEFINE__(1781362618792, function(require, module, exports) {

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
exports.checkUpdate = exports.M = exports.resolveEncryptUniModule = exports.parseUniModulesArtifacts = exports.getUniModulesEncryptType = exports.formatExtApiProviderName = exports.getUniExtApiProviderRegisters = exports.parseInjects = exports.parseUniExtApis = exports.parseUniExtApi = void 0;
__exportStar(require("./fs"), exports);
__exportStar(require("./mp"), exports);
__exportStar(require("./url"), exports);
__exportStar(require("./env"), exports);
__exportStar(require("./hbx"), exports);
__exportStar(require("./ssr"), exports);
__exportStar(require("./vue"), exports);
__exportStar(require("./uts"), exports);
__exportStar(require("./x"), exports);
__exportStar(require("./logs"), exports);
__exportStar(require("./dom2"), exports);
__exportStar(require("./i18n"), exports);
__exportStar(require("./deps"), exports);
__exportStar(require("./json"), exports);
__exportStar(require("./vite"), exports);
__exportStar(require("./utils"), exports);
__exportStar(require("./easycom"), exports);
__exportStar(require("./constants"), exports);
__exportStar(require("./preprocess"), exports);
__exportStar(require("./postcss"), exports);
__exportStar(require("./filter"), exports);
__exportStar(require("./esbuild"), exports);
__exportStar(require("./resolve"), exports);
__exportStar(require("./scripts"), exports);
__exportStar(require("./platform"), exports);
__exportStar(require("./utsUtils"), exports);
__exportStar(require("./workers"), exports);
var uni_modules_1 = require("./uni_modules");
Object.defineProperty(exports, "parseUniExtApi", { enumerable: true, get: function () { return uni_modules_1.parseUniExtApi; } });
Object.defineProperty(exports, "parseUniExtApis", { enumerable: true, get: function () { return uni_modules_1.parseUniExtApis; } });
Object.defineProperty(exports, "parseInjects", { enumerable: true, get: function () { return uni_modules_1.parseInjects; } });
Object.defineProperty(exports, "getUniExtApiProviderRegisters", { enumerable: true, get: function () { return uni_modules_1.getUniExtApiProviderRegisters; } });
Object.defineProperty(exports, "formatExtApiProviderName", { enumerable: true, get: function () { return uni_modules_1.formatExtApiProviderName; } });
var uni_modules_cloud_1 = require("./uni_modules.cloud");
Object.defineProperty(exports, "getUniModulesEncryptType", { enumerable: true, get: function () { return uni_modules_cloud_1.getUniModulesEncryptType; } });
Object.defineProperty(exports, "parseUniModulesArtifacts", { enumerable: true, get: function () { return uni_modules_cloud_1.parseUniModulesArtifacts; } });
Object.defineProperty(exports, "resolveEncryptUniModule", { enumerable: true, get: function () { return uni_modules_cloud_1.resolveEncryptUniModule; } });
var messages_1 = require("./messages");
Object.defineProperty(exports, "M", { enumerable: true, get: function () { return messages_1.M; } });
__exportStar(require("./exports"), exports);
var checkUpdate_1 = require("./checkUpdate");
Object.defineProperty(exports, "checkUpdate", { enumerable: true, get: function () { return checkUpdate_1.checkUpdate; } });

}, function(modId) {var map = {"./fs":1781362618793,"./mp":1781362618794,"./url":1781362618886,"./env":1781362618887,"./hbx":1781362618896,"./ssr":1781362618992,"./vue":1781362618917,"./uts":1781362618901,"./x":1781362618905,"./dom2":1781362618908,"./i18n":1781362618843,"./deps":1781362618994,"./json":1781362618806,"./vite":1781362618943,"./easycom":1781362618930,"./constants":1781362618871,"./postcss":1781362618997,"./filter":1781362619001,"./esbuild":1781362619002,"./resolve":1781362618891,"./platform":1781362619004,"./utsUtils":1781362618929,"./workers":1781362618899,"./messages":1781362618844,"./exports":1781362619007,"./checkUpdate":1781362619008}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618793, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.emptyDir = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
function emptyDir(dir, skip = []) {
    try {
        for (const file of fs_1.default.readdirSync(dir)) {
            if (skip.includes(file)) {
                continue;
            }
            // node >= 14.14.0
            fs_1.default.rmSync(path_1.default.resolve(dir, file), { recursive: true, force: true });
        }
    }
    catch (e) { }
}
exports.emptyDir = emptyDir;

}, function(modId) { var map = {"fs":1781362618793}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618794, function(require, module, exports) {

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
exports.createDefaultSFCStyleBlock = exports.clearPageExternalClasses = exports.addPageExternalClasses = exports.updatePageExternalClasses = exports.findPageExternalClasses = exports.findMiniProgramComponentStyleIsolation = exports.updateMiniProgramComponentStyleIsolation = exports.updateMiniProgramComponentExternalClasses = exports.findMiniProgramComponentExternalClasses = exports.parseStyleIsolation = exports.parseExternalClasses = exports.hasExternalClasses = exports.getGlobalComponentSource = exports.updateMiniProgramComponentsByTemplateFilename = exports.updateMiniProgramComponentsByScriptFilename = exports.updateMiniProgramComponentsByMainFilename = exports.updateMiniProgramGlobalComponents = exports.transformDynamicImports = exports.parseTemplateDescriptor = exports.parseScriptDescriptor = exports.parseMainDescriptor = exports.createCopyPluginTarget = exports.copyMiniProgramThemeJson = exports.copyMiniProgramPluginJson = exports.HTML_TO_MINI_PROGRAM_TAGS = void 0;
__exportStar(require("./ast"), exports);
__exportStar(require("./wxs"), exports);
__exportStar(require("./nvue"), exports);
__exportStar(require("./event"), exports);
__exportStar(require("./style"), exports);
__exportStar(require("./assets"), exports);
__exportStar(require("./template"), exports);
__exportStar(require("./constants"), exports);
var tags_1 = require("./tags");
Object.defineProperty(exports, "HTML_TO_MINI_PROGRAM_TAGS", { enumerable: true, get: function () { return tags_1.HTML_TO_MINI_PROGRAM_TAGS; } });
var plugin_1 = require("./plugin");
Object.defineProperty(exports, "copyMiniProgramPluginJson", { enumerable: true, get: function () { return plugin_1.copyMiniProgramPluginJson; } });
Object.defineProperty(exports, "copyMiniProgramThemeJson", { enumerable: true, get: function () { return plugin_1.copyMiniProgramThemeJson; } });
Object.defineProperty(exports, "createCopyPluginTarget", { enumerable: true, get: function () { return plugin_1.createCopyPluginTarget; } });
var usingComponents_1 = require("./usingComponents");
Object.defineProperty(exports, "parseMainDescriptor", { enumerable: true, get: function () { return usingComponents_1.parseMainDescriptor; } });
Object.defineProperty(exports, "parseScriptDescriptor", { enumerable: true, get: function () { return usingComponents_1.parseScriptDescriptor; } });
Object.defineProperty(exports, "parseTemplateDescriptor", { enumerable: true, get: function () { return usingComponents_1.parseTemplateDescriptor; } });
Object.defineProperty(exports, "transformDynamicImports", { enumerable: true, get: function () { return usingComponents_1.transformDynamicImports; } });
Object.defineProperty(exports, "updateMiniProgramGlobalComponents", { enumerable: true, get: function () { return usingComponents_1.updateMiniProgramGlobalComponents; } });
Object.defineProperty(exports, "updateMiniProgramComponentsByMainFilename", { enumerable: true, get: function () { return usingComponents_1.updateMiniProgramComponentsByMainFilename; } });
Object.defineProperty(exports, "updateMiniProgramComponentsByScriptFilename", { enumerable: true, get: function () { return usingComponents_1.updateMiniProgramComponentsByScriptFilename; } });
Object.defineProperty(exports, "updateMiniProgramComponentsByTemplateFilename", { enumerable: true, get: function () { return usingComponents_1.updateMiniProgramComponentsByTemplateFilename; } });
Object.defineProperty(exports, "getGlobalComponentSource", { enumerable: true, get: function () { return usingComponents_1.getGlobalComponentSource; } });
var externalClasses_1 = require("./externalClasses");
Object.defineProperty(exports, "hasExternalClasses", { enumerable: true, get: function () { return externalClasses_1.hasExternalClasses; } });
Object.defineProperty(exports, "parseExternalClasses", { enumerable: true, get: function () { return externalClasses_1.parseExternalClasses; } });
Object.defineProperty(exports, "parseStyleIsolation", { enumerable: true, get: function () { return externalClasses_1.parseStyleIsolation; } });
Object.defineProperty(exports, "findMiniProgramComponentExternalClasses", { enumerable: true, get: function () { return externalClasses_1.findMiniProgramComponentExternalClasses; } });
Object.defineProperty(exports, "updateMiniProgramComponentExternalClasses", { enumerable: true, get: function () { return externalClasses_1.updateMiniProgramComponentExternalClasses; } });
Object.defineProperty(exports, "updateMiniProgramComponentStyleIsolation", { enumerable: true, get: function () { return externalClasses_1.updateMiniProgramComponentStyleIsolation; } });
Object.defineProperty(exports, "findMiniProgramComponentStyleIsolation", { enumerable: true, get: function () { return externalClasses_1.findMiniProgramComponentStyleIsolation; } });
Object.defineProperty(exports, "findPageExternalClasses", { enumerable: true, get: function () { return externalClasses_1.findPageExternalClasses; } });
Object.defineProperty(exports, "updatePageExternalClasses", { enumerable: true, get: function () { return externalClasses_1.updatePageExternalClasses; } });
Object.defineProperty(exports, "addPageExternalClasses", { enumerable: true, get: function () { return externalClasses_1.addPageExternalClasses; } });
Object.defineProperty(exports, "clearPageExternalClasses", { enumerable: true, get: function () { return externalClasses_1.clearPageExternalClasses; } });
Object.defineProperty(exports, "createDefaultSFCStyleBlock", { enumerable: true, get: function () { return externalClasses_1.createDefaultSFCStyleBlock; } });

}, function(modId) { var map = {"./ast":1781362618795,"./wxs":1781362618797,"./nvue":1781362618798,"./event":1781362618800,"./style":1781362618801,"./assets":1781362618804,"./template":1781362618865,"./constants":1781362618867,"./tags":1781362618802,"./plugin":1781362618868,"./usingComponents":1781362618870,"./externalClasses":1781362618884}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618795, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.parseProgram = void 0;
const parser_1 = require("@babel/parser");
const utils_1 = require("../utils");
function parseProgram(code, importer, { babelParserPlugins }) {
    return (0, parser_1.parse)(code, {
        plugins: (0, utils_1.normalizeParsePlugins)(importer, babelParserPlugins),
        sourceType: 'module',
    }).program;
}
exports.parseProgram = parseProgram;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618797, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.genWxsCallMethodsCode = exports.parseWxsCallMethods = void 0;
const types_1 = require("@babel/types");
const estree_walker_1 = require("estree-walker");
const ast_1 = require("./ast");
function parseWxsCallMethods(code) {
    if (!code.includes('callMethod')) {
        return [];
    }
    const ast = (0, ast_1.parseProgram)(code, '', {});
    const wxsCallMethods = new Set();
    estree_walker_1.walk(ast, {
        enter(child) {
            if (!(0, types_1.isCallExpression)(child)) {
                return;
            }
            const { callee } = child;
            // .callMethod
            if (!(0, types_1.isMemberExpression)(callee) ||
                !(0, types_1.isIdentifier)(callee.property) ||
                callee.property.name !== 'callMethod') {
                return;
            }
            // .callMethod('test',...)
            const args = child.arguments;
            if (!args.length) {
                return;
            }
            const [name] = args;
            if (!(0, types_1.isStringLiteral)(name)) {
                return;
            }
            wxsCallMethods.add(name.value);
        },
    });
    return [...wxsCallMethods];
}
exports.parseWxsCallMethods = parseWxsCallMethods;
function genWxsCallMethodsCode(code) {
    const wxsCallMethods = parseWxsCallMethods(code);
    if (!wxsCallMethods.length) {
        return `export default {}`;
    }
    return `export default (Component) => {
  if(!Component.wxsCallMethods){
    Component.wxsCallMethods = []
  }
  Component.wxsCallMethods.push(${wxsCallMethods
        .map((m) => `'${m}'`)
        .join(', ')})
}
`;
}
exports.genWxsCallMethodsCode = genWxsCallMethodsCode;

}, function(modId) { var map = {"./ast":1781362618795}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618798, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.genNVueCssCode = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const nvue_1 = require("../json/app/manifest/nvue");
function genNVueCssCode(manifestJson) {
    let nvueCssCode = fs_1.default.readFileSync(path_1.default.resolve(__dirname, '../../lib/nvue.css'), 'utf8');
    const flexDirection = (0, nvue_1.getNVueFlexDirection)(manifestJson);
    if (flexDirection !== 'column') {
        nvueCssCode = nvueCssCode.replace('column', flexDirection);
    }
    return nvueCssCode;
}
exports.genNVueCssCode = genNVueCssCode;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618800, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.formatMiniProgramEvent = void 0;
const uni_shared_1 = require("@dcloudio/uni-shared");
function formatMiniProgramEvent(eventName, { isCatch, isCapture, isComponent, }) {
    if (isComponent) {
        // 自定义组件的自定义事件需要格式化，因为 triggerEvent 时也会格式化
        eventName = (0, uni_shared_1.customizeEvent)(eventName);
    }
    if (!isComponent && eventName === 'click') {
        eventName = 'tap';
    }
    let eventType = 'bind';
    if (isCatch) {
        eventType = 'catch';
    }
    if (isCapture) {
        return `capture-${eventType}:${eventName}`;
    }
    // bind:foo-bar
    return eventType + (isSimpleExpr(eventName) ? '' : ':') + eventName;
}
exports.formatMiniProgramEvent = formatMiniProgramEvent;
function isSimpleExpr(name) {
    if (name.startsWith('_')) {
        return false;
    }
    if (name.indexOf('-') > -1) {
        return false;
    }
    if (name.indexOf(':') > -1) {
        return false;
    }
    return true;
}

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618801, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.transformPartSelector = exports.transformScopedCss = void 0;
const tags_1 = require("./tags");
const logs_1 = require("../logs");
const uni_shared_1 = require("@dcloudio/uni-shared");
function transformScopedCss(cssCode) {
    checkHtmlTagSelector(cssCode);
    return cssCode.replace(/\[(data-v-[a-f0-9]{8})\]/gi, (_, scopedId) => {
        return '.' + scopedId;
    });
}
exports.transformScopedCss = transformScopedCss;
function transformPartSelector(cssCode) {
    return cssCode.replace(/::part\(([^)]+)\)/gi, (_, partName) => {
        return ' .' + (0, uni_shared_1.getPartClass)(partName.trim());
    });
}
exports.transformPartSelector = transformPartSelector;
function checkHtmlTagSelector(cssCode) {
    for (const tag in tags_1.HTML_TO_MINI_PROGRAM_TAGS) {
        if (new RegExp(`( |\n|\t|,|})${tag}( *)(,|{)`, 'g').test(cssCode)) {
            (0, logs_1.output)('warn', `小程序端 style 暂不支持 ${tag} 标签选择器，推荐使用 class 选择器，详情参考：https://uniapp.dcloud.net.cn/tutorial/migration-to-vue3.html#style`);
            break;
        }
    }
}

}, function(modId) { var map = {"./tags":1781362618802}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618802, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.HTML_TO_MINI_PROGRAM_TAGS = void 0;
exports.HTML_TO_MINI_PROGRAM_TAGS = {
    br: 'view',
    hr: 'view',
    p: 'view',
    h1: 'view',
    h2: 'view',
    h3: 'view',
    h4: 'view',
    h5: 'view',
    h6: 'view',
    abbr: 'view',
    address: 'view',
    b: 'view',
    bdi: 'view',
    bdo: 'view',
    blockquote: 'view',
    cite: 'view',
    code: 'view',
    del: 'view',
    ins: 'view',
    dfn: 'view',
    em: 'view',
    strong: 'view',
    samp: 'view',
    kbd: 'view',
    var: 'view',
    i: 'view',
    mark: 'view',
    pre: 'view',
    q: 'view',
    ruby: 'view',
    rp: 'view',
    rt: 'view',
    s: 'view',
    small: 'view',
    sub: 'view',
    sup: 'view',
    time: 'view',
    u: 'view',
    wbr: 'view',
    // 表单元素
    // form: 'form',
    // input: 'input',
    // textarea: 'textarea',
    // button: 'button',
    select: 'picker',
    option: 'view',
    optgroup: 'view',
    fieldset: 'view',
    datalist: 'picker',
    legend: 'view',
    output: 'view',
    // 框架
    iframe: 'view',
    // 图像
    img: 'image',
    // canvas: 'canvas',
    figure: 'view',
    figcaption: 'view',
    // 音视频
    // audio: 'audio',
    source: 'audio',
    // video: 'video',
    track: 'video',
    // 链接
    a: 'navigator',
    nav: 'view',
    link: 'navigator',
    // 列表
    ul: 'view',
    ol: 'view',
    li: 'view',
    dl: 'view',
    dt: 'view',
    dd: 'view',
    menu: 'view',
    command: 'view',
    // 表格table
    table: 'view',
    caption: 'view',
    th: 'view',
    td: 'view',
    tr: 'view',
    thead: 'view',
    tbody: 'view',
    tfoot: 'view',
    col: 'view',
    colgroup: 'view',
    // 样式 节
    div: 'view',
    main: 'view',
    span: 'label',
    header: 'view',
    footer: 'view',
    section: 'view',
    article: 'view',
    aside: 'view',
    details: 'view',
    dialog: 'view',
    summary: 'view',
    // progress: 'progress',
    meter: 'progress', // todo
    head: 'view', // todo
    meta: 'view', // todo
    base: 'text', // todo
    // 'map': 'image', // TODO不是很恰当
    area: 'navigator', // j结合map使用
    script: 'view',
    noscript: 'view',
    embed: 'view',
    object: 'view',
    param: 'view',
};

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618804, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.createCopyComponentDirs = exports.isMiniProgramAssetFile = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const utils_1 = require("../vite/plugins/vitejs/utils");
const json_1 = require("../json");
const EXTNAMES = [
    '.png',
    '.jpg',
    '.jpeg',
    '.gif',
    '.svg',
    '.json',
    '.cer',
    '.mp3',
    '.aac',
    '.m4a',
    '.mp4',
    '.wav',
    '.ogg',
    '.silk',
    '.wasm',
    '.br',
    '.cert',
];
function isMiniProgramAssetFile(filename) {
    if (!path_1.default.isAbsolute(filename)) {
        return false;
    }
    return EXTNAMES.includes(path_1.default.extname(filename));
}
exports.isMiniProgramAssetFile = isMiniProgramAssetFile;
function createCopyComponentDirs(dir) {
    const dirs = [dir];
    const uniModulesDir = 'uni_modules/*/' + dir + '/**/*';
    dirs.push(uniModulesDir);
    const inputDir = process.env.UNI_INPUT_DIR;
    const platform = process.env.UNI_PLATFORM;
    if (!inputDir || !platform) {
        return dirs;
    }
    const pagesJsonFile = path_1.default.resolve((0, utils_1.normalizePath)(inputDir), 'pages.json');
    if (!fs_1.default.existsSync(pagesJsonFile)) {
        return dirs;
    }
    const { appJson } = (0, json_1.parseMiniProgramPagesJson)(fs_1.default.readFileSync(pagesJsonFile, 'utf8'), platform, { subpackages: true });
    const roots = Object.values(appJson.subPackages || appJson.subpackages || {})
        .map(({ root }) => root)
        .filter(Boolean);
    roots.forEach((root) => {
        dirs.push((0, utils_1.normalizePath)(path_1.default.join(root, dir)), (0, utils_1.normalizePath)(path_1.default.join(root, uniModulesDir)));
    });
    return dirs;
}
exports.createCopyComponentDirs = createCopyComponentDirs;

}, function(modId) { var map = {"../json":1781362618806}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618806, function(require, module, exports) {

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
exports.checkPagesJson = exports.parseUniXAppAndroidPackage = exports.getUniXPagePaths = exports.isUniXPageFile = exports.parseUniXSplashScreen = exports.parseUniXFlexDirection = exports.normalizeUniAppXAppConfig = exports.normalizeUniAppXAppPagesJson = void 0;
__exportStar(require("./mp"), exports);
__exportStar(require("./app"), exports);
__exportStar(require("./json"), exports);
__exportStar(require("./pages"), exports);
__exportStar(require("./manifest"), exports);
__exportStar(require("./theme"), exports);
var uni_x_1 = require("./uni-x");
Object.defineProperty(exports, "normalizeUniAppXAppPagesJson", { enumerable: true, get: function () { return uni_x_1.normalizeUniAppXAppPagesJson; } });
Object.defineProperty(exports, "normalizeUniAppXAppConfig", { enumerable: true, get: function () { return uni_x_1.normalizeUniAppXAppConfig; } });
Object.defineProperty(exports, "parseUniXFlexDirection", { enumerable: true, get: function () { return uni_x_1.parseUniXFlexDirection; } });
Object.defineProperty(exports, "parseUniXSplashScreen", { enumerable: true, get: function () { return uni_x_1.parseUniXSplashScreen; } });
Object.defineProperty(exports, "isUniXPageFile", { enumerable: true, get: function () { return uni_x_1.isUniXPageFile; } });
Object.defineProperty(exports, "getUniXPagePaths", { enumerable: true, get: function () { return uni_x_1.getUniXPagePaths; } });
Object.defineProperty(exports, "parseUniXAppAndroidPackage", { enumerable: true, get: function () { return uni_x_1.parseUniXAppAndroidPackage; } });
var utils_1 = require("./utils");
Object.defineProperty(exports, "checkPagesJson", { enumerable: true, get: function () { return utils_1.checkPagesJson; } });

}, function(modId) { var map = {"./mp":1781362618807,"./app":1781362618819,"./json":1781362618810,"./uni-x":1781362618856}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618807, function(require, module, exports) {

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
exports.parseMiniProgramProjectJson = exports.parseMiniProgramPagesJson = exports.mergeMiniProgramAppJson = void 0;
__exportStar(require("./jsonFile"), exports);
var pages_1 = require("./pages");
Object.defineProperty(exports, "mergeMiniProgramAppJson", { enumerable: true, get: function () { return pages_1.mergeMiniProgramAppJson; } });
Object.defineProperty(exports, "parseMiniProgramPagesJson", { enumerable: true, get: function () { return pages_1.parseMiniProgramPagesJson; } });
var project_1 = require("./project");
Object.defineProperty(exports, "parseMiniProgramProjectJson", { enumerable: true, get: function () { return project_1.parseMiniProgramProjectJson; } });

}, function(modId) { var map = {"./pages":1781362618809,"./project":1781362618816}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618809, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.mergeMiniProgramAppJson = exports.parseMiniProgramPagesJson = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const shared_1 = require("@vue/shared");
const json_1 = require("../json");
const pages_1 = require("../pages");
const utils_1 = require("./utils");
const utils_2 = require("../../utils");
const project_1 = require("./project");
const manifest_1 = require("../manifest");
const theme_1 = require("../theme");
function parseMiniProgramPagesJson(jsonStr, platform, options = { subpackages: false }) {
    return parsePagesJson(jsonStr, platform, options);
}
exports.parseMiniProgramPagesJson = parseMiniProgramPagesJson;
const NON_APP_JSON_KEYS = [
    'unipush',
    'secureNetwork',
    'usingComponents',
    'optimization',
    'scopedSlotsCompiler',
    'uniStatistics',
    'mergeVirtualHostAttributes',
    'styleIsolation',
    'enableVirtualHost',
    'nativeTags',
];
function mergeMiniProgramAppJson(appJson, platformJson = {}, source = {}) {
    Object.keys(source).forEach((key) => {
        if (!project_1.projectKeys.includes(key)) {
            project_1.projectKeys.push(key);
        }
    });
    Object.keys(platformJson).forEach((name) => {
        if (!(0, project_1.isMiniProgramProjectJsonKey)(name) &&
            !NON_APP_JSON_KEYS.includes(name)) {
            appJson[name] = platformJson[name];
        }
    });
}
exports.mergeMiniProgramAppJson = mergeMiniProgramAppJson;
function parsePagesJson(jsonStr, platform, { debug, darkmode, networkTimeout, subpackages, windowOptionsMap, tabBarOptionsMap, tabBarItemOptionsMap, } = {
    subpackages: false,
}) {
    let appJson = {
        pages: [],
    };
    let pageJsons = {};
    let nvuePages = [];
    // preprocess
    const pagesJson = (0, json_1.parseJson)(jsonStr, true, 'pages.json');
    if (!pagesJson) {
        throw new Error(`[vite] Error: pages.json parse failed.\n`);
    }
    function addPageJson(pagePath, style) {
        const filename = path_1.default.join(process.env.UNI_INPUT_DIR, pagePath);
        if (fs_1.default.existsSync(filename + '.nvue') &&
            !fs_1.default.existsSync(filename + '.vue')) {
            nvuePages.push(pagePath);
        }
        const windowOptions = {};
        if (platform === 'mp-baidu') {
            // 仅百度小程序需要页面配置 component:true
            // 快手小程序反而不能配置 component:true，故不能统一添加，目前硬编码处理
            windowOptions.component = true;
        }
        pageJsons[pagePath] = (0, shared_1.extend)(windowOptions, (0, utils_1.parseWindowOptions)(style, platform, windowOptionsMap));
    }
    // pages
    (0, pages_1.validatePages)(pagesJson, jsonStr);
    pagesJson.pages.forEach((page) => {
        appJson.pages.push(page.path);
        addPageJson(page.path, page.style);
    });
    // subpackages
    pagesJson.subPackages = pagesJson.subPackages || pagesJson.subpackages;
    if (pagesJson.subPackages) {
        if (subpackages) {
            appJson.subPackages = pagesJson.subPackages.map(({ root, pages, ...rest }) => {
                return (0, shared_1.extend)({
                    root,
                    pages: pages.map((page) => {
                        addPageJson((0, utils_2.normalizePath)(path_1.default.join(root, page.path)), page.style);
                        return page.path;
                    }),
                }, rest);
            });
        }
        else {
            pagesJson.subPackages.forEach(({ root, pages }) => {
                pages.forEach((page) => {
                    const pagePath = (0, utils_2.normalizePath)(path_1.default.join(root, page.path));
                    appJson.pages.push(pagePath);
                    addPageJson(pagePath, page.style);
                });
            });
        }
    }
    // window
    if (pagesJson.globalStyle) {
        const windowOptions = (0, utils_1.parseWindowOptions)(pagesJson.globalStyle, platform, windowOptionsMap);
        const { usingComponents } = windowOptions;
        if (usingComponents) {
            delete windowOptions.usingComponents;
            appJson.usingComponents = usingComponents;
        }
        else {
            delete appJson.usingComponents;
        }
        appJson.window = windowOptions;
    }
    // tabBar
    if (pagesJson.tabBar) {
        const tabBar = (0, utils_1.parseTabBar)(pagesJson.tabBar, platform, tabBarOptionsMap, tabBarItemOptionsMap);
        if (tabBar) {
            appJson.tabBar = tabBar;
        }
    }
    (0, pages_1.filterPlatformPages)(platform, pagesJson);
    ['preloadRule', 'workers', 'plugins', 'entryPagePath'].forEach((name) => {
        if ((0, shared_1.hasOwn)(pagesJson, name)) {
            appJson[name] = pagesJson[name];
        }
    });
    if (debug) {
        appJson.debug = debug;
    }
    if (networkTimeout) {
        appJson.networkTimeout = networkTimeout;
    }
    const manifestJson = (0, manifest_1.getPlatformManifestJsonOnce)();
    if (!darkmode) {
        const { pages, window, tabBar } = (0, theme_1.initTheme)(manifestJson, appJson);
        (0, shared_1.extend)(appJson, JSON.parse(JSON.stringify({ pages, window, tabBar })));
        delete appJson.darkmode;
        delete appJson.themeLocation;
        pageJsons = (0, theme_1.initTheme)(manifestJson, pageJsons);
    }
    else {
        const themeLocation = manifestJson.themeLocation || 'theme.json';
        if ((0, theme_1.hasThemeJson)(path_1.default.join(process.env.UNI_INPUT_DIR, themeLocation)))
            appJson.themeLocation = themeLocation;
    }
    return {
        appJson,
        pageJsons,
        nvuePages,
    };
}

}, function(modId) { var map = {"../json":1781362618810,"./utils":1781362618813,"./project":1781362618816}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618810, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.parseJson = void 0;
const jsonc_parser_1 = require("jsonc-parser");
const preprocess_1 = require("../preprocess");
function parseJson(jsonStr, shouldPre = false, filename) {
    return (0, jsonc_parser_1.parse)(shouldPre
        ? process.env.UNI_APP_X === 'true'
            ? (0, preprocess_1.preUVueJson)(jsonStr, filename)
            : (0, preprocess_1.preJson)(jsonStr, filename)
        : jsonStr);
}
exports.parseJson = parseJson;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618813, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.parseTabBar = exports.parseWindowOptions = void 0;
const shared_1 = require("@vue/shared");
const pages_1 = require("../pages");
function trimJson(json) {
    delete json.maxWidth;
    delete json.topWindow;
    delete json.leftWindow;
    delete json.rightWindow;
    if (json.tabBar) {
        delete json.tabBar.matchMedia;
    }
    return json;
}
function convert(to, from, map) {
    Object.keys(map).forEach((key) => {
        if ((0, shared_1.hasOwn)(from, map[key])) {
            to[key] = from[map[key]];
        }
    });
    return to;
}
function parseWindowOptions(style, platform, windowOptionsMap) {
    if (!style) {
        return {};
    }
    const platformStyle = style[platform] || {};
    (0, pages_1.removePlatformStyle)(trimJson(style));
    const res = {};
    if (windowOptionsMap) {
        return (0, shared_1.extend)(convert(res, style, windowOptionsMap), platformStyle);
    }
    return (0, shared_1.extend)(res, style, platformStyle);
}
exports.parseWindowOptions = parseWindowOptions;
function trimTabBarJson(tabBar) {
    ;
    [
        'fontSize',
        'height',
        'iconWidth',
        'midButton',
        'selectedIndex',
        'spacing',
    ].forEach((name) => {
        delete tabBar[name];
    });
    return tabBar;
}
function parseTabBar(tabBar, platform, tabBarOptionsMap, tabBarItemOptionsMap) {
    const platformStyle = tabBar[platform] || {};
    (0, pages_1.removePlatformStyle)(trimTabBarJson(tabBar));
    const res = {};
    if (tabBarOptionsMap) {
        if (tabBarItemOptionsMap && tabBar.list) {
            tabBar.list = tabBar.list.map((item) => {
                return convert({}, item, tabBarItemOptionsMap);
            });
        }
        convert(res, tabBar, tabBarOptionsMap);
        return (0, shared_1.extend)(res, platformStyle);
    }
    return (0, shared_1.extend)(res, tabBar, platformStyle);
}
exports.parseTabBar = parseTabBar;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618816, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.parseMiniProgramProjectJson = exports.isMiniProgramProjectJsonKey = exports.projectKeys = void 0;
const shared_1 = require("@vue/shared");
const merge_1 = require("merge");
const json_1 = require("../json");
exports.projectKeys = [
    'appid',
    'setting',
    'miniprogramRoot',
    'cloudfunctionRoot',
    'qcloudRoot',
    'pluginRoot',
    'compileType',
    'libVersion',
    'projectname',
    'packOptions',
    'debugOptions',
    'scripts',
    'cloudbaseRoot',
    'watchOptions',
];
function isMiniProgramProjectJsonKey(name) {
    return exports.projectKeys.includes(name);
}
exports.isMiniProgramProjectJsonKey = isMiniProgramProjectJsonKey;
function parseMiniProgramProjectJson(jsonStr, platform, { template, pagesJson }) {
    const projectJson = JSON.parse(JSON.stringify(template));
    const manifestJson = (0, json_1.parseJson)(jsonStr, false, '');
    if (manifestJson) {
        projectJson.projectname = manifestJson.name;
        // 用户的平台配置
        const platformConfig = manifestJson[platform];
        if (platformConfig) {
            const setProjectJson = (name) => {
                if ((0, shared_1.hasOwn)(platformConfig, name)) {
                    if ((0, shared_1.isPlainObject)(platformConfig[name]) &&
                        (0, shared_1.isPlainObject)(projectJson[name])) {
                        ;
                        projectJson[name] = (0, merge_1.recursive)(true, projectJson[name], platformConfig[name]);
                    }
                    else {
                        ;
                        projectJson[name] = platformConfig[name];
                    }
                }
            };
            // 读取 template 中的配置
            Object.keys(template).forEach((name) => {
                if (!exports.projectKeys.includes(name)) {
                    exports.projectKeys.push(name);
                }
            });
            // common mp config
            exports.projectKeys.forEach((name) => {
                setProjectJson(name);
            });
            // 使用了微信小程序手势系统，自动开启 ES6=>ES5
            platform === 'mp-weixin' &&
                weixinSkyline(platformConfig) &&
                openES62ES5(projectJson);
        }
    }
    // 其实仅开发期间 condition 生效即可，暂不做判断
    const miniprogram = parseMiniProgramCondition(pagesJson);
    if (miniprogram) {
        if (!projectJson.condition) {
            projectJson.condition = {};
        }
        projectJson.condition.miniprogram = miniprogram;
    }
    // appid
    if (!projectJson.appid) {
        projectJson.appid = 'touristappid';
    }
    return projectJson;
}
exports.parseMiniProgramProjectJson = parseMiniProgramProjectJson;
function weixinSkyline(config) {
    return (config.renderer === 'skyline' &&
        config.lazyCodeLoading === 'requiredComponents');
}
function openES62ES5(config) {
    if (!config.setting) {
        config.setting = {};
    }
    if (!config.setting.es6) {
        config.setting.es6 = true;
    }
}
function parseMiniProgramCondition(pagesJson) {
    const launchPagePath = process.env.UNI_CLI_LAUNCH_PAGE_PATH || '';
    if (launchPagePath) {
        return {
            current: 0,
            list: [
                {
                    id: 0,
                    name: launchPagePath, // 模式名称
                    pathName: launchPagePath, // 启动页面，必选
                    query: process.env.UNI_CLI_LAUNCH_PAGE_QUERY || '', // 启动参数，在页面的onLoad函数里面得到。
                },
            ],
        };
    }
    const condition = pagesJson.condition;
    if (!condition || !(0, shared_1.isArray)(condition.list) || !condition.list.length) {
        return;
    }
    condition.list.forEach(function (item, index) {
        item.id = item.id || index;
        if (item.path) {
            item.pathName = item.path;
            delete item.path;
        }
    });
    return condition;
}

}, function(modId) { var map = {"../json":1781362618810}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618819, function(require, module, exports) {

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
exports.restoreGlobalCode = exports.arrayBufferCode = exports.polyfillCode = void 0;
__exportStar(require("./pages"), exports);
__exportStar(require("./manifest"), exports);
var code_1 = require("./pages/code");
Object.defineProperty(exports, "polyfillCode", { enumerable: true, get: function () { return code_1.polyfillCode; } });
Object.defineProperty(exports, "arrayBufferCode", { enumerable: true, get: function () { return code_1.arrayBufferCode; } });
Object.defineProperty(exports, "restoreGlobalCode", { enumerable: true, get: function () { return code_1.restoreGlobalCode; } });

}, function(modId) { var map = {"./pages":1781362618820,"./manifest":1781362618827,"./pages/code":1781362618821}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618820, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.normalizeAppConfigService = exports.normalizeAppNVuePagesJson = exports.normalizeAppPagesJson = exports.staticImportPageCode = void 0;
const code_1 = require("./code");
const definePage_1 = require("./definePage");
const uniConfig_1 = require("./uniConfig");
const uniRoutes_1 = require("./uniRoutes");
var definePage_2 = require("./definePage");
Object.defineProperty(exports, "staticImportPageCode", { enumerable: true, get: function () { return definePage_2.staticImportPageCode; } });
function normalizeAppPagesJson(pagesJson, platform = 'app', dynamicImport = false) {
    return (0, definePage_1.definePageCode)(pagesJson, platform, dynamicImport);
}
exports.normalizeAppPagesJson = normalizeAppPagesJson;
function normalizeAppNVuePagesJson(pagesJson) {
    return (0, definePage_1.defineNVuePageCode)(pagesJson);
}
exports.normalizeAppNVuePagesJson = normalizeAppNVuePagesJson;
function normalizeAppConfigService(pagesJson, manifestJson) {
    return `
  ;(function(){
  let u=void 0,isReady=false,onReadyCallbacks=[],isServiceReady=false,onServiceReadyCallbacks=[];
  const __uniConfig = ${(0, uniConfig_1.normalizeAppUniConfig)(pagesJson, manifestJson)};
  const __uniRoutes = ${(0, uniRoutes_1.normalizeAppUniRoutes)(pagesJson)}.map(uniRoute=>(uniRoute.meta.route=uniRoute.path,__uniConfig.pages.push(uniRoute.path),uniRoute.path='/'+uniRoute.path,uniRoute));
  __uniConfig.styles=${process.env.UNI_NVUE_APP_STYLES || '[]'};//styles
  __uniConfig.onReady=function(callback){if(__uniConfig.ready){callback()}else{onReadyCallbacks.push(callback)}};Object.defineProperty(__uniConfig,"ready",{get:function(){return isReady},set:function(val){isReady=val;if(!isReady){return}const callbacks=onReadyCallbacks.slice(0);onReadyCallbacks.length=0;callbacks.forEach(function(callback){callback()})}});
  __uniConfig.onServiceReady=function(callback){if(__uniConfig.serviceReady){callback()}else{onServiceReadyCallbacks.push(callback)}};Object.defineProperty(__uniConfig,"serviceReady",{get:function(){return isServiceReady},set:function(val){isServiceReady=val;if(!isServiceReady){return}const callbacks=onServiceReadyCallbacks.slice(0);onServiceReadyCallbacks.length=0;callbacks.forEach(function(callback){callback()})}});
  service.register("uni-app-config",{create(a,b,c){if(!__uniConfig.viewport){var d=b.weex.config.env.scale,e=b.weex.config.env.deviceWidth,f=Math.ceil(e/d);Object.assign(__uniConfig,{viewport:f,defaultFontSize:16})}return{instance:{__uniConfig:__uniConfig,__uniRoutes:__uniRoutes,${code_1.globalCode}}}}}); 
  })();
  `;
}
exports.normalizeAppConfigService = normalizeAppConfigService;

}, function(modId) { var map = {"./code":1781362618821,"./definePage":1781362618822,"./uniRoutes":1781362618825}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618821, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.globalCode = exports.restoreGlobalCode = exports.polyfillCode = exports.arrayBufferCode = void 0;
exports.arrayBufferCode = `
if (typeof uni !== 'undefined' && uni && uni.requireGlobal) {
  const global = uni.requireGlobal()
  ArrayBuffer = global.ArrayBuffer
  Int8Array = global.Int8Array
  Uint8Array = global.Uint8Array
  Uint8ClampedArray = global.Uint8ClampedArray
  Int16Array = global.Int16Array
  Uint16Array = global.Uint16Array
  Int32Array = global.Int32Array
  Uint32Array = global.Uint32Array
  Float32Array = global.Float32Array
  Float64Array = global.Float64Array
  BigInt64Array = global.BigInt64Array
  BigUint64Array = global.BigUint64Array
};
`;
exports.polyfillCode = `
if (typeof Promise !== 'undefined' && !Promise.prototype.finally) {
  Promise.prototype.finally = function(callback) {
    const promise = this.constructor
    return this.then(
      value => promise.resolve(callback()).then(() => value),
      reason => promise.resolve(callback()).then(() => {
        throw reason
      })
    )
  }
};
${exports.arrayBufferCode}
`;
exports.restoreGlobalCode = `
if(uni.restoreGlobal){
  uni.restoreGlobal(Vue,weex,plus,setTimeout,clearTimeout,setInterval,clearInterval)
}
`;
const GLOBALS = [
    'global',
    'window',
    'document',
    'frames',
    'self',
    'location',
    'navigator',
    'localStorage',
    'history',
    'Caches',
    'screen',
    'alert',
    'confirm',
    'prompt',
    'fetch',
    'XMLHttpRequest',
    'WebSocket',
    'webkit',
    'print',
];
exports.globalCode = GLOBALS.map((g) => `${g}:u`).join(',');

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618822, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.defineNVuePageCode = exports.definePageCode = exports.staticImportPageCode = void 0;
const utils_1 = require("../../../utils");
function staticImportPageCode(pagesJson) {
    const importPagesCode = [];
    pagesJson.pages.forEach((page) => {
        const pagePath = page.path;
        const pagePathWithExtname = (0, utils_1.normalizePagePath)(pagePath, 'app');
        if (pagePathWithExtname) {
            importPagesCode.push(`import '../assets/${(0, utils_1.removeExt)(pagePathWithExtname)}'`);
        }
    });
    return importPagesCode.join('\n');
}
exports.staticImportPageCode = staticImportPageCode;
function definePageCode(pagesJson, platform = 'app', dynamicImport = false) {
    const importPagesCode = [];
    const definePagesCode = [];
    pagesJson.pages.forEach((page) => {
        if (platform === 'app' && page.style.isNVue) {
            return;
        }
        const pagePath = page.path;
        const pageIdentifier = (0, utils_1.normalizeIdentifier)(pagePath);
        const pagePathWithExtname = (0, utils_1.normalizePagePath)(pagePath, platform);
        if (pagePathWithExtname) {
            if (dynamicImport) {
                // 拆分页面
                importPagesCode.push(`const ${pageIdentifier} = ()=>import('./${pagePathWithExtname}')`);
            }
            else {
                importPagesCode.push(`import ${pageIdentifier} from './${pagePathWithExtname}'`);
            }
            definePagesCode.push(`__definePage('${pagePath}',${pageIdentifier})`);
        }
    });
    return importPagesCode.join('\n') + '\n' + definePagesCode.join('\n');
}
exports.definePageCode = definePageCode;
function defineNVuePageCode(pagesJson) {
    const importNVuePagesCode = [];
    pagesJson.pages.forEach((page) => {
        if (!page.style.isNVue) {
            return;
        }
        const pagePathWithExtname = (0, utils_1.normalizePagePath)(page.path, 'app');
        if (pagePathWithExtname) {
            importNVuePagesCode.push(`import('./${pagePathWithExtname}').then((res)=>{res()})`);
        }
    });
    return importNVuePagesCode.join('\n');
}
exports.defineNVuePageCode = defineNVuePageCode;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618825, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.normalizeAppUniRoutes = void 0;
const pages_1 = require("../../pages");
function normalizeAppUniRoutes(pagesJson) {
    return JSON.stringify((0, pages_1.normalizePagesRoute)(pagesJson));
}
exports.normalizeAppUniRoutes = normalizeAppUniRoutes;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618827, function(require, module, exports) {

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
exports.initI18n = exports.parseArguments = exports.getNVueFlexDirection = exports.getNVueStyleCompiler = exports.getNVueCompiler = exports.hasConfusionFile = exports.isConfusionFile = exports.APP_CONFUSION_FILENAME = exports.normalizeAppManifestJson = void 0;
const shared_1 = require("@vue/shared");
const merge_1 = require("./merge");
const defaultManifestJson_1 = require("./defaultManifestJson");
const statusbar_1 = require("./statusbar");
const plus_1 = require("./plus");
const nvue_1 = require("./nvue");
const arguments_1 = require("./arguments");
const safearea_1 = require("./safearea");
const splashscreen_1 = require("./splashscreen");
const confusion_1 = require("./confusion");
const uniApp_1 = require("./uniApp");
const launchwebview_1 = require("./launchwebview");
const checksystemwebview_1 = require("./checksystemwebview");
const tabBar_1 = require("./tabBar");
const i18n_1 = require("./i18n");
const theme_1 = require("../../theme");
const manifest_1 = require("../../manifest");
function normalizeAppManifestJson(userManifestJson, pagesJson) {
    const manifestJson = (0, merge_1.initRecursiveMerge)((0, defaultManifestJson_1.initDefaultManifestJson)(), userManifestJson);
    const manifestJsonPlatform = (0, manifest_1.getPlatformManifestJson)(manifestJson);
    const { pages, globalStyle, tabBar } = (0, theme_1.initTheme)(manifestJsonPlatform, pagesJson);
    (0, shared_1.extend)(pagesJson, JSON.parse(JSON.stringify({ pages, globalStyle, tabBar })));
    (0, statusbar_1.initAppStatusbar)(manifestJson, pagesJson);
    (0, arguments_1.initArguments)(manifestJson, pagesJson);
    (0, plus_1.initPlus)(manifestJson, pagesJson);
    (0, nvue_1.initNVue)(manifestJson, pagesJson);
    (0, safearea_1.initSafearea)(manifestJson, pagesJson);
    (0, splashscreen_1.initSplashscreen)(manifestJson, userManifestJson);
    (0, confusion_1.initConfusion)(manifestJson);
    (0, uniApp_1.initUniApp)(manifestJson);
    // 依赖 initArguments 先执行
    (0, tabBar_1.initTabBar)((0, launchwebview_1.initLaunchwebview)(manifestJson, pagesJson), manifestJson, pagesJson);
    // 依赖 initUniApp 先执行
    (0, checksystemwebview_1.initCheckSystemWebview)(manifestJson);
    return (0, i18n_1.initI18n)(manifestJson);
}
exports.normalizeAppManifestJson = normalizeAppManifestJson;
__exportStar(require("./env"), exports);
var confusion_2 = require("./confusion");
Object.defineProperty(exports, "APP_CONFUSION_FILENAME", { enumerable: true, get: function () { return confusion_2.APP_CONFUSION_FILENAME; } });
Object.defineProperty(exports, "isConfusionFile", { enumerable: true, get: function () { return confusion_2.isConfusionFile; } });
Object.defineProperty(exports, "hasConfusionFile", { enumerable: true, get: function () { return confusion_2.hasConfusionFile; } });
var nvue_2 = require("./nvue");
Object.defineProperty(exports, "getNVueCompiler", { enumerable: true, get: function () { return nvue_2.getNVueCompiler; } });
Object.defineProperty(exports, "getNVueStyleCompiler", { enumerable: true, get: function () { return nvue_2.getNVueStyleCompiler; } });
Object.defineProperty(exports, "getNVueFlexDirection", { enumerable: true, get: function () { return nvue_2.getNVueFlexDirection; } });
var arguments_2 = require("./arguments");
Object.defineProperty(exports, "parseArguments", { enumerable: true, get: function () { return arguments_2.parseArguments; } });
var i18n_2 = require("./i18n");
Object.defineProperty(exports, "initI18n", { enumerable: true, get: function () { return i18n_2.initI18n; } });

}, function(modId) { var map = {"./merge":1781362618828,"./defaultManifestJson":1781362618829,"./statusbar":1781362618830,"./uniApp":1781362618837,"./checksystemwebview":1781362618840,"./i18n":1781362618842}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618828, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.initRecursiveMerge = void 0;
const merge_1 = require("merge");
function initRecursiveMerge(manifestJson, userManifestJson) {
    const platformConfig = {
        plus: userManifestJson['app-plus'],
    };
    platformConfig['app-harmony'] = userManifestJson['app-harmony'];
    return (0, merge_1.recursive)(true, manifestJson, {
        id: userManifestJson.appid || '',
        name: userManifestJson.name || '',
        description: userManifestJson.description || '',
        version: {
            name: userManifestJson.versionName,
            code: userManifestJson.versionCode,
        },
        locale: userManifestJson.locale,
        uniStatistics: userManifestJson.uniStatistics,
    }, platformConfig);
}
exports.initRecursiveMerge = initRecursiveMerge;

}, function(modId) { var map = {"merge":1781362618828}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618829, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.initDefaultManifestJson = void 0;
function initDefaultManifestJson() {
    return JSON.parse(defaultManifestJson);
}
exports.initDefaultManifestJson = initDefaultManifestJson;
const defaultManifestJson = `{
    "@platforms": [
        "android",
        "iPhone",
        "iPad"
    ],
    "id": "__WEAPP_ID",
    "name": "__WEAPP_NAME",
    "version": {
        "name": "1.0",
        "code": ""
    },
    "description": "",
    "developer": {
        "name": "",
        "email": "",
        "url": ""
    },
    "permissions": {},
    "plus": {
        "useragent": {
            "value": "",
            "concatenate": true
        },
        "splashscreen": {
            "target":"id:1",
            "autoclose": true,
            "waiting": true,
            "alwaysShowBeforeRender":true
        },
        "popGesture": "close",
        "launchwebview": {}
    },
    "app-harmony": {
        "useragent": {
            "value": "",
            "concatenate": true
        }
    }
}`;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618830, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.initAppStatusbar = void 0;
function initAppStatusbar(manifestJson, pagesJson) {
    const titleColor = pagesJson.pages[0].style.navigationBar.titleColor ||
        pagesJson.globalStyle.navigationBar.titleColor ||
        '#000000';
    const backgroundColor = pagesJson.globalStyle.navigationBar.backgroundColor || '#000000';
    manifestJson.plus.statusbar = {
        immersed: 'supportedDevice',
        style: titleColor === '#ffffff' ? 'light' : 'dark',
        background: backgroundColor,
    };
    return manifestJson;
}
exports.initAppStatusbar = initAppStatusbar;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618837, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.initUniApp = void 0;
const nvue_1 = require("./nvue");
function initUniApp(manifestJson) {
    manifestJson.plus['uni-app'] = {
        control: 'uni-v3',
        vueVersion: '3',
        compilerVersion: process.env.UNI_COMPILER_VERSION,
        nvueCompiler: (0, nvue_1.getNVueCompiler)(manifestJson),
        renderer: 'auto',
        nvue: {
            'flex-direction': (0, nvue_1.getNVueFlexDirection)(manifestJson),
        },
        nvueLaunchMode: manifestJson.plus.nvueLaunchMode === 'fast' ? 'fast' : 'normal',
    };
    delete manifestJson.plus.nvueLaunchMode;
}
exports.initUniApp = initUniApp;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618840, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.initCheckSystemWebview = void 0;
function initCheckSystemWebview(manifestJson) {
    // 检查Android系统webview版本 || 下载X5后启动
    let plusWebView = manifestJson.plus.webView;
    if (plusWebView) {
        manifestJson.plus['uni-app'].webView = plusWebView;
        delete manifestJson.plus.webView;
    }
    else {
        manifestJson.plus['uni-app'].webView = {
            minUserAgentVersion: '49.0',
        };
    }
}
exports.initCheckSystemWebview = initCheckSystemWebview;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618842, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.initI18n = void 0;
const uni_i18n_1 = require("@dcloudio/uni-i18n");
const i18n_1 = require("../../../i18n");
function initI18n(manifestJson) {
    const i18nOptions = (0, i18n_1.initI18nOptions)(process.env.UNI_PLATFORM, process.env.UNI_INPUT_DIR, true);
    if (i18nOptions) {
        manifestJson = JSON.parse((0, uni_i18n_1.compileI18nJsonStr)(JSON.stringify(manifestJson), i18nOptions));
        manifestJson.fallbackLocale = i18nOptions.locale;
    }
    return manifestJson;
}
exports.initI18n = initI18n;

}, function(modId) { var map = {"../../../i18n":1781362618843}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618843, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.resolveI18nLocale = exports.initLocales = exports.getLocaleFiles = exports.isUniAppLocaleFile = exports.initI18nOptionsOnce = exports.initI18nOptions = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const fast_glob_1 = require("fast-glob");
const shared_1 = require("@vue/shared");
const uni_shared_1 = require("@dcloudio/uni-shared");
const json_1 = require("./json");
const messages_1 = require("./messages");
function initI18nOptions(platform, inputDir, warning = false, withMessages = true) {
    const locales = initLocales(path_1.default.resolve(inputDir, 'locale'), withMessages);
    if (!Object.keys(locales).length) {
        return;
    }
    const manifestJson = (0, json_1.parseManifestJsonOnce)(inputDir);
    let fallbackLocale = manifestJson.fallbackLocale || 'en';
    const IS_APP_X = process.env.UNI_APP_X === 'true';
    if (IS_APP_X) {
        fallbackLocale = manifestJson.locale || fallbackLocale;
    }
    const locale = resolveI18nLocale(platform, Object.keys(locales), fallbackLocale);
    if (warning) {
        if (!fallbackLocale) {
            console.warn(messages_1.M['i18n.fallbackLocale.default'].replace('{locale}', locale));
        }
        else if (locale !== fallbackLocale) {
            console.warn(messages_1.M['i18n.fallbackLocale.missing'].replace('{locale}', fallbackLocale));
        }
    }
    return {
        locale,
        locales,
        delimiters: uni_shared_1.I18N_JSON_DELIMITERS,
    };
}
exports.initI18nOptions = initI18nOptions;
exports.initI18nOptionsOnce = (0, uni_shared_1.once)(initI18nOptions);
const localeJsonRE = /uni-app.*.json/;
function isUniAppLocaleFile(filepath) {
    if (!filepath) {
        return false;
    }
    return localeJsonRE.test(path_1.default.basename(filepath));
}
exports.isUniAppLocaleFile = isUniAppLocaleFile;
function parseLocaleJson(filepath) {
    let jsonObj = (0, json_1.parseJson)(fs_1.default.readFileSync(filepath, 'utf8'), false, filepath);
    if (isUniAppLocaleFile(filepath)) {
        jsonObj = jsonObj.common || {};
    }
    return jsonObj;
}
function getLocaleFiles(cwd) {
    return (0, fast_glob_1.sync)('*.json', { cwd, absolute: true });
}
exports.getLocaleFiles = getLocaleFiles;
function initLocales(dir, withMessages = true) {
    if (!fs_1.default.existsSync(dir)) {
        return {};
    }
    return fs_1.default.readdirSync(dir).reduce((res, filename) => {
        if (path_1.default.extname(filename) === '.json') {
            try {
                const locale = path_1.default
                    .basename(filename)
                    .replace(/(uni-app.)?(.*).json/, '$2');
                if (withMessages) {
                    (0, shared_1.extend)(res[locale] || (res[locale] = {}), parseLocaleJson(path_1.default.join(dir, filename)));
                }
                else {
                    res[locale] = {};
                }
            }
            catch (e) { }
        }
        return res;
    }, {});
}
exports.initLocales = initLocales;
function resolveI18nLocale(platform, locales, locale) {
    if (locale && locales.includes(locale)) {
        return locale;
    }
    const defaultLocales = ['zh-Hans', 'zh-Hant'];
    if (platform === 'app' || platform === 'h5') {
        defaultLocales.unshift('en');
    }
    else {
        // 小程序
        defaultLocales.push('en');
    }
    return defaultLocales.find((locale) => locales.includes(locale)) || locales[0];
}
exports.resolveI18nLocale = resolveI18nLocale;

}, function(modId) { var map = {"fs":1781362618793,"./json":1781362618806,"./messages":1781362618844}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618844, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.M = void 0;
const os_locale_s_fix_1 = require("os-locale-s-fix");
const en_1 = __importDefault(require("./en"));
const zh_CN_1 = __importDefault(require("./zh_CN"));
function format(lang) {
    const array = lang.split(/[.,]/)[0].split(/[_-]/);
    array[0] = array[0].toLowerCase();
    if (array[0] === 'zh') {
        array[1] = (array[1] || 'CN').toUpperCase();
    }
    array.length = Math.min(array.length, 2);
    return array.join('_');
}
const locale = format(process.env.UNI_HBUILDERX_LANGID ||
    os_locale_s_fix_1.osLocale.sync({ spawn: true, cache: false }) ||
    'en');
exports.M = locale === 'zh_CN' ? zh_CN_1.default : en_1.default;

}, function(modId) { var map = {"./en":1781362618845,"./zh_CN":1781362618846}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618845, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.default = {
    'app.compiler.version': 'Compiler version: {version}',
    'style.isolation.version': 'Style isolation version: {version}',
    compiling: 'Compiling...',
    'dev.performance': 'Please note that in running mode, due to log output, sourcemap, and uncompressed source code, the performance and package size are not as good as release mode.',
    'dev.performance.nvue': 'Especially the sourcemap of app-nvue has a greater impact',
    'dev.performance.mp': 'To officially release, please click the release menu or use the cli release command to release',
    'dev.performance.web': '\nVite is compiled on demand, and clicking on an uncompiled page at runtime will compile first and then load, resulting in a slower display, and there is no such problem after release.',
    'build.done': 'DONE  Build complete.',
    'dev.watching.start': 'Incremental Compiling...',
    'dev.watching.end': 'DONE  Build complete. Watching for changes...',
    'dev.watching.end.pages': 'DONE  Build complete. PAGES:{pages}',
    'dev.watching.end.files': 'DONE  Build complete. FILES:{files}',
    'dev.watching.restart.vapor': 'DONE  Restart required. Switching vapor mode requires restarting to take effect.',
    'build.failed': 'DONE  Build failed.',
    'compiler.build.failed': 'Build failed with errors.',
    'stat.warn.appid': 'The current application is not configured with Appid, and uni statistics cannot be used. For details, see https://ask.dcloud.net.cn/article/36303',
    'stat.warn.version': 'The uni statistics version is not configured. The default version is 1.0.uni statistics version 2.0 is recommended, private deployment data is more secure and code is open source and customizable. details: https://uniapp.dcloud.io/uni-stat',
    'stat.warn.tip': 'uni statistics version: {version}',
    'i18n.fallbackLocale.default': 'fallbackLocale is missing in manifest.json, use: {locale}',
    'i18n.fallbackLocale.missing': './local/{locale}.json is missing',
    'easycom.conflict': 'easycom component conflict: ',
    'mp.component.args[0]': 'The first parameter of {0} must be a static string',
    'mp.component.args[1]': '{0} requires two parameters',
    'mp.360.unsupported': '360 is unsupported',
    'file.notfound': '{file} is not found',
    'uts.ios.tips': 'The project uses the uts plugin. After the uts plug-in code is modified, the [Custom playground native runner](https://uniapp.dcloud.net.cn/tutorial/run/run-app.html#customplayground) needs to be regenerated to take effect',
    'uts.android.compiler.server': 'The project uses the uts plugin, installing the uts Android runtime extension...',
    'uts.ios.windows.tips': 'When running on Windows to iOS mobile phone, the modification of the uts plugin code needs to be submitted to the cloud to package the custom playground to take effect.',
    'uts.ios.standard.tips': 'When the standard playground runs to an IOS phone, the uts plugin is temporarily not supported. If you need to call the uts plugin, please use a custom playground',
    'prompt.run.message': 'Run method: open {devtools}, import {outputDir} run.',
    'prompt.run.devtools.app': 'HBuilderX',
    'prompt.run.devtools.app-harmony': 'HBuilderX',
    'prompt.run.devtools.mp-harmony': 'HBuilderX',
    'prompt.run.devtools.mp-alipay': 'Alipay Mini Program Devtools',
    'prompt.run.devtools.mp-baidu': 'Baidu Mini Program Devtools',
    'prompt.run.devtools.mp-kuaishou': 'Kuaishou Mini Program Devtools',
    'prompt.run.devtools.mp-lark': 'Lark Mini Program Devtools',
    'prompt.run.devtools.mp-qq': 'QQ Mini Program Devtools',
    'prompt.run.devtools.mp-toutiao': 'Douyin Mini Program Devtools',
    'prompt.run.devtools.mp-weixin': 'Weixin Mini Program Devtools',
    'prompt.run.devtools.mp-jd': 'Jingdong Mini Program Devtools',
    'prompt.run.devtools.mp-xhs': 'Xiaohongshu Mini Program Devtools',
    'prompt.run.devtools.quickapp-webview': 'Quick App Alliance Devtools | Huawei Quick App Devtools',
    'prompt.run.devtools.quickapp-webview-huawei': 'Huawei Quick App Devtools',
    'prompt.run.devtools.quickapp-webview-union': 'Quick App Alliance Devtools',
    'uvue.unsupported': 'uvue does not support {platform} platform',
    'uvue.dev.watching.end.empty': 'The compilation outcome remains unchanged; there is no need to synchronize.',
    'uni_modules.import': 'Plug-in [{0}] only supports @/uni_modules/{1}.',
    'pages.json.page.notfound': 'The page "{pagePath}" does not exist.',
    'pages.json.page.slash': 'The Path "{pagePath}" cannot start with "/"',
    'pages.json.tabbar.page.notfound': 'The tabBar page "{pagePath}" is not declared in "pages.json".',
};

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618846, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.default = {
    'app.compiler.version': '编译器版本：{version}',
    'style.isolation.version': '当前样式隔离策略：{version}。详见：https://doc.dcloud.net.cn/uni-app-x/css/common/style-isolation.html',
    compiling: '正在编译中...',
    'dev.performance': '请注意运行模式下，因日志输出、sourcemap 以及未压缩源码等原因，性能和包体积，均不及发行模式。',
    'dev.performance.nvue': '尤其是 app-nvue 的 sourcemap 影响较大',
    'dev.performance.mp': '若要正式发布，请点击发行菜单或使用 cli 发布命令进行发布',
    'dev.performance.web': '\nvite是按需编译，运行时点击某个未编译页面会先编译后加载，导致显示较慢，发行后无此问题。',
    'build.done': 'DONE  Build complete.',
    'dev.watching.start': '开始差量编译...',
    'dev.watching.end': 'DONE  Build complete. Watching for changes...',
    'dev.watching.end.pages': 'DONE  Build complete. PAGES:{pages}',
    'dev.watching.end.files': 'DONE  Build complete. FILES:{files}',
    'dev.watching.restart.vapor': 'DONE  Restart required. 切换蒸汽模式需要重新运行才能生效。',
    'build.failed': 'DONE  Build failed.',
    'compiler.build.failed': '编译失败',
    'stat.warn.appid': '当前应用未配置 appid，无法使用 uni 统计，详情参考：https://ask.dcloud.net.cn/article/36303',
    'stat.warn.version': '当前应用未配置uni统计版本，默认使用1.0版本；建议使用uni统计2.0版本 ，私有部署数据更安全，代码开源可定制。详情：https://uniapp.dcloud.io/uni-stat',
    'stat.warn.tip': '已开启 uni统计{version} 版本',
    'i18n.fallbackLocale.default': '当前应用未在 manifest.json 配置 fallbackLocale，默认使用：{locale}',
    'i18n.fallbackLocale.missing': '当前应用配置的 fallbackLocale 或 locale 为：{locale}，但 locale 目录缺少该语言文件',
    'easycom.conflict': 'easycom组件冲突：',
    'mp.component.args[0]': '{0}的第一个参数必须为静态字符串',
    'mp.component.args[1]': '{0}需要两个参数',
    'mp.360.unsupported': 'vue3暂不支持360小程序',
    'file.notfound': '{file} 文件不存在',
    'uts.ios.tips': '项目使用了uts插件，iOS平台uts插件代码修改后需要重新生成[自定义基座](https://uniapp.dcloud.net.cn/tutorial/run/run-app.html#customplayground)才能生效',
    'uts.android.compiler.server': '项目使用了uts插件，正在安装 uts Android 运行扩展...',
    'uts.ios.windows.tips': 'iOS手机在windows上使用标准基座真机运行无法使用uts插件，如需使用uts插件请提交云端打包自定义基座',
    'uts.ios.standard.tips': 'iOS手机在标准基座真机运行暂不支持uts插件，如需调用uts插件请使用自定义基座',
    'prompt.run.message': '运行方式：打开 {devtools}, 导入 {outputDir} 运行。',
    'prompt.run.devtools.app': 'HBuilderX',
    'prompt.run.devtools.app-harmony': 'HBuilderX',
    'prompt.run.devtools.mp-harmony': 'HBuilderX',
    'prompt.run.devtools.mp-alipay': '支付宝小程序开发者工具',
    'prompt.run.devtools.mp-baidu': '百度开发者工具',
    'prompt.run.devtools.mp-kuaishou': '快手开发者工具',
    'prompt.run.devtools.mp-lark': '飞书开发者工具',
    'prompt.run.devtools.mp-qq': 'QQ小程序开发者工具',
    'prompt.run.devtools.mp-toutiao': '抖音开发者工具',
    'prompt.run.devtools.mp-weixin': '微信开发者工具',
    'prompt.run.devtools.mp-jd': '京东开发者工具',
    'prompt.run.devtools.mp-xhs': '小红书开发者工具',
    'prompt.run.devtools.quickapp-webview': '快应用联盟开发者工具 | 华为快应用开发者工具',
    'prompt.run.devtools.quickapp-webview-huawei': '华为快应用开发者工具',
    'prompt.run.devtools.quickapp-webview-union': '快应用联盟开发者工具',
    'uvue.unsupported': 'uvue 暂不支持 {platform} 平台',
    // 重要：HBuilderX也会用到，如果调整了文案，需要同步调整HBuilderX的文案
    'uvue.dev.watching.end.empty': '本次代码变更，编译结果未发生变化，跳过同步手机端程序文件。',
    'uni_modules.import': '插件[{0}]仅支持 @/uni_modules/{1} 方式引入，不支持直接导入内部文件 {2}。',
    'pages.json.page.notfound': '页面"{pagePath}"不存在，请确保填写的页面路径不包含文件后缀，且必须与真实的文件路径大小写保持一致。',
    'pages.json.page.slash': '路径 "{pagePath}" 不能以 "/" 开头',
    'pages.json.tabbar.page.notfound': 'tabBar 中配置的页面 "{pagePath}" 未在 pages.json 中注册。',
};

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618856, function(require, module, exports) {

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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getUniXPagePaths = exports.isUniXPageFile = exports.normalizeUniAppXAppConfig = exports.normalizeUniAppXAppPagesJson = void 0;
const path_1 = __importDefault(require("path"));
const shared_1 = require("@vue/shared");
const json_1 = require("../json");
const pages_1 = require("../pages");
const utils_1 = require("../../utils");
const uniRoutes_1 = require("../app/pages/uniRoutes");
const uniConfig_1 = require("./uniConfig");
const preprocess_1 = require("../../preprocess");
const utils_2 = require("../utils");
__exportStar(require("./manifest"), exports);
function normalizeUniAppXAppPagesJson(jsonStr) {
    // 先条件编译
    jsonStr = (0, preprocess_1.preUVueJson)(jsonStr, 'pages.json');
    (0, utils_2.checkPagesJson)(jsonStr, process.env.UNI_INPUT_DIR);
    const pagesJson = {
        pages: [],
        globalStyle: {},
    };
    let userPagesJson = {
        pages: [],
        globalStyle: {},
    };
    try {
        // 此处不需要条件编译了
        userPagesJson = (0, json_1.parseJson)(jsonStr, false, 'pages.json');
    }
    catch (e) {
        console.error(`[vite] Error: pages.json parse failed.\n`, jsonStr, e);
    }
    // pages
    (0, pages_1.validatePages)(userPagesJson, jsonStr);
    userPagesJson.subPackages =
        userPagesJson.subPackages || userPagesJson.subpackages;
    // subPackages
    if (userPagesJson.subPackages) {
        userPagesJson.pages.push(...normalizeSubPackages(userPagesJson.subPackages));
    }
    pagesJson.pages = userPagesJson.pages;
    // pageStyle
    normalizePages(pagesJson.pages);
    // globalStyle
    pagesJson.globalStyle = normalizePageStyle(userPagesJson.globalStyle);
    // tabBar
    if (userPagesJson.tabBar) {
        pagesJson.tabBar = userPagesJson.tabBar;
    }
    // condition
    if (userPagesJson.condition) {
        pagesJson.condition = userPagesJson.condition;
    }
    // uniIdRouter
    if (userPagesJson.uniIdRouter) {
        pagesJson.uniIdRouter = userPagesJson.uniIdRouter;
    }
    // 是否应该用 process.env.UNI_UTS_PLATFORM
    (0, pages_1.filterPlatformPages)(process.env.UNI_PLATFORM, pagesJson);
    // 缓存页面列表
    pages_1.pagesCacheSet.clear();
    pagesJson.pages.forEach((page) => pages_1.pagesCacheSet.add(page.path));
    return pagesJson;
}
exports.normalizeUniAppXAppPagesJson = normalizeUniAppXAppPagesJson;
function normalizeSubPackages(subPackages) {
    const pages = [];
    if ((0, shared_1.isArray)(subPackages)) {
        subPackages.forEach(({ root, pages: subPages }) => {
            if (root && subPages.length) {
                subPages.forEach((subPage) => {
                    subPage.path = (0, utils_1.normalizePath)(path_1.default.join(root, subPage.path));
                    subPage.style = subPage.style;
                    pages.push(subPage);
                });
            }
        });
    }
    return pages;
}
function normalizePages(pages) {
    pages.forEach((page) => {
        page.style = normalizePageStyle(page.style);
    });
}
function normalizePageStyle(pageStyle) {
    if (pageStyle) {
        (0, shared_1.extend)(pageStyle, pageStyle['app']);
        (0, pages_1.removePlatformStyle)(pageStyle);
        return pageStyle;
    }
    return {};
}
/**
 * TODO 应该闭包，通过globalThis赋值？
 * @param pagesJson
 * @param manifestJson
 * @returns
 */
function normalizeUniAppXAppConfig(pagesJson, manifestJson) {
    const uniConfig = (0, uniConfig_1.normalizeAppXUniConfig)(pagesJson, manifestJson);
    const tabBar = uniConfig.tabBar;
    delete uniConfig.tabBar;
    let appConfigJs = `const __uniConfig = ${JSON.stringify(uniConfig)};
__uniConfig.getTabBarConfig = () =>  {return ${tabBar ? JSON.stringify(tabBar) : 'undefined'}};
__uniConfig.tabBar = __uniConfig.getTabBarConfig();
const __uniRoutes = ${(0, uniRoutes_1.normalizeAppUniRoutes)(pagesJson)}.map(uniRoute=>(uniRoute.meta.route=uniRoute.path,__uniConfig.pages.push(uniRoute.path),uniRoute.path='/'+uniRoute.path,uniRoute)).concat(typeof __uniSystemRoutes !== 'undefined' ? __uniSystemRoutes : []);

`;
    if (process.env.UNI_UTS_PLATFORM === 'app-harmony') {
        appConfigJs += `globalThis.__uniConfig = __uniConfig;
globalThis.__uniRoutes = __uniRoutes;`;
    }
    return appConfigJs;
}
exports.normalizeUniAppXAppConfig = normalizeUniAppXAppConfig;
function isUniXPageFile(source, importer, inputDir = process.env.UNI_INPUT_DIR) {
    if (source.startsWith('@/')) {
        return (0, pages_1.isUniPageFile)(source.slice(2), inputDir);
    }
    if (source.startsWith('.')) {
        return (0, pages_1.isUniPageFile)(path_1.default.resolve(path_1.default.dirname(importer), source), inputDir);
    }
    return false;
}
exports.isUniXPageFile = isUniXPageFile;
function getUniXPagePaths() {
    if (process.env.UNI_COMPILE_EXT_API_PAGE_PATHS) {
        return JSON.parse(process.env.UNI_COMPILE_EXT_API_PAGE_PATHS);
    }
    return Array.from(pages_1.pagesCacheSet);
}
exports.getUniXPagePaths = getUniXPagePaths;

}, function(modId) { var map = {"../json":1781362618810,"../app/pages/uniRoutes":1781362618825,"./uniConfig":1781362618859}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618859, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.normalizeAppXUniConfig = void 0;
const uniConfig_1 = require("../app/pages/uniConfig");
// app-config.js 内容
function normalizeAppXUniConfig(pagesJson, manifestJson) {
    const config = {
        pages: [],
        globalStyle: pagesJson.globalStyle,
        appname: manifestJson.name || '',
        compilerVersion: process.env.UNI_COMPILER_VERSION,
        ...(0, uniConfig_1.parseEntryPagePath)(pagesJson),
        tabBar: pagesJson.tabBar,
        fallbackLocale: manifestJson.fallbackLocale,
    };
    if (config.realEntryPagePath) {
        config.conditionUrl = config.entryPagePath;
        config.entryPagePath = config.realEntryPagePath;
    }
    // darkmode
    if (pagesJson.themeConfig) {
        config.themeConfig = pagesJson.themeConfig;
    }
    // TODO 待支持分包
    return config;
}
exports.normalizeAppXUniConfig = normalizeAppXUniConfig;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618865, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.addMiniProgramTemplateFilter = exports.clearMiniProgramTemplateFilter = exports.addMiniProgramTemplateFile = exports.clearMiniProgramTemplateFiles = exports.findMiniProgramTemplateFiles = void 0;
const path_1 = __importDefault(require("path"));
const uni_shared_1 = require("@dcloudio/uni-shared");
const utils_1 = require("../utils");
const templateFilesCache = new Map();
const templateFiltersCache = new Map();
function relativeFilterFilename(filename, filter) {
    if (!filter.src) {
        return '';
    }
    return ('./' +
        (0, utils_1.normalizeMiniProgramFilename)(path_1.default.relative(path_1.default.dirname(filename), filter.src)));
}
function findMiniProgramTemplateFiles(genFilter) {
    const files = Object.create(null);
    templateFilesCache.forEach((code, filename) => {
        if (!genFilter) {
            files[filename] = code;
        }
        else {
            const filters = getMiniProgramTemplateFilters(filename);
            if (filters && filters.length) {
                files[filename] =
                    filters
                        .map((filter) => genFilter(filter, relativeFilterFilename(filename, filter)))
                        .join(uni_shared_1.LINEFEED) +
                        uni_shared_1.LINEFEED +
                        code;
            }
            else {
                files[filename] = code;
            }
        }
    });
    return files;
}
exports.findMiniProgramTemplateFiles = findMiniProgramTemplateFiles;
function clearMiniProgramTemplateFiles() {
    templateFilesCache.clear();
}
exports.clearMiniProgramTemplateFiles = clearMiniProgramTemplateFiles;
function addMiniProgramTemplateFile(filename, code) {
    templateFilesCache.set(filename, code);
}
exports.addMiniProgramTemplateFile = addMiniProgramTemplateFile;
function getMiniProgramTemplateFilters(filename) {
    return templateFiltersCache.get(filename);
}
function clearMiniProgramTemplateFilter(filename) {
    templateFiltersCache.delete(filename);
}
exports.clearMiniProgramTemplateFilter = clearMiniProgramTemplateFilter;
function addMiniProgramTemplateFilter(filename, filter) {
    const filters = templateFiltersCache.get(filename);
    if (filters) {
        const filterIndex = filters.findIndex((f) => f.id === filter.id);
        if (filterIndex > -1) {
            filters.splice(filterIndex, 1, filter);
        }
        else {
            filters.push(filter);
        }
    }
    else {
        templateFiltersCache.set(filename, [filter]);
    }
}
exports.addMiniProgramTemplateFilter = addMiniProgramTemplateFilter;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618867, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.MP_PLUGIN_JSON_NAME = exports.COMPONENT_CUSTOM_HIDDEN_BIND = exports.COMPONENT_CUSTOM_HIDDEN = exports.COMPONENT_BIND_LINK = exports.COMPONENT_ON_LINK = void 0;
exports.COMPONENT_ON_LINK = 'onVI';
exports.COMPONENT_BIND_LINK = '__l';
exports.COMPONENT_CUSTOM_HIDDEN = 'data-c-h';
exports.COMPONENT_CUSTOM_HIDDEN_BIND = 'bind:-' + exports.COMPONENT_CUSTOM_HIDDEN;
exports.MP_PLUGIN_JSON_NAME = 'plugin.json';

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618868, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.createCopyPluginTarget = exports.copyMiniProgramThemeJson = exports.copyMiniProgramPluginJson = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const json_1 = require("../json/json");
const manifest_1 = require("../json/manifest");
exports.copyMiniProgramPluginJson = {
    src: ['plugin.json'],
    get dest() {
        return process.env.UNI_OUTPUT_DIR;
    },
    transform(source) {
        const pluginJson = (0, json_1.parseJson)(source.toString(), true, 'plugin.json');
        if (process.env.UNI_APP_X === 'true') {
            const pluginMainJs = pluginJson.main;
            if (pluginMainJs && pluginMainJs.endsWith('.uts')) {
                pluginJson.main = pluginMainJs.replace(/\.uts$/, '.js');
            }
        }
        return JSON.stringify(pluginJson, null, 2);
    },
};
const copyMiniProgramThemeJson = () => {
    if (!process.env.UNI_INPUT_DIR)
        return [];
    const manifestJson = (0, manifest_1.getPlatformManifestJsonOnce)();
    const themeLocation = manifestJson.themeLocation || 'theme.json';
    const hasThemeJson = fs_1.default.existsSync(path_1.default.resolve(process.env.UNI_INPUT_DIR, themeLocation));
    if (hasThemeJson) {
        return [
            {
                src: [(manifestJson.themeLocation = themeLocation)],
                get dest() {
                    return process.env.UNI_OUTPUT_DIR;
                },
                transform(source) {
                    return JSON.stringify((0, json_1.parseJson)(source.toString(), true, themeLocation), null, 2);
                },
            },
        ];
    }
    return [];
};
exports.copyMiniProgramThemeJson = copyMiniProgramThemeJson;
function createCopyPluginTarget(src) {
    return {
        src,
        get dest() {
            return process.env.UNI_OUTPUT_DIR;
        },
        transform(source, filename) {
            const extJson = (0, json_1.parseJson)(source.toString(), true, path_1.default.basename(filename));
            return JSON.stringify(extJson, null, 2);
        },
    };
}
exports.createCopyPluginTarget = createCopyPluginTarget;

}, function(modId) { var map = {"../json/json":1781362618810}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618870, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.transformDynamicImports = exports.parseScriptDescriptor = exports.parseTemplateDescriptor = exports.updateMiniProgramComponentsByMainFilename = exports.updateMiniProgramGlobalComponents = exports.updateMiniProgramComponentsByTemplateFilename = exports.updateMiniProgramComponentsByScriptFilename = exports.parseMainDescriptor = exports.getGlobalComponentSource = void 0;
const types_1 = require("@babel/types");
const estree_walker_1 = require("estree-walker");
const magic_string_1 = __importDefault(require("magic-string"));
const shared_1 = require("@vue/shared");
const uni_shared_1 = require("@dcloudio/uni-shared");
const messages_1 = require("../messages");
const constants_1 = require("../constants");
const utils_1 = require("../utils");
const utils_2 = require("../vite/utils");
const jsonFile_1 = require("../json/mp/jsonFile");
const mainDescriptors = new Map();
const scriptDescriptors = new Map();
const templateDescriptors = new Map();
// 存储全局组件名到完整源文件路径的映射
const globalComponentSourceMap = new Map();
function getGlobalComponentSource(componentName) {
    return globalComponentSourceMap.get(componentName);
}
exports.getGlobalComponentSource = getGlobalComponentSource;
function findImportTemplateSource(ast) {
    const importDeclaration = ast.body.find((node) => (0, types_1.isImportDeclaration)(node) &&
        node.source.value.includes('vue&type=template'));
    if (importDeclaration) {
        return importDeclaration.source.value;
    }
}
function findImportScriptSource(ast) {
    const importDeclaration = ast.body.find((node) => (0, types_1.isImportDeclaration)(node) && node.source.value.includes('vue&type=script'));
    if (importDeclaration) {
        return importDeclaration.source.value;
    }
}
async function resolveSource(filename, source, resolve) {
    if (!source) {
        return;
    }
    const resolveId = await resolve(source, filename);
    if (resolveId) {
        return resolveId.id;
    }
}
async function parseMainDescriptor(filename, ast, resolve) {
    const script = await resolveSource(filename, findImportScriptSource(ast), resolve);
    const template = await resolveSource(filename, findImportTemplateSource(ast), resolve);
    const imports = await parseVueComponentImports(filename, ast.body.filter((node) => (0, types_1.isImportDeclaration)(node)), resolve);
    if (!script) {
        // inline script
        await parseScriptDescriptor(filename, ast, { resolve, isExternal: false });
    }
    if (!template) {
        // inline template
        await parseTemplateDescriptor(filename, ast, { resolve, isExternal: false });
    }
    const descriptor = {
        imports,
        script: script ? (0, utils_2.parseVueRequest)(script).filename : filename,
        template: template ? (0, utils_2.parseVueRequest)(template).filename : filename,
    };
    mainDescriptors.set(filename, descriptor);
    return descriptor;
}
exports.parseMainDescriptor = parseMainDescriptor;
function updateMiniProgramComponentsByScriptFilename(scriptFilename, inputDir, normalizeComponentName) {
    const mainFilename = findMainFilenameByScriptFilename(scriptFilename);
    if (mainFilename) {
        updateMiniProgramComponentsByMainFilename(mainFilename, inputDir, normalizeComponentName);
    }
}
exports.updateMiniProgramComponentsByScriptFilename = updateMiniProgramComponentsByScriptFilename;
function updateMiniProgramComponentsByTemplateFilename(templateFilename, inputDir, normalizeComponentName) {
    const mainFilename = findMainFilenameByTemplateFilename(templateFilename);
    if (mainFilename) {
        updateMiniProgramComponentsByMainFilename(mainFilename, inputDir, normalizeComponentName);
    }
}
exports.updateMiniProgramComponentsByTemplateFilename = updateMiniProgramComponentsByTemplateFilename;
function findMainFilenameByScriptFilename(scriptFilename) {
    const keys = [...mainDescriptors.keys()];
    return keys.find((key) => mainDescriptors.get(key).script === scriptFilename);
}
function findMainFilenameByTemplateFilename(templateFilename) {
    const keys = [...mainDescriptors.keys()];
    return keys.find((key) => mainDescriptors.get(key).template === templateFilename);
}
async function updateMiniProgramGlobalComponents(filename, ast, { inputDir, resolve, normalizeComponentName, }) {
    const { bindingComponents, imports } = await parseGlobalDescriptor(filename, ast, resolve);
    // 存储全局组件名到完整源文件路径的映射
    imports.forEach(({ source: { value }, specifiers: [specifier] }) => {
        const { name } = specifier.local;
        if (!bindingComponents[name]) {
            return;
        }
        if (!globalComponentSourceMap.has(bindingComponents[name].tag)) {
            globalComponentSourceMap.set(bindingComponents[name].tag, value);
        }
    });
    (0, jsonFile_1.addMiniProgramUsingComponents)('app', createUsingComponents(bindingComponents, imports, inputDir, normalizeComponentName));
    return {
        imports,
    };
}
exports.updateMiniProgramGlobalComponents = updateMiniProgramGlobalComponents;
function createUsingComponents(bindingComponents, imports, inputDir, normalizeComponentName) {
    const usingComponents = {};
    imports.forEach(({ source: { value }, specifiers: [specifier] }) => {
        const { name } = specifier.local;
        if (!bindingComponents[name]) {
            return;
        }
        const componentName = normalizeComponentName((0, shared_1.hyphenate)(bindingComponents[name].tag));
        if (!usingComponents[componentName]) {
            usingComponents[componentName] = (0, uni_shared_1.addLeadingSlash)((0, utils_1.removeExt)((0, utils_1.normalizeMiniProgramFilename)(value, inputDir)));
        }
    });
    return usingComponents;
}
function updateMiniProgramComponentsByMainFilename(mainFilename, inputDir, normalizeComponentName) {
    const mainDescriptor = mainDescriptors.get(mainFilename);
    if (!mainDescriptor) {
        return;
    }
    const templateDescriptor = templateDescriptors.get(mainDescriptor.template);
    if (!templateDescriptor) {
        return;
    }
    const scriptDescriptor = scriptDescriptors.get(mainDescriptor.script);
    if (!scriptDescriptor) {
        return;
    }
    const bindingComponents = parseBindingComponents({
        ...templateDescriptor.bindingComponents,
        ...scriptDescriptor.setupBindingComponents,
    }, scriptDescriptor.bindingComponents);
    const imports = parseImports(mainDescriptor.imports, scriptDescriptor.imports, templateDescriptor.imports);
    (0, jsonFile_1.addMiniProgramUsingComponents)((0, utils_1.removeExt)((0, utils_1.normalizeMiniProgramFilename)(mainFilename, inputDir)), createUsingComponents(bindingComponents, imports, inputDir, normalizeComponentName));
}
exports.updateMiniProgramComponentsByMainFilename = updateMiniProgramComponentsByMainFilename;
function findBindingComponent(tag, bindingComponents) {
    return Object.keys(bindingComponents).find((name) => {
        const componentTag = bindingComponents[name].tag;
        const camelName = (0, shared_1.camelize)(componentTag);
        const PascalName = (0, shared_1.capitalize)(camelName);
        return tag === componentTag || tag === camelName || tag === PascalName;
    });
}
function normalizeComponentId(id) {
    // _unref(test) => test
    if (id.includes('_unref(')) {
        return id.replace('_unref(', '').replace(')', '');
    }
    // $setup["test"] => test
    if (id.includes('$setup[')) {
        return id.replace('$setup["', '').replace('"', '');
    }
    return id;
}
function parseBindingComponents(templateBindingComponents, scriptBindingComponents) {
    const bindingComponents = {};
    Object.keys(templateBindingComponents).forEach((id) => {
        bindingComponents[normalizeComponentId(id)] = templateBindingComponents[id];
    });
    Object.keys(scriptBindingComponents).forEach((id) => {
        const { tag } = scriptBindingComponents[id];
        const name = findBindingComponent(tag, templateBindingComponents);
        if (name) {
            bindingComponents[id] = bindingComponents[name];
        }
    });
    return bindingComponents;
}
function parseImports(mainImports, scriptImports, templateImports) {
    const imports = [...mainImports, ...templateImports, ...scriptImports];
    return imports;
}
/**
 * 解析 template
 * @param filename
 * @param code
 * @param ast
 * @param options
 * @returns
 */
async function parseTemplateDescriptor(filename, ast, options) {
    // 外置时查找所有 vue component import
    const imports = options.isExternal
        ? await parseVueComponentImports(filename, ast.body.filter((node) => (0, types_1.isImportDeclaration)(node)), options.resolve)
        : [];
    const descriptor = {
        bindingComponents: findBindingComponents(ast.body),
        imports,
    };
    templateDescriptors.set(filename, descriptor);
    return descriptor;
}
exports.parseTemplateDescriptor = parseTemplateDescriptor;
async function parseGlobalDescriptor(filename, ast, resolve) {
    // 外置时查找所有 vue component import
    const imports = (await parseVueComponentImports(filename, ast.body.filter((node) => (0, types_1.isImportDeclaration)(node)), resolve)).filter((item) => !(0, utils_1.isAppVue)((0, utils_2.cleanUrl)(item.source.value)));
    return {
        bindingComponents: parseGlobalComponents(ast),
        imports,
    };
}
/**
 * 解析 script
 * @param filename
 * @param code
 * @param ast
 * @param options
 * @returns
 */
async function parseScriptDescriptor(filename, ast, options) {
    // 外置时查找所有 vue component import
    const imports = options.isExternal
        ? await parseVueComponentImports(filename, ast.body.filter((node) => (0, types_1.isImportDeclaration)(node)), options.resolve)
        : [];
    const descriptor = {
        bindingComponents: parseComponents(ast),
        setupBindingComponents: findBindingComponents(ast.body),
        imports,
    };
    scriptDescriptors.set(filename, descriptor);
    return descriptor;
}
exports.parseScriptDescriptor = parseScriptDescriptor;
/**
 * 解析编译器生成的 bindingComponents
 * @param ast
 * @returns
 */
function findBindingComponents(ast) {
    const mapping = findUnpluginComponents(ast);
    for (const node of ast) {
        if (!(0, types_1.isVariableDeclaration)(node)) {
            continue;
        }
        const declarator = node.declarations[0];
        if ((0, types_1.isIdentifier)(declarator.id) &&
            declarator.id.name === constants_1.BINDING_COMPONENTS) {
            const bindingComponents = JSON.parse(declarator.init.value);
            return Object.keys(bindingComponents).reduce((bindings, tag) => {
                const { name, type } = bindingComponents[tag];
                bindings[mapping[name] || name] = {
                    tag,
                    type: type,
                };
                return bindings;
            }, {});
        }
    }
    return {};
}
/**
 * 兼容：unplugin_components
 * https://github.com/dcloudio/uni-app/issues/3057
 * @param ast
 * @returns
 */
function findUnpluginComponents(ast) {
    const res = Object.create(null);
    // if(!Array){}
    const ifStatement = ast.find((statement) => (0, types_1.isIfStatement)(statement) &&
        (0, types_1.isUnaryExpression)(statement.test) &&
        statement.test.operator === '!' &&
        (0, types_1.isIdentifier)(statement.test.argument) &&
        statement.test.argument.name === 'Array');
    if (!ifStatement) {
        return res;
    }
    if (!(0, types_1.isBlockStatement)(ifStatement.consequent)) {
        return res;
    }
    for (const node of ifStatement.consequent.body) {
        if (!(0, types_1.isVariableDeclaration)(node)) {
            continue;
        }
        const { id, init } = node.declarations[0];
        if ((0, types_1.isIdentifier)(id) &&
            (0, types_1.isIdentifier)(init) &&
            init.name.includes('unplugin_components')) {
            res[id.name] = init.name;
        }
    }
    return res;
}
/**
 * 查找全局组件定义：app.component('component-a',{})
 * @param ast
 * @returns
 */
function parseGlobalComponents(ast) {
    const bindingComponents = {};
    estree_walker_1.walk(ast, {
        enter(child) {
            if (!(0, types_1.isCallExpression)(child)) {
                return;
            }
            const { callee } = child;
            // .component
            if (!(0, types_1.isMemberExpression)(callee) ||
                !(0, types_1.isIdentifier)(callee.property) ||
                callee.property.name !== 'component') {
                return;
            }
            // .component('component-a',{})
            const args = child.arguments;
            if (args.length !== 2) {
                return;
            }
            const [name, value] = args;
            if (!(0, types_1.isStringLiteral)(name)) {
                return console.warn(messages_1.M['mp.component.args[0]'].replace('{0}', 'app.component'));
            }
            if (!(0, types_1.isIdentifier)(value)) {
                return console.warn(messages_1.M['mp.component.args[1]'].replace('{0}', 'app.component'));
            }
            bindingComponents[value.name] = {
                tag: name.value,
                type: 'unknown',
            };
        },
    });
    return bindingComponents;
}
/**
 * 从 components 中查找定义的组件
 * @param ast
 * @param bindingComponents
 */
function parseComponents(ast) {
    const bindingComponents = {};
    estree_walker_1.walk(ast, {
        enter(child) {
            if (!(0, types_1.isObjectExpression)(child)) {
                return;
            }
            const componentsProp = child.properties.find((prop) => (0, types_1.isObjectProperty)(prop) &&
                (0, types_1.isIdentifier)(prop.key) &&
                prop.key.name === 'components');
            if (!componentsProp) {
                return;
            }
            const componentsExpr = componentsProp.value;
            if (!(0, types_1.isObjectExpression)(componentsExpr)) {
                return;
            }
            componentsExpr.properties.forEach((prop) => {
                if (!(0, types_1.isObjectProperty)(prop)) {
                    return;
                }
                if (!(0, types_1.isIdentifier)(prop.key) && !(0, types_1.isStringLiteral)(prop.key)) {
                    return;
                }
                if (!(0, types_1.isIdentifier)(prop.value)) {
                    return;
                }
                bindingComponents[prop.value.name] = {
                    tag: (0, types_1.isIdentifier)(prop.key) ? prop.key.name : prop.key.value,
                    type: 'unknown',
                };
            });
        },
    });
    return bindingComponents;
}
/**
 * vue component imports
 * @param filename
 * @param imports
 * @param resolve
 * @returns
 */
async function parseVueComponentImports(importer, imports, resolve) {
    const vueComponentImports = [];
    for (let i = 0; i < imports.length; i++) {
        const { source } = imports[i];
        if ((0, utils_2.parseVueRequest)(source.value).query.vue) {
            continue;
        }
        const resolveId = await resolve(source.value, importer);
        if (!resolveId) {
            continue;
        }
        const { filename } = (0, utils_2.parseVueRequest)(resolveId.id);
        if (constants_1.EXTNAME_VUE_RE.test(filename)) {
            source.value = resolveId.id;
            vueComponentImports.push(imports[i]);
        }
    }
    return vueComponentImports;
}
/**
 * static import => dynamic import
 * @param code
 * @param imports
 * @param dynamicImport
 * @returns
 */
async function transformDynamicImports(code, imports, { id, sourceMap, dynamicImport, }) {
    if (!imports.length) {
        return {
            code,
            map: null,
        };
    }
    const s = new magic_string_1.default(code);
    for (let i = 0; i < imports.length; i++) {
        const { start, end, specifiers: [specifier], source, } = imports[i];
        s.overwrite(start, end, dynamicImport(specifier.local.name, source.value) + ';');
    }
    return {
        code: s.toString(),
        map: null,
    };
}
exports.transformDynamicImports = transformDynamicImports;

}, function(modId) { var map = {"../messages":1781362618844,"../constants":1781362618871,"../vite/utils":1781362618873}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618871, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.TEXT_STYLE = exports.DEFAULT_ASSETS_RE = exports.KNOWN_ASSET_TYPES = exports.COMMON_EXCLUDE = exports.X_BASE_COMPONENTS_STYLE_PATH = exports.BASE_COMPONENTS_STYLE_PATH = exports.H5_COMPONENTS_STYLE_PATH = exports.H5_FRAMEWORK_STYLE_PATH = exports.H5_API_STYLE_PATH = exports.X_PAGE_EXTNAME_APP = exports.X_PAGE_EXTNAME = exports.PAGE_EXTNAME = exports.PAGE_EXTNAME_APP = exports.BINDING_COMPONENTS = exports.APP_CONFIG_SERVICE = exports.APP_CONFIG = exports.APP_SERVICE_FILENAME = exports.ASSETS_INLINE_LIMIT = exports.JSON_JS_MAP = exports.MANIFEST_JSON_UTS = exports.MANIFEST_JSON_JS = exports.PAGES_JSON_UTS = exports.PAGES_JSON_JS = exports.uni_app_x_extensions = exports.extensions = exports.SPECIAL_CHARS = exports.DEFAULT_APPID = exports.EXTNAME_TS_RE = exports.EXTNAME_JS_RE = exports.EXTNAME_VUE_RE = exports.EXTNAME_VUE_TEMPLATE = exports.X_EXTNAME_VUE = exports.EXTNAME_VUE = exports.EXTNAME_TS = exports.EXTNAME_JS = exports.PUBLIC_DIR = void 0;
exports.PUBLIC_DIR = 'static';
exports.EXTNAME_JS = ['.js', '.ts', '.jsx', '.tsx', '.uts'];
exports.EXTNAME_TS = ['.ts', '.tsx'];
exports.EXTNAME_VUE = ['.vue', '.nvue', '.uvue'];
exports.X_EXTNAME_VUE = ['.uvue', '.vue'];
exports.EXTNAME_VUE_TEMPLATE = ['.vue', '.nvue', '.uvue', '.jsx', '.tsx'];
exports.EXTNAME_VUE_RE = /\.(vue|nvue|uvue)$/;
exports.EXTNAME_JS_RE = /\.(js|jsx|ts|uts|tsx|mjs)$/;
exports.EXTNAME_TS_RE = /\.tsx?$/;
exports.DEFAULT_APPID = '__UNI__uniappx';
exports.SPECIAL_CHARS = {
    WARN_BLOCK: '\uFEFF', // 警告块前后标识
    ERROR_BLOCK: '\u2060', // 错误块前后标识
};
const COMMON_EXTENSIONS = [
    '.uts',
    '.mjs',
    '.js',
    '.ts',
    '.jsx',
    '.tsx',
    '.json',
];
exports.extensions = COMMON_EXTENSIONS.concat(exports.EXTNAME_VUE);
exports.uni_app_x_extensions = COMMON_EXTENSIONS.concat(['.uvue', '.vue']);
exports.PAGES_JSON_JS = 'pages-json-js';
exports.PAGES_JSON_UTS = 'pages-json-uts';
exports.MANIFEST_JSON_JS = 'manifest-json-js';
exports.MANIFEST_JSON_UTS = 'manifest-json-uts';
exports.JSON_JS_MAP = {
    'pages.json': exports.PAGES_JSON_JS,
    'manifest.json': exports.MANIFEST_JSON_JS,
};
exports.ASSETS_INLINE_LIMIT = 40 * 1024;
exports.APP_SERVICE_FILENAME = 'app-service.js';
exports.APP_CONFIG = 'app-config.js';
exports.APP_CONFIG_SERVICE = 'app-config-service.js';
exports.BINDING_COMPONENTS = '__BINDING_COMPONENTS__';
// APP 平台解析页面后缀的优先级
exports.PAGE_EXTNAME_APP = ['.nvue', '.vue', '.tsx', '.jsx', '.js'];
// 其他平台解析页面后缀的优先级
exports.PAGE_EXTNAME = ['.vue', '.nvue', '.tsx', '.jsx', '.js'];
exports.X_PAGE_EXTNAME = ['.uvue', '.vue', '.tsx', '.jsx', '.js'];
exports.X_PAGE_EXTNAME_APP = ['.uvue', '.tsx', '.jsx', '.js'];
exports.H5_API_STYLE_PATH = '@dcloudio/uni-h5/style/api/';
exports.H5_FRAMEWORK_STYLE_PATH = '@dcloudio/uni-h5/style/framework/';
exports.H5_COMPONENTS_STYLE_PATH = '@dcloudio/uni-h5/style/';
exports.BASE_COMPONENTS_STYLE_PATH = '@dcloudio/uni-components/style/';
exports.X_BASE_COMPONENTS_STYLE_PATH = '@dcloudio/uni-components/style-x/';
exports.COMMON_EXCLUDE = [
    /\/pages\.json\.js$/,
    /\/manifest\.json\.js$/,
    /\/vite\//,
    /\/@vue\//,
    /\/vue-router\//,
    /\/vuex\//,
    /\/vue-i18n\//,
    /\/@dcloudio\/uni-h5-vue/,
    /\/@dcloudio\/uni-shared/,
];
exports.KNOWN_ASSET_TYPES = [
    // images
    'png',
    'jpe?g',
    'gif',
    'svg',
    'ico',
    'webp',
    'avif',
    // media
    'mp4',
    'webm',
    'ogg',
    'mp3',
    'wav',
    'flac',
    'aac',
    // fonts
    'woff2?',
    'eot',
    'ttf',
    'otf',
    // other
    'pdf',
    'txt',
];
exports.DEFAULT_ASSETS_RE = new RegExp(`\\.(` + exports.KNOWN_ASSET_TYPES.join('|') + `)(\\?.*)?$`);
exports.TEXT_STYLE = ['black', 'white'];

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618873, function(require, module, exports) {

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
exports.isCombineBuiltInCss = exports.buildInCssSet = void 0;
const utils_1 = require("../../utils");
__exportStar(require("./ast"), exports);
__exportStar(require("./url"), exports);
__exportStar(require("./plugin"), exports);
__exportStar(require("./utils"), exports);
// 内置组件css列表，h5平台需要合并进去首页css中
exports.buildInCssSet = new Set();
function isCombineBuiltInCss(config) {
    if (!(0, utils_1.isNormalCompileTarget)()) {
        return false;
    }
    return config.command === 'build' && config.build.cssCodeSplit;
}
exports.isCombineBuiltInCss = isCombineBuiltInCss;

}, function(modId) { var map = {"./ast":1781362618875,"./url":1781362618876,"./plugin":1781362618877,"./utils":1781362618880}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618875, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.isCommentNode = exports.isCompoundExpressionNode = exports.isSimpleExpressionNode = exports.isDirectiveNode = exports.isAttributeNode = exports.isPlainElementNode = exports.isElementNode = exports.parseVue = exports.createCallExpression = exports.createIdentifier = exports.createLiteral = exports.isReference = exports.isExportSpecifier = exports.isMethodDefinition = exports.isMemberExpression = exports.isCallExpression = exports.isAssignmentExpression = exports.isIdentifier = exports.isProperty = void 0;
const compiler_core_1 = require("@vue/compiler-core");
const compiler_dom_1 = require("@vue/compiler-dom");
const isProperty = (node) => node.type === 'Property';
exports.isProperty = isProperty;
const isIdentifier = (node) => node.type === 'Identifier';
exports.isIdentifier = isIdentifier;
const isAssignmentExpression = (node) => node.type === 'AssignmentExpression';
exports.isAssignmentExpression = isAssignmentExpression;
const isCallExpression = (node) => node.type === 'CallExpression';
exports.isCallExpression = isCallExpression;
const isMemberExpression = (node) => node.type === 'MemberExpression';
exports.isMemberExpression = isMemberExpression;
const isMethodDefinition = (node) => node.type === 'MethodDefinition';
exports.isMethodDefinition = isMethodDefinition;
const isExportSpecifier = (node) => node.type === 'ExportSpecifier';
exports.isExportSpecifier = isExportSpecifier;
const isReference = (node, parent) => {
    if ((0, exports.isMemberExpression)(node)) {
        return !node.computed && (0, exports.isReference)(node.object, node);
    }
    if ((0, exports.isIdentifier)(node)) {
        if ((0, exports.isMemberExpression)(parent))
            return parent.computed || node === parent.object;
        // `bar` in { bar: foo }
        if ((0, exports.isProperty)(parent) && node !== parent.value)
            return false;
        // `bar` in `class Foo { bar () {...} }`
        if ((0, exports.isMethodDefinition)(parent))
            return false;
        // `bar` in `export { foo as bar }`
        if ((0, exports.isExportSpecifier)(parent) && node !== parent.local)
            return false;
        return true;
    }
    return false;
};
exports.isReference = isReference;
function createLiteral(value) {
    return {
        type: 'Literal',
        value,
        raw: `'${value}'`,
    };
}
exports.createLiteral = createLiteral;
function createIdentifier(name) {
    return {
        type: 'Identifier',
        name,
    };
}
exports.createIdentifier = createIdentifier;
function createCallExpression(callee, args) {
    return {
        type: 'CallExpression',
        callee,
        arguments: args,
    };
}
exports.createCallExpression = createCallExpression;
function parseVue(code, errors) {
    return (0, compiler_dom_1.parse)(code, {
        isNativeTag: () => true,
        isPreTag: () => true,
        parseMode: 'sfc',
        onError: (e) => {
            errors.push(e);
        },
    });
}
exports.parseVue = parseVue;
function isElementNode(node) {
    return node.type === compiler_core_1.NodeTypes.ELEMENT;
}
exports.isElementNode = isElementNode;
function isPlainElementNode(node) {
    return isElementNode(node) && node.tagType === compiler_core_1.ElementTypes.ELEMENT;
}
exports.isPlainElementNode = isPlainElementNode;
function isAttributeNode(node) {
    return node.type === compiler_core_1.NodeTypes.ATTRIBUTE;
}
exports.isAttributeNode = isAttributeNode;
function isDirectiveNode(node) {
    return node.type === compiler_core_1.NodeTypes.DIRECTIVE;
}
exports.isDirectiveNode = isDirectiveNode;
function isSimpleExpressionNode(node) {
    return node.type === compiler_core_1.NodeTypes.SIMPLE_EXPRESSION;
}
exports.isSimpleExpressionNode = isSimpleExpressionNode;
function isCompoundExpressionNode(node) {
    return node.type === compiler_core_1.NodeTypes.COMPOUND_EXPRESSION;
}
exports.isCompoundExpressionNode = isCompoundExpressionNode;
function isCommentNode(node) {
    return node.type === compiler_core_1.NodeTypes.COMMENT;
}
exports.isCommentNode = isCommentNode;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618876, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.isJsFile = exports.cleanUrl = exports.hashRE = exports.queryRE = exports.isInternalRequest = exports.ENV_PUBLIC_PATH = exports.CLIENT_PUBLIC_PATH = exports.VALID_ID_PREFIX = exports.FS_PREFIX = exports.isImportRequest = exports.parseVueRequest = void 0;
const shared_1 = require("@vue/shared");
const path_1 = __importDefault(require("path"));
const constants_1 = require("../../constants");
function parseVueRequest(id) {
    const [filename, rawQuery] = id.split(`?`, 2);
    const query = Object.fromEntries(new URLSearchParams(rawQuery));
    if (query.vue != null) {
        query.vue = true;
    }
    if (query.src != null) {
        query.src = true;
    }
    if (query.index != null) {
        query.index = Number(query.index);
    }
    if (query.raw != null) {
        query.raw = true;
    }
    return {
        filename,
        query,
    };
}
exports.parseVueRequest = parseVueRequest;
const importQueryRE = /(\?|&)import=?(?:&|$)/;
const isImportRequest = (url) => importQueryRE.test(url);
exports.isImportRequest = isImportRequest;
/**
 * Prefix for resolved fs paths, since windows paths may not be valid as URLs.
 */
exports.FS_PREFIX = `/@fs/`;
/**
 * Prefix for resolved Ids that are not valid browser import specifiers
 */
exports.VALID_ID_PREFIX = `/@id/`;
exports.CLIENT_PUBLIC_PATH = `/@vite/client`;
exports.ENV_PUBLIC_PATH = `/@vite/env`;
const internalPrefixes = [
    exports.FS_PREFIX,
    exports.VALID_ID_PREFIX,
    exports.CLIENT_PUBLIC_PATH,
    exports.ENV_PUBLIC_PATH,
];
const InternalPrefixRE = new RegExp(`^(?:${internalPrefixes.join('|')})`);
const isInternalRequest = (url) => InternalPrefixRE.test(url);
exports.isInternalRequest = isInternalRequest;
exports.queryRE = /\?.*$/;
exports.hashRE = /#.*$/;
const cleanUrl = (url) => url.replace(exports.hashRE, '').replace(exports.queryRE, '');
exports.cleanUrl = cleanUrl;
function isJsFile(id) {
    const { filename, query } = parseVueRequest(id);
    const isJs = constants_1.EXTNAME_JS_RE.test(filename);
    if (isJs) {
        return true;
    }
    const isVueJs = constants_1.EXTNAME_VUE.includes(path_1.default.extname(filename)) &&
        (!query.vue ||
            query.setup ||
            (0, shared_1.hasOwn)(query, 'lang.ts') ||
            (0, shared_1.hasOwn)(query, 'lang.js') ||
            (0, shared_1.hasOwn)(query, 'lang.uts'));
    if (isVueJs) {
        return true;
    }
    return false;
}
exports.isJsFile = isJsFile;

}, function(modId) { var map = {"../../constants":1781362618871}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618877, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.insertBeforePlugin = exports.removePlugins = exports.replacePlugins = exports.injectCssPostPlugin = exports.injectCssPlugin = exports.injectAssetPlugin = void 0;
const shared_1 = require("@vue/shared");
const asset_1 = require("../plugins/vitejs/plugins/asset");
const css_1 = require("../plugins/vitejs/plugins/css");
function injectAssetPlugin(config, options) {
    replacePlugins([(0, asset_1.assetPlugin)(config, options)], config);
}
exports.injectAssetPlugin = injectAssetPlugin;
function injectCssPlugin(config, options) {
    replacePlugins([
        (0, css_1.cssPlugin)(config, {
            isAndroidX: false,
            ...options,
        }),
    ], config);
}
exports.injectCssPlugin = injectCssPlugin;
function injectCssPostPlugin(config, newCssPostPlugin) {
    const oldCssPostPlugin = config.plugins.find((p) => p.name === newCssPostPlugin.name);
    // 直接覆盖原有方法，不能删除，替换，因为 unocss 在 pre 阶段已经获取到了旧的 css-post 插件对象
    if (oldCssPostPlugin) {
        (0, shared_1.extend)(oldCssPostPlugin, newCssPostPlugin);
    }
}
exports.injectCssPostPlugin = injectCssPostPlugin;
function replacePlugins(plugins, config) {
    plugins.forEach((plugin) => {
        const index = config.plugins.findIndex((p) => p.name === plugin.name);
        if (index > -1) {
            ;
            config.plugins.splice(index, 1, plugin);
        }
    });
}
exports.replacePlugins = replacePlugins;
function removePlugins(plugins, config) {
    if (!(0, shared_1.isArray)(plugins)) {
        plugins = [plugins];
    }
    plugins.forEach((name) => {
        const index = config.plugins.findIndex((p) => p.name === name);
        if (index > -1) {
            ;
            config.plugins.splice(index, 1);
        }
    });
}
exports.removePlugins = removePlugins;
function insertBeforePlugin(plugin, before, config) {
    const index = config.plugins.findIndex((p) => p.name === before);
    if (index > -1) {
        ;
        config.plugins.splice(index, 0, plugin);
    }
}
exports.insertBeforePlugin = insertBeforePlugin;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618880, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.generateCodeFrameColumns = exports.createRollupError = exports.isSsr = exports.isInHybridNVue = exports.withSourcemap = void 0;
const shared_1 = require("@vue/shared");
const code_frame_1 = require("@babel/code-frame");
const utils_1 = require("../plugins/vitejs/utils");
const utils_2 = require("../../utils");
function withSourcemap(config) {
    if (!process.env.UNI_APP_SOURCEMAP) {
        if (config.build && (0, shared_1.hasOwn)(config.build, 'sourcemap')) {
            if (!!config.build.sourcemap) {
                process.env.UNI_APP_SOURCEMAP = 'true';
            }
            else {
                // vite 的 build 模式默认是false，而非web端的dev也是用build模式，所以不能这样判断
                // process.env.UNI_APP_SOURCEMAP = 'false'
            }
        }
    }
    return (0, utils_2.enableSourceMap)();
}
exports.withSourcemap = withSourcemap;
function isInHybridNVue(config) {
    return config.nvue && process.env.UNI_RENDERER !== 'native';
}
exports.isInHybridNVue = isInHybridNVue;
function isSsr(command, config) {
    if (command === 'serve') {
        return !!(config.server && config.server.middlewareMode);
    }
    if (command === 'build') {
        return !!(config.build && config.build.ssr);
    }
    return false;
}
exports.isSsr = isSsr;
function createRollupError(plugin, id, error, source) {
    const { message, name, stack } = error;
    const rollupError = (0, shared_1.extend)(new Error(message), {
        id,
        plugin,
        name,
        stack,
    });
    if ('code' in error && error.loc) {
        rollupError.loc = {
            file: id,
            line: error.loc.start.line,
            column: error.loc.start.column,
        };
        if (source && source.length > 0) {
            if ('offsetStart' in error && 'offsetEnd' in error) {
                rollupError.frame = (0, code_frame_1.codeFrameColumns)(source, (0, utils_1.offsetToStartAndEnd)(source, error.offsetStart, error.offsetEnd));
            }
            else {
                rollupError.frame = (0, code_frame_1.codeFrameColumns)(source, error.loc);
            }
        }
    }
    if (id) {
        // 指定了id后，不让后续的rollup重写
        Object.defineProperty(rollupError, 'id', {
            get() {
                return id;
            },
            set(_v) { },
        });
    }
    return rollupError;
}
exports.createRollupError = createRollupError;
exports.generateCodeFrameColumns = code_frame_1.codeFrameColumns;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618884, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.createDefaultSFCStyleBlock = exports.findMiniProgramComponentStyleIsolation = exports.updateMiniProgramComponentStyleIsolation = exports.parseStyleIsolation = exports.parseExternalClasses = exports.updateMiniProgramComponentExternalClasses = exports.findMiniProgramComponentExternalClasses = exports.hasExternalClasses = exports.clearPageExternalClasses = exports.addPageExternalClasses = exports.updatePageExternalClasses = exports.findPageExternalClasses = void 0;
const types_1 = require("@babel/types");
const estree_walker_1 = require("estree-walker");
const utils_1 = require("../utils");
const externalClassesCache = new Map();
const pageStyleIsolationCache = new Map();
const pageExternalClassesCache = new Map();
function findPageExternalClasses(filename) {
    return pageExternalClassesCache.get((0, utils_1.normalizePath)(filename));
}
exports.findPageExternalClasses = findPageExternalClasses;
function updatePageExternalClasses(filename, info) {
    pageExternalClassesCache.set((0, utils_1.normalizePath)(filename), info);
}
exports.updatePageExternalClasses = updatePageExternalClasses;
function addPageExternalClasses(filename, staticClasses, hasDynamic, hasAppAndPageStyle) {
    const normalizedFilename = (0, utils_1.normalizePath)(filename);
    let info = pageExternalClassesCache.get(normalizedFilename);
    if (!info) {
        info = { staticClasses: new Set(), hasDynamic: false };
        pageExternalClassesCache.set(normalizedFilename, info);
    }
    staticClasses.forEach((cls) => info.staticClasses.add(cls));
    if (hasDynamic) {
        info.hasDynamic = true;
    }
    if (hasAppAndPageStyle) {
        info.hasAppAndPageStyle = true;
    }
}
exports.addPageExternalClasses = addPageExternalClasses;
function clearPageExternalClasses(filename) {
    pageExternalClassesCache.delete((0, utils_1.normalizePath)(filename));
}
exports.clearPageExternalClasses = clearPageExternalClasses;
function hasExternalClasses(code) {
    return code.includes('externalClasses');
}
exports.hasExternalClasses = hasExternalClasses;
function findMiniProgramComponentExternalClasses(filename) {
    return externalClassesCache.get((0, utils_1.normalizePath)(filename));
}
exports.findMiniProgramComponentExternalClasses = findMiniProgramComponentExternalClasses;
function updateMiniProgramComponentExternalClasses(filename, value) {
    externalClassesCache.set((0, utils_1.normalizePath)(filename), value);
}
exports.updateMiniProgramComponentExternalClasses = updateMiniProgramComponentExternalClasses;
function parseExternalClasses(ast) {
    const classes = [];
    estree_walker_1.walk(ast, {
        enter(child, parent) {
            if (!(0, types_1.isIdentifier)(child) || child.name !== 'externalClasses') {
                return;
            }
            // export default { externalClasses: ['my-class'] }
            if (!(0, types_1.isObjectProperty)(parent)) {
                return;
            }
            if (!(0, types_1.isArrayExpression)(parent.value)) {
                return;
            }
            parent.value.elements.forEach((element) => {
                if ((0, types_1.isStringLiteral)(element)) {
                    classes.push(element.value);
                }
            });
        },
    });
    return classes;
}
exports.parseExternalClasses = parseExternalClasses;
function parseStyleIsolation(ast) {
    let styleIsolationValue = '';
    estree_walker_1.walk(ast, {
        enter(child, parent) {
            if (!(0, types_1.isIdentifier)(child) || child.name !== 'styleIsolation') {
                return;
            }
            if (!(0, types_1.isObjectProperty)(parent)) {
                return;
            }
            if (!(0, types_1.isStringLiteral)(parent.value)) {
                return;
            }
            if (parent.value.value === 'app' ||
                parent.value.value === 'app-and-page' ||
                parent.value.value === 'isolated') {
                styleIsolationValue = parent.value.value;
            }
            return parent.value.value;
        },
    });
    return styleIsolationValue;
}
exports.parseStyleIsolation = parseStyleIsolation;
/**
 * 目前只有小程序平台才会走这个逻辑
 * @param pagePahth
 * @param value
 * @param isPage
 */
function updateMiniProgramComponentStyleIsolation(pagePahth, value, isPage = false) {
    pageStyleIsolationCache.set((0, utils_1.normalizePath)(pagePahth), {
        styleIsolation: value,
        isPage,
    });
}
exports.updateMiniProgramComponentStyleIsolation = updateMiniProgramComponentStyleIsolation;
function findMiniProgramComponentStyleIsolation(pagePahth) {
    return pageStyleIsolationCache.get((0, utils_1.normalizePath)(pagePahth));
}
exports.findMiniProgramComponentStyleIsolation = findMiniProgramComponentStyleIsolation;
function createDefaultSFCStyleBlock(source) {
    const offset = source.length;
    return {
        type: 'style',
        content: '',
        attrs: {},
        loc: {
            source: '',
            start: { line: 1, column: 1, offset },
            end: { line: 1, column: 1, offset },
        },
    };
}
exports.createDefaultSFCStyleBlock = createDefaultSFCStyleBlock;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618886, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.decodeBase64Url = exports.encodeBase64Url = void 0;
const base64url_1 = __importDefault(require("base64url"));
function encodeBase64Url(str) {
    return base64url_1.default.encode(str);
}
exports.encodeBase64Url = encodeBase64Url;
function decodeBase64Url(str) {
    return base64url_1.default.decode(str);
}
exports.decodeBase64Url = decodeBase64Url;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618887, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.initAppProvide = exports.initDefine = void 0;
var define_1 = require("./define");
Object.defineProperty(exports, "initDefine", { enumerable: true, get: function () { return define_1.initDefine; } });
var provide_1 = require("./provide");
Object.defineProperty(exports, "initAppProvide", { enumerable: true, get: function () { return provide_1.initAppProvide; } });

}, function(modId) { var map = {"./define":1781362618888,"./provide":1781362618895}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618888, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.initDefine = void 0;
const utils_1 = require("../utils");
const env_1 = require("../hbx/env");
const json_1 = require("../json");
function initDefine(stringifyBoolean = false) {
    const manifestJson = (0, json_1.parseManifestJsonOnce)(process.env.UNI_INPUT_DIR);
    const platformManifestJson = (0, json_1.getPlatformManifestJsonOnce)();
    const isRunByHBuilderX = (0, env_1.runByHBuilderX)();
    const isDebug = !!manifestJson.debug;
    const isX = process.env.UNI_APP_X === 'true';
    const isNewStyleIsolation = process.env.UNI_APP_STYLE_ISOLATION_VERSION === '2';
    // 目前仅微信小程序支持^穿透
    const isNewStyleIsolationUpArrow = isNewStyleIsolation && process.env.UNI_PLATFORM === 'mp-weixin';
    const isMP = process.env.UNI_PLATFORM && process.env.UNI_PLATFORM.startsWith('mp-');
    process.env['UNI_APP_ID'] = manifestJson.appid;
    const mpXDefine = {
        __UNI_FEATURE_VIRTUAL_HOST__: isX && isMP && platformManifestJson.enableVirtualHost !== false,
    };
    const styleIsolation = stringifyBoolean
        ? JSON.stringify(isNewStyleIsolation)
        : isNewStyleIsolation;
    return {
        ...initCustomDefine(),
        'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
        'process.env.UNI_DEBUG': stringifyBoolean
            ? JSON.stringify(isDebug)
            : isDebug,
        'process.env.UNI_APP_ID': JSON.stringify(manifestJson.appid || ''),
        'process.env.UNI_APP_NAME': JSON.stringify(manifestJson.name || ''),
        'process.env.UNI_APP_VERSION_NAME': JSON.stringify(manifestJson.versionName || ''),
        'process.env.UNI_APP_VERSION_CODE': JSON.stringify(manifestJson.versionCode || ''),
        'process.env.UNI_PLATFORM': JSON.stringify(process.env.UNI_PLATFORM),
        'process.env.UNI_SUB_PLATFORM': JSON.stringify(process.env.UNI_SUB_PLATFORM || ''),
        'process.env.UNI_MP_PLUGIN': JSON.stringify(process.env.UNI_MP_PLUGIN || ''),
        'process.env.UNI_SUBPACKAGE': JSON.stringify(process.env.UNI_SUBPACKAGE || ''),
        'process.env.UNI_COMPILER_VERSION': JSON.stringify(process.env.UNI_COMPILER_VERSION || ''),
        'process.env.RUN_BY_HBUILDERX': stringifyBoolean
            ? JSON.stringify(isRunByHBuilderX)
            : isRunByHBuilderX,
        'process.env.UNI_AUTOMATOR_WS_ENDPOINT': JSON.stringify(process.env.UNI_AUTOMATOR_WS_ENDPOINT || ''),
        'process.env.UNI_AUTOMATOR_APP_WEBVIEW_SRC': JSON.stringify(process.env.UNI_AUTOMATOR_APP_WEBVIEW_SRC || ''),
        'process.env.UNI_CLOUD_PROVIDER': JSON.stringify(process.env.UNI_CLOUD_PROVIDER || ''),
        'process.env.UNICLOUD_DEBUG': JSON.stringify(process.env.UNICLOUD_DEBUG || ''),
        // 兼容旧版本
        'process.env.VUE_APP_PLATFORM': JSON.stringify(process.env.UNI_PLATFORM || ''),
        'process.env.VUE_APP_DARK_MODE': JSON.stringify(platformManifestJson.darkmode || false),
        __UNI_PRELOAD_SHADOW_IMAGE__: JSON.stringify(process.env.UNI_PLATFORM === 'mp-weixin' ? (0, utils_1.getShadowImagePath)('grey') : ''),
        ...mpXDefine,
        __X_STYLE_ISOLATION__: styleIsolation,
        __X_STYLE_ISOLATION_UP_ARROW__: stringifyBoolean
            ? JSON.stringify(isNewStyleIsolationUpArrow)
            : isNewStyleIsolationUpArrow,
        // 下边这个主要是为web服务，因为ssr目前只能识别process.env中的内容
        'process.env.UNI_APP_X_NEW_STYLE_ISOLATION': styleIsolation,
    };
}
exports.initDefine = initDefine;
function initCustomDefine() {
    let define = {};
    if (process.env.UNI_CUSTOM_DEFINE) {
        try {
            define = JSON.parse(process.env.UNI_CUSTOM_DEFINE);
        }
        catch (e) { }
    }
    return Object.keys(define).reduce((res, name) => {
        res['process.env.' + name] = JSON.stringify(define[name]);
        return res;
    }, {});
}

}, function(modId) { var map = {"../hbx/env":1781362618890,"../json":1781362618806}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618890, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.fixBinaryPath = exports.initModulePaths = exports.runByHBuilderX = exports.isInHBuilderX = void 0;
const path_1 = __importDefault(require("path"));
const module_1 = __importDefault(require("module"));
const uni_shared_1 = require("@dcloudio/uni-shared");
const resolve_1 = require("../resolve");
const utils_1 = require("../utils");
const utils_2 = require("./utils");
var utils_3 = require("./utils");
Object.defineProperty(exports, "isInHBuilderX", { enumerable: true, get: function () { return utils_3.isInHBuilderX; } });
exports.runByHBuilderX = (0, uni_shared_1.once)(() => {
    return (!!process.env.UNI_HBUILDERX_PLUGINS &&
        (!!process.env.RUN_BY_HBUILDERX || !!process.env.HX_Version));
});
/**
 * 增加 node_modules
 */
function initModulePaths() {
    if (!(0, utils_2.isInHBuilderX)()) {
        return;
    }
    const Module = module.constructor.length > 1 ? module.constructor : module_1.default;
    const nodeModulesPath = path_1.default.resolve(process.env.UNI_CLI_CONTEXT, 'node_modules');
    const oldNodeModulePaths = Module._nodeModulePaths;
    Module._nodeModulePaths = function (from) {
        const paths = oldNodeModulePaths.call(this, from);
        if (!paths.includes(nodeModulesPath)) {
            paths.push(nodeModulesPath);
        }
        return paths;
    };
}
exports.initModulePaths = initModulePaths;
function resolveEsbuildModule(name) {
    try {
        return path_1.default.dirname(require.resolve(name + '/package.json', {
            paths: [path_1.default.dirname((0, resolve_1.resolveBuiltIn)('esbuild/package.json'))],
        }));
    }
    catch (e) { }
    return '';
}
function fixBinaryPath() {
    // cli 工程在 HBuilderX 中运行
    if (!(0, utils_2.isInHBuilderX)() && (0, exports.runByHBuilderX)()) {
        if (utils_1.isWindows) {
            const win64 = resolveEsbuildModule('esbuild-windows-64');
            if (win64) {
                process.env.ESBUILD_BINARY_PATH = path_1.default.join(win64, 'esbuild.exe');
            }
        }
        else {
            const arm64 = resolveEsbuildModule('esbuild-darwin-arm64');
            if (arm64) {
                process.env.ESBUILD_BINARY_PATH = path_1.default.join(arm64, 'bin/esbuild');
            }
        }
    }
}
exports.fixBinaryPath = fixBinaryPath;

}, function(modId) { var map = {"../resolve":1781362618891,"./utils":1781362618894}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618891, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.resolveComponentsLibDirs = exports.resolveComponentsLibPath = exports.resolveVueI18nRuntime = exports.resolveBuiltIn = exports.getBuiltInPaths = exports.resolveMainPathOnce = exports.relativeFile = exports.requireResolve = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const debug_1 = __importDefault(require("debug"));
const resolve_1 = __importDefault(require("resolve"));
const uni_shared_1 = require("@dcloudio/uni-shared");
const utils_1 = require("./utils");
const env_1 = require("./hbx/env");
const constants_1 = require("./constants");
function requireResolve(filename, basedir) {
    return resolveWithSymlinks(filename, basedir);
}
exports.requireResolve = requireResolve;
function resolveWithSymlinks(id, basedir) {
    return resolve_1.default.sync(id, {
        basedir,
        extensions: process.env.UNI_APP_X === 'true' ? constants_1.uni_app_x_extensions : constants_1.extensions,
        // necessary to work with pnpm
        preserveSymlinks: true,
    });
}
function relativeFile(from, to) {
    const relativePath = (0, utils_1.normalizePath)(path_1.default.relative(path_1.default.dirname(from), to));
    return relativePath.startsWith('.') ? relativePath : './' + relativePath;
}
exports.relativeFile = relativeFile;
exports.resolveMainPathOnce = (0, uni_shared_1.once)((inputDir) => {
    if (process.env.UNI_APP_X === 'true') {
        const mainUTSPath = path_1.default.resolve(inputDir, 'main.uts');
        if (fs_1.default.existsSync(mainUTSPath)) {
            return (0, utils_1.normalizePath)(mainUTSPath);
        }
    }
    const mainTsPath = path_1.default.resolve(inputDir, 'main.ts');
    if (fs_1.default.existsSync(mainTsPath)) {
        return (0, utils_1.normalizePath)(mainTsPath);
    }
    return (0, utils_1.normalizePath)(path_1.default.resolve(inputDir, 'main.js'));
});
const ownerModules = [
    '@dcloudio/uni-app',
    '@dcloudio/vite-plugin-uni',
    '@dcloudio/uni-cli-shared',
];
const paths = [];
function resolveNodeModulePath(modulePath) {
    const nodeModulesPaths = [];
    const nodeModulesPath = path_1.default.join(modulePath, 'node_modules');
    if (fs_1.default.existsSync(nodeModulesPath)) {
        nodeModulesPaths.push(nodeModulesPath);
    }
    const index = modulePath.lastIndexOf('node_modules');
    if (index > -1) {
        nodeModulesPaths.push(path_1.default.join(modulePath.slice(0, index), 'node_modules'));
    }
    return nodeModulesPaths;
}
function initPaths() {
    const cliContext = process.env.UNI_CLI_CONTEXT || process.cwd();
    if (cliContext) {
        const pathSet = new Set();
        pathSet.add(path_1.default.join(cliContext, 'node_modules'));
        if (!(0, env_1.isInHBuilderX)()) {
            ;
            [`@dcloudio/uni-` + process.env.UNI_PLATFORM, ...ownerModules].forEach((ownerModule) => {
                let pkgPath = '';
                try {
                    pkgPath = require.resolve(ownerModule + '/package.json', {
                        paths: [cliContext],
                    });
                }
                catch (e) { }
                if (pkgPath) {
                    resolveNodeModulePath(path_1.default.dirname(pkgPath)).forEach((nodeModulePath) => {
                        pathSet.add(nodeModulePath);
                    });
                }
            });
        }
        paths.push(...pathSet);
        (0, debug_1.default)('uni-paths')(paths);
    }
}
function getBuiltInPaths() {
    if (!paths.length) {
        initPaths();
    }
    return paths;
}
exports.getBuiltInPaths = getBuiltInPaths;
function resolveBuiltIn(module) {
    if (process.env.UNI_COMPILE_TARGET === 'ext-api' &&
        process.env.UNI_APP_NEXT_WORKSPACE &&
        module.startsWith('@dcloudio/')) {
        return path_1.default.resolve(process.env.UNI_APP_NEXT_WORKSPACE, 'packages', module.replace('@dcloudio/', ''));
    }
    return require.resolve(module, { paths: getBuiltInPaths() });
}
exports.resolveBuiltIn = resolveBuiltIn;
function resolveVueI18nRuntime() {
    return path_1.default.resolve(__dirname, '../lib/vue-i18n/dist/vue-i18n.runtime.esm-bundler.js');
}
exports.resolveVueI18nRuntime = resolveVueI18nRuntime;
let componentsLibPath = '';
function resolveComponentsLibPath() {
    if (!componentsLibPath) {
        const dir = process.env.UNI_APP_X_DOM2 === 'true'
            ? '../lib-x-vapor'
            : process.env.UNI_APP_X === 'true'
                ? '../lib-x'
                : '../lib';
        if ((0, env_1.isInHBuilderX)()) {
            componentsLibPath = path_1.default.join(resolveBuiltIn('@dcloudio/uni-components/package.json'), dir);
        }
        else {
            try {
                componentsLibPath = path_1.default.join(resolveWithSymlinks('@dcloudio/uni-components/package.json', process.env.UNI_INPUT_DIR), dir);
            }
            catch (e) {
                try {
                    componentsLibPath = path_1.default.join(resolveWithSymlinks('@dcloudio/uni-components/package.json', process.cwd()), dir);
                }
                catch (e) {
                    console.log(e);
                }
            }
        }
    }
    return componentsLibPath;
}
exports.resolveComponentsLibPath = resolveComponentsLibPath;
function resolveComponentsLibDirs() {
    return process.env.UNI_COMPILE_TARGET === 'ext-api'
        ? []
        : [resolveComponentsLibPath()];
}
exports.resolveComponentsLibDirs = resolveComponentsLibDirs;

}, function(modId) { var map = {"fs":1781362618793,"resolve":1781362618891,"./hbx/env":1781362618890,"./constants":1781362618871}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618894, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.isInHBuilderX = void 0;
const path_1 = __importDefault(require("path"));
const uni_shared_1 = require("@dcloudio/uni-shared");
exports.isInHBuilderX = (0, uni_shared_1.once)(() => {
    // 自动化测试传入了 HX_APP_ROOT(其实就是UNI_HBUILDERX_PLUGINS)
    if (process.env.HX_APP_ROOT) {
        process.env.UNI_HBUILDERX_PLUGINS = process.env.HX_APP_ROOT + '/plugins';
        return true;
    }
    try {
        const { name } = require(path_1.default.resolve(process.cwd(), '../about/package.json'));
        if (name === 'about') {
            process.env.UNI_HBUILDERX_PLUGINS = path_1.default.resolve(process.cwd(), '..');
            return true;
        }
    }
    catch (e) {
        // console.error(e)
    }
    return false;
});

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618895, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.initAppProvide = void 0;
const path_1 = __importDefault(require("path"));
const libDir = path_1.default.resolve(__dirname, '../../lib');
function initAppProvide() {
    const cryptoDefine = [path_1.default.join(libDir, 'crypto.js'), 'default'];
    return {
        __f__: ['@dcloudio/uni-app', 'formatAppLog'],
        crypto: cryptoDefine,
        'window.crypto': cryptoDefine,
        'global.crypto': cryptoDefine,
        'uni.getCurrentSubNVue': ['@dcloudio/uni-app', 'getCurrentSubNVue'],
        'uni.requireNativePlugin': ['@dcloudio/uni-app', 'requireNativePlugin'],
    };
}
exports.initAppProvide = initAppProvide;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618896, function(require, module, exports) {

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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.isEnableConsole = exports.uniHBuilderXConsolePlugin = exports.formatInstallHBuilderXPluginTips = exports.installHBuilderXPlugin = exports.initModuleAlias = exports.createErrorWithBlockFlag = exports.formatAtFilename = void 0;
const path_1 = __importDefault(require("path"));
const utils_1 = require("../utils");
const console_1 = require("../vite/plugins/console");
const workers_1 = require("../workers");
var log_1 = require("./log");
Object.defineProperty(exports, "formatAtFilename", { enumerable: true, get: function () { return log_1.formatAtFilename; } });
Object.defineProperty(exports, "createErrorWithBlockFlag", { enumerable: true, get: function () { return log_1.createErrorWithBlockFlag; } });
__exportStar(require("./env"), exports);
var alias_1 = require("./alias");
Object.defineProperty(exports, "initModuleAlias", { enumerable: true, get: function () { return alias_1.initModuleAlias; } });
Object.defineProperty(exports, "installHBuilderXPlugin", { enumerable: true, get: function () { return alias_1.installHBuilderXPlugin; } });
Object.defineProperty(exports, "formatInstallHBuilderXPluginTips", { enumerable: true, get: function () { return alias_1.formatInstallHBuilderXPluginTips; } });
function uniHBuilderXConsolePlugin(method = '__f__') {
    const exclude = [];
    if (process.env.UNI_APP_X === 'true') {
        const workersDirs = (0, workers_1.resolveWorkersDir)(process.env.UNI_INPUT_DIR);
        if (workersDirs.length) {
            // 排除workers目录
            for (const workersDir of workersDirs) {
                exclude.push((0, utils_1.pathToGlob)(path_1.default.join(process.env.UNI_INPUT_DIR, workersDir), '**/*'));
            }
        }
    }
    return (0, console_1.uniConsolePlugin)({
        method,
        exclude,
        filename(filename) {
            filename = path_1.default.relative(process.env.UNI_INPUT_DIR, filename);
            if (filename.startsWith('.') || path_1.default.isAbsolute(filename)) {
                return '';
            }
            return (0, utils_1.normalizePath)(filename);
        },
    });
}
exports.uniHBuilderXConsolePlugin = uniHBuilderXConsolePlugin;
function isEnableConsole() {
    return !!((process.env.NODE_ENV === 'development' ||
        process.env.NODE_ENV === 'test') &&
        process.env.UNI_SOCKET_HOSTS &&
        process.env.UNI_SOCKET_PORT &&
        process.env.UNI_SOCKET_ID);
}
exports.isEnableConsole = isEnableConsole;

}, function(modId) { var map = {"../workers":1781362618899,"./env":1781362618890}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618899, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.initUniXCompilerRootWorkers = exports.uniJavaScriptWorkersPlugin = exports.resolveWorkersDir = exports.uniWorkersPlugin = exports.initWorkers = exports.getWorkersRootDirs = exports.resolveWorkersRootDir = exports.getWorkers = void 0;
const path_1 = __importDefault(require("path"));
const fs_extra_1 = __importDefault(require("fs-extra"));
const debug_1 = __importDefault(require("debug"));
const fast_glob_1 = require("fast-glob");
const utils_1 = require("./utils");
const json_1 = require("./json");
const uts_1 = require("./uts");
const uni_modules_1 = require("./vite/plugins/uts/uni_modules");
const resolve_1 = require("./resolve");
const dom2_1 = require("./dom2");
const debugWorkers = (0, debug_1.default)('uni:workers');
let workersRootDir = null;
let workersRootDirs = [];
let workers = {};
function getWorkers() {
    return workers;
}
exports.getWorkers = getWorkers;
function resolveWorkersRootDir() {
    // 默认是 workers
    return workersRootDir || 'workers';
}
exports.resolveWorkersRootDir = resolveWorkersRootDir;
function getWorkersRootDirs() {
    return workersRootDirs;
}
exports.getWorkersRootDirs = getWorkersRootDirs;
/**
 * 遍历目录下的所有uts文件，读取文件内容，正则匹配出定义的worker，返回文件名和类名的映射关系
 * export class MyWorkerTask extends WorkerTaskImpl {}
 * @param dir
 */
function initWorkers(workersDirs, rootDir) {
    workers = {};
    for (const workersDir of workersDirs) {
        const dir = path_1.default.join(rootDir, workersDir);
        if (!fs_extra_1.default.existsSync(dir)) {
            continue;
        }
        (0, fast_glob_1.sync)('**/*.uts', { cwd: dir }).forEach((file) => {
            const content = fs_extra_1.default.readFileSync(path_1.default.join(dir, file), 'utf-8');
            const match = content.match(/class\s+(.*)\s+extends\s+WorkerTaskImpl/);
            if (match && match[1]) {
                const key = (0, utils_1.normalizePath)(path_1.default.join(workersDir, file));
                workers[key] = match[1];
            }
        });
    }
    debugWorkers('workers', workers);
    return workers;
}
exports.initWorkers = initWorkers;
function uniWorkersPlugin() {
    const inputDir = process.env.UNI_INPUT_DIR;
    const platform = process.env.UNI_UTS_PLATFORM;
    const resolveWorkers = () => getWorkers();
    function refreshWorkers() {
        workersRootDirs = resolveWorkersDir(inputDir);
        initWorkers(workersRootDirs, inputDir);
        return Object.keys(getWorkers()).length > 0;
    }
    refreshWorkers();
    const preprocessor = platform === 'app-android'
        ? (0, uni_modules_1.createAppAndroidUniModulesSyncFilePreprocessorOnce)(false)
        : platform === 'app-ios'
            ? (0, uni_modules_1.createAppIosUniModulesSyncFilePreprocessorOnce)(false)
            : platform === 'app-harmony'
                ? (0, uni_modules_1.createAppHarmonyUniModulesSyncFilePreprocessorOnce)(false)
                : null;
    const cache = {};
    const uniXKotlinCompiler = platform === 'app-android'
        ? (0, uts_1.resolveUTSCompiler)().createUniXKotlinCompilerOnce({
            resolveWorkers,
            sourceFileCallback: (0, dom2_1.initSourceFileCallback)(),
        })
        : null;
    const uniXSwiftCompiler = platform === 'app-ios'
        ? (0, uts_1.resolveUTSCompiler)().createUniXSwiftCompilerOnce({
            resolveWorkers,
        })
        : null;
    const uniXArkTSCompiler = platform === 'app-harmony'
        ? (0, uts_1.resolveUTSCompiler)().createUniXArkTSCompilerOnce({
            resolveWorkers,
        })
        : null;
    return {
        name: 'uni-workers',
        enforce: 'pre',
        async buildStart() {
            if (refreshWorkers()) {
                if (preprocessor) {
                    await syncWorkersFiles(platform, inputDir, preprocessor, cache);
                }
            }
            // 需要等待 workers 文件同步完之后，添加到 rootFiles 中，触发 tsc 的编译
            if (uniXKotlinCompiler) {
                await initUniXCompilerRootWorkers((0, uts_1.tscOutDir)('app-android'), uniXKotlinCompiler);
            }
            if (uniXSwiftCompiler) {
                await initUniXCompilerRootWorkers((0, uts_1.tscOutDir)('app-ios'), uniXSwiftCompiler);
            }
            if (uniXArkTSCompiler) {
                await initUniXCompilerRootWorkers((0, uts_1.tscOutDir)('app-harmony'), uniXArkTSCompiler);
            }
        },
    };
}
exports.uniWorkersPlugin = uniWorkersPlugin;
async function syncWorkersFiles(platform, inputDir, preprocessor, cache) {
    if (platform !== 'app-harmony' &&
        platform !== 'app-android' &&
        platform !== 'app-ios') {
        return;
    }
    const workersDirs = resolveWorkersDir(inputDir);
    if (workersDirs.length) {
        const { syncUTSFiles } = (0, uts_1.resolveUTSCompiler)();
        for (const workersDir of workersDirs) {
            await syncUTSFiles((0, utils_1.normalizePath)(path_1.default.join(workersDir, '**/*.uts')), inputDir, (0, uts_1.tscOutDir)(platform), true, preprocessor, cache);
        }
    }
}
function resolveWorkersDir(inputDir) {
    const workersDirs = [];
    const manifestJson = (0, json_1.parseManifestJsonOnce)(inputDir);
    if (manifestJson.workers) {
        let workersDir = typeof manifestJson.workers === 'string'
            ? manifestJson.workers
            : manifestJson.workers.path;
        if (workersDir) {
            workersDir = (0, utils_1.normalizePath)(workersDir);
            const dir = path_1.default.join(inputDir, workersDir);
            if (fs_extra_1.default.existsSync(dir)) {
                workersRootDir = workersDir;
                workersDirs.push(workersDir);
            }
        }
    }
    // 遍历uni_modules插件目录是否有workers目录
    const uniModulesDir = path_1.default.join(inputDir, 'uni_modules');
    if (fs_extra_1.default.existsSync(uniModulesDir)) {
        fs_extra_1.default.readdirSync(uniModulesDir).forEach((dir) => {
            if (fs_extra_1.default.existsSync(path_1.default.join(uniModulesDir, dir, 'workers'))) {
                workersDirs.push('uni_modules/' + dir + '/workers');
            }
        });
    }
    debugWorkers('workersDirs', workersDirs);
    return workersDirs;
}
exports.resolveWorkersDir = resolveWorkersDir;
function uniJavaScriptWorkersPlugin() {
    // 仅小程序平台外置uni-worker.mp.js
    const external = (process.env.UNI_UTS_PLATFORM || '').startsWith('mp-');
    let workerPolyfillCode = '';
    let isWrite = false;
    const UniAppWorkerJSName = external ? 'uni-worker.mp.js' : 'uni-worker.web.js';
    let viteServer = null;
    const workersRootPaths = [];
    const workerPolyfillPath = `@dcloudio/uni-app/dist-x/${UniAppWorkerJSName}`;
    const workerPolyfillAbsPath = (0, utils_1.normalizePath)((0, resolve_1.resolveBuiltIn)(workerPolyfillPath));
    function isWorkerFile(id) {
        if (workersRootPaths.length) {
            return workersRootPaths.some((dir) => id.startsWith(dir));
        }
        return false;
    }
    function parseWorkerEntryFile(workerJsPath) {
        const workerPath = workerJsPath.slice(1).replace('.js', '.uts');
        if (workerPath in workers) {
            return (0, utils_1.normalizePath)(path_1.default.resolve(process.env.UNI_INPUT_DIR, workerPath));
        }
    }
    function parseWorkerClass(id) {
        const filename = id.split('?')[0];
        if (isWorkerFile(filename)) {
            const workerPath = (0, utils_1.normalizePath)(path_1.default.relative(process.env.UNI_INPUT_DIR, filename));
            return workers[workerPath] || '';
        }
        return false;
    }
    return {
        name: 'uni:javascript-workers',
        configureServer(server) {
            viteServer = server;
        },
        buildStart() {
            if (!workerPolyfillCode && Object.keys(getWorkers()).length) {
                workerPolyfillCode = fs_extra_1.default.readFileSync(workerPolyfillAbsPath, 'utf-8');
            }
            workersRootPaths.length = 0;
            for (const workersRootDir of getWorkersRootDirs()) {
                workersRootPaths.push((0, utils_1.normalizePath)(path_1.default.resolve(process.env.UNI_INPUT_DIR, workersRootDir)));
            }
        },
        resolveId(id) {
            // uni.createWorker('workers/request/index.uts')
            // 编译阶段调整为 uni.createWorker('workers/request/index.js')，确保开发和运行时都是用.js后缀加载
            // 不调整成js后缀或.uts?import这些格式， vite 是不会走transform逻辑的，而是直接读取文件内容
            if (viteServer) {
                const workerEntryFile = parseWorkerEntryFile(id);
                if (workerEntryFile) {
                    return workerEntryFile;
                }
                if (id === workerPolyfillPath) {
                    return workerPolyfillAbsPath;
                }
            }
        },
        load(id) {
            const filename = id.split('?')[0];
            const workerClass = parseWorkerClass(filename);
            if (workerClass === false) {
                return;
            }
            if (fs_extra_1.default.existsSync(filename)) {
                let code = (viteServer ? `import '${workerPolyfillPath}';` : '') +
                    fs_extra_1.default
                        .readFileSync(filename, 'utf-8')
                        // 移除 export
                        .replace(/export\s+class\s+(.*)\s+extends\s+WorkerTaskImpl\s*{/, 'class $1 extends WorkerTaskImpl {');
                // 如果是入口文件，需要追加初始化代码
                if (workerClass) {
                    code += `\n;new ${workerClass}().entry()`;
                }
                return code;
            }
        },
        generateBundle(_, bundle) {
            const workers = getWorkers();
            const workerRootDir = resolveWorkersRootDir();
            const workerPaths = Object.keys(workers).map((key) => {
                if (key.startsWith('uni_modules')) {
                    key = workerRootDir + '/' + key;
                }
                return key.replace('.uts', '.js');
            });
            if (workerPaths.length) {
                Object.keys(bundle).forEach((file) => {
                    if (workerPaths.includes(file)) {
                        const chunk = bundle[file];
                        if (chunk.type === 'chunk') {
                            const workerCode = external
                                ? `require('${(0, utils_1.normalizePath)(path_1.default.relative(path_1.default.dirname(file), path_1.default.join(resolveWorkersRootDir(), 'uni-worker.js')))}')`
                                : workerPolyfillCode;
                            chunk.code = `${workerCode}\n${chunk.code}`;
                        }
                    }
                });
            }
        },
        writeBundle() {
            if (external && Object.keys(getWorkers()).length && !isWrite) {
                isWrite = true;
                // 写入uni-worker.js
                fs_extra_1.default.outputFileSync(path_1.default.resolve(process.env.UNI_OUTPUT_DIR, resolveWorkersRootDir(), 'uni-worker.js'), workerPolyfillCode);
            }
        },
    };
}
exports.uniJavaScriptWorkersPlugin = uniJavaScriptWorkersPlugin;
async function initUniXCompilerRootWorkers(rootDir, compiler) {
    const workers = getWorkers();
    if (Object.keys(workers).length) {
        for (const key in workers) {
            const file = path_1.default.join(rootDir, key + '.ts');
            if (fs_extra_1.default.existsSync(file)) {
                if (!compiler.hasRootFile(file)) {
                    await compiler.addRootFile(file);
                }
            }
        }
    }
}
exports.initUniXCompilerRootWorkers = initUniXCompilerRootWorkers;

}, function(modId) { var map = {"./json":1781362618806,"./uts":1781362618901,"./resolve":1781362618891,"./dom2":1781362618908}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618901, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.isUniHelpers = exports.isUTSProxy = exports.tscOutDir = exports.uvueOutDir = exports.genUniExtApiDeclarationFileOnce = exports.initUTSSwiftAutoImportsOnce = exports.initUTSKotlinAutoImportsOnce = exports.resolveUniTypeScript = exports.parseUniExtApiNamespacesJsOnce = exports.parseUniExtApiNamespacesOnce = exports.parseSwiftModuleWithPluginId = exports.parseSwiftPackageWithPluginId = exports.parseKotlinPackageWithPluginId = exports.parseCustomElementExports = exports.initUTSCustomElements = exports.initUTSComponents = exports.parseUTSCustomElement = exports.parseUTSComponent = exports.getUTSCustomElementAutoImports = exports.getUTSComponentAutoImports = exports.getUTSCustomElement = exports.isUTSCustomElement = exports.getUTSPluginCustomElements = exports.getUTSCustomElements = exports.clearUTSCustomElements = exports.isUTSComponent = exports.clearUTSComponents = exports.getUTSCustomElementsExports = exports.resolveUTSCompilerVersion = exports.resolveUTSCompiler = exports.resolveUTSModule = exports.resolveUTSAppModule = void 0;
// 重要，该文件编译后的 js 需要同步到 vue2 编译器 uni-cli-shared/lib/uts
const fs_extra_1 = __importDefault(require("fs-extra"));
const path_1 = __importDefault(require("path"));
const fast_glob_1 = __importDefault(require("fast-glob"));
const unimport_1 = require("unimport");
const hbx_1 = require("./hbx");
const utils_1 = require("./utils");
const uni_modules_1 = require("./uni_modules");
const preprocess_1 = require("./preprocess");
const x_1 = require("./x");
function once(fn, ctx = null) {
    let res;
    return ((...args) => {
        if (fn) {
            res = fn.apply(ctx, args);
            fn = null;
        }
        return res;
    });
}
/**
 * 解析 app 平台的 uts 插件，任意平台（android|ios）存在即可
 * @param id
 * @param importer
 * @returns
 */
function resolveUTSAppModule(platform, id, importer, includeUTSSDK = true) {
    id = path_1.default.resolve(importer, id);
    if (id.includes('uni_modules') || (includeUTSSDK && id.includes('utssdk'))) {
        const parts = (0, utils_1.normalizePath)(id).split('/');
        const parentDir = parts[parts.length - 2];
        if (parentDir === 'uni_modules' ||
            (includeUTSSDK && parentDir === 'utssdk')) {
            const basedir = parentDir === 'uni_modules' ? 'utssdk' : '';
            if ((0, x_1.isUniAppXJsEngine)()) {
                // js engine
                if (parentDir === 'uni_modules') {
                    const appJsIndex = path_1.default.resolve(id, basedir, 'app-js', 'index.uts');
                    if (fs_extra_1.default.existsSync(appJsIndex)) {
                        return appJsIndex;
                    }
                }
            }
            /**
             * 鸿蒙平台解析优先级
             * 1. utssdk/app-harmony/index.uts (native)
             * 2. utssdk/app-js/index.uts (js)
             * 3. utssdk/index.uts (native)
             */
            if (process.env.UNI_UTS_PLATFORM === 'app-harmony') {
                // 出于兼容历史项目考虑，鸿蒙优先使用app-harmony，无app-harmony的情况下再使用app-js
                if (parentDir === 'uni_modules') {
                    const appHarmonyIndex = path_1.default.resolve(id, basedir, 'app-harmony', 'index.uts');
                    const appJsIndex = path_1.default.resolve(id, basedir, 'app-js', 'index.uts');
                    if (!fs_extra_1.default.existsSync(appHarmonyIndex) && fs_extra_1.default.existsSync(appJsIndex)) {
                        return appJsIndex;
                    }
                }
            }
            if (fs_extra_1.default.existsSync(path_1.default.resolve(id, basedir, 'index.uts'))) {
                return id;
            }
            // customElements 组件
            if (fs_extra_1.default.existsSync(path_1.default.resolve(id, 'customElements'))) {
                return id;
            }
            const fileName = id.split('?')[0];
            const resolvePlatformDir = (p) => {
                return path_1.default.resolve(fileName, basedir, p);
            };
            const extname = ['.uts', '.vue', '.uvue'];
            if (platform === 'app-harmony') {
                if (resolveUTSFile(resolvePlatformDir(platform), extname)) {
                    return id;
                }
                return;
            }
            if (resolveUTSFile(resolvePlatformDir('app-android'), extname)) {
                return id;
            }
            if (resolveUTSFile(resolvePlatformDir('app-ios'), extname)) {
                return id;
            }
        }
    }
}
exports.resolveUTSAppModule = resolveUTSAppModule;
// 仅限 root/uni_modules/test-plugin | root/utssdk/test-plugin 格式
function resolveUTSModule(id, importer, includeUTSSDK = true) {
    if (process.env.UNI_PLATFORM === 'app' ||
        process.env.UNI_PLATFORM === 'app-plus' ||
        process.env.UNI_PLATFORM === 'app-harmony') {
        return resolveUTSAppModule(process.env.UNI_UTS_PLATFORM, id, importer);
    }
    id = path_1.default.resolve(importer, id);
    if (id.includes('uni_modules') || (includeUTSSDK && id.includes('utssdk'))) {
        const parts = (0, utils_1.normalizePath)(id).split('/');
        const parentDir = parts[parts.length - 2];
        if (parentDir === 'uni_modules' ||
            (includeUTSSDK && parentDir === 'utssdk')) {
            const basedir = parentDir === 'uni_modules' ? 'utssdk' : '';
            const resolvePlatformDir = (p) => {
                return path_1.default.resolve(id, basedir, p);
            };
            let index = resolveUTSFile(resolvePlatformDir(process.env.UNI_UTS_PLATFORM));
            const pluginId = parentDir === 'uni_modules' ? parts[parts.length - 1] : '';
            if (index) {
                return resolveUTSEncryptFile(pluginId, index) || index;
            }
            index = path_1.default.resolve(id, basedir, 'index.uts');
            if (fs_extra_1.default.existsSync(index)) {
                return resolveUTSEncryptFile(pluginId, index) || index;
            }
        }
    }
}
exports.resolveUTSModule = resolveUTSModule;
function resolveUTSEncryptFile(pluginId, index) {
    if (!pluginId) {
        return;
    }
    const cacheDir = process.env.UNI_MODULES_ENCRYPT_CACHE_DIR;
    if (!cacheDir) {
        return;
    }
    // 仅支持 uts 加密解析
    if (!index.endsWith('.uts')) {
        return;
    }
    const cacheFile = path_1.default.resolve(cacheDir, 'uni_modules', pluginId, 'index.module.js');
    if (fs_extra_1.default.existsSync(cacheFile)) {
        return cacheFile;
    }
}
function resolveUTSFile(dir, extensions = ['.uts', '.ts', '.js']) {
    for (let i = 0; i < extensions.length; i++) {
        const indexFile = path_1.default.join(dir, 'index' + extensions[i]);
        if (fs_extra_1.default.existsSync(indexFile)) {
            return indexFile;
        }
    }
}
function resolveUTSCompiler(throwError = false) {
    let compilerPath = '';
    if (process.env.UNI_COMPILE_TARGET === 'ext-api' &&
        process.env.UNI_APP_NEXT_WORKSPACE) {
        return require(path_1.default.resolve(process.env.UNI_APP_NEXT_WORKSPACE, 'packages/uni-uts-v1'));
    }
    if ((0, hbx_1.isInHBuilderX)()) {
        try {
            compilerPath = require.resolve(path_1.default.resolve(process.env.UNI_HBUILDERX_PLUGINS, 'uniapp-uts-v1'));
        }
        catch (e) { }
    }
    if (!compilerPath) {
        try {
            compilerPath = require.resolve('@dcloudio/uni-uts-v1', {
                paths: [process.env.UNI_CLI_CONTEXT || process.cwd()],
            });
        }
        catch (e) {
            if (throwError) {
                throw `Error: Cannot find module '@dcloudio/uni-uts-v1'`;
            }
            console.error((0, utils_1.installDepTips)('devDependencies', '@dcloudio/uni-uts-v1', resolveUTSCompilerVersion()));
            process.exit(0);
        }
    }
    return require(compilerPath);
}
exports.resolveUTSCompiler = resolveUTSCompiler;
function resolveUTSCompilerVersion() {
    let utsCompilerVersion = '';
    try {
        utsCompilerVersion = require('../package.json').version;
    }
    catch (e) {
        try {
            // vue2
            utsCompilerVersion = require('../../package.json').version;
        }
        catch (e) { }
    }
    if (utsCompilerVersion.startsWith('2.0.')) {
        utsCompilerVersion = '^3.0.0-alpha-3060920221117001';
    }
    return utsCompilerVersion;
}
exports.resolveUTSCompilerVersion = resolveUTSCompilerVersion;
const utsComponents = new Map();
const utsCustomElements = new Map();
const utsCustomElementsExports = new Map();
function getUTSCustomElementsExports() {
    return utsCustomElementsExports;
}
exports.getUTSCustomElementsExports = getUTSCustomElementsExports;
function clearUTSComponents() {
    utsComponents.clear();
}
exports.clearUTSComponents = clearUTSComponents;
function isUTSComponent(name) {
    return utsComponents.has(name);
}
exports.isUTSComponent = isUTSComponent;
function clearUTSCustomElements() {
    utsCustomElements.clear();
}
exports.clearUTSCustomElements = clearUTSCustomElements;
function getUTSCustomElements() {
    return utsCustomElements;
}
exports.getUTSCustomElements = getUTSCustomElements;
function getUTSPluginCustomElements() {
    const pluginCustomElements = {};
    for (const [key, value] of utsCustomElements.entries()) {
        const parts = value.source.split('?')[0].split('/');
        const pluginId = parts[parts.length - 1];
        if (!pluginId) {
            continue;
        }
        if (!pluginCustomElements[pluginId]) {
            pluginCustomElements[pluginId] = new Set();
        }
        pluginCustomElements[pluginId].add(key);
    }
    return pluginCustomElements;
}
exports.getUTSPluginCustomElements = getUTSPluginCustomElements;
function isUTSCustomElement(name) {
    // 支持内置CustomElement的本地注册开发，
    // 内置组件目录：customElements/uni-progress/uni-progress.uts
    // 实际使用时是：progress，所以需要自动补充uni-前缀做判断
    return utsCustomElements.has(name) || utsCustomElements.has('uni-' + name);
}
exports.isUTSCustomElement = isUTSCustomElement;
function getUTSCustomElement(name) {
    return utsCustomElements.get(name) || utsCustomElements.get('uni-' + name);
}
exports.getUTSCustomElement = getUTSCustomElement;
function getUTSComponentAutoImports(language) {
    const utsComponentAutoImports = {};
    utsComponents.forEach(({ kotlinPackage, swiftModule }, name) => {
        const source = language === 'kotlin' ? kotlinPackage : swiftModule;
        const className = (0, utils_1.capitalize)((0, utils_1.camelize)(name)) + 'Element';
        if (!utsComponentAutoImports[source]) {
            utsComponentAutoImports[source] = [[className]];
        }
        else {
            if (!utsComponentAutoImports[source].find((item) => item[0] === className)) {
                utsComponentAutoImports[source].push([className]);
            }
        }
    });
    return utsComponentAutoImports;
}
exports.getUTSComponentAutoImports = getUTSComponentAutoImports;
function getUTSCustomElementAutoImports(language) {
    const utsCustomElementAutoImports = {};
    utsCustomElementsExports.forEach(({ exports, kotlinPackage, swiftModule }) => {
        const source = language === 'kotlin' ? kotlinPackage : swiftModule;
        if (!utsCustomElementAutoImports[source]) {
            utsCustomElementAutoImports[source] = exports;
        }
        else {
            utsCustomElementAutoImports[source].push(...exports);
        }
    });
    return utsCustomElementAutoImports;
}
exports.getUTSCustomElementAutoImports = getUTSCustomElementAutoImports;
function parseUTSComponent(name, type) {
    const meta = utsComponents.get(name);
    if (meta) {
        const namespace = meta[type === 'swift' ? 'swiftNamespace' : 'kotlinNamespace'] || '';
        const className = (0, utils_1.capitalize)((0, utils_1.camelize)(name)) + 'Component';
        return {
            className,
            namespace,
            source: meta.source,
        };
    }
}
exports.parseUTSComponent = parseUTSComponent;
function parseUTSCustomElement(name, type) {
    const meta = getUTSCustomElement(name);
    if (meta) {
        const namespace = meta[type === 'swift' ? 'swiftNamespace' : 'kotlinNamespace'] || '';
        const className = (0, utils_1.capitalize)((0, utils_1.camelize)(name)) + 'Element';
        return {
            className,
            namespace,
            source: meta.source,
        };
    }
}
exports.parseUTSCustomElement = parseUTSCustomElement;
function initUTSComponents(inputDir, platform) {
    const components = [];
    const isApp = platform === 'app' || platform === 'app-plus';
    const easycomsObj = {};
    const dirs = resolveUTSComponentDirs(inputDir);
    dirs.forEach((dir) => {
        const is_uni_modules_utssdk = dir.endsWith('utssdk');
        const is_ussdk = !is_uni_modules_utssdk && path_1.default.dirname(dir).endsWith('utssdk');
        const pluginId = is_uni_modules_utssdk
            ? path_1.default.basename(path_1.default.dirname(dir))
            : path_1.default.basename(dir);
        if (is_uni_modules_utssdk || is_ussdk) {
            // dir 是 uni_modules/test-plugin/utssdk 或者 utssdk/test-plugin
            // 需要分平台解析，不能直接解析 utssdk 目录下的文件，因为 utssdk 目录下可能存在多个平台的文件
            const cwd = isApp
                ? dir
                : path_1.default.join(dir, platform === 'h5' ? 'web' : platform);
            fast_glob_1.default
                .sync('**/*.vue', {
                cwd,
                absolute: true,
            })
                .forEach((file) => {
                let name = parseVueComponentName(file);
                if (!name) {
                    if (file.endsWith('index.vue')) {
                        name = path_1.default.basename(is_uni_modules_utssdk ? path_1.default.dirname(dir) : dir);
                    }
                }
                if (name) {
                    const source = '@/' +
                        (0, utils_1.normalizePath)(isApp
                            ? path_1.default.relative(inputDir, is_uni_modules_utssdk ? path_1.default.dirname(dir) : dir)
                            : path_1.default.relative(inputDir, file));
                    const kotlinPackage = parseKotlinPackageWithPluginId(pluginId, is_uni_modules_utssdk);
                    const swiftModule = parseSwiftModuleWithPluginId(pluginId, is_uni_modules_utssdk);
                    const swiftNamespace = parseSwiftPackageWithPluginId(pluginId, is_uni_modules_utssdk);
                    easycomsObj[`^${name}$`] = {
                        source: isApp ? `${source}?uts-proxy` : source,
                        kotlinPackage: kotlinPackage,
                        swiftModule: swiftModule,
                        kotlinNamespace: kotlinPackage,
                        swiftNamespace: swiftNamespace,
                    };
                }
            });
        }
    });
    Object.keys(easycomsObj).forEach((name) => {
        const obj = easycomsObj[name];
        const componentName = name.slice(1, -1);
        components.push({
            name: componentName,
            pattern: new RegExp(name),
            replacement: obj.source,
        });
        utsComponents.set(componentName, {
            source: obj.source,
            kotlinPackage: obj.kotlinPackage,
            swiftModule: obj.swiftModule,
            kotlinNamespace: obj.kotlinPackage,
            swiftNamespace: obj.swiftNamespace,
        });
    });
    return components;
}
exports.initUTSComponents = initUTSComponents;
function resolveUTSComponentDirs(inputDir) {
    const utssdkDir = path_1.default.resolve(inputDir, 'utssdk');
    const uniModulesDir = path_1.default.resolve(inputDir, 'uni_modules');
    return (fs_extra_1.default.existsSync(utssdkDir)
        ? fast_glob_1.default.sync('*', {
            cwd: utssdkDir,
            absolute: true,
            onlyDirectories: true,
        })
        : []).concat(fs_extra_1.default.existsSync(uniModulesDir)
        ? fast_glob_1.default.sync('*/utssdk', {
            cwd: uniModulesDir,
            absolute: true,
            onlyDirectories: true,
        })
        : []);
}
function initUTSCustomElements(inputDir, platform) {
    const isApp = platform === 'app' || platform === 'app-plus' || platform === 'app-harmony';
    const dirs = resolveUTSCustomElementsDirs(inputDir);
    const unimport = (0, unimport_1.createUnimport)({});
    dirs.forEach((dir) => {
        fs_extra_1.default.readdirSync(dir).forEach((name) => {
            const folder = path_1.default.resolve(dir, name);
            if (!isDir(folder)) {
                return;
            }
            const files = fs_extra_1.default.readdirSync(folder);
            // 读取文件夹文件列表，比对文件名（fs.existsSync在大小写不敏感的系统会匹配不准确）
            // customElements 的文件名是 uts 后缀
            const ext = '.uts';
            if (files.includes(name + ext)) {
                const filePath = path_1.default.resolve(folder, name + ext);
                const pluginId = path_1.default.basename(path_1.default.dirname(dir));
                const source = '@/' +
                    (0, utils_1.normalizePath)(isApp
                        ? path_1.default.relative(inputDir, path_1.default.dirname(dir))
                        : path_1.default.relative(inputDir, filePath));
                const importSource = isApp ? `${source}?uts-proxy` : source;
                const kotlinPackage = parseKotlinPackageWithPluginId(pluginId, true);
                const swiftModule = parseSwiftModuleWithPluginId(pluginId, true);
                const swiftNamespace = parseSwiftPackageWithPluginId(pluginId, true);
                const meta = {
                    source: importSource,
                    kotlinPackage: kotlinPackage,
                    swiftModule: swiftModule,
                    kotlinNamespace: kotlinPackage,
                    swiftNamespace: swiftNamespace,
                };
                utsCustomElements.set(name, meta);
                parseCustomElementExports(filePath, unimport).then((exports_) => {
                    const prefix = (0, utils_1.capitalize)((0, utils_1.camelize)(name));
                    const customElementExports = exports_
                        .filter((item) => item.name.startsWith(prefix))
                        .map((item) => [item.name]);
                    if (utsCustomElementsExports.has(importSource)) {
                        utsCustomElementsExports
                            .get(importSource)
                            .exports.push(...customElementExports);
                    }
                    else {
                        utsCustomElementsExports.set(importSource, {
                            ...meta,
                            exports: customElementExports,
                        });
                    }
                });
            }
        });
    });
    // 不需要easycom匹配
    return [];
}
exports.initUTSCustomElements = initUTSCustomElements;
function parseCustomElementExports(filePath, unimport = (0, unimport_1.createUnimport)({})) {
    return unimport.scanImportsFromFile(filePath, true);
}
exports.parseCustomElementExports = parseCustomElementExports;
const isDir = (path) => {
    const stat = fs_extra_1.default.lstatSync(path);
    if (stat.isDirectory()) {
        return true;
    }
    else if (stat.isSymbolicLink()) {
        return fs_extra_1.default.lstatSync(fs_extra_1.default.realpathSync(path)).isDirectory();
    }
    return false;
};
function resolveUTSCustomElementsDirs(inputDir) {
    const uniModulesDir = path_1.default.resolve(inputDir, 'uni_modules');
    return fs_extra_1.default.existsSync(uniModulesDir)
        ? fast_glob_1.default.sync('*/customElements', {
            cwd: uniModulesDir,
            absolute: true,
            onlyDirectories: true,
        })
        : [];
}
const nameRE = /name\s*:\s*['|"](.*)['|"]/;
function parseVueComponentName(file) {
    const content = fs_extra_1.default.readFileSync(file, 'utf8');
    const matches = content.match(nameRE);
    if (matches) {
        return matches[1];
    }
}
function prefix(id) {
    if (process.env.UNI_UTS_MODULE_PREFIX &&
        !id.startsWith(process.env.UNI_UTS_MODULE_PREFIX)) {
        return process.env.UNI_UTS_MODULE_PREFIX + '-' + id;
    }
    return id;
}
function parseKotlinPackageWithPluginId(id, is_uni_modules) {
    return 'uts.sdk.' + (is_uni_modules ? 'modules.' : '') + (0, utils_1.camelize)(prefix(id));
}
exports.parseKotlinPackageWithPluginId = parseKotlinPackageWithPluginId;
function parseSwiftPackageWithPluginId(id, is_uni_modules) {
    return ('UTSSDK' +
        (is_uni_modules ? 'Modules' : '') +
        (0, utils_1.capitalize)((0, utils_1.camelize)(prefix(id))));
}
exports.parseSwiftPackageWithPluginId = parseSwiftPackageWithPluginId;
function parseSwiftModuleWithPluginId(id, is_uni_modules) {
    if (!is_uni_modules) {
        return parseSwiftPackageWithPluginId(id, is_uni_modules);
    }
    return `unimodule` + (0, utils_1.capitalize)((0, utils_1.camelize)(prefix(id)));
}
exports.parseSwiftModuleWithPluginId = parseSwiftModuleWithPluginId;
async function parseUniExtApiAutoImports(uniExtApiAutoImports, extApis, parseSource) {
    if (Object.keys(extApis).length) {
        const { parseExportIdentifiers } = resolveUTSCompiler();
        for (const name in extApis) {
            const options = extApis[name];
            if ((0, utils_1.isArray)(options) && options.length >= 2) {
                const pluginId = path_1.default.basename(options[0]);
                const source = parseSource(pluginId);
                if (uniExtApiAutoImports[source]) {
                    continue;
                }
                uniExtApiAutoImports[source] = [];
                const filename = `uni_modules/${pluginId}/utssdk/interface.uts`;
                const interfaceFileName = path_1.default.resolve(process.env.UNI_INPUT_DIR, filename);
                if (fs_extra_1.default.existsSync(interfaceFileName)) {
                    const ids = await parseExportIdentifiers(interfaceFileName, preprocess_1.preUVueJs);
                    ids
                        // 过滤掉 Uni
                        .filter((id) => id !== 'Uni')
                        .forEach((id) => {
                        uniExtApiAutoImports[source].push([id]);
                    });
                }
            }
        }
    }
    return uniExtApiAutoImports;
}
let uniExtApiKotlinAutoImports = null;
async function parseUniExtApiKotlinAutoImportsOnce(extApis) {
    if (uniExtApiKotlinAutoImports) {
        return uniExtApiKotlinAutoImports;
    }
    uniExtApiKotlinAutoImports = {};
    return parseUniExtApiAutoImports(uniExtApiKotlinAutoImports, extApis, (pluginId) => {
        return parseKotlinPackageWithPluginId(pluginId, true);
    });
}
let uniExtApiSwiftAutoImports = null;
async function parseUniExtApiSwiftAutoImportsOnce(extApis) {
    if (uniExtApiSwiftAutoImports) {
        return uniExtApiSwiftAutoImports;
    }
    uniExtApiSwiftAutoImports = {};
    return parseUniExtApiAutoImports(uniExtApiSwiftAutoImports, extApis, (pluginId) => {
        return parseSwiftModuleWithPluginId(pluginId, true);
    });
}
exports.parseUniExtApiNamespacesOnce = once((platform, language) => {
    const extApis = (0, exports.parseUniExtApiNamespacesJsOnce)(platform, language);
    const namespaces = {};
    Object.keys(extApis).forEach((name) => {
        const options = extApis[name];
        let source = options[0];
        const pluginId = path_1.default.basename(options[0]);
        if (language === 'kotlin') {
            source = parseKotlinPackageWithPluginId(pluginId, true);
        }
        else if (language === 'swift') {
            source = parseSwiftModuleWithPluginId(pluginId, true);
        }
        namespaces[name] = [source, options[1]];
    });
    return namespaces;
});
exports.parseUniExtApiNamespacesJsOnce = once((platform, language) => {
    const extApis = (0, uni_modules_1.parseUniExtApis)(true, platform, language);
    const namespaces = {};
    Object.keys(extApis).forEach((name) => {
        const options = extApis[name];
        if ((0, utils_1.isArray)(options) && options.length >= 2) {
            namespaces[name.replace('uni.', '')] = [options[0], options[1]];
        }
    });
    return namespaces;
});
function resolveUniTypeScript() {
    if ((0, hbx_1.isInHBuilderX)()) {
        return require(path_1.default.resolve(process.env.UNI_HBUILDERX_PLUGINS, 'uniapp-uts-v1', 'node_modules', '@dcloudio', 'uni-uts-v1', 'lib', 'typescript'));
    }
    return require('@dcloudio/uni-uts-v1/lib/typescript');
}
exports.resolveUniTypeScript = resolveUniTypeScript;
async function initUTSAutoImports(autoImports, platform, language) {
    const utsComponents = getUTSComponentAutoImports(language);
    Object.keys(utsComponents).forEach((source) => {
        if (autoImports[source]) {
            autoImports[source].push(...utsComponents[source]);
        }
        else {
            autoImports[source] = utsComponents[source];
        }
    });
    const utsCustomElements = getUTSCustomElementAutoImports(language);
    Object.keys(utsCustomElements).forEach((source) => {
        if (autoImports[source]) {
            autoImports[source].push(...utsCustomElements[source]);
        }
        else {
            autoImports[source] = utsCustomElements[source];
        }
    });
    const extApis = (0, uni_modules_1.parseUniExtApis)(true, platform, language);
    const extApiImports = await (language === 'kotlin'
        ? parseUniExtApiKotlinAutoImportsOnce
        : parseUniExtApiSwiftAutoImportsOnce)(extApis);
    Object.keys(extApiImports).forEach((source) => {
        if (autoImports[source]) {
            autoImports[source].push(...extApiImports[source]);
        }
        else {
            autoImports[source] = extApiImports[source];
        }
    });
    return autoImports;
}
let autoKotlinImports = null;
async function initUTSKotlinAutoImportsOnce() {
    if (autoKotlinImports) {
        return autoKotlinImports;
    }
    autoKotlinImports = {};
    return initUTSAutoImports(autoKotlinImports, 'app-android', 'kotlin');
}
exports.initUTSKotlinAutoImportsOnce = initUTSKotlinAutoImportsOnce;
let autoSwiftImports = null;
async function initUTSSwiftAutoImportsOnce() {
    if (autoSwiftImports) {
        return autoSwiftImports;
    }
    autoSwiftImports = {};
    return initUTSAutoImports(autoSwiftImports, 'app-ios', 'swift');
}
exports.initUTSSwiftAutoImportsOnce = initUTSSwiftAutoImportsOnce;
exports.genUniExtApiDeclarationFileOnce = once((tscInputDir) => {
    const extApis = (0, uni_modules_1.parseUniExtApis)(true, 'app-android', 'kotlin');
    // 之所以往上一级写，是因为 tscInputDir 会被 empty，目前时机有问题，比如先生成了d.ts，又被empty
    const fileName = path_1.default.resolve(tscInputDir, '../uni-ext-api.d.ts');
    if (fs_extra_1.default.existsSync(fileName)) {
        try {
            // 先删除
            fs_extra_1.default.unlinkSync(fileName);
        }
        catch (e) { }
    }
    if (Object.keys(extApis).length) {
        const apis = [];
        for (const name in extApis) {
            const options = extApis[name];
            if ((0, utils_1.isArray)(options) && options.length >= 2) {
                const api = name.replace('uni.', '');
                apis.push('  ' + api + `: typeof import("${options[0]}")["${options[1]}"]`);
            }
        }
        if (apis.length) {
            fs_extra_1.default.outputFileSync(fileName, `
interface Uni {
${apis.join('\n')}
}
`);
        }
    }
});
function uvueOutDir(platform) {
    return path_1.default.join(process.env.UNI_APP_X_UVUE_DIR, platform);
}
exports.uvueOutDir = uvueOutDir;
function tscOutDir(platform) {
    return path_1.default.join(process.env.UNI_APP_X_TSC_DIR, platform);
}
exports.tscOutDir = tscOutDir;
const UTSProxyRE = /\?uts-proxy$/;
const UniHelpersRE = /\?uni_helpers$/;
function isUTSProxy(id) {
    return UTSProxyRE.test(id);
}
exports.isUTSProxy = isUTSProxy;
function isUniHelpers(id) {
    return UniHelpersRE.test(id);
}
exports.isUniHelpers = isUniHelpers;

}, function(modId) { var map = {"./hbx":1781362618896,"./x":1781362618905,"../package.json":1781362618906}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618905, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.isUniAppXAndroidNative = exports.isUniAppXAndroidJsEngine = exports.isUniAppXJsEngine = exports.isUniAppXAndroidVapor = exports.isUniAppXVapor = exports.isUniAppXIOS = exports.isUniAppXAndroid = exports.isUniAppX = void 0;
function isUniAppX() {
    return process.env.UNI_APP_X === 'true';
}
exports.isUniAppX = isUniAppX;
function isUniAppXAndroid(platform = process.env.UNI_UTS_PLATFORM) {
    return isUniAppX() && platform === 'app-android';
}
exports.isUniAppXAndroid = isUniAppXAndroid;
function isUniAppXIOS(platform = process.env.UNI_UTS_PLATFORM) {
    return isUniAppX() && platform === 'app-ios';
}
exports.isUniAppXIOS = isUniAppXIOS;
function isUniAppXVapor() {
    return isUniAppX() && process.env.UNI_APP_X_DOM2 === 'true';
}
exports.isUniAppXVapor = isUniAppXVapor;
function isUniAppXAndroidVapor(platform = process.env.UNI_UTS_PLATFORM) {
    return isUniAppXAndroid(platform) && process.env.UNI_APP_X_DOM2 === 'true';
}
exports.isUniAppXAndroidVapor = isUniAppXAndroidVapor;
function isUniAppXJsEngine() {
    return isUniAppX() && process.env.UNI_APP_X_UVUE_SCRIPT_ENGINE === 'js';
}
exports.isUniAppXJsEngine = isUniAppXJsEngine;
function isUniAppXAndroidJsEngine(platform = process.env.UNI_UTS_PLATFORM) {
    return (isUniAppXAndroid(platform) &&
        process.env.UNI_APP_X_UVUE_SCRIPT_ENGINE === 'js');
}
exports.isUniAppXAndroidJsEngine = isUniAppXAndroidJsEngine;
function isUniAppXAndroidNative(platform = process.env.UNI_UTS_PLATFORM) {
    return (isUniAppXAndroid(platform) &&
        // 非 Vapor 或者 Vapor 但使用 native 引擎
        (process.env.UNI_APP_X_DOM2 !== 'true' ||
            process.env.UNI_APP_X_UVUE_SCRIPT_ENGINE === 'native'));
}
exports.isUniAppXAndroidNative = isUniAppXAndroidNative;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618906, function(require, module, exports) {
module.exports = {
  "name": "@dcloudio/uni-cli-shared",
  "version": "3.0.0-alpha-5000820260420001",
  "description": "@dcloudio/uni-cli-shared",
  "main": "dist/index.js",
  "types": "dist/index.d.ts",
  "files": [
    "dist",
    "lib"
  ],
  "engines": {
    "node": "^14.18.0 || >=16.0.0"
  },
  "repository": {
    "type": "git",
    "url": "git+https://github.com/dcloudio/uni-app.git",
    "directory": "packages/uni-cli-shared"
  },
  "license": "Apache-2.0",
  "bugs": {
    "url": "https://github.com/dcloudio/uni-app/issues"
  },
  "dependencies": {
    "@ampproject/remapping": "^2.1.2",
    "@babel/code-frame": "7.24.7",
    "@babel/core": "7.25.2",
    "@babel/parser": "7.25.6",
    "@babel/types": "7.25.6",
    "@intlify/core-base": "9.1.9",
    "@intlify/shared": "9.1.9",
    "@intlify/vue-devtools": "9.1.9",
    "@rollup/pluginutils": "5.1.0",
    "@vue/compiler-core": "3.4.21",
    "@vue/compiler-dom": "3.4.21",
    "@vue/compiler-sfc": "3.4.21",
    "@vue/compiler-ssr": "3.4.21",
    "@vue/server-renderer": "3.4.21",
    "@vue/shared": "3.4.21",
    "adm-zip": "0.5.16",
    "autoprefixer": "10.4.20",
    "base64url": "^3.0.1",
    "chokidar": "3.6.0",
    "compare-versions": "^3.6.0",
    "debug": "4.3.7",
    "entities": "^7.0.0",
    "es-module-lexer": "1.5.4",
    "esbuild": "0.20.2",
    "estree-walker": "2.0.2",
    "fast-glob": "3.3.3",
    "fs-extra": "10.1.0",
    "hash-sum": "2.0.0",
    "isbinaryfile": "5.0.2",
    "jsonc-parser": "3.3.1",
    "lines-and-columns": "^2.0.4",
    "magic-string": "0.30.11",
    "merge": "2.1.1",
    "mime": "3.0.0",
    "module-alias": "2.2.3",
    "os-locale-s-fix": "^1.0.8-fix-1",
    "picocolors": "1.1.0",
    "postcss-import": "^14.0.2",
    "postcss-load-config": "^3.1.1",
    "postcss-modules": "^4.3.0",
    "postcss-selector-parser": "6.1.2",
    "resolve": "1.22.8",
    "source-map-js": "1.2.1",
    "tapable": "^2.2.0",
    "tinycolor2": "1.6.0",
    "unimport": "4.1.1",
    "unplugin-auto-import": "19.1.0",
    "xregexp": "5.1.2",
    "@dcloudio/uni-i18n": "3.0.0-alpha-5000820260420001",
    "@dcloudio/uni-nvue-styler": "3.0.0-alpha-5000820260420001",
    "@dcloudio/uni-shared": "3.0.0-alpha-5000820260420001"
  },
  "gitHead": "33e807d66e1fe47e2ee08ad9c59247e37b8884da",
  "devDependencies": {
    "@types/adm-zip": "0.5.5",
    "@types/babel__code-frame": "^7.0.6",
    "@types/babel__core": "^7.1.19",
    "@types/debug": "4.1.12",
    "@types/estree": "1.0.5",
    "@types/fs-extra": "9.0.13",
    "@types/hash-sum": "^1.0.0",
    "@types/less": "^3.0.3",
    "@types/mime": "2.0.3",
    "@types/module-alias": "2.0.4",
    "@types/resolve": "1.20.6",
    "@types/sass": "1.43.1",
    "@types/stylus": "^0.48.36",
    "code-frame": "link:@types/@babel/code-frame",
    "postcss": "8.4.45",
    "vue": "3.4.21",
    "@dcloudio/uni-uts-v1": "3.0.0-alpha-5000820260420001"
  }
}
}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618908, function(require, module, exports) {

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
__exportStar(require("./sharedData"), exports);
__exportStar(require("./vue"), exports);
__exportStar(require("./fontFamily"), exports);

}, function(modId) { var map = {"./sharedData":1781362618909,"./vue":1781362618915,"./fontFamily":1781362618989}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618909, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.initUts2jsSharedDataOptions = exports.initSourceFileCallback = exports.uniSharedDataPlugin = void 0;
const uts_1 = require("../uts");
const pages_1 = require("../json/pages");
const utils_1 = require("../utils");
const asset_1 = require("../vite/plugins/vitejs/plugins/asset");
const x_1 = require("../x");
const manifest_1 = require("../json/uni-x/manifest");
const manifest_2 = require("../json/manifest");
function initSharedDataOptions() {
    const compiler = require('@dcloudio/compiler-vapor-dom2');
    const manifest = (0, manifest_2.parseManifestJsonOnce)(process.env.UNI_INPUT_DIR);
    return {
        platform: process.env.UNI_UTS_PLATFORM,
        compilerVaporDom2: compiler,
        utsCompiler: (0, uts_1.resolveUTSCompiler)(),
        isUniPageFile: pages_1.isUniPageFile,
        getSharedDataResult: compiler.getSharedDataResult,
        getAssetFilenameById: asset_1.getAssetFilenameById,
        uvueScriptEngine: (0, x_1.isUniAppXAndroidNative)() ? 'native' : 'js',
        compilerVersion: process.env.HX_Version || process.env.UNI_COMPILER_VERSION,
        androidOptions: (0, x_1.isUniAppXAndroidJsEngine)()
            ? {
                package: (0, manifest_1.parseUniXAppAndroidPackage)(manifest.appid),
            }
            : undefined,
    };
}
function uniSharedDataPlugin() {
    return (0, utils_1.requireUniHelpers)().USDP(initSharedDataOptions());
}
exports.uniSharedDataPlugin = uniSharedDataPlugin;
function initSourceFileCallback() {
    if (process.env.UNI_APP_X_DOM2 === 'true' && (0, x_1.isUniAppXAndroidNative)()) {
        const { TSDBSF } = (0, utils_1.requireUniHelpers)();
        const options = initSharedDataOptions();
        return (sourceFile) => {
            TSDBSF(sourceFile, options);
        };
    }
}
exports.initSourceFileCallback = initSourceFileCallback;
function initUts2jsSharedDataOptions() {
    if (process.env.UNI_APP_X_DOM2 === 'true') {
        return {
            resolveFieldMeta: require('@dcloudio/compiler-vapor-dom2')
                .resolveSharedDataFieldMeta,
        };
    }
}
exports.initUts2jsSharedDataOptions = initUts2jsSharedDataOptions;

}, function(modId) { var map = {"../uts":1781362618901,"../x":1781362618905}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618915, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.initVueTemplateCompilerExtraOptions = void 0;
const path_1 = __importDefault(require("path"));
const utils_1 = require("../utils");
const json_1 = require("../json");
const vue_1 = require("../vue");
function initVueTemplateCompilerExtraOptions(descriptor) {
    const filename = (0, utils_1.normalizePath)(descriptor.filename.split('?')[0]);
    const relativeFilename = (0, utils_1.normalizePath)(path_1.default.relative(process.env.UNI_INPUT_DIR, filename));
    const isDevX = process.env.UNI_HX_VERSION_DEV === 'true' &&
        process.env.UNI_APP_X === 'true';
    let disableStaticStyle = false;
    if (isDevX && process.env.NODE_ENV === 'development') {
        if (process.env.UNI_UTS_PLATFORM === 'app-harmony') {
            // 开发版本、开发模式下，非鸿蒙release模式打包
            disableStaticStyle = process.env.UNI_APP_HARMONY_RUN_MODE !== 'release';
        }
    }
    const helper = (0, utils_1.requireUniHelpers)();
    return {
        root: (0, utils_1.normalizePath)(process.env.UNI_INPUT_DIR),
        platform: process.env.UNI_UTS_PLATFORM,
        componentType: (0, json_1.isUniPageFile)(filename) ? 'page' : 'component',
        filename: filename,
        relativeFilename,
        helper,
        scriptCppBlocks: descriptor.scriptCppBlocks,
        disableStaticStyle,
        onVueTemplateCompileLog(type, error) {
            return (0, vue_1.onVueTemplateCompileLog)(type, error, descriptor.source, relativeFilename);
        },
        r: helper.K,
        className: helper.GCN(descriptor.filename, process.env.UNI_INPUT_DIR),
        inlineRender: process.env.UNI_UTS_PLATFORM === 'app-android',
    };
}
exports.initVueTemplateCompilerExtraOptions = initVueTemplateCompilerExtraOptions;

}, function(modId) { var map = {"../json":1781362618806,"../vue":1781362618917}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618917, function(require, module, exports) {

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
exports.isExternalUrl = exports.transformUniH5Jsx = void 0;
__exportStar(require("./transforms"), exports);
__exportStar(require("./utils"), exports);
__exportStar(require("./parse"), exports);
var babel_1 = require("./babel");
Object.defineProperty(exports, "transformUniH5Jsx", { enumerable: true, get: function () { return babel_1.transformUniH5Jsx; } });
var templateUtils_1 = require("./transforms/templateUtils");
Object.defineProperty(exports, "isExternalUrl", { enumerable: true, get: function () { return templateUtils_1.isExternalUrl; } });

}, function(modId) { var map = {"./transforms":1781362618918,"./parse":1781362618949,"./babel":1781362618988,"./transforms/templateUtils":1781362618936}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618918, function(require, module, exports) {

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
exports.transformComponentLink = exports.transformTapToClick = exports.transformMatchMedia = exports.transformH5BuiltInComponents = exports.matchTransformModel = exports.createTransformModel = exports.matchTransformOn = exports.createTransformOn = exports.ATTR_DATASET_EVENT_OPTS = exports.STRINGIFY_JSON = exports.createSrcsetTransformWithOptions = exports.createAssetUrlTransformWithOptions = void 0;
const uni_shared_1 = require("@dcloudio/uni-shared");
const transformTag_1 = require("./transformTag");
const transformEvent_1 = require("./transformEvent");
const transformComponent_1 = require("./transformComponent");
const constants_1 = require("../../mp/constants");
__exportStar(require("./transformLineBreak"), exports);
__exportStar(require("./transformRef"), exports);
__exportStar(require("./transformPageHead"), exports);
__exportStar(require("./transformComponent"), exports);
__exportStar(require("./transformEvent"), exports);
__exportStar(require("./transformTag"), exports);
__exportStar(require("./transformUTSComponent"), exports);
__exportStar(require("./transformRefresherSlot"), exports);
var templateTransformAssetUrl_1 = require("./templateTransformAssetUrl");
Object.defineProperty(exports, "createAssetUrlTransformWithOptions", { enumerable: true, get: function () { return templateTransformAssetUrl_1.createAssetUrlTransformWithOptions; } });
var templateTransformSrcset_1 = require("./templateTransformSrcset");
Object.defineProperty(exports, "createSrcsetTransformWithOptions", { enumerable: true, get: function () { return templateTransformSrcset_1.createSrcsetTransformWithOptions; } });
var vOn_1 = require("./vOn");
Object.defineProperty(exports, "STRINGIFY_JSON", { enumerable: true, get: function () { return vOn_1.STRINGIFY_JSON; } });
Object.defineProperty(exports, "ATTR_DATASET_EVENT_OPTS", { enumerable: true, get: function () { return vOn_1.ATTR_DATASET_EVENT_OPTS; } });
Object.defineProperty(exports, "createTransformOn", { enumerable: true, get: function () { return vOn_1.createTransformOn; } });
Object.defineProperty(exports, "matchTransformOn", { enumerable: true, get: function () { return vOn_1.defaultMatch; } });
var vModel_1 = require("./vModel");
Object.defineProperty(exports, "createTransformModel", { enumerable: true, get: function () { return vModel_1.createTransformModel; } });
Object.defineProperty(exports, "matchTransformModel", { enumerable: true, get: function () { return vModel_1.defaultMatch; } });
exports.transformH5BuiltInComponents = (0, transformTag_1.createTransformTag)(uni_shared_1.BUILT_IN_TAG_NAMES.reduce((tags, tag) => ((tags[tag] = uni_shared_1.COMPONENT_PREFIX + tag), tags), {}));
exports.transformMatchMedia = (0, transformTag_1.createTransformTag)({
    'match-media': 'uni-match-media',
});
exports.transformTapToClick = (0, transformEvent_1.createTransformEvent)({
    tap: (node) => {
        // 地图组件有自己特定的 tap 事件
        if (node.tag === 'map' || node.tag === 'v-uni-map') {
            return 'tap';
        }
        return 'click';
    },
});
exports.transformComponentLink = (0, transformComponent_1.createTransformComponentLink)(constants_1.COMPONENT_BIND_LINK);
__exportStar(require("./x/transformMPBuiltInTag"), exports);
__exportStar(require("./x/transformDirection"), exports);

}, function(modId) { var map = {"./transformTag":1781362618919,"./transformEvent":1781362618920,"./transformComponent":1781362618921,"../../mp/constants":1781362618867,"./transformLineBreak":1781362618923,"./transformRef":1781362618924,"./transformPageHead":1781362618926,"./transformUTSComponent":1781362618928,"./templateTransformAssetUrl":1781362618935,"./templateTransformSrcset":1781362618937,"./vOn":1781362618938,"./vModel":1781362618940,"./x/transformMPBuiltInTag":1781362618942}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618919, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.createTransformTag = void 0;
const compiler_core_1 = require("@vue/compiler-core");
const ast_1 = require("../../vite/utils/ast");
function createTransformTag(opts) {
    return function transformTag(node, context) {
        if (!(0, ast_1.isElementNode)(node)) {
            return;
        }
        const oldTag = node.tag;
        const newTag = opts[oldTag];
        if (!newTag) {
            return;
        }
        // TODO: 临时 dom2 硬编码处理 tagType，待后续优化
        if (process.env.UNI_APP_X_DOM2 === 'true' && oldTag === 'cover-view') {
            node.tagType = compiler_core_1.ElementTypes.ELEMENT;
        }
        node.tag = newTag;
    };
}
exports.createTransformTag = createTransformTag;

}, function(modId) { var map = {"../../vite/utils/ast":1781362618875}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618920, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.createTransformEvent = void 0;
const shared_1 = require("@vue/shared");
const ast_1 = require("../../vite/utils/ast");
function createTransformEvent(options) {
    return function transformEvent(node) {
        if (!(0, ast_1.isElementNode)(node)) {
            return;
        }
        node.props.forEach((prop) => {
            const { name, arg } = prop;
            if (name === 'on' && arg && (0, ast_1.isSimpleExpressionNode)(arg)) {
                const eventType = options[arg.content];
                if (eventType) {
                    // e.g tap => click
                    if ((0, shared_1.isFunction)(eventType)) {
                        arg.content = eventType(node, prop);
                    }
                    else {
                        arg.content = eventType;
                    }
                }
            }
        });
    };
}
exports.createTransformEvent = createTransformEvent;

}, function(modId) { var map = {"../../vite/utils/ast":1781362618875}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618921, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.createTransformComponentLink = void 0;
const compiler_core_1 = require("@vue/compiler-core");
const utils_1 = require("../utils");
function createTransformComponentLink(name, type = compiler_core_1.NodeTypes.DIRECTIVE) {
    return function transformComponentLink(node, context) {
        if (!(0, utils_1.isUserComponent)(node, context)) {
            return;
        }
        // 新版本的 vue，识别 template 有差异，可能认为是自定义组件
        if (node.tag === 'template') {
            return;
        }
        if (type === compiler_core_1.NodeTypes.DIRECTIVE) {
            node.props.push({
                type: compiler_core_1.NodeTypes.DIRECTIVE,
                name: 'on',
                modifiers: [],
                loc: compiler_core_1.locStub,
                arg: (0, compiler_core_1.createSimpleExpression)(name, true),
                exp: (0, compiler_core_1.createSimpleExpression)('__l', true),
            });
        }
        else {
            node.props.push((0, utils_1.createAttributeNode)(name, '__l'));
        }
    };
}
exports.createTransformComponentLink = createTransformComponentLink;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618923, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.transformLineBreak = void 0;
const transformLineBreak = (node, context) => {
    if (node.type === 2) {
        const parent = context.parent;
        if (parent && parent.type === 1 && parent.tag === 'text') {
            // 解析文本节点转义，暂时仅处理换行
            node.content = node.content.replace(/[\\]+n/g, function (match) {
                return JSON.parse(`"${match}"`);
            });
        }
    }
};
exports.transformLineBreak = transformLineBreak;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618924, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.transformRef = void 0;
const compiler_core_1 = require("@vue/compiler-core");
const utils_1 = require("../utils");
function transformRef(node, context) {
    if (!(0, utils_1.isUserComponent)(node, context)) {
        return;
    }
    addVueRef(node, context);
}
exports.transformRef = transformRef;
function addVueRef(node, context) {
    // 仅配置了 ref 属性的，才需要增补 vue-ref
    const refProp = (0, compiler_core_1.findProp)(node, 'ref');
    if (!refProp) {
        return;
    }
    if (refProp.type === compiler_core_1.NodeTypes.ATTRIBUTE) {
        refProp.name = 'u-' + utils_1.VUE_REF;
    }
    else {
        ;
        refProp.arg.content = 'u-' + utils_1.VUE_REF;
    }
    return (0, utils_1.addStaticClass)(node, 
    // ref-in-for
    // ref
    context.inVFor
        ? utils_1.VUE_REF_IN_FOR
        : utils_1.VUE_REF);
}

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618926, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.transformPageHead = void 0;
const compiler_core_1 = require("@vue/compiler-core");
const utils_1 = require("../../utils");
const transformPageHead = (node, context) => {
    // 发现是page-meta下的head,直接remove该节点
    if ((0, utils_1.checkElementNodeTag)(node, 'head') &&
        (0, utils_1.checkElementNodeTag)(context.parent, 'page-meta')) {
        ;
        node.tag = 'page-meta-head';
        node.tagType = compiler_core_1.ElementTypes.COMPONENT;
    }
};
exports.transformPageHead = transformPageHead;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618928, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.transformUTSComponent = void 0;
const ast_1 = require("../../vite/utils/ast");
const utsUtils_1 = require("../../utsUtils");
const uts_1 = require("../../uts");
/**
 * 将uts组件保存到自定义组件列表中
 * @param node
 * @param context
 * @returns
 */
const transformUTSComponent = (node, context) => {
    if (!(0, ast_1.isElementNode)(node)) {
        return;
    }
    // @ts-expect-error 同时兼容 vapor 编译器
    const components = context.component || context.components;
    if (!components) {
        return;
    }
    // 1. 增加components，让sfc生成resolveComponent代码
    // 2. easycom插件会根据resolveComponent生成import插件代码触发编译
    const utsCustomElement = (0, uts_1.getUTSCustomElement)(node.tag);
    if (utsCustomElement) {
        components.add(node.tag);
    }
    else if ((0, utsUtils_1.matchUTSComponent)(node.tag)) {
        if (!components.has(node.tag)) {
            components.add(node.tag);
        }
    }
};
exports.transformUTSComponent = transformUTSComponent;

}, function(modId) { var map = {"../../vite/utils/ast":1781362618875,"../../utsUtils":1781362618929,"../../uts":1781362618901}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618929, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.verifySymbol = exports.genUTSClassName = exports.matchUTSComponent = void 0;
const easycom_1 = require("./easycom");
const utils_1 = require("./utils");
function matchUTSComponent(tag) {
    const source = (0, easycom_1.matchEasycom)(tag);
    return !!(source && source.includes('uts-proxy'));
}
exports.matchUTSComponent = matchUTSComponent;
function genUTSClassName(fileName, prefix = 'Gen') {
    return (prefix +
        (0, utils_1.capitalize)((0, utils_1.camelize)(verifySymbol((0, utils_1.removeExt)((0, utils_1.normalizeNodeModules)(fileName)
            .replace(/[\/|_]/g, '-')
            .replace(/-+/g, '-'))))));
}
exports.genUTSClassName = genUTSClassName;
function isValidStart(c) {
    return !!c.match(/^[A-Za-z_-]$/);
}
function isValidContinue(c) {
    return !!c.match(/^[A-Za-z0-9_-]$/);
}
function verifySymbol(s) {
    const chars = Array.from(s);
    if (isValidStart(chars[0]) && chars.slice(1).every(isValidContinue)) {
        return s;
    }
    const buf = [];
    let hasStart = false;
    for (const c of chars) {
        if (!hasStart && isValidStart(c)) {
            hasStart = true;
            buf.push(c);
        }
        else if (isValidContinue(c)) {
            buf.push(c);
        }
    }
    if (buf.length === 0) {
        buf.push('_');
    }
    return buf.join('');
}
exports.verifySymbol = verifySymbol;

}, function(modId) { var map = {"./easycom":1781362618930}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618930, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.genUTSComponentPublicInstanceImported = exports.genUTSComponentPublicInstanceIdent = exports.addUTSEasyComAutoImports = exports.getUTSEasyComAutoImports = exports.UNI_EASYCOM_EXCLUDE = exports.genResolveEasycomCode = exports.addImportDeclaration = exports.matchEasycom = exports.initEasycomsOnce = exports.initEasycoms = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const debug_1 = __importDefault(require("debug"));
const shared_1 = require("@vue/shared");
const pluginutils_1 = require("@rollup/pluginutils");
const uni_shared_1 = require("@dcloudio/uni-shared");
const x_1 = require("./x");
const utils_1 = require("./utils");
const pages_1 = require("./json/pages");
const messages_1 = require("./messages");
const uts_1 = require("./uts");
const utsUtils_1 = require("./utsUtils");
const debugEasycom = (0, debug_1.default)('uni:easycom');
const easycoms = [];
const easycomsCache = new Map();
const easycomsInvalidCache = new Set();
let hasEasycom = false;
function clearEasycom() {
    easycoms.length = 0;
    easycomsCache.clear();
    easycomsInvalidCache.clear();
}
function initEasycoms(inputDir, { dirs, platform, isX, }) {
    const componentsDir = path_1.default.resolve(inputDir, 'components');
    const uniModulesDir = path_1.default.resolve(inputDir, 'uni_modules');
    const initEasycomOptions = (pagesJson) => {
        // 初始化时，从once中读取缓存，refresh时，实时读取
        const { easycom } = pagesJson || (0, pages_1.parsePagesJson)(inputDir, platform, false);
        const easycomOptions = {
            platform,
            isX,
            dirs: easycom && easycom.autoscan === false
                ? [...dirs] // 禁止自动扫描
                : [
                    ...dirs,
                    componentsDir,
                    ...initUniModulesEasycomDirs(uniModulesDir),
                ],
            rootDir: inputDir,
            autoscan: !!(easycom && easycom.autoscan),
            custom: (easycom && easycom.custom) || {},
            extensions: [...(isX ? ['.uvue'] : []), ...['.vue', '.jsx', '.tsx']],
        };
        debugEasycom(easycomOptions);
        return easycomOptions;
    };
    const easyComOptions = initEasycomOptions((0, pages_1.parsePagesJsonOnce)(inputDir, platform));
    const initUTSEasycom = () => {
        (0, uts_1.initUTSComponents)(inputDir, platform).forEach((item) => {
            const index = easycoms.findIndex((easycom) => item.name === easycom.name);
            if (index > -1) {
                easycoms.splice(index, 1, item);
            }
            else {
                easycoms.push(item);
            }
        });
        if (isX && globalThis.uts2jsSourceCodeMap) {
            ;
            globalThis.uts2jsSourceCodeMap.initUts2jsEasycom(easycoms);
        }
    };
    const initUTSEasycomCustomElements = () => {
        (0, uts_1.initUTSCustomElements)(inputDir, platform).forEach((item) => {
            const index = easycoms.findIndex((easycom) => item.name === easycom.name);
            if (index > -1) {
                easycoms.splice(index, 1, item);
            }
            else {
                easycoms.push(item);
            }
        });
    };
    const supportCustomElements = isX && process.env.UNI_APP_X_DOM2 !== 'true';
    // ext-api 模式下，不存在 easycom 特性
    if (process.env.UNI_COMPILE_TARGET !== 'ext-api') {
        clearEasycom();
        (0, uts_1.clearUTSComponents)();
        (0, uts_1.clearUTSCustomElements)();
        initEasycom(easyComOptions);
        // dom2 模式下，不需要注册 customElements 组件
        if (supportCustomElements) {
            initUTSEasycomCustomElements();
        }
        initUTSEasycom();
    }
    const componentExtNames = isX ? 'uvue|vue' : 'vue';
    const res = {
        easyComOptions,
        filter: (0, pluginutils_1.createFilter)([
            'components/*/*.(' + componentExtNames + '|jsx|tsx)',
            'uni_modules/*/components/*/*.(' + componentExtNames + '|jsx|tsx)',
            'utssdk/*/**/*.(' + componentExtNames + ')',
            'uni_modules/*/utssdk/*/*.(' + componentExtNames + ')',
            ...(supportCustomElements
                ? ['uni_modules/*/customElements/*/*.uts']
                : []),
        ], [], {
            resolve: inputDir,
        }),
        refresh() {
            res.easyComOptions = initEasycomOptions();
            if (process.env.UNI_COMPILE_TARGET !== 'ext-api') {
                clearEasycom();
                (0, uts_1.clearUTSComponents)();
                (0, uts_1.clearUTSCustomElements)();
                initEasycom(res.easyComOptions);
                if (supportCustomElements) {
                    initUTSEasycomCustomElements();
                }
                initUTSEasycom();
            }
        },
        easycoms,
    };
    return res;
}
exports.initEasycoms = initEasycoms;
exports.initEasycomsOnce = (0, uni_shared_1.once)(initEasycoms);
function initUniModulesEasycomDirs(uniModulesDir, componentsDir = 'components') {
    if (!fs_1.default.existsSync(uniModulesDir)) {
        return [];
    }
    return fs_1.default
        .readdirSync(uniModulesDir)
        .map((uniModuleDir) => {
        const uniModuleComponentsDir = path_1.default.resolve(uniModulesDir, uniModuleDir, componentsDir);
        if (fs_1.default.existsSync(uniModuleComponentsDir)) {
            return uniModuleComponentsDir;
        }
    })
        .filter(Boolean);
}
function initEasycom({ platform, isX, dirs, rootDir, custom, extensions, }) {
    rootDir = (0, utils_1.normalizePath)(rootDir);
    const easycomsObj = Object.create(null);
    if (dirs && dirs.length && rootDir) {
        const autoEasyComObj = initAutoScanEasycoms(platform, dirs, rootDir, extensions);
        if (isX) {
            Object.keys(autoEasyComObj).forEach((tagName) => {
                let source = autoEasyComObj[tagName];
                tagName = tagName.slice(1, -1);
                if (path_1.default.isAbsolute(source) && source.startsWith(rootDir)) {
                    source = '@/' + (0, utils_1.normalizePath)(path_1.default.relative(rootDir, source));
                }
                // 加密插件easycom类型导入
                if (source.includes('?uts-proxy')) {
                    const moduleId = path_1.default.basename(source.split('?uts-proxy')[0]);
                    source = `uts.sdk.modules.${(0, shared_1.camelize)(moduleId)}`;
                }
                const ident = genUTSComponentPublicInstanceIdent(tagName);
                addUTSEasyComAutoImports(source, [ident, ident]);
            });
        }
        (0, shared_1.extend)(easycomsObj, autoEasyComObj);
    }
    if (custom) {
        Object.keys(custom).forEach((name) => {
            const componentPath = custom[name];
            easycomsObj[name] = componentPath.startsWith('@/')
                ? (0, utils_1.normalizePath)(path_1.default.join(rootDir, componentPath.slice(2)))
                : componentPath;
        });
    }
    Object.keys(easycomsObj).forEach((name) => {
        easycoms.push({
            name: name.startsWith('^') && name.endsWith('$') ? name.slice(1, -1) : name,
            pattern: new RegExp(name),
            replacement: easycomsObj[name],
        });
    });
    debugEasycom(easycoms);
    hasEasycom = !!easycoms.length;
    return easycoms;
}
function matchEasycom(tag) {
    if (!hasEasycom) {
        return;
    }
    let source = easycomsCache.get(tag);
    if (source) {
        return source;
    }
    if (easycomsInvalidCache.has(tag)) {
        return false;
    }
    const matcher = easycoms.find((matcher) => matcher.pattern.test(tag));
    if (!matcher) {
        easycomsInvalidCache.add(tag);
        return false;
    }
    source = tag.replace(matcher.pattern, matcher.replacement);
    easycomsCache.set(tag, source);
    debugEasycom('matchEasycom', tag, source);
    // 检查 H5 专用组件 ask163203
    const H5_ONLY_COMPONENTS = ['custom-tab-bar'];
    if (H5_ONLY_COMPONENTS.includes(tag) && process.env.UNI_PLATFORM !== 'h5') {
        console.warn(`[uni-app] 组件 <${tag}> 是 H5 平台专用组件，在 ${process.env.UNI_PLATFORM} 平台使用可能导致错误。如果你要引入自定义 tabBar 可重命名为其他名称。`);
    }
    return source;
}
exports.matchEasycom = matchEasycom;
const isDir = (path) => {
    const stat = fs_1.default.lstatSync(path);
    if (stat.isDirectory()) {
        return true;
    }
    else if (stat.isSymbolicLink()) {
        return fs_1.default.lstatSync(fs_1.default.realpathSync(path)).isDirectory();
    }
    return false;
};
function initAutoScanEasycom(platform, dir, rootDir, extensions) {
    if (!path_1.default.isAbsolute(dir)) {
        dir = path_1.default.resolve(rootDir, dir);
    }
    const easycoms = Object.create(null);
    if (!fs_1.default.existsSync(dir)) {
        return easycoms;
    }
    const isDevX = process.env.UNI_HX_VERSION_DEV === 'true' &&
        process.env.UNI_APP_X === 'true';
    const isMp = platform.startsWith('mp-');
    const is_uni_modules = path_1.default.basename(path_1.default.resolve(dir, '../..')) === 'uni_modules';
    const has_uts_sdk_dir = fs_1.default.existsSync(path_1.default.resolve(dir, '../utssdk'));
    const is_easycom_encrypt_uni_modules = // uni_modules模式不需要此逻辑
     process.env.UNI_COMPILE_TARGET !== 'uni_modules' &&
        is_uni_modules &&
        // 前端加密插件，不能包含utssdk目录
        fs_1.default.existsSync(path_1.default.resolve(dir, '../encrypt')) &&
        !has_uts_sdk_dir;
    const uni_modules_plugin_id = is_easycom_encrypt_uni_modules && path_1.default.basename(path_1.default.resolve(dir, '..'));
    fs_1.default.readdirSync(dir).forEach((name) => {
        const folder = path_1.default.resolve(dir, name);
        if (!isDir(folder)) {
            return;
        }
        if (isDevX) {
            if (has_uts_sdk_dir &&
                process.env.UNI_UTS_PLATFORM &&
                (0, uni_shared_1.isBuiltInComponent)(name)) {
                // dev下，如果是内置组件，且存在utssdk目录，则判断utssdk下是否存在当前平台，如果不存在，则跳过
                const utsSdkPlatformDir = path_1.default.resolve(dir, '../utssdk', process.env.UNI_UTS_PLATFORM);
                if (!fs_1.default.existsSync(utsSdkPlatformDir)) {
                    return;
                }
            }
        }
        const importDir = (0, utils_1.normalizePath)(folder);
        const files = fs_1.default.readdirSync(folder);
        // 读取文件夹文件列表，比对文件名（fs.existsSync在大小写不敏感的系统会匹配不准确）
        for (let i = 0; i < extensions.length; i++) {
            const ext = extensions[i];
            if (files.includes(name + ext)) {
                easycoms[`^${name}$`] =
                    // mp平台，这里不处理，由uniEntryPlugin处理
                    is_easycom_encrypt_uni_modules && !isMp
                        ? (0, utils_1.normalizePath)(path_1.default.join(rootDir, `uni_modules/${uni_modules_plugin_id}?${(0, x_1.isUniAppXAndroidNative)() ? 'uts-proxy' : 'uni_helpers'}`))
                        : `${importDir}/${name}${ext}`;
                break;
            }
        }
    });
    return easycoms;
}
function initAutoScanEasycoms(platform, dirs, rootDir, extensions) {
    const conflict = {};
    const res = dirs.reduce((easycoms, dir) => {
        const curEasycoms = initAutoScanEasycom(platform, dir, rootDir, extensions);
        Object.keys(curEasycoms).forEach((name) => {
            // Use the first component when name conflict
            const componentPath = easycoms[name];
            if (!componentPath) {
                easycoms[name] = curEasycoms[name];
            }
            else {
                ;
                (conflict[componentPath] || (conflict[componentPath] = [])).push(normalizeComponentPath(curEasycoms[name], rootDir));
            }
        });
        return easycoms;
    }, Object.create(null));
    const conflictComponents = Object.keys(conflict);
    if (conflictComponents.length) {
        console.warn(messages_1.M['easycom.conflict']);
        conflictComponents.forEach((com) => {
            console.warn([normalizeComponentPath(com, rootDir), conflict[com]].join(','));
        });
    }
    return res;
}
function normalizeComponentPath(componentPath, rootDir) {
    return (0, utils_1.normalizePath)(path_1.default.relative(rootDir, componentPath));
}
function addImportDeclaration(importDeclarations, local, source, imported) {
    importDeclarations.push(createImportDeclaration(local, source, imported));
    return local;
}
exports.addImportDeclaration = addImportDeclaration;
function createImportDeclaration(local, source, imported) {
    if (imported && local) {
        return `import { ${imported} as ${local} } from '${source}';`;
    }
    if (local) {
        return `import ${local} from '${source}';`;
    }
    return `import '${source}';`;
}
const RESOLVE_EASYCOM_IMPORT_CODE = `import { resolveDynamicComponent as __resolveDynamicComponent } from 'vue';import { resolveEasycom } from '@dcloudio/uni-app';`;
function genResolveEasycomCode(importDeclarations, code, name) {
    if (process.env.UNI_APP_X_DOM2 === 'true' ||
        // 内部 dev 版本，需要本地开发内置组件，比如web-view，此时需要直接返回本地easycom，不然会找到基座内置的
        (process.env.UNI_HX_VERSION_DEV === 'true' &&
            process.env.UNI_APP_X === 'true')) {
        // dom2 模式下，为了性能，不再考虑优先级问题，如果开发者需要区别，可以手动导入为其他名称
        return name;
    }
    if (!importDeclarations.includes(RESOLVE_EASYCOM_IMPORT_CODE)) {
        importDeclarations.push(RESOLVE_EASYCOM_IMPORT_CODE);
    }
    return `resolveEasycom(${code
        .replace('_resolveComponent', '__resolveDynamicComponent')
        // 移除 maybeSelfReference 逻辑，easycom 优先级高于自引用组件
        .replace(', true)', ')')}, ${name})`;
}
exports.genResolveEasycomCode = genResolveEasycomCode;
exports.UNI_EASYCOM_EXCLUDE = [/@dcloudio\/uni-h5/];
const utsEasyComAutoImports = {};
function getUTSEasyComAutoImports() {
    return utsEasyComAutoImports;
}
exports.getUTSEasyComAutoImports = getUTSEasyComAutoImports;
function addUTSEasyComAutoImports(source, imports) {
    if (!utsEasyComAutoImports[source]) {
        utsEasyComAutoImports[source] = [imports];
    }
    else {
        if (!utsEasyComAutoImports[source].find((item) => item[0] === imports[0])) {
            utsEasyComAutoImports[source].push(imports);
        }
    }
}
exports.addUTSEasyComAutoImports = addUTSEasyComAutoImports;
function genUTSComponentPublicInstanceIdent(tagName) {
    return (0, shared_1.capitalize)((0, shared_1.camelize)(tagName)) + 'ComponentPublicInstance';
}
exports.genUTSComponentPublicInstanceIdent = genUTSComponentPublicInstanceIdent;
function genUTSComponentPublicInstanceImported(root, fileName) {
    root = (0, utils_1.normalizePath)(root);
    if (path_1.default.isAbsolute(fileName) && fileName.startsWith(root)) {
        fileName = (0, utils_1.normalizePath)(path_1.default.relative(root, fileName));
    }
    if (fileName.startsWith('@/')) {
        return ((0, utsUtils_1.genUTSClassName)(fileName.replace('@/', '')) + 'ComponentPublicInstance');
    }
    return (0, utsUtils_1.genUTSClassName)(fileName) + 'ComponentPublicInstance';
}
exports.genUTSComponentPublicInstanceImported = genUTSComponentPublicInstanceImported;

}, function(modId) { var map = {"fs":1781362618793,"./x":1781362618905,"./messages":1781362618844,"./uts":1781362618901,"./utsUtils":1781362618929}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618935, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.transformAssetUrl = exports.createAssetUrlTransformWithOptions = exports.normalizeOptions = exports.defaultAssetUrlOptions = void 0;
const path_1 = __importDefault(require("path"));
const compiler_core_1 = require("@vue/compiler-core");
const templateUtils_1 = require("./templateUtils");
const shared_1 = require("@vue/shared");
exports.defaultAssetUrlOptions = {
    base: null,
    includeAbsolute: false,
    tags: {
        video: ['src', 'poster'],
        source: ['src'],
        img: ['src'],
        image: ['xlink:href', 'href'],
        use: ['xlink:href', 'href'],
    },
    resolveStaticAsset: null,
};
const normalizeOptions = (options) => {
    if (Object.keys(options).some((key) => (0, shared_1.isArray)(options[key]))) {
        // legacy option format which directly passes in tags config
        return {
            ...exports.defaultAssetUrlOptions,
            tags: options,
        };
    }
    return {
        ...exports.defaultAssetUrlOptions,
        ...options,
    };
};
exports.normalizeOptions = normalizeOptions;
const createAssetUrlTransformWithOptions = (options) => {
    return (node, context) => exports.transformAssetUrl(node, context, options);
};
exports.createAssetUrlTransformWithOptions = createAssetUrlTransformWithOptions;
/**
 * A `@vue/compiler-core` plugin that transforms relative asset urls into
 * either imports or absolute urls.
 *
 * ``` js
 * // Before
 * createVNode('img', { src: './logo.png' })
 *
 * // After
 * import _imports_0 from './logo.png'
 * createVNode('img', { src: _imports_0 })
 * ```
 */
const transformAssetUrl = (node, context, options = exports.defaultAssetUrlOptions) => {
    if (node.type === compiler_core_1.NodeTypes.ELEMENT) {
        if (!node.props.length) {
            return;
        }
        const tags = options.tags || exports.defaultAssetUrlOptions.tags;
        const attrs = tags[node.tag];
        const wildCardAttrs = tags['*'];
        if (!attrs && !wildCardAttrs) {
            return;
        }
        // 策略：
        // h5 平台保留原始策略
        // 非 h5 平台
        // - 绝对路径 static 资源不做转换
        // - 相对路径 static 资源转换为绝对路径
        // - 非 static 资源转换为 import
        const assetAttrs = (attrs || []).concat(wildCardAttrs || []);
        node.props.forEach((attr, index) => {
            if (attr.type !== compiler_core_1.NodeTypes.ATTRIBUTE ||
                !assetAttrs.includes(attr.name) ||
                !attr.value ||
                (0, templateUtils_1.isExternalUrl)(attr.value.content) ||
                (0, templateUtils_1.isDataUrl)(attr.value.content) ||
                attr.value.content[0] === '#') {
                return;
            }
            // fixed by xxxxxx 区分 static 资源
            const isStaticAsset = attr.value.content.indexOf('/static/') > -1;
            // 绝对路径的静态资源不作处理
            if (isStaticAsset && !(0, templateUtils_1.isRelativeUrl)(attr.value.content)) {
                return;
            }
            const url = (0, templateUtils_1.parseUrl)(attr.value.content);
            // 这里是有问题的，static的相对路径可能是分包里的，或者uni_modules里的，不能简单的通过base来合并
            // 只不过目前编译器非有意的同时保留了vue标准的transformAssetUrl和uni-app的transformAssetUrl
            // 当static相对路径经过vue的transformAssetUrl后，就变成了 import 语句，不会再走到下边的逻辑里
            // 最初的设计，应该是用uni-app的transformAssetUrl来直接替换vue的transformAssetUrl的。
            // 如果后续要替换，需要考虑这个问题
            if (attr.value.content[0] === '.' && isStaticAsset) {
                // resolveStaticAsset 里边会处理分包、uni_modules里的 static 资源
                // 为了减少影响范围，目前仅限蒸汽模式的App端使用吧。
                if (options.resolveStaticAsset) {
                    attr.value.content = options.resolveStaticAsset(attr.value.content, context, options);
                    return;
                }
                if (options.base) {
                    // explicit base - directly rewrite relative urls into absolute url
                    // to avoid generating extra imports
                    // Allow for full hostnames provided in options.base
                    const base = (0, templateUtils_1.parseUrl)(options.base);
                    const protocol = base.protocol || '';
                    const host = base.host ? protocol + '//' + base.host : '';
                    const basePath = base.path || '/';
                    // when packaged in the browser, path will be using the posix-
                    // only version provided by rollup-plugin-node-builtins.
                    attr.value.content =
                        host +
                            (path_1.default.posix || path_1.default).join(basePath, url.path + (url.hash || ''));
                    return;
                }
            }
            // otherwise, transform the url into an import.
            // this assumes a bundler will resolve the import into the correct
            // absolute url (e.g. webpack file-loader)
            const exp = getImportsExpressionExp(url.path, url.hash, attr.loc, context);
            node.props[index] = {
                type: compiler_core_1.NodeTypes.DIRECTIVE,
                name: 'bind',
                arg: (0, compiler_core_1.createSimpleExpression)(attr.name, true, attr.loc),
                exp,
                modifiers: [],
                loc: attr.loc,
            };
        });
    }
};
exports.transformAssetUrl = transformAssetUrl;
function getImportsExpressionExp(path, hash, loc, context) {
    if (path) {
        let name;
        let exp;
        const existingIndex = context.imports.findIndex((i) => i.path === path);
        if (existingIndex > -1) {
            name = `_imports_${existingIndex}`;
            exp = context.imports[existingIndex].exp;
        }
        else {
            name = `_imports_${context.imports.length}`;
            exp = (0, compiler_core_1.createSimpleExpression)(name, false, loc, compiler_core_1.ConstantTypes.CAN_HOIST);
            context.imports.push({ exp, path });
        }
        if (!hash) {
            return exp;
        }
        const hashExp = `${name} + '${hash}'`;
        const existingHoistIndex = context.hoists.findIndex((h) => {
            return (h &&
                h.type === compiler_core_1.NodeTypes.SIMPLE_EXPRESSION &&
                !h.isStatic &&
                h.content === hashExp);
        });
        if (existingHoistIndex > -1) {
            return (0, compiler_core_1.createSimpleExpression)(`_hoisted_${existingHoistIndex + 1}`, false, loc, compiler_core_1.ConstantTypes.CAN_HOIST);
        }
        return context.hoist((0, compiler_core_1.createSimpleExpression)(hashExp, false, loc, compiler_core_1.ConstantTypes.CAN_HOIST));
    }
    else {
        return (0, compiler_core_1.createSimpleExpression)(`''`, false, loc, compiler_core_1.ConstantTypes.CAN_HOIST);
    }
}

}, function(modId) { var map = {"./templateUtils":1781362618936}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618936, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.parseUrl = exports.isDataUrl = exports.isExternalUrl = exports.isRelativeUrl = void 0;
const url_1 = require("url");
const shared_1 = require("@vue/shared");
function isRelativeUrl(url) {
    const firstChar = url.charAt(0);
    return firstChar === '.' || firstChar === '~' || firstChar === '@';
}
exports.isRelativeUrl = isRelativeUrl;
const externalRE = /^(https?:)?\/\//;
function isExternalUrl(url) {
    return externalRE.test(url);
}
exports.isExternalUrl = isExternalUrl;
const dataUrlRE = /^\s*data:/i;
function isDataUrl(url) {
    return dataUrlRE.test(url);
}
exports.isDataUrl = isDataUrl;
/**
 * Parses string url into URL object.
 */
function parseUrl(url) {
    const firstChar = url.charAt(0);
    if (firstChar === '~') {
        const secondChar = url.charAt(1);
        url = url.slice(secondChar === '/' ? 2 : 1);
    }
    return parseUriParts(url);
}
exports.parseUrl = parseUrl;
/**
 * vuejs/component-compiler-utils#22 Support uri fragment in transformed require
 * @param urlString an url as a string
 */
function parseUriParts(urlString) {
    // A TypeError is thrown if urlString is not a string
    // @see https://nodejs.org/api/url.html#url_url_parse_urlstring_parsequerystring_slashesdenotehost
    return (0, url_1.parse)((0, shared_1.isString)(urlString) ? urlString : '', false, true);
}

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618937, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.transformSrcset = exports.createSrcsetTransformWithOptions = void 0;
const path_1 = __importDefault(require("path"));
const compiler_core_1 = require("@vue/compiler-core");
const templateUtils_1 = require("./templateUtils");
const templateTransformAssetUrl_1 = require("./templateTransformAssetUrl");
const srcsetTags = ['img', 'source'];
// http://w3c.github.io/html/semantics-embedded-content.html#ref-for-image-candidate-string-5
const escapedSpaceCharacters = /( |\\t|\\n|\\f|\\r)+/g;
const createSrcsetTransformWithOptions = (options) => {
    return (node, context) => exports.transformSrcset(node, context, options);
};
exports.createSrcsetTransformWithOptions = createSrcsetTransformWithOptions;
const transformSrcset = (node, context, options = templateTransformAssetUrl_1.defaultAssetUrlOptions) => {
    if (node.type === compiler_core_1.NodeTypes.ELEMENT) {
        if (srcsetTags.includes(node.tag) && node.props.length) {
            node.props.forEach((attr, index) => {
                if (attr.name === 'srcset' && attr.type === compiler_core_1.NodeTypes.ATTRIBUTE) {
                    if (!attr.value)
                        return;
                    const value = attr.value.content;
                    if (!value)
                        return;
                    const imageCandidates = value
                        .split(',')
                        .map((s) => {
                        // The attribute value arrives here with all whitespace, except
                        // normal spaces, represented by escape sequences
                        const [url, descriptor] = s
                            .replace(escapedSpaceCharacters, ' ')
                            .trim()
                            .split(' ', 2);
                        return { url, descriptor };
                    });
                    // data urls contains comma after the ecoding so we need to re-merge
                    // them
                    for (let i = 0; i < imageCandidates.length; i++) {
                        const { url } = imageCandidates[i];
                        if ((0, templateUtils_1.isDataUrl)(url)) {
                            imageCandidates[i + 1].url =
                                url + ',' + imageCandidates[i + 1].url;
                            imageCandidates.splice(i, 1);
                        }
                    }
                    const hasQualifiedUrl = imageCandidates.some(({ url }) => {
                        return (!(0, templateUtils_1.isExternalUrl)(url) &&
                            !(0, templateUtils_1.isDataUrl)(url) &&
                            (options.includeAbsolute || (0, templateUtils_1.isRelativeUrl)(url)));
                    });
                    // When srcset does not contain any qualified URLs, skip transforming
                    if (!hasQualifiedUrl) {
                        return;
                    }
                    if (options.base) {
                        const base = options.base;
                        const set = [];
                        imageCandidates.forEach(({ url, descriptor }) => {
                            descriptor = descriptor ? ` ${descriptor}` : ``;
                            if ((0, templateUtils_1.isRelativeUrl)(url)) {
                                set.push((path_1.default.posix || path_1.default).join(base, url) + descriptor);
                            }
                            else {
                                set.push(url + descriptor);
                            }
                        });
                        attr.value.content = set.join(', ');
                        return;
                    }
                    const compoundExpression = (0, compiler_core_1.createCompoundExpression)([], attr.loc);
                    imageCandidates.forEach(({ url, descriptor }, index) => {
                        if (!(0, templateUtils_1.isExternalUrl)(url) &&
                            !(0, templateUtils_1.isDataUrl)(url) &&
                            (options.includeAbsolute || (0, templateUtils_1.isRelativeUrl)(url))) {
                            const { path } = (0, templateUtils_1.parseUrl)(url);
                            let exp;
                            if (path) {
                                const existingImportsIndex = context.imports.findIndex((i) => i.path === path);
                                if (existingImportsIndex > -1) {
                                    exp = (0, compiler_core_1.createSimpleExpression)(`_imports_${existingImportsIndex}`, false, attr.loc, compiler_core_1.ConstantTypes.CAN_HOIST);
                                }
                                else {
                                    exp = (0, compiler_core_1.createSimpleExpression)(`_imports_${context.imports.length}`, false, attr.loc, compiler_core_1.ConstantTypes.CAN_HOIST);
                                    context.imports.push({ exp, path });
                                }
                                compoundExpression.children.push(exp);
                            }
                        }
                        else {
                            const exp = (0, compiler_core_1.createSimpleExpression)(`"${url}"`, false, attr.loc, compiler_core_1.ConstantTypes.CAN_HOIST);
                            compoundExpression.children.push(exp);
                        }
                        const isNotLast = imageCandidates.length - 1 > index;
                        if (descriptor && isNotLast) {
                            compoundExpression.children.push(` + ' ${descriptor}, ' + `);
                        }
                        else if (descriptor) {
                            compoundExpression.children.push(` + ' ${descriptor}'`);
                        }
                        else if (isNotLast) {
                            compoundExpression.children.push(` + ', ' + `);
                        }
                    });
                    const hoisted = context.hoist(compoundExpression);
                    hoisted.constType = compiler_core_1.ConstantTypes.CAN_HOIST;
                    node.props[index] = {
                        type: compiler_core_1.NodeTypes.DIRECTIVE,
                        name: 'bind',
                        arg: (0, compiler_core_1.createSimpleExpression)('srcset', true, attr.loc),
                        exp: hoisted,
                        modifiers: [],
                        loc: attr.loc,
                    };
                }
            });
        }
    }
};
exports.transformSrcset = transformSrcset;

}, function(modId) { var map = {"./templateUtils":1781362618936,"./templateTransformAssetUrl":1781362618935}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618938, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.STRINGIFY_JSON = exports.ATTR_DATASET_EVENT_OPTS = exports.addEventOpts = exports.createCustomEventExpr = exports.createTransformOn = exports.defaultMatch = void 0;
const uni_shared_1 = require("@dcloudio/uni-shared");
const compiler_core_1 = require("@vue/compiler-core");
const utils_1 = require("../utils");
function defaultMatch(name, node, context) {
    return isCustomEvent(name) && (0, utils_1.isUserComponent)(node, context);
}
exports.defaultMatch = defaultMatch;
/**
 * 百度、快手小程序的自定义组件，不支持动态事件绑定，故转换为静态事件 + dataset
 * @param baseTransformOn
 * @returns
 */
function createTransformOn(baseTransformOn, { match } = {
    match: defaultMatch,
}) {
    return (dir, node, context, augmentor) => {
        const res = baseTransformOn(dir, node, context, augmentor);
        const { name, arg, exp } = dir;
        if (name !== 'on' || !arg || !exp || !(0, compiler_core_1.isStaticExp)(arg)) {
            return res;
        }
        if (!match(arg.content, node, context)) {
            return res;
        }
        const value = res.props[0].value;
        res.props[0].value = createCustomEventExpr();
        addEventOpts(node.tagType === compiler_core_1.ElementTypes.COMPONENT
            ? (0, uni_shared_1.customizeEvent)(arg.content)
            : arg.content, value, node, context);
        return res;
    };
}
exports.createTransformOn = createTransformOn;
function createCustomEventExpr() {
    return (0, compiler_core_1.createSimpleExpression)('__e', true);
}
exports.createCustomEventExpr = createCustomEventExpr;
function addEventOpts(event, value, node, context) {
    const attrName = node.tagType === compiler_core_1.ElementTypes.COMPONENT
        ? ATTR_DATA_EVENT_OPTS
        : exports.ATTR_DATASET_EVENT_OPTS;
    const opts = (0, compiler_core_1.findProp)(node, attrName, true);
    if (!opts) {
        node.props.push(createDataEventOptsProp(attrName, event, value, context));
    }
    else {
        const children = opts.exp.children;
        children.splice(children.length - 2, 0, createDataEventOptsProperty(event, value));
    }
}
exports.addEventOpts = addEventOpts;
const ATTR_DATA_EVENT_OPTS = 'eO';
exports.ATTR_DATASET_EVENT_OPTS = 'data-e-o';
function createDataEventOptsProperty(event, exp) {
    return (0, compiler_core_1.createCompoundExpression)([`'${event}'`, ': ', exp, ',']);
}
exports.STRINGIFY_JSON = Symbol(`stringifyJson`);
function createDataEventOptsProp(name, event, exp, context) {
    const children = [];
    const stringify = name === ATTR_DATA_EVENT_OPTS;
    if (stringify) {
        children.push(context.helperString(exports.STRINGIFY_JSON) + '(');
    }
    children.push('{', createDataEventOptsProperty(event, exp), '}');
    if (stringify) {
        children.push(')');
    }
    return {
        type: compiler_core_1.NodeTypes.DIRECTIVE,
        name: 'bind',
        loc: compiler_core_1.locStub,
        modifiers: [],
        arg: (0, compiler_core_1.createSimpleExpression)(name, true),
        exp: (0, compiler_core_1.createCompoundExpression)(children),
    };
}
const builtInEvents = [
    '__l', // 快手使用了该事件
    'tap',
    'longtap',
    'longpress',
    'touchstart',
    'touchmove',
    'touchcancel',
    'touchend',
    'touchforcechange',
    'transitionend',
    'animationstart',
    'animationiteration',
    'animationend',
];
function isCustomEvent(name) {
    return !builtInEvents.includes(name);
}

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618940, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.createTransformModel = exports.defaultMatch = void 0;
const utils_1 = require("../utils");
const vOn_1 = require("./vOn");
function defaultMatch(node, context) {
    return (0, utils_1.isUserComponent)(node, context);
}
exports.defaultMatch = defaultMatch;
/**
 * 百度、快手小程序的自定义组件，不支持动态事件绑定，故 v-model 也需要调整
 * @param baseTransformModel
 * @returns
 */
function createTransformModel(baseTransformModel, { match } = {
    match: defaultMatch,
}) {
    return (dir, node, context, augmentor) => {
        const res = baseTransformModel(dir, node, context, augmentor);
        if (!match(node, context)) {
            return res;
        }
        const props = res.props;
        if (props[1]) {
            // input,textarea 的 v-model 事件可能会被合并到已有的 input 中
            const { arg, exp } = props[1];
            (0, vOn_1.addEventOpts)(arg.content, exp, node, context);
            props[1].exp = (0, vOn_1.createCustomEventExpr)();
        }
        return res;
    };
}
exports.createTransformModel = createTransformModel;

}, function(modId) { var map = {"./vOn":1781362618938}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618942, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.transformMPBuiltInTag = exports.createMPBuiltInTagTransform = exports.defaultTransformMPBuiltInTagOptions = void 0;
const shared_1 = require("@vue/shared");
const vite_1 = require("../../../vite");
const utils_1 = require("../../utils");
const compiler_core_1 = require("@vue/compiler-core");
exports.defaultTransformMPBuiltInTagOptions = {
    propRename: {
        checkbox: {
            // "backgroundColor": "",
            // "borderColor": "",
            // "activeBackgroundColor": "",
            // "activeBorderColor": "",
            foreColor: 'color',
        },
        radio: {
            // "backgroundColor": "",
            // "borderColor": "",
            activeBackgroundColor: 'color',
            // "activeBorderColor": "",
            // "foreColor": ""
        },
        slider: {
            backgroundColor: 'backgroundColor',
            activeBackgroundColor: 'activeColor',
            foreColor: 'block-color',
        },
        switch: {
            // "backgroundColor": "",
            activeBackgroundColor: 'color',
            // "foreColor": "",
            // "activeForeColor": ""
        },
        'rich-text': {
            selectable: 'user-select',
        },
    },
    propAdd: {
        canvas: [
            {
                name: 'type',
                value: '2d',
            },
        ],
        'scroll-view': [
            {
                name: 'enable-flex',
                value: 'true',
            },
            {
                name: 'enhanced',
                value: 'true',
            },
        ],
    },
    tagRename: {
        'list-view': 'scroll-view',
    },
};
function createMPBuiltInTagTransform(options) {
    return function (node, context) {
        if (!(0, vite_1.isElementNode)(node)) {
            return;
        }
        if (options.tagRename && node.tag in options.tagRename) {
            node.tag = options.tagRename[node.tag];
        }
        if (options.propRename && node.tag in options.propRename) {
            const propMap = options.propRename[node.tag];
            node.props.forEach((prop) => {
                if ((0, vite_1.isAttributeNode)(prop)) {
                    const propName = (0, shared_1.camelize)(prop.name);
                    if (propName in propMap && propMap[propName]) {
                        (0, utils_1.renameProp)(propMap[propName], prop);
                    }
                }
                else if ((0, vite_1.isDirectiveNode)(prop)) {
                    if (!prop.rawName || !prop.arg || !(0, compiler_core_1.isStaticExp)(prop.arg)) {
                        return;
                    }
                    const propName = (0, shared_1.camelize)(prop.rawName.slice(1));
                    if (propName in propMap && propMap[propName]) {
                        (0, utils_1.renameProp)(propMap[propName], prop);
                    }
                }
            });
        }
        if (options.propAdd && node.tag in options.propAdd) {
            const add = options.propAdd[node.tag];
            add.forEach(({ name, value }) => {
                if (node.props.some((item) => (0, utils_1.isPropNameEquals)(item, name))) {
                    return;
                }
                node.props.push((0, utils_1.createAttributeNode)(name, value));
            });
        }
    };
}
exports.createMPBuiltInTagTransform = createMPBuiltInTagTransform;
exports.transformMPBuiltInTag = createMPBuiltInTagTransform(exports.defaultTransformMPBuiltInTagOptions);

}, function(modId) { var map = {"../../../vite":1781362618943}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618943, function(require, module, exports) {

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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.rewriteExistsSyncHasRootFile = exports.cssTarget = exports.getIsStaticFile = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
var static_1 = require("./plugins/vitejs/plugins/static");
Object.defineProperty(exports, "getIsStaticFile", { enumerable: true, get: function () { return static_1.getIsStaticFile; } });
exports.cssTarget = 'chrome53';
__exportStar(require("./utils"), exports);
__exportStar(require("./plugins"), exports);
__exportStar(require("./features"), exports);
__exportStar(require("./autoImport"), exports);
__exportStar(require("./cloud"), exports);
__exportStar(require("./extApi"), exports);
// https://github.com/vitejs/vite/blob/aac2ef77521f66ddd908f9d97020b8df532148cf/packages/vite/src/node/server/searchRoot.ts#L38
// vite 在初始化阶段会执行 initTSConfck，此时会 searchForWorkspaceRoot，如果找到了 pnpm-workspace.yaml 文件，会将其作为 root
// HBuilderX 项目，root 一定是 UNI_INPUT_DIR，所以需要重写 fs.existsSync，不重写的话，可能会找错，
// 一旦找错目录，而该目录下有 N 多文件目录，会导致遍历及其缓慢
function rewriteExistsSyncHasRootFile() {
    const existsSync = fs_1.default.existsSync;
    const pnpmWorkspaceYaml = path_1.default.join(process.env.UNI_INPUT_DIR, 'pnpm-workspace.yaml');
    fs_1.default.existsSync = (path) => {
        if (path === pnpmWorkspaceYaml) {
            return true;
        }
        return existsSync(path);
    };
}
exports.rewriteExistsSyncHasRootFile = rewriteExistsSyncHasRootFile;

}, function(modId) { var map = {"./plugins/vitejs/plugins/static":1781362618944,"./utils":1781362618873,"./plugins":1781362618946,"./features":1781362618980,"./autoImport":1781362618981,"./extApi":1781362618984}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618944, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getIsStaticFile = exports.createIsStaticFile = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const utils_1 = require("../utils");
const json_1 = require("../../../../json/json");
const uniModulesStaticRe = /^uni_modules\/[^/]+\/static\//;
function createIsStaticFile() {
    let subPackageStatics = [];
    const pagesFilename = path_1.default.join(process.env.UNI_INPUT_DIR, 'pages.json');
    if (fs_1.default.existsSync(pagesFilename)) {
        const pagesJson = (0, json_1.parseJson)(fs_1.default.readFileSync(pagesFilename, 'utf8'), true, pagesFilename);
        subPackageStatics = (pagesJson.subPackages || pagesJson.subpackages || [])
            .filter((subPackage) => subPackage.root)
            .map((subPackage) => {
            return (0, utils_1.normalizePath)(path_1.default.join(subPackage.root, 'static')) + '/';
        });
    }
    return function isStaticFile(relativeFile, onlySubPackages = false // 是否只判断子包 static 下的文件
    ) {
        if (path_1.default.isAbsolute(relativeFile)) {
            relativeFile = (0, utils_1.normalizePath)(path_1.default.relative(process.env.UNI_INPUT_DIR, relativeFile));
        }
        if (onlySubPackages) {
            return subPackageStatics.some((s) => relativeFile.startsWith(s));
        }
        return (relativeFile.startsWith('static/') ||
            uniModulesStaticRe.test(relativeFile) ||
            subPackageStatics.some((s) => relativeFile.startsWith(s)));
    };
}
exports.createIsStaticFile = createIsStaticFile;
let isStaticFile;
function getIsStaticFile() {
    if (!isStaticFile) {
        isStaticFile = createIsStaticFile();
    }
    return isStaticFile;
}
exports.getIsStaticFile = getIsStaticFile;

}, function(modId) { var map = {"../../../../json/json":1781362618810}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618946, function(require, module, exports) {

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
exports.offsetToStartAndEnd = exports.locToStartAndEnd = exports.generateCodeFrame = exports.getCssDepMap = exports.rewriteScssReadFileSync = exports.commonjsProxyRE = exports.cssLangRE = exports.minifyCSS = exports.cssPostPlugin = exports.cssPlugin = exports.isCSSRequest = exports.getAssetHash = exports.parseAssets = exports.assetPlugin = exports.uniStatsPlugin = exports.uniViteSfcSrcImportPlugin = void 0;
__exportStar(require("./cssScoped"), exports);
__exportStar(require("./copy"), exports);
__exportStar(require("./inject"), exports);
__exportStar(require("./mainJs"), exports);
__exportStar(require("./jsonJs"), exports);
__exportStar(require("./console"), exports);
__exportStar(require("./dynamicImportPolyfill"), exports);
__exportStar(require("./uts/uni_modules"), exports);
__exportStar(require("./uts/uvue"), exports);
__exportStar(require("./uts/ext-api"), exports);
__exportStar(require("./easycom"), exports);
__exportStar(require("./json"), exports);
__exportStar(require("./pre"), exports);
__exportStar(require("./sourceMap"), exports);
var sfc_1 = require("./sfc");
Object.defineProperty(exports, "uniViteSfcSrcImportPlugin", { enumerable: true, get: function () { return sfc_1.uniViteSfcSrcImportPlugin; } });
var stats_1 = require("./stats");
Object.defineProperty(exports, "uniStatsPlugin", { enumerable: true, get: function () { return stats_1.uniStatsPlugin; } });
var asset_1 = require("./vitejs/plugins/asset");
Object.defineProperty(exports, "assetPlugin", { enumerable: true, get: function () { return asset_1.assetPlugin; } });
Object.defineProperty(exports, "parseAssets", { enumerable: true, get: function () { return asset_1.parseAssets; } });
Object.defineProperty(exports, "getAssetHash", { enumerable: true, get: function () { return asset_1.getAssetHash; } });
var css_1 = require("./vitejs/plugins/css");
Object.defineProperty(exports, "isCSSRequest", { enumerable: true, get: function () { return css_1.isCSSRequest; } });
Object.defineProperty(exports, "cssPlugin", { enumerable: true, get: function () { return css_1.cssPlugin; } });
Object.defineProperty(exports, "cssPostPlugin", { enumerable: true, get: function () { return css_1.cssPostPlugin; } });
Object.defineProperty(exports, "minifyCSS", { enumerable: true, get: function () { return css_1.minifyCSS; } });
Object.defineProperty(exports, "cssLangRE", { enumerable: true, get: function () { return css_1.cssLangRE; } });
Object.defineProperty(exports, "commonjsProxyRE", { enumerable: true, get: function () { return css_1.commonjsProxyRE; } });
Object.defineProperty(exports, "rewriteScssReadFileSync", { enumerable: true, get: function () { return css_1.rewriteScssReadFileSync; } });
Object.defineProperty(exports, "getCssDepMap", { enumerable: true, get: function () { return css_1.getCssDepMap; } });
var utils_1 = require("./vitejs/utils");
Object.defineProperty(exports, "generateCodeFrame", { enumerable: true, get: function () { return utils_1.generateCodeFrame; } });
Object.defineProperty(exports, "locToStartAndEnd", { enumerable: true, get: function () { return utils_1.locToStartAndEnd; } });
Object.defineProperty(exports, "offsetToStartAndEnd", { enumerable: true, get: function () { return utils_1.offsetToStartAndEnd; } });

}, function(modId) { var map = {"./cssScoped":1781362618947,"./copy":1781362618953,"./mainJs":1781362618958,"./jsonJs":1781362618960,"./dynamicImportPolyfill":1781362618963,"./uts/uvue":1781362618965,"./uts/ext-api":1781362618966,"./easycom":1781362618968,"./json":1781362618969,"./pre":1781362618971,"./stats":1781362618975}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618947, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniCssScopedPlugin = exports.uniRemoveCssScopedPlugin = exports.addScoped = void 0;
const path_1 = __importDefault(require("path"));
const debug_1 = __importDefault(require("debug"));
const constants_1 = require("../../constants");
const preprocess_1 = require("../../preprocess");
const parse_1 = require("../../vue/parse");
const utils_1 = require("../../utils");
const utils_2 = require("../../vue/utils");
const debugScoped = (0, debug_1.default)('uni:scoped');
const SCOPED_RE = /<style\s[^>]*scoped[^>]*>/i;
function addScoped(code) {
    return code.replace(/(<style\b[^><]*)>/gi, (str, $1) => {
        if ($1.includes('scoped')) {
            return str;
        }
        return `${$1} scoped>`;
    });
}
exports.addScoped = addScoped;
function removeScoped(code) {
    if (!SCOPED_RE.test(code)) {
        return code;
    }
    return code.replace(/(<style.*)scoped(.*>)/gi, '$1$2');
}
function uniRemoveCssScopedPlugin(_ = { filter: () => false }) {
    return {
        name: 'uni:css-remove-scoped',
        enforce: 'pre',
        transform(code, id) {
            if (!(0, utils_2.isVueSfcFile)(id))
                return null;
            debugScoped(id);
            return {
                code: removeScoped(code),
                map: null,
            };
        },
    };
}
exports.uniRemoveCssScopedPlugin = uniRemoveCssScopedPlugin;
function uniCssScopedPlugin({ filter } = { filter: () => false }) {
    const isNewStyleIsolation = process.env.UNI_APP_STYLE_ISOLATION_VERSION === '2';
    return {
        name: 'uni:css-scoped',
        enforce: 'pre',
        transform(code, id) {
            if (!filter(id))
                return null;
            debugScoped(id);
            return {
                code: addScoped(code),
                map: null,
            };
        },
        // 仅 h5
        handleHotUpdate(ctx) {
            if (!constants_1.EXTNAME_VUE.includes(path_1.default.extname(ctx.file))) {
                return;
            }
            const scoped = isNewStyleIsolation || !(0, utils_1.isAppVue)(ctx.file);
            debugScoped('hmr', ctx.file);
            const oldRead = ctx.read;
            ctx.read = async () => {
                let code = await oldRead();
                // hotUpdate preprocess
                if (code.includes('#endif')) {
                    code = (0, preprocess_1.preJs)((0, preprocess_1.preHtml)(code, ctx.file), ctx.file);
                }
                if (scoped) {
                    code = addScoped(code);
                }
                // 处理 block, wxs 等
                return (0, parse_1.parseVueCode)(code).code;
            };
        },
    };
}
exports.uniCssScopedPlugin = uniCssScopedPlugin;

}, function(modId) { var map = {"../../constants":1781362618871,"../../vue/parse":1781362618949}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618949, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.onVueTemplateCompileLog = exports.parseWxsCode = exports.parseWxsNodes = exports.parseBlockCode = exports.parseVueCode = void 0;
const compiler_core_1 = require("@vue/compiler-core");
const magic_string_1 = __importDefault(require("magic-string"));
const ast_1 = require("../vite/utils/ast");
const logs_1 = require("../logs");
const BLOCK_RE = /<\/block>/;
const WXS_LANG_RE = /lang=["|'](renderjs|wxs|sjs)["|']/;
const WXS_ATTRS = ['wxs', 'renderjs', 'sjs'];
function parseVueCode(code, isNVue = false) {
    const hasBlock = BLOCK_RE.test(code);
    const hasWxs = WXS_LANG_RE.test(code);
    if (!hasBlock && !hasWxs) {
        return { code };
    }
    const errors = [];
    const files = [];
    let ast = (0, ast_1.parseVue)(code, errors);
    if (hasBlock) {
        code = parseBlockCode(ast, code);
        // 重新解析新的 code
        ast = (0, ast_1.parseVue)(code, errors);
    }
    if (!isNVue && hasWxs) {
        const wxsNodes = parseWxsNodes(ast);
        code = parseWxsCode(wxsNodes, code);
        // add watch
        for (const wxsNode of wxsNodes) {
            const srcProp = wxsNode.props.find((prop) => prop.type === compiler_core_1.NodeTypes.ATTRIBUTE && prop.name === 'src');
            if (srcProp && srcProp.value) {
                files.push(srcProp.value.content);
            }
        }
    }
    return { code, files, errors };
}
exports.parseVueCode = parseVueCode;
function traverseChildren({ children }, blockNodes) {
    children.forEach((node) => traverseNode(node, blockNodes));
}
function traverseNode(node, blockNodes) {
    if ((0, ast_1.isElementNode)(node) && node.tag === 'block') {
        blockNodes.push(node);
    }
    if (node.type === compiler_core_1.NodeTypes.IF_BRANCH ||
        node.type === compiler_core_1.NodeTypes.FOR ||
        node.type === compiler_core_1.NodeTypes.ELEMENT ||
        node.type === compiler_core_1.NodeTypes.ROOT) {
        traverseChildren(node, blockNodes);
    }
}
function parseBlockCode(ast, code) {
    const blockNodes = [];
    traverseNode(ast, blockNodes);
    if (blockNodes.length) {
        return parseBlockNode(code, blockNodes);
    }
    return code;
}
exports.parseBlockCode = parseBlockCode;
const BLOCK_END_LEN = '</block>'.length;
const BLOCK_START_LEN = '<block'.length;
function parseBlockNode(code, blocks) {
    const magicString = new magic_string_1.default(code);
    blocks.forEach(({ loc }) => {
        const startOffset = loc.start.offset;
        const endOffset = loc.end.offset;
        magicString.overwrite(startOffset, startOffset + BLOCK_START_LEN, '<template');
        magicString.overwrite(endOffset - BLOCK_END_LEN, endOffset, '</template>');
    });
    return magicString.toString();
}
function parseWxsNodes(ast) {
    return ast.children.filter((node) => node.type === compiler_core_1.NodeTypes.ELEMENT &&
        node.tag === 'script' &&
        node.props.find((prop) => prop.name === 'lang' &&
            prop.type === compiler_core_1.NodeTypes.ATTRIBUTE &&
            prop.value &&
            WXS_ATTRS.includes(prop.value.content)));
}
exports.parseWxsNodes = parseWxsNodes;
function parseWxsCode(wxsNodes, code) {
    if (wxsNodes.length) {
        code = parseWxsNode(code, wxsNodes);
    }
    return code;
}
exports.parseWxsCode = parseWxsCode;
const SCRIPT_END_LEN = '</script>'.length;
const SCRIPT_START_LEN = '<script'.length;
function parseWxsNode(code, nodes) {
    const magicString = new magic_string_1.default(code);
    nodes.forEach(({ loc, props }) => {
        const langAttr = props.find((prop) => prop.name === 'lang');
        const moduleAttr = props.find((prop) => prop.name === 'module');
        const startOffset = loc.start.offset;
        const endOffset = loc.end.offset;
        const lang = langAttr.value.content;
        const langStartOffset = langAttr.loc.start.offset;
        magicString.overwrite(startOffset, startOffset + SCRIPT_START_LEN, '<' + lang); // <renderjs or <wxs
        magicString.overwrite(langStartOffset, langStartOffset + ('lang="' + lang + '"').length, ''); // remove lang="renderjs" or lang="wxs"
        magicString.overwrite(endOffset - SCRIPT_END_LEN, endOffset, '</' + lang + '>'); //</renderjs> or </wxs>
        if (moduleAttr) {
            const moduleStartOffset = moduleAttr.loc.start.offset;
            magicString.overwrite(moduleStartOffset, moduleStartOffset + 'module'.length, 'name'); // module="echarts" => name="echarts"
        }
    });
    return magicString.toString();
}
exports.onVueTemplateCompileLog = logs_1.onCompileLog;

}, function(modId) { var map = {"../vite/utils/ast":1781362618875}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618953, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.uniViteCopyPlugin = void 0;
const watcher_1 = require("../../watcher");
const messages_1 = require("../../messages");
const logs_1 = require("../../logs");
const uni_shared_1 = require("@dcloudio/uni-shared");
function uniViteCopyPlugin({ targets, }) {
    let resolvedConfig;
    let initialized = false;
    let isFirstBuild = true;
    return {
        name: 'uni:copy',
        apply: 'build',
        configResolved(config) {
            resolvedConfig = config;
        },
        async writeBundle() {
            if (initialized) {
                return;
            }
            if (resolvedConfig.build.ssr) {
                return;
            }
            initialized = true;
            const is_prod = process.env.NODE_ENV !== 'development' ||
                process.env.UNI_AUTOMATOR_CONFIG;
            const onChange = is_prod
                ? undefined
                : (0, uni_shared_1.debounce)(() => {
                    if (isFirstBuild) {
                        return;
                    }
                    (0, logs_1.resetOutput)('log');
                    (0, logs_1.output)('log', messages_1.M['dev.watching.end']);
                }, 100, { setTimeout, clearTimeout });
            return new Promise((resolve) => {
                Promise.all(targets.map(({ watchOptions, ...target }) => {
                    return new Promise((resolve) => {
                        new watcher_1.FileWatcher({
                            ...target,
                        }).watch({
                            cwd: process.env.UNI_INPUT_DIR,
                            persistent: is_prod ? false : true,
                            ...watchOptions,
                        }, () => {
                            resolve(void 0);
                        }, onChange);
                    });
                })).then(() => resolve());
            });
        },
        closeBundle() {
            isFirstBuild = false;
        },
    };
}
exports.uniViteCopyPlugin = uniViteCopyPlugin;

}, function(modId) { var map = {"../../watcher":1781362618954,"../../messages":1781362618844}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618954, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.FileWatcher = void 0;
const fs_extra_1 = __importDefault(require("fs-extra"));
const path_1 = __importDefault(require("path"));
const debug_1 = __importDefault(require("debug"));
const chokidar_1 = require("chokidar");
const shared_1 = require("@vue/shared");
const utils_1 = require("./utils");
const debugWatcher = (0, debug_1.default)('uni:watcher');
class FileWatcher {
    constructor({ src, dest, transform }) {
        this.src = !(0, shared_1.isArray)(src) ? [src] : src;
        this.dest = dest;
        this.transform = transform;
    }
    watch(watchOptions, onReady, onChange) {
        if (!this.watcher) {
            const copy = this.copy.bind(this);
            const remove = this.remove.bind(this);
            // escape chokidar cwd
            const src = this.src.map((src) => (0, utils_1.pathToGlob)(path_1.default.resolve(watchOptions.cwd), src));
            let closeTimer;
            const checkReady = () => {
                if (closeTimer) {
                    clearTimeout(closeTimer);
                }
                closeTimer = setTimeout(() => {
                    onReady && onReady(this.watcher);
                    // 等首次change完，触发完ready，在切换到真实的onChange
                    this.onChange = onChange;
                }, watchOptions.readyTimeout || 1000); // 300ms 在部分机器上仍有问题，调整成1000ms
            };
            this.onChange = checkReady;
            this.watcher = (0, chokidar_1.watch)(src, watchOptions)
                .on('add', copy)
                // .on('addDir', copy)
                .on('change', copy)
                .on('unlink', remove)
                // .on('unlinkDir', remove)
                .on('ready', () => {
                checkReady();
            })
                .on('error', (e) => console.error('watch', e));
        }
        return this.watcher;
    }
    add(paths) {
        this.info('add', paths);
        return this.watcher.add(paths);
    }
    unwatch(paths) {
        this.info('unwatch', paths);
        return this.watcher.unwatch(paths);
    }
    close() {
        this.info('close');
        return this.watcher.close();
    }
    copy(from) {
        const to = this.to(from);
        this.info('copy', from + '=>' + to);
        let content = '';
        if (this.transform) {
            const filename = this.from(from);
            content = this.transform(fs_extra_1.default.readFileSync(filename), filename);
        }
        if (content) {
            try {
                fs_extra_1.default.outputFileSync(to, content);
            }
            catch (e) {
                // noop
            }
            this.onChange && this.onChange();
            return;
        }
        try {
            fs_extra_1.default.copySync(this.from(from), to, { overwrite: true });
        }
        catch (e) {
            // noop
        }
        this.onChange && this.onChange();
    }
    remove(from) {
        const to = this.to(from);
        this.info('remove', from + '=>' + to);
        try {
            fs_extra_1.default.removeSync(to);
        }
        catch (e) {
            // noop
        }
        this.onChange && this.onChange();
    }
    info(type, msg) {
        debugWatcher.enabled && debugWatcher(type, msg);
    }
    from(from) {
        return path_1.default.join(this.watcher.options.cwd, from);
    }
    to(from) {
        return path_1.default.join(this.dest, from);
    }
}
exports.FileWatcher = FileWatcher;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618958, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.defineUniMainJsPlugin = void 0;
const path_1 = __importDefault(require("path"));
const utils_1 = require("../../utils");
function defineUniMainJsPlugin(createUniMainJsPlugin) {
    const opts = {
        resolvedConfig: {},
        filter(id) {
            return id === mainJsPath || id === mainTsPath || id === mainUTsPath;
        },
    };
    const plugin = createUniMainJsPlugin(opts);
    const origConfigResolved = plugin.configResolved;
    let mainJsPath = '';
    let mainTsPath = '';
    let mainUTsPath = '';
    plugin.configResolved = function (config) {
        opts.resolvedConfig = config;
        const mainPath = (0, utils_1.normalizePath)(path_1.default.resolve(process.env.UNI_INPUT_DIR, 'main'));
        mainJsPath = mainPath + '.js';
        mainTsPath = mainPath + '.ts';
        mainUTsPath = mainPath + '.uts';
        return origConfigResolved && origConfigResolved(config);
    };
    return plugin;
}
exports.defineUniMainJsPlugin = defineUniMainJsPlugin;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618960, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.defineUniManifestJsonPlugin = exports.defineUniPagesJsonPlugin = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const constants_1 = require("../../constants");
const utils_1 = require("../../utils");
exports.defineUniPagesJsonPlugin = createDefineJsonJsPlugin('pages.json');
exports.defineUniManifestJsonPlugin = createDefineJsonJsPlugin('manifest.json');
function createDefineJsonJsPlugin(name) {
    const JSON_JS = constants_1.JSON_JS_MAP[name];
    return function (createVitePlugin) {
        const opts = {
            resolvedConfig: {},
            filter(id) {
                return id.endsWith(JSON_JS);
            },
        };
        const plugin = createVitePlugin(opts);
        const origLoad = plugin.load;
        const origResolveId = plugin.resolveId;
        const origConfigResolved = plugin.configResolved;
        let jsonPath = '';
        let jsonJsPath = '';
        plugin.resolveId = function (id, importer, options) {
            const res = origResolveId && origResolveId.call(this, id, importer, options);
            if (res) {
                return res;
            }
            if (id.endsWith(JSON_JS)) {
                return jsonJsPath;
            }
        };
        plugin.configResolved = function (config) {
            opts.resolvedConfig = config;
            jsonPath = (0, utils_1.normalizePath)(path_1.default.join(process.env.UNI_INPUT_DIR, name));
            jsonJsPath = (0, utils_1.normalizePath)(path_1.default.join(process.env.UNI_INPUT_DIR, JSON_JS));
            return origConfigResolved && origConfigResolved(config);
        };
        plugin.load = function (id, ssr) {
            const res = origLoad && origLoad.call(this, id, ssr);
            if (res) {
                return res;
            }
            if (!opts.filter(id)) {
                return;
            }
            return fs_1.default.readFileSync(jsonPath, 'utf8');
        };
        return plugin;
    };
}

}, function(modId) { var map = {"../../constants":1781362618871}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618963, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.dynamicImportPolyfill = void 0;
function dynamicImportPolyfill(promise = false) {
    return {
        name: 'dynamic-import-polyfill',
        renderDynamicImport() {
            return {
                left: promise ? 'Promise.resolve(' : '(',
                right: ')',
            };
        },
    };
}
exports.dynamicImportPolyfill = dynamicImportPolyfill;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618965, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.uniUVueTypeScriptPlugin = exports.uniUTSUVueJavaScriptPlugin = void 0;
const vue_1 = require("../../../vue");
function uniUTSUVueJavaScriptPlugin(options = {}) {
    process.env.UNI_UTS_USING_ROLLUP = 'true';
    return {
        name: 'uni:uts-uvue',
        enforce: 'pre',
        configResolved(config) {
            // 移除自带的 esbuild 处理 ts 文件
            const index = config.plugins.findIndex((p) => p.name === 'vite:esbuild');
            if (index > -1) {
                // @ts-expect-error
                config.plugins.splice(index, 1);
            }
        },
        transform(code, id) {
            if (!(0, vue_1.isVueSfcFile)(id)) {
                return;
            }
            const platform = process.env.UNI_PLATFORM;
            const isApp = platform === 'app' ||
                platform === 'app-plus' ||
                platform === 'app-harmony';
            return {
                code: code.replace(/<script([^>]*)>/gi, (match, attributes) => {
                    let vapor = false;
                    if (process.env.UNI_APP_X_DOM2 === 'true') {
                        if (attributes.includes('setup') && !attributes.includes('vapor')) {
                            vapor = true;
                        }
                    }
                    let result = '';
                    // 如果 <script> 标签中没有 lang 属性，添加 lang="uts"
                    if (!/lang=["']?[^"']*["']?/.test(attributes)) {
                        result = `<script${attributes} lang="uts">`;
                    }
                    else {
                        // 否则，将现有的 lang 属性替换为 lang="uts"
                        result = match.replace(/lang=["']?ts["']?/, 'lang="uts"');
                    }
                    if (vapor) {
                        // 追加 vapor 属性
                        result = result.replace('lang=', 'vapor lang=');
                    }
                    return result;
                }),
                // app平台不可返回null，否则会报错“Multiple conflicting contents for sourcemap source”
                map: isApp ? { mappings: '' } : null,
            };
        },
    };
}
exports.uniUTSUVueJavaScriptPlugin = uniUTSUVueJavaScriptPlugin;
/**
 * 将 <script> 标签中的 lang="uts" 替换为 lang="ts"
 * 主要是当前功能内部使用 x.vite.config.ts 配置
 * @param options
 * @returns
 */
function uniUVueTypeScriptPlugin(options = {}) {
    return {
        name: 'uni:uvue-ts',
        enforce: 'pre',
        transform(code, id) {
            if (!(0, vue_1.isVueSfcFile)(id)) {
                return;
            }
            return {
                code: code.replace(/<script([^>]*)>/gi, (match, attributes) => {
                    // 如果 <script> 标签中没有 lang 属性，添加 lang="uts"
                    if (!/lang=["']?[^"']*["']?/.test(attributes)) {
                        return `<script${attributes} lang="ts">`;
                    }
                    // 否则，将现有的 lang 属性替换为 lang="uts"
                    return match.replace(/lang=["']?uts["']?/, 'lang="ts"');
                }),
                map: { mappings: '' },
            };
        },
    };
}
exports.uniUVueTypeScriptPlugin = uniUVueTypeScriptPlugin;

}, function(modId) { var map = {"../../../vue":1781362618917}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618966, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.injectsToAutoImports = exports.uniUTSExtApiReplace = exports.parseUniExtApisOnce = void 0;
const shared_1 = require("@vue/shared");
const uni_shared_1 = require("@dcloudio/uni-shared");
const vite_1 = __importDefault(require("unplugin-auto-import/vite"));
const uni_modules_1 = require("../../../uni_modules");
const url_1 = require("../../utils/url");
const escape = (str) => str.replace(/[-[\]/{}()*+?.\\^$|]/g, '\\$&');
exports.parseUniExtApisOnce = (0, uni_shared_1.once)(uni_modules_1.parseUniExtApis);
function uniUTSExtApiReplace() {
    const injects = (0, exports.parseUniExtApisOnce)(true, process.env.UNI_UTS_PLATFORM, process.env.UNI_UTS_TARGET_LANGUAGE);
    const injectApis = Object.keys(injects);
    const firstPass = new RegExp(`(?:${injectApis.map(escape).join('|')})`, 'g');
    return {
        name: 'uni:uts-ext-api-replace',
        configResolved(config) {
            const index = config.plugins.findIndex((p) => p.name === 'uts');
            if (index > -1) {
                if (Object.keys(injects).length) {
                    // @ts-expect-error
                    config.plugins.splice(index, 0, (0, vite_1.default)({
                        include: [/\.[u]?ts$/, /\.[u]?vue/],
                        exclude: [/[\\/]\.git[\\/]/],
                        imports: injectsToAutoImports(injects),
                        dts: false,
                    }));
                }
            }
        },
        transform(code, id) {
            if (!injectApis.length) {
                return;
            }
            if (!(0, url_1.isJsFile)(id)) {
                return;
            }
            if (code.search(firstPass) === -1) {
                return;
            }
            injectApis.forEach((api) => {
                code = code.replaceAll(api, api.replace('.', '_'));
            });
            return {
                code,
                map: null,
            };
        },
    };
}
exports.uniUTSExtApiReplace = uniUTSExtApiReplace;
/**
 * { 'uni.getBatteryInfo': ['@/uni_modules/uni-getbatteryinfo/utssdk/web/index.uts','getBatteryInfo'] }
 * { '@/uni_modules/uni-getbatteryinfo/utssdk/web/index.ts': [['getBatteryInfo', 'uni_getBatteryInfo']] }
 * @param injects
 */
function injectsToAutoImports(injects) {
    const autoImports = {};
    Object.keys(injects).forEach((api) => {
        const options = injects[api];
        if ((0, shared_1.isArray)(options) && options.length >= 2) {
            const source = options[0];
            const name = options[1];
            if (!autoImports[source]) {
                autoImports[source] = [];
            }
            autoImports[source].push([name, api.replace('.', '_')]);
        }
    });
    return Object.keys(autoImports).map((source) => {
        return {
            from: source,
            imports: autoImports[source],
        };
    });
}
exports.injectsToAutoImports = injectsToAutoImports;

}, function(modId) { var map = {"../../utils/url":1781362618876}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618968, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniEasycomPlugin = void 0;
const path_1 = __importDefault(require("path"));
const shared_1 = require("@vue/shared");
const pluginutils_1 = require("@rollup/pluginutils");
const url_1 = require("../utils/url");
const constants_1 = require("../../constants");
const easycom_1 = require("../../easycom");
const uts_1 = require("../../uts");
const uni_shared_1 = require("@dcloudio/uni-shared");
function uniEasycomPlugin(options) {
    const filter = (0, pluginutils_1.createFilter)(options.include, options.exclude);
    return {
        name: 'uni:app-easycom',
        transform(code, id) {
            if (!filter(id)) {
                return;
            }
            const { filename } = (0, url_1.parseVueRequest)(id);
            if (!constants_1.EXTNAME_VUE_TEMPLATE.includes(path_1.default.extname(filename))) {
                return;
            }
            if (!code.includes('_resolveComponent')) {
                return;
            }
            let i = 0;
            const importDeclarations = [];
            code = code.replace(/_resolveComponent\("(.+?)"(, true)?\)/g, (str, name) => {
                if (name && !name.startsWith('_')) {
                    const source = (0, easycom_1.matchEasycom)(name);
                    if (source) {
                        // 处理easycom组件优先级
                        return (0, easycom_1.genResolveEasycomCode)(importDeclarations, str, (0, easycom_1.addImportDeclaration)(importDeclarations, `__easycom_${i++}`, source, source.includes('uts-proxy')
                            ? (0, shared_1.capitalize)((0, shared_1.camelize)(name)) + 'Component'
                            : source.includes('uni_helpers')
                                ? (0, shared_1.capitalize)((0, shared_1.camelize)(name))
                                : ''));
                    }
                    else {
                        const utsCustomElement = (0, uts_1.getUTSCustomElement)(name);
                        if (utsCustomElement) {
                            (0, easycom_1.addImportDeclaration)(importDeclarations, '', utsCustomElement.source, '');
                            return `'${name}'`;
                        }
                    }
                    if (process.env.UNI_APP_X === 'true') {
                        if ((0, uni_shared_1.isAppUVueBuiltInEasyComponent)(name)) {
                            // 内置easycom组件不传入self参数
                            return str.replace(', true)', ')');
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

}, function(modId) { var map = {"../utils/url":1781362618876,"../../constants":1781362618871,"../../easycom":1781362618930,"../../uts":1781362618901}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618969, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniJsonPlugin = void 0;
const path_1 = __importDefault(require("path"));
const json_1 = require("../../json");
const preprocess_1 = require("../../preprocess");
const utils_1 = require("../utils");
function uniJsonPlugin() {
    return {
        name: 'uni:json',
        enforce: 'pre',
        transform(code, id) {
            // 如果已经被其他插件处理过了，就不再处理，比如 commonjs 插件，ICAPRegistrar.json?commonjs-external
            if (id.startsWith('\0')) {
                return;
            }
            if (path_1.default.extname((0, utils_1.parseVueRequest)(id).filename) !== '.json') {
                return;
            }
            return {
                code: JSON.stringify((0, json_1.parseJson)((0, preprocess_1.preJson)(code, id), false, id), null, 2),
                map: {
                    mappings: '',
                },
            };
        },
    };
}
exports.uniJsonPlugin = uniJsonPlugin;

}, function(modId) { var map = {"../../json":1781362618806,"../utils":1781362618873}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618971, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniPrePlugin = void 0;
const path_1 = __importDefault(require("path"));
const debug_1 = __importDefault(require("debug"));
const pluginutils_1 = require("@rollup/pluginutils");
const constants_1 = require("../../constants");
const preprocess_1 = require("../../preprocess");
const utils_1 = require("../utils");
const debugPreJs = (0, debug_1.default)('uni:pre-js');
const debugPreHtml = (0, debug_1.default)('uni:pre-html');
// const debugPreJsTry = debug('uni:pre-js-try')
function uniPrePlugin(config, options) {
    const isX = process.env.UNI_APP_X === 'true';
    const PRE_JS_EXTNAME = ['.json', '.css']
        .concat(isX ? constants_1.X_EXTNAME_VUE : constants_1.EXTNAME_VUE)
        .concat(constants_1.EXTNAME_JS); // 因为 1.0 也会使用 uts uni_modules，所以 EXTNAME_JS 直接包含了 .uts 后缀
    const PRE_HTML_EXTNAME = isX ? constants_1.X_EXTNAME_VUE : constants_1.EXTNAME_VUE;
    const filter = (0, pluginutils_1.createFilter)(options.include, options.exclude);
    const isNVue = config.nvue;
    const preJsFile = isNVue ? preprocess_1.preNVueJs : preprocess_1.preJs;
    const preHtmlFile = isNVue ? preprocess_1.preNVueHtml : preprocess_1.preHtml;
    return {
        name: 'uni:pre',
        transform(code, id) {
            if (!filter(id)) {
                return;
            }
            const { filename, query } = (0, utils_1.parseVueRequest)(id);
            const extname = path_1.default.extname(filename);
            const isHtml = query.type === 'template' || PRE_HTML_EXTNAME.includes(extname);
            const isJs = PRE_JS_EXTNAME.includes(extname);
            const isPre = isHtml || isJs;
            if (isPre) {
                // debugPreJsTry(id)
            }
            const hasEndif = isPre && code.includes('#endif');
            if (!hasEndif) {
                return;
            }
            // 因为完整的 vue 会先条件编译，而 vue&type=template 等会再次条件编译，如果走error模式会报两次错误，且第二次错误没有正确的位置映射
            const unbalanced = query.vue ? 'skip' : 'error';
            if (isHtml) {
                code = preHtmlFile(code, id, { unbalanced });
                debugPreHtml(id);
            }
            if (isJs) {
                code = preJsFile(code, id, { unbalanced });
                debugPreJs(id);
            }
            return {
                code,
                map: (0, utils_1.withSourcemap)(config) ? this.getCombinedSourcemap() : null,
            };
        },
    };
}
exports.uniPrePlugin = uniPrePlugin;

}, function(modId) { var map = {"../../constants":1781362618871,"../utils":1781362618873}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618975, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.uniStatsPlugin = void 0;
const fs_extra_1 = __importDefault(require("fs-extra"));
const utils_1 = require("../../utils");
const json_1 = require("../../json");
const messages_1 = require("../../messages");
const emittedHashMap = new WeakMap();
function uniStatsPlugin() {
    let resolvedConfig;
    let isManifestChanged = false;
    const shouldTrackManifestChange = process.env.UNI_APP_X === 'true' &&
        (process.env.UNI_PLATFORM === 'app' ||
            process.env.UNI_PLATFORM === 'app-harmony');
    let isVapor = shouldTrackManifestChange && process.env.UNI_APP_X_DOM2 === 'true';
    return {
        name: 'uni:app-stats',
        enforce: 'post',
        configResolved(config) {
            resolvedConfig = config;
            emittedHashMap.set(resolvedConfig, new Map());
        },
        watchChange(id) {
            if (shouldTrackManifestChange && id.endsWith('manifest.json')) {
                isManifestChanged = true;
                try {
                    const manifest = (0, json_1.parseJson)(fs_extra_1.default.readFileSync(id, 'utf-8'), true, 'manifest.json');
                    const uniAppX = manifest['uni-app-x'] || {};
                    if (uniAppX.vapor !== isVapor) {
                        isVapor = uniAppX.vapor === true;
                        console.warn(messages_1.M['dev.watching.restart.vapor']);
                    }
                }
                catch (e) { }
            }
        },
        writeBundle(_, bundle) {
            if (resolvedConfig.isProduction) {
                // 仅dev生效
                return;
            }
            const emittedHash = emittedHashMap.get(resolvedConfig);
            const changedFiles = [];
            Object.keys(bundle).forEach((filename) => {
                // 不处理sourcemap
                if (filename.endsWith('.map')) {
                    return;
                }
                const outputFile = bundle[filename];
                let outputFileHash = '';
                if (outputFile.type === 'asset') {
                    outputFileHash = (0, utils_1.hash)(outputFile.source);
                }
                else {
                    outputFileHash = (0, utils_1.hash)(outputFile.code);
                }
                if (emittedHash.get(filename) !== outputFileHash) {
                    emittedHash.set(filename, outputFileHash);
                    changedFiles.push(filename);
                }
            });
            if (isManifestChanged) {
                isManifestChanged = false;
                changedFiles.unshift('manifest.json');
            }
            process.env.UNI_APP_CHANGED_FILES = changedFiles.length
                ? JSON.stringify(changedFiles)
                : '';
        },
    };
}
exports.uniStatsPlugin = uniStatsPlugin;

}, function(modId) { var map = {"../../json":1781362618806,"../../messages":1781362618844}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618980, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.initFeatures = void 0;
const fs_1 = __importDefault(require("fs"));
const path_1 = __importDefault(require("path"));
const shared_1 = require("@vue/shared");
function initProjectFeature({ inputDir }) {
    const features = {
        i18nLocale: false,
        i18nEn: true,
        i18nEs: true,
        i18nFr: true,
        i18nZhHans: true,
        i18nZhHant: true,
        uniCloud: false,
    };
    const localesDir = path_1.default.resolve(inputDir, 'locale');
    if (fs_1.default.existsSync(localesDir)) {
        if (fs_1.default.readdirSync(localesDir).find((file) => path_1.default.extname(file) === '.json')) {
            features.i18nLocale = true;
        }
    }
    if (process.env.UNI_CLOUD_PROVIDER) {
        try {
            const providers = JSON.parse(process.env.UNI_CLOUD_PROVIDER);
            if (providers.length) {
                features.uniCloud = true;
            }
        }
        catch (e) { }
    }
    return features;
}
function initPagesFeature({ pagesJson, command, inputDir, ssr, }) {
    const features = {
        nvue: true,
        pages: true,
        tabBar: true,
        tabBarMidButton: true,
        topWindow: false,
        leftWindow: false,
        rightWindow: false,
        navigationBar: true,
        pullDownRefresh: false,
        navigationBarButtons: true,
        navigationBarSearchInput: true,
        navigationBarTransparent: true,
    };
    const { tabBar, pages, topWindow, leftWindow, rightWindow, globalStyle } = pagesJson;
    // ssr 时强制启用多页面（需要用到router）
    if (!ssr && pages && pages.length === 1) {
        features.pages = false;
    }
    if (!(tabBar && tabBar.list && tabBar.list.length)) {
        features.tabBar = false;
        features.tabBarMidButton = false;
    }
    if (features.tabBar && !tabBar.midButton) {
        features.tabBarMidButton = false;
    }
    if (topWindow && topWindow.path) {
        features.topWindow = true;
    }
    if (leftWindow && leftWindow.path) {
        features.leftWindow = true;
    }
    if (rightWindow && rightWindow.path) {
        features.rightWindow = true;
    }
    if (globalStyle.enablePullDownRefresh) {
        features.pullDownRefresh = true;
    }
    else {
        if (pages.find((page) => page.style && page.style.enablePullDownRefresh)) {
            features.pullDownRefresh = true;
        }
    }
    if (command === 'build') {
        if (!pages.find((page) => fs_1.default.existsSync(path_1.default.resolve(inputDir, page.path + '.nvue')))) {
            features.nvue = false;
        }
        let isNavigationCustom = false;
        if (globalStyle.navigationBar.style === 'custom') {
            isNavigationCustom = true; // 全局custom
            if (pages.find((page) => page.style.navigationBar.style === 'default')) {
                isNavigationCustom = false;
            }
        }
        else {
            // 所有页面均custom
            if (pages.every((page) => page.style.navigationBar.style === 'custom')) {
                isNavigationCustom = true;
            }
        }
        if (isNavigationCustom) {
            features.navigationBar = false;
            features.navigationBarButtons = false;
            features.navigationBarSearchInput = false;
            features.navigationBarTransparent = false;
        }
        else {
            if (!pages.find((page) => (0, shared_1.isArray)(page.style.navigationBar.buttons) &&
                page.style.navigationBar.buttons.length)) {
                features.navigationBarButtons = false;
            }
            if (!globalStyle.navigationBar.searchInput &&
                !pages.find((page) => page.style.navigationBar.searchInput)) {
                features.navigationBarSearchInput = false;
            }
            if (globalStyle.navigationBar.type !== 'transparent' &&
                !pages.find((page) => page.style.navigationBar.type === 'transparent')) {
                features.navigationBarTransparent = false;
            }
        }
    }
    return features;
}
function initManifestFeature({ manifestJson, command, platform, }) {
    const features = {
        wx: false,
        wxs: true,
        rpx: true,
        promise: false,
        longpress: true,
        routerMode: '"hash"',
        vueOptionsApi: true,
        vueProdDevTools: false,
        vueProdHydrationMismatchDetails: false,
    };
    if (command === 'build') {
        // TODO 需要预编译一遍？
        // features.wxs = false
        // features.longpress = false
    }
    const webManifest = manifestJson.web || manifestJson.h5;
    if (webManifest &&
        webManifest.router &&
        webManifest.router.mode === 'history') {
        features.routerMode = '"history"';
    }
    // TODO other features
    return features;
}
function initFeatures(options) {
    const { wx, wxs, rpx, nvue, uniCloud, i18nEn, i18nEs, i18nFr, i18nZhHans, i18nZhHant, i18nLocale, vueOptionsApi, vueProdDevTools, vueProdHydrationMismatchDetails, pages, tabBar, tabBarMidButton, promise, longpress, routerMode, topWindow, leftWindow, rightWindow, navigationBar, pullDownRefresh, navigationBarButtons, navigationBarSearchInput, navigationBarTransparent, } = (0, shared_1.extend)(initManifestFeature(options), initPagesFeature(options), initProjectFeature(options));
    // uni-app-x运行时可调用$setPageStyle，需启用相关特性方可支持。否则$setPageStyle无效果
    const features = {
        // vue
        __VUE_OPTIONS_API__: vueOptionsApi, // enable/disable Options API support, default: true
        __VUE_PROD_DEVTOOLS__: vueProdDevTools, // enable/disable devtools support in production, default: false
        __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: vueProdHydrationMismatchDetails,
        // uni
        __UNI_FEATURE_WX__: wx, // 是否启用小程序的组件实例 API，如：selectComponent 等（uni-core/src/service/plugin/appConfig）
        __UNI_FEATURE_WXS__: wxs, // 是否启用 wxs 支持，如：getComponentDescriptor 等（uni-core/src/view/plugin/appConfig）
        __UNI_FEATURE_RPX__: rpx, // 是否启用运行时 rpx 支持
        __UNI_FEATURE_PROMISE__: promise, // 是否启用旧版本的 promise 支持（即返回[err,res]的格式）,默认返回标准
        __UNI_FEATURE_LONGPRESS__: longpress, // 是否启用longpress
        __UNI_FEATURE_I18N_EN__: i18nEn, // 是否启用en
        __UNI_FEATURE_I18N_ES__: i18nEs, // 是否启用es
        __UNI_FEATURE_I18N_FR__: i18nFr, // 是否启用fr
        __UNI_FEATURE_I18N_ZH_HANS__: i18nZhHans, // 是否启用zh_Hans
        __UNI_FEATURE_I18N_ZH_HANT__: i18nZhHant, // 是否启用zh_Hant
        // 以下特性，编译器已自动识别是否需要启用
        __UNI_FEATURE_UNI_CLOUD__: uniCloud, // 是否启用uniCloud
        __UNI_FEATURE_I18N_LOCALE__: i18nLocale, // 是否启用i18n
        __UNI_FEATURE_NVUE__: nvue, // 是否启用nvue
        __UNI_FEATURE_ROUTER_MODE__: routerMode, // 路由模式
        __UNI_FEATURE_PAGES__: pages, // 是否多页面
        __UNI_FEATURE_TABBAR__: tabBar, // 是否包含tabBar
        __UNI_FEATURE_TABBAR_MIDBUTTON__: tabBarMidButton, // 是否包含midButton
        __UNI_FEATURE_TOPWINDOW__: topWindow, // 是否包含topWindow
        __UNI_FEATURE_LEFTWINDOW__: leftWindow, // 是否包含leftWindow
        __UNI_FEATURE_RIGHTWINDOW__: rightWindow, // 是否包含rightWindow
        __UNI_FEATURE_RESPONSIVE__: topWindow || leftWindow || rightWindow, // 是否启用响应式
        __UNI_FEATURE_NAVIGATIONBAR__: navigationBar, // 是否启用标题栏
        __UNI_FEATURE_PULL_DOWN_REFRESH__: pullDownRefresh, // 是否启用下拉刷新
        __UNI_FEATURE_NAVIGATIONBAR_BUTTONS__: navigationBarButtons, // 是否启用标题栏按钮
        __UNI_FEATURE_NAVIGATIONBAR_SEARCHINPUT__: navigationBarSearchInput, // 是否启用标题栏搜索框
        __UNI_FEATURE_NAVIGATIONBAR_TRANSPARENT__: navigationBarTransparent, // 是否启用透明标题栏
    };
    // ssr nodejs features
    if (options.ssr) {
        Object.keys(features).forEach((name) => {
            const value = features[name];
            (0, shared_1.extend)(globalThis, {
                [name]: (0, shared_1.isString)(value) ? JSON.parse(value) : value,
            });
        });
    }
    return features;
}
exports.initFeatures = initFeatures;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618981, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.initAutoImportOptions = void 0;
const path_1 = __importDefault(require("path"));
const uni_shared_1 = require("@dcloudio/uni-shared");
const uts_1 = require("../uts");
const workers_1 = require("../workers");
const utils_1 = require("../utils");
const x_1 = require("../x");
const uniWebLifeCyclePreset = {
    from: '@dcloudio/uni-app',
    imports: [
        // ssr
        'ssrRef',
        'shallowSsrRef',
        // uni-app lifecycle
        // App and Page
        'onShow',
        'onHide',
        // App
        'onLaunch',
        // web平台如下生命周期和uni.xxx api冲突，发行时改为通过uni-h5导入api实现，运行时通过uni-app导入
        // 'onAppShow',
        // 'onAppHide',
        // 'onError',
        // 'onPageNotFound',
        // 'onUnhandledRejection',
        'onThemeChange',
        'onKeyboardHeightChange',
        'onLastPageBackPress',
        'onExit',
        // Page
        'onPageShow',
        'onPageHide',
        'onLoad',
        'onReady',
        'onUnload',
        'onResize',
        'onBackPress',
        'onPageScroll',
        'onTabItemTap',
        'onReachBottom',
        'onPullDownRefresh',
        // 其他
        'onShareTimeline',
        'onShareAppMessage',
        'onShareChat', // xhs-share
        'onCopyUrl',
        'onUploadDouyinVideo',
        'onLiveMount',
        'onTitleClick',
        // 辅助
        'renderComponentSlot',
    ],
};
const uniLifeCyclePreset = {
    from: '@dcloudio/uni-app',
    imports: [
        // ssr
        'ssrRef',
        'shallowSsrRef',
        // uni-app lifecycle
        // App and Page
        'onShow',
        'onHide',
        // App
        'onLaunch',
        'onAppShow',
        'onAppHide',
        'onError',
        'onThemeChange',
        'onKeyboardHeightChange',
        'onPageNotFound',
        'onUnhandledRejection',
        'onLastPageBackPress',
        'onExit',
        // Page
        'onPageShow',
        'onPageHide',
        'onLoad',
        'onReady',
        'onUnload',
        'onResize',
        'onBackPress',
        'onPageScroll',
        'onTabItemTap',
        'onReachBottom',
        'onPullDownRefresh',
        // 其他
        'onShareTimeline',
        'onShareAppMessage',
        'onShareChat', // xhs-share
        'onCopyUrl',
        'onUploadDouyinVideo',
        'onLiveMount',
        'onTitleClick',
        // 辅助
        'renderComponentSlot',
    ],
};
const uniH5Preset = {
    from: '@dcloudio/uni-h5',
    imports: [
        'UniElement',
        'UniElementImpl',
        'UniButtonElement',
        'UniCanvasElement',
        'UniCheckboxElement',
        'UniCheckboxGroupElement',
        'UniEditorElement',
        'UniFormElement',
        'UniIconElement',
        'UniImageElement',
        'UniInputElement',
        'UniLabelElement',
        'UniMovableAreaElement',
        'UniMovableViewElement',
        'UniNavigatorElement',
        'UniPickerViewElement',
        'UniPickerViewColumnElement',
        'UniProgressElement',
        'UniRadioElement',
        'UniRadioGroupElement',
        'UniRichTextElement',
        'UniScrollViewElement',
        'UniSliderElement',
        'UniSwiperElement',
        'UniSwiperItemElement',
        'UniSwitchElement',
        'UniTextElement',
        'UniTextareaElement',
        'UniViewElement',
        'UniViewElementImpl',
        'UniListViewElement',
        'UniListItemElement',
        'UniStickySectionElement',
        'UniStickyHeaderElement',
        'UniVideoElement',
        'UniWebViewElement',
        'UniMapElement',
        'UniCoverViewElement',
        'UniCoverImageElement',
        'UniPickerElement',
    ],
};
if (process.env.NODE_ENV === 'development') {
    uniWebLifeCyclePreset.imports.push(
    // web平台如下生命周期和uni.xxx api冲突，发行时改为通过uni-h5导入api实现，运行时通过uni-app导入
    'onAppShow', 'onAppHide', 'onError', 'onPageNotFound', 'onUnhandledRejection');
}
else {
    uniH5Preset.imports.push('onAppShow', 'onAppHide', 'onError', 'onPageNotFound', 'onUnhandledRejection');
}
const uniMiniProgramPreset = {
    from: 'vue',
    imports: ['UniElement', 'UniElementImpl'],
};
const cloudPreset = {
    from: '@dcloudio/uni-cloud',
    imports: ['uniCloud', 'UniCloudError'],
};
const utsJsPreset = {
    from: '@dcloudio/uni-shared',
    imports: ['UTS', 'UTSJSONObject', 'UTSValueIterable', 'UniError'],
};
const uniAppLifeCyclePreset = {
    from: 'vue',
    imports: [
        // ssr
        'ssrRef',
        'shallowSsrRef',
        // uni-app lifecycle
        // App and Page
        'onShow',
        'onHide',
        // App
        'onAppShow',
        'onAppHide',
        'onLaunch',
        'onError',
        'onThemeChange',
        'onLastPageBackPress',
        // onKeyboardHeightChange,
        'onPageNotFound',
        'onUnhandledRejection',
        'onExit',
        // Page
        'onPageShow',
        'onPageHide',
        'onLoad',
        'onReady',
        'onUnload',
        'onResize',
        'onBackPress',
        'onPageScroll',
        'onTabItemTap',
        'onReachBottom',
        'onPullDownRefresh',
        // 其他
        'onShareTimeline',
        'onShareAppMessage',
        // onShareChat, // xhs-share
        // 辅助，用于自定义render函数时，开发者可以调用此方法渲染组件的slot
        'renderComponentSlot',
    ],
};
const vuePreset = {
    from: 'vue',
    imports: [
        // vue lifecycle
        'onActivated',
        'onBeforeMount',
        'onBeforeUnmount',
        'onBeforeUpdate',
        'onErrorCaptured',
        'onDeactivated',
        'onMounted',
        'onServerPrefetch',
        'onUnmounted',
        'onUpdated',
        // uni-app specific lifecycle
        'onReuse',
        'onRecycle',
        // setup helpers
        'useAttrs',
        'useSlots',
        'useComputedStyle',
        'useRecycleState',
        // reactivity,
        'computed',
        'customRef',
        'isReadonly',
        'isRef',
        'isProxy',
        'isReactive',
        'markRaw',
        'reactive',
        'readonly',
        'ref',
        'shallowReactive',
        'shallowReadonly',
        'shallowRef',
        'triggerRef',
        'toRaw',
        'toRef',
        'toRefs',
        'toValue',
        'unref',
        'watch',
        'watchEffect',
        'watchPostEffect',
        'watchSyncEffect',
        // component
        'defineComponent',
        'defineAsyncComponent',
        'getCurrentInstance',
        'inject',
        'nextTick',
        'provide',
        'useCssModule',
        'createApp',
        'hasInjectionContext',
        // render
        'h',
        'mergeProps',
        'cloneVNode',
        'isVNode',
        'resolveComponent',
        'resolveDirective',
        'withDirectives',
        'withModifiers',
        // effect scope
        'effectScope',
        'EffectScope',
        'getCurrentScope',
        'onScopeDispose',
        // types 全部全局导入
        // vapor
        'getCurrentGenericInstance',
    ],
};
function initAutoImportOptions(platform, { imports = [], ...userOptions }) {
    rewriteAutoImportOnce();
    const autoImport = [vuePreset];
    // JS 引擎的 App 平台统一走 vue 生命周期导入
    if (platform === 'app-ios' ||
        platform === 'app-harmony' ||
        (0, x_1.isUniAppXAndroidJsEngine)(platform)) {
        autoImport.push(uniAppLifeCyclePreset);
    }
    else if (platform === 'web') {
        autoImport.push(uniWebLifeCyclePreset);
    }
    else {
        autoImport.push(uniLifeCyclePreset);
    }
    // 内置框架编译时，不能注入这些内容
    if (!process.env.UNI_COMPILE_EXT_API_TYPE) {
        autoImport.push(cloudPreset);
    }
    if (platform === 'web') {
        autoImport.push(uniH5Preset);
    }
    else if (platform.startsWith('mp-')) {
        autoImport.push(uniMiniProgramPreset);
        if (process.env.UNI_APP_X === 'true') {
            // 小程序端使用autoImport
            autoImport.push(utsJsPreset);
        }
    }
    const exclude = [/[\\/]\.git[\\/]/];
    if (process.env.UNI_INPUT_DIR) {
        exclude.push(...(0, workers_1.resolveWorkersDir)(process.env.UNI_INPUT_DIR).map((dir) => (0, utils_1.normalizePath)(path_1.default.join(process.env.UNI_INPUT_DIR, dir, '*'))));
    }
    return {
        ...userOptions,
        include: [/\.[u]?ts$/, /\.[u]?vue/],
        exclude,
        imports: imports.concat(
        // 旧版 Android x 仍由专有编译流程处理，Android Vapor 对齐 iOS 走通用自动导入
        (0, x_1.isUniAppXAndroidNative)(platform) ? [] : autoImport),
        dts: false,
    };
}
exports.initAutoImportOptions = initAutoImportOptions;
const rewriteAutoImportOnce = (0, uni_shared_1.once)(() => {
    const unimport = require('unimport');
    const originalCreateUnimport = unimport.createUnimport;
    unimport.createUnimport = (opts) => {
        const unimport_ = originalCreateUnimport(opts);
        const originalScanImportsFromDir = unimport_.scanImportsFromDir;
        unimport_.scanImportsFromDir = async (dir, options) => {
            const exports_ = (await originalScanImportsFromDir(dir, options));
            scanCustomElements(exports_);
            return exports_;
        };
        return unimport_;
    };
});
function scanCustomElements(exports_) {
    const utsCustomElementsExports = (0, uts_1.getUTSCustomElementsExports)();
    for (const [key, value] of utsCustomElementsExports.entries()) {
        value.exports.forEach((export_) => {
            exports_.push({
                from: key,
                name: export_[0],
            });
        });
    }
}

}, function(modId) { var map = {"../uts":1781362618901,"../workers":1781362618899,"../x":1781362618905}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618984, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.uniUniModulesExtApiPlugin = void 0;
const uts_1 = require("../uts");
const resolve_1 = require("../resolve");
const x_1 = require("../x");
function uniUniModulesExtApiPlugin() {
    return {
        name: 'uni:uni-modules_ext-api',
        apply: 'build',
        config() {
            const rollupOptions = {
                input: (0, resolve_1.resolveMainPathOnce)(process.env.UNI_INPUT_DIR),
                external: ['vue'],
                output: {
                    format: 'iife',
                    entryFileNames: `${process.env.UNI_COMPILE_EXT_API_OUT_FILE_NAME || 'components'}.js`,
                    globals: {
                        vue: 'Vue',
                        uni: 'uni',
                    },
                },
            };
            const build = {};
            if (process.env.UNI_UTS_PLATFORM === 'app-ios' ||
                process.env.UNI_UTS_PLATFORM === 'app-harmony' ||
                (0, x_1.isUniAppXAndroidJsEngine)()) {
                build.rollupOptions = rollupOptions;
            }
            return {
                build,
            };
        },
        load(id) {
            if ((0, uts_1.isUTSProxy)(id)) {
                return '';
            }
        },
        // generateBundle(_, bundle) {
        //   Object.keys(bundle).forEach((fileName) => {
        //     console.log('fileName', fileName)
        //   })
        // },
    };
}
exports.uniUniModulesExtApiPlugin = uniUniModulesExtApiPlugin;

}, function(modId) { var map = {"../uts":1781362618901,"../resolve":1781362618891,"../x":1781362618905}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618988, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.transformUniH5Jsx = void 0;
const uni_shared_1 = require("@dcloudio/uni-shared");
function transformUniH5Jsx({ types }) {
    return {
        name: 'babel-plugin-uni-h5-jsx',
        visitor: {
            JSXOpeningElement({ node: { name } }) {
                if (types.isJSXIdentifier(name) && (0, uni_shared_1.isBuiltInComponent)(name.name)) {
                    name.name = 'v-uni-' + name.name;
                }
            },
        },
    };
}
exports.transformUniH5Jsx = transformUniH5Jsx;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618989, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ANY_JS_STYLE_PLACEHOLDER_RE = exports.createJsStylePlaceholderRegExp = exports.createJsStylePlaceholder = exports.JS_STYLE_PLACEHOLDER_MARKER = void 0;
const hash_sum_1 = __importDefault(require("hash-sum"));
exports.JS_STYLE_PLACEHOLDER_MARKER = '__js_style_placeholder__';
function createJsStylePlaceholder(id) {
    const hashId = (0, hash_sum_1.default)(id);
    return JSON.stringify({
        [exports.JS_STYLE_PLACEHOLDER_MARKER]: hashId,
    });
}
exports.createJsStylePlaceholder = createJsStylePlaceholder;
function createJsStylePlaceholderRegExp(id) {
    const hashId = (0, hash_sum_1.default)(id);
    return new RegExp(`\\{\\s*"${exports.JS_STYLE_PLACEHOLDER_MARKER}"\\s*:\\s*"${hashId}"\\s*\\}`, 'g');
}
exports.createJsStylePlaceholderRegExp = createJsStylePlaceholderRegExp;
exports.ANY_JS_STYLE_PLACEHOLDER_RE = new RegExp(`\\{\\s*"${exports.JS_STYLE_PLACEHOLDER_MARKER}"\\s*:\\s*".+?"\\s*\\}`, 'g');

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618992, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.stripOptions = void 0;
exports.stripOptions = {
    include: ['**/*.js', '**/*.ts', '**/*.tsx', '**/*.vue'],
    functions: [
        'onBeforeMount',
        'onMounted',
        'onBeforeUpdate',
        'onUpdated',
        'onActivated',
        'onDeactivated',
        'onBeforeActivate',
        'onBeforeDeactivate',
        'onBeforeUnmount',
        'onUnmounted',
        'onRenderTracked',
        'onRenderTriggered',
    ],
};

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618994, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.COMPONENT_DEPS_CSS = exports.API_DEPS_CSS = void 0;
const constants_1 = require("./constants");
const RESIZE_SENSOR_CSS = constants_1.BASE_COMPONENTS_STYLE_PATH + 'resize-sensor.css';
const REFRESHER_CSS = constants_1.BASE_COMPONENTS_STYLE_PATH + 'refresher.css';
const API_DEPS_CSS = (isX) => {
    const deps_css = {
        showModal: [],
        showToast: [`${constants_1.H5_API_STYLE_PATH}toast.css`],
        showActionSheet: [],
        previewImage: [
            RESIZE_SENSOR_CSS,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}swiper.css`,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}swiper-item.css`,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}movable-area.css`,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}movable-view.css`,
        ],
        openLocation: [`${constants_1.H5_API_STYLE_PATH}location-view.css`],
        chooseLocation: [],
        showLoading: [],
    };
    if (isX) {
        deps_css.showModal = [`${constants_1.BASE_COMPONENTS_STYLE_PATH}textarea.css`];
        deps_css.chooseLocation = [
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}/view.css`,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}/text.css`,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}/input.css`,
            `${constants_1.H5_COMPONENTS_STYLE_PATH}/map.css`,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}/scroll-view.css`,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}/image.css`,
        ];
        deps_css.showLoading = [`${constants_1.X_BASE_COMPONENTS_STYLE_PATH}loading.css`];
    }
    else {
        deps_css.showModal = [`${constants_1.H5_API_STYLE_PATH}modal.css`];
        deps_css.showActionSheet = [`${constants_1.H5_API_STYLE_PATH}action-sheet.css`];
        deps_css.chooseLocation = [
            `${constants_1.H5_API_STYLE_PATH}/location-picker.css`,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}/input.css`,
            `${constants_1.H5_COMPONENTS_STYLE_PATH}/map.css`,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}/scroll-view.css`,
        ];
    }
    return deps_css;
};
exports.API_DEPS_CSS = API_DEPS_CSS;
const COMPONENT_DEPS_CSS = (isX) => {
    return {
        canvas: [RESIZE_SENSOR_CSS],
        image: [RESIZE_SENSOR_CSS],
        'movable-area': [RESIZE_SENSOR_CSS],
        'picker-view': [RESIZE_SENSOR_CSS],
        'picker-view-column': [RESIZE_SENSOR_CSS],
        'rich-text': [RESIZE_SENSOR_CSS],
        textarea: [RESIZE_SENSOR_CSS],
        'web-view': [RESIZE_SENSOR_CSS],
        picker: [
            RESIZE_SENSOR_CSS,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}picker-view.css`,
            `${constants_1.BASE_COMPONENTS_STYLE_PATH}picker-view-column.css`,
        ],
        'scroll-view': [REFRESHER_CSS],
        'list-view': [RESIZE_SENSOR_CSS, REFRESHER_CSS],
    };
};
exports.COMPONENT_DEPS_CSS = COMPONENT_DEPS_CSS;

}, function(modId) { var map = {"./constants":1781362618871}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618997, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.initPostcssPlugin = exports.uniPostcssExternalPlugin = exports.uniPostcssScopedPlugin = exports.uniPostcssPlugin = void 0;
const stylePluginScoped_1 = __importDefault(require("./plugins/stylePluginScoped"));
exports.uniPostcssScopedPlugin = stylePluginScoped_1.default;
const stylePluginExternal_1 = __importDefault(require("./plugins/stylePluginExternal"));
exports.uniPostcssExternalPlugin = stylePluginExternal_1.default;
const uniapp_1 = __importDefault(require("./plugins/uniapp"));
exports.uniPostcssPlugin = uniapp_1.default;
function initPostcssPlugin({ uniApp, autoprefixer, } = {}) {
    const plugins = [(0, uniapp_1.default)(uniApp)];
    // nvue 不需要 autoprefixer
    if (autoprefixer !== false) {
        plugins.push(require('autoprefixer')(autoprefixer));
    }
    return plugins;
}
exports.initPostcssPlugin = initPostcssPlugin;

}, function(modId) { var map = {"./plugins/stylePluginScoped":1781362618998}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618998, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const postcss_selector_parser_1 = __importDefault(require("postcss-selector-parser"));
const uni_shared_1 = require("@dcloudio/uni-shared");
const scopedPlugin = () => {
    return {
        postcssPlugin: 'uni-sfc-scoped',
        prepare({ processor: { plugins } }) {
            const hasVueSfcScoped = !!plugins.find((plugin) => plugin.postcssPlugin === 'vue-sfc-scoped');
            return {
                Rule(rule) {
                    processRule(rule, hasVueSfcScoped);
                },
            };
        },
    };
};
const processedRules = new WeakSet();
function processRule(rule, hasVueSfcScoped) {
    if (processedRules.has(rule) ||
        (rule.parent &&
            rule.parent.type === 'atrule' &&
            /-?keyframes$/.test(rule.parent.name))) {
        return;
    }
    processedRules.add(rule);
    const isXWeb = process.env.UNI_APP_X === 'true' && process.env.UNI_UTS_PLATFORM === 'web';
    rule.selector = (0, postcss_selector_parser_1.default)((selectorRoot) => {
        selectorRoot.each((selector) => {
            hasVueSfcScoped
                ? rewriteDeprecatedSelector(selector)
                : rewriteSelector(selector, selectorRoot);
            if (isXWeb) {
                rewritePartSelector(selector, selectorRoot);
            }
        });
    }).processSync(rule.selector);
}
function rewritePartSelector(selector, selectorRoot) {
    if (selector.some((n) => {
        return n.type === 'pseudo' && n.value === '::part';
    })) {
        const uniPartSelector = selector.clone();
        _rewritePartSelector(uniPartSelector, selectorRoot);
        selectorRoot.insertAfter(selector, uniPartSelector);
    }
}
function _rewritePartSelector(selector, selectorRoot) {
    selector.each((n) => {
        if (n.type === 'pseudo' && n.value === '::part') {
            const partClass = (0, uni_shared_1.getPartClass)(n.nodes[0].toString());
            selector.insertAfter(n, postcss_selector_parser_1.default.pseudo({
                value: ':deep',
                nodes: [
                    postcss_selector_parser_1.default.className({
                        value: partClass,
                    }),
                ],
            }));
            selector.removeChild(n);
        }
    });
}
/**
 * @param selector
 * @returns
 */
function rewriteDeprecatedSelector(selector) {
    const nodes = [];
    let deepNode;
    selector.each((n) => {
        if (deepNode) {
            nodes.push(n);
            selector.removeChild(n);
        }
        else {
            const { type, value } = n;
            if (type === 'pseudo' && value === '::v-deep') {
                deepNode = n;
            }
            else if (type === 'combinator' &&
                (value === '>>>' || value === '/deep/')) {
                deepNode = n;
            }
        }
    });
    if (!deepNode) {
        return;
    }
    if (deepNode.type === 'combinator') {
        const index = selector.index(deepNode);
        if (index > 0) {
            selector.insertBefore(deepNode, postcss_selector_parser_1.default.combinator({ value: ' ' }));
        }
    }
    // remove first combinator
    // ::v-deep a{color:red;} => :deep(a){color:red;}
    const firstNode = nodes[0];
    if (firstNode && firstNode.type === 'combinator' && firstNode.value === ' ') {
        nodes.shift();
    }
    selector.insertBefore(deepNode, postcss_selector_parser_1.default.pseudo({
        value: ':deep',
        nodes: [postcss_selector_parser_1.default.selector({ value: '', nodes })],
    }));
    selector.removeChild(deepNode);
}
function rewriteSelector(selector, selectorRoot) {
    let node = null;
    // find the last child node to insert attribute selector
    selector.each((n) => {
        // DEPRECATED ">>>" and "/deep/" combinator
        if (n.type === 'combinator' &&
            (n.value === '>>>' || n.value === '/deep/')) {
            n.value = ' ';
            n.spaces.before = n.spaces.after = '';
            // warn(
            //   `the >>> and /deep/ combinators have been deprecated. ` +
            //     `Use :deep() instead.`
            // )
            return false;
        }
        if (n.type === 'pseudo') {
            const { value } = n;
            // deep: inject [id] attribute at the node before the ::v-deep
            // combinator.
            if (value === ':deep' || value === '::v-deep') {
                if (n.nodes.length) {
                    // .foo ::v-deep(.bar) -> .foo[xxxxxxx] .bar
                    // replace the current node with ::v-deep's inner selector
                    let last = n;
                    n.nodes[0].each((ss) => {
                        selector.insertAfter(last, ss);
                        last = ss;
                    });
                    // insert a space combinator before if it doesn't already have one
                    const prev = selector.at(selector.index(n) - 1);
                    if (!prev || !isSpaceCombinator(prev)) {
                        selector.insertAfter(n, postcss_selector_parser_1.default.combinator({
                            value: ' ',
                        }));
                    }
                    selector.removeChild(n);
                }
                else {
                    // DEPRECATED usage
                    // .foo ::v-deep .bar -> .foo[xxxxxxx] .bar
                    // warn(
                    //   `::v-deep usage as a combinator has ` +
                    //     `been deprecated. Use :deep(<inner-selector>) instead.`
                    // )
                    const prev = selector.at(selector.index(n) - 1);
                    if (prev && isSpaceCombinator(prev)) {
                        selector.removeChild(prev);
                    }
                    selector.removeChild(n);
                }
                return false;
            }
            // slot: use selector inside `::v-slotted` and inject [id + '-s']
            // instead.
            // ::v-slotted(.foo) -> .foo[xxxxxxx-s]
            if (value === ':slotted' || value === '::v-slotted') {
                rewriteSelector(n.nodes[0], selectorRoot);
                let last = n;
                n.nodes[0].each((ss) => {
                    selector.insertAfter(last, ss);
                    last = ss;
                });
                // selector.insertAfter(n, n.nodes[0])
                selector.removeChild(n);
                // since slotted attribute already scopes the selector there's no
                // need for the non-slot attribute.
                return false;
            }
            // global: replace with inner selector and do not inject [id].
            // ::v-global(.foo) -> .foo
            if (value === ':global' || value === '::v-global') {
                selectorRoot.insertAfter(selector, n.nodes[0]);
                selectorRoot.removeChild(selector);
                return false;
            }
        }
        if (n.type !== 'pseudo' && n.type !== 'combinator') {
            node = n;
        }
    });
    if (node) {
        ;
        node.spaces.after = '';
    }
    else {
        // For deep selectors & standalone pseudo selectors,
        // the attribute selectors are prepended rather than appended.
        // So all leading spaces must be eliminated to avoid problems.
        selector.first.spaces.before = '';
    }
}
function isSpaceCombinator(node) {
    return node.type === 'combinator' && /^\s+$/.test(node.value);
}
scopedPlugin.postcss = true;
exports.default = scopedPlugin;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362619001, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.parseFilterNames = exports.missingModuleName = exports.parseRenderjs = exports.isRenderjs = exports.isSjs = exports.isWxs = void 0;
const url_1 = require("./vite/utils/url");
const WXS_RE = /vue&type=wxs/;
function isWxs(id) {
    return WXS_RE.test(id);
}
exports.isWxs = isWxs;
const SJS_RE = /vue&type=sjs/;
function isSjs(id) {
    return SJS_RE.test(id);
}
exports.isSjs = isSjs;
const RENDERJS_RE = /vue&type=renderjs/;
function isRenderjs(id) {
    return RENDERJS_RE.test(id);
}
exports.isRenderjs = isRenderjs;
function parseRenderjs(id) {
    if (isWxs(id) || isRenderjs(id) || isSjs(id)) {
        const { query, filename } = (0, url_1.parseVueRequest)(id);
        return {
            type: query.type,
            name: query.name,
            filename,
        };
    }
    return {
        type: '',
        name: '',
        filename: '',
    };
}
exports.parseRenderjs = parseRenderjs;
function missingModuleName(type, code) {
    return `<script module="missing module name" lang="${type}">
${code}
</script>`;
}
exports.missingModuleName = missingModuleName;
const moduleRE = /module=["'](.*?)["']/;
function parseFilterNames(lang, code) {
    const names = [];
    const scriptTags = code.match(/<script\b[^>]*>/gm);
    if (!scriptTags) {
        return names;
    }
    const langRE = new RegExp(`lang=["']${lang}["']`);
    scriptTags.forEach((scriptTag) => {
        if (langRE.test(scriptTag)) {
            const matches = scriptTag.match(moduleRE);
            if (matches) {
                names.push(matches[1]);
            }
        }
    });
    return names;
}
exports.parseFilterNames = parseFilterNames;

}, function(modId) { var map = {"./vite/utils/url":1781362618876}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362619002, function(require, module, exports) {

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
exports.esbuild = exports.transformWithEsbuild = void 0;
const path_1 = __importDefault(require("path"));
function transformWithEsbuild(code, filename, options) {
    options.stdin = {
        contents: code,
        resolveDir: path_1.default.dirname(filename),
    };
    return Promise.resolve().then(() => __importStar(require('esbuild'))).then((esbuild) => {
        return esbuild.build(options);
    });
}
exports.transformWithEsbuild = transformWithEsbuild;
function esbuild(options) {
    return Promise.resolve().then(() => __importStar(require('esbuild'))).then((esbuild) => {
        return esbuild.build(options);
    });
}
exports.esbuild = esbuild;

}, function(modId) { var map = {"esbuild":1781362619002}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362619004, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.isMiniProgramPlatform = exports.getPlatformDir = exports.getPlatforms = exports.registerPlatform = void 0;
const BUILT_IN_PLATFORMS = [
    'app',
    'app-plus',
    'app-harmony',
    'app-ios',
    'app-android',
    'h5',
    'web',
    'mp-360',
    'mp-alipay',
    'mp-baidu',
    'mp-jd',
    'mp-kuaishou',
    'mp-lark',
    'mp-qq',
    'mp-toutiao',
    'mp-weixin',
    'mp-xhs',
    'quickapp-webview',
    'quickapp-webview-huawei',
    'quickapp-webview-union',
];
const platforms = [...BUILT_IN_PLATFORMS];
function registerPlatform(platform) {
    if (platform === 'mp') {
        return;
    }
    if (!platforms.includes(platform)) {
        platforms.push(platform);
    }
}
exports.registerPlatform = registerPlatform;
function getPlatforms() {
    return platforms;
}
exports.getPlatforms = getPlatforms;
function getPlatformDir() {
    if (process.env.UNI_APP_X && process.env.UNI_PLATFORM === 'app') {
        return process.env.UNI_UTS_PLATFORM;
    }
    return process.env.UNI_SUB_PLATFORM || process.env.UNI_PLATFORM;
}
exports.getPlatformDir = getPlatformDir;
function isMiniProgramPlatform() {
    return !['app', 'app-plus', 'h5', 'web'].includes(process.env.UNI_PLATFORM);
}
exports.isMiniProgramPlatform = isMiniProgramPlatform;

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362619007, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.chokidar = void 0;
var chokidar_1 = require("chokidar");
Object.defineProperty(exports, "chokidar", { enumerable: true, get: function () { return __importDefault(chokidar_1).default; } });

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362619008, function(require, module, exports) {
/* eslint-disable */
// @ts-ignore

Object.defineProperty(exports, "__esModule", { value: true });
exports.checkUpdate = void 0;
var __importDefault = this && this.__importDefault || function (mod) { return mod && mod.__esModule ? mod : { default: mod }; };
Object.defineProperty(exports, "__esModule", { value: !0 }), exports.createPostData = exports.getMac = exports.md5 = exports.checkLocalCache = exports.checkUpdate1 = void 0;
const fs_extra_1 = __importDefault(require("fs-extra")), os_1 = __importDefault(require("os")), path_1 = __importDefault(require("path")), debug_1 = __importDefault(require("debug")), crypto_1 = __importDefault(require("crypto")), https_1 = require("https"), compare_versions_1 = __importDefault(require("compare-versions")), shared_1 = require("@vue/shared"), json_1 = require("./json"), hbx_1 = require("./hbx"), debugCheckUpdate = (0, debug_1.default)("uni:check-update"), INTERVAL = 864e5;
async function checkUpdate1(options) { if (process.env.CI)
    return void debugCheckUpdate("isInCI"); if ((0, hbx_1.isInHBuilderX)())
    return void debugCheckUpdate("isInHBuilderX"); const { inputDir: inputDir, compilerVersion: compilerVersion } = options, updateCache = readCheckUpdateCache(inputDir); debugCheckUpdate("read.cache", updateCache); const res = checkLocalCache(updateCache, compilerVersion); res ? (0, shared_1.isString)(res) && (console.log(), console.log(res)) : await checkVersion(options, normalizeUpdateCache(updateCache, (0, json_1.parseManifestJsonOnce)(inputDir))), writeCheckUpdateCache(inputDir, statUpdateCache(normalizeUpdateCache(updateCache))); }
function normalizeUpdateCache(updateCache, manifestJson) { const platform = process.env.UNI_PLATFORM; if (updateCache[platform] || (updateCache[platform] = { appid: "", dev: 0, build: 0 }), manifestJson) {
    const platformOptions = manifestJson["app" === platform ? "app-plus" : platform];
    updateCache[platform].appid = platformOptions && (platformOptions.appid || platformOptions.package) || "";
} return updateCache; }
function statUpdateCache(updateCache) { debugCheckUpdate("stat.before", updateCache); const platform = process.env.UNI_PLATFORM, type = "production" === process.env.NODE_ENV ? "build" : "dev", platformOptions = updateCache[platform]; return platformOptions[type] = (platformOptions[type] || 0) + 1, debugCheckUpdate("stat.after", updateCache), updateCache; }
function getFilepath(inputDir, filename) { return path_1.default.resolve(os_1.default.tmpdir(), "uni-app-cli", md5(inputDir), filename); }
function getCheckUpdateFilepath(inputDir) { return getFilepath(inputDir, "check-update.json"); }
function generateVid() { let result = ""; for (let i = 0; i < 4; i++)
    result += (65536 * (1 + Math.random()) | 0).toString(16).substring(1); return "UNI_" + result.toUpperCase(); }
function createCheckUpdateCache(vid = generateVid()) { return { vid: generateVid(), lastCheck: 0 }; }
function readCheckUpdateCache(inputDir) { const updateFilepath = getCheckUpdateFilepath(inputDir); if (debugCheckUpdate("read:", updateFilepath), fs_extra_1.default.existsSync(updateFilepath))
    try {
        return require(updateFilepath);
    }
    catch (e) {
        debugCheckUpdate("read.error", e);
    } return createCheckUpdateCache(); }
function checkLocalCache(updateCache, compilerVersion, interval = INTERVAL) { return updateCache.lastCheck ? Date.now() - updateCache.lastCheck > interval ? (debugCheckUpdate("cache: lastCheck > interval"), !1) : !(updateCache.newVersion && (0, compare_versions_1.default)(updateCache.newVersion, compilerVersion) > 0) || (debugCheckUpdate("cache: find new version"), updateCache.note) : (debugCheckUpdate("cache: lastCheck not found"), !1); }
function writeCheckUpdateCache(inputDir, updateCache) { const filepath = getCheckUpdateFilepath(inputDir); debugCheckUpdate("write:", filepath, updateCache); try {
    fs_extra_1.default.outputFileSync(filepath, JSON.stringify(updateCache));
}
catch (e) {
    debugCheckUpdate("write.error", e);
} }
function md5(str) { return crypto_1.default.createHash("md5").update(str).digest("hex"); }
function getMac() { let mac = ""; const network = os_1.default.networkInterfaces(); for (const key in network) {
    const array = network[key];
    for (let i = 0; i < array.length; i++) {
        const item = array[i];
        if (item.family && (!item.mac || "00:00:00:00:00:00" !== item.mac)) {
            if ((0, shared_1.isString)(item.family) && ("IPv4" === item.family || "IPv6" === item.family)) {
                mac = item.mac;
                break;
            }
            if ("number" == typeof item.family && (4 === item.family || 6 === item.family)) {
                mac = item.mac;
                break;
            }
        }
    }
} return mac; }
function createPostData({ versionType: versionType, compilerVersion: compilerVersion }, manifestJson, updateCache) { const data = { vv: 3, device: md5(getMac()), vtype: versionType, vcode: compilerVersion }; return manifestJson.appid ? data.appid = manifestJson.appid : data.vid = updateCache.vid, Object.keys(updateCache).forEach((name => { const value = updateCache[name]; (0, shared_1.isPlainObject)(value) && ((0, shared_1.hasOwn)(value, "dev") || (0, shared_1.hasOwn)(value, "build")) && (data[name] = value); })), JSON.stringify(data); }
function handleCheckVersion({ code: code, isUpdate: isUpdate, newVersion: newVersion, note: note }, updateCache) { 0 === code && (Object.keys(updateCache).forEach((key => { "vid" !== key && delete updateCache[key]; })), updateCache.lastCheck = Date.now(), isUpdate ? (updateCache.note = note, updateCache.newVersion = newVersion) : (delete updateCache.note, delete updateCache.newVersion)); }
exports.checkUpdate1 = checkUpdate1, exports.checkLocalCache = checkLocalCache, exports.md5 = md5, exports.getMac = getMac, exports.createPostData = createPostData;
const HOSTNAME = "uniapp.dcloud.net.cn", PATH = "/update/cli";
function checkVersion(options, updateCache) { return new Promise((resolve => { const postData = JSON.stringify({ id: createPostData(options, (0, json_1.parseManifestJsonOnce)(options.inputDir), updateCache) }); let responseData = ""; const req = (0, https_1.request)({ hostname: HOSTNAME, path: PATH, port: 443, method: "POST", headers: { "Content-Type": "application/json", "Content-Length": postData.length } }, (res => { res.setEncoding("utf8"), res.on("data", (chunk => { responseData += chunk; })), res.on("end", (() => { debugCheckUpdate("response: ", responseData); try {
    handleCheckVersion(JSON.parse(responseData), updateCache);
}
catch (e) { } resolve(!0); })), res.on("error", (e => { debugCheckUpdate("response.error:", e), resolve(!1); })); })).on("error", (e => { debugCheckUpdate("request.error:", e), resolve(!1); })); debugCheckUpdate("request: ", postData), req.write(postData), req.end(); })); }
exports.checkUpdate = checkUpdate1;

}, function(modId) { var map = {"./json":1781362618806,"./hbx":1781362618896}; return __REQUIRE__(map[modId], modId); })
return __REQUIRE__(1781362618792);
})()
//miniprogram-npm-outsideDeps=["./logs","./utils","./preprocess","./scripts","./uni_modules","./uni_modules.cloud","path","@babel/parser","../utils","@babel/types","estree-walker","fs","../json/app/manifest/nvue","@dcloudio/uni-shared","../logs","../vite/plugins/vitejs/utils","./pages","./manifest","./theme","./jsonFile","@vue/shared","../pages","../../utils","../manifest","../theme","jsonc-parser","../preprocess","merge","./uniConfig","../../../utils","../../pages","./plus","./nvue","./arguments","./safearea","./splashscreen","./confusion","./launchwebview","./tabBar","../../theme","../../manifest","./env","@dcloudio/uni-i18n","fast-glob","os-locale-s-fix","../../preprocess","../app/pages/uniConfig","../json/manifest","magic-string","../json/mp/jsonFile","@vue/compiler-core","@vue/compiler-dom","../plugins/vitejs/plugins/asset","../plugins/vitejs/plugins/css","@babel/code-frame","../plugins/vitejs/utils","base64url","module","debug","../vite/plugins/console","./log","./alias","fs-extra","./vite/plugins/uts/uni_modules","unimport","../../package.json","@dcloudio/uni-uts-v1/lib/typescript","../json/pages","../vite/plugins/vitejs/plugins/asset","../json/uni-x/manifest","@dcloudio/compiler-vapor-dom2","./transformRefresherSlot","./x/transformDirection","@rollup/pluginutils","./json/pages","url","./cloud","./inject","./console","./uts/uni_modules","./sourceMap","./sfc","./vitejs/plugins/asset","./vitejs/plugins/css","./vitejs/utils","../../vue/utils","../../logs","chokidar","unplugin-auto-import/vite","../../../uni_modules","hash-sum","./plugins/stylePluginExternal","./plugins/uniapp","autoprefixer","postcss-selector-parser","os","crypto","https","compare-versions"]
//# sourceMappingURL=index.js.map