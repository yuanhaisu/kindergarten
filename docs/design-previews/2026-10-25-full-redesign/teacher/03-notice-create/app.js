(function () {
'use strict';
var CATS = { homework: '作业', bring: '带物', activity: '活动', health: '健康', other: '其他' };
var CLASSES = [{ id: 'c1', name: '小一班', count: 28 }, { id: 'c2', name: '中二班', count: 26 }];
var TEMPLATES = [
{ title: '秋游通知', category: 'activity', content: '明天上午 8:30 幼儿园门口集合，请为孩子准备水杯、湿巾和一套换洗衣物。' },
{ title: '手足口病预防提醒', category: 'health', content: '近期为高发期，请家长每天检查孩子手心、脚心和口腔，发现红点及时就医并告知老师。' },
{ title: '带物提醒', category: 'bring', content: '明天请带一包湿巾和一盒彩笔，谢谢配合。' }
];
var DEFAULT = {
category: 'homework',
title: '明天请带一盒 12 色水彩笔',
content: '明天上午的美术课要用，请家长今晚帮孩子放进书包。水彩笔请贴上名字，避免拿错。',
attachments: [{ art: 'pens' }, { art: 'blobs' }],
classIds: ['c1'], needConfirm: true, scheduled: false, date: '2026-10-27', time: '09:00'
};
var ART = ['pens', 'blobs'];
var YEARS = [2025, 2026, 2027, 2028];
var ITEM = 44;
var state = { view: 'normal' };
var errors = { title: false, content: false };
var publishing = false;
/* 发布成功后置位：源码 toast 后 navigateBack 离页；原型留页复位，并锁住发布防重复群发 */
var published = false;
var draftClasses = [];
var dateDraft = { y: 2026, m: 10, d: 27 };
var toastTimer = null, toastHide = null;
function $(id) { return document.getElementById(id); }
function each(list, fn) { Array.prototype.forEach.call(list, fn); }
function pad2(n) { return (n < 10 ? '0' : '') + n; }
function clamp(n, min, max) { return Math.max(min, Math.min(max, n)); }
function daysInMonth(y, m) { return new Date(y, m, 0).getDate(); }
function catLabel(id) { return CATS[id] || '其他'; }
function shortDate(iso) { var p = iso.split('-'); return Number(p[1]) + '月' + Number(p[2]) + '日'; }
function fullDate(iso) { var p = iso.split('-'); return p[0] + '年' + Number(p[1]) + '月' + Number(p[2]) + '日'; }
function selectedClasses() { return CLASSES.filter(function (c) { return state.classIds.indexOf(c.id) !== -1; }); }
var app = $('app'), scroll = $('scroll');
var titleInput = $('titleInput'), contentInput = $('contentInput');
var titleCount = $('titleCount'), titleError = $('titleError'), contentError = $('contentError');
var contentCount = $('contentCount');
var thumbs = $('thumbs'), attachCount = $('attachCount'), categoryChips = $('categoryChips');
var classRow = $('classRow'), classValue = $('classValue'), classSkeleton = $('classSkeleton');
var classError = $('classError'), classChev = $('classChev'), classFoot = $('classFoot'), retryBtn = $('retryBtn');
var confirmSwitch = $('confirmSwitch'), schedSwitch = $('schedSwitch'), schedHint = $('schedHint');
var scheduleFields = $('scheduleFields'), dateValue = $('dateValue'), timeValue = $('timeValue');
var tplRow = $('tplRow'), tplCount = $('tplCount'), tplSkeleton = $('tplSkeleton');
var dockSummary = $('dockSummary'), publishBtn = $('publishBtn');
var publishLabel = $('publishLabel'), publishSpinner = $('publishSpinner');
var toast = $('toast'), toastText = $('toastText'), toastIcon = $('toastIcon');
function syncChips(view) {each(document.querySelectorAll('[data-mp-state]'), function (chip) {
chip.setAttribute('aria-pressed', String(chip.dataset.mpState === view));
});
}
var classLayer = $('classLayer'), classList = $('classList'), classDone = $('classDone');
var dateLayer = $('dateLayer'), timeLayer = $('timeLayer'), tplLayer = $('tplLayer');
var tplList = $('tplList'), confirmLayer = $('confirmLayer');
var dlgDesc = $('dlgDesc'), dlgCat = $('dlgCat'), dlgWhen = $('dlgWhen');
var confirmPublish = $('confirmPublish'), confirmLabel = $('confirmLabel');
var dialogCancel = $('dialogCancel');
var yearWheel = $('yearWheel'), monthWheel = $('monthWheel'), dayWheel = $('dayWheel');
var hourWheel = $('hourWheel'), minuteWheel = $('minuteWheel');
var dateDone = $('dateDone'), timeDone = $('timeDone');
function resetValues() {
state.category = DEFAULT.category;
state.title = DEFAULT.title;
state.content = DEFAULT.content;
state.attachments = DEFAULT.attachments.map(function (a) { return { art: a.art }; });
state.classIds = DEFAULT.classIds.slice();
state.needConfirm = DEFAULT.needConfirm;
state.scheduled = DEFAULT.scheduled;
state.date = DEFAULT.date;
state.time = DEFAULT.time;
errors.title = false;
errors.content = false;
}
/* 复位成空表单：不保留已发出的内容，因此无法原样重发 */
function resetDraft() {
resetValues();
state.title = '';
state.content = '';
state.attachments = [];
}
resetValues();
function autoGrow() {
contentInput.style.height = 'auto';
contentInput.style.height = clamp(contentInput.scrollHeight, 84, 260) + 'px';
}
function renderCategory() {
each(categoryChips.querySelectorAll('.chip'), function (btn) {
var on = btn.dataset.cat === state.category;
btn.classList.toggle('is-on', on);
btn.setAttribute('aria-checked', String(on));
});
}
function renderAttachments() {
var html = state.attachments.map(function (a, i) {
var inner = a.art === 'pens' ? '<i></i><i></i><i></i><i></i>' : '<i></i><i></i><i></i>';
return '<figure class="thumb" aria-label="附件 ' + (i + 1) + '">' +
'<span class="art art-' + a.art + '" aria-hidden="true">' + inner + '</span>' +
'<button class="thumb-del" type="button" data-del="' + i + '" aria-label="删除附件 ' + (i + 1) + '"><span aria-hidden="true">×</span></button>' +
'</figure>';
}).join('');
if (state.attachments.length < 9) {
html += '<button class="add-thumb" type="button" id="addThumb" aria-label="添加图片或视频"><span class="plus" aria-hidden="true">+</span><span>添加</span></button>';
}
thumbs.innerHTML = html;
attachCount.textContent = state.attachments.length + '/9';
}
function renderSchedule() {
confirmSwitch.setAttribute('aria-checked', String(state.needConfirm));
schedSwitch.setAttribute('aria-checked', String(state.scheduled));
schedHint.textContent = state.scheduled ? shortDate(state.date) + ' ' + state.time + ' 自动发送' : '立即发送';
scheduleFields.hidden = !state.scheduled;
dateValue.textContent = fullDate(state.date);
timeValue.textContent = state.time;
}
function renderClassRow() {
var usable = state.view === 'normal' || state.view === 'empty';
var sel = selectedClasses();
var total = sel.reduce(function (sum, c) { return sum + c.count; }, 0);
classRow.setAttribute('aria-disabled', String(!usable));
classRow.setAttribute('tabindex', usable ? '0' : '-1');
classValue.textContent = sel.length ? sel.map(function (c) { return c.name; }).join('、') + '全体家长' : '请选择班级';
if (state.view === 'loading') classFoot.textContent = '正在加载任教班级…';
else if (state.view === 'error') classFoot.textContent = '请检查网络后重试';
else if (sel.length) classFoot.textContent = sel.length + ' 个班级 · ' + total + ' 位家长将收到通知';
else classFoot.textContent = '还没有选择接收班级';
}
function renderTemplatesRow() {
tplRow.closest('.group').hidden = !TEMPLATES.length;
var loading = state.view === 'loading';
tplRow.setAttribute('aria-disabled', String(loading));
tplRow.setAttribute('tabindex', loading ? '-1' : '0');
tplCount.textContent = TEMPLATES.length + ' 条可用';
}
function summaryText() {
if (published) return '通知已发布 · 表单已复位';
if (state.view === 'loading') return '正在加载班级信息…';
if (state.view === 'error') return '班级信息未加载，暂不能发布';
var missing = [];
if (!state.title.trim()) missing.push('标题');
if (!state.content.trim()) missing.push('正文');
if (missing.length) return '还差' + missing.join('和');
var sel = selectedClasses();
var scope = sel.length ? sel.map(function (c) { return c.name; }).join('、') : '请选择班级';
var when = state.scheduled ? '定时 ' + shortDate(state.date) + ' ' + state.time : '立即发送';
var confirm = state.needConfirm ? '需家长确认' : '无需确认';
return scope + ' · ' + when + ' · ' + confirm;
}
function renderPublish() {
var disabled = published || state.view !== 'normal' || publishing;
publishBtn.disabled = disabled;
publishBtn.setAttribute('aria-disabled', String(disabled));
publishBtn.classList.toggle('is-publishing', publishing);
publishLabel.textContent = publishing ? '发布中…' : (published ? '已发布' : '发布');
publishSpinner.hidden = !publishing;
}
function renderChrome() {
if (published && (state.title.trim() || state.content.trim() || state.attachments.length)) published = false;
app.dataset.state = state.view;
syncChips(state.view);
titleCount.textContent = state.title.length + ' / 80';
contentCount.textContent = state.content.length + ' 字';
titleError.hidden = !errors.title;
contentError.hidden = !errors.content;
renderClassRow();
renderSchedule();
renderTemplatesRow();
dockSummary.textContent = summaryText();
renderPublish();
each(document.querySelectorAll('.demo-chip'), function (chip) {
chip.classList.toggle('is-on', chip.dataset.view === state.view);
});
}
function renderAll() {
titleInput.value = state.title;
contentInput.value = state.content;
autoGrow();
renderCategory();
renderAttachments();
renderChrome();
}
function setView(view) {
published = false;
state.view = view;
resetValues();
if (view === 'empty') {
state.title = '';
state.content = '';
state.attachments = [];
errors.title = true;
errors.content = true;
}
renderAll();
window.scrollTo({ top: 0, behavior: 'smooth' });
}
function showToast(message, kind) {
if (toastHide) { clearTimeout(toastHide); toastHide = null; }
toastText.textContent = message;
toastIcon.textContent = kind === 'warn' ? '!' : '✓';
toast.hidden = false;
requestAnimationFrame(function () { toast.classList.add('is-show'); });
if (toastTimer) clearTimeout(toastTimer);
toastTimer = setTimeout(function () {
toast.classList.remove('is-show');
toastHide = setTimeout(function () { toast.hidden = true; }, 220);
}, 1900);
}
categoryChips.addEventListener('click', function (e) {
var btn = e.target.closest('[data-cat]');
if (!btn) return;
state.category = btn.dataset.cat;
renderCategory();
renderChrome();
});
titleInput.addEventListener('input', function () {
state.title = titleInput.value;
errors.title = false;
if (state.view === 'empty') state.view = 'normal';
renderChrome();
});
contentInput.addEventListener('input', function () {
state.content = contentInput.value;
errors.content = false;
if (state.view === 'empty') state.view = 'normal';
autoGrow();
renderChrome();
});
thumbs.addEventListener('click', function (e) {
var del = e.target.closest('[data-del]');
if (del) {
state.attachments.splice(Number(del.dataset.del), 1);
if (state.view === 'empty') state.view = 'normal';
renderAttachments();
renderChrome();
return;
}
if (e.target.closest('#addThumb')) {
if (state.attachments.length >= 9) return;
state.attachments.push({ art: ART[state.attachments.length % ART.length] });
if (state.view === 'empty') state.view = 'normal';
renderAttachments();
renderChrome();
}
});
confirmSwitch.addEventListener('click', function () {
state.needConfirm = !state.needConfirm;
renderChrome();
});
schedSwitch.addEventListener('click', function () {
state.scheduled = !state.scheduled;
renderChrome();
});
$('dateRow').addEventListener('click', openDateSheet);
$('timeRow').addEventListener('click', openTimeSheet);
$('dateRow').addEventListener('keydown', function (e) {
if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openDateSheet(); }
});
$('timeRow').addEventListener('keydown', function (e) {
if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openTimeSheet(); }
});
classRow.addEventListener('click', function (e) {
if (e.target.closest('#retryBtn')) return;
if (state.view !== 'normal' && state.view !== 'empty') return;
openClassSheet();
});
classRow.addEventListener('keydown', function (e) {
if (e.target.closest('#retryBtn')) return;
if (e.key !== 'Enter' && e.key !== ' ') return;
if (state.view !== 'normal' && state.view !== 'empty') return;
e.preventDefault();
openClassSheet();
});
retryBtn.addEventListener('click', function (e) {
e.stopPropagation();
setView('normal');
showToast('班级列表已更新');
});
tplRow.addEventListener('click', function () {
if (state.view !== 'loading') openLayer(tplLayer);
});
tplRow.addEventListener('keydown', function (e) {
if (e.key !== 'Enter' && e.key !== ' ') return;
if (state.view === 'loading') return;
e.preventDefault();
openLayer(tplLayer);
});
function openLayer(layer) { layer.classList.add('is-open'); }
function closeLayer(layer) { layer.classList.remove('is-open'); }
each(document.querySelectorAll('[data-close]'), function (el) {
el.addEventListener('click', function () {
if (publishing && el.dataset.close === 'confirm') return;
var map = { 'class': classLayer, date: dateLayer, time: timeLayer, tpl: tplLayer, confirm: confirmLayer };
if (map[el.dataset.close]) closeLayer(map[el.dataset.close]);
});
});
document.addEventListener('keydown', function (e) {
if (e.key !== 'Escape') return;
var open = document.querySelector('.layer.is-open');
if (!open) return;
if (publishing && open === confirmLayer) return;
closeLayer(open);
});
function renderClassList() {
classList.innerHTML = CLASSES.map(function (c) {
var on = draftClasses.indexOf(c.id) !== -1;
return '<button class="pick-row" type="button" data-class-id="' + c.id + '" aria-pressed="' + on + '">' +
'<span class="pick-name">' + c.name + '</span><span class="pick-meta">' + c.count + ' 人</span>' +
'<span class="check-dot" aria-hidden="true"></span></button>';
}).join('');
}
function openClassSheet() {
draftClasses = state.classIds.slice();
renderClassList();
openLayer(classLayer);
}
classList.addEventListener('click', function (e) {
var row = e.target.closest('[data-class-id]');
if (!row) return;
var at = draftClasses.indexOf(row.dataset.classId);
if (at === -1) draftClasses.push(row.dataset.classId);
else draftClasses.splice(at, 1);
renderClassList();
});
classDone.addEventListener('click', function () {
state.classIds = draftClasses.slice();
closeLayer(classLayer);
renderChrome();
});
function renderTplSheet() {
tplList.innerHTML = TEMPLATES.map(function (t, i) {
return '<div class="tpl-row" role="button" tabindex="0" data-tpl="' + i + '">' +
'<div class="tpl-text"><p class="tpl-title">' + t.title + '</p><p class="tpl-preview">' + t.content + '</p></div>' +
'<span class="use-btn">使用</span></div>';
}).join('');
}
function applyTemplate(row) {
var t = TEMPLATES[Number(row.dataset.tpl)];
if (!t) return;
state.category = t.category;
state.title = t.title;
state.content = t.content;
errors.title = false;
errors.content = false;
if (state.view === 'empty') state.view = 'normal';
closeLayer(tplLayer);
renderAll();
window.scrollTo({ top: 0, behavior: 'smooth' });
showToast('已填入「' + t.title + '」');
}
tplList.addEventListener('click', function (e) {
var row = e.target.closest('[data-tpl]');
if (row) applyTemplate(row);
});
tplList.addEventListener('keydown', function (e) {
if (e.key !== 'Enter' && e.key !== ' ') return;
var row = e.target.closest('[data-tpl]');
if (!row) return;
e.preventDefault();
applyTemplate(row);
});
function paintWheel(el) {
each(el.children, function (child, i) { child.classList.toggle('is-active', i === el._idx); });
}
function buildWheel(el, labels, index, onPick) {
el.innerHTML = labels.map(function (label) { return '<div class="wheel-item">' + label + '</div>'; }).join('');
el._onPick = onPick;
el._idx = clamp(index, 0, labels.length - 1);
el.scrollTop = el._idx * ITEM;
paintWheel(el);
}
function setWheel(el, index) {
el._idx = clamp(index, 0, el.children.length - 1);
el.scrollTop = el._idx * ITEM;
paintWheel(el);
}
function attachWheelScroll(el) {
el.addEventListener('scroll', function () {
if (el._raf) return;
el._raf = requestAnimationFrame(function () {
el._raf = null;
if (!el.children.length) return;
var idx = clamp(Math.round(el.scrollTop / ITEM), 0, el.children.length - 1);
if (idx !== el._idx) {
el._idx = idx;
paintWheel(el);
if (el._onPick) el._onPick(idx);
}
});
}, { passive: true });
}
[yearWheel, monthWheel, dayWheel, hourWheel, minuteWheel].forEach(attachWheelScroll);
function buildMonthWheel() {
var labels = [];
for (var m = 1; m <= 12; m++) labels.push(m + '月');
buildWheel(monthWheel, labels, dateDraft.m - 1, function (i) {
dateDraft.m = i + 1;
buildDayWheel();
});
}
function buildDayWheel() {
var max = daysInMonth(dateDraft.y, dateDraft.m);
if (dateDraft.d > max) dateDraft.d = max;
var labels = [];
for (var d = 1; d <= max; d++) labels.push(d + '日');
buildWheel(dayWheel, labels, dateDraft.d - 1, function (i) { dateDraft.d = i + 1; });
}
function openDateSheet() {
var p = state.date.split('-');
dateDraft = { y: Number(p[0]), m: Number(p[1]), d: Number(p[2]) };
var yi = Math.max(0, YEARS.indexOf(dateDraft.y));
buildWheel(yearWheel, YEARS.map(function (y) { return y + '年'; }), yi, function (i) {
dateDraft.y = YEARS[i];
buildDayWheel();
});
buildMonthWheel();
buildDayWheel();
openLayer(dateLayer);
requestAnimationFrame(function () {
setWheel(yearWheel, yi);
setWheel(monthWheel, dateDraft.m - 1);
setWheel(dayWheel, dateDraft.d - 1);
});
}
dateDone.addEventListener('click', function () {
var yi = clamp(Math.round(yearWheel.scrollTop / ITEM), 0, YEARS.length - 1);
var mi = clamp(Math.round(monthWheel.scrollTop / ITEM), 0, 11);
var max = daysInMonth(YEARS[yi], mi + 1);
var di = clamp(Math.round(dayWheel.scrollTop / ITEM), 0, max - 1);
state.date = YEARS[yi] + '-' + pad2(mi + 1) + '-' + pad2(di + 1);
closeLayer(dateLayer);
renderChrome();
});
function openTimeSheet() {
var p = state.time.split(':');
var h = Number(p[0]), mi = Number(p[1]);
var hours = [], minutes = [];
for (var i = 0; i < 24; i++) hours.push(pad2(i) + ' 时');
for (var j = 0; j < 60; j++) minutes.push(pad2(j) + ' 分');
buildWheel(hourWheel, hours, h, function (i) { h = i; });
buildWheel(minuteWheel, minutes, mi, function (i) { mi = i; });
openLayer(timeLayer);
requestAnimationFrame(function () {
setWheel(hourWheel, h);
setWheel(minuteWheel, mi);
});
}
timeDone.addEventListener('click', function () {
var h = clamp(Math.round(hourWheel.scrollTop / ITEM), 0, 23);
var mi = clamp(Math.round(minuteWheel.scrollTop / ITEM), 0, 59);
state.time = pad2(h) + ':' + pad2(mi);
closeLayer(timeLayer);
renderChrome();
});
function validateForm() {
errors.title = !state.title.trim();
errors.content = !state.content.trim();
if (errors.title || errors.content) {
renderChrome();
if (errors.title) { showToast('请输入通知标题', 'warn'); titleInput.focus({ preventScroll: true }); }
else { showToast('请输入通知内容', 'warn'); contentInput.focus({ preventScroll: true }); }
return false;
}
if (!state.classIds.length) {
showToast('请选择接收班级', 'warn');
openClassSheet();
return false;
}
if (state.scheduled && (!state.date || !state.time)) {
showToast('请选择发送时间', 'warn');
openDateSheet();
return false;
}
return true;
}
function openConfirm() {
var sel = selectedClasses();
var names = sel.length ? sel.map(function (c) { return c.name; }).join('、') : '所选班级';
dlgDesc.textContent = '发布后' + names + '全体家长会收到微信提醒。';
dlgCat.textContent = catLabel(state.category);
dlgWhen.textContent = state.scheduled ? '定时 · ' + shortDate(state.date) + ' ' + state.time : '立即发送';
openLayer(confirmLayer);
setTimeout(function () { confirmPublish.focus(); }, 60);
}
publishBtn.addEventListener('click', function () {
if (publishing || published || state.view !== 'normal') return;
if (!validateForm()) return;
openConfirm();
});
confirmPublish.addEventListener('click', function () {
if (publishing) return;
publishing = true;
confirmPublish.disabled = true;
dialogCancel.disabled = true;
closeLayer(confirmLayer);
renderChrome();
setTimeout(function () {
publishing = false;
published = true;
confirmPublish.disabled = false;
dialogCancel.disabled = false;
confirmLabel.textContent = '确认发布';
/* 源码 toast 后 600ms navigateBack；单页原型留页复位 */
resetDraft();
renderAll();
showToast('已发布，返回老师工作台');
window.scrollTo({ top: 0, behavior: 'smooth' });
}, 900);
});
/* 状态切换由 ../../_shared/shell.js 负责：它把 chip 的状态写到 .mp-app[data-state]
   并派发 mp:state，这里只把同一个状态接到页面自己的渲染逻辑上。 */
document.addEventListener('mp:state', function (e) {
var next = e.detail && e.detail.state;
if (!next) return;
/* 发布成功留页复位后，点当前芯片也要能回到该演示状态，否则表单停在空白 */
if (next === state.view && !published) renderChrome();
else setView(next);
});
renderTplSheet();
renderAll();
})();
