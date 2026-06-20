module.exports = (function() {
var __MODS__ = {};
var __DEFINE__ = function(modId, func, req) { var m = { exports: {}, _tempexports: {} }; __MODS__[modId] = { status: 0, func: func, req: req, m: m }; };
var __REQUIRE__ = function(modId, source) { if(!__MODS__[modId]) return require(source); if(!__MODS__[modId].status) { var m = __MODS__[modId].m; m._exports = m._tempexports; var desp = Object.getOwnPropertyDescriptor(m, "exports"); if (desp && desp.configurable) Object.defineProperty(m, "exports", { set: function (val) { if(typeof val === "object" && val !== m._exports) { m._exports.__proto__ = val.__proto__; Object.keys(val).forEach(function (k) { m._exports[k] = val[k]; }); } m._tempexports = val }, get: function () { return m._tempexports; } }); __MODS__[modId].status = 1; __MODS__[modId].func(__MODS__[modId].req, m, m.exports); } return __MODS__[modId].m.exports; };
var __REQUIRE_WILDCARD__ = function(obj) { if(obj && obj.__esModule) { return obj; } else { var newObj = {}; if(obj != null) { for(var k in obj) { if (Object.prototype.hasOwnProperty.call(obj, k)) newObj[k] = obj[k]; } } newObj.default = obj; return newObj; } };
var __REQUIRE_DEFAULT__ = function(obj) { return obj && obj.__esModule ? obj.default : obj; };
__DEFINE__(1781530615284, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = generate;
var _sourceMap = require("./source-map.js");
var _printer = require("./printer.js");
function normalizeOptions(code, opts) {
  const format = {
    auxiliaryCommentBefore: opts.auxiliaryCommentBefore,
    auxiliaryCommentAfter: opts.auxiliaryCommentAfter,
    shouldPrintComment: opts.shouldPrintComment,
    retainLines: opts.retainLines,
    retainFunctionParens: opts.retainFunctionParens,
    comments: opts.comments == null || opts.comments,
    compact: opts.compact,
    minified: opts.minified,
    concise: opts.concise,
    indent: {
      adjustMultilineComment: true,
      style: "  "
    },
    jsescOption: Object.assign({
      quotes: "double",
      wrap: true,
      minimal: false
    }, opts.jsescOption),
    topicToken: opts.topicToken,
    importAttributesKeyword: opts.importAttributesKeyword
  };
  {
    var _opts$recordAndTupleS;
    format.decoratorsBeforeExport = opts.decoratorsBeforeExport;
    format.jsescOption.json = opts.jsonCompatibleStrings;
    format.recordAndTupleSyntaxType = (_opts$recordAndTupleS = opts.recordAndTupleSyntaxType) != null ? _opts$recordAndTupleS : "hash";
  }
  if (format.minified) {
    format.compact = true;
    format.shouldPrintComment = format.shouldPrintComment || (() => format.comments);
  } else {
    format.shouldPrintComment = format.shouldPrintComment || (value => format.comments || value.includes("@license") || value.includes("@preserve"));
  }
  if (format.compact === "auto") {
    format.compact = typeof code === "string" && code.length > 500000;
    if (format.compact) {
      console.error("[BABEL] Note: The code generator has deoptimised the styling of " + `${opts.filename} as it exceeds the max of ${"500KB"}.`);
    }
  }
  if (format.compact) {
    format.indent.adjustMultilineComment = false;
  }
  const {
    auxiliaryCommentBefore,
    auxiliaryCommentAfter,
    shouldPrintComment
  } = format;
  if (auxiliaryCommentBefore && !shouldPrintComment(auxiliaryCommentBefore)) {
    format.auxiliaryCommentBefore = undefined;
  }
  if (auxiliaryCommentAfter && !shouldPrintComment(auxiliaryCommentAfter)) {
    format.auxiliaryCommentAfter = undefined;
  }
  return format;
}
{
  exports.CodeGenerator = class CodeGenerator {
    constructor(ast, opts = {}, code) {
      this._ast = void 0;
      this._format = void 0;
      this._map = void 0;
      this._ast = ast;
      this._format = normalizeOptions(code, opts);
      this._map = opts.sourceMaps ? new _sourceMap.default(opts, code) : null;
    }
    generate() {
      const printer = new _printer.default(this._format, this._map);
      return printer.generate(this._ast);
    }
  };
}
function generate(ast, opts = {}, code) {
  const format = normalizeOptions(code, opts);
  const map = opts.sourceMaps ? new _sourceMap.default(opts, code) : null;
  const printer = new _printer.default(format, map);
  return printer.generate(ast);
}

//# sourceMappingURL=index.js.map

}, function(modId) {var map = {"./source-map.js":1781530615285,"./printer.js":1781530615286}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615285, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
var _genMapping = require("@jridgewell/gen-mapping");
var _traceMapping = require("@jridgewell/trace-mapping");
class SourceMap {
  constructor(opts, code) {
    var _opts$sourceFileName;
    this._map = void 0;
    this._rawMappings = void 0;
    this._sourceFileName = void 0;
    this._lastGenLine = 0;
    this._lastSourceLine = 0;
    this._lastSourceColumn = 0;
    this._inputMap = void 0;
    const map = this._map = new _genMapping.GenMapping({
      sourceRoot: opts.sourceRoot
    });
    this._sourceFileName = (_opts$sourceFileName = opts.sourceFileName) == null ? void 0 : _opts$sourceFileName.replace(/\\/g, "/");
    this._rawMappings = undefined;
    if (opts.inputSourceMap) {
      this._inputMap = new _traceMapping.TraceMap(opts.inputSourceMap);
      const resolvedSources = this._inputMap.resolvedSources;
      if (resolvedSources.length) {
        for (let i = 0; i < resolvedSources.length; i++) {
          var _this$_inputMap$sourc;
          (0, _genMapping.setSourceContent)(map, resolvedSources[i], (_this$_inputMap$sourc = this._inputMap.sourcesContent) == null ? void 0 : _this$_inputMap$sourc[i]);
        }
      }
    }
    if (typeof code === "string" && !opts.inputSourceMap) {
      (0, _genMapping.setSourceContent)(map, this._sourceFileName, code);
    } else if (typeof code === "object") {
      for (const sourceFileName of Object.keys(code)) {
        (0, _genMapping.setSourceContent)(map, sourceFileName.replace(/\\/g, "/"), code[sourceFileName]);
      }
    }
  }
  get() {
    return (0, _genMapping.toEncodedMap)(this._map);
  }
  getDecoded() {
    return (0, _genMapping.toDecodedMap)(this._map);
  }
  getRawMappings() {
    return this._rawMappings || (this._rawMappings = (0, _genMapping.allMappings)(this._map));
  }
  mark(generated, line, column, identifierName, identifierNamePos, filename) {
    var _originalMapping;
    this._rawMappings = undefined;
    let originalMapping;
    if (line != null) {
      if (this._inputMap) {
        originalMapping = (0, _traceMapping.originalPositionFor)(this._inputMap, {
          line,
          column
        });
        if (!originalMapping.name && identifierNamePos) {
          const originalIdentifierMapping = (0, _traceMapping.originalPositionFor)(this._inputMap, identifierNamePos);
          if (originalIdentifierMapping.name) {
            identifierName = originalIdentifierMapping.name;
          }
        }
      } else {
        originalMapping = {
          source: (filename == null ? void 0 : filename.replace(/\\/g, "/")) || this._sourceFileName,
          line: line,
          column: column
        };
      }
    }
    (0, _genMapping.maybeAddMapping)(this._map, {
      name: identifierName,
      generated,
      source: (_originalMapping = originalMapping) == null ? void 0 : _originalMapping.source,
      original: originalMapping
    });
  }
}
exports.default = SourceMap;

//# sourceMappingURL=source-map.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615286, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
var _buffer = require("./buffer.js");
var n = require("./node/index.js");
var _t = require("@babel/types");
var generatorFunctions = require("./generators/index.js");
const {
  isFunction,
  isStatement,
  isClassBody,
  isTSInterfaceBody,
  isTSEnumDeclaration
} = _t;
const SCIENTIFIC_NOTATION = /e/i;
const ZERO_DECIMAL_INTEGER = /\.0+$/;
const HAS_NEWLINE = /[\n\r\u2028\u2029]/;
const HAS_NEWLINE_OR_BlOCK_COMMENT_END = /[\n\r\u2028\u2029]|\*\//;
const {
  needsParens
} = n;
class Printer {
  constructor(format, map) {
    this.inForStatementInit = false;
    this.tokenContext = 0;
    this._currentNode = null;
    this._indent = 0;
    this._indentRepeat = 0;
    this._insideAux = false;
    this._parenPushNewlineState = null;
    this._noLineTerminator = false;
    this._printAuxAfterOnNextUserNode = false;
    this._printedComments = new Set();
    this._endsWithInteger = false;
    this._endsWithWord = false;
    this._endsWithDiv = false;
    this._lastCommentLine = 0;
    this._endsWithInnerRaw = false;
    this._indentInnerComments = true;
    this.format = format;
    this._indentRepeat = format.indent.style.length;
    this._inputMap = map == null ? void 0 : map._inputMap;
    this._buf = new _buffer.default(map, format.indent.style[0]);
  }
  enterForStatementInit(val) {
    const old = this.inForStatementInit;
    if (old === val) return () => {};
    this.inForStatementInit = val;
    return () => {
      this.inForStatementInit = old;
    };
  }
  generate(ast) {
    this.print(ast);
    this._maybeAddAuxComment();
    return this._buf.get();
  }
  indent() {
    if (this.format.compact || this.format.concise) return;
    this._indent++;
  }
  dedent() {
    if (this.format.compact || this.format.concise) return;
    this._indent--;
  }
  semicolon(force = false) {
    this._maybeAddAuxComment();
    if (force) {
      this._appendChar(59);
    } else {
      this._queue(59);
    }
    this._noLineTerminator = false;
  }
  rightBrace(node) {
    if (this.format.minified) {
      this._buf.removeLastSemicolon();
    }
    this.sourceWithOffset("end", node.loc, -1);
    this.tokenChar(125);
  }
  rightParens(node) {
    this.sourceWithOffset("end", node.loc, -1);
    this.tokenChar(41);
  }
  space(force = false) {
    if (this.format.compact) return;
    if (force) {
      this._space();
    } else if (this._buf.hasContent()) {
      const lastCp = this.getLastChar();
      if (lastCp !== 32 && lastCp !== 10) {
        this._space();
      }
    }
  }
  word(str, noLineTerminatorAfter = false) {
    this.tokenContext = 0;
    this._maybePrintInnerComments();
    if (this._endsWithWord || this._endsWithDiv && str.charCodeAt(0) === 47) {
      this._space();
    }
    this._maybeAddAuxComment();
    this._append(str, false);
    this._endsWithWord = true;
    this._noLineTerminator = noLineTerminatorAfter;
  }
  number(str, number) {
    function isNonDecimalLiteral(str) {
      if (str.length > 2 && str.charCodeAt(0) === 48) {
        const secondChar = str.charCodeAt(1);
        return secondChar === 98 || secondChar === 111 || secondChar === 120;
      }
      return false;
    }
    this.word(str);
    this._endsWithInteger = Number.isInteger(number) && !isNonDecimalLiteral(str) && !SCIENTIFIC_NOTATION.test(str) && !ZERO_DECIMAL_INTEGER.test(str) && str.charCodeAt(str.length - 1) !== 46;
  }
  token(str, maybeNewline = false) {
    this.tokenContext = 0;
    this._maybePrintInnerComments();
    const lastChar = this.getLastChar();
    const strFirst = str.charCodeAt(0);
    if (lastChar === 33 && (str === "--" || strFirst === 61) || strFirst === 43 && lastChar === 43 || strFirst === 45 && lastChar === 45 || strFirst === 46 && this._endsWithInteger) {
      this._space();
    }
    this._maybeAddAuxComment();
    this._append(str, maybeNewline);
    this._noLineTerminator = false;
  }
  tokenChar(char) {
    this.tokenContext = 0;
    this._maybePrintInnerComments();
    const lastChar = this.getLastChar();
    if (char === 43 && lastChar === 43 || char === 45 && lastChar === 45 || char === 46 && this._endsWithInteger) {
      this._space();
    }
    this._maybeAddAuxComment();
    this._appendChar(char);
    this._noLineTerminator = false;
  }
  newline(i = 1, force) {
    if (i <= 0) return;
    if (!force) {
      if (this.format.retainLines || this.format.compact) return;
      if (this.format.concise) {
        this.space();
        return;
      }
    }
    if (i > 2) i = 2;
    i -= this._buf.getNewlineCount();
    for (let j = 0; j < i; j++) {
      this._newline();
    }
    return;
  }
  endsWith(char) {
    return this.getLastChar() === char;
  }
  getLastChar() {
    return this._buf.getLastChar();
  }
  endsWithCharAndNewline() {
    return this._buf.endsWithCharAndNewline();
  }
  removeTrailingNewline() {
    this._buf.removeTrailingNewline();
  }
  exactSource(loc, cb) {
    if (!loc) {
      cb();
      return;
    }
    this._catchUp("start", loc);
    this._buf.exactSource(loc, cb);
  }
  source(prop, loc) {
    if (!loc) return;
    this._catchUp(prop, loc);
    this._buf.source(prop, loc);
  }
  sourceWithOffset(prop, loc, columnOffset) {
    if (!loc) return;
    this._catchUp(prop, loc);
    this._buf.sourceWithOffset(prop, loc, columnOffset);
  }
  sourceIdentifierName(identifierName, pos) {
    if (!this._buf._canMarkIdName) return;
    const sourcePosition = this._buf._sourcePosition;
    sourcePosition.identifierNamePos = pos;
    sourcePosition.identifierName = identifierName;
  }
  _space() {
    this._queue(32);
  }
  _newline() {
    this._queue(10);
  }
  _append(str, maybeNewline) {
    this._maybeAddParen(str);
    this._maybeIndent(str.charCodeAt(0));
    this._buf.append(str, maybeNewline);
    this._endsWithWord = false;
    this._endsWithInteger = false;
    this._endsWithDiv = false;
  }
  _appendChar(char) {
    this._maybeAddParenChar(char);
    this._maybeIndent(char);
    this._buf.appendChar(char);
    this._endsWithWord = false;
    this._endsWithInteger = false;
    this._endsWithDiv = false;
  }
  _queue(char) {
    this._maybeAddParenChar(char);
    this._maybeIndent(char);
    this._buf.queue(char);
    this._endsWithWord = false;
    this._endsWithInteger = false;
  }
  _maybeIndent(firstChar) {
    if (this._indent && firstChar !== 10 && this.endsWith(10)) {
      this._buf.queueIndentation(this._getIndent());
    }
  }
  _shouldIndent(firstChar) {
    if (this._indent && firstChar !== 10 && this.endsWith(10)) {
      return true;
    }
  }
  _maybeAddParenChar(char) {
    const parenPushNewlineState = this._parenPushNewlineState;
    if (!parenPushNewlineState) return;
    if (char === 32) {
      return;
    }
    if (char !== 10) {
      this._parenPushNewlineState = null;
      return;
    }
    this.tokenChar(40);
    this.indent();
    parenPushNewlineState.printed = true;
  }
  _maybeAddParen(str) {
    const parenPushNewlineState = this._parenPushNewlineState;
    if (!parenPushNewlineState) return;
    const len = str.length;
    let i;
    for (i = 0; i < len && str.charCodeAt(i) === 32; i++) continue;
    if (i === len) {
      return;
    }
    const cha = str.charCodeAt(i);
    if (cha !== 10) {
      if (cha !== 47 || i + 1 === len) {
        this._parenPushNewlineState = null;
        return;
      }
      const chaPost = str.charCodeAt(i + 1);
      if (chaPost === 42) {
        return;
      } else if (chaPost !== 47) {
        this._parenPushNewlineState = null;
        return;
      }
    }
    this.tokenChar(40);
    this.indent();
    parenPushNewlineState.printed = true;
  }
  catchUp(line) {
    if (!this.format.retainLines) return;
    const count = line - this._buf.getCurrentLine();
    for (let i = 0; i < count; i++) {
      this._newline();
    }
  }
  _catchUp(prop, loc) {
    var _loc$prop;
    if (!this.format.retainLines) return;
    const line = loc == null || (_loc$prop = loc[prop]) == null ? void 0 : _loc$prop.line;
    if (line != null) {
      const count = line - this._buf.getCurrentLine();
      for (let i = 0; i < count; i++) {
        this._newline();
      }
    }
  }
  _getIndent() {
    return this._indentRepeat * this._indent;
  }
  printTerminatorless(node, isLabel) {
    if (isLabel) {
      this._noLineTerminator = true;
      this.print(node);
    } else {
      const terminatorState = {
        printed: false
      };
      this._parenPushNewlineState = terminatorState;
      this.print(node);
      if (terminatorState.printed) {
        this.dedent();
        this.newline();
        this.tokenChar(41);
      }
    }
  }
  print(node, noLineTerminatorAfter, trailingCommentsLineOffset, forceParens) {
    var _node$extra, _node$leadingComments;
    if (!node) return;
    this._endsWithInnerRaw = false;
    const nodeType = node.type;
    const format = this.format;
    const oldConcise = format.concise;
    if (node._compact) {
      format.concise = true;
    }
    const printMethod = this[nodeType];
    if (printMethod === undefined) {
      throw new ReferenceError(`unknown node of type ${JSON.stringify(nodeType)} with constructor ${JSON.stringify(node.constructor.name)}`);
    }
    const parent = this._currentNode;
    this._currentNode = node;
    const oldInAux = this._insideAux;
    this._insideAux = node.loc == null;
    this._maybeAddAuxComment(this._insideAux && !oldInAux);
    const parenthesized = (_node$extra = node.extra) == null ? void 0 : _node$extra.parenthesized;
    let shouldPrintParens = forceParens || parenthesized && format.retainFunctionParens && nodeType === "FunctionExpression" || needsParens(node, parent, this.tokenContext, this.inForStatementInit);
    if (!shouldPrintParens && parenthesized && (_node$leadingComments = node.leadingComments) != null && _node$leadingComments.length && node.leadingComments[0].type === "CommentBlock") {
      const parentType = parent == null ? void 0 : parent.type;
      switch (parentType) {
        case "ExpressionStatement":
        case "VariableDeclarator":
        case "AssignmentExpression":
        case "ReturnStatement":
          break;
        case "CallExpression":
        case "OptionalCallExpression":
        case "NewExpression":
          if (parent.callee !== node) break;
        default:
          shouldPrintParens = true;
      }
    }
    let exitInForStatementInit;
    if (shouldPrintParens) {
      this.tokenChar(40);
      this._endsWithInnerRaw = false;
      exitInForStatementInit = this.enterForStatementInit(false);
    }
    this._lastCommentLine = 0;
    this._printLeadingComments(node, parent);
    const loc = nodeType === "Program" || nodeType === "File" ? null : node.loc;
    this.exactSource(loc, printMethod.bind(this, node, parent));
    if (shouldPrintParens) {
      this._printTrailingComments(node, parent);
      this.tokenChar(41);
      this._noLineTerminator = noLineTerminatorAfter;
      exitInForStatementInit();
    } else if (noLineTerminatorAfter && !this._noLineTerminator) {
      this._noLineTerminator = true;
      this._printTrailingComments(node, parent);
    } else {
      this._printTrailingComments(node, parent, trailingCommentsLineOffset);
    }
    this._currentNode = parent;
    format.concise = oldConcise;
    this._insideAux = oldInAux;
    this._endsWithInnerRaw = false;
  }
  _maybeAddAuxComment(enteredPositionlessNode) {
    if (enteredPositionlessNode) this._printAuxBeforeComment();
    if (!this._insideAux) this._printAuxAfterComment();
  }
  _printAuxBeforeComment() {
    if (this._printAuxAfterOnNextUserNode) return;
    this._printAuxAfterOnNextUserNode = true;
    const comment = this.format.auxiliaryCommentBefore;
    if (comment) {
      this._printComment({
        type: "CommentBlock",
        value: comment
      }, 0);
    }
  }
  _printAuxAfterComment() {
    if (!this._printAuxAfterOnNextUserNode) return;
    this._printAuxAfterOnNextUserNode = false;
    const comment = this.format.auxiliaryCommentAfter;
    if (comment) {
      this._printComment({
        type: "CommentBlock",
        value: comment
      }, 0);
    }
  }
  getPossibleRaw(node) {
    const extra = node.extra;
    if ((extra == null ? void 0 : extra.raw) != null && extra.rawValue != null && node.value === extra.rawValue) {
      return extra.raw;
    }
  }
  printJoin(nodes, opts = {}) {
    if (!(nodes != null && nodes.length)) return;
    let {
      indent
    } = opts;
    if (indent == null && this.format.retainLines) {
      var _nodes$0$loc;
      const startLine = (_nodes$0$loc = nodes[0].loc) == null ? void 0 : _nodes$0$loc.start.line;
      if (startLine != null && startLine !== this._buf.getCurrentLine()) {
        indent = true;
      }
    }
    if (indent) this.indent();
    const newlineOpts = {
      addNewlines: opts.addNewlines,
      nextNodeStartLine: 0
    };
    const separator = opts.separator ? opts.separator.bind(this) : null;
    const len = nodes.length;
    for (let i = 0; i < len; i++) {
      const node = nodes[i];
      if (!node) continue;
      if (opts.statement) this._printNewline(i === 0, newlineOpts);
      this.print(node, undefined, opts.trailingCommentsLineOffset || 0);
      opts.iterator == null || opts.iterator(node, i);
      if (i < len - 1) separator == null || separator();
      if (opts.statement) {
        var _node$trailingComment;
        if (!((_node$trailingComment = node.trailingComments) != null && _node$trailingComment.length)) {
          this._lastCommentLine = 0;
        }
        if (i + 1 === len) {
          this.newline(1);
        } else {
          var _nextNode$loc;
          const nextNode = nodes[i + 1];
          newlineOpts.nextNodeStartLine = ((_nextNode$loc = nextNode.loc) == null ? void 0 : _nextNode$loc.start.line) || 0;
          this._printNewline(true, newlineOpts);
        }
      }
    }
    if (indent) this.dedent();
  }
  printAndIndentOnComments(node) {
    const indent = node.leadingComments && node.leadingComments.length > 0;
    if (indent) this.indent();
    this.print(node);
    if (indent) this.dedent();
  }
  printBlock(parent) {
    const node = parent.body;
    if (node.type !== "EmptyStatement") {
      this.space();
    }
    this.print(node);
  }
  _printTrailingComments(node, parent, lineOffset) {
    const {
      innerComments,
      trailingComments
    } = node;
    if (innerComments != null && innerComments.length) {
      this._printComments(2, innerComments, node, parent, lineOffset);
    }
    if (trailingComments != null && trailingComments.length) {
      this._printComments(2, trailingComments, node, parent, lineOffset);
    }
  }
  _printLeadingComments(node, parent) {
    const comments = node.leadingComments;
    if (!(comments != null && comments.length)) return;
    this._printComments(0, comments, node, parent);
  }
  _maybePrintInnerComments() {
    if (this._endsWithInnerRaw) this.printInnerComments();
    this._endsWithInnerRaw = true;
    this._indentInnerComments = true;
  }
  printInnerComments() {
    const node = this._currentNode;
    const comments = node.innerComments;
    if (!(comments != null && comments.length)) return;
    const hasSpace = this.endsWith(32);
    const indent = this._indentInnerComments;
    const printedCommentsCount = this._printedComments.size;
    if (indent) this.indent();
    this._printComments(1, comments, node);
    if (hasSpace && printedCommentsCount !== this._printedComments.size) {
      this.space();
    }
    if (indent) this.dedent();
  }
  noIndentInnerCommentsHere() {
    this._indentInnerComments = false;
  }
  printSequence(nodes, opts = {}) {
    var _opts$indent;
    opts.statement = true;
    (_opts$indent = opts.indent) != null ? _opts$indent : opts.indent = false;
    this.printJoin(nodes, opts);
  }
  printList(items, opts = {}) {
    if (opts.separator == null) {
      opts.separator = commaSeparator;
    }
    this.printJoin(items, opts);
  }
  _printNewline(newLine, opts) {
    const format = this.format;
    if (format.retainLines || format.compact) return;
    if (format.concise) {
      this.space();
      return;
    }
    if (!newLine) {
      return;
    }
    const startLine = opts.nextNodeStartLine;
    const lastCommentLine = this._lastCommentLine;
    if (startLine > 0 && lastCommentLine > 0) {
      const offset = startLine - lastCommentLine;
      if (offset >= 0) {
        this.newline(offset || 1);
        return;
      }
    }
    if (this._buf.hasContent()) {
      this.newline(1);
    }
  }
  _shouldPrintComment(comment) {
    if (comment.ignore) return 0;
    if (this._printedComments.has(comment)) return 0;
    if (this._noLineTerminator && HAS_NEWLINE_OR_BlOCK_COMMENT_END.test(comment.value)) {
      return 2;
    }
    this._printedComments.add(comment);
    if (!this.format.shouldPrintComment(comment.value)) {
      return 0;
    }
    return 1;
  }
  _printComment(comment, skipNewLines) {
    const noLineTerminator = this._noLineTerminator;
    const isBlockComment = comment.type === "CommentBlock";
    const printNewLines = isBlockComment && skipNewLines !== 1 && !this._noLineTerminator;
    if (printNewLines && this._buf.hasContent() && skipNewLines !== 2) {
      this.newline(1);
    }
    const lastCharCode = this.getLastChar();
    if (lastCharCode !== 91 && lastCharCode !== 123 && lastCharCode !== 40) {
      this.space();
    }
    let val;
    if (isBlockComment) {
      const {
        _parenPushNewlineState
      } = this;
      if ((_parenPushNewlineState == null ? void 0 : _parenPushNewlineState.printed) === false && HAS_NEWLINE.test(comment.value)) {
        this.tokenChar(40);
        this.indent();
        _parenPushNewlineState.printed = true;
      }
      val = `/*${comment.value}*/`;
      if (this.format.indent.adjustMultilineComment) {
        var _comment$loc;
        const offset = (_comment$loc = comment.loc) == null ? void 0 : _comment$loc.start.column;
        if (offset) {
          const newlineRegex = new RegExp("\\n\\s{1," + offset + "}", "g");
          val = val.replace(newlineRegex, "\n");
        }
        if (this.format.concise) {
          val = val.replace(/\n(?!$)/g, `\n`);
        } else {
          let indentSize = this.format.retainLines ? 0 : this._buf.getCurrentColumn();
          if (this._shouldIndent(47) || this.format.retainLines) {
            indentSize += this._getIndent();
          }
          val = val.replace(/\n(?!$)/g, `\n${" ".repeat(indentSize)}`);
        }
      }
    } else if (!noLineTerminator) {
      val = `//${comment.value}`;
    } else {
      val = `/*${comment.value}*/`;
    }
    if (this._endsWithDiv) this._space();
    this.source("start", comment.loc);
    this._append(val, isBlockComment);
    if (!isBlockComment && !noLineTerminator) {
      this.newline(1, true);
    }
    if (printNewLines && skipNewLines !== 3) {
      this.newline(1);
    }
  }
  _printComments(type, comments, node, parent, lineOffset = 0) {
    const nodeLoc = node.loc;
    const len = comments.length;
    let hasLoc = !!nodeLoc;
    const nodeStartLine = hasLoc ? nodeLoc.start.line : 0;
    const nodeEndLine = hasLoc ? nodeLoc.end.line : 0;
    let lastLine = 0;
    let leadingCommentNewline = 0;
    const maybeNewline = this._noLineTerminator ? function () {} : this.newline.bind(this);
    for (let i = 0; i < len; i++) {
      const comment = comments[i];
      const shouldPrint = this._shouldPrintComment(comment);
      if (shouldPrint === 2) {
        hasLoc = false;
        break;
      }
      if (hasLoc && comment.loc && shouldPrint === 1) {
        const commentStartLine = comment.loc.start.line;
        const commentEndLine = comment.loc.end.line;
        if (type === 0) {
          let offset = 0;
          if (i === 0) {
            if (this._buf.hasContent() && (comment.type === "CommentLine" || commentStartLine !== commentEndLine)) {
              offset = leadingCommentNewline = 1;
            }
          } else {
            offset = commentStartLine - lastLine;
          }
          lastLine = commentEndLine;
          maybeNewline(offset);
          this._printComment(comment, 1);
          if (i + 1 === len) {
            maybeNewline(Math.max(nodeStartLine - lastLine, leadingCommentNewline));
            lastLine = nodeStartLine;
          }
        } else if (type === 1) {
          const offset = commentStartLine - (i === 0 ? nodeStartLine : lastLine);
          lastLine = commentEndLine;
          maybeNewline(offset);
          this._printComment(comment, 1);
          if (i + 1 === len) {
            maybeNewline(Math.min(1, nodeEndLine - lastLine));
            lastLine = nodeEndLine;
          }
        } else {
          const offset = commentStartLine - (i === 0 ? nodeEndLine - lineOffset : lastLine);
          lastLine = commentEndLine;
          maybeNewline(offset);
          this._printComment(comment, 1);
        }
      } else {
        hasLoc = false;
        if (shouldPrint !== 1) {
          continue;
        }
        if (len === 1) {
          const singleLine = comment.loc ? comment.loc.start.line === comment.loc.end.line : !HAS_NEWLINE.test(comment.value);
          const shouldSkipNewline = singleLine && !isStatement(node) && !isClassBody(parent) && !isTSInterfaceBody(parent) && !isTSEnumDeclaration(parent);
          if (type === 0) {
            this._printComment(comment, shouldSkipNewline && node.type !== "ObjectExpression" || singleLine && isFunction(parent, {
              body: node
            }) ? 1 : 0);
          } else if (shouldSkipNewline && type === 2) {
            this._printComment(comment, 1);
          } else {
            this._printComment(comment, 0);
          }
        } else if (type === 1 && !(node.type === "ObjectExpression" && node.properties.length > 1) && node.type !== "ClassBody" && node.type !== "TSInterfaceBody") {
          this._printComment(comment, i === 0 ? 2 : i === len - 1 ? 3 : 0);
        } else {
          this._printComment(comment, 0);
        }
      }
    }
    if (type === 2 && hasLoc && lastLine) {
      this._lastCommentLine = lastLine;
    }
  }
}
Object.assign(Printer.prototype, generatorFunctions);
{
  Printer.prototype.Noop = function Noop() {};
}
var _default = exports.default = Printer;
function commaSeparator() {
  this.tokenChar(44);
  this.space();
}

//# sourceMappingURL=printer.js.map

}, function(modId) { var map = {"./buffer.js":1781530615287,"./node/index.js":1781530615288,"./generators/index.js":1781530615291}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615287, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = void 0;
class Buffer {
  constructor(map, indentChar) {
    this._map = null;
    this._buf = "";
    this._str = "";
    this._appendCount = 0;
    this._last = 0;
    this._queue = [];
    this._queueCursor = 0;
    this._canMarkIdName = true;
    this._indentChar = "";
    this._fastIndentations = [];
    this._position = {
      line: 1,
      column: 0
    };
    this._sourcePosition = {
      identifierName: undefined,
      identifierNamePos: undefined,
      line: undefined,
      column: undefined,
      filename: undefined
    };
    this._map = map;
    this._indentChar = indentChar;
    for (let i = 0; i < 64; i++) {
      this._fastIndentations.push(indentChar.repeat(i));
    }
    this._allocQueue();
  }
  _allocQueue() {
    const queue = this._queue;
    for (let i = 0; i < 16; i++) {
      queue.push({
        char: 0,
        repeat: 1,
        line: undefined,
        column: undefined,
        identifierName: undefined,
        identifierNamePos: undefined,
        filename: ""
      });
    }
  }
  _pushQueue(char, repeat, line, column, filename) {
    const cursor = this._queueCursor;
    if (cursor === this._queue.length) {
      this._allocQueue();
    }
    const item = this._queue[cursor];
    item.char = char;
    item.repeat = repeat;
    item.line = line;
    item.column = column;
    item.filename = filename;
    this._queueCursor++;
  }
  _popQueue() {
    if (this._queueCursor === 0) {
      throw new Error("Cannot pop from empty queue");
    }
    return this._queue[--this._queueCursor];
  }
  get() {
    this._flush();
    const map = this._map;
    const result = {
      code: (this._buf + this._str).trimRight(),
      decodedMap: map == null ? void 0 : map.getDecoded(),
      get __mergedMap() {
        return this.map;
      },
      get map() {
        const resultMap = map ? map.get() : null;
        result.map = resultMap;
        return resultMap;
      },
      set map(value) {
        Object.defineProperty(result, "map", {
          value,
          writable: true
        });
      },
      get rawMappings() {
        const mappings = map == null ? void 0 : map.getRawMappings();
        result.rawMappings = mappings;
        return mappings;
      },
      set rawMappings(value) {
        Object.defineProperty(result, "rawMappings", {
          value,
          writable: true
        });
      }
    };
    return result;
  }
  append(str, maybeNewline) {
    this._flush();
    this._append(str, this._sourcePosition, maybeNewline);
  }
  appendChar(char) {
    this._flush();
    this._appendChar(char, 1, this._sourcePosition);
  }
  queue(char) {
    if (char === 10) {
      while (this._queueCursor !== 0) {
        const char = this._queue[this._queueCursor - 1].char;
        if (char !== 32 && char !== 9) {
          break;
        }
        this._queueCursor--;
      }
    }
    const sourcePosition = this._sourcePosition;
    this._pushQueue(char, 1, sourcePosition.line, sourcePosition.column, sourcePosition.filename);
  }
  queueIndentation(repeat) {
    if (repeat === 0) return;
    this._pushQueue(-1, repeat, undefined, undefined, undefined);
  }
  _flush() {
    const queueCursor = this._queueCursor;
    const queue = this._queue;
    for (let i = 0; i < queueCursor; i++) {
      const item = queue[i];
      this._appendChar(item.char, item.repeat, item);
    }
    this._queueCursor = 0;
  }
  _appendChar(char, repeat, sourcePos) {
    this._last = char;
    if (char === -1) {
      const fastIndentation = this._fastIndentations[repeat];
      if (fastIndentation !== undefined) {
        this._str += fastIndentation;
      } else {
        this._str += repeat > 1 ? this._indentChar.repeat(repeat) : this._indentChar;
      }
    } else {
      this._str += repeat > 1 ? String.fromCharCode(char).repeat(repeat) : String.fromCharCode(char);
    }
    if (char !== 10) {
      this._mark(sourcePos.line, sourcePos.column, sourcePos.identifierName, sourcePos.identifierNamePos, sourcePos.filename);
      this._position.column += repeat;
    } else {
      this._position.line++;
      this._position.column = 0;
    }
    if (this._canMarkIdName) {
      sourcePos.identifierName = undefined;
      sourcePos.identifierNamePos = undefined;
    }
  }
  _append(str, sourcePos, maybeNewline) {
    const len = str.length;
    const position = this._position;
    this._last = str.charCodeAt(len - 1);
    if (++this._appendCount > 4096) {
      +this._str;
      this._buf += this._str;
      this._str = str;
      this._appendCount = 0;
    } else {
      this._str += str;
    }
    if (!maybeNewline && !this._map) {
      position.column += len;
      return;
    }
    const {
      column,
      identifierName,
      identifierNamePos,
      filename
    } = sourcePos;
    let line = sourcePos.line;
    if ((identifierName != null || identifierNamePos != null) && this._canMarkIdName) {
      sourcePos.identifierName = undefined;
      sourcePos.identifierNamePos = undefined;
    }
    let i = str.indexOf("\n");
    let last = 0;
    if (i !== 0) {
      this._mark(line, column, identifierName, identifierNamePos, filename);
    }
    while (i !== -1) {
      position.line++;
      position.column = 0;
      last = i + 1;
      if (last < len && line !== undefined) {
        this._mark(++line, 0, null, null, filename);
      }
      i = str.indexOf("\n", last);
    }
    position.column += len - last;
  }
  _mark(line, column, identifierName, identifierNamePos, filename) {
    var _this$_map;
    (_this$_map = this._map) == null || _this$_map.mark(this._position, line, column, identifierName, identifierNamePos, filename);
  }
  removeTrailingNewline() {
    const queueCursor = this._queueCursor;
    if (queueCursor !== 0 && this._queue[queueCursor - 1].char === 10) {
      this._queueCursor--;
    }
  }
  removeLastSemicolon() {
    const queueCursor = this._queueCursor;
    if (queueCursor !== 0 && this._queue[queueCursor - 1].char === 59) {
      this._queueCursor--;
    }
  }
  getLastChar() {
    const queueCursor = this._queueCursor;
    return queueCursor !== 0 ? this._queue[queueCursor - 1].char : this._last;
  }
  getNewlineCount() {
    const queueCursor = this._queueCursor;
    let count = 0;
    if (queueCursor === 0) return this._last === 10 ? 1 : 0;
    for (let i = queueCursor - 1; i >= 0; i--) {
      if (this._queue[i].char !== 10) {
        break;
      }
      count++;
    }
    return count === queueCursor && this._last === 10 ? count + 1 : count;
  }
  endsWithCharAndNewline() {
    const queue = this._queue;
    const queueCursor = this._queueCursor;
    if (queueCursor !== 0) {
      const lastCp = queue[queueCursor - 1].char;
      if (lastCp !== 10) return;
      if (queueCursor > 1) {
        return queue[queueCursor - 2].char;
      } else {
        return this._last;
      }
    }
  }
  hasContent() {
    return this._queueCursor !== 0 || !!this._last;
  }
  exactSource(loc, cb) {
    if (!this._map) {
      cb();
      return;
    }
    this.source("start", loc);
    const identifierName = loc.identifierName;
    const sourcePos = this._sourcePosition;
    if (identifierName) {
      this._canMarkIdName = false;
      sourcePos.identifierName = identifierName;
    }
    cb();
    if (identifierName) {
      this._canMarkIdName = true;
      sourcePos.identifierName = undefined;
      sourcePos.identifierNamePos = undefined;
    }
    this.source("end", loc);
  }
  source(prop, loc) {
    if (!this._map) return;
    this._normalizePosition(prop, loc, 0);
  }
  sourceWithOffset(prop, loc, columnOffset) {
    if (!this._map) return;
    this._normalizePosition(prop, loc, columnOffset);
  }
  _normalizePosition(prop, loc, columnOffset) {
    const pos = loc[prop];
    const target = this._sourcePosition;
    if (pos) {
      target.line = pos.line;
      target.column = Math.max(pos.column + columnOffset, 0);
      target.filename = loc.filename;
    }
  }
  getCurrentColumn() {
    const queue = this._queue;
    const queueCursor = this._queueCursor;
    let lastIndex = -1;
    let len = 0;
    for (let i = 0; i < queueCursor; i++) {
      const item = queue[i];
      if (item.char === 10) {
        lastIndex = len;
      }
      len += item.repeat;
    }
    return lastIndex === -1 ? this._position.column + len : len - 1 - lastIndex;
  }
  getCurrentLine() {
    let count = 0;
    const queue = this._queue;
    for (let i = 0; i < this._queueCursor; i++) {
      if (queue[i].char === 10) {
        count++;
      }
    }
    return this._position.line + count;
  }
}
exports.default = Buffer;

//# sourceMappingURL=buffer.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615288, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.TokenContext = void 0;
exports.needsParens = needsParens;
exports.needsWhitespace = needsWhitespace;
exports.needsWhitespaceAfter = needsWhitespaceAfter;
exports.needsWhitespaceBefore = needsWhitespaceBefore;
var whitespace = require("./whitespace.js");
var parens = require("./parentheses.js");
var _t = require("@babel/types");
const {
  FLIPPED_ALIAS_KEYS,
  isCallExpression,
  isDecorator,
  isExpressionStatement,
  isMemberExpression,
  isNewExpression,
  isParenthesizedExpression
} = _t;
const TokenContext = exports.TokenContext = {
  expressionStatement: 1,
  arrowBody: 2,
  exportDefault: 4,
  forHead: 8,
  forInHead: 16,
  forOfHead: 32,
  arrowFlowReturnType: 64
};
function expandAliases(obj) {
  const map = new Map();
  function add(type, func) {
    const fn = map.get(type);
    map.set(type, fn ? function (node, parent, stack, inForInit) {
      var _fn;
      return (_fn = fn(node, parent, stack, inForInit)) != null ? _fn : func(node, parent, stack, inForInit);
    } : func);
  }
  for (const type of Object.keys(obj)) {
    const aliases = FLIPPED_ALIAS_KEYS[type];
    if (aliases) {
      for (const alias of aliases) {
        add(alias, obj[type]);
      }
    } else {
      add(type, obj[type]);
    }
  }
  return map;
}
const expandedParens = expandAliases(parens);
const expandedWhitespaceNodes = expandAliases(whitespace.nodes);
function isOrHasCallExpression(node) {
  if (isCallExpression(node)) {
    return true;
  }
  return isMemberExpression(node) && isOrHasCallExpression(node.object);
}
function needsWhitespace(node, parent, type) {
  var _expandedWhitespaceNo;
  if (!node) return false;
  if (isExpressionStatement(node)) {
    node = node.expression;
  }
  const flag = (_expandedWhitespaceNo = expandedWhitespaceNodes.get(node.type)) == null ? void 0 : _expandedWhitespaceNo(node, parent);
  if (typeof flag === "number") {
    return (flag & type) !== 0;
  }
  return false;
}
function needsWhitespaceBefore(node, parent) {
  return needsWhitespace(node, parent, 1);
}
function needsWhitespaceAfter(node, parent) {
  return needsWhitespace(node, parent, 2);
}
function needsParens(node, parent, tokenContext, inForInit) {
  var _expandedParens$get;
  if (!parent) return false;
  if (isNewExpression(parent) && parent.callee === node) {
    if (isOrHasCallExpression(node)) return true;
  }
  if (isDecorator(parent)) {
    return !isDecoratorMemberExpression(node) && !(isCallExpression(node) && isDecoratorMemberExpression(node.callee)) && !isParenthesizedExpression(node);
  }
  return (_expandedParens$get = expandedParens.get(node.type)) == null ? void 0 : _expandedParens$get(node, parent, tokenContext, inForInit);
}
function isDecoratorMemberExpression(node) {
  switch (node.type) {
    case "Identifier":
      return true;
    case "MemberExpression":
      return !node.computed && node.property.type === "Identifier" && isDecoratorMemberExpression(node.object);
    default:
      return false;
  }
}

//# sourceMappingURL=index.js.map

}, function(modId) { var map = {"./whitespace.js":1781530615289,"./parentheses.js":1781530615290}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615289, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.nodes = void 0;
var _t = require("@babel/types");
const {
  FLIPPED_ALIAS_KEYS,
  isArrayExpression,
  isAssignmentExpression,
  isBinary,
  isBlockStatement,
  isCallExpression,
  isFunction,
  isIdentifier,
  isLiteral,
  isMemberExpression,
  isObjectExpression,
  isOptionalCallExpression,
  isOptionalMemberExpression,
  isStringLiteral
} = _t;
function crawlInternal(node, state) {
  if (!node) return state;
  if (isMemberExpression(node) || isOptionalMemberExpression(node)) {
    crawlInternal(node.object, state);
    if (node.computed) crawlInternal(node.property, state);
  } else if (isBinary(node) || isAssignmentExpression(node)) {
    crawlInternal(node.left, state);
    crawlInternal(node.right, state);
  } else if (isCallExpression(node) || isOptionalCallExpression(node)) {
    state.hasCall = true;
    crawlInternal(node.callee, state);
  } else if (isFunction(node)) {
    state.hasFunction = true;
  } else if (isIdentifier(node)) {
    state.hasHelper = state.hasHelper || node.callee && isHelper(node.callee);
  }
  return state;
}
function crawl(node) {
  return crawlInternal(node, {
    hasCall: false,
    hasFunction: false,
    hasHelper: false
  });
}
function isHelper(node) {
  if (!node) return false;
  if (isMemberExpression(node)) {
    return isHelper(node.object) || isHelper(node.property);
  } else if (isIdentifier(node)) {
    return node.name === "require" || node.name.charCodeAt(0) === 95;
  } else if (isCallExpression(node)) {
    return isHelper(node.callee);
  } else if (isBinary(node) || isAssignmentExpression(node)) {
    return isIdentifier(node.left) && isHelper(node.left) || isHelper(node.right);
  } else {
    return false;
  }
}
function isType(node) {
  return isLiteral(node) || isObjectExpression(node) || isArrayExpression(node) || isIdentifier(node) || isMemberExpression(node);
}
const nodes = exports.nodes = {
  AssignmentExpression(node) {
    const state = crawl(node.right);
    if (state.hasCall && state.hasHelper || state.hasFunction) {
      return state.hasFunction ? 1 | 2 : 2;
    }
  },
  SwitchCase(node, parent) {
    return (!!node.consequent.length || parent.cases[0] === node ? 1 : 0) | (!node.consequent.length && parent.cases[parent.cases.length - 1] === node ? 2 : 0);
  },
  LogicalExpression(node) {
    if (isFunction(node.left) || isFunction(node.right)) {
      return 2;
    }
  },
  Literal(node) {
    if (isStringLiteral(node) && node.value === "use strict") {
      return 2;
    }
  },
  CallExpression(node) {
    if (isFunction(node.callee) || isHelper(node)) {
      return 1 | 2;
    }
  },
  OptionalCallExpression(node) {
    if (isFunction(node.callee)) {
      return 1 | 2;
    }
  },
  VariableDeclaration(node) {
    for (let i = 0; i < node.declarations.length; i++) {
      const declar = node.declarations[i];
      let enabled = isHelper(declar.id) && !isType(declar.init);
      if (!enabled && declar.init) {
        const state = crawl(declar.init);
        enabled = isHelper(declar.init) && state.hasCall || state.hasFunction;
      }
      if (enabled) {
        return 1 | 2;
      }
    }
  },
  IfStatement(node) {
    if (isBlockStatement(node.consequent)) {
      return 1 | 2;
    }
  }
};
nodes.ObjectProperty = nodes.ObjectTypeProperty = nodes.ObjectMethod = function (node, parent) {
  if (parent.properties[0] === node) {
    return 1;
  }
};
nodes.ObjectTypeCallProperty = function (node, parent) {
  var _parent$properties;
  if (parent.callProperties[0] === node && !((_parent$properties = parent.properties) != null && _parent$properties.length)) {
    return 1;
  }
};
nodes.ObjectTypeIndexer = function (node, parent) {
  var _parent$properties2, _parent$callPropertie;
  if (parent.indexers[0] === node && !((_parent$properties2 = parent.properties) != null && _parent$properties2.length) && !((_parent$callPropertie = parent.callProperties) != null && _parent$callPropertie.length)) {
    return 1;
  }
};
nodes.ObjectTypeInternalSlot = function (node, parent) {
  var _parent$properties3, _parent$callPropertie2, _parent$indexers;
  if (parent.internalSlots[0] === node && !((_parent$properties3 = parent.properties) != null && _parent$properties3.length) && !((_parent$callPropertie2 = parent.callProperties) != null && _parent$callPropertie2.length) && !((_parent$indexers = parent.indexers) != null && _parent$indexers.length)) {
    return 1;
  }
};
[["Function", true], ["Class", true], ["Loop", true], ["LabeledStatement", true], ["SwitchStatement", true], ["TryStatement", true]].forEach(function ([type, amounts]) {
  [type].concat(FLIPPED_ALIAS_KEYS[type] || []).forEach(function (type) {
    const ret = amounts ? 1 | 2 : 0;
    nodes[type] = () => ret;
  });
});

//# sourceMappingURL=whitespace.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615290, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.AssignmentExpression = AssignmentExpression;
exports.Binary = Binary;
exports.BinaryExpression = BinaryExpression;
exports.ClassExpression = ClassExpression;
exports.ArrowFunctionExpression = exports.ConditionalExpression = ConditionalExpression;
exports.DoExpression = DoExpression;
exports.FunctionExpression = FunctionExpression;
exports.FunctionTypeAnnotation = FunctionTypeAnnotation;
exports.Identifier = Identifier;
exports.LogicalExpression = LogicalExpression;
exports.NullableTypeAnnotation = NullableTypeAnnotation;
exports.ObjectExpression = ObjectExpression;
exports.OptionalIndexedAccessType = OptionalIndexedAccessType;
exports.OptionalCallExpression = exports.OptionalMemberExpression = OptionalMemberExpression;
exports.SequenceExpression = SequenceExpression;
exports.TSSatisfiesExpression = exports.TSAsExpression = TSAsExpression;
exports.TSInferType = TSInferType;
exports.TSInstantiationExpression = TSInstantiationExpression;
exports.UnaryLike = exports.TSTypeAssertion = UnaryLike;
exports.TSIntersectionType = exports.TSUnionType = TSUnionType;
exports.IntersectionTypeAnnotation = exports.UnionTypeAnnotation = UnionTypeAnnotation;
exports.UpdateExpression = UpdateExpression;
exports.AwaitExpression = exports.YieldExpression = YieldExpression;
var _t = require("@babel/types");
var _index = require("./index.js");
const {
  isArrayTypeAnnotation,
  isBinaryExpression,
  isCallExpression,
  isForOfStatement,
  isIndexedAccessType,
  isMemberExpression,
  isObjectPattern,
  isOptionalMemberExpression,
  isYieldExpression,
  isStatement
} = _t;
const PRECEDENCE = new Map([["||", 0], ["??", 0], ["|>", 0], ["&&", 1], ["|", 2], ["^", 3], ["&", 4], ["==", 5], ["===", 5], ["!=", 5], ["!==", 5], ["<", 6], [">", 6], ["<=", 6], [">=", 6], ["in", 6], ["instanceof", 6], [">>", 7], ["<<", 7], [">>>", 7], ["+", 8], ["-", 8], ["*", 9], ["/", 9], ["%", 9], ["**", 10]]);
function getBinaryPrecedence(node, nodeType) {
  if (nodeType === "BinaryExpression" || nodeType === "LogicalExpression") {
    return PRECEDENCE.get(node.operator);
  }
  if (nodeType === "TSAsExpression" || nodeType === "TSSatisfiesExpression") {
    return PRECEDENCE.get("in");
  }
}
function isTSTypeExpression(nodeType) {
  return nodeType === "TSAsExpression" || nodeType === "TSSatisfiesExpression" || nodeType === "TSTypeAssertion";
}
const isClassExtendsClause = (node, parent) => {
  const parentType = parent.type;
  return (parentType === "ClassDeclaration" || parentType === "ClassExpression") && parent.superClass === node;
};
const hasPostfixPart = (node, parent) => {
  const parentType = parent.type;
  return (parentType === "MemberExpression" || parentType === "OptionalMemberExpression") && parent.object === node || (parentType === "CallExpression" || parentType === "OptionalCallExpression" || parentType === "NewExpression") && parent.callee === node || parentType === "TaggedTemplateExpression" && parent.tag === node || parentType === "TSNonNullExpression";
};
function NullableTypeAnnotation(node, parent) {
  return isArrayTypeAnnotation(parent);
}
function FunctionTypeAnnotation(node, parent, tokenContext) {
  const parentType = parent.type;
  return parentType === "UnionTypeAnnotation" || parentType === "IntersectionTypeAnnotation" || parentType === "ArrayTypeAnnotation" || Boolean(tokenContext & _index.TokenContext.arrowFlowReturnType);
}
function UpdateExpression(node, parent) {
  return hasPostfixPart(node, parent) || isClassExtendsClause(node, parent);
}
function needsParenBeforeExpressionBrace(tokenContext) {
  return Boolean(tokenContext & (_index.TokenContext.expressionStatement | _index.TokenContext.arrowBody));
}
function ObjectExpression(node, parent, tokenContext) {
  return needsParenBeforeExpressionBrace(tokenContext);
}
function DoExpression(node, parent, tokenContext) {
  return !node.async && Boolean(tokenContext & _index.TokenContext.expressionStatement);
}
function Binary(node, parent) {
  const parentType = parent.type;
  if (node.type === "BinaryExpression" && node.operator === "**" && parentType === "BinaryExpression" && parent.operator === "**") {
    return parent.left === node;
  }
  if (isClassExtendsClause(node, parent)) {
    return true;
  }
  if (hasPostfixPart(node, parent) || parentType === "UnaryExpression" || parentType === "SpreadElement" || parentType === "AwaitExpression") {
    return true;
  }
  const parentPos = getBinaryPrecedence(parent, parentType);
  if (parentPos != null) {
    const nodePos = getBinaryPrecedence(node, node.type);
    if (parentPos === nodePos && parentType === "BinaryExpression" && parent.right === node || parentPos > nodePos) {
      return true;
    }
  }
  return undefined;
}
function UnionTypeAnnotation(node, parent) {
  const parentType = parent.type;
  return parentType === "ArrayTypeAnnotation" || parentType === "NullableTypeAnnotation" || parentType === "IntersectionTypeAnnotation" || parentType === "UnionTypeAnnotation";
}
function OptionalIndexedAccessType(node, parent) {
  return isIndexedAccessType(parent) && parent.objectType === node;
}
function TSAsExpression(node, parent) {
  if ((parent.type === "AssignmentExpression" || parent.type === "AssignmentPattern") && parent.left === node) {
    return true;
  }
  if (parent.type === "BinaryExpression" && (parent.operator === "|" || parent.operator === "&") && node === parent.left) {
    return true;
  }
  return Binary(node, parent);
}
function TSUnionType(node, parent) {
  const parentType = parent.type;
  return parentType === "TSArrayType" || parentType === "TSOptionalType" || parentType === "TSIntersectionType" || parentType === "TSRestType";
}
function TSInferType(node, parent) {
  const parentType = parent.type;
  return parentType === "TSArrayType" || parentType === "TSOptionalType";
}
function TSInstantiationExpression(node, parent) {
  const parentType = parent.type;
  return (parentType === "CallExpression" || parentType === "OptionalCallExpression" || parentType === "NewExpression" || parentType === "TSInstantiationExpression") && !!parent.typeParameters;
}
function BinaryExpression(node, parent, tokenContext, inForStatementInit) {
  return node.operator === "in" && inForStatementInit;
}
function SequenceExpression(node, parent) {
  const parentType = parent.type;
  if (parentType === "SequenceExpression" || parentType === "ParenthesizedExpression" || parentType === "MemberExpression" && parent.property === node || parentType === "OptionalMemberExpression" && parent.property === node || parentType === "TemplateLiteral") {
    return false;
  }
  if (parentType === "ClassDeclaration") {
    return true;
  }
  if (parentType === "ForOfStatement") {
    return parent.right === node;
  }
  if (parentType === "ExportDefaultDeclaration") {
    return true;
  }
  return !isStatement(parent);
}
function YieldExpression(node, parent) {
  const parentType = parent.type;
  return parentType === "BinaryExpression" || parentType === "LogicalExpression" || parentType === "UnaryExpression" || parentType === "SpreadElement" || hasPostfixPart(node, parent) || parentType === "AwaitExpression" && isYieldExpression(node) || parentType === "ConditionalExpression" && node === parent.test || isClassExtendsClause(node, parent) || isTSTypeExpression(parentType);
}
function ClassExpression(node, parent, tokenContext) {
  return Boolean(tokenContext & (_index.TokenContext.expressionStatement | _index.TokenContext.exportDefault));
}
function UnaryLike(node, parent) {
  return hasPostfixPart(node, parent) || isBinaryExpression(parent) && parent.operator === "**" && parent.left === node || isClassExtendsClause(node, parent);
}
function FunctionExpression(node, parent, tokenContext) {
  return Boolean(tokenContext & (_index.TokenContext.expressionStatement | _index.TokenContext.exportDefault));
}
function ConditionalExpression(node, parent) {
  const parentType = parent.type;
  if (parentType === "UnaryExpression" || parentType === "SpreadElement" || parentType === "BinaryExpression" || parentType === "LogicalExpression" || parentType === "ConditionalExpression" && parent.test === node || parentType === "AwaitExpression" || isTSTypeExpression(parentType)) {
    return true;
  }
  return UnaryLike(node, parent);
}
function OptionalMemberExpression(node, parent) {
  return isCallExpression(parent) && parent.callee === node || isMemberExpression(parent) && parent.object === node;
}
function AssignmentExpression(node, parent, tokenContext) {
  if (needsParenBeforeExpressionBrace(tokenContext) && isObjectPattern(node.left)) {
    return true;
  } else {
    return ConditionalExpression(node, parent);
  }
}
function LogicalExpression(node, parent) {
  const parentType = parent.type;
  if (isTSTypeExpression(parentType)) return true;
  if (parentType !== "LogicalExpression") return false;
  switch (node.operator) {
    case "||":
      return parent.operator === "??" || parent.operator === "&&";
    case "&&":
      return parent.operator === "??";
    case "??":
      return parent.operator !== "??";
  }
}
function Identifier(node, parent, tokenContext) {
  var _node$extra;
  const parentType = parent.type;
  if ((_node$extra = node.extra) != null && _node$extra.parenthesized && parentType === "AssignmentExpression" && parent.left === node) {
    const rightType = parent.right.type;
    if ((rightType === "FunctionExpression" || rightType === "ClassExpression") && parent.right.id == null) {
      return true;
    }
  }
  if (node.name === "let") {
    const isFollowedByBracket = isMemberExpression(parent, {
      object: node,
      computed: true
    }) || isOptionalMemberExpression(parent, {
      object: node,
      computed: true,
      optional: false
    });
    if (isFollowedByBracket && tokenContext & (_index.TokenContext.expressionStatement | _index.TokenContext.forHead | _index.TokenContext.forInHead)) {
      return true;
    }
    return Boolean(tokenContext & _index.TokenContext.forOfHead);
  }
  return node.name === "async" && isForOfStatement(parent, {
    left: node,
    await: false
  });
}

//# sourceMappingURL=parentheses.js.map

}, function(modId) { var map = {"./index.js":1781530615288}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615291, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
var _templateLiterals = require("./template-literals.js");
Object.keys(_templateLiterals).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _templateLiterals[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _templateLiterals[key];
    }
  });
});
var _expressions = require("./expressions.js");
Object.keys(_expressions).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _expressions[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _expressions[key];
    }
  });
});
var _statements = require("./statements.js");
Object.keys(_statements).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _statements[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _statements[key];
    }
  });
});
var _classes = require("./classes.js");
Object.keys(_classes).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _classes[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _classes[key];
    }
  });
});
var _methods = require("./methods.js");
Object.keys(_methods).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _methods[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _methods[key];
    }
  });
});
var _modules = require("./modules.js");
Object.keys(_modules).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _modules[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _modules[key];
    }
  });
});
var _types = require("./types.js");
Object.keys(_types).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _types[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _types[key];
    }
  });
});
var _flow = require("./flow.js");
Object.keys(_flow).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _flow[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _flow[key];
    }
  });
});
var _base = require("./base.js");
Object.keys(_base).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _base[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _base[key];
    }
  });
});
var _jsx = require("./jsx.js");
Object.keys(_jsx).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _jsx[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _jsx[key];
    }
  });
});
var _typescript = require("./typescript.js");
Object.keys(_typescript).forEach(function (key) {
  if (key === "default" || key === "__esModule") return;
  if (key in exports && exports[key] === _typescript[key]) return;
  Object.defineProperty(exports, key, {
    enumerable: true,
    get: function () {
      return _typescript[key];
    }
  });
});

//# sourceMappingURL=index.js.map

}, function(modId) { var map = {"./template-literals.js":1781530615292,"./expressions.js":1781530615293,"./statements.js":1781530615294,"./classes.js":1781530615295,"./methods.js":1781530615296,"./modules.js":1781530615297,"./types.js":1781530615298,"./flow.js":1781530615299,"./base.js":1781530615300,"./jsx.js":1781530615301,"./typescript.js":1781530615302}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615292, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.TaggedTemplateExpression = TaggedTemplateExpression;
exports.TemplateElement = TemplateElement;
exports.TemplateLiteral = TemplateLiteral;
function TaggedTemplateExpression(node) {
  this.print(node.tag);
  this.print(node.typeParameters);
  this.print(node.quasi);
}
function TemplateElement() {
  throw new Error("TemplateElement printing is handled in TemplateLiteral");
}
function TemplateLiteral(node) {
  const quasis = node.quasis;
  let partRaw = "`";
  for (let i = 0; i < quasis.length; i++) {
    partRaw += quasis[i].value.raw;
    if (i + 1 < quasis.length) {
      this.token(partRaw + "${", true);
      this.print(node.expressions[i]);
      partRaw = "}";
    }
  }
  this.token(partRaw + "`", true);
}

//# sourceMappingURL=template-literals.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615293, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.LogicalExpression = exports.BinaryExpression = exports.AssignmentExpression = AssignmentExpression;
exports.AssignmentPattern = AssignmentPattern;
exports.AwaitExpression = AwaitExpression;
exports.BindExpression = BindExpression;
exports.CallExpression = CallExpression;
exports.ConditionalExpression = ConditionalExpression;
exports.Decorator = Decorator;
exports.DoExpression = DoExpression;
exports.EmptyStatement = EmptyStatement;
exports.ExpressionStatement = ExpressionStatement;
exports.Import = Import;
exports.MemberExpression = MemberExpression;
exports.MetaProperty = MetaProperty;
exports.ModuleExpression = ModuleExpression;
exports.NewExpression = NewExpression;
exports.OptionalCallExpression = OptionalCallExpression;
exports.OptionalMemberExpression = OptionalMemberExpression;
exports.ParenthesizedExpression = ParenthesizedExpression;
exports.PrivateName = PrivateName;
exports.SequenceExpression = SequenceExpression;
exports.Super = Super;
exports.ThisExpression = ThisExpression;
exports.UnaryExpression = UnaryExpression;
exports.UpdateExpression = UpdateExpression;
exports.V8IntrinsicIdentifier = V8IntrinsicIdentifier;
exports.YieldExpression = YieldExpression;
exports._shouldPrintDecoratorsBeforeExport = _shouldPrintDecoratorsBeforeExport;
var _t = require("@babel/types");
var _index = require("../node/index.js");
const {
  isCallExpression,
  isLiteral,
  isMemberExpression,
  isNewExpression
} = _t;
function UnaryExpression(node) {
  const {
    operator
  } = node;
  if (operator === "void" || operator === "delete" || operator === "typeof" || operator === "throw") {
    this.word(operator);
    this.space();
  } else {
    this.token(operator);
  }
  this.print(node.argument);
}
function DoExpression(node) {
  if (node.async) {
    this.word("async", true);
    this.space();
  }
  this.word("do");
  this.space();
  this.print(node.body);
}
function ParenthesizedExpression(node) {
  this.tokenChar(40);
  this.print(node.expression);
  this.rightParens(node);
}
function UpdateExpression(node) {
  if (node.prefix) {
    this.token(node.operator);
    this.print(node.argument);
  } else {
    this.printTerminatorless(node.argument, true);
    this.token(node.operator);
  }
}
function ConditionalExpression(node) {
  this.print(node.test);
  this.space();
  this.tokenChar(63);
  this.space();
  this.print(node.consequent);
  this.space();
  this.tokenChar(58);
  this.space();
  this.print(node.alternate);
}
function NewExpression(node, parent) {
  this.word("new");
  this.space();
  this.print(node.callee);
  if (this.format.minified && node.arguments.length === 0 && !node.optional && !isCallExpression(parent, {
    callee: node
  }) && !isMemberExpression(parent) && !isNewExpression(parent)) {
    return;
  }
  this.print(node.typeArguments);
  this.print(node.typeParameters);
  if (node.optional) {
    this.token("?.");
  }
  this.tokenChar(40);
  const exit = this.enterForStatementInit(false);
  this.printList(node.arguments);
  exit();
  this.rightParens(node);
}
function SequenceExpression(node) {
  this.printList(node.expressions);
}
function ThisExpression() {
  this.word("this");
}
function Super() {
  this.word("super");
}
function _shouldPrintDecoratorsBeforeExport(node) {
  if (typeof this.format.decoratorsBeforeExport === "boolean") {
    return this.format.decoratorsBeforeExport;
  }
  return typeof node.start === "number" && node.start === node.declaration.start;
}
function Decorator(node) {
  this.tokenChar(64);
  this.print(node.expression);
  this.newline();
}
function OptionalMemberExpression(node) {
  let {
    computed
  } = node;
  const {
    optional,
    property
  } = node;
  this.print(node.object);
  if (!computed && isMemberExpression(property)) {
    throw new TypeError("Got a MemberExpression for MemberExpression property");
  }
  if (isLiteral(property) && typeof property.value === "number") {
    computed = true;
  }
  if (optional) {
    this.token("?.");
  }
  if (computed) {
    this.tokenChar(91);
    this.print(property);
    this.tokenChar(93);
  } else {
    if (!optional) {
      this.tokenChar(46);
    }
    this.print(property);
  }
}
function OptionalCallExpression(node) {
  this.print(node.callee);
  this.print(node.typeParameters);
  if (node.optional) {
    this.token("?.");
  }
  this.print(node.typeArguments);
  this.tokenChar(40);
  const exit = this.enterForStatementInit(false);
  this.printList(node.arguments);
  exit();
  this.rightParens(node);
}
function CallExpression(node) {
  this.print(node.callee);
  this.print(node.typeArguments);
  this.print(node.typeParameters);
  this.tokenChar(40);
  const exit = this.enterForStatementInit(false);
  this.printList(node.arguments);
  exit();
  this.rightParens(node);
}
function Import() {
  this.word("import");
}
function AwaitExpression(node) {
  this.word("await");
  if (node.argument) {
    this.space();
    this.printTerminatorless(node.argument, false);
  }
}
function YieldExpression(node) {
  this.word("yield", true);
  if (node.delegate) {
    this.tokenChar(42);
    if (node.argument) {
      this.space();
      this.print(node.argument);
    }
  } else {
    if (node.argument) {
      this.space();
      this.printTerminatorless(node.argument, false);
    }
  }
}
function EmptyStatement() {
  this.semicolon(true);
}
function ExpressionStatement(node) {
  this.tokenContext |= _index.TokenContext.expressionStatement;
  this.print(node.expression);
  this.semicolon();
}
function AssignmentPattern(node) {
  this.print(node.left);
  if (node.left.type === "Identifier") {
    if (node.left.optional) this.tokenChar(63);
    this.print(node.left.typeAnnotation);
  }
  this.space();
  this.tokenChar(61);
  this.space();
  this.print(node.right);
}
function AssignmentExpression(node) {
  this.print(node.left);
  this.space();
  if (node.operator === "in" || node.operator === "instanceof") {
    this.word(node.operator);
  } else {
    this.token(node.operator);
    this._endsWithDiv = node.operator === "/";
  }
  this.space();
  this.print(node.right);
}
function BindExpression(node) {
  this.print(node.object);
  this.token("::");
  this.print(node.callee);
}
function MemberExpression(node) {
  this.print(node.object);
  if (!node.computed && isMemberExpression(node.property)) {
    throw new TypeError("Got a MemberExpression for MemberExpression property");
  }
  let computed = node.computed;
  if (isLiteral(node.property) && typeof node.property.value === "number") {
    computed = true;
  }
  if (computed) {
    const exit = this.enterForStatementInit(false);
    this.tokenChar(91);
    this.print(node.property);
    this.tokenChar(93);
    exit();
  } else {
    this.tokenChar(46);
    this.print(node.property);
  }
}
function MetaProperty(node) {
  this.print(node.meta);
  this.tokenChar(46);
  this.print(node.property);
}
function PrivateName(node) {
  this.tokenChar(35);
  this.print(node.id);
}
function V8IntrinsicIdentifier(node) {
  this.tokenChar(37);
  this.word(node.name);
}
function ModuleExpression(node) {
  this.word("module", true);
  this.space();
  this.tokenChar(123);
  this.indent();
  const {
    body
  } = node;
  if (body.body.length || body.directives.length) {
    this.newline();
  }
  this.print(body);
  this.dedent();
  this.rightBrace(node);
}

//# sourceMappingURL=expressions.js.map

}, function(modId) { var map = {"../node/index.js":1781530615288}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615294, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.BreakStatement = BreakStatement;
exports.CatchClause = CatchClause;
exports.ContinueStatement = ContinueStatement;
exports.DebuggerStatement = DebuggerStatement;
exports.DoWhileStatement = DoWhileStatement;
exports.ForOfStatement = exports.ForInStatement = void 0;
exports.ForStatement = ForStatement;
exports.IfStatement = IfStatement;
exports.LabeledStatement = LabeledStatement;
exports.ReturnStatement = ReturnStatement;
exports.SwitchCase = SwitchCase;
exports.SwitchStatement = SwitchStatement;
exports.ThrowStatement = ThrowStatement;
exports.TryStatement = TryStatement;
exports.VariableDeclaration = VariableDeclaration;
exports.VariableDeclarator = VariableDeclarator;
exports.WhileStatement = WhileStatement;
exports.WithStatement = WithStatement;
var _t = require("@babel/types");
var _index = require("../node/index.js");
const {
  isFor,
  isForStatement,
  isIfStatement,
  isStatement
} = _t;
function WithStatement(node) {
  this.word("with");
  this.space();
  this.tokenChar(40);
  this.print(node.object);
  this.tokenChar(41);
  this.printBlock(node);
}
function IfStatement(node) {
  this.word("if");
  this.space();
  this.tokenChar(40);
  this.print(node.test);
  this.tokenChar(41);
  this.space();
  const needsBlock = node.alternate && isIfStatement(getLastStatement(node.consequent));
  if (needsBlock) {
    this.tokenChar(123);
    this.newline();
    this.indent();
  }
  this.printAndIndentOnComments(node.consequent);
  if (needsBlock) {
    this.dedent();
    this.newline();
    this.tokenChar(125);
  }
  if (node.alternate) {
    if (this.endsWith(125)) this.space();
    this.word("else");
    this.space();
    this.printAndIndentOnComments(node.alternate);
  }
}
function getLastStatement(statement) {
  const {
    body
  } = statement;
  if (isStatement(body) === false) {
    return statement;
  }
  return getLastStatement(body);
}
function ForStatement(node) {
  this.word("for");
  this.space();
  this.tokenChar(40);
  {
    const exit = this.enterForStatementInit(true);
    this.tokenContext |= _index.TokenContext.forHead;
    this.print(node.init);
    exit();
  }
  this.tokenChar(59);
  if (node.test) {
    this.space();
    this.print(node.test);
  }
  this.tokenChar(59);
  if (node.update) {
    this.space();
    this.print(node.update);
  }
  this.tokenChar(41);
  this.printBlock(node);
}
function WhileStatement(node) {
  this.word("while");
  this.space();
  this.tokenChar(40);
  this.print(node.test);
  this.tokenChar(41);
  this.printBlock(node);
}
function ForXStatement(node) {
  this.word("for");
  this.space();
  const isForOf = node.type === "ForOfStatement";
  if (isForOf && node.await) {
    this.word("await");
    this.space();
  }
  this.noIndentInnerCommentsHere();
  this.tokenChar(40);
  {
    const exit = isForOf ? null : this.enterForStatementInit(true);
    this.tokenContext |= isForOf ? _index.TokenContext.forOfHead : _index.TokenContext.forInHead;
    this.print(node.left);
    exit == null || exit();
  }
  this.space();
  this.word(isForOf ? "of" : "in");
  this.space();
  this.print(node.right);
  this.tokenChar(41);
  this.printBlock(node);
}
const ForInStatement = exports.ForInStatement = ForXStatement;
const ForOfStatement = exports.ForOfStatement = ForXStatement;
function DoWhileStatement(node) {
  this.word("do");
  this.space();
  this.print(node.body);
  this.space();
  this.word("while");
  this.space();
  this.tokenChar(40);
  this.print(node.test);
  this.tokenChar(41);
  this.semicolon();
}
function printStatementAfterKeyword(printer, node, isLabel) {
  if (node) {
    printer.space();
    printer.printTerminatorless(node, isLabel);
  }
  printer.semicolon();
}
function BreakStatement(node) {
  this.word("break");
  printStatementAfterKeyword(this, node.label, true);
}
function ContinueStatement(node) {
  this.word("continue");
  printStatementAfterKeyword(this, node.label, true);
}
function ReturnStatement(node) {
  this.word("return");
  printStatementAfterKeyword(this, node.argument, false);
}
function ThrowStatement(node) {
  this.word("throw");
  printStatementAfterKeyword(this, node.argument, false);
}
function LabeledStatement(node) {
  this.print(node.label);
  this.tokenChar(58);
  this.space();
  this.print(node.body);
}
function TryStatement(node) {
  this.word("try");
  this.space();
  this.print(node.block);
  this.space();
  if (node.handlers) {
    this.print(node.handlers[0]);
  } else {
    this.print(node.handler);
  }
  if (node.finalizer) {
    this.space();
    this.word("finally");
    this.space();
    this.print(node.finalizer);
  }
}
function CatchClause(node) {
  this.word("catch");
  this.space();
  if (node.param) {
    this.tokenChar(40);
    this.print(node.param);
    this.print(node.param.typeAnnotation);
    this.tokenChar(41);
    this.space();
  }
  this.print(node.body);
}
function SwitchStatement(node) {
  this.word("switch");
  this.space();
  this.tokenChar(40);
  this.print(node.discriminant);
  this.tokenChar(41);
  this.space();
  this.tokenChar(123);
  this.printSequence(node.cases, {
    indent: true,
    addNewlines(leading, cas) {
      if (!leading && node.cases[node.cases.length - 1] === cas) return -1;
    }
  });
  this.rightBrace(node);
}
function SwitchCase(node) {
  if (node.test) {
    this.word("case");
    this.space();
    this.print(node.test);
    this.tokenChar(58);
  } else {
    this.word("default");
    this.tokenChar(58);
  }
  if (node.consequent.length) {
    this.newline();
    this.printSequence(node.consequent, {
      indent: true
    });
  }
}
function DebuggerStatement() {
  this.word("debugger");
  this.semicolon();
}
function VariableDeclaration(node, parent) {
  if (node.declare) {
    this.word("declare");
    this.space();
  }
  const {
    kind
  } = node;
  if (kind === "await using") {
    this.word("await");
    this.space();
    this.word("using", true);
  } else {
    this.word(kind, kind === "using");
  }
  this.space();
  let hasInits = false;
  if (!isFor(parent)) {
    for (const declar of node.declarations) {
      if (declar.init) {
        hasInits = true;
      }
    }
  }
  this.printList(node.declarations, {
    separator: hasInits ? function () {
      this.tokenChar(44);
      this.newline();
    } : undefined,
    indent: node.declarations.length > 1 ? true : false
  });
  if (isFor(parent)) {
    if (isForStatement(parent)) {
      if (parent.init === node) return;
    } else {
      if (parent.left === node) return;
    }
  }
  this.semicolon();
}
function VariableDeclarator(node) {
  this.print(node.id);
  if (node.definite) this.tokenChar(33);
  this.print(node.id.typeAnnotation);
  if (node.init) {
    this.space();
    this.tokenChar(61);
    this.space();
    this.print(node.init);
  }
}

//# sourceMappingURL=statements.js.map

}, function(modId) { var map = {"../node/index.js":1781530615288}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615295, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ClassAccessorProperty = ClassAccessorProperty;
exports.ClassBody = ClassBody;
exports.ClassExpression = exports.ClassDeclaration = ClassDeclaration;
exports.ClassMethod = ClassMethod;
exports.ClassPrivateMethod = ClassPrivateMethod;
exports.ClassPrivateProperty = ClassPrivateProperty;
exports.ClassProperty = ClassProperty;
exports.StaticBlock = StaticBlock;
exports._classMethodHead = _classMethodHead;
var _t = require("@babel/types");
const {
  isExportDefaultDeclaration,
  isExportNamedDeclaration
} = _t;
function ClassDeclaration(node, parent) {
  const inExport = isExportDefaultDeclaration(parent) || isExportNamedDeclaration(parent);
  if (!inExport || !this._shouldPrintDecoratorsBeforeExport(parent)) {
    this.printJoin(node.decorators);
  }
  if (node.declare) {
    this.word("declare");
    this.space();
  }
  if (node.abstract) {
    this.word("abstract");
    this.space();
  }
  this.word("class");
  if (node.id) {
    this.space();
    this.print(node.id);
  }
  this.print(node.typeParameters);
  if (node.superClass) {
    this.space();
    this.word("extends");
    this.space();
    this.print(node.superClass);
    this.print(node.superTypeParameters);
  }
  if (node.implements) {
    this.space();
    this.word("implements");
    this.space();
    this.printList(node.implements);
  }
  this.space();
  this.print(node.body);
}
function ClassBody(node) {
  this.tokenChar(123);
  if (node.body.length === 0) {
    this.tokenChar(125);
  } else {
    this.newline();
    const exit = this.enterForStatementInit(false);
    this.printSequence(node.body, {
      indent: true
    });
    exit();
    if (!this.endsWith(10)) this.newline();
    this.rightBrace(node);
  }
}
function ClassProperty(node) {
  var _node$key$loc;
  this.printJoin(node.decorators);
  const endLine = (_node$key$loc = node.key.loc) == null || (_node$key$loc = _node$key$loc.end) == null ? void 0 : _node$key$loc.line;
  if (endLine) this.catchUp(endLine);
  this.tsPrintClassMemberModifiers(node);
  if (node.computed) {
    this.tokenChar(91);
    this.print(node.key);
    this.tokenChar(93);
  } else {
    this._variance(node);
    this.print(node.key);
  }
  if (node.optional) {
    this.tokenChar(63);
  }
  if (node.definite) {
    this.tokenChar(33);
  }
  this.print(node.typeAnnotation);
  if (node.value) {
    this.space();
    this.tokenChar(61);
    this.space();
    this.print(node.value);
  }
  this.semicolon();
}
function ClassAccessorProperty(node) {
  var _node$key$loc2;
  this.printJoin(node.decorators);
  const endLine = (_node$key$loc2 = node.key.loc) == null || (_node$key$loc2 = _node$key$loc2.end) == null ? void 0 : _node$key$loc2.line;
  if (endLine) this.catchUp(endLine);
  this.tsPrintClassMemberModifiers(node);
  this.word("accessor", true);
  this.space();
  if (node.computed) {
    this.tokenChar(91);
    this.print(node.key);
    this.tokenChar(93);
  } else {
    this._variance(node);
    this.print(node.key);
  }
  if (node.optional) {
    this.tokenChar(63);
  }
  if (node.definite) {
    this.tokenChar(33);
  }
  this.print(node.typeAnnotation);
  if (node.value) {
    this.space();
    this.tokenChar(61);
    this.space();
    this.print(node.value);
  }
  this.semicolon();
}
function ClassPrivateProperty(node) {
  this.printJoin(node.decorators);
  if (node.static) {
    this.word("static");
    this.space();
  }
  this.print(node.key);
  this.print(node.typeAnnotation);
  if (node.value) {
    this.space();
    this.tokenChar(61);
    this.space();
    this.print(node.value);
  }
  this.semicolon();
}
function ClassMethod(node) {
  this._classMethodHead(node);
  this.space();
  this.print(node.body);
}
function ClassPrivateMethod(node) {
  this._classMethodHead(node);
  this.space();
  this.print(node.body);
}
function _classMethodHead(node) {
  var _node$key$loc3;
  this.printJoin(node.decorators);
  const endLine = (_node$key$loc3 = node.key.loc) == null || (_node$key$loc3 = _node$key$loc3.end) == null ? void 0 : _node$key$loc3.line;
  if (endLine) this.catchUp(endLine);
  this.tsPrintClassMemberModifiers(node);
  this._methodHead(node);
}
function StaticBlock(node) {
  this.word("static");
  this.space();
  this.tokenChar(123);
  if (node.body.length === 0) {
    this.tokenChar(125);
  } else {
    this.newline();
    this.printSequence(node.body, {
      indent: true
    });
    this.rightBrace(node);
  }
}

//# sourceMappingURL=classes.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615296, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ArrowFunctionExpression = ArrowFunctionExpression;
exports.FunctionDeclaration = exports.FunctionExpression = FunctionExpression;
exports._functionHead = _functionHead;
exports._methodHead = _methodHead;
exports._param = _param;
exports._parameters = _parameters;
exports._params = _params;
exports._predicate = _predicate;
var _t = require("@babel/types");
var _index = require("../node/index.js");
const {
  isIdentifier
} = _t;
function _params(node, idNode, parentNode) {
  this.print(node.typeParameters);
  const nameInfo = _getFuncIdName.call(this, idNode, parentNode);
  if (nameInfo) {
    this.sourceIdentifierName(nameInfo.name, nameInfo.pos);
  }
  this.tokenChar(40);
  this._parameters(node.params);
  this.tokenChar(41);
  const noLineTerminator = node.type === "ArrowFunctionExpression";
  this.print(node.returnType, noLineTerminator);
  this._noLineTerminator = noLineTerminator;
}
function _parameters(parameters) {
  const exit = this.enterForStatementInit(false);
  const paramLength = parameters.length;
  for (let i = 0; i < paramLength; i++) {
    this._param(parameters[i]);
    if (i < parameters.length - 1) {
      this.tokenChar(44);
      this.space();
    }
  }
  exit();
}
function _param(parameter) {
  this.printJoin(parameter.decorators);
  this.print(parameter);
  if (parameter.optional) {
    this.tokenChar(63);
  }
  this.print(parameter.typeAnnotation);
}
function _methodHead(node) {
  const kind = node.kind;
  const key = node.key;
  if (kind === "get" || kind === "set") {
    this.word(kind);
    this.space();
  }
  if (node.async) {
    this.word("async", true);
    this.space();
  }
  if (kind === "method" || kind === "init") {
    if (node.generator) {
      this.tokenChar(42);
    }
  }
  if (node.computed) {
    this.tokenChar(91);
    this.print(key);
    this.tokenChar(93);
  } else {
    this.print(key);
  }
  if (node.optional) {
    this.tokenChar(63);
  }
  this._params(node, node.computed && node.key.type !== "StringLiteral" ? undefined : node.key, undefined);
}
function _predicate(node, noLineTerminatorAfter) {
  if (node.predicate) {
    if (!node.returnType) {
      this.tokenChar(58);
    }
    this.space();
    this.print(node.predicate, noLineTerminatorAfter);
  }
}
function _functionHead(node, parent) {
  if (node.async) {
    this.word("async");
    this._endsWithInnerRaw = false;
    this.space();
  }
  this.word("function");
  if (node.generator) {
    this._endsWithInnerRaw = false;
    this.tokenChar(42);
  }
  this.space();
  if (node.id) {
    this.print(node.id);
  }
  this._params(node, node.id, parent);
  if (node.type !== "TSDeclareFunction") {
    this._predicate(node);
  }
}
function FunctionExpression(node, parent) {
  this._functionHead(node, parent);
  this.space();
  this.print(node.body);
}
function ArrowFunctionExpression(node, parent) {
  if (node.async) {
    this.word("async", true);
    this.space();
  }
  let firstParam;
  if (!this.format.retainLines && node.params.length === 1 && isIdentifier(firstParam = node.params[0]) && !hasTypesOrComments(node, firstParam)) {
    this.print(firstParam, true);
  } else {
    this._params(node, undefined, parent);
  }
  this._predicate(node, true);
  this.space();
  this.printInnerComments();
  this.token("=>");
  this.space();
  this.tokenContext |= _index.TokenContext.arrowBody;
  this.print(node.body);
}
function hasTypesOrComments(node, param) {
  var _param$leadingComment, _param$trailingCommen;
  return !!(node.typeParameters || node.returnType || node.predicate || param.typeAnnotation || param.optional || (_param$leadingComment = param.leadingComments) != null && _param$leadingComment.length || (_param$trailingCommen = param.trailingComments) != null && _param$trailingCommen.length);
}
function _getFuncIdName(idNode, parent) {
  let id = idNode;
  if (!id && parent) {
    const parentType = parent.type;
    if (parentType === "VariableDeclarator") {
      id = parent.id;
    } else if (parentType === "AssignmentExpression" || parentType === "AssignmentPattern") {
      id = parent.left;
    } else if (parentType === "ObjectProperty" || parentType === "ClassProperty") {
      if (!parent.computed || parent.key.type === "StringLiteral") {
        id = parent.key;
      }
    } else if (parentType === "ClassPrivateProperty" || parentType === "ClassAccessorProperty") {
      id = parent.key;
    }
  }
  if (!id) return;
  let nameInfo;
  if (id.type === "Identifier") {
    var _id$loc, _id$loc2;
    nameInfo = {
      pos: (_id$loc = id.loc) == null ? void 0 : _id$loc.start,
      name: ((_id$loc2 = id.loc) == null ? void 0 : _id$loc2.identifierName) || id.name
    };
  } else if (id.type === "PrivateName") {
    var _id$loc3;
    nameInfo = {
      pos: (_id$loc3 = id.loc) == null ? void 0 : _id$loc3.start,
      name: "#" + id.id.name
    };
  } else if (id.type === "StringLiteral") {
    var _id$loc4;
    nameInfo = {
      pos: (_id$loc4 = id.loc) == null ? void 0 : _id$loc4.start,
      name: id.value
    };
  }
  return nameInfo;
}

//# sourceMappingURL=methods.js.map

}, function(modId) { var map = {"../node/index.js":1781530615288}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615297, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ExportAllDeclaration = ExportAllDeclaration;
exports.ExportDefaultDeclaration = ExportDefaultDeclaration;
exports.ExportDefaultSpecifier = ExportDefaultSpecifier;
exports.ExportNamedDeclaration = ExportNamedDeclaration;
exports.ExportNamespaceSpecifier = ExportNamespaceSpecifier;
exports.ExportSpecifier = ExportSpecifier;
exports.ImportAttribute = ImportAttribute;
exports.ImportDeclaration = ImportDeclaration;
exports.ImportDefaultSpecifier = ImportDefaultSpecifier;
exports.ImportExpression = ImportExpression;
exports.ImportNamespaceSpecifier = ImportNamespaceSpecifier;
exports.ImportSpecifier = ImportSpecifier;
exports._printAttributes = _printAttributes;
var _t = require("@babel/types");
var _index = require("../node/index.js");
const {
  isClassDeclaration,
  isExportDefaultSpecifier,
  isExportNamespaceSpecifier,
  isImportDefaultSpecifier,
  isImportNamespaceSpecifier,
  isStatement
} = _t;
function ImportSpecifier(node) {
  if (node.importKind === "type" || node.importKind === "typeof") {
    this.word(node.importKind);
    this.space();
  }
  this.print(node.imported);
  if (node.local && node.local.name !== node.imported.name) {
    this.space();
    this.word("as");
    this.space();
    this.print(node.local);
  }
}
function ImportDefaultSpecifier(node) {
  this.print(node.local);
}
function ExportDefaultSpecifier(node) {
  this.print(node.exported);
}
function ExportSpecifier(node) {
  if (node.exportKind === "type") {
    this.word("type");
    this.space();
  }
  this.print(node.local);
  if (node.exported && node.local.name !== node.exported.name) {
    this.space();
    this.word("as");
    this.space();
    this.print(node.exported);
  }
}
function ExportNamespaceSpecifier(node) {
  this.tokenChar(42);
  this.space();
  this.word("as");
  this.space();
  this.print(node.exported);
}
let warningShown = false;
function _printAttributes(node) {
  const {
    importAttributesKeyword
  } = this.format;
  const {
    attributes,
    assertions
  } = node;
  if (attributes && !importAttributesKeyword && !warningShown) {
    warningShown = true;
    console.warn(`\
You are using import attributes, without specifying the desired output syntax.
Please specify the "importAttributesKeyword" generator option, whose value can be one of:
 - "with"        : \`import { a } from "b" with { type: "json" };\`
 - "assert"      : \`import { a } from "b" assert { type: "json" };\`
 - "with-legacy" : \`import { a } from "b" with type: "json";\`
`);
  }
  const useAssertKeyword = importAttributesKeyword === "assert" || !importAttributesKeyword && assertions;
  this.word(useAssertKeyword ? "assert" : "with");
  this.space();
  if (!useAssertKeyword && importAttributesKeyword !== "with") {
    this.printList(attributes || assertions);
    return;
  }
  this.tokenChar(123);
  this.space();
  this.printList(attributes || assertions);
  this.space();
  this.tokenChar(125);
}
function ExportAllDeclaration(node) {
  var _node$attributes, _node$assertions;
  this.word("export");
  this.space();
  if (node.exportKind === "type") {
    this.word("type");
    this.space();
  }
  this.tokenChar(42);
  this.space();
  this.word("from");
  this.space();
  if ((_node$attributes = node.attributes) != null && _node$attributes.length || (_node$assertions = node.assertions) != null && _node$assertions.length) {
    this.print(node.source, true);
    this.space();
    this._printAttributes(node);
  } else {
    this.print(node.source);
  }
  this.semicolon();
}
function maybePrintDecoratorsBeforeExport(printer, node) {
  if (isClassDeclaration(node.declaration) && printer._shouldPrintDecoratorsBeforeExport(node)) {
    printer.printJoin(node.declaration.decorators);
  }
}
function ExportNamedDeclaration(node) {
  maybePrintDecoratorsBeforeExport(this, node);
  this.word("export");
  this.space();
  if (node.declaration) {
    const declar = node.declaration;
    this.print(declar);
    if (!isStatement(declar)) this.semicolon();
  } else {
    if (node.exportKind === "type") {
      this.word("type");
      this.space();
    }
    const specifiers = node.specifiers.slice(0);
    let hasSpecial = false;
    for (;;) {
      const first = specifiers[0];
      if (isExportDefaultSpecifier(first) || isExportNamespaceSpecifier(first)) {
        hasSpecial = true;
        this.print(specifiers.shift());
        if (specifiers.length) {
          this.tokenChar(44);
          this.space();
        }
      } else {
        break;
      }
    }
    if (specifiers.length || !specifiers.length && !hasSpecial) {
      this.tokenChar(123);
      if (specifiers.length) {
        this.space();
        this.printList(specifiers);
        this.space();
      }
      this.tokenChar(125);
    }
    if (node.source) {
      var _node$attributes2, _node$assertions2;
      this.space();
      this.word("from");
      this.space();
      if ((_node$attributes2 = node.attributes) != null && _node$attributes2.length || (_node$assertions2 = node.assertions) != null && _node$assertions2.length) {
        this.print(node.source, true);
        this.space();
        this._printAttributes(node);
      } else {
        this.print(node.source);
      }
    }
    this.semicolon();
  }
}
function ExportDefaultDeclaration(node) {
  maybePrintDecoratorsBeforeExport(this, node);
  this.word("export");
  this.noIndentInnerCommentsHere();
  this.space();
  this.word("default");
  this.space();
  this.tokenContext |= _index.TokenContext.exportDefault;
  const declar = node.declaration;
  this.print(declar);
  if (!isStatement(declar)) this.semicolon();
}
function ImportDeclaration(node) {
  var _node$attributes3, _node$assertions3;
  this.word("import");
  this.space();
  const isTypeKind = node.importKind === "type" || node.importKind === "typeof";
  if (isTypeKind) {
    this.noIndentInnerCommentsHere();
    this.word(node.importKind);
    this.space();
  } else if (node.module) {
    this.noIndentInnerCommentsHere();
    this.word("module");
    this.space();
  } else if (node.phase) {
    this.noIndentInnerCommentsHere();
    this.word(node.phase);
    this.space();
  }
  const specifiers = node.specifiers.slice(0);
  const hasSpecifiers = !!specifiers.length;
  while (hasSpecifiers) {
    const first = specifiers[0];
    if (isImportDefaultSpecifier(first) || isImportNamespaceSpecifier(first)) {
      this.print(specifiers.shift());
      if (specifiers.length) {
        this.tokenChar(44);
        this.space();
      }
    } else {
      break;
    }
  }
  if (specifiers.length) {
    this.tokenChar(123);
    this.space();
    this.printList(specifiers);
    this.space();
    this.tokenChar(125);
  } else if (isTypeKind && !hasSpecifiers) {
    this.tokenChar(123);
    this.tokenChar(125);
  }
  if (hasSpecifiers || isTypeKind) {
    this.space();
    this.word("from");
    this.space();
  }
  if ((_node$attributes3 = node.attributes) != null && _node$attributes3.length || (_node$assertions3 = node.assertions) != null && _node$assertions3.length) {
    this.print(node.source, true);
    this.space();
    this._printAttributes(node);
  } else {
    this.print(node.source);
  }
  this.semicolon();
}
function ImportAttribute(node) {
  this.print(node.key);
  this.tokenChar(58);
  this.space();
  this.print(node.value);
}
function ImportNamespaceSpecifier(node) {
  this.tokenChar(42);
  this.space();
  this.word("as");
  this.space();
  this.print(node.local);
}
function ImportExpression(node) {
  this.word("import");
  if (node.phase) {
    this.tokenChar(46);
    this.word(node.phase);
  }
  this.tokenChar(40);
  this.print(node.source);
  if (node.options != null) {
    this.tokenChar(44);
    this.space();
    this.print(node.options);
  }
  this.tokenChar(41);
}

//# sourceMappingURL=modules.js.map

}, function(modId) { var map = {"../node/index.js":1781530615288}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615298, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ArgumentPlaceholder = ArgumentPlaceholder;
exports.ArrayPattern = exports.ArrayExpression = ArrayExpression;
exports.BigIntLiteral = BigIntLiteral;
exports.BooleanLiteral = BooleanLiteral;
exports.DecimalLiteral = DecimalLiteral;
exports.Identifier = Identifier;
exports.NullLiteral = NullLiteral;
exports.NumericLiteral = NumericLiteral;
exports.ObjectPattern = exports.ObjectExpression = ObjectExpression;
exports.ObjectMethod = ObjectMethod;
exports.ObjectProperty = ObjectProperty;
exports.PipelineBareFunction = PipelineBareFunction;
exports.PipelinePrimaryTopicReference = PipelinePrimaryTopicReference;
exports.PipelineTopicExpression = PipelineTopicExpression;
exports.RecordExpression = RecordExpression;
exports.RegExpLiteral = RegExpLiteral;
exports.SpreadElement = exports.RestElement = RestElement;
exports.StringLiteral = StringLiteral;
exports.TopicReference = TopicReference;
exports.TupleExpression = TupleExpression;
var _t = require("@babel/types");
var _jsesc = require("jsesc");
const {
  isAssignmentPattern,
  isIdentifier
} = _t;
function Identifier(node) {
  var _node$loc;
  this.sourceIdentifierName(((_node$loc = node.loc) == null ? void 0 : _node$loc.identifierName) || node.name);
  this.word(node.name);
}
function ArgumentPlaceholder() {
  this.tokenChar(63);
}
function RestElement(node) {
  this.token("...");
  this.print(node.argument);
}
function ObjectExpression(node) {
  const props = node.properties;
  this.tokenChar(123);
  if (props.length) {
    const exit = this.enterForStatementInit(false);
    this.space();
    this.printList(props, {
      indent: true,
      statement: true
    });
    this.space();
    exit();
  }
  this.sourceWithOffset("end", node.loc, -1);
  this.tokenChar(125);
}
function ObjectMethod(node) {
  this.printJoin(node.decorators);
  this._methodHead(node);
  this.space();
  this.print(node.body);
}
function ObjectProperty(node) {
  this.printJoin(node.decorators);
  if (node.computed) {
    this.tokenChar(91);
    this.print(node.key);
    this.tokenChar(93);
  } else {
    if (isAssignmentPattern(node.value) && isIdentifier(node.key) && node.key.name === node.value.left.name) {
      this.print(node.value);
      return;
    }
    this.print(node.key);
    if (node.shorthand && isIdentifier(node.key) && isIdentifier(node.value) && node.key.name === node.value.name) {
      return;
    }
  }
  this.tokenChar(58);
  this.space();
  this.print(node.value);
}
function ArrayExpression(node) {
  const elems = node.elements;
  const len = elems.length;
  this.tokenChar(91);
  const exit = this.enterForStatementInit(false);
  for (let i = 0; i < elems.length; i++) {
    const elem = elems[i];
    if (elem) {
      if (i > 0) this.space();
      this.print(elem);
      if (i < len - 1) this.tokenChar(44);
    } else {
      this.tokenChar(44);
    }
  }
  exit();
  this.tokenChar(93);
}
function RecordExpression(node) {
  const props = node.properties;
  let startToken;
  let endToken;
  {
    if (this.format.recordAndTupleSyntaxType === "bar") {
      startToken = "{|";
      endToken = "|}";
    } else if (this.format.recordAndTupleSyntaxType !== "hash" && this.format.recordAndTupleSyntaxType != null) {
      throw new Error(`The "recordAndTupleSyntaxType" generator option must be "bar" or "hash" (${JSON.stringify(this.format.recordAndTupleSyntaxType)} received).`);
    } else {
      startToken = "#{";
      endToken = "}";
    }
  }
  this.token(startToken);
  if (props.length) {
    this.space();
    this.printList(props, {
      indent: true,
      statement: true
    });
    this.space();
  }
  this.token(endToken);
}
function TupleExpression(node) {
  const elems = node.elements;
  const len = elems.length;
  let startToken;
  let endToken;
  {
    if (this.format.recordAndTupleSyntaxType === "bar") {
      startToken = "[|";
      endToken = "|]";
    } else if (this.format.recordAndTupleSyntaxType === "hash") {
      startToken = "#[";
      endToken = "]";
    } else {
      throw new Error(`${this.format.recordAndTupleSyntaxType} is not a valid recordAndTuple syntax type`);
    }
  }
  this.token(startToken);
  for (let i = 0; i < elems.length; i++) {
    const elem = elems[i];
    if (elem) {
      if (i > 0) this.space();
      this.print(elem);
      if (i < len - 1) this.tokenChar(44);
    }
  }
  this.token(endToken);
}
function RegExpLiteral(node) {
  this.word(`/${node.pattern}/${node.flags}`);
}
function BooleanLiteral(node) {
  this.word(node.value ? "true" : "false");
}
function NullLiteral() {
  this.word("null");
}
function NumericLiteral(node) {
  const raw = this.getPossibleRaw(node);
  const opts = this.format.jsescOption;
  const value = node.value;
  const str = value + "";
  if (opts.numbers) {
    this.number(_jsesc(value, opts), value);
  } else if (raw == null) {
    this.number(str, value);
  } else if (this.format.minified) {
    this.number(raw.length < str.length ? raw : str, value);
  } else {
    this.number(raw, value);
  }
}
function StringLiteral(node) {
  const raw = this.getPossibleRaw(node);
  if (!this.format.minified && raw !== undefined) {
    this.token(raw);
    return;
  }
  const val = _jsesc(node.value, this.format.jsescOption);
  this.token(val);
}
function BigIntLiteral(node) {
  const raw = this.getPossibleRaw(node);
  if (!this.format.minified && raw !== undefined) {
    this.word(raw);
    return;
  }
  this.word(node.value + "n");
}
function DecimalLiteral(node) {
  const raw = this.getPossibleRaw(node);
  if (!this.format.minified && raw !== undefined) {
    this.word(raw);
    return;
  }
  this.word(node.value + "m");
}
const validTopicTokenSet = new Set(["^^", "@@", "^", "%", "#"]);
function TopicReference() {
  const {
    topicToken
  } = this.format;
  if (validTopicTokenSet.has(topicToken)) {
    this.token(topicToken);
  } else {
    const givenTopicTokenJSON = JSON.stringify(topicToken);
    const validTopics = Array.from(validTopicTokenSet, v => JSON.stringify(v));
    throw new Error(`The "topicToken" generator option must be one of ` + `${validTopics.join(", ")} (${givenTopicTokenJSON} received instead).`);
  }
}
function PipelineTopicExpression(node) {
  this.print(node.expression);
}
function PipelineBareFunction(node) {
  this.print(node.callee);
}
function PipelinePrimaryTopicReference() {
  this.tokenChar(35);
}

//# sourceMappingURL=types.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615299, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.AnyTypeAnnotation = AnyTypeAnnotation;
exports.ArrayTypeAnnotation = ArrayTypeAnnotation;
exports.BooleanLiteralTypeAnnotation = BooleanLiteralTypeAnnotation;
exports.BooleanTypeAnnotation = BooleanTypeAnnotation;
exports.DeclareClass = DeclareClass;
exports.DeclareExportAllDeclaration = DeclareExportAllDeclaration;
exports.DeclareExportDeclaration = DeclareExportDeclaration;
exports.DeclareFunction = DeclareFunction;
exports.DeclareInterface = DeclareInterface;
exports.DeclareModule = DeclareModule;
exports.DeclareModuleExports = DeclareModuleExports;
exports.DeclareOpaqueType = DeclareOpaqueType;
exports.DeclareTypeAlias = DeclareTypeAlias;
exports.DeclareVariable = DeclareVariable;
exports.DeclaredPredicate = DeclaredPredicate;
exports.EmptyTypeAnnotation = EmptyTypeAnnotation;
exports.EnumBooleanBody = EnumBooleanBody;
exports.EnumBooleanMember = EnumBooleanMember;
exports.EnumDeclaration = EnumDeclaration;
exports.EnumDefaultedMember = EnumDefaultedMember;
exports.EnumNumberBody = EnumNumberBody;
exports.EnumNumberMember = EnumNumberMember;
exports.EnumStringBody = EnumStringBody;
exports.EnumStringMember = EnumStringMember;
exports.EnumSymbolBody = EnumSymbolBody;
exports.ExistsTypeAnnotation = ExistsTypeAnnotation;
exports.FunctionTypeAnnotation = FunctionTypeAnnotation;
exports.FunctionTypeParam = FunctionTypeParam;
exports.IndexedAccessType = IndexedAccessType;
exports.InferredPredicate = InferredPredicate;
exports.InterfaceDeclaration = InterfaceDeclaration;
exports.GenericTypeAnnotation = exports.ClassImplements = exports.InterfaceExtends = InterfaceExtends;
exports.InterfaceTypeAnnotation = InterfaceTypeAnnotation;
exports.IntersectionTypeAnnotation = IntersectionTypeAnnotation;
exports.MixedTypeAnnotation = MixedTypeAnnotation;
exports.NullLiteralTypeAnnotation = NullLiteralTypeAnnotation;
exports.NullableTypeAnnotation = NullableTypeAnnotation;
Object.defineProperty(exports, "NumberLiteralTypeAnnotation", {
  enumerable: true,
  get: function () {
    return _types2.NumericLiteral;
  }
});
exports.NumberTypeAnnotation = NumberTypeAnnotation;
exports.ObjectTypeAnnotation = ObjectTypeAnnotation;
exports.ObjectTypeCallProperty = ObjectTypeCallProperty;
exports.ObjectTypeIndexer = ObjectTypeIndexer;
exports.ObjectTypeInternalSlot = ObjectTypeInternalSlot;
exports.ObjectTypeProperty = ObjectTypeProperty;
exports.ObjectTypeSpreadProperty = ObjectTypeSpreadProperty;
exports.OpaqueType = OpaqueType;
exports.OptionalIndexedAccessType = OptionalIndexedAccessType;
exports.QualifiedTypeIdentifier = QualifiedTypeIdentifier;
Object.defineProperty(exports, "StringLiteralTypeAnnotation", {
  enumerable: true,
  get: function () {
    return _types2.StringLiteral;
  }
});
exports.StringTypeAnnotation = StringTypeAnnotation;
exports.SymbolTypeAnnotation = SymbolTypeAnnotation;
exports.ThisTypeAnnotation = ThisTypeAnnotation;
exports.TupleTypeAnnotation = TupleTypeAnnotation;
exports.TypeAlias = TypeAlias;
exports.TypeAnnotation = TypeAnnotation;
exports.TypeCastExpression = TypeCastExpression;
exports.TypeParameter = TypeParameter;
exports.TypeParameterDeclaration = exports.TypeParameterInstantiation = TypeParameterInstantiation;
exports.TypeofTypeAnnotation = TypeofTypeAnnotation;
exports.UnionTypeAnnotation = UnionTypeAnnotation;
exports.Variance = Variance;
exports.VoidTypeAnnotation = VoidTypeAnnotation;
exports._interfaceish = _interfaceish;
exports._variance = _variance;
var _t = require("@babel/types");
var _modules = require("./modules.js");
var _index = require("../node/index.js");
var _types2 = require("./types.js");
const {
  isDeclareExportDeclaration,
  isStatement
} = _t;
function AnyTypeAnnotation() {
  this.word("any");
}
function ArrayTypeAnnotation(node) {
  this.print(node.elementType, true);
  this.tokenChar(91);
  this.tokenChar(93);
}
function BooleanTypeAnnotation() {
  this.word("boolean");
}
function BooleanLiteralTypeAnnotation(node) {
  this.word(node.value ? "true" : "false");
}
function NullLiteralTypeAnnotation() {
  this.word("null");
}
function DeclareClass(node, parent) {
  if (!isDeclareExportDeclaration(parent)) {
    this.word("declare");
    this.space();
  }
  this.word("class");
  this.space();
  this._interfaceish(node);
}
function DeclareFunction(node, parent) {
  if (!isDeclareExportDeclaration(parent)) {
    this.word("declare");
    this.space();
  }
  this.word("function");
  this.space();
  this.print(node.id);
  this.print(node.id.typeAnnotation.typeAnnotation);
  if (node.predicate) {
    this.space();
    this.print(node.predicate);
  }
  this.semicolon();
}
function InferredPredicate() {
  this.tokenChar(37);
  this.word("checks");
}
function DeclaredPredicate(node) {
  this.tokenChar(37);
  this.word("checks");
  this.tokenChar(40);
  this.print(node.value);
  this.tokenChar(41);
}
function DeclareInterface(node) {
  this.word("declare");
  this.space();
  this.InterfaceDeclaration(node);
}
function DeclareModule(node) {
  this.word("declare");
  this.space();
  this.word("module");
  this.space();
  this.print(node.id);
  this.space();
  this.print(node.body);
}
function DeclareModuleExports(node) {
  this.word("declare");
  this.space();
  this.word("module");
  this.tokenChar(46);
  this.word("exports");
  this.print(node.typeAnnotation);
}
function DeclareTypeAlias(node) {
  this.word("declare");
  this.space();
  this.TypeAlias(node);
}
function DeclareOpaqueType(node, parent) {
  if (!isDeclareExportDeclaration(parent)) {
    this.word("declare");
    this.space();
  }
  this.OpaqueType(node);
}
function DeclareVariable(node, parent) {
  if (!isDeclareExportDeclaration(parent)) {
    this.word("declare");
    this.space();
  }
  this.word("var");
  this.space();
  this.print(node.id);
  this.print(node.id.typeAnnotation);
  this.semicolon();
}
function DeclareExportDeclaration(node) {
  this.word("declare");
  this.space();
  this.word("export");
  this.space();
  if (node.default) {
    this.word("default");
    this.space();
  }
  FlowExportDeclaration.call(this, node);
}
function DeclareExportAllDeclaration(node) {
  this.word("declare");
  this.space();
  _modules.ExportAllDeclaration.call(this, node);
}
function EnumDeclaration(node) {
  const {
    id,
    body
  } = node;
  this.word("enum");
  this.space();
  this.print(id);
  this.print(body);
}
function enumExplicitType(context, name, hasExplicitType) {
  if (hasExplicitType) {
    context.space();
    context.word("of");
    context.space();
    context.word(name);
  }
  context.space();
}
function enumBody(context, node) {
  const {
    members
  } = node;
  context.token("{");
  context.indent();
  context.newline();
  for (const member of members) {
    context.print(member);
    context.newline();
  }
  if (node.hasUnknownMembers) {
    context.token("...");
    context.newline();
  }
  context.dedent();
  context.token("}");
}
function EnumBooleanBody(node) {
  const {
    explicitType
  } = node;
  enumExplicitType(this, "boolean", explicitType);
  enumBody(this, node);
}
function EnumNumberBody(node) {
  const {
    explicitType
  } = node;
  enumExplicitType(this, "number", explicitType);
  enumBody(this, node);
}
function EnumStringBody(node) {
  const {
    explicitType
  } = node;
  enumExplicitType(this, "string", explicitType);
  enumBody(this, node);
}
function EnumSymbolBody(node) {
  enumExplicitType(this, "symbol", true);
  enumBody(this, node);
}
function EnumDefaultedMember(node) {
  const {
    id
  } = node;
  this.print(id);
  this.tokenChar(44);
}
function enumInitializedMember(context, node) {
  context.print(node.id);
  context.space();
  context.token("=");
  context.space();
  context.print(node.init);
  context.token(",");
}
function EnumBooleanMember(node) {
  enumInitializedMember(this, node);
}
function EnumNumberMember(node) {
  enumInitializedMember(this, node);
}
function EnumStringMember(node) {
  enumInitializedMember(this, node);
}
function FlowExportDeclaration(node) {
  if (node.declaration) {
    const declar = node.declaration;
    this.print(declar);
    if (!isStatement(declar)) this.semicolon();
  } else {
    this.tokenChar(123);
    if (node.specifiers.length) {
      this.space();
      this.printList(node.specifiers);
      this.space();
    }
    this.tokenChar(125);
    if (node.source) {
      this.space();
      this.word("from");
      this.space();
      this.print(node.source);
    }
    this.semicolon();
  }
}
function ExistsTypeAnnotation() {
  this.tokenChar(42);
}
function FunctionTypeAnnotation(node, parent) {
  this.print(node.typeParameters);
  this.tokenChar(40);
  if (node.this) {
    this.word("this");
    this.tokenChar(58);
    this.space();
    this.print(node.this.typeAnnotation);
    if (node.params.length || node.rest) {
      this.tokenChar(44);
      this.space();
    }
  }
  this.printList(node.params);
  if (node.rest) {
    if (node.params.length) {
      this.tokenChar(44);
      this.space();
    }
    this.token("...");
    this.print(node.rest);
  }
  this.tokenChar(41);
  const type = parent == null ? void 0 : parent.type;
  if (type != null && (type === "ObjectTypeCallProperty" || type === "ObjectTypeInternalSlot" || type === "DeclareFunction" || type === "ObjectTypeProperty" && parent.method)) {
    this.tokenChar(58);
  } else {
    this.space();
    this.token("=>");
  }
  this.space();
  this.print(node.returnType);
}
function FunctionTypeParam(node) {
  this.print(node.name);
  if (node.optional) this.tokenChar(63);
  if (node.name) {
    this.tokenChar(58);
    this.space();
  }
  this.print(node.typeAnnotation);
}
function InterfaceExtends(node) {
  this.print(node.id);
  this.print(node.typeParameters, true);
}
function _interfaceish(node) {
  var _node$extends;
  this.print(node.id);
  this.print(node.typeParameters);
  if ((_node$extends = node.extends) != null && _node$extends.length) {
    this.space();
    this.word("extends");
    this.space();
    this.printList(node.extends);
  }
  if (node.type === "DeclareClass") {
    var _node$mixins, _node$implements;
    if ((_node$mixins = node.mixins) != null && _node$mixins.length) {
      this.space();
      this.word("mixins");
      this.space();
      this.printList(node.mixins);
    }
    if ((_node$implements = node.implements) != null && _node$implements.length) {
      this.space();
      this.word("implements");
      this.space();
      this.printList(node.implements);
    }
  }
  this.space();
  this.print(node.body);
}
function _variance(node) {
  var _node$variance;
  const kind = (_node$variance = node.variance) == null ? void 0 : _node$variance.kind;
  if (kind != null) {
    if (kind === "plus") {
      this.tokenChar(43);
    } else if (kind === "minus") {
      this.tokenChar(45);
    }
  }
}
function InterfaceDeclaration(node) {
  this.word("interface");
  this.space();
  this._interfaceish(node);
}
function andSeparator() {
  this.space();
  this.tokenChar(38);
  this.space();
}
function InterfaceTypeAnnotation(node) {
  var _node$extends2;
  this.word("interface");
  if ((_node$extends2 = node.extends) != null && _node$extends2.length) {
    this.space();
    this.word("extends");
    this.space();
    this.printList(node.extends);
  }
  this.space();
  this.print(node.body);
}
function IntersectionTypeAnnotation(node) {
  this.printJoin(node.types, {
    separator: andSeparator
  });
}
function MixedTypeAnnotation() {
  this.word("mixed");
}
function EmptyTypeAnnotation() {
  this.word("empty");
}
function NullableTypeAnnotation(node) {
  this.tokenChar(63);
  this.print(node.typeAnnotation);
}
function NumberTypeAnnotation() {
  this.word("number");
}
function StringTypeAnnotation() {
  this.word("string");
}
function ThisTypeAnnotation() {
  this.word("this");
}
function TupleTypeAnnotation(node) {
  this.tokenChar(91);
  this.printList(node.types);
  this.tokenChar(93);
}
function TypeofTypeAnnotation(node) {
  this.word("typeof");
  this.space();
  this.print(node.argument);
}
function TypeAlias(node) {
  this.word("type");
  this.space();
  this.print(node.id);
  this.print(node.typeParameters);
  this.space();
  this.tokenChar(61);
  this.space();
  this.print(node.right);
  this.semicolon();
}
function TypeAnnotation(node, parent) {
  this.tokenChar(58);
  this.space();
  if (parent.type === "ArrowFunctionExpression") {
    this.tokenContext |= _index.TokenContext.arrowFlowReturnType;
  } else if (node.optional) {
    this.tokenChar(63);
  }
  this.print(node.typeAnnotation);
}
function TypeParameterInstantiation(node) {
  this.tokenChar(60);
  this.printList(node.params, {});
  this.tokenChar(62);
}
function TypeParameter(node) {
  this._variance(node);
  this.word(node.name);
  if (node.bound) {
    this.print(node.bound);
  }
  if (node.default) {
    this.space();
    this.tokenChar(61);
    this.space();
    this.print(node.default);
  }
}
function OpaqueType(node) {
  this.word("opaque");
  this.space();
  this.word("type");
  this.space();
  this.print(node.id);
  this.print(node.typeParameters);
  if (node.supertype) {
    this.tokenChar(58);
    this.space();
    this.print(node.supertype);
  }
  if (node.impltype) {
    this.space();
    this.tokenChar(61);
    this.space();
    this.print(node.impltype);
  }
  this.semicolon();
}
function ObjectTypeAnnotation(node) {
  if (node.exact) {
    this.token("{|");
  } else {
    this.tokenChar(123);
  }
  const props = [...node.properties, ...(node.callProperties || []), ...(node.indexers || []), ...(node.internalSlots || [])];
  if (props.length) {
    this.newline();
    this.space();
    this.printJoin(props, {
      addNewlines(leading) {
        if (leading && !props[0]) return 1;
      },
      indent: true,
      statement: true,
      iterator: () => {
        if (props.length !== 1 || node.inexact) {
          this.tokenChar(44);
          this.space();
        }
      }
    });
    this.space();
  }
  if (node.inexact) {
    this.indent();
    this.token("...");
    if (props.length) {
      this.newline();
    }
    this.dedent();
  }
  if (node.exact) {
    this.token("|}");
  } else {
    this.tokenChar(125);
  }
}
function ObjectTypeInternalSlot(node) {
  if (node.static) {
    this.word("static");
    this.space();
  }
  this.tokenChar(91);
  this.tokenChar(91);
  this.print(node.id);
  this.tokenChar(93);
  this.tokenChar(93);
  if (node.optional) this.tokenChar(63);
  if (!node.method) {
    this.tokenChar(58);
    this.space();
  }
  this.print(node.value);
}
function ObjectTypeCallProperty(node) {
  if (node.static) {
    this.word("static");
    this.space();
  }
  this.print(node.value);
}
function ObjectTypeIndexer(node) {
  if (node.static) {
    this.word("static");
    this.space();
  }
  this._variance(node);
  this.tokenChar(91);
  if (node.id) {
    this.print(node.id);
    this.tokenChar(58);
    this.space();
  }
  this.print(node.key);
  this.tokenChar(93);
  this.tokenChar(58);
  this.space();
  this.print(node.value);
}
function ObjectTypeProperty(node) {
  if (node.proto) {
    this.word("proto");
    this.space();
  }
  if (node.static) {
    this.word("static");
    this.space();
  }
  if (node.kind === "get" || node.kind === "set") {
    this.word(node.kind);
    this.space();
  }
  this._variance(node);
  this.print(node.key);
  if (node.optional) this.tokenChar(63);
  if (!node.method) {
    this.tokenChar(58);
    this.space();
  }
  this.print(node.value);
}
function ObjectTypeSpreadProperty(node) {
  this.token("...");
  this.print(node.argument);
}
function QualifiedTypeIdentifier(node) {
  this.print(node.qualification);
  this.tokenChar(46);
  this.print(node.id);
}
function SymbolTypeAnnotation() {
  this.word("symbol");
}
function orSeparator() {
  this.space();
  this.tokenChar(124);
  this.space();
}
function UnionTypeAnnotation(node) {
  this.printJoin(node.types, {
    separator: orSeparator
  });
}
function TypeCastExpression(node) {
  this.tokenChar(40);
  this.print(node.expression);
  this.print(node.typeAnnotation);
  this.tokenChar(41);
}
function Variance(node) {
  if (node.kind === "plus") {
    this.tokenChar(43);
  } else {
    this.tokenChar(45);
  }
}
function VoidTypeAnnotation() {
  this.word("void");
}
function IndexedAccessType(node) {
  this.print(node.objectType, true);
  this.tokenChar(91);
  this.print(node.indexType);
  this.tokenChar(93);
}
function OptionalIndexedAccessType(node) {
  this.print(node.objectType);
  if (node.optional) {
    this.token("?.");
  }
  this.tokenChar(91);
  this.print(node.indexType);
  this.tokenChar(93);
}

//# sourceMappingURL=flow.js.map

}, function(modId) { var map = {"./modules.js":1781530615297,"../node/index.js":1781530615288,"./types.js":1781530615298}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615300, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.BlockStatement = BlockStatement;
exports.Directive = Directive;
exports.DirectiveLiteral = DirectiveLiteral;
exports.File = File;
exports.InterpreterDirective = InterpreterDirective;
exports.Placeholder = Placeholder;
exports.Program = Program;
function File(node) {
  if (node.program) {
    this.print(node.program.interpreter);
  }
  this.print(node.program);
}
function Program(node) {
  var _node$directives;
  this.noIndentInnerCommentsHere();
  this.printInnerComments();
  const directivesLen = (_node$directives = node.directives) == null ? void 0 : _node$directives.length;
  if (directivesLen) {
    var _node$directives$trai;
    const newline = node.body.length ? 2 : 1;
    this.printSequence(node.directives, {
      trailingCommentsLineOffset: newline
    });
    if (!((_node$directives$trai = node.directives[directivesLen - 1].trailingComments) != null && _node$directives$trai.length)) {
      this.newline(newline);
    }
  }
  this.printSequence(node.body);
}
function BlockStatement(node) {
  var _node$directives2;
  this.tokenChar(123);
  const directivesLen = (_node$directives2 = node.directives) == null ? void 0 : _node$directives2.length;
  if (directivesLen) {
    var _node$directives$trai2;
    const newline = node.body.length ? 2 : 1;
    this.printSequence(node.directives, {
      indent: true,
      trailingCommentsLineOffset: newline
    });
    if (!((_node$directives$trai2 = node.directives[directivesLen - 1].trailingComments) != null && _node$directives$trai2.length)) {
      this.newline(newline);
    }
  }
  const exit = this.enterForStatementInit(false);
  this.printSequence(node.body, {
    indent: true
  });
  exit();
  this.rightBrace(node);
}
function Directive(node) {
  this.print(node.value);
  this.semicolon();
}
const unescapedSingleQuoteRE = /(?:^|[^\\])(?:\\\\)*'/;
const unescapedDoubleQuoteRE = /(?:^|[^\\])(?:\\\\)*"/;
function DirectiveLiteral(node) {
  const raw = this.getPossibleRaw(node);
  if (!this.format.minified && raw !== undefined) {
    this.token(raw);
    return;
  }
  const {
    value
  } = node;
  if (!unescapedDoubleQuoteRE.test(value)) {
    this.token(`"${value}"`);
  } else if (!unescapedSingleQuoteRE.test(value)) {
    this.token(`'${value}'`);
  } else {
    throw new Error("Malformed AST: it is not possible to print a directive containing" + " both unescaped single and double quotes.");
  }
}
function InterpreterDirective(node) {
  this.token(`#!${node.value}`);
  this.newline(1, true);
}
function Placeholder(node) {
  this.token("%%");
  this.print(node.name);
  this.token("%%");
  if (node.expectedNode === "Statement") {
    this.semicolon();
  }
}

//# sourceMappingURL=base.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615301, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.JSXAttribute = JSXAttribute;
exports.JSXClosingElement = JSXClosingElement;
exports.JSXClosingFragment = JSXClosingFragment;
exports.JSXElement = JSXElement;
exports.JSXEmptyExpression = JSXEmptyExpression;
exports.JSXExpressionContainer = JSXExpressionContainer;
exports.JSXFragment = JSXFragment;
exports.JSXIdentifier = JSXIdentifier;
exports.JSXMemberExpression = JSXMemberExpression;
exports.JSXNamespacedName = JSXNamespacedName;
exports.JSXOpeningElement = JSXOpeningElement;
exports.JSXOpeningFragment = JSXOpeningFragment;
exports.JSXSpreadAttribute = JSXSpreadAttribute;
exports.JSXSpreadChild = JSXSpreadChild;
exports.JSXText = JSXText;
function JSXAttribute(node) {
  this.print(node.name);
  if (node.value) {
    this.tokenChar(61);
    this.print(node.value);
  }
}
function JSXIdentifier(node) {
  this.word(node.name);
}
function JSXNamespacedName(node) {
  this.print(node.namespace);
  this.tokenChar(58);
  this.print(node.name);
}
function JSXMemberExpression(node) {
  this.print(node.object);
  this.tokenChar(46);
  this.print(node.property);
}
function JSXSpreadAttribute(node) {
  this.tokenChar(123);
  this.token("...");
  this.print(node.argument);
  this.rightBrace(node);
}
function JSXExpressionContainer(node) {
  this.tokenChar(123);
  this.print(node.expression);
  this.rightBrace(node);
}
function JSXSpreadChild(node) {
  this.tokenChar(123);
  this.token("...");
  this.print(node.expression);
  this.rightBrace(node);
}
function JSXText(node) {
  const raw = this.getPossibleRaw(node);
  if (raw !== undefined) {
    this.token(raw, true);
  } else {
    this.token(node.value, true);
  }
}
function JSXElement(node) {
  const open = node.openingElement;
  this.print(open);
  if (open.selfClosing) return;
  this.indent();
  for (const child of node.children) {
    this.print(child);
  }
  this.dedent();
  this.print(node.closingElement);
}
function spaceSeparator() {
  this.space();
}
function JSXOpeningElement(node) {
  this.tokenChar(60);
  this.print(node.name);
  this.print(node.typeParameters);
  if (node.attributes.length > 0) {
    this.space();
    this.printJoin(node.attributes, {
      separator: spaceSeparator
    });
  }
  if (node.selfClosing) {
    this.space();
    this.token("/>");
  } else {
    this.tokenChar(62);
  }
}
function JSXClosingElement(node) {
  this.token("</");
  this.print(node.name);
  this.tokenChar(62);
}
function JSXEmptyExpression() {
  this.printInnerComments();
}
function JSXFragment(node) {
  this.print(node.openingFragment);
  this.indent();
  for (const child of node.children) {
    this.print(child);
  }
  this.dedent();
  this.print(node.closingFragment);
}
function JSXOpeningFragment() {
  this.tokenChar(60);
  this.tokenChar(62);
}
function JSXClosingFragment() {
  this.token("</");
  this.tokenChar(62);
}

//# sourceMappingURL=jsx.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
__DEFINE__(1781530615302, function(require, module, exports) {


Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.TSAnyKeyword = TSAnyKeyword;
exports.TSArrayType = TSArrayType;
exports.TSSatisfiesExpression = exports.TSAsExpression = TSTypeExpression;
exports.TSBigIntKeyword = TSBigIntKeyword;
exports.TSBooleanKeyword = TSBooleanKeyword;
exports.TSCallSignatureDeclaration = TSCallSignatureDeclaration;
exports.TSConditionalType = TSConditionalType;
exports.TSConstructSignatureDeclaration = TSConstructSignatureDeclaration;
exports.TSConstructorType = TSConstructorType;
exports.TSDeclareFunction = TSDeclareFunction;
exports.TSDeclareMethod = TSDeclareMethod;
exports.TSEnumDeclaration = TSEnumDeclaration;
exports.TSEnumMember = TSEnumMember;
exports.TSExportAssignment = TSExportAssignment;
exports.TSExpressionWithTypeArguments = TSExpressionWithTypeArguments;
exports.TSExternalModuleReference = TSExternalModuleReference;
exports.TSFunctionType = TSFunctionType;
exports.TSImportEqualsDeclaration = TSImportEqualsDeclaration;
exports.TSImportType = TSImportType;
exports.TSIndexSignature = TSIndexSignature;
exports.TSIndexedAccessType = TSIndexedAccessType;
exports.TSInferType = TSInferType;
exports.TSInstantiationExpression = TSInstantiationExpression;
exports.TSInterfaceBody = TSInterfaceBody;
exports.TSInterfaceDeclaration = TSInterfaceDeclaration;
exports.TSIntersectionType = TSIntersectionType;
exports.TSIntrinsicKeyword = TSIntrinsicKeyword;
exports.TSLiteralType = TSLiteralType;
exports.TSMappedType = TSMappedType;
exports.TSMethodSignature = TSMethodSignature;
exports.TSModuleBlock = TSModuleBlock;
exports.TSModuleDeclaration = TSModuleDeclaration;
exports.TSNamedTupleMember = TSNamedTupleMember;
exports.TSNamespaceExportDeclaration = TSNamespaceExportDeclaration;
exports.TSNeverKeyword = TSNeverKeyword;
exports.TSNonNullExpression = TSNonNullExpression;
exports.TSNullKeyword = TSNullKeyword;
exports.TSNumberKeyword = TSNumberKeyword;
exports.TSObjectKeyword = TSObjectKeyword;
exports.TSOptionalType = TSOptionalType;
exports.TSParameterProperty = TSParameterProperty;
exports.TSParenthesizedType = TSParenthesizedType;
exports.TSPropertySignature = TSPropertySignature;
exports.TSQualifiedName = TSQualifiedName;
exports.TSRestType = TSRestType;
exports.TSStringKeyword = TSStringKeyword;
exports.TSSymbolKeyword = TSSymbolKeyword;
exports.TSThisType = TSThisType;
exports.TSTupleType = TSTupleType;
exports.TSTypeAliasDeclaration = TSTypeAliasDeclaration;
exports.TSTypeAnnotation = TSTypeAnnotation;
exports.TSTypeAssertion = TSTypeAssertion;
exports.TSTypeLiteral = TSTypeLiteral;
exports.TSTypeOperator = TSTypeOperator;
exports.TSTypeParameter = TSTypeParameter;
exports.TSTypeParameterDeclaration = exports.TSTypeParameterInstantiation = TSTypeParameterInstantiation;
exports.TSTypePredicate = TSTypePredicate;
exports.TSTypeQuery = TSTypeQuery;
exports.TSTypeReference = TSTypeReference;
exports.TSUndefinedKeyword = TSUndefinedKeyword;
exports.TSUnionType = TSUnionType;
exports.TSUnknownKeyword = TSUnknownKeyword;
exports.TSVoidKeyword = TSVoidKeyword;
exports.tsPrintClassMemberModifiers = tsPrintClassMemberModifiers;
exports.tsPrintFunctionOrConstructorType = tsPrintFunctionOrConstructorType;
exports.tsPrintPropertyOrMethodName = tsPrintPropertyOrMethodName;
exports.tsPrintSignatureDeclarationBase = tsPrintSignatureDeclarationBase;
exports.tsPrintTypeLiteralOrInterfaceBody = tsPrintTypeLiteralOrInterfaceBody;
function TSTypeAnnotation(node) {
  this.tokenChar(58);
  this.space();
  if (node.optional) this.tokenChar(63);
  this.print(node.typeAnnotation);
}
function TSTypeParameterInstantiation(node, parent) {
  this.tokenChar(60);
  this.printList(node.params, {});
  if (parent.type === "ArrowFunctionExpression" && node.params.length === 1) {
    this.tokenChar(44);
  }
  this.tokenChar(62);
}
function TSTypeParameter(node) {
  if (node.in) {
    this.word("in");
    this.space();
  }
  if (node.out) {
    this.word("out");
    this.space();
  }
  this.word(node.name);
  if (node.constraint) {
    this.space();
    this.word("extends");
    this.space();
    this.print(node.constraint);
  }
  if (node.default) {
    this.space();
    this.tokenChar(61);
    this.space();
    this.print(node.default);
  }
}
function TSParameterProperty(node) {
  if (node.accessibility) {
    this.word(node.accessibility);
    this.space();
  }
  if (node.readonly) {
    this.word("readonly");
    this.space();
  }
  this._param(node.parameter);
}
function TSDeclareFunction(node, parent) {
  if (node.declare) {
    this.word("declare");
    this.space();
  }
  this._functionHead(node, parent);
  this.semicolon();
}
function TSDeclareMethod(node) {
  this._classMethodHead(node);
  this.semicolon();
}
function TSQualifiedName(node) {
  this.print(node.left);
  this.tokenChar(46);
  this.print(node.right);
}
function TSCallSignatureDeclaration(node) {
  this.tsPrintSignatureDeclarationBase(node);
  this.semicolon();
}
function TSConstructSignatureDeclaration(node) {
  this.word("new");
  this.space();
  this.tsPrintSignatureDeclarationBase(node);
  this.semicolon();
}
function TSPropertySignature(node) {
  const {
    readonly
  } = node;
  if (readonly) {
    this.word("readonly");
    this.space();
  }
  this.tsPrintPropertyOrMethodName(node);
  this.print(node.typeAnnotation);
  this.semicolon();
}
function tsPrintPropertyOrMethodName(node) {
  if (node.computed) {
    this.tokenChar(91);
  }
  this.print(node.key);
  if (node.computed) {
    this.tokenChar(93);
  }
  if (node.optional) {
    this.tokenChar(63);
  }
}
function TSMethodSignature(node) {
  const {
    kind
  } = node;
  if (kind === "set" || kind === "get") {
    this.word(kind);
    this.space();
  }
  this.tsPrintPropertyOrMethodName(node);
  this.tsPrintSignatureDeclarationBase(node);
  this.semicolon();
}
function TSIndexSignature(node) {
  const {
    readonly,
    static: isStatic
  } = node;
  if (isStatic) {
    this.word("static");
    this.space();
  }
  if (readonly) {
    this.word("readonly");
    this.space();
  }
  this.tokenChar(91);
  this._parameters(node.parameters);
  this.tokenChar(93);
  this.print(node.typeAnnotation);
  this.semicolon();
}
function TSAnyKeyword() {
  this.word("any");
}
function TSBigIntKeyword() {
  this.word("bigint");
}
function TSUnknownKeyword() {
  this.word("unknown");
}
function TSNumberKeyword() {
  this.word("number");
}
function TSObjectKeyword() {
  this.word("object");
}
function TSBooleanKeyword() {
  this.word("boolean");
}
function TSStringKeyword() {
  this.word("string");
}
function TSSymbolKeyword() {
  this.word("symbol");
}
function TSVoidKeyword() {
  this.word("void");
}
function TSUndefinedKeyword() {
  this.word("undefined");
}
function TSNullKeyword() {
  this.word("null");
}
function TSNeverKeyword() {
  this.word("never");
}
function TSIntrinsicKeyword() {
  this.word("intrinsic");
}
function TSThisType() {
  this.word("this");
}
function TSFunctionType(node) {
  this.tsPrintFunctionOrConstructorType(node);
}
function TSConstructorType(node) {
  if (node.abstract) {
    this.word("abstract");
    this.space();
  }
  this.word("new");
  this.space();
  this.tsPrintFunctionOrConstructorType(node);
}
function tsPrintFunctionOrConstructorType(node) {
  const {
    typeParameters
  } = node;
  const parameters = node.parameters;
  this.print(typeParameters);
  this.tokenChar(40);
  this._parameters(parameters);
  this.tokenChar(41);
  this.space();
  this.token("=>");
  this.space();
  const returnType = node.typeAnnotation;
  this.print(returnType.typeAnnotation);
}
function TSTypeReference(node) {
  this.print(node.typeName, true);
  this.print(node.typeParameters, true);
}
function TSTypePredicate(node) {
  if (node.asserts) {
    this.word("asserts");
    this.space();
  }
  this.print(node.parameterName);
  if (node.typeAnnotation) {
    this.space();
    this.word("is");
    this.space();
    this.print(node.typeAnnotation.typeAnnotation);
  }
}
function TSTypeQuery(node) {
  this.word("typeof");
  this.space();
  this.print(node.exprName);
  if (node.typeParameters) {
    this.print(node.typeParameters);
  }
}
function TSTypeLiteral(node) {
  this.tsPrintTypeLiteralOrInterfaceBody(node.members, node);
}
function tsPrintTypeLiteralOrInterfaceBody(members, node) {
  tsPrintBraced(this, members, node);
}
function tsPrintBraced(printer, members, node) {
  printer.token("{");
  if (members.length) {
    printer.indent();
    printer.newline();
    for (const member of members) {
      printer.print(member);
      printer.newline();
    }
    printer.dedent();
  }
  printer.rightBrace(node);
}
function TSArrayType(node) {
  this.print(node.elementType, true);
  this.tokenChar(91);
  this.tokenChar(93);
}
function TSTupleType(node) {
  this.tokenChar(91);
  this.printList(node.elementTypes);
  this.tokenChar(93);
}
function TSOptionalType(node) {
  this.print(node.typeAnnotation);
  this.tokenChar(63);
}
function TSRestType(node) {
  this.token("...");
  this.print(node.typeAnnotation);
}
function TSNamedTupleMember(node) {
  this.print(node.label);
  if (node.optional) this.tokenChar(63);
  this.tokenChar(58);
  this.space();
  this.print(node.elementType);
}
function TSUnionType(node) {
  tsPrintUnionOrIntersectionType(this, node, "|");
}
function TSIntersectionType(node) {
  tsPrintUnionOrIntersectionType(this, node, "&");
}
function tsPrintUnionOrIntersectionType(printer, node, sep) {
  printer.printJoin(node.types, {
    separator() {
      this.space();
      this.token(sep);
      this.space();
    }
  });
}
function TSConditionalType(node) {
  this.print(node.checkType);
  this.space();
  this.word("extends");
  this.space();
  this.print(node.extendsType);
  this.space();
  this.tokenChar(63);
  this.space();
  this.print(node.trueType);
  this.space();
  this.tokenChar(58);
  this.space();
  this.print(node.falseType);
}
function TSInferType(node) {
  this.token("infer");
  this.space();
  this.print(node.typeParameter);
}
function TSParenthesizedType(node) {
  this.tokenChar(40);
  this.print(node.typeAnnotation);
  this.tokenChar(41);
}
function TSTypeOperator(node) {
  this.word(node.operator);
  this.space();
  this.print(node.typeAnnotation);
}
function TSIndexedAccessType(node) {
  this.print(node.objectType, true);
  this.tokenChar(91);
  this.print(node.indexType);
  this.tokenChar(93);
}
function TSMappedType(node) {
  const {
    nameType,
    optional,
    readonly,
    typeAnnotation
  } = node;
  this.tokenChar(123);
  this.space();
  if (readonly) {
    tokenIfPlusMinus(this, readonly);
    this.word("readonly");
    this.space();
  }
  this.tokenChar(91);
  {
    this.word(node.typeParameter.name);
  }
  this.space();
  this.word("in");
  this.space();
  {
    this.print(node.typeParameter.constraint);
  }
  if (nameType) {
    this.space();
    this.word("as");
    this.space();
    this.print(nameType);
  }
  this.tokenChar(93);
  if (optional) {
    tokenIfPlusMinus(this, optional);
    this.tokenChar(63);
  }
  if (typeAnnotation) {
    this.tokenChar(58);
    this.space();
    this.print(typeAnnotation);
  }
  this.space();
  this.tokenChar(125);
}
function tokenIfPlusMinus(self, tok) {
  if (tok !== true) {
    self.token(tok);
  }
}
function TSLiteralType(node) {
  this.print(node.literal);
}
function TSExpressionWithTypeArguments(node) {
  this.print(node.expression);
  this.print(node.typeParameters);
}
function TSInterfaceDeclaration(node) {
  const {
    declare,
    id,
    typeParameters,
    extends: extendz,
    body
  } = node;
  if (declare) {
    this.word("declare");
    this.space();
  }
  this.word("interface");
  this.space();
  this.print(id);
  this.print(typeParameters);
  if (extendz != null && extendz.length) {
    this.space();
    this.word("extends");
    this.space();
    this.printList(extendz);
  }
  this.space();
  this.print(body);
}
function TSInterfaceBody(node) {
  this.tsPrintTypeLiteralOrInterfaceBody(node.body, node);
}
function TSTypeAliasDeclaration(node) {
  const {
    declare,
    id,
    typeParameters,
    typeAnnotation
  } = node;
  if (declare) {
    this.word("declare");
    this.space();
  }
  this.word("type");
  this.space();
  this.print(id);
  this.print(typeParameters);
  this.space();
  this.tokenChar(61);
  this.space();
  this.print(typeAnnotation);
  this.semicolon();
}
function TSTypeExpression(node) {
  var _expression$trailingC;
  const {
    type,
    expression,
    typeAnnotation
  } = node;
  const forceParens = (_expression$trailingC = expression.trailingComments) == null ? void 0 : _expression$trailingC.some(c => c.type === "CommentLine" || /[\r\n\u2028\u2029]/.test(c.value));
  this.print(expression, true, undefined, forceParens);
  this.space();
  this.word(type === "TSAsExpression" ? "as" : "satisfies");
  this.space();
  this.print(typeAnnotation);
}
function TSTypeAssertion(node) {
  const {
    typeAnnotation,
    expression
  } = node;
  this.tokenChar(60);
  this.print(typeAnnotation);
  this.tokenChar(62);
  this.space();
  this.print(expression);
}
function TSInstantiationExpression(node) {
  this.print(node.expression);
  this.print(node.typeParameters);
}
function TSEnumDeclaration(node) {
  const {
    declare,
    const: isConst,
    id,
    members
  } = node;
  if (declare) {
    this.word("declare");
    this.space();
  }
  if (isConst) {
    this.word("const");
    this.space();
  }
  this.word("enum");
  this.space();
  this.print(id);
  this.space();
  tsPrintBraced(this, members, node);
}
function TSEnumMember(node) {
  const {
    id,
    initializer
  } = node;
  this.print(id);
  if (initializer) {
    this.space();
    this.tokenChar(61);
    this.space();
    this.print(initializer);
  }
  this.tokenChar(44);
}
function TSModuleDeclaration(node) {
  const {
    declare,
    id
  } = node;
  if (declare) {
    this.word("declare");
    this.space();
  }
  if (!node.global) {
    this.word(id.type === "Identifier" ? "namespace" : "module");
    this.space();
  }
  this.print(id);
  if (!node.body) {
    this.semicolon();
    return;
  }
  let body = node.body;
  while (body.type === "TSModuleDeclaration") {
    this.tokenChar(46);
    this.print(body.id);
    body = body.body;
  }
  this.space();
  this.print(body);
}
function TSModuleBlock(node) {
  tsPrintBraced(this, node.body, node);
}
function TSImportType(node) {
  const {
    argument,
    qualifier,
    typeParameters
  } = node;
  this.word("import");
  this.tokenChar(40);
  this.print(argument);
  this.tokenChar(41);
  if (qualifier) {
    this.tokenChar(46);
    this.print(qualifier);
  }
  if (typeParameters) {
    this.print(typeParameters);
  }
}
function TSImportEqualsDeclaration(node) {
  const {
    isExport,
    id,
    moduleReference
  } = node;
  if (isExport) {
    this.word("export");
    this.space();
  }
  this.word("import");
  this.space();
  this.print(id);
  this.space();
  this.tokenChar(61);
  this.space();
  this.print(moduleReference);
  this.semicolon();
}
function TSExternalModuleReference(node) {
  this.token("require(");
  this.print(node.expression);
  this.tokenChar(41);
}
function TSNonNullExpression(node) {
  this.print(node.expression);
  this.tokenChar(33);
}
function TSExportAssignment(node) {
  this.word("export");
  this.space();
  this.tokenChar(61);
  this.space();
  this.print(node.expression);
  this.semicolon();
}
function TSNamespaceExportDeclaration(node) {
  this.word("export");
  this.space();
  this.word("as");
  this.space();
  this.word("namespace");
  this.space();
  this.print(node.id);
  this.semicolon();
}
function tsPrintSignatureDeclarationBase(node) {
  const {
    typeParameters
  } = node;
  const parameters = node.parameters;
  this.print(typeParameters);
  this.tokenChar(40);
  this._parameters(parameters);
  this.tokenChar(41);
  const returnType = node.typeAnnotation;
  this.print(returnType);
}
function tsPrintClassMemberModifiers(node) {
  const isField = node.type === "ClassAccessorProperty" || node.type === "ClassProperty";
  if (isField && node.declare) {
    this.word("declare");
    this.space();
  }
  if (node.accessibility) {
    this.word(node.accessibility);
    this.space();
  }
  if (node.static) {
    this.word("static");
    this.space();
  }
  if (node.override) {
    this.word("override");
    this.space();
  }
  if (node.abstract) {
    this.word("abstract");
    this.space();
  }
  if (isField && node.readonly) {
    this.word("readonly");
    this.space();
  }
}

//# sourceMappingURL=typescript.js.map

}, function(modId) { var map = {}; return __REQUIRE__(map[modId], modId); })
return __REQUIRE__(1781530615284);
})()
//miniprogram-npm-outsideDeps=["@jridgewell/gen-mapping","@jridgewell/trace-mapping","@babel/types","jsesc"]
//# sourceMappingURL=index.js.map