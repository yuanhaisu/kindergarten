module.exports = (function() {
var __MODS__ = {};
var __DEFINE__ = function(modId, func, req) { var m = { exports: {}, _tempexports: {} }; __MODS__[modId] = { status: 0, func: func, req: req, m: m }; };
var __REQUIRE__ = function(modId, source) { if(!__MODS__[modId]) return require(source); if(!__MODS__[modId].status) { var m = __MODS__[modId].m; m._exports = m._tempexports; var desp = Object.getOwnPropertyDescriptor(m, "exports"); if (desp && desp.configurable) Object.defineProperty(m, "exports", { set: function (val) { if(typeof val === "object" && val !== m._exports) { m._exports.__proto__ = val.__proto__; Object.keys(val).forEach(function (k) { m._exports[k] = val[k]; }); } m._tempexports = val }, get: function () { return m._tempexports; } }); __MODS__[modId].status = 1; __MODS__[modId].func(__MODS__[modId].req, m, m.exports); } return __MODS__[modId].m.exports; };
var __REQUIRE_WILDCARD__ = function(obj) { if(obj && obj.__esModule) { return obj; } else { var newObj = {}; if(obj != null) { for(var k in obj) { if (Object.prototype.hasOwnProperty.call(obj, k)) newObj[k] = obj[k]; } } newObj.default = obj; return newObj; } };
var __REQUIRE_DEFAULT__ = function(obj) { return obj && obj.__esModule ? obj.default : obj; };
__DEFINE__(1781530614617, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
Object.defineProperty(exports, "buildDynamicImport", {
  enumerable: true,
  get: function () {
    return _dynamicImport.buildDynamicImport;
  }
});
exports.buildNamespaceInitStatements = buildNamespaceInitStatements;
exports.ensureStatementsHoisted = ensureStatementsHoisted;
Object.defineProperty(exports, "getModuleName", {
  enumerable: true,
  get: function () {
    return _getModuleName.default;
  }
});
Object.defineProperty(exports, "hasExports", {
  enumerable: true,
  get: function () {
    return _normalizeAndLoadMetadata.hasExports;
  }
});
Object.defineProperty(exports, "isModule", {
  enumerable: true,
  get: function () {
    return _helperModuleImports.isModule;
  }
});
Object.defineProperty(exports, "isSideEffectImport", {
  enumerable: true,
  get: function () {
    return _normalizeAndLoadMetadata.isSideEffectImport;
  }
});
exports.rewriteModuleStatementsAndPrepareHeader = rewriteModuleStatementsAndPrepareHeader;
Object.defineProperty(exports, "rewriteThis", {
  enumerable: true,
  get: function () {
    return _rewriteThis.default;
  }
});
exports.wrapInterop = wrapInterop;
var _assert = require("assert");
var _core = require("@babel/core");
var _helperModuleImports = require("@babel/helper-module-imports");
var _rewriteThis = require("./rewrite-this.js");
var _rewriteLiveReferences = require("./rewrite-live-references.js");
var _normalizeAndLoadMetadata = require("./normalize-and-load-metadata.js");
var Lazy = require("./lazy-modules.js");
var _dynamicImport = require("./dynamic-import.js");
var _getModuleName = require("./get-module-name.js");
exports.getDynamicImportSource = require("./dynamic-import").getDynamicImportSource;
function rewriteModuleStatementsAndPrepareHeader(path, {
  exportName,
  strict,
  allowTopLevelThis,
  strictMode,
  noInterop,
  importInterop = noInterop ? "none" : "babel",
  lazy,
  getWrapperPayload = Lazy.toGetWrapperPayload(lazy != null ? lazy : false),
  wrapReference = Lazy.wrapReference,
  esNamespaceOnly,
  filename,
  constantReexports = arguments[1].loose,
  enumerableModuleMeta = arguments[1].loose,
  noIncompleteNsImportDetection
}) {
  (0, _normalizeAndLoadMetadata.validateImportInteropOption)(importInterop);
  _assert((0, _helperModuleImports.isModule)(path), "Cannot process module statements in a script");
  path.node.sourceType = "script";
  const meta = (0, _normalizeAndLoadMetadata.default)(path, exportName, {
    importInterop,
    initializeReexports: constantReexports,
    getWrapperPayload,
    esNamespaceOnly,
    filename
  });
  if (!allowTopLevelThis) {
    (0, _rewriteThis.default)(path);
  }
  (0, _rewriteLiveReferences.default)(path, meta, wrapReference);
  if (strictMode !== false) {
    const hasStrict = path.node.directives.some(directive => {
      return directive.value.value === "use strict";
    });
    if (!hasStrict) {
      path.unshiftContainer("directives", _core.types.directive(_core.types.directiveLiteral("use strict")));
    }
  }
  const headers = [];
  if ((0, _normalizeAndLoadMetadata.hasExports)(meta) && !strict) {
    headers.push(buildESModuleHeader(meta, enumerableModuleMeta));
  }
  const nameList = buildExportNameListDeclaration(path, meta);
  if (nameList) {
    meta.exportNameListName = nameList.name;
    headers.push(nameList.statement);
  }
  headers.push(...buildExportInitializationStatements(path, meta, wrapReference, constantReexports, noIncompleteNsImportDetection));
  return {
    meta,
    headers
  };
}
function ensureStatementsHoisted(statements) {
  statements.forEach(header => {
    header._blockHoist = 3;
  });
}
function wrapInterop(programPath, expr, type) {
  if (type === "none") {
    return null;
  }
  if (type === "node-namespace") {
    return _core.types.callExpression(programPath.hub.addHelper("interopRequireWildcard"), [expr, _core.types.booleanLiteral(true)]);
  } else if (type === "node-default") {
    return null;
  }
  let helper;
  if (type === "default") {
    helper = "interopRequireDefault";
  } else if (type === "namespace") {
    helper = "interopRequireWildcard";
  } else {
    throw new Error(`Unknown interop: ${type}`);
  }
  return _core.types.callExpression(programPath.hub.addHelper(helper), [expr]);
}
function buildNamespaceInitStatements(metadata, sourceMetadata, constantReexports = false, wrapReference = Lazy.wrapReference) {
  var _wrapReference;
  const statements = [];
  const srcNamespaceId = _core.types.identifier(sourceMetadata.name);
  for (const localName of sourceMetadata.importsNamespace) {
    if (localName === sourceMetadata.name) continue;
    statements.push(_core.template.statement`var NAME = SOURCE;`({
      NAME: localName,
      SOURCE: _core.types.cloneNode(srcNamespaceId)
    }));
  }
  const srcNamespace = (_wrapReference = wrapReference(srcNamespaceId, sourceMetadata.wrap)) != null ? _wrapReference : srcNamespaceId;
  if (constantReexports) {
    statements.push(...buildReexportsFromMeta(metadata, sourceMetadata, true, wrapReference));
  }
  for (const exportName of sourceMetadata.reexportNamespace) {
    statements.push((!_core.types.isIdentifier(srcNamespace) ? _core.template.statement`
            Object.defineProperty(EXPORTS, "NAME", {
              enumerable: true,
              get: function() {
                return NAMESPACE;
              }
            });
          ` : _core.template.statement`EXPORTS.NAME = NAMESPACE;`)({
      EXPORTS: metadata.exportName,
      NAME: exportName,
      NAMESPACE: _core.types.cloneNode(srcNamespace)
    }));
  }
  if (sourceMetadata.reexportAll) {
    const statement = buildNamespaceReexport(metadata, _core.types.cloneNode(srcNamespace), constantReexports);
    statement.loc = sourceMetadata.reexportAll.loc;
    statements.push(statement);
  }
  return statements;
}
const ReexportTemplate = {
  constant: ({
    exports,
    exportName,
    namespaceImport
  }) => _core.template.statement.ast`
      ${exports}.${exportName} = ${namespaceImport};
    `,
  constantComputed: ({
    exports,
    exportName,
    namespaceImport
  }) => _core.template.statement.ast`
      ${exports}["${exportName}"] = ${namespaceImport};
    `,
  spec: ({
    exports,
    exportName,
    namespaceImport
  }) => _core.template.statement.ast`
      Object.defineProperty(${exports}, "${exportName}", {
        enumerable: true,
        get: function() {
          return ${namespaceImport};
        },
      });
    `
};
function buildReexportsFromMeta(meta, metadata, constantReexports, wrapReference) {
  var _wrapReference2;
  let namespace = _core.types.identifier(metadata.name);
  namespace = (_wrapReference2 = wrapReference(namespace, metadata.wrap)) != null ? _wrapReference2 : namespace;
  const {
    stringSpecifiers
  } = meta;
  return Array.from(metadata.reexports, ([exportName, importName]) => {
    let namespaceImport = _core.types.cloneNode(namespace);
    if (importName === "default" && metadata.interop === "node-default") {} else if (stringSpecifiers.has(importName)) {
      namespaceImport = _core.types.memberExpression(namespaceImport, _core.types.stringLiteral(importName), true);
    } else {
      namespaceImport = _core.types.memberExpression(namespaceImport, _core.types.identifier(importName));
    }
    const astNodes = {
      exports: meta.exportName,
      exportName,
      namespaceImport
    };
    if (constantReexports || _core.types.isIdentifier(namespaceImport)) {
      if (stringSpecifiers.has(exportName)) {
        return ReexportTemplate.constantComputed(astNodes);
      } else {
        return ReexportTemplate.constant(astNodes);
      }
    } else {
      return ReexportTemplate.spec(astNodes);
    }
  });
}
function buildESModuleHeader(metadata, enumerableModuleMeta = false) {
  return (enumerableModuleMeta ? _core.template.statement`
        EXPORTS.__esModule = true;
      ` : _core.template.statement`
        Object.defineProperty(EXPORTS, "__esModule", {
          value: true,
        });
      `)({
    EXPORTS: metadata.exportName
  });
}
function buildNamespaceReexport(metadata, namespace, constantReexports) {
  return (constantReexports ? _core.template.statement`
        Object.keys(NAMESPACE).forEach(function(key) {
          if (key === "default" || key === "__esModule") return;
          VERIFY_NAME_LIST;
          if (key in EXPORTS && EXPORTS[key] === NAMESPACE[key]) return;

          EXPORTS[key] = NAMESPACE[key];
        });
      ` : _core.template.statement`
        Object.keys(NAMESPACE).forEach(function(key) {
          if (key === "default" || key === "__esModule") return;
          VERIFY_NAME_LIST;
          if (key in EXPORTS && EXPORTS[key] === NAMESPACE[key]) return;

          Object.defineProperty(EXPORTS, key, {
            enumerable: true,
            get: function() {
              return NAMESPACE[key];
            },
          });
        });
    `)({
    NAMESPACE: namespace,
    EXPORTS: metadata.exportName,
    VERIFY_NAME_LIST: metadata.exportNameListName ? (0, _core.template)`
            if (Object.prototype.hasOwnProperty.call(EXPORTS_LIST, key)) return;
          `({
      EXPORTS_LIST: metadata.exportNameListName
    }) : null
  });
}
function buildExportNameListDeclaration(programPath, metadata) {
  const exportedVars = Object.create(null);
  for (const data of metadata.local.values()) {
    for (const name of data.names) {
      exportedVars[name] = true;
    }
  }
  let hasReexport = false;
  for (const data of metadata.source.values()) {
    for (const exportName of data.reexports.keys()) {
      exportedVars[exportName] = true;
    }
    for (const exportName of data.reexportNamespace) {
      exportedVars[exportName] = true;
    }
    hasReexport = hasReexport || !!data.reexportAll;
  }
  if (!hasReexport || Object.keys(exportedVars).length === 0) return null;
  const name = programPath.scope.generateUidIdentifier("exportNames");
  delete exportedVars.default;
  return {
    name: name.name,
    statement: _core.types.variableDeclaration("var", [_core.types.variableDeclarator(name, _core.types.valueToNode(exportedVars))])
  };
}
function buildExportInitializationStatements(programPath, metadata, wrapReference, constantReexports = false, noIncompleteNsImportDetection = false) {
  const initStatements = [];
  for (const [localName, data] of metadata.local) {
    if (data.kind === "import") {} else if (data.kind === "hoisted") {
      initStatements.push([data.names[0], buildInitStatement(metadata, data.names, _core.types.identifier(localName))]);
    } else if (!noIncompleteNsImportDetection) {
      for (const exportName of data.names) {
        initStatements.push([exportName, null]);
      }
    }
  }
  for (const data of metadata.source.values()) {
    if (!constantReexports) {
      const reexportsStatements = buildReexportsFromMeta(metadata, data, false, wrapReference);
      const reexports = [...data.reexports.keys()];
      for (let i = 0; i < reexportsStatements.length; i++) {
        initStatements.push([reexports[i], reexportsStatements[i]]);
      }
    }
    if (!noIncompleteNsImportDetection) {
      for (const exportName of data.reexportNamespace) {
        initStatements.push([exportName, null]);
      }
    }
  }
  initStatements.sort(([a], [b]) => {
    if (a < b) return -1;
    if (b < a) return 1;
    return 0;
  });
  const results = [];
  if (noIncompleteNsImportDetection) {
    for (const [, initStatement] of initStatements) {
      results.push(initStatement);
    }
  } else {
    const chunkSize = 100;
    for (let i = 0; i < initStatements.length; i += chunkSize) {
      let uninitializedExportNames = [];
      for (let j = 0; j < chunkSize && i + j < initStatements.length; j++) {
        const [exportName, initStatement] = initStatements[i + j];
        if (initStatement !== null) {
          if (uninitializedExportNames.length > 0) {
            results.push(buildInitStatement(metadata, uninitializedExportNames, programPath.scope.buildUndefinedNode()));
            uninitializedExportNames = [];
          }
          results.push(initStatement);
        } else {
          uninitializedExportNames.push(exportName);
        }
      }
      if (uninitializedExportNames.length > 0) {
        results.push(buildInitStatement(metadata, uninitializedExportNames, programPath.scope.buildUndefinedNode()));
      }
    }
  }
  return results;
}
const InitTemplate = {
  computed: ({
    exports,
    name,
    value
  }) => _core.template.expression.ast`${exports}["${name}"] = ${value}`,
  default: ({
    exports,
    name,
    value
  }) => _core.template.expression.ast`${exports}.${name} = ${value}`,
  define: ({
    exports,
    name,
    value
  }) => _core.template.expression.ast`
      Object.defineProperty(${exports}, "${name}", {
        enumerable: true,
        value: void 0,
        writable: true
      })["${name}"] = ${value}`
};
function buildInitStatement(metadata, exportNames, initExpr) {
  const {
    stringSpecifiers,
    exportName: exports
  } = metadata;
  return _core.types.expressionStatement(exportNames.reduce((value, name) => {
    const params = {
      exports,
      name,
      value
    };
    if (name === "__proto__") {
      return InitTemplate.define(params);
    }
    if (stringSpecifiers.has(name)) {
      return InitTemplate.computed(params);
    }
    return InitTemplate.default(params);
  }, initExpr));
}

//# sourceMappingURL=index.js.map

}, function(modId) {var map = {"./rewrite-this.js":1781530614618,"./rewrite-live-references.js":1781530614619,"./normalize-and-load-metadata.js":1781530614620,"./lazy-modules.js":1781530614621,"./dynamic-import.js":1781530614622,"./get-module-name.js":1781530614623,"./dynamic-import":1781530614622}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614618, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = rewriteThis;
var _core = require("@babel/core");
var _traverse = require("@babel/traverse");
let rewriteThisVisitor;
function rewriteThis(programPath) {
  if (!rewriteThisVisitor) {
    rewriteThisVisitor = _traverse.visitors.environmentVisitor({
      ThisExpression(path) {
        path.replaceWith(_core.types.unaryExpression("void", _core.types.numericLiteral(0), true));
      }
    });
    rewriteThisVisitor.noScope = true;
  }
  (0, _traverse.default)(programPath.node, rewriteThisVisitor);
}

//# sourceMappingURL=rewrite-this.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614619, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = rewriteLiveReferences;
var _core = require("@babel/core");
function isInType(path) {
  do {
    switch (path.parent.type) {
      case "TSTypeAnnotation":
      case "TSTypeAliasDeclaration":
      case "TSTypeReference":
      case "TypeAnnotation":
      case "TypeAlias":
        return true;
      case "ExportSpecifier":
        return path.parentPath.parent.exportKind === "type";
      default:
        if (path.parentPath.isStatement() || path.parentPath.isExpression()) {
          return false;
        }
    }
  } while (path = path.parentPath);
}
function rewriteLiveReferences(programPath, metadata, wrapReference) {
  const imported = new Map();
  const exported = new Map();
  const requeueInParent = path => {
    programPath.requeue(path);
  };
  for (const [source, data] of metadata.source) {
    for (const [localName, importName] of data.imports) {
      imported.set(localName, [source, importName, null]);
    }
    for (const localName of data.importsNamespace) {
      imported.set(localName, [source, null, localName]);
    }
  }
  for (const [local, data] of metadata.local) {
    let exportMeta = exported.get(local);
    if (!exportMeta) {
      exportMeta = [];
      exported.set(local, exportMeta);
    }
    exportMeta.push(...data.names);
  }
  const rewriteBindingInitVisitorState = {
    metadata,
    requeueInParent,
    scope: programPath.scope,
    exported
  };
  programPath.traverse(rewriteBindingInitVisitor, rewriteBindingInitVisitorState);
  const rewriteReferencesVisitorState = {
    seen: new WeakSet(),
    metadata,
    requeueInParent,
    scope: programPath.scope,
    imported,
    exported,
    buildImportReference([source, importName, localName], identNode) {
      const meta = metadata.source.get(source);
      meta.referenced = true;
      if (localName) {
        if (meta.wrap) {
          var _wrapReference;
          identNode = (_wrapReference = wrapReference(identNode, meta.wrap)) != null ? _wrapReference : identNode;
        }
        return identNode;
      }
      let namespace = _core.types.identifier(meta.name);
      if (meta.wrap) {
        var _wrapReference2;
        namespace = (_wrapReference2 = wrapReference(namespace, meta.wrap)) != null ? _wrapReference2 : namespace;
      }
      if (importName === "default" && meta.interop === "node-default") {
        return namespace;
      }
      const computed = metadata.stringSpecifiers.has(importName);
      return _core.types.memberExpression(namespace, computed ? _core.types.stringLiteral(importName) : _core.types.identifier(importName), computed);
    }
  };
  programPath.traverse(rewriteReferencesVisitor, rewriteReferencesVisitorState);
}
const rewriteBindingInitVisitor = {
  Scope(path) {
    path.skip();
  },
  ClassDeclaration(path) {
    const {
      requeueInParent,
      exported,
      metadata
    } = this;
    const {
      id
    } = path.node;
    if (!id) throw new Error("Expected class to have a name");
    const localName = id.name;
    const exportNames = exported.get(localName) || [];
    if (exportNames.length > 0) {
      const statement = _core.types.expressionStatement(buildBindingExportAssignmentExpression(metadata, exportNames, _core.types.identifier(localName), path.scope));
      statement._blockHoist = path.node._blockHoist;
      requeueInParent(path.insertAfter(statement)[0]);
    }
  },
  VariableDeclaration(path) {
    const {
      requeueInParent,
      exported,
      metadata
    } = this;
    const isVar = path.node.kind === "var";
    for (const decl of path.get("declarations")) {
      const {
        id
      } = decl.node;
      let {
        init
      } = decl.node;
      if (_core.types.isIdentifier(id) && exported.has(id.name) && !_core.types.isArrowFunctionExpression(init) && (!_core.types.isFunctionExpression(init) || init.id) && (!_core.types.isClassExpression(init) || init.id)) {
        if (!init) {
          if (isVar) {
            continue;
          } else {
            init = path.scope.buildUndefinedNode();
          }
        }
        decl.node.init = buildBindingExportAssignmentExpression(metadata, exported.get(id.name), init, path.scope);
        requeueInParent(decl.get("init"));
      } else {
        for (const localName of Object.keys(decl.getOuterBindingIdentifiers())) {
          if (exported.has(localName)) {
            const statement = _core.types.expressionStatement(buildBindingExportAssignmentExpression(metadata, exported.get(localName), _core.types.identifier(localName), path.scope));
            statement._blockHoist = path.node._blockHoist;
            requeueInParent(path.insertAfter(statement)[0]);
          }
        }
      }
    }
  }
};
const buildBindingExportAssignmentExpression = (metadata, exportNames, localExpr, scope) => {
  const exportsObjectName = metadata.exportName;
  for (let currentScope = scope; currentScope != null; currentScope = currentScope.parent) {
    if (currentScope.hasOwnBinding(exportsObjectName)) {
      currentScope.rename(exportsObjectName);
    }
  }
  return (exportNames || []).reduce((expr, exportName) => {
    const {
      stringSpecifiers
    } = metadata;
    const computed = stringSpecifiers.has(exportName);
    return _core.types.assignmentExpression("=", _core.types.memberExpression(_core.types.identifier(exportsObjectName), computed ? _core.types.stringLiteral(exportName) : _core.types.identifier(exportName), computed), expr);
  }, localExpr);
};
const buildImportThrow = localName => {
  return _core.template.expression.ast`
    (function() {
      throw new Error('"' + '${localName}' + '" is read-only.');
    })()
  `;
};
const rewriteReferencesVisitor = {
  ReferencedIdentifier(path) {
    const {
      seen,
      buildImportReference,
      scope,
      imported,
      requeueInParent
    } = this;
    if (seen.has(path.node)) return;
    seen.add(path.node);
    const localName = path.node.name;
    const importData = imported.get(localName);
    if (importData) {
      if (isInType(path)) {
        throw path.buildCodeFrameError(`Cannot transform the imported binding "${localName}" since it's also used in a type annotation. ` + `Please strip type annotations using @babel/preset-typescript or @babel/preset-flow.`);
      }
      const localBinding = path.scope.getBinding(localName);
      const rootBinding = scope.getBinding(localName);
      if (rootBinding !== localBinding) return;
      const ref = buildImportReference(importData, path.node);
      ref.loc = path.node.loc;
      if ((path.parentPath.isCallExpression({
        callee: path.node
      }) || path.parentPath.isOptionalCallExpression({
        callee: path.node
      }) || path.parentPath.isTaggedTemplateExpression({
        tag: path.node
      })) && _core.types.isMemberExpression(ref)) {
        path.replaceWith(_core.types.sequenceExpression([_core.types.numericLiteral(0), ref]));
      } else if (path.isJSXIdentifier() && _core.types.isMemberExpression(ref)) {
        const {
          object,
          property
        } = ref;
        path.replaceWith(_core.types.jsxMemberExpression(_core.types.jsxIdentifier(object.name), _core.types.jsxIdentifier(property.name)));
      } else {
        path.replaceWith(ref);
      }
      requeueInParent(path);
      path.skip();
    }
  },
  UpdateExpression(path) {
    const {
      scope,
      seen,
      imported,
      exported,
      requeueInParent,
      buildImportReference
    } = this;
    if (seen.has(path.node)) return;
    seen.add(path.node);
    const arg = path.get("argument");
    if (arg.isMemberExpression()) return;
    const update = path.node;
    if (arg.isIdentifier()) {
      const localName = arg.node.name;
      if (scope.getBinding(localName) !== path.scope.getBinding(localName)) {
        return;
      }
      const exportedNames = exported.get(localName);
      const importData = imported.get(localName);
      if ((exportedNames == null ? void 0 : exportedNames.length) > 0 || importData) {
        if (importData) {
          path.replaceWith(_core.types.assignmentExpression(update.operator[0] + "=", buildImportReference(importData, arg.node), buildImportThrow(localName)));
        } else if (update.prefix) {
          path.replaceWith(buildBindingExportAssignmentExpression(this.metadata, exportedNames, _core.types.cloneNode(update), path.scope));
        } else {
          const ref = scope.generateDeclaredUidIdentifier(localName);
          path.replaceWith(_core.types.sequenceExpression([_core.types.assignmentExpression("=", _core.types.cloneNode(ref), _core.types.cloneNode(update)), buildBindingExportAssignmentExpression(this.metadata, exportedNames, _core.types.identifier(localName), path.scope), _core.types.cloneNode(ref)]));
        }
      }
    }
    requeueInParent(path);
    path.skip();
  },
  AssignmentExpression: {
    exit(path) {
      const {
        scope,
        seen,
        imported,
        exported,
        requeueInParent,
        buildImportReference
      } = this;
      if (seen.has(path.node)) return;
      seen.add(path.node);
      const left = path.get("left");
      if (left.isMemberExpression()) return;
      if (left.isIdentifier()) {
        const localName = left.node.name;
        if (scope.getBinding(localName) !== path.scope.getBinding(localName)) {
          return;
        }
        const exportedNames = exported.get(localName);
        const importData = imported.get(localName);
        if ((exportedNames == null ? void 0 : exportedNames.length) > 0 || importData) {
          const assignment = path.node;
          if (importData) {
            assignment.left = buildImportReference(importData, left.node);
            assignment.right = _core.types.sequenceExpression([assignment.right, buildImportThrow(localName)]);
          }
          const {
            operator
          } = assignment;
          let newExpr;
          if (operator === "=") {
            newExpr = assignment;
          } else if (operator === "&&=" || operator === "||=" || operator === "??=") {
            newExpr = _core.types.assignmentExpression("=", assignment.left, _core.types.logicalExpression(operator.slice(0, -1), _core.types.cloneNode(assignment.left), assignment.right));
          } else {
            newExpr = _core.types.assignmentExpression("=", assignment.left, _core.types.binaryExpression(operator.slice(0, -1), _core.types.cloneNode(assignment.left), assignment.right));
          }
          path.replaceWith(buildBindingExportAssignmentExpression(this.metadata, exportedNames, newExpr, path.scope));
          requeueInParent(path);
          path.skip();
        }
      } else {
        const ids = left.getOuterBindingIdentifiers();
        const programScopeIds = Object.keys(ids).filter(localName => scope.getBinding(localName) === path.scope.getBinding(localName));
        const id = programScopeIds.find(localName => imported.has(localName));
        if (id) {
          path.node.right = _core.types.sequenceExpression([path.node.right, buildImportThrow(id)]);
        }
        const items = [];
        programScopeIds.forEach(localName => {
          const exportedNames = exported.get(localName) || [];
          if (exportedNames.length > 0) {
            items.push(buildBindingExportAssignmentExpression(this.metadata, exportedNames, _core.types.identifier(localName), path.scope));
          }
        });
        if (items.length > 0) {
          let node = _core.types.sequenceExpression(items);
          if (path.parentPath.isExpressionStatement()) {
            node = _core.types.expressionStatement(node);
            node._blockHoist = path.parentPath.node._blockHoist;
          }
          const statement = path.insertAfter(node)[0];
          requeueInParent(statement);
        }
      }
    }
  },
  ForXStatement(path) {
    const {
      scope,
      node
    } = path;
    const {
      left
    } = node;
    const {
      exported,
      imported,
      scope: programScope
    } = this;
    if (!_core.types.isVariableDeclaration(left)) {
      let didTransformExport = false,
        importConstViolationName;
      const loopBodyScope = path.get("body").scope;
      for (const name of Object.keys(_core.types.getOuterBindingIdentifiers(left))) {
        if (programScope.getBinding(name) === scope.getBinding(name)) {
          if (exported.has(name)) {
            didTransformExport = true;
            if (loopBodyScope.hasOwnBinding(name)) {
              loopBodyScope.rename(name);
            }
          }
          if (imported.has(name) && !importConstViolationName) {
            importConstViolationName = name;
          }
        }
      }
      if (!didTransformExport && !importConstViolationName) {
        return;
      }
      path.ensureBlock();
      const bodyPath = path.get("body");
      const newLoopId = scope.generateUidIdentifierBasedOnNode(left);
      path.get("left").replaceWith(_core.types.variableDeclaration("let", [_core.types.variableDeclarator(_core.types.cloneNode(newLoopId))]));
      scope.registerDeclaration(path.get("left"));
      if (didTransformExport) {
        bodyPath.unshiftContainer("body", _core.types.expressionStatement(_core.types.assignmentExpression("=", left, newLoopId)));
      }
      if (importConstViolationName) {
        bodyPath.unshiftContainer("body", _core.types.expressionStatement(buildImportThrow(importConstViolationName)));
      }
    }
  }
};

//# sourceMappingURL=rewrite-live-references.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614620, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = normalizeModuleAndLoadMetadata;
exports.hasExports = hasExports;
exports.isSideEffectImport = isSideEffectImport;
exports.validateImportInteropOption = validateImportInteropOption;
var _path = require("path");
var _helperValidatorIdentifier = require("@babel/helper-validator-identifier");
function hasExports(metadata) {
  return metadata.hasExports;
}
function isSideEffectImport(source) {
  return source.imports.size === 0 && source.importsNamespace.size === 0 && source.reexports.size === 0 && source.reexportNamespace.size === 0 && !source.reexportAll;
}
function validateImportInteropOption(importInterop) {
  if (typeof importInterop !== "function" && importInterop !== "none" && importInterop !== "babel" && importInterop !== "node") {
    throw new Error(`.importInterop must be one of "none", "babel", "node", or a function returning one of those values (received ${importInterop}).`);
  }
  return importInterop;
}
function resolveImportInterop(importInterop, source, filename) {
  if (typeof importInterop === "function") {
    return validateImportInteropOption(importInterop(source, filename));
  }
  return importInterop;
}
function normalizeModuleAndLoadMetadata(programPath, exportName, {
  importInterop,
  initializeReexports = false,
  getWrapperPayload,
  esNamespaceOnly = false,
  filename
}) {
  if (!exportName) {
    exportName = programPath.scope.generateUidIdentifier("exports").name;
  }
  const stringSpecifiers = new Set();
  nameAnonymousExports(programPath);
  const {
    local,
    sources,
    hasExports
  } = getModuleMetadata(programPath, {
    initializeReexports,
    getWrapperPayload
  }, stringSpecifiers);
  removeImportExportDeclarations(programPath);
  for (const [source, metadata] of sources) {
    const {
      importsNamespace,
      imports
    } = metadata;
    if (importsNamespace.size > 0 && imports.size === 0) {
      const [nameOfnamespace] = importsNamespace;
      metadata.name = nameOfnamespace;
    }
    const resolvedInterop = resolveImportInterop(importInterop, source, filename);
    if (resolvedInterop === "none") {
      metadata.interop = "none";
    } else if (resolvedInterop === "node" && metadata.interop === "namespace") {
      metadata.interop = "node-namespace";
    } else if (resolvedInterop === "node" && metadata.interop === "default") {
      metadata.interop = "node-default";
    } else if (esNamespaceOnly && metadata.interop === "namespace") {
      metadata.interop = "default";
    }
  }
  return {
    exportName,
    exportNameListName: null,
    hasExports,
    local,
    source: sources,
    stringSpecifiers
  };
}
function getExportSpecifierName(path, stringSpecifiers) {
  if (path.isIdentifier()) {
    return path.node.name;
  } else if (path.isStringLiteral()) {
    const stringValue = path.node.value;
    if (!(0, _helperValidatorIdentifier.isIdentifierName)(stringValue)) {
      stringSpecifiers.add(stringValue);
    }
    return stringValue;
  } else {
    throw new Error(`Expected export specifier to be either Identifier or StringLiteral, got ${path.node.type}`);
  }
}
function assertExportSpecifier(path) {
  if (path.isExportSpecifier()) {
    return;
  } else if (path.isExportNamespaceSpecifier()) {
    throw path.buildCodeFrameError("Export namespace should be first transformed by `@babel/plugin-transform-export-namespace-from`.");
  } else {
    throw path.buildCodeFrameError("Unexpected export specifier type");
  }
}
function getModuleMetadata(programPath, {
  getWrapperPayload,
  initializeReexports
}, stringSpecifiers) {
  const localData = getLocalExportMetadata(programPath, initializeReexports, stringSpecifiers);
  const importNodes = new Map();
  const sourceData = new Map();
  const getData = (sourceNode, node) => {
    const source = sourceNode.value;
    let data = sourceData.get(source);
    if (!data) {
      data = {
        name: programPath.scope.generateUidIdentifier((0, _path.basename)(source, (0, _path.extname)(source))).name,
        interop: "none",
        loc: null,
        imports: new Map(),
        importsNamespace: new Set(),
        reexports: new Map(),
        reexportNamespace: new Set(),
        reexportAll: null,
        wrap: null,
        get lazy() {
          return this.wrap === "lazy";
        },
        referenced: false
      };
      sourceData.set(source, data);
      importNodes.set(source, [node]);
    } else {
      importNodes.get(source).push(node);
    }
    return data;
  };
  let hasExports = false;
  programPath.get("body").forEach(child => {
    if (child.isImportDeclaration()) {
      const data = getData(child.node.source, child.node);
      if (!data.loc) data.loc = child.node.loc;
      child.get("specifiers").forEach(spec => {
        if (spec.isImportDefaultSpecifier()) {
          const localName = spec.get("local").node.name;
          data.imports.set(localName, "default");
          const reexport = localData.get(localName);
          if (reexport) {
            localData.delete(localName);
            reexport.names.forEach(name => {
              data.reexports.set(name, "default");
            });
            data.referenced = true;
          }
        } else if (spec.isImportNamespaceSpecifier()) {
          const localName = spec.get("local").node.name;
          data.importsNamespace.add(localName);
          const reexport = localData.get(localName);
          if (reexport) {
            localData.delete(localName);
            reexport.names.forEach(name => {
              data.reexportNamespace.add(name);
            });
            data.referenced = true;
          }
        } else if (spec.isImportSpecifier()) {
          const importName = getExportSpecifierName(spec.get("imported"), stringSpecifiers);
          const localName = spec.get("local").node.name;
          data.imports.set(localName, importName);
          const reexport = localData.get(localName);
          if (reexport) {
            localData.delete(localName);
            reexport.names.forEach(name => {
              data.reexports.set(name, importName);
            });
            data.referenced = true;
          }
        }
      });
    } else if (child.isExportAllDeclaration()) {
      hasExports = true;
      const data = getData(child.node.source, child.node);
      if (!data.loc) data.loc = child.node.loc;
      data.reexportAll = {
        loc: child.node.loc
      };
      data.referenced = true;
    } else if (child.isExportNamedDeclaration() && child.node.source) {
      hasExports = true;
      const data = getData(child.node.source, child.node);
      if (!data.loc) data.loc = child.node.loc;
      child.get("specifiers").forEach(spec => {
        assertExportSpecifier(spec);
        const importName = getExportSpecifierName(spec.get("local"), stringSpecifiers);
        const exportName = getExportSpecifierName(spec.get("exported"), stringSpecifiers);
        data.reexports.set(exportName, importName);
        data.referenced = true;
        if (exportName === "__esModule") {
          throw spec.get("exported").buildCodeFrameError('Illegal export "__esModule".');
        }
      });
    } else if (child.isExportNamedDeclaration() || child.isExportDefaultDeclaration()) {
      hasExports = true;
    }
  });
  for (const metadata of sourceData.values()) {
    let needsDefault = false;
    let needsNamed = false;
    if (metadata.importsNamespace.size > 0) {
      needsDefault = true;
      needsNamed = true;
    }
    if (metadata.reexportAll) {
      needsNamed = true;
    }
    for (const importName of metadata.imports.values()) {
      if (importName === "default") needsDefault = true;else needsNamed = true;
    }
    for (const importName of metadata.reexports.values()) {
      if (importName === "default") needsDefault = true;else needsNamed = true;
    }
    if (needsDefault && needsNamed) {
      metadata.interop = "namespace";
    } else if (needsDefault) {
      metadata.interop = "default";
    }
  }
  if (getWrapperPayload) {
    for (const [source, metadata] of sourceData) {
      metadata.wrap = getWrapperPayload(source, metadata, importNodes.get(source));
    }
  }
  return {
    hasExports,
    local: localData,
    sources: sourceData
  };
}
function getLocalExportMetadata(programPath, initializeReexports, stringSpecifiers) {
  const bindingKindLookup = new Map();
  const programScope = programPath.scope;
  const programChildren = programPath.get("body");
  programChildren.forEach(child => {
    let kind;
    if (child.isImportDeclaration()) {
      kind = "import";
    } else {
      if (child.isExportDefaultDeclaration()) {
        child = child.get("declaration");
      }
      if (child.isExportNamedDeclaration()) {
        if (child.node.declaration) {
          child = child.get("declaration");
        } else if (initializeReexports && child.node.source && child.get("source").isStringLiteral()) {
          child.get("specifiers").forEach(spec => {
            assertExportSpecifier(spec);
            bindingKindLookup.set(spec.get("local").node.name, "block");
          });
          return;
        }
      }
      if (child.isFunctionDeclaration()) {
        kind = "hoisted";
      } else if (child.isClassDeclaration()) {
        kind = "block";
      } else if (child.isVariableDeclaration({
        kind: "var"
      })) {
        kind = "var";
      } else if (child.isVariableDeclaration()) {
        kind = "block";
      } else {
        return;
      }
    }
    Object.keys(child.getOuterBindingIdentifiers()).forEach(name => {
      bindingKindLookup.set(name, kind);
    });
  });
  const localMetadata = new Map();
  const getLocalMetadata = idPath => {
    const localName = idPath.node.name;
    let metadata = localMetadata.get(localName);
    if (!metadata) {
      var _bindingKindLookup$ge, _programScope$getBind;
      const kind = (_bindingKindLookup$ge = bindingKindLookup.get(localName)) != null ? _bindingKindLookup$ge : (_programScope$getBind = programScope.getBinding(localName)) == null ? void 0 : _programScope$getBind.kind;
      if (kind === undefined) {
        throw idPath.buildCodeFrameError(`Exporting local "${localName}", which is not declared.`);
      }
      metadata = {
        names: [],
        kind
      };
      localMetadata.set(localName, metadata);
    }
    return metadata;
  };
  programChildren.forEach(child => {
    if (child.isExportNamedDeclaration() && (initializeReexports || !child.node.source)) {
      if (child.node.declaration) {
        const declaration = child.get("declaration");
        const ids = declaration.getOuterBindingIdentifierPaths();
        Object.keys(ids).forEach(name => {
          if (name === "__esModule") {
            throw declaration.buildCodeFrameError('Illegal export "__esModule".');
          }
          getLocalMetadata(ids[name]).names.push(name);
        });
      } else {
        child.get("specifiers").forEach(spec => {
          const local = spec.get("local");
          const exported = spec.get("exported");
          const localMetadata = getLocalMetadata(local);
          const exportName = getExportSpecifierName(exported, stringSpecifiers);
          if (exportName === "__esModule") {
            throw exported.buildCodeFrameError('Illegal export "__esModule".');
          }
          localMetadata.names.push(exportName);
        });
      }
    } else if (child.isExportDefaultDeclaration()) {
      const declaration = child.get("declaration");
      if (declaration.isFunctionDeclaration() || declaration.isClassDeclaration()) {
        getLocalMetadata(declaration.get("id")).names.push("default");
      } else {
        throw declaration.buildCodeFrameError("Unexpected default expression export.");
      }
    }
  });
  return localMetadata;
}
function nameAnonymousExports(programPath) {
  programPath.get("body").forEach(child => {
    var _child$splitExportDec;
    if (!child.isExportDefaultDeclaration()) return;
    (_child$splitExportDec = child.splitExportDeclaration) != null ? _child$splitExportDec : child.splitExportDeclaration = require("@babel/traverse").NodePath.prototype.splitExportDeclaration;
    child.splitExportDeclaration();
  });
}
function removeImportExportDeclarations(programPath) {
  programPath.get("body").forEach(child => {
    if (child.isImportDeclaration()) {
      child.remove();
    } else if (child.isExportNamedDeclaration()) {
      if (child.node.declaration) {
        child.node.declaration._blockHoist = child.node._blockHoist;
        child.replaceWith(child.node.declaration);
      } else {
        child.remove();
      }
    } else if (child.isExportDefaultDeclaration()) {
      const declaration = child.get("declaration");
      if (declaration.isFunctionDeclaration() || declaration.isClassDeclaration()) {
        declaration._blockHoist = child.node._blockHoist;
        child.replaceWith(declaration);
      } else {
        throw declaration.buildCodeFrameError("Unexpected default expression export.");
      }
    } else if (child.isExportAllDeclaration()) {
      child.remove();
    }
  });
}

//# sourceMappingURL=normalize-and-load-metadata.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614621, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.toGetWrapperPayload = toGetWrapperPayload;
exports.wrapReference = wrapReference;
var _core = require("@babel/core");
var _normalizeAndLoadMetadata = require("./normalize-and-load-metadata.js");
function toGetWrapperPayload(lazy) {
  return (source, metadata) => {
    if (lazy === false) return null;
    if ((0, _normalizeAndLoadMetadata.isSideEffectImport)(metadata) || metadata.reexportAll) return null;
    if (lazy === true) {
      return source.includes(".") ? null : "lazy";
    }
    if (Array.isArray(lazy)) {
      return !lazy.includes(source) ? null : "lazy";
    }
    if (typeof lazy === "function") {
      return lazy(source) ? "lazy" : null;
    }
    throw new Error(`.lazy must be a boolean, string array, or function`);
  };
}
function wrapReference(ref, payload) {
  if (payload === "lazy") return _core.types.callExpression(ref, []);
  return null;
}

//# sourceMappingURL=lazy-modules.js.map

}, function(modId) { var map = {"./normalize-and-load-metadata.js":1781530614620}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614622, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.buildDynamicImport = buildDynamicImport;
var _core = require("@babel/core");
exports.getDynamicImportSource = function getDynamicImportSource(node) {
  const [source] = node.arguments;
  return _core.types.isStringLiteral(source) || _core.types.isTemplateLiteral(source) ? source : _core.template.expression.ast`\`\${${source}}\``;
};
function buildDynamicImport(node, deferToThen, wrapWithPromise, builder) {
  const specifier = _core.types.isCallExpression(node) ? node.arguments[0] : node.source;
  if (_core.types.isStringLiteral(specifier) || _core.types.isTemplateLiteral(specifier) && specifier.quasis.length === 0) {
    if (deferToThen) {
      return _core.template.expression.ast`
        Promise.resolve().then(() => ${builder(specifier)})
      `;
    } else return builder(specifier);
  }
  const specifierToString = _core.types.isTemplateLiteral(specifier) ? _core.types.identifier("specifier") : _core.types.templateLiteral([_core.types.templateElement({
    raw: ""
  }), _core.types.templateElement({
    raw: ""
  })], [_core.types.identifier("specifier")]);
  if (deferToThen) {
    return _core.template.expression.ast`
      (specifier =>
        new Promise(r => r(${specifierToString}))
          .then(s => ${builder(_core.types.identifier("s"))})
      )(${specifier})
    `;
  } else if (wrapWithPromise) {
    return _core.template.expression.ast`
      (specifier =>
        new Promise(r => r(${builder(specifierToString)}))
      )(${specifier})
    `;
  } else {
    return _core.template.expression.ast`
      (specifier => ${builder(specifierToString)})(${specifier})
    `;
  }
}

//# sourceMappingURL=dynamic-import.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530614623, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = getModuleName;
const originalGetModuleName = getModuleName;
exports.default = getModuleName = function getModuleName(rootOpts, pluginOpts) {
  var _pluginOpts$moduleId, _pluginOpts$moduleIds, _pluginOpts$getModule, _pluginOpts$moduleRoo;
  return originalGetModuleName(rootOpts, {
    moduleId: (_pluginOpts$moduleId = pluginOpts.moduleId) != null ? _pluginOpts$moduleId : rootOpts.moduleId,
    moduleIds: (_pluginOpts$moduleIds = pluginOpts.moduleIds) != null ? _pluginOpts$moduleIds : rootOpts.moduleIds,
    getModuleId: (_pluginOpts$getModule = pluginOpts.getModuleId) != null ? _pluginOpts$getModule : rootOpts.getModuleId,
    moduleRoot: (_pluginOpts$moduleRoo = pluginOpts.moduleRoot) != null ? _pluginOpts$moduleRoo : rootOpts.moduleRoot
  });
};
function getModuleName(rootOpts, pluginOpts) {
  const {
    filename,
    filenameRelative = filename,
    sourceRoot = pluginOpts.moduleRoot
  } = rootOpts;
  const {
    moduleId,
    moduleIds = !!moduleId,
    getModuleId,
    moduleRoot = sourceRoot
  } = pluginOpts;
  if (!moduleIds) return null;
  if (moduleId != null && !getModuleId) {
    return moduleId;
  }
  let moduleName = moduleRoot != null ? moduleRoot + "/" : "";
  if (filenameRelative) {
    const sourceRootReplacer = sourceRoot != null ? new RegExp("^" + sourceRoot + "/?") : "";
    moduleName += filenameRelative.replace(sourceRootReplacer, "").replace(/\.\w*$/, "");
  }
  moduleName = moduleName.replace(/\\/g, "/");
  if (getModuleId) {
    return getModuleId(moduleName) || moduleName;
  } else {
    return moduleName;
  }
}

//# sourceMappingURL=get-module-name.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
return __REQUIRE__(1781530614617);
})()
//miniprogram-npm-outsideDeps=["assert","@babel/core","@babel/helper-module-imports","@babel/traverse","path","@babel/helper-validator-identifier"]
//# sourceMappingURL=index.js.map