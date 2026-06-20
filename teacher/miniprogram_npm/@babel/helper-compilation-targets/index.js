module.exports = (function() {
var __MODS__ = {};
var __DEFINE__ = function(modId, func, req) { var m = { exports: {}, _tempexports: {} }; __MODS__[modId] = { status: 0, func: func, req: req, m: m }; };
var __REQUIRE__ = function(modId, source) { if(!__MODS__[modId]) return require(source); if(!__MODS__[modId].status) { var m = __MODS__[modId].m; m._exports = m._tempexports; var desp = Object.getOwnPropertyDescriptor(m, "exports"); if (desp && desp.configurable) Object.defineProperty(m, "exports", { set: function (val) { if(typeof val === "object" && val !== m._exports) { m._exports.__proto__ = val.__proto__; Object.keys(val).forEach(function (k) { m._exports[k] = val[k]; }); } m._tempexports = val }, get: function () { return m._tempexports; } }); __MODS__[modId].status = 1; __MODS__[modId].func(__MODS__[modId].req, m, m.exports); } return __MODS__[modId].m.exports; };
var __REQUIRE_WILDCARD__ = function(obj) { if(obj && obj.__esModule) { return obj; } else { var newObj = {}; if(obj != null) { for(var k in obj) { if (Object.prototype.hasOwnProperty.call(obj, k)) newObj[k] = obj[k]; } } newObj.default = obj; return newObj; } };
var __REQUIRE_DEFAULT__ = function(obj) { return obj && obj.__esModule ? obj.default : obj; };
__DEFINE__(1781362618296, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
Object.defineProperty(exports, "TargetNames", {
  enumerable: true,
  get: function () {
    return _options.TargetNames;
  }
});
exports.default = getTargets;
Object.defineProperty(exports, "filterItems", {
  enumerable: true,
  get: function () {
    return _filterItems.default;
  }
});
Object.defineProperty(exports, "getInclusionReasons", {
  enumerable: true,
  get: function () {
    return _debug.getInclusionReasons;
  }
});
exports.isBrowsersQueryValid = isBrowsersQueryValid;
Object.defineProperty(exports, "isRequired", {
  enumerable: true,
  get: function () {
    return _filterItems.isRequired;
  }
});
Object.defineProperty(exports, "prettifyTargets", {
  enumerable: true,
  get: function () {
    return _pretty.prettifyTargets;
  }
});
Object.defineProperty(exports, "unreleasedLabels", {
  enumerable: true,
  get: function () {
    return _targets.unreleasedLabels;
  }
});
var _browserslist = require("browserslist");
var _helperValidatorOption = require("@babel/helper-validator-option");
var _lruCache = require("lru-cache");
var _utils = require("./utils.js");
var _targets = require("./targets.js");
var _options = require("./options.js");
var _pretty = require("./pretty.js");
var _debug = require("./debug.js");
var _filterItems = require("./filter-items.js");
const browserModulesData = require("@babel/compat-data/native-modules");
const ESM_SUPPORT = browserModulesData["es6.module"];
const v = new _helperValidatorOption.OptionValidator("@babel/helper-compilation-targets");
function validateTargetNames(targets) {
  const validTargets = Object.keys(_options.TargetNames);
  for (const target of Object.keys(targets)) {
    if (!(target in _options.TargetNames)) {
      throw new Error(v.formatMessage(`'${target}' is not a valid target
- Did you mean '${(0, _helperValidatorOption.findSuggestion)(target, validTargets)}'?`));
    }
  }
  return targets;
}
function isBrowsersQueryValid(browsers) {
  return typeof browsers === "string" || Array.isArray(browsers) && browsers.every(b => typeof b === "string");
}
function validateBrowsers(browsers) {
  v.invariant(browsers === undefined || isBrowsersQueryValid(browsers), `'${String(browsers)}' is not a valid browserslist query`);
  return browsers;
}
function getLowestVersions(browsers) {
  return browsers.reduce((all, browser) => {
    const [browserName, browserVersion] = browser.split(" ");
    const target = _targets.browserNameMap[browserName];
    if (!target) {
      return all;
    }
    try {
      const splitVersion = browserVersion.split("-")[0].toLowerCase();
      const isSplitUnreleased = (0, _utils.isUnreleasedVersion)(splitVersion, target);
      if (!all[target]) {
        all[target] = isSplitUnreleased ? splitVersion : (0, _utils.semverify)(splitVersion);
        return all;
      }
      const version = all[target];
      const isUnreleased = (0, _utils.isUnreleasedVersion)(version, target);
      if (isUnreleased && isSplitUnreleased) {
        all[target] = (0, _utils.getLowestUnreleased)(version, splitVersion, target);
      } else if (isUnreleased) {
        all[target] = (0, _utils.semverify)(splitVersion);
      } else if (!isUnreleased && !isSplitUnreleased) {
        const parsedBrowserVersion = (0, _utils.semverify)(splitVersion);
        all[target] = (0, _utils.semverMin)(version, parsedBrowserVersion);
      }
    } catch (_) {}
    return all;
  }, {});
}
function outputDecimalWarning(decimalTargets) {
  if (!decimalTargets.length) {
    return;
  }
  console.warn("Warning, the following targets are using a decimal version:\n");
  decimalTargets.forEach(({
    target,
    value
  }) => console.warn(`  ${target}: ${value}`));
  console.warn(`
We recommend using a string for minor/patch versions to avoid numbers like 6.10
getting parsed as 6.1, which can lead to unexpected behavior.
`);
}
function semverifyTarget(target, value) {
  try {
    return (0, _utils.semverify)(value);
  } catch (_) {
    throw new Error(v.formatMessage(`'${value}' is not a valid value for 'targets.${target}'.`));
  }
}
function nodeTargetParser(value) {
  const parsed = value === true || value === "current" ? process.versions.node.split("-")[0] : semverifyTarget("node", value);
  return ["node", parsed];
}
function defaultTargetParser(target, value) {
  const version = (0, _utils.isUnreleasedVersion)(value, target) ? value.toLowerCase() : semverifyTarget(target, value);
  return [target, version];
}
function generateTargets(inputTargets) {
  const input = Object.assign({}, inputTargets);
  delete input.esmodules;
  delete input.browsers;
  return input;
}
function resolveTargets(queries, env) {
  const resolved = _browserslist(queries, {
    mobileToDesktop: true,
    env
  });
  return getLowestVersions(resolved);
}
const targetsCache = new _lruCache({
  max: 64
});
function resolveTargetsCached(queries, env) {
  const cacheKey = typeof queries === "string" ? queries : queries.join() + env;
  let cached = targetsCache.get(cacheKey);
  if (!cached) {
    cached = resolveTargets(queries, env);
    targetsCache.set(cacheKey, cached);
  }
  return Object.assign({}, cached);
}
function getTargets(inputTargets = {}, options = {}) {
  var _browsers, _browsers2;
  let {
    browsers,
    esmodules
  } = inputTargets;
  const {
    configPath = ".",
    onBrowserslistConfigFound
  } = options;
  validateBrowsers(browsers);
  const input = generateTargets(inputTargets);
  let targets = validateTargetNames(input);
  const shouldParseBrowsers = !!browsers;
  const hasTargets = shouldParseBrowsers || Object.keys(targets).length > 0;
  const shouldSearchForConfig = !options.ignoreBrowserslistConfig && !hasTargets;
  if (!browsers && shouldSearchForConfig) {
    browsers = process.env.BROWSERSLIST;
    if (!browsers) {
      const configFile = options.configFile || process.env.BROWSERSLIST_CONFIG || _browserslist.findConfigFile(configPath);
      if (configFile != null) {
        onBrowserslistConfigFound == null || onBrowserslistConfigFound(configFile);
        browsers = _browserslist.loadConfig({
          config: configFile,
          env: options.browserslistEnv
        });
      }
    }
    if (browsers == null) {
      browsers = [];
    }
  }
  if (esmodules && (esmodules !== "intersect" || !((_browsers = browsers) != null && _browsers.length))) {
    browsers = Object.keys(ESM_SUPPORT).map(browser => `${browser} >= ${ESM_SUPPORT[browser]}`).join(", ");
    esmodules = false;
  }
  if ((_browsers2 = browsers) != null && _browsers2.length) {
    const queryBrowsers = resolveTargetsCached(browsers, options.browserslistEnv);
    if (esmodules === "intersect") {
      for (const browser of Object.keys(queryBrowsers)) {
        if (browser !== "deno" && browser !== "ie") {
          const esmSupportVersion = ESM_SUPPORT[browser === "opera_mobile" ? "op_mob" : browser];
          if (esmSupportVersion) {
            const version = queryBrowsers[browser];
            queryBrowsers[browser] = (0, _utils.getHighestUnreleased)(version, (0, _utils.semverify)(esmSupportVersion), browser);
          } else {
            delete queryBrowsers[browser];
          }
        } else {
          delete queryBrowsers[browser];
        }
      }
    }
    targets = Object.assign(queryBrowsers, targets);
  }
  const result = {};
  const decimalWarnings = [];
  for (const target of Object.keys(targets).sort()) {
    const value = targets[target];
    if (typeof value === "number" && value % 1 !== 0) {
      decimalWarnings.push({
        target,
        value
      });
    }
    const [parsedTarget, parsedValue] = target === "node" ? nodeTargetParser(value) : defaultTargetParser(target, value);
    if (parsedValue) {
      result[parsedTarget] = parsedValue;
    }
  }
  outputDecimalWarning(decimalWarnings);
  return result;
}

//# sourceMappingURL=index.js.map

}, function(modId) {var map = {"./utils.js":1781362618297,"./targets.js":1781362618298,"./options.js":1781362618299,"./pretty.js":1781362618300,"./debug.js":1781362618301,"./filter-items.js":1781362618302}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618297, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.getHighestUnreleased = getHighestUnreleased;
exports.getLowestImplementedVersion = getLowestImplementedVersion;
exports.getLowestUnreleased = getLowestUnreleased;
exports.isUnreleasedVersion = isUnreleasedVersion;
exports.semverMin = semverMin;
exports.semverify = semverify;
var _semver = require("semver");
var _helperValidatorOption = require("@babel/helper-validator-option");
var _targets = require("./targets.js");
const versionRegExp = /^(?:\d+|\d(?:\d?[^\d\n\r\u2028\u2029]\d+|\d{2,}(?:[^\d\n\r\u2028\u2029]\d+)?))$/;
const v = new _helperValidatorOption.OptionValidator("@babel/helper-compilation-targets");
function semverMin(first, second) {
  return first && _semver.lt(first, second) ? first : second;
}
function semverify(version) {
  if (typeof version === "string" && _semver.valid(version)) {
    return version;
  }
  v.invariant(typeof version === "number" || typeof version === "string" && versionRegExp.test(version), `'${version}' is not a valid version`);
  version = version.toString();
  let pos = 0;
  let num = 0;
  while ((pos = version.indexOf(".", pos + 1)) > 0) {
    num++;
  }
  return version + ".0".repeat(2 - num);
}
function isUnreleasedVersion(version, env) {
  const unreleasedLabel = _targets.unreleasedLabels[env];
  return !!unreleasedLabel && unreleasedLabel === version.toString().toLowerCase();
}
function getLowestUnreleased(a, b, env) {
  const unreleasedLabel = _targets.unreleasedLabels[env];
  if (a === unreleasedLabel) {
    return b;
  }
  if (b === unreleasedLabel) {
    return a;
  }
  return semverMin(a, b);
}
function getHighestUnreleased(a, b, env) {
  return getLowestUnreleased(a, b, env) === a ? b : a;
}
function getLowestImplementedVersion(plugin, environment) {
  const result = plugin[environment];
  if (!result && environment === "android") {
    return plugin.chrome;
  }
  return result;
}

//# sourceMappingURL=utils.js.map

}, function(modId) { var map = {"./targets.js":1781362618298}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618298, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.unreleasedLabels = exports.browserNameMap = void 0;
const unreleasedLabels = exports.unreleasedLabels = {
  safari: "tp"
};
const browserNameMap = exports.browserNameMap = {
  and_chr: "chrome",
  and_ff: "firefox",
  android: "android",
  chrome: "chrome",
  edge: "edge",
  firefox: "firefox",
  ie: "ie",
  ie_mob: "ie",
  ios_saf: "ios",
  node: "node",
  deno: "deno",
  op_mob: "opera_mobile",
  opera: "opera",
  safari: "safari",
  samsung: "samsung"
};

//# sourceMappingURL=targets.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618299, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.TargetNames = void 0;
const TargetNames = exports.TargetNames = {
  node: "node",
  deno: "deno",
  chrome: "chrome",
  opera: "opera",
  edge: "edge",
  firefox: "firefox",
  safari: "safari",
  ie: "ie",
  ios: "ios",
  android: "android",
  electron: "electron",
  samsung: "samsung",
  rhino: "rhino",
  opera_mobile: "opera_mobile"
};

//# sourceMappingURL=options.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618300, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.prettifyTargets = prettifyTargets;
exports.prettifyVersion = prettifyVersion;
var _semver = require("semver");
var _targets = require("./targets.js");
function prettifyVersion(version) {
  if (typeof version !== "string") {
    return version;
  }
  const {
    major,
    minor,
    patch
  } = _semver.parse(version);
  const parts = [major];
  if (minor || patch) {
    parts.push(minor);
  }
  if (patch) {
    parts.push(patch);
  }
  return parts.join(".");
}
function prettifyTargets(targets) {
  return Object.keys(targets).reduce((results, target) => {
    let value = targets[target];
    const unreleasedLabel = _targets.unreleasedLabels[target];
    if (typeof value === "string" && unreleasedLabel !== value) {
      value = prettifyVersion(value);
    }
    results[target] = value;
    return results;
  }, {});
}

//# sourceMappingURL=pretty.js.map

}, function(modId) { var map = {"./targets.js":1781362618298}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618301, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.getInclusionReasons = getInclusionReasons;
var _semver = require("semver");
var _pretty = require("./pretty.js");
var _utils = require("./utils.js");
function getInclusionReasons(item, targetVersions, list) {
  const minVersions = list[item] || {};
  return Object.keys(targetVersions).reduce((result, env) => {
    const minVersion = (0, _utils.getLowestImplementedVersion)(minVersions, env);
    const targetVersion = targetVersions[env];
    if (!minVersion) {
      result[env] = (0, _pretty.prettifyVersion)(targetVersion);
    } else {
      const minIsUnreleased = (0, _utils.isUnreleasedVersion)(minVersion, env);
      const targetIsUnreleased = (0, _utils.isUnreleasedVersion)(targetVersion, env);
      if (!targetIsUnreleased && (minIsUnreleased || _semver.lt(targetVersion.toString(), (0, _utils.semverify)(minVersion)))) {
        result[env] = (0, _pretty.prettifyVersion)(targetVersion);
      }
    }
    return result;
  }, {});
}

//# sourceMappingURL=debug.js.map

}, function(modId) { var map = {"./pretty.js":1781362618300,"./utils.js":1781362618297}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781362618302, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = filterItems;
exports.isRequired = isRequired;
exports.targetsSupported = targetsSupported;
var _semver = require("semver");
var _utils = require("./utils.js");
const pluginsCompatData = require("@babel/compat-data/plugins");
function targetsSupported(target, support) {
  const targetEnvironments = Object.keys(target);
  if (targetEnvironments.length === 0) {
    return false;
  }
  const unsupportedEnvironments = targetEnvironments.filter(environment => {
    const lowestImplementedVersion = (0, _utils.getLowestImplementedVersion)(support, environment);
    if (!lowestImplementedVersion) {
      return true;
    }
    const lowestTargetedVersion = target[environment];
    if ((0, _utils.isUnreleasedVersion)(lowestTargetedVersion, environment)) {
      return false;
    }
    if ((0, _utils.isUnreleasedVersion)(lowestImplementedVersion, environment)) {
      return true;
    }
    if (!_semver.valid(lowestTargetedVersion.toString())) {
      throw new Error(`Invalid version passed for target "${environment}": "${lowestTargetedVersion}". ` + "Versions must be in semver format (major.minor.patch)");
    }
    return _semver.gt((0, _utils.semverify)(lowestImplementedVersion), lowestTargetedVersion.toString());
  });
  return unsupportedEnvironments.length === 0;
}
function isRequired(name, targets, {
  compatData = pluginsCompatData,
  includes,
  excludes
} = {}) {
  if (excludes != null && excludes.has(name)) return false;
  if (includes != null && includes.has(name)) return true;
  return !targetsSupported(targets, compatData[name]);
}
function filterItems(list, includes, excludes, targets, defaultIncludes, defaultExcludes, pluginSyntaxMap) {
  const result = new Set();
  const options = {
    compatData: list,
    includes,
    excludes
  };
  for (const item in list) {
    if (isRequired(item, targets, options)) {
      result.add(item);
    } else if (pluginSyntaxMap) {
      const shippedProposalsSyntax = pluginSyntaxMap.get(item);
      if (shippedProposalsSyntax) {
        result.add(shippedProposalsSyntax);
      }
    }
  }
  defaultIncludes == null || defaultIncludes.forEach(item => !excludes.has(item) && result.add(item));
  defaultExcludes == null || defaultExcludes.forEach(item => !includes.has(item) && result.delete(item));
  return result;
}

//# sourceMappingURL=filter-items.js.map

}, function(modId) { var map = {"./utils.js":1781362618297}; return __REQUIRE__(map[modId], modId); })
return __REQUIRE__(1781362618296);
})()
//miniprogram-npm-outsideDeps=["browserslist","@babel/helper-validator-option","lru-cache","@babel/compat-data/native-modules","semver","@babel/compat-data/plugins"]
//# sourceMappingURL=index.js.map