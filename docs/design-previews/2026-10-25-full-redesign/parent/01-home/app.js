/* 养成星球 · 家长首页交互
 * 只做页面自己的事：孩子切换、轻提示、失败重试。
 * 四态切换由 _shared/shell.js 写 data-state 并派发 mp:state。
 */
(function () {
  'use strict';

  var toast = document.getElementById('toast');
  var toastTimer = null;

  function showToast(message) {
    if (!toast || !message) return;
    toast.textContent = message;
    toast.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  /* 所有带 data-toast 的静态入口 --------------------------------------- */
  document.querySelectorAll('[data-toast]').forEach(function (el) {
    el.addEventListener('click', function () {
      showToast(el.getAttribute('data-toast'));
    });
  });

  /* 多孩切换 ------------------------------------------------------------ */
  var profiles = {
    lele: {
      name: '乐乐',
      className: '小一班',
      greeting: '妈妈 您好~',
      initial: '乐',
      attendance: '已入园',
      arrive: '07:52',
      meal: '很好',
      sleep: '很好',
      nap: '120分钟',
      mood: '很好',
      award: '今日获得「小小艺术家」奖章',
      taskName: '亲子阅读 15 分钟',
      taskDesc: '和孩子一起读《好饿的毛毛虫》，说说毛毛虫都吃了什么',
      taskBadge: '3'
    },
    xiaoyu: {
      name: '小雨',
      className: '大二班',
      greeting: '奶奶 您好~',
      initial: '雨',
      attendance: '已入园',
      arrive: '08:05',
      meal: '一般',
      sleep: '很好',
      nap: '120分钟',
      mood: '很好',
      award: '',
      taskName: '亲子运动 10 分钟',
      taskDesc: '和奶奶在小区里散步 10 分钟，数一数路边有几棵树',
      taskBadge: '1'
    }
  };

  var fieldMap = {
    name: 'child-name',
    className: 'class-pill',
    greeting: 'greeting',
    initial: 'avatar-initial',
    attendance: 'stat-attendance',
    arrive: 'stat-arrive',
    meal: 'stat-meal',
    sleep: 'stat-sleep',
    nap: 'stat-nap',
    mood: 'stat-mood',
    taskName: 'tonight-name',
    taskDesc: 'tonight-desc'
  };

  var currentKey = 'lele';

  function setText(id, value) {
    var node = document.getElementById(id);
    if (node) node.textContent = value;
  }

  function applyProfile(key) {
    var profile = profiles[key];
    if (!profile) return;
    currentKey = key;

    Object.keys(fieldMap).forEach(function (field) {
      setText(fieldMap[field], profile[field]);
    });

    setText('task-badge', profile.taskBadge);
    setText('task-badge-sr', '，' + profile.taskBadge + ' 个待办任务');

    var perfBar = document.getElementById('perf-bar');
    var perfText = document.getElementById('perf-text');
    if (perfBar) perfBar.hidden = !profile.award;
    if (perfText && profile.award) perfText.textContent = profile.award;
  }

  /* 空态下没有新通知，通知角标必须隐藏；任务角标按审查口径一并隐藏，
     避免「空态却仍有待办数字」的观感冲突。 */
  function applyBadgesForState(state) {
    var empty = state === 'empty';
    ['notice-badge', 'notice-badge-sr', 'task-badge', 'task-badge-sr'].forEach(function (id) {
      var node = document.getElementById(id);
      if (node) node.hidden = empty;
    });
  }

  var childChips = Array.prototype.slice.call(document.querySelectorAll('.child-pill'));

  childChips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      if (chip.getAttribute('aria-current') === 'true') return;
      var key = chip.getAttribute('data-child');
      if (!profiles[key]) return;

      applyProfile(key);
      childChips.forEach(function (other) {
        other.setAttribute('aria-current', other === chip ? 'true' : 'false');
      });
      showToast('已切换到 ' + profiles[key].name);
    });
  });

  /* 失败重试 ------------------------------------------------------------ */
  var retry = document.getElementById('retry-btn');
  var retryTimer = null;

  function resetRetry() {
    window.clearTimeout(retryTimer);
    retryTimer = null;
    if (!retry) return;
    retry.disabled = false;
    retry.textContent = '重试';
  }

  if (retry) {
    retry.addEventListener('click', function () {
      if (retry.disabled) return;
      retry.disabled = true;
      retry.textContent = '重试中…';
      window.clearTimeout(retryTimer);
      retryTimer = window.setTimeout(function () {
        retryTimer = null;
        retry.disabled = false;
        retry.textContent = '重试';
        showToast('已重新加载');
      }, 1000);
    });
  }

  /* 四态切换后的收尾 ---------------------------------------------------- */
  var home = document.querySelector('.home');
  document.addEventListener('mp:state', function (event) {
    var state = event && event.detail ? event.detail.state : '';
    if (home) home.setAttribute('aria-busy', state === 'loading' ? 'true' : 'false');
    applyBadgesForState(state);
    resetRetry();
  });
})();
