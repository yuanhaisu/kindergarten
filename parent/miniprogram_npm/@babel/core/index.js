module.exports = (function() {
var __MODS__ = {};
var __DEFINE__ = function(modId, func, req) { var m = { exports: {}, _tempexports: {} }; __MODS__[modId] = { status: 0, func: func, req: req, m: m }; };
var __REQUIRE__ = function(modId, source) { if(!__MODS__[modId]) return require(source); if(!__MODS__[modId].status) { var m = __MODS__[modId].m; m._exports = m._tempexports; var desp = Object.getOwnPropertyDescriptor(m, "exports"); if (desp && desp.configurable) Object.defineProperty(m, "exports", { set: function (val) { if(typeof val === "object" && val !== m._exports) { m._exports.__proto__ = val.__proto__; Object.keys(val).forEach(function (k) { m._exports[k] = val[k]; }); } m._tempexports = val }, get: function () { return m._tempexports; } }); __MODS__[modId].status = 1; __MODS__[modId].func(__MODS__[modId].req, m, m.exports); } return __MODS__[modId].m.exports; };
var __REQUIRE_WILDCARD__ = function(obj) { if(obj && obj.__esModule) { return obj; } else { var newObj = {}; if(obj != null) { for(var k in obj) { if (Object.prototype.hasOwnProperty.call(obj, k)) newObj[k] = obj[k]; } } newObj.default = obj; return newObj; } };
var __REQUIRE_DEFAULT__ = function(obj) { return obj && obj.__esModule ? obj.default : obj; };
__DEFINE__(1781530614368, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.DEFAULT_EXTENSIONS = void 0;
Object.defineProperty(exports, "File", {
  enumerable: true,
  get: function () {
    return _file.default;
  }
});
Object.defineProperty(exports, "buildExternalHelpers", {
  enumerable: true,
  get: function () {
    return _buildExternalHelpers.default;
  }
});
Object.defineProperty(exports, "createConfigItem", {
  enumerable: true,
  get: function () {
    return _index2.createConfigItem;
  }
});
Object.defineProperty(exports, "createConfigItemAsync", {
  enumerable: true,
  get: function () {
    return _index2.createConfigItemAsync;
  }
});
Object.defineProperty(exports, "createConfigItemSync", {
  enumerable: true,
  get: function () {
    return _index2.createConfigItemSync;
  }
});
Object.defineProperty(exports, "getEnv", {
  enumerable: true,
  get: function () {
    return _environment.getEnv;
  }
});
Object.defineProperty(exports, "loadOptions", {
  enumerable: true,
  get: function () {
    return _index2.loadOptions;
  }
});
Object.defineProperty(exports, "loadOptionsAsync", {
  enumerable: true,
  get: function () {
    return _index2.loadOptionsAsync;
  }
});
Object.defineProperty(exports, "loadOptionsSync", {
  enumerable: true,
  get: function () {
    return _index2.loadOptionsSync;
  }
});
Object.defineProperty(exports, "loadPartialConfig", {
  enumerable: true,
  get: function () {
    return _index2.loadPartialConfig;
  }
});
Object.defineProperty(exports, "loadPartialConfigAsync", {
  enumerable: true,
  get: function () {
    return _index2.loadPartialConfigAsync;
  }
});
Object.defineProperty(exports, "loadPartialConfigSync", {
  enumerable: true,
  get: function () {
    return _index2.loadPartialConfigSync;
  }
});
Object.defineProperty(exports, "parse", {
  enumerable: true,
  get: function () {
    return _parse.parse;
  }
});
Object.defineProperty(exports, "parseAsync", {
  enumerable: true,
  get: function () {
    return _parse.parseAsync;
  }
});
Object.defineProperty(exports, "parseSync", {
  enumerable: true,
  get: function () {
    return _parse.parseSync;
  }
});
Object.defineProperty(exports, "resolvePlugin", {
  enumerable: true,
  get: function () {
    return _index.resolvePlugin;
  }
});
Object.defineProperty(exports, "resolvePreset", {
  enumerable: true,
  get: function () {
    return _index.resolvePreset;
  }
});
Object.defineProperty((0, exports), "template", {
  enumerable: true,
  get: function () {
    return _template().default;
  }
});
Object.defineProperty((0, exports), "tokTypes", {
  enumerable: true,
  get: function () {
    return _parser().tokTypes;
  }
});
Object.defineProperty(exports, "transform", {
  enumerable: true,
  get: function () {
    return _transform.transform;
  }
});
Object.defineProperty(exports, "transformAsync", {
  enumerable: true,
  get: function () {
    return _transform.transformAsync;
  }
});
Object.defineProperty(exports, "transformFile", {
  enumerable: true,
  get: function () {
    return _transformFile.transformFile;
  }
});
Object.defineProperty(exports, "transformFileAsync", {
  enumerable: true,
  get: function () {
    return _transformFile.transformFileAsync;
  }
});
Object.defineProperty(exports, "transformFileSync", {
  enumerable: true,
  get: function () {
    return _transformFile.transformFileSync;
  }
});
Object.defineProperty(exports, "transformFromAst", {
  enumerable: true,
  get: function () {
    return _transformAst.transformFromAst;
  }
});
Object.defineProperty(exports, "transformFromAstAsync", {
  enumerable: true,
  get: function () {
    return _transformAst.transformFromAstAsync;
  }
});
Object.defineProperty(exports, "transformFromAstSync", {
  enumerable: true,
  get: function () {
    return _transformAst.transformFromAstSync;
  }
});
Object.defineProperty(exports, "transformSync", {
  enumerable: true,
  get: function () {
    return _transform.transformSync;
  }
});
Object.defineProperty((0, exports), "traverse", {
  enumerable: true,
  get: function () {
    return _traverse().default;
  }
});
exports.version = exports.types = void 0;
var _file = require("./transformation/file/file.js");
var _buildExternalHelpers = require("./tools/build-external-helpers.js");
var _index = require("./config/files/index.js");
var _environment = require("./config/helpers/environment.js");
function _types() {
  const data = require("@babel/types");
  _types = function () {
    return data;
  };
  return data;
}
Object.defineProperty((0, exports), "types", {
  enumerable: true,
  get: function () {
    return _types();
  }
});
function _parser() {
  const data = require("@babel/parser");
  _parser = function () {
    return data;
  };
  return data;
}
function _traverse() {
  const data = require("@babel/traverse");
  _traverse = function () {
    return data;
  };
  return data;
}
function _template() {
  const data = require("@babel/template");
  _template = function () {
    return data;
  };
  return data;
}
var _index2 = require("./config/index.js");
var _transform = require("./transform.js");
var _transformFile = require("./transform-file.js");
var _transformAst = require("./transform-ast.js");
var _parse = require("./parse.js");
var thisFile = require("./index.js");
;
const version = exports.version = "7.25.2";
const DEFAULT_EXTENSIONS = exports.DEFAULT_EXTENSIONS = Object.freeze([".js", ".jsx", ".es6", ".es", ".mjs", ".cjs"]);
;
{
  exports.OptionManager = class OptionManager {
    init(opts) {
      return (0, _index2.loadOptionsSync)(opts);
    }
  };
  exports.Plugin = function Plugin(alias) {
    throw new Error(`The (${alias}) Babel 5 plugin is being run with an unsupported Babel version.`);
  };
}
0 && (exports.types = exports.traverse = exports.tokTypes = exports.template = 0);

//# sourceMappingURL=index.js.map

}, function(modId) {var map = {"./transformation/file/file.js":1781530614369,"./tools/build-external-helpers.js":1781530614370,"./config/files/index.js":1781530614371,"./config/helpers/environment.js":1781530614400,"./config/index.js":1781530614384,"./transform.js":1781530614413,"./transform-file.js":1781530614383,"./transform-ast.js":1781530614414,"./parse.js":1781530614415,"./index.js":1781530614368}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614369, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
function helpers() {
  const data = require("@babel/helpers");
  helpers = function () {
    return data;
  };
  return data;
}
function _traverse() {
  const data = require("@babel/traverse");
  _traverse = function () {
    return data;
  };
  return data;
}
function _codeFrame() {
  const data = require("@babel/code-frame");
  _codeFrame = function () {
    return data;
  };
  return data;
}
function _t() {
  const data = require("@babel/types");
  _t = function () {
    return data;
  };
  return data;
}
function _semver() {
  const data = require("semver");
  _semver = function () {
    return data;
  };
  return data;
}
var babel7 = _interopRequireWildcard(require("./babel-7-helpers.cjs"), true);
function _getRequireWildcardCache(e) { if ("function" != typeof WeakMap) return null; var r = new WeakMap(), t = new WeakMap(); return (_getRequireWildcardCache = function (e) { return e ? t : r; })(e); }
function _interopRequireWildcard(e, r) { if (!r && e && e.__esModule) return e; if (null === e || "object" != typeof e && "function" != typeof e) return { default: e }; var t = _getRequireWildcardCache(r); if (t && t.has(e)) return t.get(e); var n = { __proto__: null }, a = Object.defineProperty && Object.getOwnPropertyDescriptor; for (var u in e) if ("default" !== u && {}.hasOwnProperty.call(e, u)) { var i = a ? Object.getOwnPropertyDescriptor(e, u) : null; i && (i.get || i.set) ? Object.defineProperty(n, u, i) : n[u] = e[u]; } return n.default = e, t && t.set(e, n), n; }
const {
  cloneNode,
  interpreterDirective
} = _t();
const errorVisitor = {
  enter(path, state) {
    const loc = path.node.loc;
    if (loc) {
      state.loc = loc;
      path.stop();
    }
  }
};
class File {
  constructor(options, {
    code,
    ast,
    inputMap
  }) {
    this._map = new Map();
    this.opts = void 0;
    this.declarations = {};
    this.path = void 0;
    this.ast = void 0;
    this.scope = void 0;
    this.metadata = {};
    this.code = "";
    this.inputMap = void 0;
    this.hub = {
      file: this,
      getCode: () => this.code,
      getScope: () => this.scope,
      addHelper: this.addHelper.bind(this),
      buildError: this.buildCodeFrameError.bind(this)
    };
    this.opts = options;
    this.code = code;
    this.ast = ast;
    this.inputMap = inputMap;
    this.path = _traverse().NodePath.get({
      hub: this.hub,
      parentPath: null,
      parent: this.ast,
      container: this.ast,
      key: "program"
    }).setContext();
    this.scope = this.path.scope;
  }
  get shebang() {
    const {
      interpreter
    } = this.path.node;
    return interpreter ? interpreter.value : "";
  }
  set shebang(value) {
    if (value) {
      this.path.get("interpreter").replaceWith(interpreterDirective(value));
    } else {
      this.path.get("interpreter").remove();
    }
  }
  set(key, val) {
    {
      if (key === "helpersNamespace") {
        throw new Error("Babel 7.0.0-beta.56 has dropped support for the 'helpersNamespace' utility." + "If you are using @babel/plugin-external-helpers you will need to use a newer " + "version than the one you currently have installed. " + "If you have your own implementation, you'll want to explore using 'helperGenerator' " + "alongside 'file.availableHelper()'.");
      }
    }
    this._map.set(key, val);
  }
  get(key) {
    return this._map.get(key);
  }
  has(key) {
    return this._map.has(key);
  }
  availableHelper(name, versionRange) {
    let minVersion;
    try {
      minVersion = helpers().minVersion(name);
    } catch (err) {
      if (err.code !== "BABEL_HELPER_UNKNOWN") throw err;
      return false;
    }
    if (typeof versionRange !== "string") return true;
    if (_semver().valid(versionRange)) versionRange = `^${versionRange}`;
    {
      return !_semver().intersects(`<${minVersion}`, versionRange) && !_semver().intersects(`>=8.0.0`, versionRange);
    }
  }
  addHelper(name) {
    const declar = this.declarations[name];
    if (declar) return cloneNode(declar);
    const generator = this.get("helperGenerator");
    if (generator) {
      const res = generator(name);
      if (res) return res;
    }
    helpers().minVersion(name);
    const uid = this.declarations[name] = this.scope.generateUidIdentifier(name);
    const dependencies = {};
    for (const dep of helpers().getDependencies(name)) {
      dependencies[dep] = this.addHelper(dep);
    }
    const {
      nodes,
      globals
    } = helpers().get(name, dep => dependencies[dep], uid.name, Object.keys(this.scope.getAllBindings()));
    globals.forEach(name => {
      if (this.path.scope.hasBinding(name, true)) {
        this.path.scope.rename(name);
      }
    });
    nodes.forEach(node => {
      node._compact = true;
    });
    const added = this.path.unshiftContainer("body", nodes);
    for (const path of added) {
      if (path.isVariableDeclaration()) this.scope.registerDeclaration(path);
    }
    return uid;
  }
  buildCodeFrameError(node, msg, _Error = SyntaxError) {
    let loc = node == null ? void 0 : node.loc;
    if (!loc && node) {
      const state = {
        loc: null
      };
      (0, _traverse().default)(node, errorVisitor, this.scope, state);
      loc = state.loc;
      let txt = "This is an error on an internal node. Probably an internal error.";
      if (loc) txt += " Location has been estimated.";
      msg += ` (${txt})`;
    }
    if (loc) {
      const {
        highlightCode = true
      } = this.opts;
      msg += "\n" + (0, _codeFrame().codeFrameColumns)(this.code, {
        start: {
          line: loc.start.line,
          column: loc.start.column + 1
        },
        end: loc.end && loc.start.line === loc.end.line ? {
          line: loc.end.line,
          column: loc.end.column + 1
        } : undefined
      }, {
        highlightCode
      });
    }
    return new _Error(msg);
  }
}
exports.default = File;
{
  File.prototype.addImport = function addImport() {
    throw new Error("This API has been removed. If you're looking for this " + "functionality in Babel 7, you should import the " + "'@babel/helper-module-imports' module and use the functions exposed " + " from that module, such as 'addNamed' or 'addDefault'.");
  };
  File.prototype.addTemplateObject = function addTemplateObject() {
    throw new Error("This function has been moved into the template literal transform itself.");
  };
  {
    File.prototype.getModuleName = function getModuleName() {
      return babel7.getModuleName()(this.opts, this.opts);
    };
  }
}
0 && 0;

//# sourceMappingURL=file.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614370, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = _default;
function helpers() {
  const data = require("@babel/helpers");
  helpers = function () {
    return data;
  };
  return data;
}
function _generator() {
  const data = require("@babel/generator");
  _generator = function () {
    return data;
  };
  return data;
}
function _template() {
  const data = require("@babel/template");
  _template = function () {
    return data;
  };
  return data;
}
function _t() {
  const data = require("@babel/types");
  _t = function () {
    return data;
  };
  return data;
}
const {
  arrayExpression,
  assignmentExpression,
  binaryExpression,
  blockStatement,
  callExpression,
  cloneNode,
  conditionalExpression,
  exportNamedDeclaration,
  exportSpecifier,
  expressionStatement,
  functionExpression,
  identifier,
  memberExpression,
  objectExpression,
  program,
  stringLiteral,
  unaryExpression,
  variableDeclaration,
  variableDeclarator
} = _t();
const buildUmdWrapper = replacements => _template().default.statement`
    (function (root, factory) {
      if (typeof define === "function" && define.amd) {
        define(AMD_ARGUMENTS, factory);
      } else if (typeof exports === "object") {
        factory(COMMON_ARGUMENTS);
      } else {
        factory(BROWSER_ARGUMENTS);
      }
    })(UMD_ROOT, function (FACTORY_PARAMETERS) {
      FACTORY_BODY
    });
  `(replacements);
function buildGlobal(allowlist) {
  const namespace = identifier("babelHelpers");
  const body = [];
  const container = functionExpression(null, [identifier("global")], blockStatement(body));
  const tree = program([expressionStatement(callExpression(container, [conditionalExpression(binaryExpression("===", unaryExpression("typeof", identifier("global")), stringLiteral("undefined")), identifier("self"), identifier("global"))]))]);
  body.push(variableDeclaration("var", [variableDeclarator(namespace, assignmentExpression("=", memberExpression(identifier("global"), namespace), objectExpression([])))]));
  buildHelpers(body, namespace, allowlist);
  return tree;
}
function buildModule(allowlist) {
  const body = [];
  const refs = buildHelpers(body, null, allowlist);
  body.unshift(exportNamedDeclaration(null, Object.keys(refs).map(name => {
    return exportSpecifier(cloneNode(refs[name]), identifier(name));
  })));
  return program(body, [], "module");
}
function buildUmd(allowlist) {
  const namespace = identifier("babelHelpers");
  const body = [];
  body.push(variableDeclaration("var", [variableDeclarator(namespace, identifier("global"))]));
  buildHelpers(body, namespace, allowlist);
  return program([buildUmdWrapper({
    FACTORY_PARAMETERS: identifier("global"),
    BROWSER_ARGUMENTS: assignmentExpression("=", memberExpression(identifier("root"), namespace), objectExpression([])),
    COMMON_ARGUMENTS: identifier("exports"),
    AMD_ARGUMENTS: arrayExpression([stringLiteral("exports")]),
    FACTORY_BODY: body,
    UMD_ROOT: identifier("this")
  })]);
}
function buildVar(allowlist) {
  const namespace = identifier("babelHelpers");
  const body = [];
  body.push(variableDeclaration("var", [variableDeclarator(namespace, objectExpression([]))]));
  const tree = program(body);
  buildHelpers(body, namespace, allowlist);
  body.push(expressionStatement(namespace));
  return tree;
}
function buildHelpers(body, namespace, allowlist) {
  const getHelperReference = name => {
    return namespace ? memberExpression(namespace, identifier(name)) : identifier(`_${name}`);
  };
  const refs = {};
  helpers().list.forEach(function (name) {
    if (allowlist && !allowlist.includes(name)) return;
    const ref = refs[name] = getHelperReference(name);
    const {
      nodes
    } = helpers().get(name, getHelperReference, namespace ? null : `_${name}`, [], namespace ? (ast, exportName, mapExportBindingAssignments) => {
      mapExportBindingAssignments(node => assignmentExpression("=", ref, node));
      ast.body.push(expressionStatement(assignmentExpression("=", ref, identifier(exportName))));
    } : null);
    body.push(...nodes);
  });
  return refs;
}
function _default(allowlist, outputType = "global") {
  let tree;
  const build = {
    global: buildGlobal,
    module: buildModule,
    umd: buildUmd,
    var: buildVar
  }[outputType];
  if (build) {
    tree = build(allowlist);
  } else {
    throw new Error(`Unsupported output type ${outputType}`);
  }
  return (0, _generator().default)(tree).code;
}
0 && 0;

//# sourceMappingURL=build-external-helpers.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614371, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
Object.defineProperty(exports, "ROOT_CONFIG_FILENAMES", {
  enumerable: true,
  get: function () {
    return _configuration.ROOT_CONFIG_FILENAMES;
  }
});
Object.defineProperty(exports, "findConfigUpwards", {
  enumerable: true,
  get: function () {
    return _configuration.findConfigUpwards;
  }
});
Object.defineProperty(exports, "findPackageData", {
  enumerable: true,
  get: function () {
    return _package.findPackageData;
  }
});
Object.defineProperty(exports, "findRelativeConfig", {
  enumerable: true,
  get: function () {
    return _configuration.findRelativeConfig;
  }
});
Object.defineProperty(exports, "findRootConfig", {
  enumerable: true,
  get: function () {
    return _configuration.findRootConfig;
  }
});
Object.defineProperty(exports, "loadConfig", {
  enumerable: true,
  get: function () {
    return _configuration.loadConfig;
  }
});
Object.defineProperty(exports, "loadPlugin", {
  enumerable: true,
  get: function () {
    return _plugins.loadPlugin;
  }
});
Object.defineProperty(exports, "loadPreset", {
  enumerable: true,
  get: function () {
    return _plugins.loadPreset;
  }
});
Object.defineProperty(exports, "resolvePlugin", {
  enumerable: true,
  get: function () {
    return _plugins.resolvePlugin;
  }
});
Object.defineProperty(exports, "resolvePreset", {
  enumerable: true,
  get: function () {
    return _plugins.resolvePreset;
  }
});
Object.defineProperty(exports, "resolveShowConfigPath", {
  enumerable: true,
  get: function () {
    return _configuration.resolveShowConfigPath;
  }
});
var _package = require("./package.js");
var _configuration = require("./configuration.js");
var _plugins = require("./plugins.js");
({});
0 && 0;

//# sourceMappingURL=index.js.map

}, function(modId) { var map = {"./package.js":1781530614372,"./configuration.js":1781530614380,"./plugins.js":1781530614411}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614372, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.findPackageData = findPackageData;
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
var _utils = require("./utils.js");
var _configError = require("../../errors/config-error.js");
const PACKAGE_FILENAME = "package.json";
const readConfigPackage = (0, _utils.makeStaticFileCache)((filepath, content) => {
  let options;
  try {
    options = JSON.parse(content);
  } catch (err) {
    throw new _configError.default(`Error while parsing JSON - ${err.message}`, filepath);
  }
  if (!options) throw new Error(`${filepath}: No config detected`);
  if (typeof options !== "object") {
    throw new _configError.default(`Config returned typeof ${typeof options}`, filepath);
  }
  if (Array.isArray(options)) {
    throw new _configError.default(`Expected config object but found array`, filepath);
  }
  return {
    filepath,
    dirname: _path().dirname(filepath),
    options
  };
});
function* findPackageData(filepath) {
  let pkg = null;
  const directories = [];
  let isPackage = true;
  let dirname = _path().dirname(filepath);
  while (!pkg && _path().basename(dirname) !== "node_modules") {
    directories.push(dirname);
    pkg = yield* readConfigPackage(_path().join(dirname, PACKAGE_FILENAME));
    const nextLoc = _path().dirname(dirname);
    if (dirname === nextLoc) {
      isPackage = false;
      break;
    }
    dirname = nextLoc;
  }
  return {
    filepath,
    directories,
    pkg,
    isPackage
  };
}
0 && 0;

//# sourceMappingURL=package.js.map

}, function(modId) { var map = {"./utils.js":1781530614373,"../../errors/config-error.js":1781530614378}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614373, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.makeStaticFileCache = makeStaticFileCache;
var _caching = require("../caching.js");
var fs = require("../../gensync-utils/fs.js");
function _fs2() {
  const data = require("fs");
  _fs2 = function () {
    return data;
  };
  return data;
}
function makeStaticFileCache(fn) {
  return (0, _caching.makeStrongCache)(function* (filepath, cache) {
    const cached = cache.invalidate(() => fileMtime(filepath));
    if (cached === null) {
      return null;
    }
    return fn(filepath, yield* fs.readFile(filepath, "utf8"));
  });
}
function fileMtime(filepath) {
  if (!_fs2().existsSync(filepath)) return null;
  try {
    return +_fs2().statSync(filepath).mtime;
  } catch (e) {
    if (e.code !== "ENOENT" && e.code !== "ENOTDIR") throw e;
  }
  return null;
}
0 && 0;

//# sourceMappingURL=utils.js.map

}, function(modId) { var map = {"../caching.js":1781530614374,"../../gensync-utils/fs.js":1781530614377}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614374, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.assertSimpleType = assertSimpleType;
exports.makeStrongCache = makeStrongCache;
exports.makeStrongCacheSync = makeStrongCacheSync;
exports.makeWeakCache = makeWeakCache;
exports.makeWeakCacheSync = makeWeakCacheSync;
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
var _async = require("../gensync-utils/async.js");
var _util = require("./util.js");
const synchronize = gen => {
  return _gensync()(gen).sync;
};
function* genTrue() {
  return true;
}
function makeWeakCache(handler) {
  return makeCachedFunction(WeakMap, handler);
}
function makeWeakCacheSync(handler) {
  return synchronize(makeWeakCache(handler));
}
function makeStrongCache(handler) {
  return makeCachedFunction(Map, handler);
}
function makeStrongCacheSync(handler) {
  return synchronize(makeStrongCache(handler));
}
function makeCachedFunction(CallCache, handler) {
  const callCacheSync = new CallCache();
  const callCacheAsync = new CallCache();
  const futureCache = new CallCache();
  return function* cachedFunction(arg, data) {
    const asyncContext = yield* (0, _async.isAsync)();
    const callCache = asyncContext ? callCacheAsync : callCacheSync;
    const cached = yield* getCachedValueOrWait(asyncContext, callCache, futureCache, arg, data);
    if (cached.valid) return cached.value;
    const cache = new CacheConfigurator(data);
    const handlerResult = handler(arg, cache);
    let finishLock;
    let value;
    if ((0, _util.isIterableIterator)(handlerResult)) {
      value = yield* (0, _async.onFirstPause)(handlerResult, () => {
        finishLock = setupAsyncLocks(cache, futureCache, arg);
      });
    } else {
      value = handlerResult;
    }
    updateFunctionCache(callCache, cache, arg, value);
    if (finishLock) {
      futureCache.delete(arg);
      finishLock.release(value);
    }
    return value;
  };
}
function* getCachedValue(cache, arg, data) {
  const cachedValue = cache.get(arg);
  if (cachedValue) {
    for (const {
      value,
      valid
    } of cachedValue) {
      if (yield* valid(data)) return {
        valid: true,
        value
      };
    }
  }
  return {
    valid: false,
    value: null
  };
}
function* getCachedValueOrWait(asyncContext, callCache, futureCache, arg, data) {
  const cached = yield* getCachedValue(callCache, arg, data);
  if (cached.valid) {
    return cached;
  }
  if (asyncContext) {
    const cached = yield* getCachedValue(futureCache, arg, data);
    if (cached.valid) {
      const value = yield* (0, _async.waitFor)(cached.value.promise);
      return {
        valid: true,
        value
      };
    }
  }
  return {
    valid: false,
    value: null
  };
}
function setupAsyncLocks(config, futureCache, arg) {
  const finishLock = new Lock();
  updateFunctionCache(futureCache, config, arg, finishLock);
  return finishLock;
}
function updateFunctionCache(cache, config, arg, value) {
  if (!config.configured()) config.forever();
  let cachedValue = cache.get(arg);
  config.deactivate();
  switch (config.mode()) {
    case "forever":
      cachedValue = [{
        value,
        valid: genTrue
      }];
      cache.set(arg, cachedValue);
      break;
    case "invalidate":
      cachedValue = [{
        value,
        valid: config.validator()
      }];
      cache.set(arg, cachedValue);
      break;
    case "valid":
      if (cachedValue) {
        cachedValue.push({
          value,
          valid: config.validator()
        });
      } else {
        cachedValue = [{
          value,
          valid: config.validator()
        }];
        cache.set(arg, cachedValue);
      }
  }
}
class CacheConfigurator {
  constructor(data) {
    this._active = true;
    this._never = false;
    this._forever = false;
    this._invalidate = false;
    this._configured = false;
    this._pairs = [];
    this._data = void 0;
    this._data = data;
  }
  simple() {
    return makeSimpleConfigurator(this);
  }
  mode() {
    if (this._never) return "never";
    if (this._forever) return "forever";
    if (this._invalidate) return "invalidate";
    return "valid";
  }
  forever() {
    if (!this._active) {
      throw new Error("Cannot change caching after evaluation has completed.");
    }
    if (this._never) {
      throw new Error("Caching has already been configured with .never()");
    }
    this._forever = true;
    this._configured = true;
  }
  never() {
    if (!this._active) {
      throw new Error("Cannot change caching after evaluation has completed.");
    }
    if (this._forever) {
      throw new Error("Caching has already been configured with .forever()");
    }
    this._never = true;
    this._configured = true;
  }
  using(handler) {
    if (!this._active) {
      throw new Error("Cannot change caching after evaluation has completed.");
    }
    if (this._never || this._forever) {
      throw new Error("Caching has already been configured with .never or .forever()");
    }
    this._configured = true;
    const key = handler(this._data);
    const fn = (0, _async.maybeAsync)(handler, `You appear to be using an async cache handler, but Babel has been called synchronously`);
    if ((0, _async.isThenable)(key)) {
      return key.then(key => {
        this._pairs.push([key, fn]);
        return key;
      });
    }
    this._pairs.push([key, fn]);
    return key;
  }
  invalidate(handler) {
    this._invalidate = true;
    return this.using(handler);
  }
  validator() {
    const pairs = this._pairs;
    return function* (data) {
      for (const [key, fn] of pairs) {
        if (key !== (yield* fn(data))) return false;
      }
      return true;
    };
  }
  deactivate() {
    this._active = false;
  }
  configured() {
    return this._configured;
  }
}
function makeSimpleConfigurator(cache) {
  function cacheFn(val) {
    if (typeof val === "boolean") {
      if (val) cache.forever();else cache.never();
      return;
    }
    return cache.using(() => assertSimpleType(val()));
  }
  cacheFn.forever = () => cache.forever();
  cacheFn.never = () => cache.never();
  cacheFn.using = cb => cache.using(() => assertSimpleType(cb()));
  cacheFn.invalidate = cb => cache.invalidate(() => assertSimpleType(cb()));
  return cacheFn;
}
function assertSimpleType(value) {
  if ((0, _async.isThenable)(value)) {
    throw new Error(`You appear to be using an async cache handler, ` + `which your current version of Babel does not support. ` + `We may add support for this in the future, ` + `but if you're on the most recent version of @babel/core and still ` + `seeing this error, then you'll need to synchronously handle your caching logic.`);
  }
  if (value != null && typeof value !== "string" && typeof value !== "boolean" && typeof value !== "number") {
    throw new Error("Cache keys must be either string, boolean, number, null, or undefined.");
  }
  return value;
}
class Lock {
  constructor() {
    this.released = false;
    this.promise = void 0;
    this._resolve = void 0;
    this.promise = new Promise(resolve => {
      this._resolve = resolve;
    });
  }
  release(value) {
    this.released = true;
    this._resolve(value);
  }
}
0 && 0;

//# sourceMappingURL=caching.js.map

}, function(modId) { var map = {"../gensync-utils/async.js":1781530614375,"./util.js":1781530614376}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614375, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.forwardAsync = forwardAsync;
exports.isAsync = void 0;
exports.isThenable = isThenable;
exports.maybeAsync = maybeAsync;
exports.waitFor = exports.onFirstPause = void 0;
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
function asyncGeneratorStep(gen, resolve, reject, _next, _throw, key, arg) { try { var info = gen[key](arg); var value = info.value; } catch (error) { reject(error); return; } if (info.done) { resolve(value); } else { Promise.resolve(value).then(_next, _throw); } }
function _asyncToGenerator(fn) { return function () { var self = this, args = arguments; return new Promise(function (resolve, reject) { var gen = fn.apply(self, args); function _next(value) { asyncGeneratorStep(gen, resolve, reject, _next, _throw, "next", value); } function _throw(err) { asyncGeneratorStep(gen, resolve, reject, _next, _throw, "throw", err); } _next(undefined); }); }; }
const runGenerator = _gensync()(function* (item) {
  return yield* item;
});
const isAsync = exports.isAsync = _gensync()({
  sync: () => false,
  errback: cb => cb(null, true)
});
function maybeAsync(fn, message) {
  return _gensync()({
    sync(...args) {
      const result = fn.apply(this, args);
      if (isThenable(result)) throw new Error(message);
      return result;
    },
    async(...args) {
      return Promise.resolve(fn.apply(this, args));
    }
  });
}
const withKind = _gensync()({
  sync: cb => cb("sync"),
  async: function () {
    var _ref = _asyncToGenerator(function* (cb) {
      return cb("async");
    });
    return function async(_x) {
      return _ref.apply(this, arguments);
    };
  }()
});
function forwardAsync(action, cb) {
  const g = _gensync()(action);
  return withKind(kind => {
    const adapted = g[kind];
    return cb(adapted);
  });
}
const onFirstPause = exports.onFirstPause = _gensync()({
  name: "onFirstPause",
  arity: 2,
  sync: function (item) {
    return runGenerator.sync(item);
  },
  errback: function (item, firstPause, cb) {
    let completed = false;
    runGenerator.errback(item, (err, value) => {
      completed = true;
      cb(err, value);
    });
    if (!completed) {
      firstPause();
    }
  }
});
const waitFor = exports.waitFor = _gensync()({
  sync: x => x,
  async: function () {
    var _ref2 = _asyncToGenerator(function* (x) {
      return x;
    });
    return function async(_x2) {
      return _ref2.apply(this, arguments);
    };
  }()
});
function isThenable(val) {
  return !!val && (typeof val === "object" || typeof val === "function") && !!val.then && typeof val.then === "function";
}
0 && 0;

//# sourceMappingURL=async.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614376, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.isIterableIterator = isIterableIterator;
exports.mergeOptions = mergeOptions;
function mergeOptions(target, source) {
  for (const k of Object.keys(source)) {
    if ((k === "parserOpts" || k === "generatorOpts" || k === "assumptions") && source[k]) {
      const parserOpts = source[k];
      const targetObj = target[k] || (target[k] = {});
      mergeDefaultFields(targetObj, parserOpts);
    } else {
      const val = source[k];
      if (val !== undefined) target[k] = val;
    }
  }
}
function mergeDefaultFields(target, source) {
  for (const k of Object.keys(source)) {
    const val = source[k];
    if (val !== undefined) target[k] = val;
  }
}
function isIterableIterator(value) {
  return !!value && typeof value.next === "function" && typeof value[Symbol.iterator] === "function";
}
0 && 0;

//# sourceMappingURL=util.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614377, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.stat = exports.readFile = void 0;
function _fs() {
  const data = require("fs");
  _fs = function () {
    return data;
  };
  return data;
}
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
const readFile = exports.readFile = _gensync()({
  sync: _fs().readFileSync,
  errback: _fs().readFile
});
const stat = exports.stat = _gensync()({
  sync: _fs().statSync,
  errback: _fs().stat
});
0 && 0;

//# sourceMappingURL=fs.js.map

}, function(modId) { var map = {"fs":1781530614377}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614378, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
var _rewriteStackTrace = require("./rewrite-stack-trace.js");
class ConfigError extends Error {
  constructor(message, filename) {
    super(message);
    (0, _rewriteStackTrace.expectedError)(this);
    if (filename) (0, _rewriteStackTrace.injectVirtualStackFrame)(this, filename);
  }
}
exports.default = ConfigError;
0 && 0;

//# sourceMappingURL=config-error.js.map

}, function(modId) { var map = {"./rewrite-stack-trace.js":1781530614379}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614379, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.beginHiddenCallStack = beginHiddenCallStack;
exports.endHiddenCallStack = endHiddenCallStack;
exports.expectedError = expectedError;
exports.injectVirtualStackFrame = injectVirtualStackFrame;
var _Object$getOwnPropert;
const ErrorToString = Function.call.bind(Error.prototype.toString);
const SUPPORTED = !!Error.captureStackTrace && ((_Object$getOwnPropert = Object.getOwnPropertyDescriptor(Error, "stackTraceLimit")) == null ? void 0 : _Object$getOwnPropert.writable) === true;
const START_HIDING = "startHiding - secret - don't use this - v1";
const STOP_HIDING = "stopHiding - secret - don't use this - v1";
const expectedErrors = new WeakSet();
const virtualFrames = new WeakMap();
function CallSite(filename) {
  return Object.create({
    isNative: () => false,
    isConstructor: () => false,
    isToplevel: () => true,
    getFileName: () => filename,
    getLineNumber: () => undefined,
    getColumnNumber: () => undefined,
    getFunctionName: () => undefined,
    getMethodName: () => undefined,
    getTypeName: () => undefined,
    toString: () => filename
  });
}
function injectVirtualStackFrame(error, filename) {
  if (!SUPPORTED) return;
  let frames = virtualFrames.get(error);
  if (!frames) virtualFrames.set(error, frames = []);
  frames.push(CallSite(filename));
  return error;
}
function expectedError(error) {
  if (!SUPPORTED) return;
  expectedErrors.add(error);
  return error;
}
function beginHiddenCallStack(fn) {
  if (!SUPPORTED) return fn;
  return Object.defineProperty(function (...args) {
    setupPrepareStackTrace();
    return fn(...args);
  }, "name", {
    value: STOP_HIDING
  });
}
function endHiddenCallStack(fn) {
  if (!SUPPORTED) return fn;
  return Object.defineProperty(function (...args) {
    return fn(...args);
  }, "name", {
    value: START_HIDING
  });
}
function setupPrepareStackTrace() {
  setupPrepareStackTrace = () => {};
  const {
    prepareStackTrace = defaultPrepareStackTrace
  } = Error;
  const MIN_STACK_TRACE_LIMIT = 50;
  Error.stackTraceLimit && (Error.stackTraceLimit = Math.max(Error.stackTraceLimit, MIN_STACK_TRACE_LIMIT));
  Error.prepareStackTrace = function stackTraceRewriter(err, trace) {
    let newTrace = [];
    const isExpected = expectedErrors.has(err);
    let status = isExpected ? "hiding" : "unknown";
    for (let i = 0; i < trace.length; i++) {
      const name = trace[i].getFunctionName();
      if (name === START_HIDING) {
        status = "hiding";
      } else if (name === STOP_HIDING) {
        if (status === "hiding") {
          status = "showing";
          if (virtualFrames.has(err)) {
            newTrace.unshift(...virtualFrames.get(err));
          }
        } else if (status === "unknown") {
          newTrace = trace;
          break;
        }
      } else if (status !== "hiding") {
        newTrace.push(trace[i]);
      }
    }
    return prepareStackTrace(err, newTrace);
  };
}
function defaultPrepareStackTrace(err, trace) {
  if (trace.length === 0) return ErrorToString(err);
  return `${ErrorToString(err)}\n    at ${trace.join("\n    at ")}`;
}
0 && 0;

//# sourceMappingURL=rewrite-stack-trace.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614380, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ROOT_CONFIG_FILENAMES = void 0;
exports.findConfigUpwards = findConfigUpwards;
exports.findRelativeConfig = findRelativeConfig;
exports.findRootConfig = findRootConfig;
exports.loadConfig = loadConfig;
exports.resolveShowConfigPath = resolveShowConfigPath;
function _debug() {
  const data = require("debug");
  _debug = function () {
    return data;
  };
  return data;
}
function _fs() {
  const data = require("fs");
  _fs = function () {
    return data;
  };
  return data;
}
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
function _json() {
  const data = require("json5");
  _json = function () {
    return data;
  };
  return data;
}
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
var _caching = require("../caching.js");
var _configApi = require("../helpers/config-api.js");
var _utils = require("./utils.js");
var _moduleTypes = require("./module-types.js");
var _patternToRegex = require("../pattern-to-regex.js");
var _configError = require("../../errors/config-error.js");
var fs = require("../../gensync-utils/fs.js");
var _rewriteStackTrace = require("../../errors/rewrite-stack-trace.js");
const debug = _debug()("babel:config:loading:files:configuration");
const ROOT_CONFIG_FILENAMES = exports.ROOT_CONFIG_FILENAMES = ["babel.config.js", "babel.config.cjs", "babel.config.mjs", "babel.config.json", "babel.config.cts"];
const RELATIVE_CONFIG_FILENAMES = [".babelrc", ".babelrc.js", ".babelrc.cjs", ".babelrc.mjs", ".babelrc.json", ".babelrc.cts"];
const BABELIGNORE_FILENAME = ".babelignore";
const runConfig = (0, _caching.makeWeakCache)(function* runConfig(options, cache) {
  yield* [];
  return {
    options: (0, _rewriteStackTrace.endHiddenCallStack)(options)((0, _configApi.makeConfigAPI)(cache)),
    cacheNeedsConfiguration: !cache.configured()
  };
});
function* readConfigCode(filepath, data) {
  if (!_fs().existsSync(filepath)) return null;
  let options = yield* (0, _moduleTypes.default)(filepath, "You appear to be using a native ECMAScript module configuration " + "file, which is only supported when running Babel asynchronously.");
  let cacheNeedsConfiguration = false;
  if (typeof options === "function") {
    ({
      options,
      cacheNeedsConfiguration
    } = yield* runConfig(options, data));
  }
  if (!options || typeof options !== "object" || Array.isArray(options)) {
    throw new _configError.default(`Configuration should be an exported JavaScript object.`, filepath);
  }
  if (typeof options.then === "function") {
    options.catch == null || options.catch(() => {});
    throw new _configError.default(`You appear to be using an async configuration, ` + `which your current version of Babel does not support. ` + `We may add support for this in the future, ` + `but if you're on the most recent version of @babel/core and still ` + `seeing this error, then you'll need to synchronously return your config.`, filepath);
  }
  if (cacheNeedsConfiguration) throwConfigError(filepath);
  return buildConfigFileObject(options, filepath);
}
const cfboaf = new WeakMap();
function buildConfigFileObject(options, filepath) {
  let configFilesByFilepath = cfboaf.get(options);
  if (!configFilesByFilepath) {
    cfboaf.set(options, configFilesByFilepath = new Map());
  }
  let configFile = configFilesByFilepath.get(filepath);
  if (!configFile) {
    configFile = {
      filepath,
      dirname: _path().dirname(filepath),
      options
    };
    configFilesByFilepath.set(filepath, configFile);
  }
  return configFile;
}
const packageToBabelConfig = (0, _caching.makeWeakCacheSync)(file => {
  const babel = file.options["babel"];
  if (typeof babel === "undefined") return null;
  if (typeof babel !== "object" || Array.isArray(babel) || babel === null) {
    throw new _configError.default(`.babel property must be an object`, file.filepath);
  }
  return {
    filepath: file.filepath,
    dirname: file.dirname,
    options: babel
  };
});
const readConfigJSON5 = (0, _utils.makeStaticFileCache)((filepath, content) => {
  let options;
  try {
    options = _json().parse(content);
  } catch (err) {
    throw new _configError.default(`Error while parsing config - ${err.message}`, filepath);
  }
  if (!options) throw new _configError.default(`No config detected`, filepath);
  if (typeof options !== "object") {
    throw new _configError.default(`Config returned typeof ${typeof options}`, filepath);
  }
  if (Array.isArray(options)) {
    throw new _configError.default(`Expected config object but found array`, filepath);
  }
  delete options["$schema"];
  return {
    filepath,
    dirname: _path().dirname(filepath),
    options
  };
});
const readIgnoreConfig = (0, _utils.makeStaticFileCache)((filepath, content) => {
  const ignoreDir = _path().dirname(filepath);
  const ignorePatterns = content.split("\n").map(line => line.replace(/#.*$/, "").trim()).filter(line => !!line);
  for (const pattern of ignorePatterns) {
    if (pattern[0] === "!") {
      throw new _configError.default(`Negation of file paths is not supported.`, filepath);
    }
  }
  return {
    filepath,
    dirname: _path().dirname(filepath),
    ignore: ignorePatterns.map(pattern => (0, _patternToRegex.default)(pattern, ignoreDir))
  };
});
function findConfigUpwards(rootDir) {
  let dirname = rootDir;
  for (;;) {
    for (const filename of ROOT_CONFIG_FILENAMES) {
      if (_fs().existsSync(_path().join(dirname, filename))) {
        return dirname;
      }
    }
    const nextDir = _path().dirname(dirname);
    if (dirname === nextDir) break;
    dirname = nextDir;
  }
  return null;
}
function* findRelativeConfig(packageData, envName, caller) {
  let config = null;
  let ignore = null;
  const dirname = _path().dirname(packageData.filepath);
  for (const loc of packageData.directories) {
    if (!config) {
      var _packageData$pkg;
      config = yield* loadOneConfig(RELATIVE_CONFIG_FILENAMES, loc, envName, caller, ((_packageData$pkg = packageData.pkg) == null ? void 0 : _packageData$pkg.dirname) === loc ? packageToBabelConfig(packageData.pkg) : null);
    }
    if (!ignore) {
      const ignoreLoc = _path().join(loc, BABELIGNORE_FILENAME);
      ignore = yield* readIgnoreConfig(ignoreLoc);
      if (ignore) {
        debug("Found ignore %o from %o.", ignore.filepath, dirname);
      }
    }
  }
  return {
    config,
    ignore
  };
}
function findRootConfig(dirname, envName, caller) {
  return loadOneConfig(ROOT_CONFIG_FILENAMES, dirname, envName, caller);
}
function* loadOneConfig(names, dirname, envName, caller, previousConfig = null) {
  const configs = yield* _gensync().all(names.map(filename => readConfig(_path().join(dirname, filename), envName, caller)));
  const config = configs.reduce((previousConfig, config) => {
    if (config && previousConfig) {
      throw new _configError.default(`Multiple configuration files found. Please remove one:\n` + ` - ${_path().basename(previousConfig.filepath)}\n` + ` - ${config.filepath}\n` + `from ${dirname}`);
    }
    return config || previousConfig;
  }, previousConfig);
  if (config) {
    debug("Found configuration %o from %o.", config.filepath, dirname);
  }
  return config;
}
function* loadConfig(name, dirname, envName, caller) {
  const filepath = (((v, w) => (v = v.split("."), w = w.split("."), +v[0] > +w[0] || v[0] == w[0] && +v[1] >= +w[1]))(process.versions.node, "8.9") ? require.resolve : (r, {
    paths: [b]
  }, M = require("module")) => {
    let f = M._findPath(r, M._nodeModulePaths(b).concat(b));
    if (f) return f;
    f = new Error(`Cannot resolve module '${r}'`);
    f.code = "MODULE_NOT_FOUND";
    throw f;
  })(name, {
    paths: [dirname]
  });
  const conf = yield* readConfig(filepath, envName, caller);
  if (!conf) {
    throw new _configError.default(`Config file contains no configuration data`, filepath);
  }
  debug("Loaded config %o from %o.", name, dirname);
  return conf;
}
function readConfig(filepath, envName, caller) {
  const ext = _path().extname(filepath);
  switch (ext) {
    case ".js":
    case ".cjs":
    case ".mjs":
    case ".cts":
      return readConfigCode(filepath, {
        envName,
        caller
      });
    default:
      return readConfigJSON5(filepath);
  }
}
function* resolveShowConfigPath(dirname) {
  const targetPath = process.env.BABEL_SHOW_CONFIG_FOR;
  if (targetPath != null) {
    const absolutePath = _path().resolve(dirname, targetPath);
    const stats = yield* fs.stat(absolutePath);
    if (!stats.isFile()) {
      throw new Error(`${absolutePath}: BABEL_SHOW_CONFIG_FOR must refer to a regular file, directories are not supported.`);
    }
    return absolutePath;
  }
  return null;
}
function throwConfigError(filepath) {
  throw new _configError.default(`\
Caching was left unconfigured. Babel's plugins, presets, and .babelrc.js files can be configured
for various types of caching, using the first param of their handler functions:

module.exports = function(api) {
  // The API exposes the following:

  // Cache the returned value forever and don't call this function again.
  api.cache(true);

  // Don't cache at all. Not recommended because it will be very slow.
  api.cache(false);

  // Cached based on the value of some function. If this function returns a value different from
  // a previously-encountered value, the plugins will re-evaluate.
  var env = api.cache(() => process.env.NODE_ENV);

  // If testing for a specific env, we recommend specifics to avoid instantiating a plugin for
  // any possible NODE_ENV value that might come up during plugin execution.
  var isProd = api.cache(() => process.env.NODE_ENV === "production");

  // .cache(fn) will perform a linear search though instances to find the matching plugin based
  // based on previous instantiated plugins. If you want to recreate the plugin and discard the
  // previous instance whenever something changes, you may use:
  var isProd = api.cache.invalidate(() => process.env.NODE_ENV === "production");

  // Note, we also expose the following more-verbose versions of the above examples:
  api.cache.forever(); // api.cache(true)
  api.cache.never();   // api.cache(false)
  api.cache.using(fn); // api.cache(fn)

  // Return the value that will be cached.
  return { };
};`, filepath);
}
0 && 0;

//# sourceMappingURL=configuration.js.map

}, function(modId) { var map = {"../caching.js":1781530614374,"../helpers/config-api.js":1781530614381,"./utils.js":1781530614373,"./module-types.js":1781530614382,"../pattern-to-regex.js":1781530614396,"../../errors/config-error.js":1781530614378,"../../gensync-utils/fs.js":1781530614377,"../../errors/rewrite-stack-trace.js":1781530614379}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614381, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.makeConfigAPI = makeConfigAPI;
exports.makePluginAPI = makePluginAPI;
exports.makePresetAPI = makePresetAPI;
function _semver() {
  const data = require("semver");
  _semver = function () {
    return data;
  };
  return data;
}
var _index = require("../../index.js");
var _caching = require("../caching.js");
function makeConfigAPI(cache) {
  const env = value => cache.using(data => {
    if (typeof value === "undefined") return data.envName;
    if (typeof value === "function") {
      return (0, _caching.assertSimpleType)(value(data.envName));
    }
    return (Array.isArray(value) ? value : [value]).some(entry => {
      if (typeof entry !== "string") {
        throw new Error("Unexpected non-string value");
      }
      return entry === data.envName;
    });
  });
  const caller = cb => cache.using(data => (0, _caching.assertSimpleType)(cb(data.caller)));
  return {
    version: _index.version,
    cache: cache.simple(),
    env,
    async: () => false,
    caller,
    assertVersion
  };
}
function makePresetAPI(cache, externalDependencies) {
  const targets = () => JSON.parse(cache.using(data => JSON.stringify(data.targets)));
  const addExternalDependency = ref => {
    externalDependencies.push(ref);
  };
  return Object.assign({}, makeConfigAPI(cache), {
    targets,
    addExternalDependency
  });
}
function makePluginAPI(cache, externalDependencies) {
  const assumption = name => cache.using(data => data.assumptions[name]);
  return Object.assign({}, makePresetAPI(cache, externalDependencies), {
    assumption
  });
}
function assertVersion(range) {
  if (typeof range === "number") {
    if (!Number.isInteger(range)) {
      throw new Error("Expected string or integer value.");
    }
    range = `^${range}.0.0-0`;
  }
  if (typeof range !== "string") {
    throw new Error("Expected string or integer value.");
  }
  if (range === "*" || _semver().satisfies(_index.version, range)) return;
  const limit = Error.stackTraceLimit;
  if (typeof limit === "number" && limit < 25) {
    Error.stackTraceLimit = 25;
  }
  const err = new Error(`Requires Babel "${range}", but was loaded with "${_index.version}". ` + `If you are sure you have a compatible version of @babel/core, ` + `it is likely that something in your build process is loading the ` + `wrong version. Inspect the stack trace of this error to look for ` + `the first entry that doesn't mention "@babel/core" or "babel-core" ` + `to see what is calling Babel.`);
  if (typeof limit === "number") {
    Error.stackTraceLimit = limit;
  }
  throw Object.assign(err, {
    code: "BABEL_VERSION_UNSUPPORTED",
    version: _index.version,
    range
  });
}
0 && 0;

//# sourceMappingURL=config-api.js.map

}, function(modId) { var map = {"../../index.js":1781530614368,"../caching.js":1781530614374}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614382, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = loadCodeDefault;
exports.supportsESM = void 0;
var _async = require("../../gensync-utils/async.js");
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
function _url() {
  const data = require("url");
  _url = function () {
    return data;
  };
  return data;
}
function _semver() {
  const data = require("semver");
  _semver = function () {
    return data;
  };
  return data;
}
function _debug() {
  const data = require("debug");
  _debug = function () {
    return data;
  };
  return data;
}
var _rewriteStackTrace = require("../../errors/rewrite-stack-trace.js");
var _configError = require("../../errors/config-error.js");
var _transformFile = require("../../transform-file.js");
function asyncGeneratorStep(gen, resolve, reject, _next, _throw, key, arg) { try { var info = gen[key](arg); var value = info.value; } catch (error) { reject(error); return; } if (info.done) { resolve(value); } else { Promise.resolve(value).then(_next, _throw); } }
function _asyncToGenerator(fn) { return function () { var self = this, args = arguments; return new Promise(function (resolve, reject) { var gen = fn.apply(self, args); function _next(value) { asyncGeneratorStep(gen, resolve, reject, _next, _throw, "next", value); } function _throw(err) { asyncGeneratorStep(gen, resolve, reject, _next, _throw, "throw", err); } _next(undefined); }); }; }
const debug = _debug()("babel:config:loading:files:module-types");
{
  try {
    var import_ = require("./import.cjs");
  } catch (_unused) {}
}
const supportsESM = exports.supportsESM = _semver().satisfies(process.versions.node, "^12.17 || >=13.2");
const LOADING_CJS_FILES = new Set();
function loadCjsDefault(filepath) {
  if (LOADING_CJS_FILES.has(filepath)) {
    debug("Auto-ignoring usage of config %o.", filepath);
    return {};
  }
  let module;
  try {
    LOADING_CJS_FILES.add(filepath);
    module = (0, _rewriteStackTrace.endHiddenCallStack)(require)(filepath);
  } finally {
    LOADING_CJS_FILES.delete(filepath);
  }
  {
    var _module;
    return (_module = module) != null && _module.__esModule ? module.default || (arguments[1] ? module : undefined) : module;
  }
}
const loadMjsDefault = (0, _rewriteStackTrace.endHiddenCallStack)(function () {
  var _loadMjsDefault = _asyncToGenerator(function* (filepath) {
    const url = (0, _url().pathToFileURL)(filepath).toString();
    {
      if (!import_) {
        throw new _configError.default("Internal error: Native ECMAScript modules aren't supported by this platform.\n", filepath);
      }
      return (yield import_(url)).default;
    }
  });
  function loadMjsDefault(_x) {
    return _loadMjsDefault.apply(this, arguments);
  }
  return loadMjsDefault;
}());
function* loadCodeDefault(filepath, asyncError) {
  switch (_path().extname(filepath)) {
    case ".cjs":
      {
        return loadCjsDefault(filepath, arguments[2]);
      }
    case ".mjs":
      break;
    case ".cts":
      return loadCtsDefault(filepath);
    default:
      try {
        {
          return loadCjsDefault(filepath, arguments[2]);
        }
      } catch (e) {
        if (e.code !== "ERR_REQUIRE_ESM") throw e;
      }
  }
  if (yield* (0, _async.isAsync)()) {
    return yield* (0, _async.waitFor)(loadMjsDefault(filepath));
  }
  throw new _configError.default(asyncError, filepath);
}
function loadCtsDefault(filepath) {
  const ext = ".cts";
  const hasTsSupport = !!(require.extensions[".ts"] || require.extensions[".cts"] || require.extensions[".mts"]);
  let handler;
  if (!hasTsSupport) {
    const opts = {
      babelrc: false,
      configFile: false,
      sourceType: "unambiguous",
      sourceMaps: "inline",
      sourceFileName: _path().basename(filepath),
      presets: [[getTSPreset(filepath), Object.assign({
        onlyRemoveTypeImports: true,
        optimizeConstEnums: true
      }, {
        allowDeclareFields: true
      })]]
    };
    handler = function (m, filename) {
      if (handler && filename.endsWith(ext)) {
        try {
          return m._compile((0, _transformFile.transformFileSync)(filename, Object.assign({}, opts, {
            filename
          })).code, filename);
        } catch (error) {
          if (!hasTsSupport) {
            const packageJson = require("@babel/preset-typescript/package.json");
            if (_semver().lt(packageJson.version, "7.21.4")) {
              console.error("`.cts` configuration file failed to load, please try to update `@babel/preset-typescript`.");
            }
          }
          throw error;
        }
      }
      return require.extensions[".js"](m, filename);
    };
    require.extensions[ext] = handler;
  }
  try {
    return loadCjsDefault(filepath);
  } finally {
    if (!hasTsSupport) {
      if (require.extensions[ext] === handler) delete require.extensions[ext];
      handler = undefined;
    }
  }
}
function getTSPreset(filepath) {
  try {
    return require("@babel/preset-typescript");
  } catch (error) {
    if (error.code !== "MODULE_NOT_FOUND") throw error;
    let message = "You appear to be using a .cts file as Babel configuration, but the `@babel/preset-typescript` package was not found: please install it!";
    {
      if (process.versions.pnp) {
        message += `
If you are using Yarn Plug'n'Play, you may also need to add the following configuration to your .yarnrc.yml file:

packageExtensions:
\t"@babel/core@*":
\t\tpeerDependencies:
\t\t\t"@babel/preset-typescript": "*"
`;
      }
    }
    throw new _configError.default(message, filepath);
  }
}
0 && 0;

//# sourceMappingURL=module-types.js.map

}, function(modId) { var map = {"../../gensync-utils/async.js":1781530614375,"../../errors/rewrite-stack-trace.js":1781530614379,"../../errors/config-error.js":1781530614378,"../../transform-file.js":1781530614383}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614383, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.transformFile = transformFile;
exports.transformFileAsync = transformFileAsync;
exports.transformFileSync = transformFileSync;
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
var _index = require("./config/index.js");
var _index2 = require("./transformation/index.js");
var fs = require("./gensync-utils/fs.js");
({});
const transformFileRunner = _gensync()(function* (filename, opts) {
  const options = Object.assign({}, opts, {
    filename
  });
  const config = yield* (0, _index.default)(options);
  if (config === null) return null;
  const code = yield* fs.readFile(filename, "utf8");
  return yield* (0, _index2.run)(config, code);
});
function transformFile(...args) {
  transformFileRunner.errback(...args);
}
function transformFileSync(...args) {
  return transformFileRunner.sync(...args);
}
function transformFileAsync(...args) {
  return transformFileRunner.async(...args);
}
0 && 0;

//# sourceMappingURL=transform-file.js.map

}, function(modId) { var map = {"./config/index.js":1781530614384,"./transformation/index.js":1781530614401,"./gensync-utils/fs.js":1781530614377}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614384, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.createConfigItem = createConfigItem;
exports.createConfigItemAsync = createConfigItemAsync;
exports.createConfigItemSync = createConfigItemSync;
Object.defineProperty(exports, "default", {
  enumerable: true,
  get: function () {
    return _full.default;
  }
});
exports.loadOptions = loadOptions;
exports.loadOptionsAsync = loadOptionsAsync;
exports.loadOptionsSync = loadOptionsSync;
exports.loadPartialConfig = loadPartialConfig;
exports.loadPartialConfigAsync = loadPartialConfigAsync;
exports.loadPartialConfigSync = loadPartialConfigSync;
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
var _full = require("./full.js");
var _partial = require("./partial.js");
var _item = require("./item.js");
var _rewriteStackTrace = require("../errors/rewrite-stack-trace.js");
const loadPartialConfigRunner = _gensync()(_partial.loadPartialConfig);
function loadPartialConfigAsync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(loadPartialConfigRunner.async)(...args);
}
function loadPartialConfigSync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(loadPartialConfigRunner.sync)(...args);
}
function loadPartialConfig(opts, callback) {
  if (callback !== undefined) {
    (0, _rewriteStackTrace.beginHiddenCallStack)(loadPartialConfigRunner.errback)(opts, callback);
  } else if (typeof opts === "function") {
    (0, _rewriteStackTrace.beginHiddenCallStack)(loadPartialConfigRunner.errback)(undefined, opts);
  } else {
    {
      return loadPartialConfigSync(opts);
    }
  }
}
function* loadOptionsImpl(opts) {
  var _config$options;
  const config = yield* (0, _full.default)(opts);
  return (_config$options = config == null ? void 0 : config.options) != null ? _config$options : null;
}
const loadOptionsRunner = _gensync()(loadOptionsImpl);
function loadOptionsAsync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(loadOptionsRunner.async)(...args);
}
function loadOptionsSync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(loadOptionsRunner.sync)(...args);
}
function loadOptions(opts, callback) {
  if (callback !== undefined) {
    (0, _rewriteStackTrace.beginHiddenCallStack)(loadOptionsRunner.errback)(opts, callback);
  } else if (typeof opts === "function") {
    (0, _rewriteStackTrace.beginHiddenCallStack)(loadOptionsRunner.errback)(undefined, opts);
  } else {
    {
      return loadOptionsSync(opts);
    }
  }
}
const createConfigItemRunner = _gensync()(_item.createConfigItem);
function createConfigItemAsync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(createConfigItemRunner.async)(...args);
}
function createConfigItemSync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(createConfigItemRunner.sync)(...args);
}
function createConfigItem(target, options, callback) {
  if (callback !== undefined) {
    (0, _rewriteStackTrace.beginHiddenCallStack)(createConfigItemRunner.errback)(target, options, callback);
  } else if (typeof options === "function") {
    (0, _rewriteStackTrace.beginHiddenCallStack)(createConfigItemRunner.errback)(target, undefined, callback);
  } else {
    {
      return createConfigItemSync(target, options);
    }
  }
}
0 && 0;

//# sourceMappingURL=index.js.map

}, function(modId) { var map = {"./full.js":1781530614385,"./partial.js":1781530614399,"./item.js":1781530614388,"../errors/rewrite-stack-trace.js":1781530614379}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614385, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
var _async = require("../gensync-utils/async.js");
var _util = require("./util.js");
var context = require("../index.js");
var _plugin = require("./plugin.js");
var _item = require("./item.js");
var _configChain = require("./config-chain.js");
var _deepArray = require("./helpers/deep-array.js");
function _traverse() {
  const data = require("@babel/traverse");
  _traverse = function () {
    return data;
  };
  return data;
}
var _caching = require("./caching.js");
var _options = require("./validation/options.js");
var _plugins = require("./validation/plugins.js");
var _configApi = require("./helpers/config-api.js");
var _partial = require("./partial.js");
var _configError = require("../errors/config-error.js");
var _default = exports.default = _gensync()(function* loadFullConfig(inputOpts) {
  var _opts$assumptions;
  const result = yield* (0, _partial.default)(inputOpts);
  if (!result) {
    return null;
  }
  const {
    options,
    context,
    fileHandling
  } = result;
  if (fileHandling === "ignored") {
    return null;
  }
  const optionDefaults = {};
  const {
    plugins,
    presets
  } = options;
  if (!plugins || !presets) {
    throw new Error("Assertion failure - plugins and presets exist");
  }
  const presetContext = Object.assign({}, context, {
    targets: options.targets
  });
  const toDescriptor = item => {
    const desc = (0, _item.getItemDescriptor)(item);
    if (!desc) {
      throw new Error("Assertion failure - must be config item");
    }
    return desc;
  };
  const presetsDescriptors = presets.map(toDescriptor);
  const initialPluginsDescriptors = plugins.map(toDescriptor);
  const pluginDescriptorsByPass = [[]];
  const passes = [];
  const externalDependencies = [];
  const ignored = yield* enhanceError(context, function* recursePresetDescriptors(rawPresets, pluginDescriptorsPass) {
    const presets = [];
    for (let i = 0; i < rawPresets.length; i++) {
      const descriptor = rawPresets[i];
      if (descriptor.options !== false) {
        try {
          var preset = yield* loadPresetDescriptor(descriptor, presetContext);
        } catch (e) {
          if (e.code === "BABEL_UNKNOWN_OPTION") {
            (0, _options.checkNoUnwrappedItemOptionPairs)(rawPresets, i, "preset", e);
          }
          throw e;
        }
        externalDependencies.push(preset.externalDependencies);
        if (descriptor.ownPass) {
          presets.push({
            preset: preset.chain,
            pass: []
          });
        } else {
          presets.unshift({
            preset: preset.chain,
            pass: pluginDescriptorsPass
          });
        }
      }
    }
    if (presets.length > 0) {
      pluginDescriptorsByPass.splice(1, 0, ...presets.map(o => o.pass).filter(p => p !== pluginDescriptorsPass));
      for (const {
        preset,
        pass
      } of presets) {
        if (!preset) return true;
        pass.push(...preset.plugins);
        const ignored = yield* recursePresetDescriptors(preset.presets, pass);
        if (ignored) return true;
        preset.options.forEach(opts => {
          (0, _util.mergeOptions)(optionDefaults, opts);
        });
      }
    }
  })(presetsDescriptors, pluginDescriptorsByPass[0]);
  if (ignored) return null;
  const opts = optionDefaults;
  (0, _util.mergeOptions)(opts, options);
  const pluginContext = Object.assign({}, presetContext, {
    assumptions: (_opts$assumptions = opts.assumptions) != null ? _opts$assumptions : {}
  });
  yield* enhanceError(context, function* loadPluginDescriptors() {
    pluginDescriptorsByPass[0].unshift(...initialPluginsDescriptors);
    for (const descs of pluginDescriptorsByPass) {
      const pass = [];
      passes.push(pass);
      for (let i = 0; i < descs.length; i++) {
        const descriptor = descs[i];
        if (descriptor.options !== false) {
          try {
            var plugin = yield* loadPluginDescriptor(descriptor, pluginContext);
          } catch (e) {
            if (e.code === "BABEL_UNKNOWN_PLUGIN_PROPERTY") {
              (0, _options.checkNoUnwrappedItemOptionPairs)(descs, i, "plugin", e);
            }
            throw e;
          }
          pass.push(plugin);
          externalDependencies.push(plugin.externalDependencies);
        }
      }
    }
  })();
  opts.plugins = passes[0];
  opts.presets = passes.slice(1).filter(plugins => plugins.length > 0).map(plugins => ({
    plugins
  }));
  opts.passPerPreset = opts.presets.length > 0;
  return {
    options: opts,
    passes: passes,
    externalDependencies: (0, _deepArray.finalize)(externalDependencies)
  };
});
function enhanceError(context, fn) {
  return function* (arg1, arg2) {
    try {
      return yield* fn(arg1, arg2);
    } catch (e) {
      if (!/^\[BABEL\]/.test(e.message)) {
        var _context$filename;
        e.message = `[BABEL] ${(_context$filename = context.filename) != null ? _context$filename : "unknown file"}: ${e.message}`;
      }
      throw e;
    }
  };
}
const makeDescriptorLoader = apiFactory => (0, _caching.makeWeakCache)(function* ({
  value,
  options,
  dirname,
  alias
}, cache) {
  if (options === false) throw new Error("Assertion failure");
  options = options || {};
  const externalDependencies = [];
  let item = value;
  if (typeof value === "function") {
    const factory = (0, _async.maybeAsync)(value, `You appear to be using an async plugin/preset, but Babel has been called synchronously`);
    const api = Object.assign({}, context, apiFactory(cache, externalDependencies));
    try {
      item = yield* factory(api, options, dirname);
    } catch (e) {
      if (alias) {
        e.message += ` (While processing: ${JSON.stringify(alias)})`;
      }
      throw e;
    }
  }
  if (!item || typeof item !== "object") {
    throw new Error("Plugin/Preset did not return an object.");
  }
  if ((0, _async.isThenable)(item)) {
    yield* [];
    throw new Error(`You appear to be using a promise as a plugin, ` + `which your current version of Babel does not support. ` + `If you're using a published plugin, ` + `you may need to upgrade your @babel/core version. ` + `As an alternative, you can prefix the promise with "await". ` + `(While processing: ${JSON.stringify(alias)})`);
  }
  if (externalDependencies.length > 0 && (!cache.configured() || cache.mode() === "forever")) {
    let error = `A plugin/preset has external untracked dependencies ` + `(${externalDependencies[0]}), but the cache `;
    if (!cache.configured()) {
      error += `has not been configured to be invalidated when the external dependencies change. `;
    } else {
      error += ` has been configured to never be invalidated. `;
    }
    error += `Plugins/presets should configure their cache to be invalidated when the external ` + `dependencies change, for example using \`api.cache.invalidate(() => ` + `statSync(filepath).mtimeMs)\` or \`api.cache.never()\`\n` + `(While processing: ${JSON.stringify(alias)})`;
    throw new Error(error);
  }
  return {
    value: item,
    options,
    dirname,
    alias,
    externalDependencies: (0, _deepArray.finalize)(externalDependencies)
  };
});
const pluginDescriptorLoader = makeDescriptorLoader(_configApi.makePluginAPI);
const presetDescriptorLoader = makeDescriptorLoader(_configApi.makePresetAPI);
const instantiatePlugin = (0, _caching.makeWeakCache)(function* ({
  value,
  options,
  dirname,
  alias,
  externalDependencies
}, cache) {
  const pluginObj = (0, _plugins.validatePluginObject)(value);
  const plugin = Object.assign({}, pluginObj);
  if (plugin.visitor) {
    plugin.visitor = _traverse().default.explode(Object.assign({}, plugin.visitor));
  }
  if (plugin.inherits) {
    const inheritsDescriptor = {
      name: undefined,
      alias: `${alias}$inherits`,
      value: plugin.inherits,
      options,
      dirname
    };
    const inherits = yield* (0, _async.forwardAsync)(loadPluginDescriptor, run => {
      return cache.invalidate(data => run(inheritsDescriptor, data));
    });
    plugin.pre = chain(inherits.pre, plugin.pre);
    plugin.post = chain(inherits.post, plugin.post);
    plugin.manipulateOptions = chain(inherits.manipulateOptions, plugin.manipulateOptions);
    plugin.visitor = _traverse().default.visitors.merge([inherits.visitor || {}, plugin.visitor || {}]);
    if (inherits.externalDependencies.length > 0) {
      if (externalDependencies.length === 0) {
        externalDependencies = inherits.externalDependencies;
      } else {
        externalDependencies = (0, _deepArray.finalize)([externalDependencies, inherits.externalDependencies]);
      }
    }
  }
  return new _plugin.default(plugin, options, alias, externalDependencies);
});
function* loadPluginDescriptor(descriptor, context) {
  if (descriptor.value instanceof _plugin.default) {
    if (descriptor.options) {
      throw new Error("Passed options to an existing Plugin instance will not work.");
    }
    return descriptor.value;
  }
  return yield* instantiatePlugin(yield* pluginDescriptorLoader(descriptor, context), context);
}
const needsFilename = val => val && typeof val !== "function";
const validateIfOptionNeedsFilename = (options, descriptor) => {
  if (needsFilename(options.test) || needsFilename(options.include) || needsFilename(options.exclude)) {
    const formattedPresetName = descriptor.name ? `"${descriptor.name}"` : "/* your preset */";
    throw new _configError.default([`Preset ${formattedPresetName} requires a filename to be set when babel is called directly,`, `\`\`\``, `babel.transformSync(code, { filename: 'file.ts', presets: [${formattedPresetName}] });`, `\`\`\``, `See https://babeljs.io/docs/en/options#filename for more information.`].join("\n"));
  }
};
const validatePreset = (preset, context, descriptor) => {
  if (!context.filename) {
    var _options$overrides;
    const {
      options
    } = preset;
    validateIfOptionNeedsFilename(options, descriptor);
    (_options$overrides = options.overrides) == null || _options$overrides.forEach(overrideOptions => validateIfOptionNeedsFilename(overrideOptions, descriptor));
  }
};
const instantiatePreset = (0, _caching.makeWeakCacheSync)(({
  value,
  dirname,
  alias,
  externalDependencies
}) => {
  return {
    options: (0, _options.validate)("preset", value),
    alias,
    dirname,
    externalDependencies
  };
});
function* loadPresetDescriptor(descriptor, context) {
  const preset = instantiatePreset(yield* presetDescriptorLoader(descriptor, context));
  validatePreset(preset, context, descriptor);
  return {
    chain: yield* (0, _configChain.buildPresetChain)(preset, context),
    externalDependencies: preset.externalDependencies
  };
}
function chain(a, b) {
  const fns = [a, b].filter(Boolean);
  if (fns.length <= 1) return fns[0];
  return function (...args) {
    for (const fn of fns) {
      fn.apply(this, args);
    }
  };
}
0 && 0;

//# sourceMappingURL=full.js.map

}, function(modId) { var map = {"../gensync-utils/async.js":1781530614375,"./util.js":1781530614376,"../index.js":1781530614368,"./plugin.js":1781530614386,"./item.js":1781530614388,"./config-chain.js":1781530614392,"./helpers/deep-array.js":1781530614387,"./caching.js":1781530614374,"./validation/options.js":1781530614393,"./validation/plugins.js":1781530614398,"./helpers/config-api.js":1781530614381,"./partial.js":1781530614399,"../errors/config-error.js":1781530614378}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614386, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
var _deepArray = require("./helpers/deep-array.js");
class Plugin {
  constructor(plugin, options, key, externalDependencies = (0, _deepArray.finalize)([])) {
    this.key = void 0;
    this.manipulateOptions = void 0;
    this.post = void 0;
    this.pre = void 0;
    this.visitor = void 0;
    this.parserOverride = void 0;
    this.generatorOverride = void 0;
    this.options = void 0;
    this.externalDependencies = void 0;
    this.key = plugin.name || key;
    this.manipulateOptions = plugin.manipulateOptions;
    this.post = plugin.post;
    this.pre = plugin.pre;
    this.visitor = plugin.visitor || {};
    this.parserOverride = plugin.parserOverride;
    this.generatorOverride = plugin.generatorOverride;
    this.options = options;
    this.externalDependencies = externalDependencies;
  }
}
exports.default = Plugin;
0 && 0;

//# sourceMappingURL=plugin.js.map

}, function(modId) { var map = {"./helpers/deep-array.js":1781530614387}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614387, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.finalize = finalize;
exports.flattenToSet = flattenToSet;
function finalize(deepArr) {
  return Object.freeze(deepArr);
}
function flattenToSet(arr) {
  const result = new Set();
  const stack = [arr];
  while (stack.length > 0) {
    for (const el of stack.pop()) {
      if (Array.isArray(el)) stack.push(el);else result.add(el);
    }
  }
  return result;
}
0 && 0;

//# sourceMappingURL=deep-array.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614388, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.createConfigItem = createConfigItem;
exports.createItemFromDescriptor = createItemFromDescriptor;
exports.getItemDescriptor = getItemDescriptor;
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
var _configDescriptors = require("./config-descriptors.js");
function createItemFromDescriptor(desc) {
  return new ConfigItem(desc);
}
function* createConfigItem(value, {
  dirname = ".",
  type
} = {}) {
  const descriptor = yield* (0, _configDescriptors.createDescriptor)(value, _path().resolve(dirname), {
    type,
    alias: "programmatic item"
  });
  return createItemFromDescriptor(descriptor);
}
const CONFIG_ITEM_BRAND = Symbol.for("@babel/core@7 - ConfigItem");
function getItemDescriptor(item) {
  if (item != null && item[CONFIG_ITEM_BRAND]) {
    return item._descriptor;
  }
  return undefined;
}
class ConfigItem {
  constructor(descriptor) {
    this._descriptor = void 0;
    this[CONFIG_ITEM_BRAND] = true;
    this.value = void 0;
    this.options = void 0;
    this.dirname = void 0;
    this.name = void 0;
    this.file = void 0;
    this._descriptor = descriptor;
    Object.defineProperty(this, "_descriptor", {
      enumerable: false
    });
    Object.defineProperty(this, CONFIG_ITEM_BRAND, {
      enumerable: false
    });
    this.value = this._descriptor.value;
    this.options = this._descriptor.options;
    this.dirname = this._descriptor.dirname;
    this.name = this._descriptor.name;
    this.file = this._descriptor.file ? {
      request: this._descriptor.file.request,
      resolved: this._descriptor.file.resolved
    } : undefined;
    Object.freeze(this);
  }
}
Object.freeze(ConfigItem.prototype);
0 && 0;

//# sourceMappingURL=item.js.map

}, function(modId) { var map = {"./config-descriptors.js":1781530614389}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614389, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.createCachedDescriptors = createCachedDescriptors;
exports.createDescriptor = createDescriptor;
exports.createUncachedDescriptors = createUncachedDescriptors;
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
var _functional = require("../gensync-utils/functional.js");
var _index = require("./files/index.js");
var _item = require("./item.js");
var _caching = require("./caching.js");
var _resolveTargets = require("./resolve-targets.js");
function isEqualDescriptor(a, b) {
  var _a$file, _b$file, _a$file2, _b$file2;
  return a.name === b.name && a.value === b.value && a.options === b.options && a.dirname === b.dirname && a.alias === b.alias && a.ownPass === b.ownPass && ((_a$file = a.file) == null ? void 0 : _a$file.request) === ((_b$file = b.file) == null ? void 0 : _b$file.request) && ((_a$file2 = a.file) == null ? void 0 : _a$file2.resolved) === ((_b$file2 = b.file) == null ? void 0 : _b$file2.resolved);
}
function* handlerOf(value) {
  return value;
}
function optionsWithResolvedBrowserslistConfigFile(options, dirname) {
  if (typeof options.browserslistConfigFile === "string") {
    options.browserslistConfigFile = (0, _resolveTargets.resolveBrowserslistConfigFile)(options.browserslistConfigFile, dirname);
  }
  return options;
}
function createCachedDescriptors(dirname, options, alias) {
  const {
    plugins,
    presets,
    passPerPreset
  } = options;
  return {
    options: optionsWithResolvedBrowserslistConfigFile(options, dirname),
    plugins: plugins ? () => createCachedPluginDescriptors(plugins, dirname)(alias) : () => handlerOf([]),
    presets: presets ? () => createCachedPresetDescriptors(presets, dirname)(alias)(!!passPerPreset) : () => handlerOf([])
  };
}
function createUncachedDescriptors(dirname, options, alias) {
  return {
    options: optionsWithResolvedBrowserslistConfigFile(options, dirname),
    plugins: (0, _functional.once)(() => createPluginDescriptors(options.plugins || [], dirname, alias)),
    presets: (0, _functional.once)(() => createPresetDescriptors(options.presets || [], dirname, alias, !!options.passPerPreset))
  };
}
const PRESET_DESCRIPTOR_CACHE = new WeakMap();
const createCachedPresetDescriptors = (0, _caching.makeWeakCacheSync)((items, cache) => {
  const dirname = cache.using(dir => dir);
  return (0, _caching.makeStrongCacheSync)(alias => (0, _caching.makeStrongCache)(function* (passPerPreset) {
    const descriptors = yield* createPresetDescriptors(items, dirname, alias, passPerPreset);
    return descriptors.map(desc => loadCachedDescriptor(PRESET_DESCRIPTOR_CACHE, desc));
  }));
});
const PLUGIN_DESCRIPTOR_CACHE = new WeakMap();
const createCachedPluginDescriptors = (0, _caching.makeWeakCacheSync)((items, cache) => {
  const dirname = cache.using(dir => dir);
  return (0, _caching.makeStrongCache)(function* (alias) {
    const descriptors = yield* createPluginDescriptors(items, dirname, alias);
    return descriptors.map(desc => loadCachedDescriptor(PLUGIN_DESCRIPTOR_CACHE, desc));
  });
});
const DEFAULT_OPTIONS = {};
function loadCachedDescriptor(cache, desc) {
  const {
    value,
    options = DEFAULT_OPTIONS
  } = desc;
  if (options === false) return desc;
  let cacheByOptions = cache.get(value);
  if (!cacheByOptions) {
    cacheByOptions = new WeakMap();
    cache.set(value, cacheByOptions);
  }
  let possibilities = cacheByOptions.get(options);
  if (!possibilities) {
    possibilities = [];
    cacheByOptions.set(options, possibilities);
  }
  if (!possibilities.includes(desc)) {
    const matches = possibilities.filter(possibility => isEqualDescriptor(possibility, desc));
    if (matches.length > 0) {
      return matches[0];
    }
    possibilities.push(desc);
  }
  return desc;
}
function* createPresetDescriptors(items, dirname, alias, passPerPreset) {
  return yield* createDescriptors("preset", items, dirname, alias, passPerPreset);
}
function* createPluginDescriptors(items, dirname, alias) {
  return yield* createDescriptors("plugin", items, dirname, alias);
}
function* createDescriptors(type, items, dirname, alias, ownPass) {
  const descriptors = yield* _gensync().all(items.map((item, index) => createDescriptor(item, dirname, {
    type,
    alias: `${alias}$${index}`,
    ownPass: !!ownPass
  })));
  assertNoDuplicates(descriptors);
  return descriptors;
}
function* createDescriptor(pair, dirname, {
  type,
  alias,
  ownPass
}) {
  const desc = (0, _item.getItemDescriptor)(pair);
  if (desc) {
    return desc;
  }
  let name;
  let options;
  let value = pair;
  if (Array.isArray(value)) {
    if (value.length === 3) {
      [value, options, name] = value;
    } else {
      [value, options] = value;
    }
  }
  let file = undefined;
  let filepath = null;
  if (typeof value === "string") {
    if (typeof type !== "string") {
      throw new Error("To resolve a string-based item, the type of item must be given");
    }
    const resolver = type === "plugin" ? _index.loadPlugin : _index.loadPreset;
    const request = value;
    ({
      filepath,
      value
    } = yield* resolver(value, dirname));
    file = {
      request,
      resolved: filepath
    };
  }
  if (!value) {
    throw new Error(`Unexpected falsy value: ${String(value)}`);
  }
  if (typeof value === "object" && value.__esModule) {
    if (value.default) {
      value = value.default;
    } else {
      throw new Error("Must export a default export when using ES6 modules.");
    }
  }
  if (typeof value !== "object" && typeof value !== "function") {
    throw new Error(`Unsupported format: ${typeof value}. Expected an object or a function.`);
  }
  if (filepath !== null && typeof value === "object" && value) {
    throw new Error(`Plugin/Preset files are not allowed to export objects, only functions. In ${filepath}`);
  }
  return {
    name,
    alias: filepath || alias,
    value,
    options,
    dirname,
    ownPass,
    file
  };
}
function assertNoDuplicates(items) {
  const map = new Map();
  for (const item of items) {
    if (typeof item.value !== "function") continue;
    let nameMap = map.get(item.value);
    if (!nameMap) {
      nameMap = new Set();
      map.set(item.value, nameMap);
    }
    if (nameMap.has(item.name)) {
      const conflicts = items.filter(i => i.value === item.value);
      throw new Error([`Duplicate plugin/preset detected.`, `If you'd like to use two separate instances of a plugin,`, `they need separate names, e.g.`, ``, `  plugins: [`, `    ['some-plugin', {}],`, `    ['some-plugin', {}, 'some unique name'],`, `  ]`, ``, `Duplicates detected are:`, `${JSON.stringify(conflicts, null, 2)}`].join("\n"));
    }
    nameMap.add(item.name);
  }
}
0 && 0;

//# sourceMappingURL=config-descriptors.js.map

}, function(modId) { var map = {"../gensync-utils/functional.js":1781530614390,"./files/index.js":1781530614371,"./item.js":1781530614388,"./caching.js":1781530614374,"./resolve-targets.js":1781530614391}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614390, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.once = once;
var _async = require("./async.js");
function once(fn) {
  let result;
  let resultP;
  let promiseReferenced = false;
  return function* () {
    if (!result) {
      if (resultP) {
        promiseReferenced = true;
        return yield* (0, _async.waitFor)(resultP);
      }
      if (!(yield* (0, _async.isAsync)())) {
        try {
          result = {
            ok: true,
            value: yield* fn()
          };
        } catch (error) {
          result = {
            ok: false,
            value: error
          };
        }
      } else {
        let resolve, reject;
        resultP = new Promise((res, rej) => {
          resolve = res;
          reject = rej;
        });
        try {
          result = {
            ok: true,
            value: yield* fn()
          };
          resultP = null;
          if (promiseReferenced) resolve(result.value);
        } catch (error) {
          result = {
            ok: false,
            value: error
          };
          resultP = null;
          if (promiseReferenced) reject(error);
        }
      }
    }
    if (result.ok) return result.value;else throw result.value;
  };
}
0 && 0;

//# sourceMappingURL=functional.js.map

}, function(modId) { var map = {"./async.js":1781530614375}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614391, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.resolveBrowserslistConfigFile = resolveBrowserslistConfigFile;
exports.resolveTargets = resolveTargets;
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
function _helperCompilationTargets() {
  const data = require("@babel/helper-compilation-targets");
  _helperCompilationTargets = function () {
    return data;
  };
  return data;
}
({});
function resolveBrowserslistConfigFile(browserslistConfigFile, configFileDir) {
  return _path().resolve(configFileDir, browserslistConfigFile);
}
function resolveTargets(options, root) {
  const optTargets = options.targets;
  let targets;
  if (typeof optTargets === "string" || Array.isArray(optTargets)) {
    targets = {
      browsers: optTargets
    };
  } else if (optTargets) {
    if ("esmodules" in optTargets) {
      targets = Object.assign({}, optTargets, {
        esmodules: "intersect"
      });
    } else {
      targets = optTargets;
    }
  }
  const {
    browserslistConfigFile
  } = options;
  let configFile;
  let ignoreBrowserslistConfig = false;
  if (typeof browserslistConfigFile === "string") {
    configFile = browserslistConfigFile;
  } else {
    ignoreBrowserslistConfig = browserslistConfigFile === false;
  }
  return (0, _helperCompilationTargets().default)(targets, {
    ignoreBrowserslistConfig,
    configFile,
    configPath: root,
    browserslistEnv: options.browserslistEnv
  });
}
0 && 0;

//# sourceMappingURL=resolve-targets.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614392, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.buildPresetChain = buildPresetChain;
exports.buildPresetChainWalker = void 0;
exports.buildRootChain = buildRootChain;
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
function _debug() {
  const data = require("debug");
  _debug = function () {
    return data;
  };
  return data;
}
var _options = require("./validation/options.js");
var _patternToRegex = require("./pattern-to-regex.js");
var _printer = require("./printer.js");
var _rewriteStackTrace = require("../errors/rewrite-stack-trace.js");
var _configError = require("../errors/config-error.js");
var _index = require("./files/index.js");
var _caching = require("./caching.js");
var _configDescriptors = require("./config-descriptors.js");
const debug = _debug()("babel:config:config-chain");
function* buildPresetChain(arg, context) {
  const chain = yield* buildPresetChainWalker(arg, context);
  if (!chain) return null;
  return {
    plugins: dedupDescriptors(chain.plugins),
    presets: dedupDescriptors(chain.presets),
    options: chain.options.map(o => normalizeOptions(o)),
    files: new Set()
  };
}
const buildPresetChainWalker = exports.buildPresetChainWalker = makeChainWalker({
  root: preset => loadPresetDescriptors(preset),
  env: (preset, envName) => loadPresetEnvDescriptors(preset)(envName),
  overrides: (preset, index) => loadPresetOverridesDescriptors(preset)(index),
  overridesEnv: (preset, index, envName) => loadPresetOverridesEnvDescriptors(preset)(index)(envName),
  createLogger: () => () => {}
});
const loadPresetDescriptors = (0, _caching.makeWeakCacheSync)(preset => buildRootDescriptors(preset, preset.alias, _configDescriptors.createUncachedDescriptors));
const loadPresetEnvDescriptors = (0, _caching.makeWeakCacheSync)(preset => (0, _caching.makeStrongCacheSync)(envName => buildEnvDescriptors(preset, preset.alias, _configDescriptors.createUncachedDescriptors, envName)));
const loadPresetOverridesDescriptors = (0, _caching.makeWeakCacheSync)(preset => (0, _caching.makeStrongCacheSync)(index => buildOverrideDescriptors(preset, preset.alias, _configDescriptors.createUncachedDescriptors, index)));
const loadPresetOverridesEnvDescriptors = (0, _caching.makeWeakCacheSync)(preset => (0, _caching.makeStrongCacheSync)(index => (0, _caching.makeStrongCacheSync)(envName => buildOverrideEnvDescriptors(preset, preset.alias, _configDescriptors.createUncachedDescriptors, index, envName))));
function* buildRootChain(opts, context) {
  let configReport, babelRcReport;
  const programmaticLogger = new _printer.ConfigPrinter();
  const programmaticChain = yield* loadProgrammaticChain({
    options: opts,
    dirname: context.cwd
  }, context, undefined, programmaticLogger);
  if (!programmaticChain) return null;
  const programmaticReport = yield* programmaticLogger.output();
  let configFile;
  if (typeof opts.configFile === "string") {
    configFile = yield* (0, _index.loadConfig)(opts.configFile, context.cwd, context.envName, context.caller);
  } else if (opts.configFile !== false) {
    configFile = yield* (0, _index.findRootConfig)(context.root, context.envName, context.caller);
  }
  let {
    babelrc,
    babelrcRoots
  } = opts;
  let babelrcRootsDirectory = context.cwd;
  const configFileChain = emptyChain();
  const configFileLogger = new _printer.ConfigPrinter();
  if (configFile) {
    const validatedFile = validateConfigFile(configFile);
    const result = yield* loadFileChain(validatedFile, context, undefined, configFileLogger);
    if (!result) return null;
    configReport = yield* configFileLogger.output();
    if (babelrc === undefined) {
      babelrc = validatedFile.options.babelrc;
    }
    if (babelrcRoots === undefined) {
      babelrcRootsDirectory = validatedFile.dirname;
      babelrcRoots = validatedFile.options.babelrcRoots;
    }
    mergeChain(configFileChain, result);
  }
  let ignoreFile, babelrcFile;
  let isIgnored = false;
  const fileChain = emptyChain();
  if ((babelrc === true || babelrc === undefined) && typeof context.filename === "string") {
    const pkgData = yield* (0, _index.findPackageData)(context.filename);
    if (pkgData && babelrcLoadEnabled(context, pkgData, babelrcRoots, babelrcRootsDirectory)) {
      ({
        ignore: ignoreFile,
        config: babelrcFile
      } = yield* (0, _index.findRelativeConfig)(pkgData, context.envName, context.caller));
      if (ignoreFile) {
        fileChain.files.add(ignoreFile.filepath);
      }
      if (ignoreFile && shouldIgnore(context, ignoreFile.ignore, null, ignoreFile.dirname)) {
        isIgnored = true;
      }
      if (babelrcFile && !isIgnored) {
        const validatedFile = validateBabelrcFile(babelrcFile);
        const babelrcLogger = new _printer.ConfigPrinter();
        const result = yield* loadFileChain(validatedFile, context, undefined, babelrcLogger);
        if (!result) {
          isIgnored = true;
        } else {
          babelRcReport = yield* babelrcLogger.output();
          mergeChain(fileChain, result);
        }
      }
      if (babelrcFile && isIgnored) {
        fileChain.files.add(babelrcFile.filepath);
      }
    }
  }
  if (context.showConfig) {
    console.log(`Babel configs on "${context.filename}" (ascending priority):\n` + [configReport, babelRcReport, programmaticReport].filter(x => !!x).join("\n\n") + "\n-----End Babel configs-----");
  }
  const chain = mergeChain(mergeChain(mergeChain(emptyChain(), configFileChain), fileChain), programmaticChain);
  return {
    plugins: isIgnored ? [] : dedupDescriptors(chain.plugins),
    presets: isIgnored ? [] : dedupDescriptors(chain.presets),
    options: isIgnored ? [] : chain.options.map(o => normalizeOptions(o)),
    fileHandling: isIgnored ? "ignored" : "transpile",
    ignore: ignoreFile || undefined,
    babelrc: babelrcFile || undefined,
    config: configFile || undefined,
    files: chain.files
  };
}
function babelrcLoadEnabled(context, pkgData, babelrcRoots, babelrcRootsDirectory) {
  if (typeof babelrcRoots === "boolean") return babelrcRoots;
  const absoluteRoot = context.root;
  if (babelrcRoots === undefined) {
    return pkgData.directories.includes(absoluteRoot);
  }
  let babelrcPatterns = babelrcRoots;
  if (!Array.isArray(babelrcPatterns)) {
    babelrcPatterns = [babelrcPatterns];
  }
  babelrcPatterns = babelrcPatterns.map(pat => {
    return typeof pat === "string" ? _path().resolve(babelrcRootsDirectory, pat) : pat;
  });
  if (babelrcPatterns.length === 1 && babelrcPatterns[0] === absoluteRoot) {
    return pkgData.directories.includes(absoluteRoot);
  }
  return babelrcPatterns.some(pat => {
    if (typeof pat === "string") {
      pat = (0, _patternToRegex.default)(pat, babelrcRootsDirectory);
    }
    return pkgData.directories.some(directory => {
      return matchPattern(pat, babelrcRootsDirectory, directory, context);
    });
  });
}
const validateConfigFile = (0, _caching.makeWeakCacheSync)(file => ({
  filepath: file.filepath,
  dirname: file.dirname,
  options: (0, _options.validate)("configfile", file.options, file.filepath)
}));
const validateBabelrcFile = (0, _caching.makeWeakCacheSync)(file => ({
  filepath: file.filepath,
  dirname: file.dirname,
  options: (0, _options.validate)("babelrcfile", file.options, file.filepath)
}));
const validateExtendFile = (0, _caching.makeWeakCacheSync)(file => ({
  filepath: file.filepath,
  dirname: file.dirname,
  options: (0, _options.validate)("extendsfile", file.options, file.filepath)
}));
const loadProgrammaticChain = makeChainWalker({
  root: input => buildRootDescriptors(input, "base", _configDescriptors.createCachedDescriptors),
  env: (input, envName) => buildEnvDescriptors(input, "base", _configDescriptors.createCachedDescriptors, envName),
  overrides: (input, index) => buildOverrideDescriptors(input, "base", _configDescriptors.createCachedDescriptors, index),
  overridesEnv: (input, index, envName) => buildOverrideEnvDescriptors(input, "base", _configDescriptors.createCachedDescriptors, index, envName),
  createLogger: (input, context, baseLogger) => buildProgrammaticLogger(input, context, baseLogger)
});
const loadFileChainWalker = makeChainWalker({
  root: file => loadFileDescriptors(file),
  env: (file, envName) => loadFileEnvDescriptors(file)(envName),
  overrides: (file, index) => loadFileOverridesDescriptors(file)(index),
  overridesEnv: (file, index, envName) => loadFileOverridesEnvDescriptors(file)(index)(envName),
  createLogger: (file, context, baseLogger) => buildFileLogger(file.filepath, context, baseLogger)
});
function* loadFileChain(input, context, files, baseLogger) {
  const chain = yield* loadFileChainWalker(input, context, files, baseLogger);
  chain == null || chain.files.add(input.filepath);
  return chain;
}
const loadFileDescriptors = (0, _caching.makeWeakCacheSync)(file => buildRootDescriptors(file, file.filepath, _configDescriptors.createUncachedDescriptors));
const loadFileEnvDescriptors = (0, _caching.makeWeakCacheSync)(file => (0, _caching.makeStrongCacheSync)(envName => buildEnvDescriptors(file, file.filepath, _configDescriptors.createUncachedDescriptors, envName)));
const loadFileOverridesDescriptors = (0, _caching.makeWeakCacheSync)(file => (0, _caching.makeStrongCacheSync)(index => buildOverrideDescriptors(file, file.filepath, _configDescriptors.createUncachedDescriptors, index)));
const loadFileOverridesEnvDescriptors = (0, _caching.makeWeakCacheSync)(file => (0, _caching.makeStrongCacheSync)(index => (0, _caching.makeStrongCacheSync)(envName => buildOverrideEnvDescriptors(file, file.filepath, _configDescriptors.createUncachedDescriptors, index, envName))));
function buildFileLogger(filepath, context, baseLogger) {
  if (!baseLogger) {
    return () => {};
  }
  return baseLogger.configure(context.showConfig, _printer.ChainFormatter.Config, {
    filepath
  });
}
function buildRootDescriptors({
  dirname,
  options
}, alias, descriptors) {
  return descriptors(dirname, options, alias);
}
function buildProgrammaticLogger(_, context, baseLogger) {
  var _context$caller;
  if (!baseLogger) {
    return () => {};
  }
  return baseLogger.configure(context.showConfig, _printer.ChainFormatter.Programmatic, {
    callerName: (_context$caller = context.caller) == null ? void 0 : _context$caller.name
  });
}
function buildEnvDescriptors({
  dirname,
  options
}, alias, descriptors, envName) {
  var _options$env;
  const opts = (_options$env = options.env) == null ? void 0 : _options$env[envName];
  return opts ? descriptors(dirname, opts, `${alias}.env["${envName}"]`) : null;
}
function buildOverrideDescriptors({
  dirname,
  options
}, alias, descriptors, index) {
  var _options$overrides;
  const opts = (_options$overrides = options.overrides) == null ? void 0 : _options$overrides[index];
  if (!opts) throw new Error("Assertion failure - missing override");
  return descriptors(dirname, opts, `${alias}.overrides[${index}]`);
}
function buildOverrideEnvDescriptors({
  dirname,
  options
}, alias, descriptors, index, envName) {
  var _options$overrides2, _override$env;
  const override = (_options$overrides2 = options.overrides) == null ? void 0 : _options$overrides2[index];
  if (!override) throw new Error("Assertion failure - missing override");
  const opts = (_override$env = override.env) == null ? void 0 : _override$env[envName];
  return opts ? descriptors(dirname, opts, `${alias}.overrides[${index}].env["${envName}"]`) : null;
}
function makeChainWalker({
  root,
  env,
  overrides,
  overridesEnv,
  createLogger
}) {
  return function* chainWalker(input, context, files = new Set(), baseLogger) {
    const {
      dirname
    } = input;
    const flattenedConfigs = [];
    const rootOpts = root(input);
    if (configIsApplicable(rootOpts, dirname, context, input.filepath)) {
      flattenedConfigs.push({
        config: rootOpts,
        envName: undefined,
        index: undefined
      });
      const envOpts = env(input, context.envName);
      if (envOpts && configIsApplicable(envOpts, dirname, context, input.filepath)) {
        flattenedConfigs.push({
          config: envOpts,
          envName: context.envName,
          index: undefined
        });
      }
      (rootOpts.options.overrides || []).forEach((_, index) => {
        const overrideOps = overrides(input, index);
        if (configIsApplicable(overrideOps, dirname, context, input.filepath)) {
          flattenedConfigs.push({
            config: overrideOps,
            index,
            envName: undefined
          });
          const overrideEnvOpts = overridesEnv(input, index, context.envName);
          if (overrideEnvOpts && configIsApplicable(overrideEnvOpts, dirname, context, input.filepath)) {
            flattenedConfigs.push({
              config: overrideEnvOpts,
              index,
              envName: context.envName
            });
          }
        }
      });
    }
    if (flattenedConfigs.some(({
      config: {
        options: {
          ignore,
          only
        }
      }
    }) => shouldIgnore(context, ignore, only, dirname))) {
      return null;
    }
    const chain = emptyChain();
    const logger = createLogger(input, context, baseLogger);
    for (const {
      config,
      index,
      envName
    } of flattenedConfigs) {
      if (!(yield* mergeExtendsChain(chain, config.options, dirname, context, files, baseLogger))) {
        return null;
      }
      logger(config, index, envName);
      yield* mergeChainOpts(chain, config);
    }
    return chain;
  };
}
function* mergeExtendsChain(chain, opts, dirname, context, files, baseLogger) {
  if (opts.extends === undefined) return true;
  const file = yield* (0, _index.loadConfig)(opts.extends, dirname, context.envName, context.caller);
  if (files.has(file)) {
    throw new Error(`Configuration cycle detected loading ${file.filepath}.\n` + `File already loaded following the config chain:\n` + Array.from(files, file => ` - ${file.filepath}`).join("\n"));
  }
  files.add(file);
  const fileChain = yield* loadFileChain(validateExtendFile(file), context, files, baseLogger);
  files.delete(file);
  if (!fileChain) return false;
  mergeChain(chain, fileChain);
  return true;
}
function mergeChain(target, source) {
  target.options.push(...source.options);
  target.plugins.push(...source.plugins);
  target.presets.push(...source.presets);
  for (const file of source.files) {
    target.files.add(file);
  }
  return target;
}
function* mergeChainOpts(target, {
  options,
  plugins,
  presets
}) {
  target.options.push(options);
  target.plugins.push(...(yield* plugins()));
  target.presets.push(...(yield* presets()));
  return target;
}
function emptyChain() {
  return {
    options: [],
    presets: [],
    plugins: [],
    files: new Set()
  };
}
function normalizeOptions(opts) {
  const options = Object.assign({}, opts);
  delete options.extends;
  delete options.env;
  delete options.overrides;
  delete options.plugins;
  delete options.presets;
  delete options.passPerPreset;
  delete options.ignore;
  delete options.only;
  delete options.test;
  delete options.include;
  delete options.exclude;
  if (hasOwnProperty.call(options, "sourceMap")) {
    options.sourceMaps = options.sourceMap;
    delete options.sourceMap;
  }
  return options;
}
function dedupDescriptors(items) {
  const map = new Map();
  const descriptors = [];
  for (const item of items) {
    if (typeof item.value === "function") {
      const fnKey = item.value;
      let nameMap = map.get(fnKey);
      if (!nameMap) {
        nameMap = new Map();
        map.set(fnKey, nameMap);
      }
      let desc = nameMap.get(item.name);
      if (!desc) {
        desc = {
          value: item
        };
        descriptors.push(desc);
        if (!item.ownPass) nameMap.set(item.name, desc);
      } else {
        desc.value = item;
      }
    } else {
      descriptors.push({
        value: item
      });
    }
  }
  return descriptors.reduce((acc, desc) => {
    acc.push(desc.value);
    return acc;
  }, []);
}
function configIsApplicable({
  options
}, dirname, context, configName) {
  return (options.test === undefined || configFieldIsApplicable(context, options.test, dirname, configName)) && (options.include === undefined || configFieldIsApplicable(context, options.include, dirname, configName)) && (options.exclude === undefined || !configFieldIsApplicable(context, options.exclude, dirname, configName));
}
function configFieldIsApplicable(context, test, dirname, configName) {
  const patterns = Array.isArray(test) ? test : [test];
  return matchesPatterns(context, patterns, dirname, configName);
}
function ignoreListReplacer(_key, value) {
  if (value instanceof RegExp) {
    return String(value);
  }
  return value;
}
function shouldIgnore(context, ignore, only, dirname) {
  if (ignore && matchesPatterns(context, ignore, dirname)) {
    var _context$filename;
    const message = `No config is applied to "${(_context$filename = context.filename) != null ? _context$filename : "(unknown)"}" because it matches one of \`ignore: ${JSON.stringify(ignore, ignoreListReplacer)}\` from "${dirname}"`;
    debug(message);
    if (context.showConfig) {
      console.log(message);
    }
    return true;
  }
  if (only && !matchesPatterns(context, only, dirname)) {
    var _context$filename2;
    const message = `No config is applied to "${(_context$filename2 = context.filename) != null ? _context$filename2 : "(unknown)"}" because it fails to match one of \`only: ${JSON.stringify(only, ignoreListReplacer)}\` from "${dirname}"`;
    debug(message);
    if (context.showConfig) {
      console.log(message);
    }
    return true;
  }
  return false;
}
function matchesPatterns(context, patterns, dirname, configName) {
  return patterns.some(pattern => matchPattern(pattern, dirname, context.filename, context, configName));
}
function matchPattern(pattern, dirname, pathToTest, context, configName) {
  if (typeof pattern === "function") {
    return !!(0, _rewriteStackTrace.endHiddenCallStack)(pattern)(pathToTest, {
      dirname,
      envName: context.envName,
      caller: context.caller
    });
  }
  if (typeof pathToTest !== "string") {
    throw new _configError.default(`Configuration contains string/RegExp pattern, but no filename was passed to Babel`, configName);
  }
  if (typeof pattern === "string") {
    pattern = (0, _patternToRegex.default)(pattern, dirname);
  }
  return pattern.test(pathToTest);
}
0 && 0;

//# sourceMappingURL=config-chain.js.map

}, function(modId) { var map = {"./validation/options.js":1781530614393,"./pattern-to-regex.js":1781530614396,"./printer.js":1781530614397,"../errors/rewrite-stack-trace.js":1781530614379,"../errors/config-error.js":1781530614378,"./files/index.js":1781530614371,"./caching.js":1781530614374,"./config-descriptors.js":1781530614389}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614393, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.assumptionsNames = void 0;
exports.checkNoUnwrappedItemOptionPairs = checkNoUnwrappedItemOptionPairs;
exports.validate = validate;
var _removed = require("./removed.js");
var _optionAssertions = require("./option-assertions.js");
var _configError = require("../../errors/config-error.js");
const ROOT_VALIDATORS = {
  cwd: _optionAssertions.assertString,
  root: _optionAssertions.assertString,
  rootMode: _optionAssertions.assertRootMode,
  configFile: _optionAssertions.assertConfigFileSearch,
  caller: _optionAssertions.assertCallerMetadata,
  filename: _optionAssertions.assertString,
  filenameRelative: _optionAssertions.assertString,
  code: _optionAssertions.assertBoolean,
  ast: _optionAssertions.assertBoolean,
  cloneInputAst: _optionAssertions.assertBoolean,
  envName: _optionAssertions.assertString
};
const BABELRC_VALIDATORS = {
  babelrc: _optionAssertions.assertBoolean,
  babelrcRoots: _optionAssertions.assertBabelrcSearch
};
const NONPRESET_VALIDATORS = {
  extends: _optionAssertions.assertString,
  ignore: _optionAssertions.assertIgnoreList,
  only: _optionAssertions.assertIgnoreList,
  targets: _optionAssertions.assertTargets,
  browserslistConfigFile: _optionAssertions.assertConfigFileSearch,
  browserslistEnv: _optionAssertions.assertString
};
const COMMON_VALIDATORS = {
  inputSourceMap: _optionAssertions.assertInputSourceMap,
  presets: _optionAssertions.assertPluginList,
  plugins: _optionAssertions.assertPluginList,
  passPerPreset: _optionAssertions.assertBoolean,
  assumptions: _optionAssertions.assertAssumptions,
  env: assertEnvSet,
  overrides: assertOverridesList,
  test: _optionAssertions.assertConfigApplicableTest,
  include: _optionAssertions.assertConfigApplicableTest,
  exclude: _optionAssertions.assertConfigApplicableTest,
  retainLines: _optionAssertions.assertBoolean,
  comments: _optionAssertions.assertBoolean,
  shouldPrintComment: _optionAssertions.assertFunction,
  compact: _optionAssertions.assertCompact,
  minified: _optionAssertions.assertBoolean,
  auxiliaryCommentBefore: _optionAssertions.assertString,
  auxiliaryCommentAfter: _optionAssertions.assertString,
  sourceType: _optionAssertions.assertSourceType,
  wrapPluginVisitorMethod: _optionAssertions.assertFunction,
  highlightCode: _optionAssertions.assertBoolean,
  sourceMaps: _optionAssertions.assertSourceMaps,
  sourceMap: _optionAssertions.assertSourceMaps,
  sourceFileName: _optionAssertions.assertString,
  sourceRoot: _optionAssertions.assertString,
  parserOpts: _optionAssertions.assertObject,
  generatorOpts: _optionAssertions.assertObject
};
{
  Object.assign(COMMON_VALIDATORS, {
    getModuleId: _optionAssertions.assertFunction,
    moduleRoot: _optionAssertions.assertString,
    moduleIds: _optionAssertions.assertBoolean,
    moduleId: _optionAssertions.assertString
  });
}
const knownAssumptions = ["arrayLikeIsIterable", "constantReexports", "constantSuper", "enumerableModuleMeta", "ignoreFunctionLength", "ignoreToPrimitiveHint", "iterableIsArray", "mutableTemplateObject", "noClassCalls", "noDocumentAll", "noIncompleteNsImportDetection", "noNewArrows", "noUninitializedPrivateFieldAccess", "objectRestNoSymbols", "privateFieldsAsSymbols", "privateFieldsAsProperties", "pureGetters", "setClassMethods", "setComputedProperties", "setPublicClassFields", "setSpreadProperties", "skipForOfIteratorClosing", "superIsCallableConstructor"];
const assumptionsNames = exports.assumptionsNames = new Set(knownAssumptions);
function getSource(loc) {
  return loc.type === "root" ? loc.source : getSource(loc.parent);
}
function validate(type, opts, filename) {
  try {
    return validateNested({
      type: "root",
      source: type
    }, opts);
  } catch (error) {
    const configError = new _configError.default(error.message, filename);
    if (error.code) configError.code = error.code;
    throw configError;
  }
}
function validateNested(loc, opts) {
  const type = getSource(loc);
  assertNoDuplicateSourcemap(opts);
  Object.keys(opts).forEach(key => {
    const optLoc = {
      type: "option",
      name: key,
      parent: loc
    };
    if (type === "preset" && NONPRESET_VALIDATORS[key]) {
      throw new Error(`${(0, _optionAssertions.msg)(optLoc)} is not allowed in preset options`);
    }
    if (type !== "arguments" && ROOT_VALIDATORS[key]) {
      throw new Error(`${(0, _optionAssertions.msg)(optLoc)} is only allowed in root programmatic options`);
    }
    if (type !== "arguments" && type !== "configfile" && BABELRC_VALIDATORS[key]) {
      if (type === "babelrcfile" || type === "extendsfile") {
        throw new Error(`${(0, _optionAssertions.msg)(optLoc)} is not allowed in .babelrc or "extends"ed files, only in root programmatic options, ` + `or babel.config.js/config file options`);
      }
      throw new Error(`${(0, _optionAssertions.msg)(optLoc)} is only allowed in root programmatic options, or babel.config.js/config file options`);
    }
    const validator = COMMON_VALIDATORS[key] || NONPRESET_VALIDATORS[key] || BABELRC_VALIDATORS[key] || ROOT_VALIDATORS[key] || throwUnknownError;
    validator(optLoc, opts[key]);
  });
  return opts;
}
function throwUnknownError(loc) {
  const key = loc.name;
  if (_removed.default[key]) {
    const {
      message,
      version = 5
    } = _removed.default[key];
    throw new Error(`Using removed Babel ${version} option: ${(0, _optionAssertions.msg)(loc)} - ${message}`);
  } else {
    const unknownOptErr = new Error(`Unknown option: ${(0, _optionAssertions.msg)(loc)}. Check out https://babeljs.io/docs/en/babel-core/#options for more information about options.`);
    unknownOptErr.code = "BABEL_UNKNOWN_OPTION";
    throw unknownOptErr;
  }
}
function assertNoDuplicateSourcemap(opts) {
  if (hasOwnProperty.call(opts, "sourceMap") && hasOwnProperty.call(opts, "sourceMaps")) {
    throw new Error(".sourceMap is an alias for .sourceMaps, cannot use both");
  }
}
function assertEnvSet(loc, value) {
  if (loc.parent.type === "env") {
    throw new Error(`${(0, _optionAssertions.msg)(loc)} is not allowed inside of another .env block`);
  }
  const parent = loc.parent;
  const obj = (0, _optionAssertions.assertObject)(loc, value);
  if (obj) {
    for (const envName of Object.keys(obj)) {
      const env = (0, _optionAssertions.assertObject)((0, _optionAssertions.access)(loc, envName), obj[envName]);
      if (!env) continue;
      const envLoc = {
        type: "env",
        name: envName,
        parent
      };
      validateNested(envLoc, env);
    }
  }
  return obj;
}
function assertOverridesList(loc, value) {
  if (loc.parent.type === "env") {
    throw new Error(`${(0, _optionAssertions.msg)(loc)} is not allowed inside an .env block`);
  }
  if (loc.parent.type === "overrides") {
    throw new Error(`${(0, _optionAssertions.msg)(loc)} is not allowed inside an .overrides block`);
  }
  const parent = loc.parent;
  const arr = (0, _optionAssertions.assertArray)(loc, value);
  if (arr) {
    for (const [index, item] of arr.entries()) {
      const objLoc = (0, _optionAssertions.access)(loc, index);
      const env = (0, _optionAssertions.assertObject)(objLoc, item);
      if (!env) throw new Error(`${(0, _optionAssertions.msg)(objLoc)} must be an object`);
      const overridesLoc = {
        type: "overrides",
        index,
        parent
      };
      validateNested(overridesLoc, env);
    }
  }
  return arr;
}
function checkNoUnwrappedItemOptionPairs(items, index, type, e) {
  if (index === 0) return;
  const lastItem = items[index - 1];
  const thisItem = items[index];
  if (lastItem.file && lastItem.options === undefined && typeof thisItem.value === "object") {
    e.message += `\n- Maybe you meant to use\n` + `"${type}s": [\n  ["${lastItem.file.request}", ${JSON.stringify(thisItem.value, undefined, 2)}]\n]\n` + `To be a valid ${type}, its name and options should be wrapped in a pair of brackets`;
  }
}
0 && 0;

//# sourceMappingURL=options.js.map

}, function(modId) { var map = {"./removed.js":1781530614394,"./option-assertions.js":1781530614395,"../../errors/config-error.js":1781530614378}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614394, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
var _default = exports.default = {
  auxiliaryComment: {
    message: "Use `auxiliaryCommentBefore` or `auxiliaryCommentAfter`"
  },
  blacklist: {
    message: "Put the specific transforms you want in the `plugins` option"
  },
  breakConfig: {
    message: "This is not a necessary option in Babel 6"
  },
  experimental: {
    message: "Put the specific transforms you want in the `plugins` option"
  },
  externalHelpers: {
    message: "Use the `external-helpers` plugin instead. " + "Check out http://babeljs.io/docs/plugins/external-helpers/"
  },
  extra: {
    message: ""
  },
  jsxPragma: {
    message: "use the `pragma` option in the `react-jsx` plugin. " + "Check out http://babeljs.io/docs/plugins/transform-react-jsx/"
  },
  loose: {
    message: "Specify the `loose` option for the relevant plugin you are using " + "or use a preset that sets the option."
  },
  metadataUsedHelpers: {
    message: "Not required anymore as this is enabled by default"
  },
  modules: {
    message: "Use the corresponding module transform plugin in the `plugins` option. " + "Check out http://babeljs.io/docs/plugins/#modules"
  },
  nonStandard: {
    message: "Use the `react-jsx` and `flow-strip-types` plugins to support JSX and Flow. " + "Also check out the react preset http://babeljs.io/docs/plugins/preset-react/"
  },
  optional: {
    message: "Put the specific transforms you want in the `plugins` option"
  },
  sourceMapName: {
    message: "The `sourceMapName` option has been removed because it makes more sense for the " + "tooling that calls Babel to assign `map.file` themselves."
  },
  stage: {
    message: "Check out the corresponding stage-x presets http://babeljs.io/docs/plugins/#presets"
  },
  whitelist: {
    message: "Put the specific transforms you want in the `plugins` option"
  },
  resolveModuleSource: {
    version: 6,
    message: "Use `babel-plugin-module-resolver@3`'s 'resolvePath' options"
  },
  metadata: {
    version: 6,
    message: "Generated plugin metadata is always included in the output result"
  },
  sourceMapTarget: {
    version: 6,
    message: "The `sourceMapTarget` option has been removed because it makes more sense for the tooling " + "that calls Babel to assign `map.file` themselves."
  }
};
0 && 0;

//# sourceMappingURL=removed.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614395, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.access = access;
exports.assertArray = assertArray;
exports.assertAssumptions = assertAssumptions;
exports.assertBabelrcSearch = assertBabelrcSearch;
exports.assertBoolean = assertBoolean;
exports.assertCallerMetadata = assertCallerMetadata;
exports.assertCompact = assertCompact;
exports.assertConfigApplicableTest = assertConfigApplicableTest;
exports.assertConfigFileSearch = assertConfigFileSearch;
exports.assertFunction = assertFunction;
exports.assertIgnoreList = assertIgnoreList;
exports.assertInputSourceMap = assertInputSourceMap;
exports.assertObject = assertObject;
exports.assertPluginList = assertPluginList;
exports.assertRootMode = assertRootMode;
exports.assertSourceMaps = assertSourceMaps;
exports.assertSourceType = assertSourceType;
exports.assertString = assertString;
exports.assertTargets = assertTargets;
exports.msg = msg;
function _helperCompilationTargets() {
  const data = require("@babel/helper-compilation-targets");
  _helperCompilationTargets = function () {
    return data;
  };
  return data;
}
var _options = require("./options.js");
function msg(loc) {
  switch (loc.type) {
    case "root":
      return ``;
    case "env":
      return `${msg(loc.parent)}.env["${loc.name}"]`;
    case "overrides":
      return `${msg(loc.parent)}.overrides[${loc.index}]`;
    case "option":
      return `${msg(loc.parent)}.${loc.name}`;
    case "access":
      return `${msg(loc.parent)}[${JSON.stringify(loc.name)}]`;
    default:
      throw new Error(`Assertion failure: Unknown type ${loc.type}`);
  }
}
function access(loc, name) {
  return {
    type: "access",
    name,
    parent: loc
  };
}
function assertRootMode(loc, value) {
  if (value !== undefined && value !== "root" && value !== "upward" && value !== "upward-optional") {
    throw new Error(`${msg(loc)} must be a "root", "upward", "upward-optional" or undefined`);
  }
  return value;
}
function assertSourceMaps(loc, value) {
  if (value !== undefined && typeof value !== "boolean" && value !== "inline" && value !== "both") {
    throw new Error(`${msg(loc)} must be a boolean, "inline", "both", or undefined`);
  }
  return value;
}
function assertCompact(loc, value) {
  if (value !== undefined && typeof value !== "boolean" && value !== "auto") {
    throw new Error(`${msg(loc)} must be a boolean, "auto", or undefined`);
  }
  return value;
}
function assertSourceType(loc, value) {
  if (value !== undefined && value !== "module" && value !== "script" && value !== "unambiguous") {
    throw new Error(`${msg(loc)} must be "module", "script", "unambiguous", or undefined`);
  }
  return value;
}
function assertCallerMetadata(loc, value) {
  const obj = assertObject(loc, value);
  if (obj) {
    if (typeof obj.name !== "string") {
      throw new Error(`${msg(loc)} set but does not contain "name" property string`);
    }
    for (const prop of Object.keys(obj)) {
      const propLoc = access(loc, prop);
      const value = obj[prop];
      if (value != null && typeof value !== "boolean" && typeof value !== "string" && typeof value !== "number") {
        throw new Error(`${msg(propLoc)} must be null, undefined, a boolean, a string, or a number.`);
      }
    }
  }
  return value;
}
function assertInputSourceMap(loc, value) {
  if (value !== undefined && typeof value !== "boolean" && (typeof value !== "object" || !value)) {
    throw new Error(`${msg(loc)} must be a boolean, object, or undefined`);
  }
  return value;
}
function assertString(loc, value) {
  if (value !== undefined && typeof value !== "string") {
    throw new Error(`${msg(loc)} must be a string, or undefined`);
  }
  return value;
}
function assertFunction(loc, value) {
  if (value !== undefined && typeof value !== "function") {
    throw new Error(`${msg(loc)} must be a function, or undefined`);
  }
  return value;
}
function assertBoolean(loc, value) {
  if (value !== undefined && typeof value !== "boolean") {
    throw new Error(`${msg(loc)} must be a boolean, or undefined`);
  }
  return value;
}
function assertObject(loc, value) {
  if (value !== undefined && (typeof value !== "object" || Array.isArray(value) || !value)) {
    throw new Error(`${msg(loc)} must be an object, or undefined`);
  }
  return value;
}
function assertArray(loc, value) {
  if (value != null && !Array.isArray(value)) {
    throw new Error(`${msg(loc)} must be an array, or undefined`);
  }
  return value;
}
function assertIgnoreList(loc, value) {
  const arr = assertArray(loc, value);
  arr == null || arr.forEach((item, i) => assertIgnoreItem(access(loc, i), item));
  return arr;
}
function assertIgnoreItem(loc, value) {
  if (typeof value !== "string" && typeof value !== "function" && !(value instanceof RegExp)) {
    throw new Error(`${msg(loc)} must be an array of string/Function/RegExp values, or undefined`);
  }
  return value;
}
function assertConfigApplicableTest(loc, value) {
  if (value === undefined) {
    return value;
  }
  if (Array.isArray(value)) {
    value.forEach((item, i) => {
      if (!checkValidTest(item)) {
        throw new Error(`${msg(access(loc, i))} must be a string/Function/RegExp.`);
      }
    });
  } else if (!checkValidTest(value)) {
    throw new Error(`${msg(loc)} must be a string/Function/RegExp, or an array of those`);
  }
  return value;
}
function checkValidTest(value) {
  return typeof value === "string" || typeof value === "function" || value instanceof RegExp;
}
function assertConfigFileSearch(loc, value) {
  if (value !== undefined && typeof value !== "boolean" && typeof value !== "string") {
    throw new Error(`${msg(loc)} must be a undefined, a boolean, a string, ` + `got ${JSON.stringify(value)}`);
  }
  return value;
}
function assertBabelrcSearch(loc, value) {
  if (value === undefined || typeof value === "boolean") {
    return value;
  }
  if (Array.isArray(value)) {
    value.forEach((item, i) => {
      if (!checkValidTest(item)) {
        throw new Error(`${msg(access(loc, i))} must be a string/Function/RegExp.`);
      }
    });
  } else if (!checkValidTest(value)) {
    throw new Error(`${msg(loc)} must be a undefined, a boolean, a string/Function/RegExp ` + `or an array of those, got ${JSON.stringify(value)}`);
  }
  return value;
}
function assertPluginList(loc, value) {
  const arr = assertArray(loc, value);
  if (arr) {
    arr.forEach((item, i) => assertPluginItem(access(loc, i), item));
  }
  return arr;
}
function assertPluginItem(loc, value) {
  if (Array.isArray(value)) {
    if (value.length === 0) {
      throw new Error(`${msg(loc)} must include an object`);
    }
    if (value.length > 3) {
      throw new Error(`${msg(loc)} may only be a two-tuple or three-tuple`);
    }
    assertPluginTarget(access(loc, 0), value[0]);
    if (value.length > 1) {
      const opts = value[1];
      if (opts !== undefined && opts !== false && (typeof opts !== "object" || Array.isArray(opts) || opts === null)) {
        throw new Error(`${msg(access(loc, 1))} must be an object, false, or undefined`);
      }
    }
    if (value.length === 3) {
      const name = value[2];
      if (name !== undefined && typeof name !== "string") {
        throw new Error(`${msg(access(loc, 2))} must be a string, or undefined`);
      }
    }
  } else {
    assertPluginTarget(loc, value);
  }
  return value;
}
function assertPluginTarget(loc, value) {
  if ((typeof value !== "object" || !value) && typeof value !== "string" && typeof value !== "function") {
    throw new Error(`${msg(loc)} must be a string, object, function`);
  }
  return value;
}
function assertTargets(loc, value) {
  if ((0, _helperCompilationTargets().isBrowsersQueryValid)(value)) return value;
  if (typeof value !== "object" || !value || Array.isArray(value)) {
    throw new Error(`${msg(loc)} must be a string, an array of strings or an object`);
  }
  const browsersLoc = access(loc, "browsers");
  const esmodulesLoc = access(loc, "esmodules");
  assertBrowsersList(browsersLoc, value.browsers);
  assertBoolean(esmodulesLoc, value.esmodules);
  for (const key of Object.keys(value)) {
    const val = value[key];
    const subLoc = access(loc, key);
    if (key === "esmodules") assertBoolean(subLoc, val);else if (key === "browsers") assertBrowsersList(subLoc, val);else if (!hasOwnProperty.call(_helperCompilationTargets().TargetNames, key)) {
      const validTargets = Object.keys(_helperCompilationTargets().TargetNames).join(", ");
      throw new Error(`${msg(subLoc)} is not a valid target. Supported targets are ${validTargets}`);
    } else assertBrowserVersion(subLoc, val);
  }
  return value;
}
function assertBrowsersList(loc, value) {
  if (value !== undefined && !(0, _helperCompilationTargets().isBrowsersQueryValid)(value)) {
    throw new Error(`${msg(loc)} must be undefined, a string or an array of strings`);
  }
}
function assertBrowserVersion(loc, value) {
  if (typeof value === "number" && Math.round(value) === value) return;
  if (typeof value === "string") return;
  throw new Error(`${msg(loc)} must be a string or an integer number`);
}
function assertAssumptions(loc, value) {
  if (value === undefined) return;
  if (typeof value !== "object" || value === null) {
    throw new Error(`${msg(loc)} must be an object or undefined.`);
  }
  let root = loc;
  do {
    root = root.parent;
  } while (root.type !== "root");
  const inPreset = root.source === "preset";
  for (const name of Object.keys(value)) {
    const subLoc = access(loc, name);
    if (!_options.assumptionsNames.has(name)) {
      throw new Error(`${msg(subLoc)} is not a supported assumption.`);
    }
    if (typeof value[name] !== "boolean") {
      throw new Error(`${msg(subLoc)} must be a boolean.`);
    }
    if (inPreset && value[name] === false) {
      throw new Error(`${msg(subLoc)} cannot be set to 'false' inside presets.`);
    }
  }
  return value;
}
0 && 0;

//# sourceMappingURL=option-assertions.js.map

}, function(modId) { var map = {"./options.js":1781530614393}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614396, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = pathToPattern;
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
const sep = `\\${_path().sep}`;
const endSep = `(?:${sep}|$)`;
const substitution = `[^${sep}]+`;
const starPat = `(?:${substitution}${sep})`;
const starPatLast = `(?:${substitution}${endSep})`;
const starStarPat = `${starPat}*?`;
const starStarPatLast = `${starPat}*?${starPatLast}?`;
function escapeRegExp(string) {
  return string.replace(/[|\\{}()[\]^$+*?.]/g, "\\$&");
}
function pathToPattern(pattern, dirname) {
  const parts = _path().resolve(dirname, pattern).split(_path().sep);
  return new RegExp(["^", ...parts.map((part, i) => {
    const last = i === parts.length - 1;
    if (part === "**") return last ? starStarPatLast : starStarPat;
    if (part === "*") return last ? starPatLast : starPat;
    if (part.indexOf("*.") === 0) {
      return substitution + escapeRegExp(part.slice(1)) + (last ? endSep : sep);
    }
    return escapeRegExp(part) + (last ? endSep : sep);
  })].join(""));
}
0 && 0;

//# sourceMappingURL=pattern-to-regex.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614397, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ConfigPrinter = exports.ChainFormatter = void 0;
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
const ChainFormatter = exports.ChainFormatter = {
  Programmatic: 0,
  Config: 1
};
const Formatter = {
  title(type, callerName, filepath) {
    let title = "";
    if (type === ChainFormatter.Programmatic) {
      title = "programmatic options";
      if (callerName) {
        title += " from " + callerName;
      }
    } else {
      title = "config " + filepath;
    }
    return title;
  },
  loc(index, envName) {
    let loc = "";
    if (index != null) {
      loc += `.overrides[${index}]`;
    }
    if (envName != null) {
      loc += `.env["${envName}"]`;
    }
    return loc;
  },
  *optionsAndDescriptors(opt) {
    const content = Object.assign({}, opt.options);
    delete content.overrides;
    delete content.env;
    const pluginDescriptors = [...(yield* opt.plugins())];
    if (pluginDescriptors.length) {
      content.plugins = pluginDescriptors.map(d => descriptorToConfig(d));
    }
    const presetDescriptors = [...(yield* opt.presets())];
    if (presetDescriptors.length) {
      content.presets = [...presetDescriptors].map(d => descriptorToConfig(d));
    }
    return JSON.stringify(content, undefined, 2);
  }
};
function descriptorToConfig(d) {
  var _d$file;
  let name = (_d$file = d.file) == null ? void 0 : _d$file.request;
  if (name == null) {
    if (typeof d.value === "object") {
      name = d.value;
    } else if (typeof d.value === "function") {
      name = `[Function: ${d.value.toString().slice(0, 50)} ... ]`;
    }
  }
  if (name == null) {
    name = "[Unknown]";
  }
  if (d.options === undefined) {
    return name;
  } else if (d.name == null) {
    return [name, d.options];
  } else {
    return [name, d.options, d.name];
  }
}
class ConfigPrinter {
  constructor() {
    this._stack = [];
  }
  configure(enabled, type, {
    callerName,
    filepath
  }) {
    if (!enabled) return () => {};
    return (content, index, envName) => {
      this._stack.push({
        type,
        callerName,
        filepath,
        content,
        index,
        envName
      });
    };
  }
  static *format(config) {
    let title = Formatter.title(config.type, config.callerName, config.filepath);
    const loc = Formatter.loc(config.index, config.envName);
    if (loc) title += ` ${loc}`;
    const content = yield* Formatter.optionsAndDescriptors(config.content);
    return `${title}\n${content}`;
  }
  *output() {
    if (this._stack.length === 0) return "";
    const configs = yield* _gensync().all(this._stack.map(s => ConfigPrinter.format(s)));
    return configs.join("\n\n");
  }
}
exports.ConfigPrinter = ConfigPrinter;
0 && 0;

//# sourceMappingURL=printer.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614398, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.validatePluginObject = validatePluginObject;
var _optionAssertions = require("./option-assertions.js");
const VALIDATORS = {
  name: _optionAssertions.assertString,
  manipulateOptions: _optionAssertions.assertFunction,
  pre: _optionAssertions.assertFunction,
  post: _optionAssertions.assertFunction,
  inherits: _optionAssertions.assertFunction,
  visitor: assertVisitorMap,
  parserOverride: _optionAssertions.assertFunction,
  generatorOverride: _optionAssertions.assertFunction
};
function assertVisitorMap(loc, value) {
  const obj = (0, _optionAssertions.assertObject)(loc, value);
  if (obj) {
    Object.keys(obj).forEach(prop => {
      if (prop !== "_exploded" && prop !== "_verified") {
        assertVisitorHandler(prop, obj[prop]);
      }
    });
    if (obj.enter || obj.exit) {
      throw new Error(`${(0, _optionAssertions.msg)(loc)} cannot contain catch-all "enter" or "exit" handlers. Please target individual nodes.`);
    }
  }
  return obj;
}
function assertVisitorHandler(key, value) {
  if (value && typeof value === "object") {
    Object.keys(value).forEach(handler => {
      if (handler !== "enter" && handler !== "exit") {
        throw new Error(`.visitor["${key}"] may only have .enter and/or .exit handlers.`);
      }
    });
  } else if (typeof value !== "function") {
    throw new Error(`.visitor["${key}"] must be a function`);
  }
}
function validatePluginObject(obj) {
  const rootPath = {
    type: "root",
    source: "plugin"
  };
  Object.keys(obj).forEach(key => {
    const validator = VALIDATORS[key];
    if (validator) {
      const optLoc = {
        type: "option",
        name: key,
        parent: rootPath
      };
      validator(optLoc, obj[key]);
    } else {
      const invalidPluginPropertyError = new Error(`.${key} is not a valid Plugin property`);
      invalidPluginPropertyError.code = "BABEL_UNKNOWN_PLUGIN_PROPERTY";
      throw invalidPluginPropertyError;
    }
  });
  return obj;
}
0 && 0;

//# sourceMappingURL=plugins.js.map

}, function(modId) { var map = {"./option-assertions.js":1781530614395}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614399, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = loadPrivatePartialConfig;
exports.loadPartialConfig = loadPartialConfig;
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
var _plugin = require("./plugin.js");
var _util = require("./util.js");
var _item = require("./item.js");
var _configChain = require("./config-chain.js");
var _environment = require("./helpers/environment.js");
var _options = require("./validation/options.js");
var _index = require("./files/index.js");
var _resolveTargets = require("./resolve-targets.js");
const _excluded = ["showIgnoredFiles"];
function _objectWithoutPropertiesLoose(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }
function resolveRootMode(rootDir, rootMode) {
  switch (rootMode) {
    case "root":
      return rootDir;
    case "upward-optional":
      {
        const upwardRootDir = (0, _index.findConfigUpwards)(rootDir);
        return upwardRootDir === null ? rootDir : upwardRootDir;
      }
    case "upward":
      {
        const upwardRootDir = (0, _index.findConfigUpwards)(rootDir);
        if (upwardRootDir !== null) return upwardRootDir;
        throw Object.assign(new Error(`Babel was run with rootMode:"upward" but a root could not ` + `be found when searching upward from "${rootDir}".\n` + `One of the following config files must be in the directory tree: ` + `"${_index.ROOT_CONFIG_FILENAMES.join(", ")}".`), {
          code: "BABEL_ROOT_NOT_FOUND",
          dirname: rootDir
        });
      }
    default:
      throw new Error(`Assertion failure - unknown rootMode value.`);
  }
}
function* loadPrivatePartialConfig(inputOpts) {
  if (inputOpts != null && (typeof inputOpts !== "object" || Array.isArray(inputOpts))) {
    throw new Error("Babel options must be an object, null, or undefined");
  }
  const args = inputOpts ? (0, _options.validate)("arguments", inputOpts) : {};
  const {
    envName = (0, _environment.getEnv)(),
    cwd = ".",
    root: rootDir = ".",
    rootMode = "root",
    caller,
    cloneInputAst = true
  } = args;
  const absoluteCwd = _path().resolve(cwd);
  const absoluteRootDir = resolveRootMode(_path().resolve(absoluteCwd, rootDir), rootMode);
  const filename = typeof args.filename === "string" ? _path().resolve(cwd, args.filename) : undefined;
  const showConfigPath = yield* (0, _index.resolveShowConfigPath)(absoluteCwd);
  const context = {
    filename,
    cwd: absoluteCwd,
    root: absoluteRootDir,
    envName,
    caller,
    showConfig: showConfigPath === filename
  };
  const configChain = yield* (0, _configChain.buildRootChain)(args, context);
  if (!configChain) return null;
  const merged = {
    assumptions: {}
  };
  configChain.options.forEach(opts => {
    (0, _util.mergeOptions)(merged, opts);
  });
  const options = Object.assign({}, merged, {
    targets: (0, _resolveTargets.resolveTargets)(merged, absoluteRootDir),
    cloneInputAst,
    babelrc: false,
    configFile: false,
    browserslistConfigFile: false,
    passPerPreset: false,
    envName: context.envName,
    cwd: context.cwd,
    root: context.root,
    rootMode: "root",
    filename: typeof context.filename === "string" ? context.filename : undefined,
    plugins: configChain.plugins.map(descriptor => (0, _item.createItemFromDescriptor)(descriptor)),
    presets: configChain.presets.map(descriptor => (0, _item.createItemFromDescriptor)(descriptor))
  });
  return {
    options,
    context,
    fileHandling: configChain.fileHandling,
    ignore: configChain.ignore,
    babelrc: configChain.babelrc,
    config: configChain.config,
    files: configChain.files
  };
}
function* loadPartialConfig(opts) {
  let showIgnoredFiles = false;
  if (typeof opts === "object" && opts !== null && !Array.isArray(opts)) {
    var _opts = opts;
    ({
      showIgnoredFiles
    } = _opts);
    opts = _objectWithoutPropertiesLoose(_opts, _excluded);
    _opts;
  }
  const result = yield* loadPrivatePartialConfig(opts);
  if (!result) return null;
  const {
    options,
    babelrc,
    ignore,
    config,
    fileHandling,
    files
  } = result;
  if (fileHandling === "ignored" && !showIgnoredFiles) {
    return null;
  }
  (options.plugins || []).forEach(item => {
    if (item.value instanceof _plugin.default) {
      throw new Error("Passing cached plugin instances is not supported in " + "babel.loadPartialConfig()");
    }
  });
  return new PartialConfig(options, babelrc ? babelrc.filepath : undefined, ignore ? ignore.filepath : undefined, config ? config.filepath : undefined, fileHandling, files);
}
class PartialConfig {
  constructor(options, babelrc, ignore, config, fileHandling, files) {
    this.options = void 0;
    this.babelrc = void 0;
    this.babelignore = void 0;
    this.config = void 0;
    this.fileHandling = void 0;
    this.files = void 0;
    this.options = options;
    this.babelignore = ignore;
    this.babelrc = babelrc;
    this.config = config;
    this.fileHandling = fileHandling;
    this.files = files;
    Object.freeze(this);
  }
  hasFilesystemConfig() {
    return this.babelrc !== undefined || this.config !== undefined;
  }
}
Object.freeze(PartialConfig.prototype);
0 && 0;

//# sourceMappingURL=partial.js.map

}, function(modId) { var map = {"./plugin.js":1781530614386,"./util.js":1781530614376,"./item.js":1781530614388,"./config-chain.js":1781530614392,"./helpers/environment.js":1781530614400,"./validation/options.js":1781530614393,"./files/index.js":1781530614371,"./resolve-targets.js":1781530614391}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614400, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.getEnv = getEnv;
function getEnv(defaultValue = "development") {
  return process.env.BABEL_ENV || process.env.NODE_ENV || defaultValue;
}
0 && 0;

//# sourceMappingURL=environment.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614401, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.run = run;
function _traverse() {
  const data = require("@babel/traverse");
  _traverse = function () {
    return data;
  };
  return data;
}
var _pluginPass = require("./plugin-pass.js");
var _blockHoistPlugin = require("./block-hoist-plugin.js");
var _normalizeOpts = require("./normalize-opts.js");
var _normalizeFile = require("./normalize-file.js");
var _generate = require("./file/generate.js");
var _deepArray = require("../config/helpers/deep-array.js");
function* run(config, code, ast) {
  const file = yield* (0, _normalizeFile.default)(config.passes, (0, _normalizeOpts.default)(config), code, ast);
  const opts = file.opts;
  try {
    yield* transformFile(file, config.passes);
  } catch (e) {
    var _opts$filename;
    e.message = `${(_opts$filename = opts.filename) != null ? _opts$filename : "unknown file"}: ${e.message}`;
    if (!e.code) {
      e.code = "BABEL_TRANSFORM_ERROR";
    }
    throw e;
  }
  let outputCode, outputMap;
  try {
    if (opts.code !== false) {
      ({
        outputCode,
        outputMap
      } = (0, _generate.default)(config.passes, file));
    }
  } catch (e) {
    var _opts$filename2;
    e.message = `${(_opts$filename2 = opts.filename) != null ? _opts$filename2 : "unknown file"}: ${e.message}`;
    if (!e.code) {
      e.code = "BABEL_GENERATE_ERROR";
    }
    throw e;
  }
  return {
    metadata: file.metadata,
    options: opts,
    ast: opts.ast === true ? file.ast : null,
    code: outputCode === undefined ? null : outputCode,
    map: outputMap === undefined ? null : outputMap,
    sourceType: file.ast.program.sourceType,
    externalDependencies: (0, _deepArray.flattenToSet)(config.externalDependencies)
  };
}
function* transformFile(file, pluginPasses) {
  for (const pluginPairs of pluginPasses) {
    const passPairs = [];
    const passes = [];
    const visitors = [];
    for (const plugin of pluginPairs.concat([(0, _blockHoistPlugin.default)()])) {
      const pass = new _pluginPass.default(file, plugin.key, plugin.options);
      passPairs.push([plugin, pass]);
      passes.push(pass);
      visitors.push(plugin.visitor);
    }
    for (const [plugin, pass] of passPairs) {
      const fn = plugin.pre;
      if (fn) {
        const result = fn.call(pass, file);
        yield* [];
        if (isThenable(result)) {
          throw new Error(`You appear to be using an plugin with an async .pre, ` + `which your current version of Babel does not support. ` + `If you're using a published plugin, you may need to upgrade ` + `your @babel/core version.`);
        }
      }
    }
    const visitor = _traverse().default.visitors.merge(visitors, passes, file.opts.wrapPluginVisitorMethod);
    {
      (0, _traverse().default)(file.ast, visitor, file.scope);
    }
    for (const [plugin, pass] of passPairs) {
      const fn = plugin.post;
      if (fn) {
        const result = fn.call(pass, file);
        yield* [];
        if (isThenable(result)) {
          throw new Error(`You appear to be using an plugin with an async .post, ` + `which your current version of Babel does not support. ` + `If you're using a published plugin, you may need to upgrade ` + `your @babel/core version.`);
        }
      }
    }
  }
}
function isThenable(val) {
  return !!val && (typeof val === "object" || typeof val === "function") && !!val.then && typeof val.then === "function";
}
0 && 0;

//# sourceMappingURL=index.js.map

}, function(modId) { var map = {"./plugin-pass.js":1781530614402,"./block-hoist-plugin.js":1781530614403,"./normalize-opts.js":1781530614404,"./normalize-file.js":1781530614405,"./file/generate.js":1781530614409,"../config/helpers/deep-array.js":1781530614387}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614402, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
class PluginPass {
  constructor(file, key, options) {
    this._map = new Map();
    this.key = void 0;
    this.file = void 0;
    this.opts = void 0;
    this.cwd = void 0;
    this.filename = void 0;
    this.key = key;
    this.file = file;
    this.opts = options || {};
    this.cwd = file.opts.cwd;
    this.filename = file.opts.filename;
  }
  set(key, val) {
    this._map.set(key, val);
  }
  get(key) {
    return this._map.get(key);
  }
  availableHelper(name, versionRange) {
    return this.file.availableHelper(name, versionRange);
  }
  addHelper(name) {
    return this.file.addHelper(name);
  }
  buildCodeFrameError(node, msg, _Error) {
    return this.file.buildCodeFrameError(node, msg, _Error);
  }
}
exports.default = PluginPass;
{
  PluginPass.prototype.getModuleName = function getModuleName() {
    return this.file.getModuleName();
  };
  PluginPass.prototype.addImport = function addImport() {
    this.file.addImport();
  };
}
0 && 0;

//# sourceMappingURL=plugin-pass.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614403, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = loadBlockHoistPlugin;
function _traverse() {
  const data = require("@babel/traverse");
  _traverse = function () {
    return data;
  };
  return data;
}
var _plugin = require("../config/plugin.js");
let LOADED_PLUGIN;
const blockHoistPlugin = {
  name: "internal.blockHoist",
  visitor: {
    Block: {
      exit({
        node
      }) {
        node.body = performHoisting(node.body);
      }
    },
    SwitchCase: {
      exit({
        node
      }) {
        node.consequent = performHoisting(node.consequent);
      }
    }
  }
};
function performHoisting(body) {
  let max = Math.pow(2, 30) - 1;
  let hasChange = false;
  for (let i = 0; i < body.length; i++) {
    const n = body[i];
    const p = priority(n);
    if (p > max) {
      hasChange = true;
      break;
    }
    max = p;
  }
  if (!hasChange) return body;
  return stableSort(body.slice());
}
function loadBlockHoistPlugin() {
  if (!LOADED_PLUGIN) {
    LOADED_PLUGIN = new _plugin.default(Object.assign({}, blockHoistPlugin, {
      visitor: _traverse().default.explode(blockHoistPlugin.visitor)
    }), {});
  }
  return LOADED_PLUGIN;
}
function priority(bodyNode) {
  const priority = bodyNode == null ? void 0 : bodyNode._blockHoist;
  if (priority == null) return 1;
  if (priority === true) return 2;
  return priority;
}
function stableSort(body) {
  const buckets = Object.create(null);
  for (let i = 0; i < body.length; i++) {
    const n = body[i];
    const p = priority(n);
    const bucket = buckets[p] || (buckets[p] = []);
    bucket.push(n);
  }
  const keys = Object.keys(buckets).map(k => +k).sort((a, b) => b - a);
  let index = 0;
  for (const key of keys) {
    const bucket = buckets[key];
    for (const n of bucket) {
      body[index++] = n;
    }
  }
  return body;
}
0 && 0;

//# sourceMappingURL=block-hoist-plugin.js.map

}, function(modId) { var map = {"../config/plugin.js":1781530614386}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614404, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = normalizeOptions;
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
function normalizeOptions(config) {
  const {
    filename,
    cwd,
    filenameRelative = typeof filename === "string" ? _path().relative(cwd, filename) : "unknown",
    sourceType = "module",
    inputSourceMap,
    sourceMaps = !!inputSourceMap,
    sourceRoot = config.options.moduleRoot,
    sourceFileName = _path().basename(filenameRelative),
    comments = true,
    compact = "auto"
  } = config.options;
  const opts = config.options;
  const options = Object.assign({}, opts, {
    parserOpts: Object.assign({
      sourceType: _path().extname(filenameRelative) === ".mjs" ? "module" : sourceType,
      sourceFileName: filename,
      plugins: []
    }, opts.parserOpts),
    generatorOpts: Object.assign({
      filename,
      auxiliaryCommentBefore: opts.auxiliaryCommentBefore,
      auxiliaryCommentAfter: opts.auxiliaryCommentAfter,
      retainLines: opts.retainLines,
      comments,
      shouldPrintComment: opts.shouldPrintComment,
      compact,
      minified: opts.minified,
      sourceMaps,
      sourceRoot,
      sourceFileName
    }, opts.generatorOpts)
  });
  for (const plugins of config.passes) {
    for (const plugin of plugins) {
      if (plugin.manipulateOptions) {
        plugin.manipulateOptions(options, options.parserOpts);
      }
    }
  }
  return options;
}
0 && 0;

//# sourceMappingURL=normalize-opts.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614405, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = normalizeFile;
function _fs() {
  const data = require("fs");
  _fs = function () {
    return data;
  };
  return data;
}
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
function _debug() {
  const data = require("debug");
  _debug = function () {
    return data;
  };
  return data;
}
function _t() {
  const data = require("@babel/types");
  _t = function () {
    return data;
  };
  return data;
}
function _convertSourceMap() {
  const data = require("convert-source-map");
  _convertSourceMap = function () {
    return data;
  };
  return data;
}
var _file = require("./file/file.js");
var _index = require("../parser/index.js");
var _cloneDeep = require("./util/clone-deep.js");
const {
  file,
  traverseFast
} = _t();
const debug = _debug()("babel:transform:file");
const INLINE_SOURCEMAP_REGEX = /^[@#]\s+sourceMappingURL=data:(?:application|text)\/json;(?:charset[:=]\S+?;)?base64,.*$/;
const EXTERNAL_SOURCEMAP_REGEX = /^[@#][ \t]+sourceMappingURL=([^\s'"`]+)[ \t]*$/;
function* normalizeFile(pluginPasses, options, code, ast) {
  code = `${code || ""}`;
  if (ast) {
    if (ast.type === "Program") {
      ast = file(ast, [], []);
    } else if (ast.type !== "File") {
      throw new Error("AST root must be a Program or File node");
    }
    if (options.cloneInputAst) {
      ast = (0, _cloneDeep.default)(ast);
    }
  } else {
    ast = yield* (0, _index.default)(pluginPasses, options, code);
  }
  let inputMap = null;
  if (options.inputSourceMap !== false) {
    if (typeof options.inputSourceMap === "object") {
      inputMap = _convertSourceMap().fromObject(options.inputSourceMap);
    }
    if (!inputMap) {
      const lastComment = extractComments(INLINE_SOURCEMAP_REGEX, ast);
      if (lastComment) {
        try {
          inputMap = _convertSourceMap().fromComment("//" + lastComment);
        } catch (err) {
          {
            debug("discarding unknown inline input sourcemap");
          }
        }
      }
    }
    if (!inputMap) {
      const lastComment = extractComments(EXTERNAL_SOURCEMAP_REGEX, ast);
      if (typeof options.filename === "string" && lastComment) {
        try {
          const match = EXTERNAL_SOURCEMAP_REGEX.exec(lastComment);
          const inputMapContent = _fs().readFileSync(_path().resolve(_path().dirname(options.filename), match[1]), "utf8");
          inputMap = _convertSourceMap().fromJSON(inputMapContent);
        } catch (err) {
          debug("discarding unknown file input sourcemap", err);
        }
      } else if (lastComment) {
        debug("discarding un-loadable file input sourcemap");
      }
    }
  }
  return new _file.default(options, {
    code,
    ast: ast,
    inputMap
  });
}
function extractCommentsFromList(regex, comments, lastComment) {
  if (comments) {
    comments = comments.filter(({
      value
    }) => {
      if (regex.test(value)) {
        lastComment = value;
        return false;
      }
      return true;
    });
  }
  return [comments, lastComment];
}
function extractComments(regex, ast) {
  let lastComment = null;
  traverseFast(ast, node => {
    [node.leadingComments, lastComment] = extractCommentsFromList(regex, node.leadingComments, lastComment);
    [node.innerComments, lastComment] = extractCommentsFromList(regex, node.innerComments, lastComment);
    [node.trailingComments, lastComment] = extractCommentsFromList(regex, node.trailingComments, lastComment);
  });
  return lastComment;
}
0 && 0;

//# sourceMappingURL=normalize-file.js.map

}, function(modId) { var map = {"./file/file.js":1781530614369,"../parser/index.js":1781530614406,"./util/clone-deep.js":1781530614408}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614406, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = parser;
function _parser() {
  const data = require("@babel/parser");
  _parser = function () {
    return data;
  };
  return data;
}
function _codeFrame() {
  const data = require("@babel/code-frame");
  _codeFrame = function () {
    return data;
  };
  return data;
}
var _missingPluginHelper = require("./util/missing-plugin-helper.js");
function* parser(pluginPasses, {
  parserOpts,
  highlightCode = true,
  filename = "unknown"
}, code) {
  try {
    const results = [];
    for (const plugins of pluginPasses) {
      for (const plugin of plugins) {
        const {
          parserOverride
        } = plugin;
        if (parserOverride) {
          const ast = parserOverride(code, parserOpts, _parser().parse);
          if (ast !== undefined) results.push(ast);
        }
      }
    }
    if (results.length === 0) {
      return (0, _parser().parse)(code, parserOpts);
    } else if (results.length === 1) {
      yield* [];
      if (typeof results[0].then === "function") {
        throw new Error(`You appear to be using an async parser plugin, ` + `which your current version of Babel does not support. ` + `If you're using a published plugin, you may need to upgrade ` + `your @babel/core version.`);
      }
      return results[0];
    }
    throw new Error("More than one plugin attempted to override parsing.");
  } catch (err) {
    if (err.code === "BABEL_PARSER_SOURCETYPE_MODULE_REQUIRED") {
      err.message += "\nConsider renaming the file to '.mjs', or setting sourceType:module " + "or sourceType:unambiguous in your Babel config for this file.";
    }
    const {
      loc,
      missingPlugin
    } = err;
    if (loc) {
      const codeFrame = (0, _codeFrame().codeFrameColumns)(code, {
        start: {
          line: loc.line,
          column: loc.column + 1
        }
      }, {
        highlightCode
      });
      if (missingPlugin) {
        err.message = `${filename}: ` + (0, _missingPluginHelper.default)(missingPlugin[0], loc, codeFrame, filename);
      } else {
        err.message = `${filename}: ${err.message}\n\n` + codeFrame;
      }
      err.code = "BABEL_PARSE_ERROR";
    }
    throw err;
  }
}
0 && 0;

//# sourceMappingURL=index.js.map

}, function(modId) { var map = {"./util/missing-plugin-helper.js":1781530614407}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614407, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = generateMissingPluginMessage;
const pluginNameMap = {
  asyncDoExpressions: {
    syntax: {
      name: "@babel/plugin-syntax-async-do-expressions",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-async-do-expressions"
    }
  },
  decimal: {
    syntax: {
      name: "@babel/plugin-syntax-decimal",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-decimal"
    }
  },
  decorators: {
    syntax: {
      name: "@babel/plugin-syntax-decorators",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-decorators"
    },
    transform: {
      name: "@babel/plugin-proposal-decorators",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-proposal-decorators"
    }
  },
  doExpressions: {
    syntax: {
      name: "@babel/plugin-syntax-do-expressions",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-do-expressions"
    },
    transform: {
      name: "@babel/plugin-proposal-do-expressions",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-proposal-do-expressions"
    }
  },
  exportDefaultFrom: {
    syntax: {
      name: "@babel/plugin-syntax-export-default-from",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-export-default-from"
    },
    transform: {
      name: "@babel/plugin-proposal-export-default-from",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-proposal-export-default-from"
    }
  },
  flow: {
    syntax: {
      name: "@babel/plugin-syntax-flow",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-flow"
    },
    transform: {
      name: "@babel/preset-flow",
      url: "https://github.com/babel/babel/tree/main/packages/babel-preset-flow"
    }
  },
  functionBind: {
    syntax: {
      name: "@babel/plugin-syntax-function-bind",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-function-bind"
    },
    transform: {
      name: "@babel/plugin-proposal-function-bind",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-proposal-function-bind"
    }
  },
  functionSent: {
    syntax: {
      name: "@babel/plugin-syntax-function-sent",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-function-sent"
    },
    transform: {
      name: "@babel/plugin-proposal-function-sent",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-proposal-function-sent"
    }
  },
  jsx: {
    syntax: {
      name: "@babel/plugin-syntax-jsx",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-jsx"
    },
    transform: {
      name: "@babel/preset-react",
      url: "https://github.com/babel/babel/tree/main/packages/babel-preset-react"
    }
  },
  importAttributes: {
    syntax: {
      name: "@babel/plugin-syntax-import-attributes",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-import-attributes"
    }
  },
  pipelineOperator: {
    syntax: {
      name: "@babel/plugin-syntax-pipeline-operator",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-pipeline-operator"
    },
    transform: {
      name: "@babel/plugin-proposal-pipeline-operator",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-proposal-pipeline-operator"
    }
  },
  recordAndTuple: {
    syntax: {
      name: "@babel/plugin-syntax-record-and-tuple",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-record-and-tuple"
    }
  },
  throwExpressions: {
    syntax: {
      name: "@babel/plugin-syntax-throw-expressions",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-throw-expressions"
    },
    transform: {
      name: "@babel/plugin-proposal-throw-expressions",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-proposal-throw-expressions"
    }
  },
  typescript: {
    syntax: {
      name: "@babel/plugin-syntax-typescript",
      url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-typescript"
    },
    transform: {
      name: "@babel/preset-typescript",
      url: "https://github.com/babel/babel/tree/main/packages/babel-preset-typescript"
    }
  }
};
{
  Object.assign(pluginNameMap, {
    asyncGenerators: {
      syntax: {
        name: "@babel/plugin-syntax-async-generators",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-async-generators"
      },
      transform: {
        name: "@babel/plugin-transform-async-generator-functions",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-async-generator-functions"
      }
    },
    classProperties: {
      syntax: {
        name: "@babel/plugin-syntax-class-properties",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-class-properties"
      },
      transform: {
        name: "@babel/plugin-transform-class-properties",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-class-properties"
      }
    },
    classPrivateProperties: {
      syntax: {
        name: "@babel/plugin-syntax-class-properties",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-class-properties"
      },
      transform: {
        name: "@babel/plugin-transform-class-properties",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-class-properties"
      }
    },
    classPrivateMethods: {
      syntax: {
        name: "@babel/plugin-syntax-class-properties",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-class-properties"
      },
      transform: {
        name: "@babel/plugin-transform-private-methods",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-private-methods"
      }
    },
    classStaticBlock: {
      syntax: {
        name: "@babel/plugin-syntax-class-static-block",
        url: "https://github.com/babel/babel/tree/HEAD/packages/babel-plugin-syntax-class-static-block"
      },
      transform: {
        name: "@babel/plugin-transform-class-static-block",
        url: "https://github.com/babel/babel/tree/HEAD/packages/babel-plugin-transform-class-static-block"
      }
    },
    dynamicImport: {
      syntax: {
        name: "@babel/plugin-syntax-dynamic-import",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-dynamic-import"
      }
    },
    exportNamespaceFrom: {
      syntax: {
        name: "@babel/plugin-syntax-export-namespace-from",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-export-namespace-from"
      },
      transform: {
        name: "@babel/plugin-transform-export-namespace-from",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-export-namespace-from"
      }
    },
    importAssertions: {
      syntax: {
        name: "@babel/plugin-syntax-import-assertions",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-import-assertions"
      }
    },
    importMeta: {
      syntax: {
        name: "@babel/plugin-syntax-import-meta",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-import-meta"
      }
    },
    logicalAssignment: {
      syntax: {
        name: "@babel/plugin-syntax-logical-assignment-operators",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-logical-assignment-operators"
      },
      transform: {
        name: "@babel/plugin-transform-logical-assignment-operators",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-logical-assignment-operators"
      }
    },
    moduleStringNames: {
      syntax: {
        name: "@babel/plugin-syntax-module-string-names",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-module-string-names"
      }
    },
    numericSeparator: {
      syntax: {
        name: "@babel/plugin-syntax-numeric-separator",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-numeric-separator"
      },
      transform: {
        name: "@babel/plugin-transform-numeric-separator",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-numeric-separator"
      }
    },
    nullishCoalescingOperator: {
      syntax: {
        name: "@babel/plugin-syntax-nullish-coalescing-operator",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-nullish-coalescing-operator"
      },
      transform: {
        name: "@babel/plugin-transform-nullish-coalescing-operator",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-nullish-coalescing-opearator"
      }
    },
    objectRestSpread: {
      syntax: {
        name: "@babel/plugin-syntax-object-rest-spread",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-object-rest-spread"
      },
      transform: {
        name: "@babel/plugin-transform-object-rest-spread",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-object-rest-spread"
      }
    },
    optionalCatchBinding: {
      syntax: {
        name: "@babel/plugin-syntax-optional-catch-binding",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-optional-catch-binding"
      },
      transform: {
        name: "@babel/plugin-transform-optional-catch-binding",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-optional-catch-binding"
      }
    },
    optionalChaining: {
      syntax: {
        name: "@babel/plugin-syntax-optional-chaining",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-optional-chaining"
      },
      transform: {
        name: "@babel/plugin-transform-optional-chaining",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-optional-chaining"
      }
    },
    privateIn: {
      syntax: {
        name: "@babel/plugin-syntax-private-property-in-object",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-syntax-private-property-in-object"
      },
      transform: {
        name: "@babel/plugin-transform-private-property-in-object",
        url: "https://github.com/babel/babel/tree/main/packages/babel-plugin-transform-private-property-in-object"
      }
    },
    regexpUnicodeSets: {
      syntax: {
        name: "@babel/plugin-syntax-unicode-sets-regex",
        url: "https://github.com/babel/babel/blob/main/packages/babel-plugin-syntax-unicode-sets-regex/README.md"
      },
      transform: {
        name: "@babel/plugin-transform-unicode-sets-regex",
        url: "https://github.com/babel/babel/blob/main/packages/babel-plugin-proposalunicode-sets-regex/README.md"
      }
    }
  });
}
const getNameURLCombination = ({
  name,
  url
}) => `${name} (${url})`;
function generateMissingPluginMessage(missingPluginName, loc, codeFrame, filename) {
  let helpMessage = `Support for the experimental syntax '${missingPluginName}' isn't currently enabled ` + `(${loc.line}:${loc.column + 1}):\n\n` + codeFrame;
  const pluginInfo = pluginNameMap[missingPluginName];
  if (pluginInfo) {
    const {
      syntax: syntaxPlugin,
      transform: transformPlugin
    } = pluginInfo;
    if (syntaxPlugin) {
      const syntaxPluginInfo = getNameURLCombination(syntaxPlugin);
      if (transformPlugin) {
        const transformPluginInfo = getNameURLCombination(transformPlugin);
        const sectionType = transformPlugin.name.startsWith("@babel/plugin") ? "plugins" : "presets";
        helpMessage += `\n\nAdd ${transformPluginInfo} to the '${sectionType}' section of your Babel config to enable transformation.
If you want to leave it as-is, add ${syntaxPluginInfo} to the 'plugins' section to enable parsing.`;
      } else {
        helpMessage += `\n\nAdd ${syntaxPluginInfo} to the 'plugins' section of your Babel config ` + `to enable parsing.`;
      }
    }
  }
  const msgFilename = filename === "unknown" ? "<name of the input file>" : filename;
  helpMessage += `

If you already added the plugin for this syntax to your config, it's possible that your config \
isn't being loaded.
You can re-run Babel with the BABEL_SHOW_CONFIG_FOR environment variable to show the loaded \
configuration:
\tnpx cross-env BABEL_SHOW_CONFIG_FOR=${msgFilename} <your build command>
See https://babeljs.io/docs/configuration#print-effective-configs for more info.
`;
  return helpMessage;
}
0 && 0;

//# sourceMappingURL=missing-plugin-helper.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614408, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = _default;
function deepClone(value, cache) {
  if (value !== null) {
    if (cache.has(value)) return cache.get(value);
    let cloned;
    if (Array.isArray(value)) {
      cloned = new Array(value.length);
      cache.set(value, cloned);
      for (let i = 0; i < value.length; i++) {
        cloned[i] = typeof value[i] !== "object" ? value[i] : deepClone(value[i], cache);
      }
    } else {
      cloned = {};
      cache.set(value, cloned);
      const keys = Object.keys(value);
      for (let i = 0; i < keys.length; i++) {
        const key = keys[i];
        cloned[key] = typeof value[key] !== "object" ? value[key] : deepClone(value[key], cache);
      }
    }
    return cloned;
  }
  return value;
}
function _default(value) {
  if (typeof value !== "object") return value;
  return deepClone(value, new Map());
}
0 && 0;

//# sourceMappingURL=clone-deep.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614409, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = generateCode;
function _convertSourceMap() {
  const data = require("convert-source-map");
  _convertSourceMap = function () {
    return data;
  };
  return data;
}
function _generator() {
  const data = require("@babel/generator");
  _generator = function () {
    return data;
  };
  return data;
}
var _mergeMap = require("./merge-map.js");
function generateCode(pluginPasses, file) {
  const {
    opts,
    ast,
    code,
    inputMap
  } = file;
  const {
    generatorOpts
  } = opts;
  generatorOpts.inputSourceMap = inputMap == null ? void 0 : inputMap.toObject();
  const results = [];
  for (const plugins of pluginPasses) {
    for (const plugin of plugins) {
      const {
        generatorOverride
      } = plugin;
      if (generatorOverride) {
        const result = generatorOverride(ast, generatorOpts, code, _generator().default);
        if (result !== undefined) results.push(result);
      }
    }
  }
  let result;
  if (results.length === 0) {
    result = (0, _generator().default)(ast, generatorOpts, code);
  } else if (results.length === 1) {
    result = results[0];
    if (typeof result.then === "function") {
      throw new Error(`You appear to be using an async codegen plugin, ` + `which your current version of Babel does not support. ` + `If you're using a published plugin, ` + `you may need to upgrade your @babel/core version.`);
    }
  } else {
    throw new Error("More than one plugin attempted to override codegen.");
  }
  let {
    code: outputCode,
    decodedMap: outputMap = result.map
  } = result;
  if (result.__mergedMap) {
    outputMap = Object.assign({}, result.map);
  } else {
    if (outputMap) {
      if (inputMap) {
        outputMap = (0, _mergeMap.default)(inputMap.toObject(), outputMap, generatorOpts.sourceFileName);
      } else {
        outputMap = result.map;
      }
    }
  }
  if (opts.sourceMaps === "inline" || opts.sourceMaps === "both") {
    outputCode += "\n" + _convertSourceMap().fromObject(outputMap).toComment();
  }
  if (opts.sourceMaps === "inline") {
    outputMap = null;
  }
  return {
    outputCode,
    outputMap
  };
}
0 && 0;

//# sourceMappingURL=generate.js.map

}, function(modId) { var map = {"./merge-map.js":1781530614410}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614410, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = mergeSourceMap;
function _remapping() {
  const data = require("@ampproject/remapping");
  _remapping = function () {
    return data;
  };
  return data;
}
function mergeSourceMap(inputMap, map, sourceFileName) {
  const source = sourceFileName.replace(/\\/g, "/");
  let found = false;
  const result = _remapping()(rootless(map), (s, ctx) => {
    if (s === source && !found) {
      found = true;
      ctx.source = "";
      return rootless(inputMap);
    }
    return null;
  });
  if (typeof inputMap.sourceRoot === "string") {
    result.sourceRoot = inputMap.sourceRoot;
  }
  return Object.assign({}, result);
}
function rootless(map) {
  return Object.assign({}, map, {
    sourceRoot: null
  });
}
0 && 0;

//# sourceMappingURL=merge-map.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614411, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.loadPlugin = loadPlugin;
exports.loadPreset = loadPreset;
exports.resolvePreset = exports.resolvePlugin = void 0;
function _debug() {
  const data = require("debug");
  _debug = function () {
    return data;
  };
  return data;
}
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
var _async = require("../../gensync-utils/async.js");
var _moduleTypes = require("./module-types.js");
function _url() {
  const data = require("url");
  _url = function () {
    return data;
  };
  return data;
}
var _importMetaResolve = require("../../vendor/import-meta-resolve.js");
function _fs() {
  const data = require("fs");
  _fs = function () {
    return data;
  };
  return data;
}
const debug = _debug()("babel:config:loading:files:plugins");
const EXACT_RE = /^module:/;
const BABEL_PLUGIN_PREFIX_RE = /^(?!@|module:|[^/]+\/|babel-plugin-)/;
const BABEL_PRESET_PREFIX_RE = /^(?!@|module:|[^/]+\/|babel-preset-)/;
const BABEL_PLUGIN_ORG_RE = /^(@babel\/)(?!plugin-|[^/]+\/)/;
const BABEL_PRESET_ORG_RE = /^(@babel\/)(?!preset-|[^/]+\/)/;
const OTHER_PLUGIN_ORG_RE = /^(@(?!babel\/)[^/]+\/)(?![^/]*babel-plugin(?:-|\/|$)|[^/]+\/)/;
const OTHER_PRESET_ORG_RE = /^(@(?!babel\/)[^/]+\/)(?![^/]*babel-preset(?:-|\/|$)|[^/]+\/)/;
const OTHER_ORG_DEFAULT_RE = /^(@(?!babel$)[^/]+)$/;
const resolvePlugin = exports.resolvePlugin = resolveStandardizedName.bind(null, "plugin");
const resolvePreset = exports.resolvePreset = resolveStandardizedName.bind(null, "preset");
function* loadPlugin(name, dirname) {
  const filepath = resolvePlugin(name, dirname, yield* (0, _async.isAsync)());
  const value = yield* requireModule("plugin", filepath);
  debug("Loaded plugin %o from %o.", name, dirname);
  return {
    filepath,
    value
  };
}
function* loadPreset(name, dirname) {
  const filepath = resolvePreset(name, dirname, yield* (0, _async.isAsync)());
  const value = yield* requireModule("preset", filepath);
  debug("Loaded preset %o from %o.", name, dirname);
  return {
    filepath,
    value
  };
}
function standardizeName(type, name) {
  if (_path().isAbsolute(name)) return name;
  const isPreset = type === "preset";
  return name.replace(isPreset ? BABEL_PRESET_PREFIX_RE : BABEL_PLUGIN_PREFIX_RE, `babel-${type}-`).replace(isPreset ? BABEL_PRESET_ORG_RE : BABEL_PLUGIN_ORG_RE, `$1${type}-`).replace(isPreset ? OTHER_PRESET_ORG_RE : OTHER_PLUGIN_ORG_RE, `$1babel-${type}-`).replace(OTHER_ORG_DEFAULT_RE, `$1/babel-${type}`).replace(EXACT_RE, "");
}
function* resolveAlternativesHelper(type, name) {
  const standardizedName = standardizeName(type, name);
  const {
    error,
    value
  } = yield standardizedName;
  if (!error) return value;
  if (error.code !== "MODULE_NOT_FOUND") throw error;
  if (standardizedName !== name && !(yield name).error) {
    error.message += `\n- If you want to resolve "${name}", use "module:${name}"`;
  }
  if (!(yield standardizeName(type, "@babel/" + name)).error) {
    error.message += `\n- Did you mean "@babel/${name}"?`;
  }
  const oppositeType = type === "preset" ? "plugin" : "preset";
  if (!(yield standardizeName(oppositeType, name)).error) {
    error.message += `\n- Did you accidentally pass a ${oppositeType} as a ${type}?`;
  }
  if (type === "plugin") {
    const transformName = standardizedName.replace("-proposal-", "-transform-");
    if (transformName !== standardizedName && !(yield transformName).error) {
      error.message += `\n- Did you mean "${transformName}"?`;
    }
  }
  error.message += `\n
Make sure that all the Babel plugins and presets you are using
are defined as dependencies or devDependencies in your package.json
file. It's possible that the missing plugin is loaded by a preset
you are using that forgot to add the plugin to its dependencies: you
can workaround this problem by explicitly adding the missing package
to your top-level package.json.
`;
  throw error;
}
function tryRequireResolve(id, dirname) {
  try {
    if (dirname) {
      return {
        error: null,
        value: (((v, w) => (v = v.split("."), w = w.split("."), +v[0] > +w[0] || v[0] == w[0] && +v[1] >= +w[1]))(process.versions.node, "8.9") ? require.resolve : (r, {
          paths: [b]
        }, M = require("module")) => {
          let f = M._findPath(r, M._nodeModulePaths(b).concat(b));
          if (f) return f;
          f = new Error(`Cannot resolve module '${r}'`);
          f.code = "MODULE_NOT_FOUND";
          throw f;
        })(id, {
          paths: [dirname]
        })
      };
    } else {
      return {
        error: null,
        value: require.resolve(id)
      };
    }
  } catch (error) {
    return {
      error,
      value: null
    };
  }
}
function tryImportMetaResolve(id, options) {
  try {
    return {
      error: null,
      value: (0, _importMetaResolve.resolve)(id, options)
    };
  } catch (error) {
    return {
      error,
      value: null
    };
  }
}
function resolveStandardizedNameForRequire(type, name, dirname) {
  const it = resolveAlternativesHelper(type, name);
  let res = it.next();
  while (!res.done) {
    res = it.next(tryRequireResolve(res.value, dirname));
  }
  return res.value;
}
function resolveStandardizedNameForImport(type, name, dirname) {
  const parentUrl = (0, _url().pathToFileURL)(_path().join(dirname, "./babel-virtual-resolve-base.js")).href;
  const it = resolveAlternativesHelper(type, name);
  let res = it.next();
  while (!res.done) {
    res = it.next(tryImportMetaResolve(res.value, parentUrl));
  }
  return (0, _url().fileURLToPath)(res.value);
}
function resolveStandardizedName(type, name, dirname, resolveESM) {
  if (!_moduleTypes.supportsESM || !resolveESM) {
    return resolveStandardizedNameForRequire(type, name, dirname);
  }
  try {
    const resolved = resolveStandardizedNameForImport(type, name, dirname);
    if (!(0, _fs().existsSync)(resolved)) {
      throw Object.assign(new Error(`Could not resolve "${name}" in file ${dirname}.`), {
        type: "MODULE_NOT_FOUND"
      });
    }
    return resolved;
  } catch (e) {
    try {
      return resolveStandardizedNameForRequire(type, name, dirname);
    } catch (e2) {
      if (e.type === "MODULE_NOT_FOUND") throw e;
      if (e2.type === "MODULE_NOT_FOUND") throw e2;
      throw e;
    }
  }
}
{
  var LOADING_MODULES = new Set();
}
function* requireModule(type, name) {
  {
    if (!(yield* (0, _async.isAsync)()) && LOADING_MODULES.has(name)) {
      throw new Error(`Reentrant ${type} detected trying to load "${name}". This module is not ignored ` + "and is trying to load itself while compiling itself, leading to a dependency cycle. " + 'We recommend adding it to your "ignore" list in your babelrc, or to a .babelignore.');
    }
  }
  try {
    {
      LOADING_MODULES.add(name);
    }
    {
      return yield* (0, _moduleTypes.default)(name, `You appear to be using a native ECMAScript module ${type}, ` + "which is only supported when running Babel asynchronously.", true);
    }
  } catch (err) {
    err.message = `[BABEL]: ${err.message} (While processing: ${name})`;
    throw err;
  } finally {
    {
      LOADING_MODULES.delete(name);
    }
  }
}
0 && 0;

//# sourceMappingURL=plugins.js.map

}, function(modId) { var map = {"../../gensync-utils/async.js":1781530614375,"./module-types.js":1781530614382,"../../vendor/import-meta-resolve.js":1781530614412}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614412, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.moduleResolve = moduleResolve;
exports.resolve = resolve;
function _assert() {
  const data = require("assert");
  _assert = function () {
    return data;
  };
  return data;
}
function _fs() {
  const data = _interopRequireWildcard(require("fs"), true);
  _fs = function () {
    return data;
  };
  return data;
}
function _process() {
  const data = require("process");
  _process = function () {
    return data;
  };
  return data;
}
function _url() {
  const data = require("url");
  _url = function () {
    return data;
  };
  return data;
}
function _path() {
  const data = require("path");
  _path = function () {
    return data;
  };
  return data;
}
function _module() {
  const data = require("module");
  _module = function () {
    return data;
  };
  return data;
}
function _v() {
  const data = require("v8");
  _v = function () {
    return data;
  };
  return data;
}
function _util() {
  const data = require("util");
  _util = function () {
    return data;
  };
  return data;
}
function _getRequireWildcardCache(e) { if ("function" != typeof WeakMap) return null; var r = new WeakMap(), t = new WeakMap(); return (_getRequireWildcardCache = function (e) { return e ? t : r; })(e); }
function _interopRequireWildcard(e, r) { if (!r && e && e.__esModule) return e; if (null === e || "object" != typeof e && "function" != typeof e) return { default: e }; var t = _getRequireWildcardCache(r); if (t && t.has(e)) return t.get(e); var n = { __proto__: null }, a = Object.defineProperty && Object.getOwnPropertyDescriptor; for (var u in e) if ("default" !== u && {}.hasOwnProperty.call(e, u)) { var i = a ? Object.getOwnPropertyDescriptor(e, u) : null; i && (i.get || i.set) ? Object.defineProperty(n, u, i) : n[u] = e[u]; } return n.default = e, t && t.set(e, n), n; }
const own$1 = {}.hasOwnProperty;
const classRegExp = /^([A-Z][a-z\d]*)+$/;
const kTypes = new Set(['string', 'function', 'number', 'object', 'Function', 'Object', 'boolean', 'bigint', 'symbol']);
const codes = {};
function formatList(array, type = 'and') {
  return array.length < 3 ? array.join(` ${type} `) : `${array.slice(0, -1).join(', ')}, ${type} ${array[array.length - 1]}`;
}
const messages = new Map();
const nodeInternalPrefix = '__node_internal_';
let userStackTraceLimit;
codes.ERR_INVALID_ARG_TYPE = createError('ERR_INVALID_ARG_TYPE', (name, expected, actual) => {
  _assert()(typeof name === 'string', "'name' must be a string");
  if (!Array.isArray(expected)) {
    expected = [expected];
  }
  let message = 'The ';
  if (name.endsWith(' argument')) {
    message += `${name} `;
  } else {
    const type = name.includes('.') ? 'property' : 'argument';
    message += `"${name}" ${type} `;
  }
  message += 'must be ';
  const types = [];
  const instances = [];
  const other = [];
  for (const value of expected) {
    _assert()(typeof value === 'string', 'All expected entries have to be of type string');
    if (kTypes.has(value)) {
      types.push(value.toLowerCase());
    } else if (classRegExp.exec(value) === null) {
      _assert()(value !== 'object', 'The value "object" should be written as "Object"');
      other.push(value);
    } else {
      instances.push(value);
    }
  }
  if (instances.length > 0) {
    const pos = types.indexOf('object');
    if (pos !== -1) {
      types.slice(pos, 1);
      instances.push('Object');
    }
  }
  if (types.length > 0) {
    message += `${types.length > 1 ? 'one of type' : 'of type'} ${formatList(types, 'or')}`;
    if (instances.length > 0 || other.length > 0) message += ' or ';
  }
  if (instances.length > 0) {
    message += `an instance of ${formatList(instances, 'or')}`;
    if (other.length > 0) message += ' or ';
  }
  if (other.length > 0) {
    if (other.length > 1) {
      message += `one of ${formatList(other, 'or')}`;
    } else {
      if (other[0].toLowerCase() !== other[0]) message += 'an ';
      message += `${other[0]}`;
    }
  }
  message += `. Received ${determineSpecificType(actual)}`;
  return message;
}, TypeError);
codes.ERR_INVALID_MODULE_SPECIFIER = createError('ERR_INVALID_MODULE_SPECIFIER', (request, reason, base = undefined) => {
  return `Invalid module "${request}" ${reason}${base ? ` imported from ${base}` : ''}`;
}, TypeError);
codes.ERR_INVALID_PACKAGE_CONFIG = createError('ERR_INVALID_PACKAGE_CONFIG', (path, base, message) => {
  return `Invalid package config ${path}${base ? ` while importing ${base}` : ''}${message ? `. ${message}` : ''}`;
}, Error);
codes.ERR_INVALID_PACKAGE_TARGET = createError('ERR_INVALID_PACKAGE_TARGET', (packagePath, key, target, isImport = false, base = undefined) => {
  const relatedError = typeof target === 'string' && !isImport && target.length > 0 && !target.startsWith('./');
  if (key === '.') {
    _assert()(isImport === false);
    return `Invalid "exports" main target ${JSON.stringify(target)} defined ` + `in the package config ${packagePath}package.json${base ? ` imported from ${base}` : ''}${relatedError ? '; targets must start with "./"' : ''}`;
  }
  return `Invalid "${isImport ? 'imports' : 'exports'}" target ${JSON.stringify(target)} defined for '${key}' in the package config ${packagePath}package.json${base ? ` imported from ${base}` : ''}${relatedError ? '; targets must start with "./"' : ''}`;
}, Error);
codes.ERR_MODULE_NOT_FOUND = createError('ERR_MODULE_NOT_FOUND', (path, base, exactUrl = false) => {
  return `Cannot find ${exactUrl ? 'module' : 'package'} '${path}' imported from ${base}`;
}, Error);
codes.ERR_NETWORK_IMPORT_DISALLOWED = createError('ERR_NETWORK_IMPORT_DISALLOWED', "import of '%s' by %s is not supported: %s", Error);
codes.ERR_PACKAGE_IMPORT_NOT_DEFINED = createError('ERR_PACKAGE_IMPORT_NOT_DEFINED', (specifier, packagePath, base) => {
  return `Package import specifier "${specifier}" is not defined${packagePath ? ` in package ${packagePath}package.json` : ''} imported from ${base}`;
}, TypeError);
codes.ERR_PACKAGE_PATH_NOT_EXPORTED = createError('ERR_PACKAGE_PATH_NOT_EXPORTED', (packagePath, subpath, base = undefined) => {
  if (subpath === '.') return `No "exports" main defined in ${packagePath}package.json${base ? ` imported from ${base}` : ''}`;
  return `Package subpath '${subpath}' is not defined by "exports" in ${packagePath}package.json${base ? ` imported from ${base}` : ''}`;
}, Error);
codes.ERR_UNSUPPORTED_DIR_IMPORT = createError('ERR_UNSUPPORTED_DIR_IMPORT', "Directory import '%s' is not supported " + 'resolving ES modules imported from %s', Error);
codes.ERR_UNSUPPORTED_RESOLVE_REQUEST = createError('ERR_UNSUPPORTED_RESOLVE_REQUEST', 'Failed to resolve module specifier "%s" from "%s": Invalid relative URL or base scheme is not hierarchical.', TypeError);
codes.ERR_UNKNOWN_FILE_EXTENSION = createError('ERR_UNKNOWN_FILE_EXTENSION', (extension, path) => {
  return `Unknown file extension "${extension}" for ${path}`;
}, TypeError);
codes.ERR_INVALID_ARG_VALUE = createError('ERR_INVALID_ARG_VALUE', (name, value, reason = 'is invalid') => {
  let inspected = (0, _util().inspect)(value);
  if (inspected.length > 128) {
    inspected = `${inspected.slice(0, 128)}...`;
  }
  const type = name.includes('.') ? 'property' : 'argument';
  return `The ${type} '${name}' ${reason}. Received ${inspected}`;
}, TypeError);
function createError(sym, value, constructor) {
  messages.set(sym, value);
  return makeNodeErrorWithCode(constructor, sym);
}
function makeNodeErrorWithCode(Base, key) {
  return NodeError;
  function NodeError(...parameters) {
    const limit = Error.stackTraceLimit;
    if (isErrorStackTraceLimitWritable()) Error.stackTraceLimit = 0;
    const error = new Base();
    if (isErrorStackTraceLimitWritable()) Error.stackTraceLimit = limit;
    const message = getMessage(key, parameters, error);
    Object.defineProperties(error, {
      message: {
        value: message,
        enumerable: false,
        writable: true,
        configurable: true
      },
      toString: {
        value() {
          return `${this.name} [${key}]: ${this.message}`;
        },
        enumerable: false,
        writable: true,
        configurable: true
      }
    });
    captureLargerStackTrace(error);
    error.code = key;
    return error;
  }
}
function isErrorStackTraceLimitWritable() {
  try {
    if (_v().startupSnapshot.isBuildingSnapshot()) {
      return false;
    }
  } catch (_unused) {}
  const desc = Object.getOwnPropertyDescriptor(Error, 'stackTraceLimit');
  if (desc === undefined) {
    return Object.isExtensible(Error);
  }
  return own$1.call(desc, 'writable') && desc.writable !== undefined ? desc.writable : desc.set !== undefined;
}
function hideStackFrames(wrappedFunction) {
  const hidden = nodeInternalPrefix + wrappedFunction.name;
  Object.defineProperty(wrappedFunction, 'name', {
    value: hidden
  });
  return wrappedFunction;
}
const captureLargerStackTrace = hideStackFrames(function (error) {
  const stackTraceLimitIsWritable = isErrorStackTraceLimitWritable();
  if (stackTraceLimitIsWritable) {
    userStackTraceLimit = Error.stackTraceLimit;
    Error.stackTraceLimit = Number.POSITIVE_INFINITY;
  }
  Error.captureStackTrace(error);
  if (stackTraceLimitIsWritable) Error.stackTraceLimit = userStackTraceLimit;
  return error;
});
function getMessage(key, parameters, self) {
  const message = messages.get(key);
  _assert()(message !== undefined, 'expected `message` to be found');
  if (typeof message === 'function') {
    _assert()(message.length <= parameters.length, `Code: ${key}; The provided arguments length (${parameters.length}) does not ` + `match the required ones (${message.length}).`);
    return Reflect.apply(message, self, parameters);
  }
  const regex = /%[dfijoOs]/g;
  let expectedLength = 0;
  while (regex.exec(message) !== null) expectedLength++;
  _assert()(expectedLength === parameters.length, `Code: ${key}; The provided arguments length (${parameters.length}) does not ` + `match the required ones (${expectedLength}).`);
  if (parameters.length === 0) return message;
  parameters.unshift(message);
  return Reflect.apply(_util().format, null, parameters);
}
function determineSpecificType(value) {
  if (value === null || value === undefined) {
    return String(value);
  }
  if (typeof value === 'function' && value.name) {
    return `function ${value.name}`;
  }
  if (typeof value === 'object') {
    if (value.constructor && value.constructor.name) {
      return `an instance of ${value.constructor.name}`;
    }
    return `${(0, _util().inspect)(value, {
      depth: -1
    })}`;
  }
  let inspected = (0, _util().inspect)(value, {
    colors: false
  });
  if (inspected.length > 28) {
    inspected = `${inspected.slice(0, 25)}...`;
  }
  return `type ${typeof value} (${inspected})`;
}
const hasOwnProperty$1 = {}.hasOwnProperty;
const {
  ERR_INVALID_PACKAGE_CONFIG: ERR_INVALID_PACKAGE_CONFIG$1
} = codes;
const cache = new Map();
function read(jsonPath, {
  base,
  specifier
}) {
  const existing = cache.get(jsonPath);
  if (existing) {
    return existing;
  }
  let string;
  try {
    string = _fs().default.readFileSync(_path().toNamespacedPath(jsonPath), 'utf8');
  } catch (error) {
    const exception = error;
    if (exception.code !== 'ENOENT') {
      throw exception;
    }
  }
  const result = {
    exists: false,
    pjsonPath: jsonPath,
    main: undefined,
    name: undefined,
    type: 'none',
    exports: undefined,
    imports: undefined
  };
  if (string !== undefined) {
    let parsed;
    try {
      parsed = JSON.parse(string);
    } catch (error_) {
      const cause = error_;
      const error = new ERR_INVALID_PACKAGE_CONFIG$1(jsonPath, (base ? `"${specifier}" from ` : '') + (0, _url().fileURLToPath)(base || specifier), cause.message);
      error.cause = cause;
      throw error;
    }
    result.exists = true;
    if (hasOwnProperty$1.call(parsed, 'name') && typeof parsed.name === 'string') {
      result.name = parsed.name;
    }
    if (hasOwnProperty$1.call(parsed, 'main') && typeof parsed.main === 'string') {
      result.main = parsed.main;
    }
    if (hasOwnProperty$1.call(parsed, 'exports')) {
      result.exports = parsed.exports;
    }
    if (hasOwnProperty$1.call(parsed, 'imports')) {
      result.imports = parsed.imports;
    }
    if (hasOwnProperty$1.call(parsed, 'type') && (parsed.type === 'commonjs' || parsed.type === 'module')) {
      result.type = parsed.type;
    }
  }
  cache.set(jsonPath, result);
  return result;
}
function getPackageScopeConfig(resolved) {
  let packageJSONUrl = new URL('package.json', resolved);
  while (true) {
    const packageJSONPath = packageJSONUrl.pathname;
    if (packageJSONPath.endsWith('node_modules/package.json')) {
      break;
    }
    const packageConfig = read((0, _url().fileURLToPath)(packageJSONUrl), {
      specifier: resolved
    });
    if (packageConfig.exists) {
      return packageConfig;
    }
    const lastPackageJSONUrl = packageJSONUrl;
    packageJSONUrl = new URL('../package.json', packageJSONUrl);
    if (packageJSONUrl.pathname === lastPackageJSONUrl.pathname) {
      break;
    }
  }
  const packageJSONPath = (0, _url().fileURLToPath)(packageJSONUrl);
  return {
    pjsonPath: packageJSONPath,
    exists: false,
    type: 'none'
  };
}
function getPackageType(url) {
  return getPackageScopeConfig(url).type;
}
const {
  ERR_UNKNOWN_FILE_EXTENSION
} = codes;
const hasOwnProperty = {}.hasOwnProperty;
const extensionFormatMap = {
  __proto__: null,
  '.cjs': 'commonjs',
  '.js': 'module',
  '.json': 'json',
  '.mjs': 'module'
};
function mimeToFormat(mime) {
  if (mime && /\s*(text|application)\/javascript\s*(;\s*charset=utf-?8\s*)?/i.test(mime)) return 'module';
  if (mime === 'application/json') return 'json';
  return null;
}
const protocolHandlers = {
  __proto__: null,
  'data:': getDataProtocolModuleFormat,
  'file:': getFileProtocolModuleFormat,
  'http:': getHttpProtocolModuleFormat,
  'https:': getHttpProtocolModuleFormat,
  'node:'() {
    return 'builtin';
  }
};
function getDataProtocolModuleFormat(parsed) {
  const {
    1: mime
  } = /^([^/]+\/[^;,]+)[^,]*?(;base64)?,/.exec(parsed.pathname) || [null, null, null];
  return mimeToFormat(mime);
}
function extname(url) {
  const pathname = url.pathname;
  let index = pathname.length;
  while (index--) {
    const code = pathname.codePointAt(index);
    if (code === 47) {
      return '';
    }
    if (code === 46) {
      return pathname.codePointAt(index - 1) === 47 ? '' : pathname.slice(index);
    }
  }
  return '';
}
function getFileProtocolModuleFormat(url, _context, ignoreErrors) {
  const value = extname(url);
  if (value === '.js') {
    const packageType = getPackageType(url);
    if (packageType !== 'none') {
      return packageType;
    }
    return 'commonjs';
  }
  if (value === '') {
    const packageType = getPackageType(url);
    if (packageType === 'none' || packageType === 'commonjs') {
      return 'commonjs';
    }
    return 'module';
  }
  const format = extensionFormatMap[value];
  if (format) return format;
  if (ignoreErrors) {
    return undefined;
  }
  const filepath = (0, _url().fileURLToPath)(url);
  throw new ERR_UNKNOWN_FILE_EXTENSION(value, filepath);
}
function getHttpProtocolModuleFormat() {}
function defaultGetFormatWithoutErrors(url, context) {
  const protocol = url.protocol;
  if (!hasOwnProperty.call(protocolHandlers, protocol)) {
    return null;
  }
  return protocolHandlers[protocol](url, context, true) || null;
}
const {
  ERR_INVALID_ARG_VALUE
} = codes;
const DEFAULT_CONDITIONS = Object.freeze(['node', 'import']);
const DEFAULT_CONDITIONS_SET = new Set(DEFAULT_CONDITIONS);
function getDefaultConditions() {
  return DEFAULT_CONDITIONS;
}
function getDefaultConditionsSet() {
  return DEFAULT_CONDITIONS_SET;
}
function getConditionsSet(conditions) {
  if (conditions !== undefined && conditions !== getDefaultConditions()) {
    if (!Array.isArray(conditions)) {
      throw new ERR_INVALID_ARG_VALUE('conditions', conditions, 'expected an array');
    }
    return new Set(conditions);
  }
  return getDefaultConditionsSet();
}
const RegExpPrototypeSymbolReplace = RegExp.prototype[Symbol.replace];
const {
  ERR_NETWORK_IMPORT_DISALLOWED,
  ERR_INVALID_MODULE_SPECIFIER,
  ERR_INVALID_PACKAGE_CONFIG,
  ERR_INVALID_PACKAGE_TARGET,
  ERR_MODULE_NOT_FOUND,
  ERR_PACKAGE_IMPORT_NOT_DEFINED,
  ERR_PACKAGE_PATH_NOT_EXPORTED,
  ERR_UNSUPPORTED_DIR_IMPORT,
  ERR_UNSUPPORTED_RESOLVE_REQUEST
} = codes;
const own = {}.hasOwnProperty;
const invalidSegmentRegEx = /(^|\\|\/)((\.|%2e)(\.|%2e)?|(n|%6e|%4e)(o|%6f|%4f)(d|%64|%44)(e|%65|%45)(_|%5f)(m|%6d|%4d)(o|%6f|%4f)(d|%64|%44)(u|%75|%55)(l|%6c|%4c)(e|%65|%45)(s|%73|%53))?(\\|\/|$)/i;
const deprecatedInvalidSegmentRegEx = /(^|\\|\/)((\.|%2e)(\.|%2e)?|(n|%6e|%4e)(o|%6f|%4f)(d|%64|%44)(e|%65|%45)(_|%5f)(m|%6d|%4d)(o|%6f|%4f)(d|%64|%44)(u|%75|%55)(l|%6c|%4c)(e|%65|%45)(s|%73|%53))(\\|\/|$)/i;
const invalidPackageNameRegEx = /^\.|%|\\/;
const patternRegEx = /\*/g;
const encodedSeparatorRegEx = /%2f|%5c/i;
const emittedPackageWarnings = new Set();
const doubleSlashRegEx = /[/\\]{2}/;
function emitInvalidSegmentDeprecation(target, request, match, packageJsonUrl, internal, base, isTarget) {
  if (_process().noDeprecation) {
    return;
  }
  const pjsonPath = (0, _url().fileURLToPath)(packageJsonUrl);
  const double = doubleSlashRegEx.exec(isTarget ? target : request) !== null;
  _process().emitWarning(`Use of deprecated ${double ? 'double slash' : 'leading or trailing slash matching'} resolving "${target}" for module ` + `request "${request}" ${request === match ? '' : `matched to "${match}" `}in the "${internal ? 'imports' : 'exports'}" field module resolution of the package at ${pjsonPath}${base ? ` imported from ${(0, _url().fileURLToPath)(base)}` : ''}.`, 'DeprecationWarning', 'DEP0166');
}
function emitLegacyIndexDeprecation(url, packageJsonUrl, base, main) {
  if (_process().noDeprecation) {
    return;
  }
  const format = defaultGetFormatWithoutErrors(url, {
    parentURL: base.href
  });
  if (format !== 'module') return;
  const urlPath = (0, _url().fileURLToPath)(url.href);
  const packagePath = (0, _url().fileURLToPath)(new (_url().URL)('.', packageJsonUrl));
  const basePath = (0, _url().fileURLToPath)(base);
  if (!main) {
    _process().emitWarning(`No "main" or "exports" field defined in the package.json for ${packagePath} resolving the main entry point "${urlPath.slice(packagePath.length)}", imported from ${basePath}.\nDefault "index" lookups for the main are deprecated for ES modules.`, 'DeprecationWarning', 'DEP0151');
  } else if (_path().resolve(packagePath, main) !== urlPath) {
    _process().emitWarning(`Package ${packagePath} has a "main" field set to "${main}", ` + `excluding the full filename and extension to the resolved file at "${urlPath.slice(packagePath.length)}", imported from ${basePath}.\n Automatic extension resolution of the "main" field is ` + 'deprecated for ES modules.', 'DeprecationWarning', 'DEP0151');
  }
}
function tryStatSync(path) {
  try {
    return (0, _fs().statSync)(path);
  } catch (_unused2) {}
}
function fileExists(url) {
  const stats = (0, _fs().statSync)(url, {
    throwIfNoEntry: false
  });
  const isFile = stats ? stats.isFile() : undefined;
  return isFile === null || isFile === undefined ? false : isFile;
}
function legacyMainResolve(packageJsonUrl, packageConfig, base) {
  let guess;
  if (packageConfig.main !== undefined) {
    guess = new (_url().URL)(packageConfig.main, packageJsonUrl);
    if (fileExists(guess)) return guess;
    const tries = [`./${packageConfig.main}.js`, `./${packageConfig.main}.json`, `./${packageConfig.main}.node`, `./${packageConfig.main}/index.js`, `./${packageConfig.main}/index.json`, `./${packageConfig.main}/index.node`];
    let i = -1;
    while (++i < tries.length) {
      guess = new (_url().URL)(tries[i], packageJsonUrl);
      if (fileExists(guess)) break;
      guess = undefined;
    }
    if (guess) {
      emitLegacyIndexDeprecation(guess, packageJsonUrl, base, packageConfig.main);
      return guess;
    }
  }
  const tries = ['./index.js', './index.json', './index.node'];
  let i = -1;
  while (++i < tries.length) {
    guess = new (_url().URL)(tries[i], packageJsonUrl);
    if (fileExists(guess)) break;
    guess = undefined;
  }
  if (guess) {
    emitLegacyIndexDeprecation(guess, packageJsonUrl, base, packageConfig.main);
    return guess;
  }
  throw new ERR_MODULE_NOT_FOUND((0, _url().fileURLToPath)(new (_url().URL)('.', packageJsonUrl)), (0, _url().fileURLToPath)(base));
}
function finalizeResolution(resolved, base, preserveSymlinks) {
  if (encodedSeparatorRegEx.exec(resolved.pathname) !== null) {
    throw new ERR_INVALID_MODULE_SPECIFIER(resolved.pathname, 'must not include encoded "/" or "\\" characters', (0, _url().fileURLToPath)(base));
  }
  let filePath;
  try {
    filePath = (0, _url().fileURLToPath)(resolved);
  } catch (error) {
    const cause = error;
    Object.defineProperty(cause, 'input', {
      value: String(resolved)
    });
    Object.defineProperty(cause, 'module', {
      value: String(base)
    });
    throw cause;
  }
  const stats = tryStatSync(filePath.endsWith('/') ? filePath.slice(-1) : filePath);
  if (stats && stats.isDirectory()) {
    const error = new ERR_UNSUPPORTED_DIR_IMPORT(filePath, (0, _url().fileURLToPath)(base));
    error.url = String(resolved);
    throw error;
  }
  if (!stats || !stats.isFile()) {
    const error = new ERR_MODULE_NOT_FOUND(filePath || resolved.pathname, base && (0, _url().fileURLToPath)(base), true);
    error.url = String(resolved);
    throw error;
  }
  if (!preserveSymlinks) {
    const real = (0, _fs().realpathSync)(filePath);
    const {
      search,
      hash
    } = resolved;
    resolved = (0, _url().pathToFileURL)(real + (filePath.endsWith(_path().sep) ? '/' : ''));
    resolved.search = search;
    resolved.hash = hash;
  }
  return resolved;
}
function importNotDefined(specifier, packageJsonUrl, base) {
  return new ERR_PACKAGE_IMPORT_NOT_DEFINED(specifier, packageJsonUrl && (0, _url().fileURLToPath)(new (_url().URL)('.', packageJsonUrl)), (0, _url().fileURLToPath)(base));
}
function exportsNotFound(subpath, packageJsonUrl, base) {
  return new ERR_PACKAGE_PATH_NOT_EXPORTED((0, _url().fileURLToPath)(new (_url().URL)('.', packageJsonUrl)), subpath, base && (0, _url().fileURLToPath)(base));
}
function throwInvalidSubpath(request, match, packageJsonUrl, internal, base) {
  const reason = `request is not a valid match in pattern "${match}" for the "${internal ? 'imports' : 'exports'}" resolution of ${(0, _url().fileURLToPath)(packageJsonUrl)}`;
  throw new ERR_INVALID_MODULE_SPECIFIER(request, reason, base && (0, _url().fileURLToPath)(base));
}
function invalidPackageTarget(subpath, target, packageJsonUrl, internal, base) {
  target = typeof target === 'object' && target !== null ? JSON.stringify(target, null, '') : `${target}`;
  return new ERR_INVALID_PACKAGE_TARGET((0, _url().fileURLToPath)(new (_url().URL)('.', packageJsonUrl)), subpath, target, internal, base && (0, _url().fileURLToPath)(base));
}
function resolvePackageTargetString(target, subpath, match, packageJsonUrl, base, pattern, internal, isPathMap, conditions) {
  if (subpath !== '' && !pattern && target[target.length - 1] !== '/') throw invalidPackageTarget(match, target, packageJsonUrl, internal, base);
  if (!target.startsWith('./')) {
    if (internal && !target.startsWith('../') && !target.startsWith('/')) {
      let isURL = false;
      try {
        new (_url().URL)(target);
        isURL = true;
      } catch (_unused3) {}
      if (!isURL) {
        const exportTarget = pattern ? RegExpPrototypeSymbolReplace.call(patternRegEx, target, () => subpath) : target + subpath;
        return packageResolve(exportTarget, packageJsonUrl, conditions);
      }
    }
    throw invalidPackageTarget(match, target, packageJsonUrl, internal, base);
  }
  if (invalidSegmentRegEx.exec(target.slice(2)) !== null) {
    if (deprecatedInvalidSegmentRegEx.exec(target.slice(2)) === null) {
      if (!isPathMap) {
        const request = pattern ? match.replace('*', () => subpath) : match + subpath;
        const resolvedTarget = pattern ? RegExpPrototypeSymbolReplace.call(patternRegEx, target, () => subpath) : target;
        emitInvalidSegmentDeprecation(resolvedTarget, request, match, packageJsonUrl, internal, base, true);
      }
    } else {
      throw invalidPackageTarget(match, target, packageJsonUrl, internal, base);
    }
  }
  const resolved = new (_url().URL)(target, packageJsonUrl);
  const resolvedPath = resolved.pathname;
  const packagePath = new (_url().URL)('.', packageJsonUrl).pathname;
  if (!resolvedPath.startsWith(packagePath)) throw invalidPackageTarget(match, target, packageJsonUrl, internal, base);
  if (subpath === '') return resolved;
  if (invalidSegmentRegEx.exec(subpath) !== null) {
    const request = pattern ? match.replace('*', () => subpath) : match + subpath;
    if (deprecatedInvalidSegmentRegEx.exec(subpath) === null) {
      if (!isPathMap) {
        const resolvedTarget = pattern ? RegExpPrototypeSymbolReplace.call(patternRegEx, target, () => subpath) : target;
        emitInvalidSegmentDeprecation(resolvedTarget, request, match, packageJsonUrl, internal, base, false);
      }
    } else {
      throwInvalidSubpath(request, match, packageJsonUrl, internal, base);
    }
  }
  if (pattern) {
    return new (_url().URL)(RegExpPrototypeSymbolReplace.call(patternRegEx, resolved.href, () => subpath));
  }
  return new (_url().URL)(subpath, resolved);
}
function isArrayIndex(key) {
  const keyNumber = Number(key);
  if (`${keyNumber}` !== key) return false;
  return keyNumber >= 0 && keyNumber < 0xffffffff;
}
function resolvePackageTarget(packageJsonUrl, target, subpath, packageSubpath, base, pattern, internal, isPathMap, conditions) {
  if (typeof target === 'string') {
    return resolvePackageTargetString(target, subpath, packageSubpath, packageJsonUrl, base, pattern, internal, isPathMap, conditions);
  }
  if (Array.isArray(target)) {
    const targetList = target;
    if (targetList.length === 0) return null;
    let lastException;
    let i = -1;
    while (++i < targetList.length) {
      const targetItem = targetList[i];
      let resolveResult;
      try {
        resolveResult = resolvePackageTarget(packageJsonUrl, targetItem, subpath, packageSubpath, base, pattern, internal, isPathMap, conditions);
      } catch (error) {
        const exception = error;
        lastException = exception;
        if (exception.code === 'ERR_INVALID_PACKAGE_TARGET') continue;
        throw error;
      }
      if (resolveResult === undefined) continue;
      if (resolveResult === null) {
        lastException = null;
        continue;
      }
      return resolveResult;
    }
    if (lastException === undefined || lastException === null) {
      return null;
    }
    throw lastException;
  }
  if (typeof target === 'object' && target !== null) {
    const keys = Object.getOwnPropertyNames(target);
    let i = -1;
    while (++i < keys.length) {
      const key = keys[i];
      if (isArrayIndex(key)) {
        throw new ERR_INVALID_PACKAGE_CONFIG((0, _url().fileURLToPath)(packageJsonUrl), base, '"exports" cannot contain numeric property keys.');
      }
    }
    i = -1;
    while (++i < keys.length) {
      const key = keys[i];
      if (key === 'default' || conditions && conditions.has(key)) {
        const conditionalTarget = target[key];
        const resolveResult = resolvePackageTarget(packageJsonUrl, conditionalTarget, subpath, packageSubpath, base, pattern, internal, isPathMap, conditions);
        if (resolveResult === undefined) continue;
        return resolveResult;
      }
    }
    return null;
  }
  if (target === null) {
    return null;
  }
  throw invalidPackageTarget(packageSubpath, target, packageJsonUrl, internal, base);
}
function isConditionalExportsMainSugar(exports, packageJsonUrl, base) {
  if (typeof exports === 'string' || Array.isArray(exports)) return true;
  if (typeof exports !== 'object' || exports === null) return false;
  const keys = Object.getOwnPropertyNames(exports);
  let isConditionalSugar = false;
  let i = 0;
  let keyIndex = -1;
  while (++keyIndex < keys.length) {
    const key = keys[keyIndex];
    const currentIsConditionalSugar = key === '' || key[0] !== '.';
    if (i++ === 0) {
      isConditionalSugar = currentIsConditionalSugar;
    } else if (isConditionalSugar !== currentIsConditionalSugar) {
      throw new ERR_INVALID_PACKAGE_CONFIG((0, _url().fileURLToPath)(packageJsonUrl), base, '"exports" cannot contain some keys starting with \'.\' and some not.' + ' The exports object must either be an object of package subpath keys' + ' or an object of main entry condition name keys only.');
    }
  }
  return isConditionalSugar;
}
function emitTrailingSlashPatternDeprecation(match, pjsonUrl, base) {
  if (_process().noDeprecation) {
    return;
  }
  const pjsonPath = (0, _url().fileURLToPath)(pjsonUrl);
  if (emittedPackageWarnings.has(pjsonPath + '|' + match)) return;
  emittedPackageWarnings.add(pjsonPath + '|' + match);
  _process().emitWarning(`Use of deprecated trailing slash pattern mapping "${match}" in the ` + `"exports" field module resolution of the package at ${pjsonPath}${base ? ` imported from ${(0, _url().fileURLToPath)(base)}` : ''}. Mapping specifiers ending in "/" is no longer supported.`, 'DeprecationWarning', 'DEP0155');
}
function packageExportsResolve(packageJsonUrl, packageSubpath, packageConfig, base, conditions) {
  let exports = packageConfig.exports;
  if (isConditionalExportsMainSugar(exports, packageJsonUrl, base)) {
    exports = {
      '.': exports
    };
  }
  if (own.call(exports, packageSubpath) && !packageSubpath.includes('*') && !packageSubpath.endsWith('/')) {
    const target = exports[packageSubpath];
    const resolveResult = resolvePackageTarget(packageJsonUrl, target, '', packageSubpath, base, false, false, false, conditions);
    if (resolveResult === null || resolveResult === undefined) {
      throw exportsNotFound(packageSubpath, packageJsonUrl, base);
    }
    return resolveResult;
  }
  let bestMatch = '';
  let bestMatchSubpath = '';
  const keys = Object.getOwnPropertyNames(exports);
  let i = -1;
  while (++i < keys.length) {
    const key = keys[i];
    const patternIndex = key.indexOf('*');
    if (patternIndex !== -1 && packageSubpath.startsWith(key.slice(0, patternIndex))) {
      if (packageSubpath.endsWith('/')) {
        emitTrailingSlashPatternDeprecation(packageSubpath, packageJsonUrl, base);
      }
      const patternTrailer = key.slice(patternIndex + 1);
      if (packageSubpath.length >= key.length && packageSubpath.endsWith(patternTrailer) && patternKeyCompare(bestMatch, key) === 1 && key.lastIndexOf('*') === patternIndex) {
        bestMatch = key;
        bestMatchSubpath = packageSubpath.slice(patternIndex, packageSubpath.length - patternTrailer.length);
      }
    }
  }
  if (bestMatch) {
    const target = exports[bestMatch];
    const resolveResult = resolvePackageTarget(packageJsonUrl, target, bestMatchSubpath, bestMatch, base, true, false, packageSubpath.endsWith('/'), conditions);
    if (resolveResult === null || resolveResult === undefined) {
      throw exportsNotFound(packageSubpath, packageJsonUrl, base);
    }
    return resolveResult;
  }
  throw exportsNotFound(packageSubpath, packageJsonUrl, base);
}
function patternKeyCompare(a, b) {
  const aPatternIndex = a.indexOf('*');
  const bPatternIndex = b.indexOf('*');
  const baseLengthA = aPatternIndex === -1 ? a.length : aPatternIndex + 1;
  const baseLengthB = bPatternIndex === -1 ? b.length : bPatternIndex + 1;
  if (baseLengthA > baseLengthB) return -1;
  if (baseLengthB > baseLengthA) return 1;
  if (aPatternIndex === -1) return 1;
  if (bPatternIndex === -1) return -1;
  if (a.length > b.length) return -1;
  if (b.length > a.length) return 1;
  return 0;
}
function packageImportsResolve(name, base, conditions) {
  if (name === '#' || name.startsWith('#/') || name.endsWith('/')) {
    const reason = 'is not a valid internal imports specifier name';
    throw new ERR_INVALID_MODULE_SPECIFIER(name, reason, (0, _url().fileURLToPath)(base));
  }
  let packageJsonUrl;
  const packageConfig = getPackageScopeConfig(base);
  if (packageConfig.exists) {
    packageJsonUrl = (0, _url().pathToFileURL)(packageConfig.pjsonPath);
    const imports = packageConfig.imports;
    if (imports) {
      if (own.call(imports, name) && !name.includes('*')) {
        const resolveResult = resolvePackageTarget(packageJsonUrl, imports[name], '', name, base, false, true, false, conditions);
        if (resolveResult !== null && resolveResult !== undefined) {
          return resolveResult;
        }
      } else {
        let bestMatch = '';
        let bestMatchSubpath = '';
        const keys = Object.getOwnPropertyNames(imports);
        let i = -1;
        while (++i < keys.length) {
          const key = keys[i];
          const patternIndex = key.indexOf('*');
          if (patternIndex !== -1 && name.startsWith(key.slice(0, -1))) {
            const patternTrailer = key.slice(patternIndex + 1);
            if (name.length >= key.length && name.endsWith(patternTrailer) && patternKeyCompare(bestMatch, key) === 1 && key.lastIndexOf('*') === patternIndex) {
              bestMatch = key;
              bestMatchSubpath = name.slice(patternIndex, name.length - patternTrailer.length);
            }
          }
        }
        if (bestMatch) {
          const target = imports[bestMatch];
          const resolveResult = resolvePackageTarget(packageJsonUrl, target, bestMatchSubpath, bestMatch, base, true, true, false, conditions);
          if (resolveResult !== null && resolveResult !== undefined) {
            return resolveResult;
          }
        }
      }
    }
  }
  throw importNotDefined(name, packageJsonUrl, base);
}
function parsePackageName(specifier, base) {
  let separatorIndex = specifier.indexOf('/');
  let validPackageName = true;
  let isScoped = false;
  if (specifier[0] === '@') {
    isScoped = true;
    if (separatorIndex === -1 || specifier.length === 0) {
      validPackageName = false;
    } else {
      separatorIndex = specifier.indexOf('/', separatorIndex + 1);
    }
  }
  const packageName = separatorIndex === -1 ? specifier : specifier.slice(0, separatorIndex);
  if (invalidPackageNameRegEx.exec(packageName) !== null) {
    validPackageName = false;
  }
  if (!validPackageName) {
    throw new ERR_INVALID_MODULE_SPECIFIER(specifier, 'is not a valid package name', (0, _url().fileURLToPath)(base));
  }
  const packageSubpath = '.' + (separatorIndex === -1 ? '' : specifier.slice(separatorIndex));
  return {
    packageName,
    packageSubpath,
    isScoped
  };
}
function packageResolve(specifier, base, conditions) {
  if (_module().builtinModules.includes(specifier)) {
    return new (_url().URL)('node:' + specifier);
  }
  const {
    packageName,
    packageSubpath,
    isScoped
  } = parsePackageName(specifier, base);
  const packageConfig = getPackageScopeConfig(base);
  if (packageConfig.exists) {
    const packageJsonUrl = (0, _url().pathToFileURL)(packageConfig.pjsonPath);
    if (packageConfig.name === packageName && packageConfig.exports !== undefined && packageConfig.exports !== null) {
      return packageExportsResolve(packageJsonUrl, packageSubpath, packageConfig, base, conditions);
    }
  }
  let packageJsonUrl = new (_url().URL)('./node_modules/' + packageName + '/package.json', base);
  let packageJsonPath = (0, _url().fileURLToPath)(packageJsonUrl);
  let lastPath;
  do {
    const stat = tryStatSync(packageJsonPath.slice(0, -13));
    if (!stat || !stat.isDirectory()) {
      lastPath = packageJsonPath;
      packageJsonUrl = new (_url().URL)((isScoped ? '../../../../node_modules/' : '../../../node_modules/') + packageName + '/package.json', packageJsonUrl);
      packageJsonPath = (0, _url().fileURLToPath)(packageJsonUrl);
      continue;
    }
    const packageConfig = read(packageJsonPath, {
      base,
      specifier
    });
    if (packageConfig.exports !== undefined && packageConfig.exports !== null) {
      return packageExportsResolve(packageJsonUrl, packageSubpath, packageConfig, base, conditions);
    }
    if (packageSubpath === '.') {
      return legacyMainResolve(packageJsonUrl, packageConfig, base);
    }
    return new (_url().URL)(packageSubpath, packageJsonUrl);
  } while (packageJsonPath.length !== lastPath.length);
  throw new ERR_MODULE_NOT_FOUND(packageName, (0, _url().fileURLToPath)(base), false);
}
function isRelativeSpecifier(specifier) {
  if (specifier[0] === '.') {
    if (specifier.length === 1 || specifier[1] === '/') return true;
    if (specifier[1] === '.' && (specifier.length === 2 || specifier[2] === '/')) {
      return true;
    }
  }
  return false;
}
function shouldBeTreatedAsRelativeOrAbsolutePath(specifier) {
  if (specifier === '') return false;
  if (specifier[0] === '/') return true;
  return isRelativeSpecifier(specifier);
}
function moduleResolve(specifier, base, conditions, preserveSymlinks) {
  const protocol = base.protocol;
  const isData = protocol === 'data:';
  const isRemote = isData || protocol === 'http:' || protocol === 'https:';
  let resolved;
  if (shouldBeTreatedAsRelativeOrAbsolutePath(specifier)) {
    try {
      resolved = new (_url().URL)(specifier, base);
    } catch (error_) {
      const error = new ERR_UNSUPPORTED_RESOLVE_REQUEST(specifier, base);
      error.cause = error_;
      throw error;
    }
  } else if (protocol === 'file:' && specifier[0] === '#') {
    resolved = packageImportsResolve(specifier, base, conditions);
  } else {
    try {
      resolved = new (_url().URL)(specifier);
    } catch (error_) {
      if (isRemote && !_module().builtinModules.includes(specifier)) {
        const error = new ERR_UNSUPPORTED_RESOLVE_REQUEST(specifier, base);
        error.cause = error_;
        throw error;
      }
      resolved = packageResolve(specifier, base, conditions);
    }
  }
  _assert()(resolved !== undefined, 'expected to be defined');
  if (resolved.protocol !== 'file:') {
    return resolved;
  }
  return finalizeResolution(resolved, base, preserveSymlinks);
}
function checkIfDisallowedImport(specifier, parsed, parsedParentURL) {
  if (parsedParentURL) {
    const parentProtocol = parsedParentURL.protocol;
    if (parentProtocol === 'http:' || parentProtocol === 'https:') {
      if (shouldBeTreatedAsRelativeOrAbsolutePath(specifier)) {
        const parsedProtocol = parsed == null ? void 0 : parsed.protocol;
        if (parsedProtocol && parsedProtocol !== 'https:' && parsedProtocol !== 'http:') {
          throw new ERR_NETWORK_IMPORT_DISALLOWED(specifier, parsedParentURL, 'remote imports cannot import from a local location.');
        }
        return {
          url: (parsed == null ? void 0 : parsed.href) || ''
        };
      }
      if (_module().builtinModules.includes(specifier)) {
        throw new ERR_NETWORK_IMPORT_DISALLOWED(specifier, parsedParentURL, 'remote imports cannot import from a local location.');
      }
      throw new ERR_NETWORK_IMPORT_DISALLOWED(specifier, parsedParentURL, 'only relative and absolute specifiers are supported.');
    }
  }
}
function isURL(self) {
  return Boolean(self && typeof self === 'object' && 'href' in self && typeof self.href === 'string' && 'protocol' in self && typeof self.protocol === 'string' && self.href && self.protocol);
}
function throwIfInvalidParentURL(parentURL) {
  if (parentURL === undefined) {
    return;
  }
  if (typeof parentURL !== 'string' && !isURL(parentURL)) {
    throw new codes.ERR_INVALID_ARG_TYPE('parentURL', ['string', 'URL'], parentURL);
  }
}
function defaultResolve(specifier, context = {}) {
  const {
    parentURL
  } = context;
  _assert()(parentURL !== undefined, 'expected `parentURL` to be defined');
  throwIfInvalidParentURL(parentURL);
  let parsedParentURL;
  if (parentURL) {
    try {
      parsedParentURL = new (_url().URL)(parentURL);
    } catch (_unused4) {}
  }
  let parsed;
  let protocol;
  try {
    parsed = shouldBeTreatedAsRelativeOrAbsolutePath(specifier) ? new (_url().URL)(specifier, parsedParentURL) : new (_url().URL)(specifier);
    protocol = parsed.protocol;
    if (protocol === 'data:') {
      return {
        url: parsed.href,
        format: null
      };
    }
  } catch (_unused5) {}
  const maybeReturn = checkIfDisallowedImport(specifier, parsed, parsedParentURL);
  if (maybeReturn) return maybeReturn;
  if (protocol === undefined && parsed) {
    protocol = parsed.protocol;
  }
  if (protocol === 'node:') {
    return {
      url: specifier
    };
  }
  if (parsed && parsed.protocol === 'node:') return {
    url: specifier
  };
  const conditions = getConditionsSet(context.conditions);
  const url = moduleResolve(specifier, new (_url().URL)(parentURL), conditions, false);
  return {
    url: url.href,
    format: defaultGetFormatWithoutErrors(url, {
      parentURL
    })
  };
}
function resolve(specifier, parent) {
  if (!parent) {
    throw new Error('Please pass `parent`: `import-meta-resolve` cannot ponyfill that');
  }
  try {
    return defaultResolve(specifier, {
      parentURL: parent
    }).url;
  } catch (error) {
    const exception = error;
    if ((exception.code === 'ERR_UNSUPPORTED_DIR_IMPORT' || exception.code === 'ERR_MODULE_NOT_FOUND') && typeof exception.url === 'string') {
      return exception.url;
    }
    throw error;
  }
}
0 && 0;

//# sourceMappingURL=import-meta-resolve.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614413, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.transform = void 0;
exports.transformAsync = transformAsync;
exports.transformSync = transformSync;
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
var _index = require("./config/index.js");
var _index2 = require("./transformation/index.js");
var _rewriteStackTrace = require("./errors/rewrite-stack-trace.js");
const transformRunner = _gensync()(function* transform(code, opts) {
  const config = yield* (0, _index.default)(opts);
  if (config === null) return null;
  return yield* (0, _index2.run)(config, code);
});
const transform = exports.transform = function transform(code, optsOrCallback, maybeCallback) {
  let opts;
  let callback;
  if (typeof optsOrCallback === "function") {
    callback = optsOrCallback;
    opts = undefined;
  } else {
    opts = optsOrCallback;
    callback = maybeCallback;
  }
  if (callback === undefined) {
    {
      return (0, _rewriteStackTrace.beginHiddenCallStack)(transformRunner.sync)(code, opts);
    }
  }
  (0, _rewriteStackTrace.beginHiddenCallStack)(transformRunner.errback)(code, opts, callback);
};
function transformSync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(transformRunner.sync)(...args);
}
function transformAsync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(transformRunner.async)(...args);
}
0 && 0;

//# sourceMappingURL=transform.js.map

}, function(modId) { var map = {"./config/index.js":1781530614384,"./transformation/index.js":1781530614401,"./errors/rewrite-stack-trace.js":1781530614379}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614414, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.transformFromAst = void 0;
exports.transformFromAstAsync = transformFromAstAsync;
exports.transformFromAstSync = transformFromAstSync;
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
var _index = require("./config/index.js");
var _index2 = require("./transformation/index.js");
var _rewriteStackTrace = require("./errors/rewrite-stack-trace.js");
const transformFromAstRunner = _gensync()(function* (ast, code, opts) {
  const config = yield* (0, _index.default)(opts);
  if (config === null) return null;
  if (!ast) throw new Error("No AST given");
  return yield* (0, _index2.run)(config, code, ast);
});
const transformFromAst = exports.transformFromAst = function transformFromAst(ast, code, optsOrCallback, maybeCallback) {
  let opts;
  let callback;
  if (typeof optsOrCallback === "function") {
    callback = optsOrCallback;
    opts = undefined;
  } else {
    opts = optsOrCallback;
    callback = maybeCallback;
  }
  if (callback === undefined) {
    {
      return (0, _rewriteStackTrace.beginHiddenCallStack)(transformFromAstRunner.sync)(ast, code, opts);
    }
  }
  (0, _rewriteStackTrace.beginHiddenCallStack)(transformFromAstRunner.errback)(ast, code, opts, callback);
};
function transformFromAstSync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(transformFromAstRunner.sync)(...args);
}
function transformFromAstAsync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(transformFromAstRunner.async)(...args);
}
0 && 0;

//# sourceMappingURL=transform-ast.js.map

}, function(modId) { var map = {"./config/index.js":1781530614384,"./transformation/index.js":1781530614401,"./errors/rewrite-stack-trace.js":1781530614379}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614415, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.parse = void 0;
exports.parseAsync = parseAsync;
exports.parseSync = parseSync;
function _gensync() {
  const data = require("gensync");
  _gensync = function () {
    return data;
  };
  return data;
}
var _index = require("./config/index.js");
var _index2 = require("./parser/index.js");
var _normalizeOpts = require("./transformation/normalize-opts.js");
var _rewriteStackTrace = require("./errors/rewrite-stack-trace.js");
const parseRunner = _gensync()(function* parse(code, opts) {
  const config = yield* (0, _index.default)(opts);
  if (config === null) {
    return null;
  }
  return yield* (0, _index2.default)(config.passes, (0, _normalizeOpts.default)(config), code);
});
const parse = exports.parse = function parse(code, opts, callback) {
  if (typeof opts === "function") {
    callback = opts;
    opts = undefined;
  }
  if (callback === undefined) {
    {
      return (0, _rewriteStackTrace.beginHiddenCallStack)(parseRunner.sync)(code, opts);
    }
  }
  (0, _rewriteStackTrace.beginHiddenCallStack)(parseRunner.errback)(code, opts, callback);
};
function parseSync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(parseRunner.sync)(...args);
}
function parseAsync(...args) {
  return (0, _rewriteStackTrace.beginHiddenCallStack)(parseRunner.async)(...args);
}
0 && 0;

//# sourceMappingURL=parse.js.map

}, function(modId) { var map = {"./config/index.js":1781530614384,"./parser/index.js":1781530614406,"./transformation/normalize-opts.js":1781530614404,"./errors/rewrite-stack-trace.js":1781530614379}; return __REQUIRE__(map[modId], modId); })
return __REQUIRE__(1781530614368);
})()
//miniprogram-npm-outsideDeps=["@babel/types","@babel/parser","@babel/traverse","@babel/template","@babel/helpers","@babel/code-frame","semver","./babel-7-helpers.cjs","@babel/generator","path","fs","gensync","debug","json5","module","url","./import.cjs","@babel/preset-typescript/package.json","@babel/preset-typescript","@babel/helper-compilation-targets","convert-source-map","@ampproject/remapping","assert","process","v8","util"]
//# sourceMappingURL=index.js.map