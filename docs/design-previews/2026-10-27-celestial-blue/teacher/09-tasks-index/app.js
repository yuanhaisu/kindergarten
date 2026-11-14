(function () {
'use strict';
var app = document.querySelector('.mp-app');
var main = document.querySelector('.mp-main');
var taskList = document.getElementById('taskList');
var filters = Array.prototype.slice.call(document.querySelectorAll('.filter'));
var taskTotal = document.getElementById('taskTotal');
var pendingTotal = document.getElementById('pendingTotal');
var toastEl = document.querySelector('.toast');
var backBtn = document.querySelector('.mp-back');
var chips = Array.prototype.slice.call(document.querySelectorAll('.mp-demo-chip[data-mp-state]'));
var studentsChip = document.querySelector('[data-demo-students]');
var jarChip = document.querySelector('[data-demo-jar]');
var retryBtn = document.getElementById('retryBtn');
var jarCard = document.getElementById('jarCard');
var jarStars = document.getElementById('jarStars');
var jarGoal = document.getElementById('jarGoal');
var jarPercent = document.getElementById('jarPercent');
var jarFill = document.getElementById('jarFill');
var jarOver = document.getElementById('jarOver');
var jarTrack = jarFill.parentNode;
var jarForm = document.getElementById('jarForm');
var jarGoalInput = document.getElementById('jarGoalInput');
var jarSave = document.getElementById('jarSave');
var classGoalInline = document.getElementById('classGoalInline');
var layer = document.getElementById('composeLayer');
var sheet = document.getElementById('composeSheet');
var sheetBody = sheet.querySelector('.sheet-body');
var form = document.getElementById('composeForm');
var titleInput = document.getElementById('taskTitle');
var descInput = document.getElementById('taskDesc');
var linkInput = document.getElementById('taskLink');
var startDate = document.getElementById('startDate');
var endDate = document.getElementById('endDate');
var startLabel = document.getElementById('startLabel');
var endLabel = document.getElementById('endLabel');
var starGoalInput = document.getElementById('starGoal');
var starGoalField = document.getElementById('starGoalField');
var starGoalHint = document.getElementById('starGoalHint');
var publicField = document.getElementById('publicField');
var allowPublic = document.getElementById('allowPublic');
var publicHint = document.getElementById('publicHint');
var targetHint = document.getElementById('targetHint');
var scopeTrigger = document.getElementById('scopeTrigger');
var scopeValue = document.getElementById('scopeValue');
var scopeSub = document.getElementById('scopeSub');
var scopeMenu = document.getElementById('scopeMenu');
var studentsErrorHint = document.getElementById('studentsErrorHint');
var studentsRetry = document.getElementById('studentsRetry');
var classScopeHint = document.getElementById('classScopeHint');
var scopeDisabledHint = document.getElementById('scopeDisabledHint');
var scopeClass = document.getElementById('scopeClass');
var createBtn = document.getElementById('createBtn');
var actionCreate = document.querySelector('.action-create');
var openButtons = Array.prototype.slice.call(document.querySelectorAll('[data-open-compose]'));
var closeButtons = Array.prototype.slice.call(document.querySelectorAll('[data-close-compose]'));
var freqButtons = Array.prototype.slice.call(document.querySelectorAll('[data-frequency]'));
var ritualButtons = Array.prototype.slice.call(document.querySelectorAll('[data-ritual]'));
var starScopeButtons = Array.prototype.slice.call(document.querySelectorAll('[data-star-scope]'));
var DAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
var DEFAULT_START = '2026-10-26';
var DEFAULT_END = '2026-11-02';
var CLASS_SIZE = 30;
// 演示名单：对应 GET /teacher/class/students 一次全量返回（无分页），数量与 CLASS_SIZE 口径一致。
var STUDENTS = ['乐乐', '小雨', '朵朵', '果果', '豆豆', '米米', '安安', '晨晨', '航航', '琪琪', '然然', '糖糖', '一诺', '子涵', '浩然', '欣怡', '梓萱', '宇轩', '思远', '若溪', '嘉禾', '可心', '天佑', '雨桐', '沐沐', '辰辰', '诗涵', '俊熙', '小北', '团团'];
var studentsError = false;
var studentsRetryTimer = null;
var FREQ = { once: '一次性', daily: '每日', weekly: '每周' };
var RITUAL = { sticker: '贴纸', voice: '语音', class_star: '班级星星' };
var jar = { stars: 46, goal: 60 };
var compose = {
target: 0,
targetName: '班级',
frequency: 'once',
ritual: 'sticker',
starScope: 'task',
allowPublic: false
};
var dateState = { start: DEFAULT_START, end: DEFAULT_END };
var currentFilter = 'all';
var lastFocus = null;
var toastTimer = null;
var retryTimer = null;
function make(tag, className, text) {
var node = document.createElement(tag);
if (className) node.className = className;
if (typeof text === 'string') node.textContent = text;
return node;
}
function setPressed(buttons, match) {
buttons.forEach(function (button) {
var on = match(button);
button.classList.toggle('is-on', on);
button.setAttribute('aria-pressed', String(on));
});
}
function showToast(message) {
toastEl.textContent = message;
toastEl.classList.add('is-on');
window.clearTimeout(toastTimer);
toastTimer = window.setTimeout(function () {
toastEl.classList.remove('is-on');
}, 2200);
}
function starLabel(scope, count, goal) {
if (count === null || typeof count === 'undefined') return '';
if (scope === 'class') return '班级星星罐 · 本任务已集 ' + count + ' 星';
if (!(goal > 0)) return '';
if (count > goal) return '星星已超额 · ' + count + '/' + goal;
if (count === goal) return '星星已集满 · ' + count + '/' + goal;
return '★ ' + count + '/' + goal;
}
function allCards() {
return Array.prototype.slice.call(taskList.querySelectorAll('.task-card'));
}
function syncCounts() {
var cards = allCards();
var active = 0;
var ended = 0;
var pending = 0;
cards.forEach(function (card) {
if (card.getAttribute('data-status') === '1') active += 1;
else ended += 1;
pending += Number(card.getAttribute('data-pending')) || 0;
});
setCount('all', cards.length);
setCount('1', active);
setCount('0', ended);
taskTotal.textContent = '共 ' + cards.length + ' 个任务';
pendingTotal.textContent = String(pending);
}
function setCount(key, value) {
var node = document.querySelector('.filter-count[data-count="' + key + '"]');
if (node) node.textContent = String(value);
}
function setFilter(value) {
currentFilter = value;
filters.forEach(function (button) {
var on = button.getAttribute('data-filter') === value;
button.classList.toggle('is-active', on);
button.setAttribute('aria-pressed', String(on));
});
allCards().forEach(function (card) {
card.hidden = value !== 'all' && card.getAttribute('data-status') !== value;
});
}
function renderScopeOptions() {
scopeMenu.textContent = '';
var options = [{ index: 0, name: '班级', sub: '全班 ' + STUDENTS.length + ' 人' }];
if (!studentsError) {
STUDENTS.forEach(function (name, i) {
options.push({ index: i + 1, name: name, sub: '' });
});
}
options.forEach(function (item) {
var option = make('button', 'scope-option');
option.type = 'button';
option.setAttribute('role', 'option');
option.setAttribute('data-target', String(item.index));
option.setAttribute('data-name', item.name);
option.setAttribute('aria-selected', String(item.index === compose.target));
option.tabIndex = item.index === compose.target ? 0 : -1;
var main = make('span', 'scope-option-main');
main.appendChild(make('span', 'scope-option-label', item.name));
if (item.sub) main.appendChild(make('span', 'scope-option-sub', item.sub));
option.appendChild(main);
var check = make('span', 'scope-option-check', '✓');
check.setAttribute('aria-hidden', 'true');
option.appendChild(check);
scopeMenu.appendChild(option);
});
}
function syncScopeUI() {
scopeValue.textContent = compose.target > 0 ? compose.targetName : '班级';
scopeSub.textContent = compose.target > 0 ? '个体任务 · 仅该家长可见' : (studentsError ? '学生名单未加载' : '全班 ' + STUDENTS.length + ' 人');
Array.prototype.slice.call(scopeMenu.querySelectorAll('.scope-option')).forEach(function (option) {
var on = Number(option.getAttribute('data-target')) === compose.target;
option.setAttribute('aria-selected', String(on));
option.tabIndex = on ? 0 : -1;
});
}
function openScopeMenu() {
scopeMenu.hidden = false;
scopeTrigger.setAttribute('aria-expanded', 'true');
var selected = scopeMenu.querySelector('.scope-option[aria-selected="true"]') || scopeMenu.querySelector('.scope-option');
if (selected) selected.focus();
}
function closeScopeMenu(restoreFocus) {
if (scopeMenu.hidden) return;
scopeMenu.hidden = true;
scopeTrigger.setAttribute('aria-expanded', 'false');
if (restoreFocus) scopeTrigger.focus();
}
function setStudentsError(next) {
studentsError = next;
window.clearTimeout(studentsRetryTimer);
studentsRetryTimer = null;
studentsRetry.disabled = false;
studentsRetry.textContent = '点击重试';
studentsChip.setAttribute('aria-pressed', String(next));
studentsErrorHint.hidden = !next;
if (next && compose.target > 0) {
compose.target = 0;
compose.targetName = '班级';
}
closeScopeMenu(false);
renderScopeOptions();
refreshCompose();
}
function setDemoState(next) {
if (retryTimer) {
window.clearTimeout(retryTimer);
retryTimer = null;
}
app.setAttribute('data-state', next);
chips.forEach(function (chip) {
chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === next));
});
if (main) main.scrollTop = 0;
if (next === 'normal') setFilter(currentFilter);
retryBtn.disabled = false;
retryBtn.textContent = '重试';
}
function closeDemo() {
if (window.OdShell && typeof window.OdShell.closeDemo === 'function') {
window.OdShell.closeDemo();
}
}
function openResponse(card) {
var title = card.getAttribute('data-title');
if (!title) {
var titleNode = card.querySelector('.task-title');
title = titleNode ? titleNode.textContent : '任务';
}
showToast('正在打开「' + title + '」回应页');
}
function fmtDate(iso) {
var parts = iso.split('-');
var month = Number(parts[1]);
var day = Number(parts[2]);
var date = new Date(Number(parts[0]), month - 1, day);
return month + '月' + day + '日 ' + DAYS[date.getDay()];
}
function updateDateLabels() {
startLabel.textContent = fmtDate(dateState.start);
endLabel.textContent = fmtDate(dateState.end);
}
function handleDateChange() {
var start = startDate.value;
var end = endDate.value;
if (!start) {
start = dateState.start;
startDate.value = start;
showToast('请选择开始日期');
}
if (!end) {
end = dateState.end;
endDate.value = end;
showToast('请选择结束日期');
}
if (end < start) {
end = start;
endDate.value = end;
showToast('结束日期不能早于开始日期');
}
dateState.start = start;
dateState.end = end;
endDate.min = start;
updateDateLabels();
}
function renderJar() {
var configured = jar.goal > 0;
var percent = configured ? Math.min(100, Math.round((jar.stars / jar.goal) * 100)) : 0;
jarStars.textContent = String(jar.stars);
jarGoal.textContent = configured ? String(jar.goal) : '—';
jarPercent.textContent = percent + '%';
jarFill.style.width = percent + '%';
jarTrack.setAttribute('aria-valuenow', String(percent));
jarTrack.setAttribute('aria-valuetext', configured ? '已集 ' + jar.stars + ' / ' + jar.goal : '未设置学期目标');
jarOver.hidden = !(configured && jar.stars > jar.goal);
jarGoalInput.value = configured ? String(jar.goal) : '';
jarSave.textContent = configured ? '更新目标' : '设置目标';
jarChip.setAttribute('aria-pressed', String(!configured));
classGoalInline.textContent = configured ? String(jar.goal) : '—';
jarCard.classList.toggle('is-unset', !configured);
scopeClass.disabled = !configured;
scopeClass.setAttribute('aria-disabled', String(!configured));
scopeDisabledHint.hidden = configured;
}
function syncCreateBtn() {
var ready = titleInput.value.trim() !== '' && linkInput.value.trim() !== '';
createBtn.disabled = !ready;
createBtn.setAttribute('aria-disabled', String(!ready));
}
function refreshCompose() {
var jarReady = jar.goal > 0;
var individual = compose.target > 0;
if (!jarReady) compose.starScope = 'task';
if (individual) {
compose.allowPublic = false;
allowPublic.checked = false;
}
targetHint.hidden = !individual;
publicField.hidden = individual;
publicHint.hidden = individual || !compose.allowPublic;
scopeClass.disabled = !jarReady;
scopeClass.setAttribute('aria-disabled', String(!jarReady));
scopeDisabledHint.hidden = jarReady;
classScopeHint.hidden = compose.starScope !== 'class';
starGoalField.hidden = compose.starScope === 'class';
if (compose.starScope === 'class') {
starGoalHint.hidden = true;
starGoalInput.removeAttribute('aria-invalid');
}
var needsGoal = compose.ritual === 'class_star' && compose.starScope === 'task';
starGoalInput.setAttribute('aria-required', String(needsGoal));
if (!needsGoal) {
starGoalHint.hidden = true;
starGoalInput.removeAttribute('aria-invalid');
}
syncScopeUI();
setPressed(freqButtons, function (button) {
return button.getAttribute('data-frequency') === compose.frequency;
});
setPressed(ritualButtons, function (button) {
return button.getAttribute('data-ritual') === compose.ritual;
});
setPressed(starScopeButtons, function (button) {
return button.getAttribute('data-star-scope') === compose.starScope;
});
syncCreateBtn();
}
function resetCompose() {
titleInput.value = '';
descInput.value = '';
linkInput.value = '';
starGoalInput.value = '';
allowPublic.checked = false;
compose.target = 0;
compose.targetName = '班级';
compose.frequency = 'once';
compose.ritual = 'sticker';
compose.starScope = 'task';
compose.allowPublic = false;
dateState.start = DEFAULT_START;
dateState.end = DEFAULT_END;
startDate.value = DEFAULT_START;
endDate.value = DEFAULT_END;
endDate.min = DEFAULT_START;
starGoalHint.hidden = true;
starGoalHint.textContent = '';
starGoalInput.removeAttribute('aria-invalid');
updateDateLabels();
closeScopeMenu(false);
refreshCompose();
}
function openCompose(opener) {
if (!layer.hidden) return;
lastFocus = opener || document.activeElement;
resetCompose();
layer.hidden = false;
layer.setAttribute('aria-hidden', 'false');
document.body.classList.add('is-locked');
window.setTimeout(function () {
sheetBody.scrollTop = 0;
titleInput.focus();
}, 40);
}
function closeCompose(restoreFocus) {
if (layer.hidden) return;
layer.hidden = true;
layer.setAttribute('aria-hidden', 'true');
document.body.classList.remove('is-locked');
resetCompose();
if (restoreFocus !== false) {
var target = (lastFocus && document.documentElement.contains(lastFocus) && lastFocus.offsetParent !== null) ? lastFocus : actionCreate;
if (target && target.offsetParent !== null) target.focus();
}
}
function buildTaskCard(task) {
var card = make('article', 'task-card');
card.setAttribute('role', 'button');
card.tabIndex = 0;
card.setAttribute('data-status', '1');
card.setAttribute('data-pending', '0');
card.setAttribute('data-title', task.title);
card.setAttribute('data-od-id', 'task-card-new');
var head = make('div', 'task-head');
head.appendChild(make('h2', 'task-title', task.title));
var pill = make('span', 'status-pill is-live');
var dot = make('span', 'status-dot');
dot.setAttribute('aria-hidden', 'true');
pill.appendChild(dot);
pill.appendChild(document.createTextNode('进行中'));
head.appendChild(pill);
card.appendChild(head);
var meta = make('p', 'task-meta');
var firstDot = make('span', 'meta-dot', '·');
var secondDot = make('span', 'meta-dot', '·');
firstDot.setAttribute('aria-hidden', 'true');
secondDot.setAttribute('aria-hidden', 'true');
meta.appendChild(document.createTextNode(task.frequency));
meta.appendChild(firstDot);
meta.appendChild(document.createTextNode(task.ritual));
meta.appendChild(secondDot);
meta.appendChild(document.createTextNode(task.range));
if (task.individual) meta.appendChild(make('span', 'scope-tag', '个体任务'));
card.appendChild(meta);
if (task.desc) card.appendChild(make('p', 'task-desc', task.desc));
var stats = make('div', 'task-stats' + (task.individual ? ' is-single' : ''));
if (task.individual) {
stats.innerHTML = '<div class="progress-head"><span class="progress-label">已提交 <b>0</b> 条</span></div>';
} else {
stats.innerHTML = '<div class="progress-head"><span class="progress-label">已参与 <b>0</b>/' + CLASS_SIZE + '</span><span class="progress-pct">0%</span></div><div class="progress-track" role="progressbar" aria-valuemin="0" aria-valuemax="' + CLASS_SIZE + '" aria-valuenow="0" aria-valuetext="已参与 0/' + CLASS_SIZE + '"><span class="progress-fill" style="width:0%"></span></div>';
}
card.appendChild(stats);
if (task.star) card.appendChild(make('p', 'task-star', task.star));
var foot = make('div', 'task-foot');
foot.appendChild(make('span', 'pending-badge is-zero', '待回应 0'));
var link = make('span', 'task-link');
link.innerHTML = '查看回应<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 3.5 4.5 4.5L6 12.5"/></svg>';
foot.appendChild(link);
card.appendChild(foot);
return card;
}
function createTask() {
var title = titleInput.value.trim();
var desc = descInput.value.trim();
var link = linkInput.value.trim();
if (!title) {
showToast('请填写任务标题');
titleInput.focus();
return;
}
if (!link) {
showToast('请填写和幼儿园学习的关系');
linkInput.focus();
return;
}
var goal = Number(starGoalInput.value.trim() || 0);
var classScope = compose.starScope === 'class';
if (!classScope && compose.ritual === 'class_star' && !(goal > 0)) {
starGoalHint.textContent = '选择班级星星仪式时，请填写大于 0 的目标';
starGoalHint.hidden = false;
starGoalInput.setAttribute('aria-invalid', 'true');
showToast('请填写大于 0 的星星目标');
starGoalInput.focus();
return;
}
if (dateState.end < dateState.start) {
showToast('结束日期不能早于开始日期');
return;
}
var individual = compose.target > 0;
var displayTitle = individual ? compose.targetName + ' · ' + title : title;
var count = goal > 0 || classScope ? 0 : null;
var task = {
title: displayTitle,
desc: desc,
frequency: FREQ[compose.frequency],
ritual: RITUAL[compose.ritual],
range: fmtDate(dateState.start) + '–' + fmtDate(dateState.end),
individual: individual,
star: starLabel(classScope ? 'class' : 'task', count, goal)
};
taskList.insertBefore(buildTaskCard(task), taskList.firstChild);
setFilter('all');
syncCounts();
if (app.getAttribute('data-state') !== 'normal') setDemoState('normal');
closeCompose(false);
if (main) main.scrollTop = 0;
if (actionCreate && actionCreate.offsetParent !== null) actionCreate.focus();
showToast('已创建');
}
openButtons.forEach(function (button) {
button.addEventListener('click', function () {
openCompose(button);
});
});
closeButtons.forEach(function (button) {
button.addEventListener('click', function () {
closeCompose(true);
});
});
scopeTrigger.addEventListener('click', function () {
if (scopeMenu.hidden) openScopeMenu();
else closeScopeMenu(false);
});
scopeMenu.addEventListener('click', function (event) {
var option = event.target.closest ? event.target.closest('.scope-option') : null;
if (!option || !scopeMenu.contains(option)) return;
compose.target = Number(option.getAttribute('data-target'));
compose.targetName = option.getAttribute('data-name') || '班级';
closeScopeMenu(true);
refreshCompose();
});
scopeMenu.addEventListener('keydown', function (event) {
var options = Array.prototype.slice.call(scopeMenu.querySelectorAll('.scope-option'));
if (!options.length) return;
var index = options.indexOf(document.activeElement);
if (event.key === 'ArrowDown') {
event.preventDefault();
options[Math.min(options.length - 1, index + 1)].focus();
} else if (event.key === 'ArrowUp') {
event.preventDefault();
options[Math.max(0, index - 1)].focus();
} else if (event.key === 'Home') {
event.preventDefault();
options[0].focus();
} else if (event.key === 'End') {
event.preventDefault();
options[options.length - 1].focus();
}
});
sheetBody.addEventListener('click', function (event) {
if (scopeMenu.hidden) return;
if (scopeMenu.contains(event.target) || scopeTrigger.contains(event.target)) return;
closeScopeMenu(false);
});
studentsRetry.addEventListener('click', function () {
if (studentsRetry.disabled) return;
studentsRetry.disabled = true;
studentsRetry.textContent = '重试中…';
studentsRetryTimer = window.setTimeout(function () {
studentsRetryTimer = null;
setStudentsError(false);
showToast('学生名单已恢复');
}, 600);
});
studentsChip.addEventListener('click', function () {
setStudentsError(!studentsError);
showToast(studentsError ? '已模拟学生名单加载失败' : '学生名单已恢复');
});
jarChip.addEventListener('click', function () {
jar.goal = jar.goal > 0 ? 0 : 60;
renderJar();
refreshCompose();
showToast(jar.goal > 0 ? '已恢复学期目标' : '已切换为「未配置」');
});
freqButtons.forEach(function (button) {
button.addEventListener('click', function () {
compose.frequency = button.getAttribute('data-frequency');
refreshCompose();
});
});
ritualButtons.forEach(function (button) {
button.addEventListener('click', function () {
compose.ritual = button.getAttribute('data-ritual');
refreshCompose();
});
});
starScopeButtons.forEach(function (button) {
button.addEventListener('click', function () {
if (button.disabled) return;
compose.starScope = button.getAttribute('data-star-scope');
refreshCompose();
});
});
allowPublic.addEventListener('change', function () {
compose.allowPublic = allowPublic.checked;
refreshCompose();
});
titleInput.addEventListener('input', syncCreateBtn);
linkInput.addEventListener('input', syncCreateBtn);
starGoalInput.addEventListener('input', function () {
starGoalInput.value = starGoalInput.value.replace(/[^\d]/g, '').slice(0, 4);
starGoalInput.removeAttribute('aria-invalid');
starGoalHint.hidden = true;
});
startDate.addEventListener('change', handleDateChange);
endDate.addEventListener('change', handleDateChange);
form.addEventListener('submit', function (event) {
event.preventDefault();
createTask();
});
jarGoalInput.addEventListener('input', function () {
jarGoalInput.value = jarGoalInput.value.replace(/[^\d]/g, '').slice(0, 4);
jarGoalInput.removeAttribute('aria-invalid');
});
jarForm.addEventListener('submit', function (event) {
event.preventDefault();
var raw = jarGoalInput.value.trim();
var goal = Number(raw);
if (!/^\d+$/.test(raw) || !(goal > 0)) {
jarGoalInput.setAttribute('aria-invalid', 'true');
jarGoalInput.focus();
showToast('目标需为大于 0 的整数');
return;
}
jarGoalInput.removeAttribute('aria-invalid');
jar.goal = goal;
renderJar();
refreshCompose();
showToast('已保存');
});
filters.forEach(function (button) {
button.addEventListener('click', function () {
setFilter(button.getAttribute('data-filter'));
});
});
taskList.addEventListener('click', function (event) {
var card = event.target.closest ? event.target.closest('.task-card') : null;
if (card && taskList.contains(card)) openResponse(card);
});
taskList.addEventListener('keydown', function (event) {
if (event.key !== 'Enter' && event.key !== ' ' && event.key !== 'Spacebar') return;
if (!event.target.classList || !event.target.classList.contains('task-card')) return;
event.preventDefault();
openResponse(event.target);
});
if (backBtn) {
backBtn.addEventListener('click', function () {
showToast('已返回老师工作台');
});
}
chips.forEach(function (chip) {
chip.addEventListener('click', function () {
setDemoState(chip.getAttribute('data-mp-state'));
closeDemo();
});
});
if (retryBtn) {
retryBtn.addEventListener('click', function () {
setDemoState('loading');
retryBtn.disabled = true;
retryBtn.textContent = '重试中…';
retryTimer = window.setTimeout(function () {
retryTimer = null;
setDemoState('normal');
retryBtn.disabled = false;
retryBtn.textContent = '重试';
}, 600);
});
}
document.addEventListener('keydown', function (event) {
if (layer.hidden) return;
if (event.key === 'Escape') {
event.preventDefault();
if (!scopeMenu.hidden) {
closeScopeMenu(true);
return;
}
closeCompose(true);
return;
}
if (event.key !== 'Tab') return;
var focusable = Array.prototype.slice.call(sheet.querySelectorAll('button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])')).filter(function (node) {
return node.offsetParent !== null;
});
if (!focusable.length) return;
var first = focusable[0];
var last = focusable[focusable.length - 1];
var active = document.activeElement;
if (!sheet.contains(active)) {
event.preventDefault();
first.focus();
return;
}
if (event.shiftKey && active === first) {
event.preventDefault();
last.focus();
} else if (!event.shiftKey && active === last) {
event.preventDefault();
first.focus();
}
});
renderScopeOptions();
renderJar();
resetCompose();
syncCounts();
setFilter('all');
chips.forEach(function (chip) {
chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === 'normal'));
});
})();
