module.exports = (function() {
var __MODS__ = {};
var __DEFINE__ = function(modId, func, req) { var m = { exports: {}, _tempexports: {} }; __MODS__[modId] = { status: 0, func: func, req: req, m: m }; };
var __REQUIRE__ = function(modId, source) { if(!__MODS__[modId]) return require(source); if(!__MODS__[modId].status) { var m = __MODS__[modId].m; m._exports = m._tempexports; var desp = Object.getOwnPropertyDescriptor(m, "exports"); if (desp && desp.configurable) Object.defineProperty(m, "exports", { set: function (val) { if(typeof val === "object" && val !== m._exports) { m._exports.__proto__ = val.__proto__; Object.keys(val).forEach(function (k) { m._exports[k] = val[k]; }); } m._tempexports = val }, get: function () { return m._tempexports; } }); __MODS__[modId].status = 1; __MODS__[modId].func(__MODS__[modId].req, m, m.exports); } return __MODS__[modId].m.exports; };
var __REQUIRE_WILDCARD__ = function(obj) { if(obj && obj.__esModule) { return obj; } else { var newObj = {}; if(obj != null) { for(var k in obj) { if (Object.prototype.hasOwnProperty.call(obj, k)) newObj[k] = obj[k]; } } newObj.default = obj; return newObj; } };
var __REQUIRE_DEFAULT__ = function(obj) { return obj && obj.__esModule ? obj.default : obj; };
__DEFINE__(1781530615304, function(require, module, exports) {

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
exports.compile = exports.parse = exports.transformModel = exports.transformOn = exports.isForElementNode = exports.rewriteExpression = exports.genExpr = exports.findProp = void 0;
const shared_1 = require("@vue/shared");
const compiler_core_1 = require("@vue/compiler-core");
const compile_1 = require("./compile");
const parserOptions_1 = require("./parserOptions");
var compiler_core_2 = require("@vue/compiler-core");
Object.defineProperty(exports, "findProp", { enumerable: true, get: function () { return compiler_core_2.findProp; } });
var codegen_1 = require("./codegen");
Object.defineProperty(exports, "genExpr", { enumerable: true, get: function () { return codegen_1.genExpr; } });
var utils_1 = require("./transforms/utils");
Object.defineProperty(exports, "rewriteExpression", { enumerable: true, get: function () { return utils_1.rewriteExpression; } });
var vFor_1 = require("./transforms/vFor");
Object.defineProperty(exports, "isForElementNode", { enumerable: true, get: function () { return vFor_1.isForElementNode; } });
var vOn_1 = require("./transforms/vOn");
Object.defineProperty(exports, "transformOn", { enumerable: true, get: function () { return vOn_1.transformOn; } });
var vModel_1 = require("./transforms/vModel");
Object.defineProperty(exports, "transformModel", { enumerable: true, get: function () { return vModel_1.transformModel; } });
__exportStar(require("./runtimeHelpers"), exports);
function parse(template, options = {}) {
    return (0, compiler_core_1.baseParse)(template, (0, shared_1.extend)({}, parserOptions_1.parserOptions, options));
}
exports.parse = parse;
function compile(template, options = {}) {
    return (0, compile_1.baseCompile)(template, (0, shared_1.extend)({}, parserOptions_1.parserOptions, options, {
        directiveTransforms: (0, shared_1.extend)({}, options.directiveTransforms || {}),
    }));
}
exports.compile = compile;

}, function(modId) {var map = {"./parserOptions":1781530615306,"./codegen":1781530615307,"./transforms/vFor":1781530615309,"./runtimeHelpers":1781530615314}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615306, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.parserOptions = void 0;
const compiler_core_1 = require("@vue/compiler-core");
const shared_1 = require("@vue/shared");
exports.parserOptions = {
    isVoidTag(tag) {
        // 开启 input 标签判断，会导致 <input type="text"> 编译失败
        // 微信小程序允许 Input 嵌套其他组件 https://ask.dcloud.net.cn/question/202776
        // if (tag === 'input') {
        //   return false
        // }
        return (0, shared_1.isVoidTag)(tag);
    },
    isNativeTag: (tag) => (0, shared_1.isHTMLTag)(tag) || (0, shared_1.isSVGTag)(tag),
    isPreTag: (tag) => tag === 'pre',
    // https://html.spec.whatwg.org/multipage/parsing.html#tree-construction-dispatcher
    getNamespace(tag, parent) {
        let ns = parent ? parent.ns : 0 /* DOMNamespaces.HTML */;
        if (parent && ns === 2 /* DOMNamespaces.MATH_ML */) {
            if (parent.tag === 'annotation-xml') {
                if (tag === 'svg') {
                    return 1 /* DOMNamespaces.SVG */;
                }
                if (parent.props.some((a) => a.type === compiler_core_1.NodeTypes.ATTRIBUTE &&
                    a.name === 'encoding' &&
                    a.value != null &&
                    (a.value.content === 'text/html' ||
                        a.value.content === 'application/xhtml+xml'))) {
                    ns = 0 /* DOMNamespaces.HTML */;
                }
            }
            else if (/^m(?:[ions]|text)$/.test(parent.tag) &&
                tag !== 'mglyph' &&
                tag !== 'malignmark') {
                ns = 0 /* DOMNamespaces.HTML */;
            }
        }
        else if (parent && ns === 1 /* DOMNamespaces.SVG */) {
            if (parent.tag === 'foreignObject' ||
                parent.tag === 'desc' ||
                parent.tag === 'title') {
                ns = 0 /* DOMNamespaces.HTML */;
            }
        }
        if (ns === 0 /* DOMNamespaces.HTML */) {
            if (tag === 'svg') {
                return 1 /* DOMNamespaces.SVG */;
            }
            if (tag === 'math') {
                return 2 /* DOMNamespaces.MATH_ML */;
            }
        }
        return ns;
    },
    parseMode: 'html',
};

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615307, function(require, module, exports) {

var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.genExpr = exports.genBabelExpr = exports.generate = void 0;
const shared_1 = require("@vue/shared");
const compiler_core_1 = require("@vue/compiler-core");
const generator_1 = __importDefault(require("@babel/generator"));
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
function generate(ast, options) {
    const context = createCodegenContext(ast, options);
    const { mode, push, indent, deindent, newline, prefixIdentifiers } = context;
    const helpers = Array.from(ast.helpers);
    const hasHelpers = helpers.length > 0;
    const useWithBlock = !prefixIdentifiers && mode !== 'module';
    const isSetupInlined = !!options.inline;
    // preambles
    // in setup() inline mode, the preamble is generated in a sub context
    // and returned separately.
    const preambleContext = isSetupInlined
        ? createCodegenContext(ast, options)
        : context;
    if (mode === 'module') {
        genModulePreamble(ast, preambleContext, isSetupInlined);
    }
    else {
        genFunctionPreamble(ast, preambleContext);
    }
    // enter render function
    const functionName = `render`;
    const args = ['_ctx', '_cache'];
    if (options.bindingMetadata && !options.inline) {
        // binding optimization args
        args.push('$props', '$setup', '$data', '$options');
    }
    const signature = options.isTS
        ? args.map((arg) => `${arg}: any`).join(',')
        : args.join(', ');
    const rawJs = options.isX ? ' "raw js"' : '';
    if (isSetupInlined) {
        push(`(${signature}) => {${rawJs}`);
    }
    else {
        push(`function ${functionName}(${signature}) {${rawJs}`);
    }
    indent();
    if (useWithBlock) {
        push(`with (_ctx) {`);
        indent();
        if (hasHelpers) {
            push(`const { ${helpers
                .map((s) => `${compiler_core_1.helperNameMap[s]}: _${compiler_core_1.helperNameMap[s]}`)
                .join(', ')} } = _Vue`);
            push(`\n`);
            newline();
        }
    }
    if (isSetupInlined && options.isX) {
        push(`const __returned__ = `);
        push(genBabelExpr(ast.renderData, options.generatorOpts));
        newline();
        push(`return __returned__`);
    }
    else {
        push(`return `);
        push(genBabelExpr(ast.renderData, options.generatorOpts));
    }
    if (useWithBlock) {
        deindent();
        push(`}`);
    }
    deindent();
    push(`}`);
    return {
        code: context.code,
        preamble: isSetupInlined ? preambleContext.code : ``,
        // SourceMapGenerator does have toJSON() method but it's not in the types
        map: context.map ? context.map.toJSON() : undefined,
    };
}
exports.generate = generate;
function createCodegenContext(ast, { mode = 'function', prefixIdentifiers = mode === 'module', filename = `template.vue.html`, scopeId = null, runtimeGlobalName = `Vue`, runtimeModuleName = `vue`, isTS = false, sourceMap = false, }) {
    const context = {
        mode,
        prefixIdentifiers,
        filename,
        scopeId,
        runtimeGlobalName,
        runtimeModuleName,
        bindingComponents: ast.bindingComponents,
        isTS,
        source: ast.loc.source,
        code: ``,
        column: 1,
        line: 1,
        offset: 0,
        indentLevel: 0,
        push(code, node) {
            context.code += code;
            if (context.map) {
                if (node) {
                    let name;
                    if ((0, uni_cli_shared_1.isSimpleExpressionNode)(node) && !node.isStatic) {
                        const content = node.content.replace(/^_ctx\./, '');
                        if (content !== node.content && (0, compiler_core_1.isSimpleIdentifier)(content)) {
                            name = content;
                        }
                    }
                    addMapping(node.loc.start, name);
                }
                (0, compiler_core_1.advancePositionWithMutation)(context, code);
                if (node && node.loc !== compiler_core_1.locStub) {
                    addMapping(node.loc.end);
                }
            }
        },
        indent() {
            newline(++context.indentLevel);
        },
        deindent(withoutNewLine = false) {
            if (withoutNewLine) {
                --context.indentLevel;
            }
            else {
                newline(--context.indentLevel);
            }
        },
        newline() {
            newline(context.indentLevel);
        },
    };
    function newline(n) {
        context.push('\n' + `  `.repeat(n));
    }
    function addMapping(loc, name) {
        context.map.addMapping({
            name,
            source: context.filename || '',
            original: {
                line: loc.line,
                column: loc.column - 1, // source-map column is 0 based
            },
            generated: {
                line: context.line,
                column: context.column - 1,
            },
        });
    }
    // 暂时无需提供 sourcemap 支持
    // if (sourceMap) {
    //   // lazy require source-map implementation
    //   context.map = new SourceMapGenerator()
    //   context.map!.setSourceContent(filename, context.source)
    // }
    return context;
}
function genComponentImports(bindingComponents, { push, newline }) {
    const tags = Object.keys(bindingComponents);
    const importDeclarations = [];
    // 仅记录easycom和setup组件
    const components = [];
    tags.forEach((tag) => {
        const { name, type } = bindingComponents[tag];
        if (type === "unknown" /* BindingComponentTypes.UNKNOWN */) {
            const source = (0, uni_cli_shared_1.matchEasycom)(tag);
            if (source) {
                // 调整为easycom命名
                const easycomName = name.replace('component', 'easycom');
                bindingComponents[tag].name = easycomName;
                components.push(easycomName);
                (0, uni_cli_shared_1.addImportDeclaration)(importDeclarations, easycomName, source);
            }
        }
        else if (type === "setup" /* BindingComponentTypes.SETUP */) {
            components.push(name);
        }
    });
    if (tags.length) {
        push(`const __BINDING_COMPONENTS__ = '` +
            JSON.stringify(bindingComponents) +
            `'`);
        const resolveComponents = [];
        const names = [];
        Object.keys(bindingComponents).forEach((id) => {
            const { type, name } = bindingComponents[id];
            if (type === "unknown" /* BindingComponentTypes.UNKNOWN */) {
                resolveComponents.push(`const ${name} = _${compiler_core_1.helperNameMap[compiler_core_1.RESOLVE_COMPONENT]}("${id}");`);
                names.push(name);
            }
        });
        if (resolveComponents.length) {
            newline();
            push(`if (!Array) {`);
            resolveComponents.forEach((code) => {
                push(code);
            });
            push(`(${names.join('+')})()`);
            push(`}`);
        }
        newline();
        importDeclarations.forEach((str) => push(str));
        if (importDeclarations.length) {
            newline();
        }
        if (components.length) {
            push(`if (!Math) {`);
            push(` (${components.map((name) => name).join('+')})() `);
            push(`}`);
            newline();
        }
    }
}
function genFunctionPreamble(ast, context) {
    const { prefixIdentifiers, push, newline, runtimeGlobalName, bindingComponents, } = context;
    const VueBinding = runtimeGlobalName;
    const aliasHelper = (s) => `${compiler_core_1.helperNameMap[s]}: _${compiler_core_1.helperNameMap[s]}`;
    const helpers = Array.from(ast.helpers);
    if (helpers.length > 0) {
        if (prefixIdentifiers) {
            push(`const { ${helpers.map(aliasHelper).join(', ')} } = ${VueBinding}\n`);
        }
        else {
            push(`const _Vue = ${VueBinding}\n`);
        }
    }
    genComponentImports(bindingComponents, context);
    newline();
    push(`return `);
}
function genModulePreamble(ast, context, inline) {
    const { push, newline, runtimeModuleName, bindingComponents } = context;
    const helpers = Array.from(ast.helpers);
    if (helpers.length) {
        push(`import { ${helpers
            .map((s) => `${compiler_core_1.helperNameMap[s]} as _${compiler_core_1.helperNameMap[s]}`)
            .join(', ')} } from ${JSON.stringify(runtimeModuleName)}\n`);
    }
    if (ast.imports.length) {
        genImports(ast.imports, context);
    }
    genComponentImports(bindingComponents, context);
    newline();
    if (!inline) {
        push(`export `);
    }
}
function genImports(importsOptions, { push, newline }) {
    if (!importsOptions.length) {
        return;
    }
    importsOptions.forEach((imports) => {
        push(`import `);
        push(genExpr(imports.exp));
        push(` from '${imports.path}'`);
        newline();
    });
}
function createGenNodeContext() {
    const context = {
        code: '',
        helper(key) {
            return `_${compiler_core_1.helperNameMap[key]}`;
        },
        push(code) {
            context.code += code;
        },
    };
    return context;
}
function genBabelExpr(expr, opts = {}) {
    if (!(0, shared_1.hasOwn)(opts, 'jsescOption')) {
        opts.jsescOption = {};
    }
    opts.jsescOption.quotes = 'single';
    return (0, generator_1.default)(expr, opts).code;
}
exports.genBabelExpr = genBabelExpr;
function genExpr(node, context) {
    return genNode(node, context).code;
}
exports.genExpr = genExpr;
function genNode(node, context) {
    if (!context) {
        context = createGenNodeContext();
    }
    if ((0, shared_1.isString)(node)) {
        context.push(node);
        return context;
    }
    if ((0, shared_1.isSymbol)(node)) {
        context.push(context.helper(node));
        return context;
    }
    switch (node.type) {
        case compiler_core_1.NodeTypes.TEXT:
            genText(node, context);
            break;
        case compiler_core_1.NodeTypes.SIMPLE_EXPRESSION:
            genExpression(node, context);
            break;
        case compiler_core_1.NodeTypes.INTERPOLATION:
            genInterpolation(node, context);
            break;
        case compiler_core_1.NodeTypes.COMPOUND_EXPRESSION:
            genCompoundExpression(node, context);
            break;
    }
    return context;
}
function genText(node, context) {
    context.push(JSON.stringify(node.content), node);
}
function genExpression(node, context) {
    const { content, isStatic } = node;
    context.push(isStatic ? JSON.stringify(content) : content, node);
}
function genInterpolation(node, context) {
    const { push, helper } = context;
    push(`${helper(compiler_core_1.TO_DISPLAY_STRING)}(`);
    genExpr(node.content, context);
    push(`)`);
}
function genCompoundExpression(node, context) {
    for (let i = 0; i < node.children.length; i++) {
        const child = node.children[i];
        if ((0, shared_1.isString)(child)) {
            context.push(child);
        }
        else {
            genExpr(child, context);
        }
    }
}

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615309, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.createVForArrowFunctionExpression = exports.createForLoopParams = exports.parseForExpression = exports.parseVForScope = exports.transformFor = exports.isForElementNode = void 0;
const shared_1 = require("@vue/shared");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const compiler_core_1 = require("@vue/compiler-core");
const ast_1 = require("../ast");
const transform_1 = require("../transform");
const transformExpression_1 = require("./transformExpression");
const codegen_1 = require("../codegen");
const types_1 = require("@babel/types");
const utils_1 = require("./utils");
const runtimeHelpers_1 = require("../runtimeHelpers");
const vSlot_1 = require("./vSlot");
function isForElementNode(node) {
    return !!node.vFor;
}
exports.isForElementNode = isForElementNode;
exports.transformFor = (0, transform_1.createStructuralDirectiveTransform)('for', (node, dir, context) => {
    if (!dir.exp) {
        context.onError((0, compiler_core_1.createCompilerError)(compiler_core_1.ErrorCodes.X_V_FOR_NO_EXPRESSION, dir.loc));
        return;
    }
    const parseResult = parseForExpression(dir.exp, context);
    if (!parseResult) {
        context.onError((0, compiler_core_1.createCompilerError)(compiler_core_1.ErrorCodes.X_V_FOR_MALFORMED_EXPRESSION, dir.loc));
        return;
    }
    parseResult.tagType = node.tagType;
    const { addIdentifiers, removeIdentifiers } = context;
    const { source, value, key, index } = parseResult;
    if (context.prefixIdentifiers) {
        addIdentifiers(value);
        addIdentifiers(key);
        addIdentifiers(index);
    }
    const { currentScope: parentScope, scopes, popScope } = context;
    const sourceExpr = (0, ast_1.parseExpr)(source, context);
    const valueCode = (0, codegen_1.genExpr)(value);
    const valueExpr = (0, ast_1.parseParam)(valueCode, context, value);
    const valueAlias = parseAlias(valueExpr, valueCode, 'v' + scopes.vFor);
    const keyCode = (0, codegen_1.genExpr)(key);
    const keyExpr = (0, ast_1.parseParam)(keyCode, context, key);
    const keyAlias = parseAlias(keyExpr, keyCode, 'k' + scopes.vFor);
    const indexCode = (0, codegen_1.genExpr)(index);
    const indexExpr = (0, ast_1.parseParam)(indexCode, context, index);
    const indexAlias = parseAlias(indexExpr, indexCode, 'i' + scopes.vFor);
    // 先占位 vFor，后续更新 cloneSourceExpr 为 CallExpression
    const cloneSourceExpr = (0, types_1.cloneNode)(sourceExpr, false);
    const sourceAliasReferencedScope = (0, utils_1.findReferencedScope)(cloneSourceExpr, context.currentScope, 
    // vFor 嵌套时，始终保持嵌套关系，issues/3263
    false);
    // 寻找子节点中 if 指令作用域
    const vIfReferencedScope = findVIfReferencedScope(node, context.currentScope, context);
    // 取最近的作用域
    const referencedScope = vIfReferencedScope &&
        context.getScopeIndex(vIfReferencedScope) >
            context.getScopeIndex(sourceAliasReferencedScope)
        ? vIfReferencedScope
        : sourceAliasReferencedScope;
    const sourceAlias = (0, utils_1.rewriteExpression)(source, context, cloneSourceExpr, parentScope, 
    // 强制 rewrite，因为即使是字符串，数字，也要走 vFor 函数
    {
        property: true,
        ignoreLiteral: true,
        referencedScope,
    }).content;
    const sourceCode = `{{${sourceAlias}}}`;
    const vForData = {
        source,
        sourceExpr,
        sourceAlias,
        sourceCode,
        value,
        valueCode,
        valueExpr,
        valueAlias,
        key,
        keyCode,
        keyExpr,
        keyAlias,
        index,
        indexCode,
        indexExpr,
        indexAlias,
        node,
    };
    const vForScope = context.addVForScope({
        ...vForData,
        locals: findVForLocals(parseResult),
    });
    const vFor = {
        ...vForData,
    };
    const isScopedSlot = (0, transform_1.isScopedSlotVFor)(vForScope);
    node.vFor = vFor;
    scopes.vFor++;
    return () => {
        scopes.vFor--;
        if ((0, compiler_core_1.isTemplateNode)(node)) {
            node.children.some((c) => {
                if (c.type === compiler_core_1.NodeTypes.ELEMENT && !isForElementNode(c)) {
                    const key = (0, compiler_core_1.findProp)(c, 'key');
                    if (key) {
                        context.onError((0, compiler_core_1.createCompilerError)(compiler_core_1.ErrorCodes.X_V_FOR_TEMPLATE_KEY_PLACEMENT, key.loc));
                        return true;
                    }
                }
            });
        }
        if (context.prefixIdentifiers) {
            value && removeIdentifiers(value);
            key && removeIdentifiers(key);
            index && removeIdentifiers(index);
        }
        (0, shared_1.extend)(clearExpr(cloneSourceExpr), isScopedSlot
            ? (0, vSlot_1.createVSlotCallExpression)(node
                .slotComponent, vForScope, context)
            : createVForCallExpression(vForScope, context));
        popScope();
    };
});
function clearExpr(expr) {
    Object.keys(expr).forEach((key) => {
        delete expr[key];
    });
    return expr;
}
function parseAlias(babelExpr, exprCode, fallback) {
    if ((0, types_1.isIdentifier)(babelExpr)) {
        return exprCode;
    }
    return fallback;
}
function parseVForScope(currentScope) {
    while (currentScope) {
        if ((0, transform_1.isVForScope)(currentScope) && !(0, transform_1.isScopedSlotVFor)(currentScope)) {
            return currentScope;
        }
        currentScope = currentScope.parent;
    }
}
exports.parseVForScope = parseVForScope;
function findVIfReferencedScope(node, currentScope, context) {
    if (!currentScope) {
        return;
    }
    const vForScope = parseVForScope(currentScope);
    if (!vForScope) {
        return;
    }
    if (!node.children.find((item) => checkVIfReferenced(item, vForScope, context))) {
        return findVIfReferencedScope(node, currentScope.parent, context);
    }
    return vForScope;
}
function checkVIfReferenced(node, vForScope, context) {
    if (!(0, uni_cli_shared_1.isElementNode)(node)) {
        return false;
    }
    // 嵌套 for 不查找
    if ((0, compiler_core_1.findDir)(node, 'for')) {
        return false;
    }
    const ifDir = (0, compiler_core_1.findDir)(node, 'if');
    if (ifDir) {
        return checkDirReferenced(ifDir.exp, vForScope, context);
    }
    const elseIfDir = (0, compiler_core_1.findDir)(node, 'else-if');
    if (elseIfDir) {
        return checkDirReferenced(elseIfDir.exp, vForScope, context);
    }
    return !!node.children.find((item) => checkVIfReferenced(item, vForScope, context));
}
function checkDirReferenced(node, vForScope, context) {
    if (node) {
        const babelNode = (0, ast_1.parseExpr)(node, context);
        if (babelNode && (0, utils_1.isReferencedByIds)(babelNode, vForScope.locals)) {
            return true;
        }
    }
    return false;
}
function findVForLocals({ value, key, index }) {
    const ids = [];
    if (value) {
        findIds(value, ids);
    }
    if (key) {
        findIds(key, ids);
    }
    if (index) {
        findIds(index, ids);
    }
    return ids;
}
function findIds(exp, ids) {
    if ((0, shared_1.isString)(exp)) {
        ids.push(exp);
    }
    else if (exp.identifiers) {
        exp.identifiers.forEach((id) => ids.push(id));
    }
    else if (exp.type === compiler_core_1.NodeTypes.SIMPLE_EXPRESSION) {
        ids.push(exp.content);
    }
}
const forAliasRE = /([\s\S]*?)\s+(?:in|of)\s+([\s\S]*)/;
const forIteratorRE = /,([^,\}\]]*)(?:,([^,\}\]]*))?$/;
const stripParensRE = /^\(|\)$/g;
function parseForExpression(input, context) {
    const loc = input.loc;
    const exp = input.content;
    const inMatch = exp.match(forAliasRE);
    if (!inMatch)
        return;
    const [, LHS, RHS] = inMatch;
    const result = {
        source: createAliasExpression(loc, RHS.trim(), exp.indexOf(RHS, LHS.length)),
        value: (0, compiler_core_1.createSimpleExpression)('v' + context.scopes.vFor),
        key: (0, compiler_core_1.createSimpleExpression)('k' + context.scopes.vFor),
        index: (0, compiler_core_1.createSimpleExpression)('i' + context.scopes.vFor),
        tagType: compiler_core_1.ElementTypes.ELEMENT,
    };
    if (context.prefixIdentifiers) {
        result.source = (0, transformExpression_1.processExpression)(result.source, context);
    }
    let valueContent = LHS.trim().replace(stripParensRE, '').trim();
    const trimmedOffset = LHS.indexOf(valueContent);
    const iteratorMatch = valueContent.match(forIteratorRE);
    if (iteratorMatch) {
        valueContent = valueContent.replace(forIteratorRE, '').trim();
        const keyContent = iteratorMatch[1].trim();
        let keyOffset;
        if (keyContent) {
            keyOffset = exp.indexOf(keyContent, trimmedOffset + valueContent.length);
            result.key = createAliasExpression(loc, keyContent, keyOffset);
            if (context.prefixIdentifiers) {
                result.key = (0, transformExpression_1.processExpression)(result.key, context, true);
            }
        }
        if (iteratorMatch[2]) {
            const indexContent = iteratorMatch[2].trim();
            if (indexContent) {
                result.index = createAliasExpression(loc, indexContent, exp.indexOf(indexContent, result.key
                    ? keyOffset + keyContent.length
                    : trimmedOffset + valueContent.length));
                if (context.prefixIdentifiers) {
                    result.index = (0, transformExpression_1.processExpression)(result.index, context, true);
                }
            }
        }
    }
    if (valueContent) {
        result.value = createAliasExpression(loc, valueContent, trimmedOffset);
        if (context.prefixIdentifiers) {
            result.value = (0, transformExpression_1.processExpression)(result.value, context, true);
        }
    }
    return result;
}
exports.parseForExpression = parseForExpression;
function createAliasExpression(range, content, offset) {
    return (0, compiler_core_1.createSimpleExpression)(content, false, (0, uni_cli_shared_1.getInnerRange)(range, offset, content.length));
}
function createForLoopParams({ value, key, index }, memoArgs = []) {
    return createParamsList([value, key, index, ...memoArgs]);
}
exports.createForLoopParams = createForLoopParams;
function createParamsList(args) {
    let i = args.length;
    while (i--) {
        if (args[i])
            break;
    }
    return args
        .slice(0, i + 1)
        .map((arg, i) => arg || (0, compiler_core_1.createSimpleExpression)(`_`.repeat(i + 1), false));
}
function createVForCallExpression(vForScope, context) {
    // let sourceExpr: Expression = vForScope.sourceExpr!
    // if (isNumericLiteral(sourceExpr)) {
    //   sourceExpr = numericLiteralToArrayExpr((sourceExpr as NumericLiteral).value)
    // }
    return (0, types_1.callExpression)((0, types_1.identifier)(context.helperString(runtimeHelpers_1.V_FOR)), [
        vForScope.sourceExpr,
        createVForArrowFunctionExpression(vForScope),
    ]);
}
function createVForArrowFunctionExpression({ valueExpr, keyExpr, indexExpr, properties, }) {
    const params = [];
    if (valueExpr) {
        params.push(valueExpr);
    }
    if (keyExpr) {
        params.push(keyExpr);
    }
    if (indexExpr) {
        params.push(indexExpr);
    }
    return (0, types_1.arrowFunctionExpression)(params, (0, types_1.blockStatement)([(0, types_1.returnStatement)((0, types_1.objectExpression)(properties))]));
}
exports.createVForArrowFunctionExpression = createVForArrowFunctionExpression;

}, function(modId) { var map = {"../ast":1781530615310,"./transformExpression":1781530615312,"../codegen":1781530615307,"../runtimeHelpers":1781530615314,"./vSlot":1781530615315}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615310, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.parseStringLiteral = exports.isTrueExpr = exports.isUndefined = exports.parseParam = exports.parseExpr = exports.createVIfSpreadElement = exports.createVIfConditionalExpression = exports.createVIfProperty = exports.createObjectExpression = exports.createSpreadElement = exports.createObjectProperty = exports.createIdentifier = void 0;
const shared_1 = require("@vue/shared");
const parser_1 = require("@babel/parser");
const types_1 = require("@babel/types");
const compiler_core_1 = require("@vue/compiler-core");
const codegen_1 = require("./codegen");
function createIdentifier(name) {
    return (0, types_1.identifier)(name);
}
exports.createIdentifier = createIdentifier;
function createObjectProperty(name, value) {
    return (0, types_1.objectProperty)((0, types_1.identifier)(name), value);
}
exports.createObjectProperty = createObjectProperty;
function createSpreadElement(argument) {
    return (0, types_1.spreadElement)(argument);
}
exports.createSpreadElement = createSpreadElement;
function createObjectExpression(properties) {
    return (0, types_1.objectExpression)(properties);
}
exports.createObjectExpression = createObjectExpression;
function createVIfProperty(condition, { id }) {
    return (0, types_1.objectProperty)((0, types_1.identifier)(id.next()), condition);
}
exports.createVIfProperty = createVIfProperty;
function createVIfConditionalExpression({ condition, properties, }) {
    return (0, types_1.conditionalExpression)(condition, (0, types_1.objectExpression)(properties), (0, types_1.objectExpression)([]));
}
exports.createVIfConditionalExpression = createVIfConditionalExpression;
function createVIfSpreadElement(vIfScope) {
    return (0, types_1.spreadElement)(createVIfConditionalExpression(vIfScope));
}
exports.createVIfSpreadElement = createVIfSpreadElement;
// function numericLiteralToArrayExpr(num: number) {
//   const elements: NumericLiteral[] = []
//   for (let i = 0; i < num; i++) {
//     elements.push(numericLiteral(i + 1))
//   }
//   return arrayExpression(elements)
// }
function parseExpr(code, context, node) {
    if (!(0, shared_1.isString)(code)) {
        node = code;
        code = (0, codegen_1.genExpr)(code);
    }
    try {
        return (0, parser_1.parseExpression)(code, {
            plugins: context.expressionPlugins,
        });
    }
    catch (e) {
        context.onError((0, compiler_core_1.createCompilerError)(compiler_core_1.ErrorCodes.X_INVALID_EXPRESSION, node && node.loc, undefined, '\n' + code + '\n' + e.message));
    }
}
exports.parseExpr = parseExpr;
function parseParam(code, context, node) {
    const { params: [expr], } = parseExpr(`(${code})=>{}`, context, node);
    return expr;
}
exports.parseParam = parseParam;
function isUndefined(expr) {
    return (0, types_1.isIdentifier)(expr) && expr.name === 'undefined';
}
exports.isUndefined = isUndefined;
function isTrueExpr(expr) {
    if ((0, types_1.isNullLiteral)(expr)) {
        return false;
    }
    if ((0, types_1.isStringLiteral)(expr) ||
        (0, types_1.isNumericLiteral)(expr) ||
        (0, types_1.isBooleanLiteral)(expr) ||
        (0, types_1.isBigIntLiteral)(expr) ||
        (0, types_1.isDecimalLiteral)(expr)) {
        return !!expr.value;
    }
    return true;
}
exports.isTrueExpr = isTrueExpr;
function parseStringLiteral(expr) {
    if ((0, types_1.isIdentifier)(expr)) {
        return (0, types_1.stringLiteral)(expr.name);
    }
    if ((0, types_1.isStringLiteral)(expr)) {
        return (0, types_1.stringLiteral)(expr.value);
    }
    return (0, types_1.stringLiteral)('');
}
exports.parseStringLiteral = parseStringLiteral;

}, function(modId) { var map = {"./codegen":1781530615307}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615312, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.isBuiltInIdentifier = exports.processExpression = exports.transformExpression = void 0;
const shared_1 = require("@vue/shared");
const compiler_core_1 = require("@vue/compiler-core");
const parser_1 = require("@babel/parser");
const isLiteralWhitelisted = /*#__PURE__*/ (0, shared_1.makeMap)('true,false,null,this');
const transformExpression = (node, context) => {
    if (node.type === compiler_core_1.NodeTypes.INTERPOLATION) {
        node.content = processExpression(node.content, context);
    }
    else if (node.type === compiler_core_1.NodeTypes.ELEMENT) {
        // handle directives on element
        for (let i = 0; i < node.props.length; i++) {
            const dir = node.props[i];
            // do not process for v-on & v-for since they are special handled
            if (dir.type === compiler_core_1.NodeTypes.DIRECTIVE && dir.name !== 'for') {
                const exp = dir.exp;
                const arg = dir.arg;
                // do not process exp if this is v-on:arg - we need special handling
                // for wrapping inline statements.
                if (exp &&
                    exp.type === compiler_core_1.NodeTypes.SIMPLE_EXPRESSION &&
                    !(dir.name === 'on' && arg)) {
                    dir.exp = processExpression(exp, context, 
                    // slot args must be processed as function params
                    dir.name === 'slot');
                }
                if (arg && arg.type === compiler_core_1.NodeTypes.SIMPLE_EXPRESSION && !arg.isStatic) {
                    dir.arg = processExpression(arg, context);
                }
            }
        }
    }
};
exports.transformExpression = transformExpression;
// Important: since this function uses Node.js only dependencies, it should
// always be used with a leading !__BROWSER__ check so that it can be
// tree-shaken from the browser build.
function processExpression(node, context, 
// some expressions like v-slot props & v-for aliases should be parsed as
// function params
asParams = false, 
// v-on handler values may contain multiple statements
asRawStatements = false, localVars = Object.create(context.identifiers)) {
    if (!context.prefixIdentifiers || !node.content.trim()) {
        return node;
    }
    const { inline, bindingMetadata } = context;
    const rewriteIdentifier = (raw, parent, id) => {
        const type = (0, shared_1.hasOwn)(bindingMetadata, raw) && bindingMetadata[raw];
        if (inline) {
            // x = y
            const isAssignmentLVal = parent && parent.type === 'AssignmentExpression' && parent.left === id;
            // x++
            const isUpdateArg = parent && parent.type === 'UpdateExpression' && parent.argument === id;
            // ({ x } = y)
            const isDestructureAssignment = parent && (0, compiler_core_1.isInDestructureAssignment)(parent, parentStack);
            if (isConst(type) || localVars[raw]) {
                return raw;
            }
            else if (type === compiler_core_1.BindingTypes.SETUP_REF) {
                return `${raw}.value`;
            }
            else if (type === compiler_core_1.BindingTypes.SETUP_MAYBE_REF) {
                // const binding that may or may not be ref
                // if it's not a ref, then assignments don't make sense -
                // so we ignore the non-ref assignment case and generate code
                // that assumes the value to be a ref for more efficiency
                return isAssignmentLVal || isUpdateArg || isDestructureAssignment
                    ? `${raw}.value`
                    : `${context.helperString(compiler_core_1.UNREF)}(${raw})`;
            }
            else if (type === compiler_core_1.BindingTypes.SETUP_LET) {
                if (isAssignmentLVal) {
                    // let binding.
                    // this is a bit more tricky as we need to cover the case where
                    // let is a local non-ref value, and we need to replicate the
                    // right hand side value.
                    // x = y --> isRef(x) ? x.value = y : x = y
                    const { right: rVal, operator } = parent;
                    const rExp = rawExp.slice(rVal.start - 1, rVal.end - 1);
                    const rExpString = stringifyExpression(processExpression((0, compiler_core_1.createSimpleExpression)(rExp, false), context, false, false, knownIds));
                    return `${context.helperString(compiler_core_1.IS_REF)}(${raw})${context.isTS ? ` //@ts-ignore\n` : ``} ? ${raw}.value ${operator} ${rExpString} : ${raw}`;
                }
                else if (isUpdateArg) {
                    // make id replace parent in the code range so the raw update operator
                    // is removed
                    id.start = parent.start;
                    id.end = parent.end;
                    const { prefix: isPrefix, operator } = parent;
                    const prefix = isPrefix ? operator : ``;
                    const postfix = isPrefix ? `` : operator;
                    // let binding.
                    // x++ --> isRef(a) ? a.value++ : a++
                    return `${context.helperString(compiler_core_1.IS_REF)}(${raw})${context.isTS ? ` //@ts-ignore\n` : ``} ? ${prefix}${raw}.value${postfix} : ${prefix}${raw}${postfix}`;
                }
                else if (isDestructureAssignment) {
                    // TODO
                    // let binding in a destructure assignment - it's very tricky to
                    // handle both possible cases here without altering the original
                    // structure of the code, so we just assume it's not a ref here
                    // for now
                    return raw;
                }
                else {
                    return `${context.helperString(compiler_core_1.UNREF)}(${raw})`;
                }
            }
            else if (type === compiler_core_1.BindingTypes.PROPS) {
                // use __props which is generated by compileScript so in ts mode
                // it gets correct type
                return (0, shared_1.genPropsAccessExp)(raw);
            }
            else if (type === compiler_core_1.BindingTypes.PROPS_ALIASED) {
                // prop with a different local alias (from defineProps() destructure)
                return (0, shared_1.genPropsAccessExp)(bindingMetadata.__propsAliases[raw]);
            }
        }
        else {
            if ((type && type.startsWith('setup')) ||
                type === compiler_core_1.BindingTypes.LITERAL_CONST) {
                // setup bindings in non-inline mode
                return `$setup.${raw}`;
            }
            else if (type === compiler_core_1.BindingTypes.PROPS_ALIASED) {
                return `$props['${bindingMetadata.__propsAliases[raw]}']`;
            }
            else if (type) {
                return `$${type}.${raw}`;
            }
        }
        // fallback to ctx
        return `_ctx.${raw}`;
    };
    // fast path if expression is a simple identifier.
    const rawExp = node.content;
    // bail constant on parens (function invocation) and dot (member access)
    const bailConstant = rawExp.indexOf(`(`) > -1 || rawExp.indexOf('.') > 0;
    if ((0, compiler_core_1.isSimpleIdentifier)(rawExp)) {
        const isScopeVarReference = context.identifiers[rawExp];
        const isAllowedGlobal = (0, shared_1.isGloballyWhitelisted)(rawExp);
        const isLiteral = isLiteralWhitelisted(rawExp);
        const isFilter = context.filters.includes(rawExp);
        const isBuiltIn = isBuiltInIdentifier(rawExp);
        if (!asParams &&
            !isScopeVarReference &&
            !isAllowedGlobal &&
            !isLiteral &&
            !isFilter &&
            !isBuiltIn) {
            // const bindings exposed from setup can be skipped for patching but
            // cannot be hoisted to module scope
            if (isConst(bindingMetadata[node.content])) {
                node.constType = compiler_core_1.ConstantTypes.CAN_SKIP_PATCH;
            }
            node.content = rewriteIdentifier(rawExp);
        }
        else if (!isScopeVarReference) {
            if (isLiteral) {
                node.constType = compiler_core_1.ConstantTypes.CAN_STRINGIFY;
            }
            else {
                node.constType = compiler_core_1.ConstantTypes.CAN_HOIST;
            }
        }
        return node;
    }
    let ast;
    // exp needs to be parsed differently:
    // 1. Multiple inline statements (v-on, with presence of `;`): parse as raw
    //    exp, but make sure to pad with spaces for consistent ranges
    // 2. Expressions: wrap with parens (for e.g. object expressions)
    // 3. Function arguments (v-for, v-slot): place in a function argument position
    const source = asRawStatements
        ? ` ${rawExp} `
        : `(${rawExp})${asParams ? `=>{}` : ``}`;
    try {
        ast = (0, parser_1.parse)(source, {
            plugins: context.expressionPlugins,
        }).program;
    }
    catch (e) {
        context.onError((0, compiler_core_1.createCompilerError)(compiler_core_1.ErrorCodes.X_INVALID_EXPRESSION, node.loc, undefined, '\n' + source + '\n' + e.message));
        return node;
    }
    const ids = [];
    const parentStack = [];
    const knownIds = Object.create(context.identifiers);
    context.filters.forEach((name) => {
        knownIds[name] = 1;
    });
    (0, compiler_core_1.walkIdentifiers)(ast, (node, parent, _, isReferenced, isLocal) => {
        if ((0, compiler_core_1.isStaticPropertyKey)(node, parent)) {
            return;
        }
        const needPrefix = isReferenced && canPrefix(node);
        if (needPrefix && !isLocal) {
            if ((0, compiler_core_1.isStaticProperty)(parent) && parent.shorthand) {
                // property shorthand like { foo }, we need to add the key since
                // we rewrite the value
                ;
                node.prefix = `${node.name}: `;
            }
            node.name = rewriteIdentifier(node.name, parent, node);
            ids.push(node);
        }
        else {
            // The identifier is considered constant unless it's pointing to a
            // local scope variable (a v-for alias, or a v-slot prop)
            if (!(needPrefix && isLocal) && !bailConstant) {
                ;
                node.isConstant = true;
            }
            // also generate sub-expressions for other identifiers for better
            // source map support. (except for property keys which are static)
            ids.push(node);
        }
    }, true, // invoke on ALL identifiers
    parentStack, knownIds);
    // We break up the compound expression into an array of strings and sub
    // expressions (for identifiers that have been prefixed). In codegen, if
    // an ExpressionNode has the `.children` property, it will be used instead of
    // `.content`.
    const children = [];
    ids.sort((a, b) => a.start - b.start);
    ids.forEach((id, i) => {
        // range is offset by -1 due to the wrapping parens when parsed
        const start = id.start - 1;
        const end = id.end - 1;
        const last = ids[i - 1];
        const leadingText = rawExp.slice(last ? last.end - 1 : 0, start);
        if (leadingText.length || id.prefix) {
            children.push(leadingText + (id.prefix || ``));
        }
        const source = rawExp.slice(start, end);
        children.push((0, compiler_core_1.createSimpleExpression)(id.name, false, {
            source,
            start: (0, compiler_core_1.advancePositionWithClone)(node.loc.start, source, start),
            end: (0, compiler_core_1.advancePositionWithClone)(node.loc.start, source, end),
        }, id.isConstant ? compiler_core_1.ConstantTypes.CAN_STRINGIFY : compiler_core_1.ConstantTypes.NOT_CONSTANT));
        if (i === ids.length - 1 && end < rawExp.length) {
            children.push(rawExp.slice(end));
        }
    });
    let ret;
    if (children.length) {
        ret = (0, compiler_core_1.createCompoundExpression)(children, node.loc);
    }
    else {
        ret = node;
        ret.constType = bailConstant
            ? compiler_core_1.ConstantTypes.NOT_CONSTANT
            : compiler_core_1.ConstantTypes.CAN_STRINGIFY;
    }
    ret.identifiers = Object.keys(knownIds);
    return ret;
}
exports.processExpression = processExpression;
function canPrefix(id) {
    // skip whitelisted globals
    if ((0, shared_1.isGloballyWhitelisted)(id.name)) {
        return false;
    }
    // special case for webpack compilation
    if (id.name === 'require') {
        return false;
    }
    return true;
}
function stringifyExpression(exp) {
    if ((0, shared_1.isString)(exp)) {
        return exp;
    }
    else if (exp.type === compiler_core_1.NodeTypes.SIMPLE_EXPRESSION) {
        return exp.content;
    }
    else {
        return exp.children
            .map(stringifyExpression)
            .join('');
    }
}
const builtInIdentifiers = ['__l'];
function isBuiltInIdentifier(id) {
    if (!(0, shared_1.isString)(id)) {
        if (id.type !== compiler_core_1.NodeTypes.SIMPLE_EXPRESSION) {
            return false;
        }
        id = id.content;
    }
    return builtInIdentifiers.includes(id);
}
exports.isBuiltInIdentifier = isBuiltInIdentifier;
function isConst(type) {
    return (type === compiler_core_1.BindingTypes.SETUP_CONST ||
        type === compiler_core_1.BindingTypes.LITERAL_CONST ||
        type === compiler_core_1.BindingTypes.SETUP_REACTIVE_CONST);
}

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615314, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.PARSE_VIRTUAL_HOST_CLASS = exports.MERGE_PART_CLASS = exports.GEN_UNI_ELEMENT_ID = exports.SET_UNI_ELEMENT_STYLE = exports.SET_UNI_ELEMENT_ID = exports.WITH_MODEL_MODIFIERS = exports.TO_DISPLAY_STRING = exports.NORMALIZE_CLASS = exports.STRINGIFY_STYLE = exports.WITH_SCOPED_SLOT = exports.DYNAMIC_SLOT = exports.RENDER_SLOT = exports.RENDER_PROPS = exports.HYPHENATE = exports.CAMELIZE = exports.SET_REF = exports.EXTEND = exports.V_FOR = exports.V_ON = void 0;
const compiler_core_1 = require("@vue/compiler-core");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
exports.V_ON = Symbol(`vOn`);
exports.V_FOR = Symbol(`vFor`);
exports.EXTEND = Symbol(`extend`);
exports.SET_REF = Symbol(`setRef`);
exports.CAMELIZE = Symbol(`camelize`);
exports.HYPHENATE = Symbol(`hyphenate`);
exports.RENDER_PROPS = Symbol(`renderProps`);
exports.RENDER_SLOT = Symbol(`renderSlot`);
exports.DYNAMIC_SLOT = Symbol(`dynamicSlot`);
exports.WITH_SCOPED_SLOT = Symbol(`withScopedSlot`);
exports.STRINGIFY_STYLE = Symbol(`stringifyStyle`);
exports.NORMALIZE_CLASS = Symbol(`normalizeClass`);
exports.TO_DISPLAY_STRING = Symbol(`toDisplayString`);
exports.WITH_MODEL_MODIFIERS = Symbol(`withModelModifiers`);
exports.SET_UNI_ELEMENT_ID = Symbol(`setUniElementId`);
exports.SET_UNI_ELEMENT_STYLE = Symbol(`setUniElementStyle`);
exports.GEN_UNI_ELEMENT_ID = Symbol(`genUniElementId`);
exports.MERGE_PART_CLASS = Symbol(`mergePartClass`);
exports.PARSE_VIRTUAL_HOST_CLASS = Symbol(`parseVirtualHostClass`);
(0, compiler_core_1.registerRuntimeHelpers)({
    [exports.V_ON]: 'o',
    [exports.V_FOR]: 'f',
    [exports.EXTEND]: 'e',
    [exports.SET_REF]: 'sr',
    [exports.CAMELIZE]: 'c',
    [exports.HYPHENATE]: 'h',
    [exports.RENDER_PROPS]: 'p',
    [exports.RENDER_SLOT]: 'r',
    [exports.DYNAMIC_SLOT]: 'd',
    [exports.WITH_SCOPED_SLOT]: 'w',
    [exports.STRINGIFY_STYLE]: 's',
    [exports.NORMALIZE_CLASS]: 'n',
    [exports.TO_DISPLAY_STRING]: 't',
    [exports.WITH_MODEL_MODIFIERS]: 'm',
    [uni_cli_shared_1.STRINGIFY_JSON]: 'j',
    [exports.SET_UNI_ELEMENT_ID]: 'sei',
    [exports.SET_UNI_ELEMENT_STYLE]: 'ses',
    [exports.GEN_UNI_ELEMENT_ID]: 'gei',
    [exports.MERGE_PART_CLASS]: 'mpc',
    [exports.PARSE_VIRTUAL_HOST_CLASS]: 'pvhc',
});

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615315, function(require, module, exports) {

Object.defineProperty(exports, "__esModule", { value: true });
exports.createVSlotCallExpression = exports.rewriteScopedSlotVForScope = exports.findCurrentPath = exports.findSlotName = exports.rewriteVSlot = exports.transformSlot = void 0;
const types_1 = require("@babel/types");
const shared_1 = require("@vue/shared");
const compiler_core_1 = require("@vue/compiler-core");
const uni_shared_1 = require("@dcloudio/uni-shared");
const uni_cli_shared_1 = require("@dcloudio/uni-cli-shared");
const runtimeHelpers_1 = require("../runtimeHelpers");
const ast_1 = require("../ast");
const codegen_1 = require("../codegen");
const transform_1 = require("../transform");
const utils_1 = require("./utils");
const vFor_1 = require("./vFor");
const runtimeHelpers_2 = require("../runtimeHelpers");
const transformSlot = (node, context) => {
    if (!(0, uni_cli_shared_1.isUserComponent)(node, context)) {
        return;
    }
    const { tag, children } = node;
    const slots = new Set();
    const onComponentSlot = (0, compiler_core_1.findDir)(node, 'slot', true);
    const implicitDefaultChildren = [];
    const isMiniProgramComponent = context.isMiniProgramComponent(tag);
    for (let i = 0; i < children.length; i++) {
        const slotElement = children[i];
        if ((0, uni_cli_shared_1.isElementNode)(slotElement) &&
            slotElement.tag === 'template' &&
            slotElement.children.filter((node) => !(0, uni_cli_shared_1.isCommentNode)(node)).length === 0) {
            // 如果是 template 且 没有子节点 或者 子节点 都是 注释节点，直接移除节点
            children.splice(i, 1);
            i -= 1;
            continue;
        }
        let slotDir;
        if (!(0, compiler_core_1.isTemplateNode)(slotElement) ||
            !(slotDir = (0, compiler_core_1.findDir)(slotElement, 'slot', true))) {
            // not a <template v-slot>, skip.
            if (!(0, uni_cli_shared_1.isCommentNode)(slotElement)) {
                implicitDefaultChildren.push(slotElement);
            }
            continue;
        }
        if (onComponentSlot) {
            // already has on-component slot - this is incorrect usage.
            context.onError((0, compiler_core_1.createCompilerError)(compiler_core_1.ErrorCodes.X_V_SLOT_MIXED_SLOT_USAGE, slotDir.loc));
            break;
        }
        if (!slotDir.arg) {
            // v-slot => v-slot:default
            slotDir.arg = (0, compiler_core_1.createSimpleExpression)('default', true);
        }
        const slotName = transformTemplateSlotElement(slotDir, slotElement, node, context);
        // 小程序组件默认插槽，直接移除<template #default>节点
        if (isMiniProgramComponent) {
            if (slotName === 'default' && slotElement.children.length === 1) {
                children.splice(i, 1, slotElement.children[0]);
            }
            continue;
        }
        if (slotName) {
            slots.add(slotName);
        }
    }
    if (isMiniProgramComponent) {
        return;
    }
    if (implicitDefaultChildren.length) {
        slots.add(uni_shared_1.SLOT_DEFAULT_NAME);
    }
    if (onComponentSlot) {
        // <unicloud-db v-slot:default="{data, loading, error, options}"/>
        // => <unicloud-db collection=""><template v-slot:default="{data, loading, error, options}"/></unicloud-db>
        slots.add(uni_shared_1.SLOT_DEFAULT_NAME);
        const templateNode = createTemplateNode(onComponentSlot, implicitDefaultChildren);
        transformTemplateSlotElement(onComponentSlot, templateNode, node, context);
        node.children = [templateNode];
    }
    // 不支持 $slots, 则自动补充 props
    if (slots.size && !context.miniProgram.slot.$slots) {
        const slotsArr = [...slots];
        const hasDynamic = slotsArr.find((name) => !(0, shared_1.isString)(name));
        let value;
        if (hasDynamic) {
            const children = [];
            const len = slotsArr.length - 1;
            slotsArr.forEach((name, index) => {
                if ((0, shared_1.isString)(name)) {
                    children.push(`'${(0, uni_shared_1.dynamicSlotName)(name)}'`);
                }
                else {
                    children.push(name);
                }
                if (index < len) {
                    children.push(',');
                }
            });
            value = (0, compiler_core_1.createCompoundExpression)([
                context.helperString(runtimeHelpers_2.DYNAMIC_SLOT) + '([',
                ...children,
                '])',
            ]);
        }
        else {
            value = `[${slotsArr
                .map((name) => `'${(0, uni_shared_1.dynamicSlotName)(name)}'`)
                .join(',')}]`;
        }
        node.props.unshift((0, uni_cli_shared_1.createBindDirectiveNode)(utils_1.ATTR_VUE_SLOTS, value));
    }
};
exports.transformSlot = transformSlot;
function rewriteVSlot(dir, context) {
    dir.arg = (0, utils_1.rewriteExpression)((0, compiler_core_1.createCompoundExpression)([
        context.helperString(runtimeHelpers_2.DYNAMIC_SLOT) + '(',
        dir.arg,
        ')',
    ]), context);
}
exports.rewriteVSlot = rewriteVSlot;
function transformTemplateSlotElement(slotDir, slotTemplate, slotComponent, context) {
    const slotName = findSlotName(slotDir);
    if (!slotName) {
        return;
    }
    const { exp } = slotDir;
    // non scoped slots
    if (!exp) {
        return slotName;
    }
    // empty
    if (exp.type === compiler_core_1.NodeTypes.SIMPLE_EXPRESSION && !exp.content.trim()) {
        return slotName;
    }
    // 使用vFor来简单处理scoped slot作用域问题
    slotTemplate.children = [
        createVForTemplate(slotTemplate, { name: slotName, value: (0, codegen_1.genExpr)(exp), slotComponent }, context),
    ];
    if (context.miniProgram.slot.dynamicSlotNames) {
        // 已经在 vFor 中补充 slot，故需要移除 slotTemplate 中的
        const index = slotTemplate.props.indexOf(slotDir);
        if (index > -1) {
            slotTemplate.props.splice(index, 1);
        }
    }
    // v-slot="slotProps" => v-slot 避免 transformIdentifier 生成 slotProps 的变量声明
    slotDir.exp = undefined;
    return slotName;
}
function createTemplateNode(slotDir, children) {
    return {
        type: compiler_core_1.NodeTypes.ELEMENT,
        tag: 'template',
        tagType: compiler_core_1.ElementTypes.TEMPLATE,
        loc: compiler_core_1.locStub,
        isSelfClosing: false,
        codegenNode: undefined,
        ns: 0,
        props: [slotDir],
        children,
    };
}
function findSlotName(slotDir) {
    if (!slotDir.arg) {
        return uni_shared_1.SLOT_DEFAULT_NAME;
    }
    if ((0, compiler_core_1.isStaticExp)(slotDir.arg)) {
        return slotDir.arg.content;
    }
    return slotDir.arg;
}
exports.findSlotName = findSlotName;
function findCurrentVForValueAlias(context) {
    let scope = context.currentScope;
    while (scope) {
        if ((0, transform_1.isVForScope)(scope)) {
            return scope.valueAlias;
        }
        scope = scope.parent;
    }
    return '';
}
function createVForTemplate(slotElement, { name, value, slotComponent, }, context) {
    const slotName = 's' + context.scopes.vFor;
    const keyProp = (0, uni_cli_shared_1.createBindDirectiveNode)('key', 'i' + context.scopes.vFor);
    const source = (0, shared_1.isString)(name) ? `'${name}'` : (0, codegen_1.genExpr)(name);
    const vForProp = {
        type: compiler_core_1.NodeTypes.DIRECTIVE,
        name: 'for',
        loc: compiler_core_1.locStub,
        modifiers: [],
        arg: undefined,
        exp: (0, compiler_core_1.createSimpleExpression)(`(${value}, ${slotName}) in ${utils_1.SCOPED_SLOT_IDENTIFIER}(${source}, ${findCurrentVForValueAlias(context) || `''`})`),
    };
    const props = [vForProp, keyProp];
    if (context.miniProgram.slot.dynamicSlotNames) {
        props.push((0, uni_cli_shared_1.createBindDirectiveNode)('slot', slotName));
    }
    return {
        loc: slotElement.loc,
        ns: 0,
        tag: 'template',
        type: compiler_core_1.NodeTypes.ELEMENT,
        tagType: compiler_core_1.ElementTypes.TEMPLATE,
        props,
        isSelfClosing: false,
        codegenNode: undefined,
        children: slotElement.children,
        slotComponent,
    };
}
const slotNameRE = /\('(.*)',/;
/**
 * ('default','') => default
 * @param source
 * @returns
 */
function findCurrentSlotName(source) {
    return (0, types_1.stringLiteral)((0, uni_shared_1.dynamicSlotName)(source.children[1].match(slotNameRE)[1]));
}
function createPathBinaryExpr(scope, computed = true) {
    return (0, types_1.binaryExpression)('+', (0, types_1.binaryExpression)('+', (0, types_1.stringLiteral)(parseVForPath(scope.sourceAlias) + (computed ? '[' : '.')), (0, types_1.identifier)(scope.indexAlias)), (0, types_1.stringLiteral)(computed ? '].' : '.'));
}
function findCurrentPath(id, scope) {
    let parent = scope.parent;
    let binaryExpr = null;
    while (parent) {
        if ((0, transform_1.isVForScope)(parent)) {
            // const computed = !isScopedSlotVFor(parent)
            if (!binaryExpr) {
                binaryExpr = createPathBinaryExpr(parent);
            }
            else {
                binaryExpr = (0, types_1.binaryExpression)('+', createPathBinaryExpr(parent), binaryExpr);
            }
        }
        parent = parent.parent;
    }
    return ((binaryExpr && (0, types_1.binaryExpression)('+', binaryExpr, (0, types_1.stringLiteral)(id))) ||
        (0, types_1.stringLiteral)(id));
}
exports.findCurrentPath = findCurrentPath;
function findCurrentVueIdExpr(node, context) {
    if (!node) {
        return (0, types_1.stringLiteral)('');
    }
    const vueIdProp = (0, compiler_core_1.findProp)(node, utils_1.ATTR_VUE_ID);
    if (vueIdProp.type === compiler_core_1.NodeTypes.ATTRIBUTE) {
        return (0, types_1.stringLiteral)(vueIdProp.value.content);
    }
    return (0, ast_1.parseExpr)((0, codegen_1.genExpr)(vueIdProp.exp), context) || (0, types_1.stringLiteral)('');
}
/**
 * 目前无用
 * @param vForScope
 * @param parentScope
 * @param context
 */
function rewriteScopedSlotVForScope(vForScope, parentScope, context) {
    // 生成一个新的sourceAlias，用于scopedSlots
    const { source, sourceExpr } = vForScope;
    vForScope.sourceAlias = (0, utils_1.rewriteExpressionWithoutProperty)(source, context, sourceExpr, parentScope).content;
}
exports.rewriteScopedSlotVForScope = rewriteScopedSlotVForScope;
function parseVForPath(id) {
    return id.includes('.') ? id.split('.')[1] : id;
}
function createVSlotCallExpression(slotComponent, vForScope, context) {
    const { source, sourceAlias } = vForScope;
    const id = parseVForPath(sourceAlias);
    return (0, types_1.callExpression)((0, types_1.identifier)(context.helperString(runtimeHelpers_1.WITH_SCOPED_SLOT)), [
        (0, vFor_1.createVForArrowFunctionExpression)(vForScope),
        (0, types_1.objectExpression)([
            // 插槽名称，数据更新 path，vueId
            (0, types_1.objectProperty)((0, types_1.identifier)('name'), findCurrentSlotName(source)),
            // 暂不生成 path
            (0, types_1.objectProperty)((0, types_1.identifier)('path'), findCurrentPath(id, vForScope)),
            (0, types_1.objectProperty)((0, types_1.identifier)('vueId'), findCurrentVueIdExpr(slotComponent, context)),
        ]),
    ]);
}
exports.createVSlotCallExpression = createVSlotCallExpression;

}, function(modId) { var map = {"../runtimeHelpers":1781530615314,"../ast":1781530615310,"../codegen":1781530615307,"./vFor":1781530615309}; return __REQUIRE__(map[modId], modId); })
return __REQUIRE__(1781530615304);
})()
//miniprogram-npm-outsideDeps=["@vue/shared","@vue/compiler-core","./compile","./transforms/utils","./transforms/vOn","./transforms/vModel","@babel/generator","@dcloudio/uni-cli-shared","../transform","@babel/types","./utils","@babel/parser","@dcloudio/uni-shared"]
//# sourceMappingURL=index.js.map