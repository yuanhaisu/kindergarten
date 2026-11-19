(function () {
  var toast = document.getElementById('toast');
  var hideTimer = null;
  var swapTimer = null;

  function showToast(message) {
    if (!toast || !message) return;
    window.clearTimeout(hideTimer);
    toast.textContent = message;
    toast.classList.add('is-on');
    hideTimer = window.setTimeout(function () {
      toast.classList.remove('is-on');
    }, 2000);
  }

  function runAction(el) {
    var message = el.getAttribute('data-toast');
    var done = el.getAttribute('data-toast-done');
    var delay = parseInt(el.getAttribute('data-toast-delay'), 10);

    window.clearTimeout(swapTimer);
    showToast(message);

    if (done) {
      swapTimer = window.setTimeout(function () {
        showToast(done);
      }, isNaN(delay) ? 1200 : delay);
    }
  }

  var triggers = document.querySelectorAll('[data-toast]');
  Array.prototype.forEach.call(triggers, function (el) {
    el.addEventListener('click', function () {
      runAction(el);
    });
  });

  var reports = {
    '2026-fall': {
      overline: '2026 秋季学期 · 乐乐 · 小一班',
      title: '本学期成长报告',
      introTitle: '乐乐，正在长成他自己',
      intro: '这个学期他学会了自己穿外套、系鞋带，也开始在集体里主动说话。比起「学会了什么」，我们更高兴看到他<span class="hl">愿意试一试</span>。',
      stats: { attendance: '48', attendanceDenom: '/50', points: '+218', tasks: '10', tasksDenom: '/12', records: '5' },
      dims: {
        language: ['88', '能用完整的句子讲清一件事，最近总爱追问「为什么」。'],
        social: ['82', '开始主动邀请同伴一起玩，遇到争执会先说自己的想法。'],
        motor: ['76', '跑跳稳了，攀爬时还是会在最高一格停下来看看。'],
        selfcare: ['90', '自己穿外套、系鞋带，午睡后会把被子叠好。']
      },
      quote: '乐乐，你系鞋带的时候特别认真。老师希望你知道，<span class="hl hl--marigold">慢一点没关系</span>，愿意再试一次就很了不起。',
      sign: '王老师 · 11 月 12 日',
      highlightNote: '从本学期的成长记录中选出的 3 张照片'
    },
    '2026-spring': {
      overline: '2026 春季学期 · 乐乐 · 小一班',
      title: '2026 春季学期成长报告',
      introTitle: '乐乐，开始主动走向同伴',
      intro: '这个学期他开始主动邀请小朋友一起玩，也愿意在集体面前讲自己的发现。比起学会多少，我们更高兴看到他<span class="hl">愿意表达</span>。',
      stats: { attendance: '46', attendanceDenom: '/50', points: '+186', tasks: '9', tasksDenom: '/12', records: '6' },
      dims: {
        language: ['83', '能讲清一件完整的事，开始使用「因为…所以…」。'],
        social: ['77', '愿意主动加入同伴的游戏，遇到分歧会试着商量。'],
        motor: ['71', '跑跳更稳，攀爬时愿意多试一格。'],
        selfcare: ['85', '能自己穿外套、整理书包，午睡后收好被子。']
      },
      quote: '乐乐，你开始愿意把自己的想法讲给同伴听。老师希望你知道，<span class="hl hl--marigold">慢一点没关系</span>，愿意说出来就很了不起。',
      sign: '李老师 · 2026 年 6 月 26 日',
      highlightNote: '从 2026 春季学期的成长记录中选出的 3 张照片'
    },
    '2025-fall': {
      overline: '2025 秋季学期 · 乐乐 · 小一班',
      title: '2025 秋季学期成长报告',
      introTitle: '乐乐，正在熟悉这里',
      intro: '刚入园时他更习惯先看一会儿，再慢慢加入。这个学期他学会了认识自己的杯子、书包和座位，也开始<span class="hl">试着自己来</span>。',
      stats: { attendance: '42', attendanceDenom: '/50', points: '+150', tasks: '8', tasksDenom: '/12', records: '7' },
      dims: {
        language: ['77', '愿意用短句表达需要，开始复述老师讲过的故事。'],
        social: ['72', '多数时候先观察同伴，熟悉后会跟着一起玩。'],
        motor: ['66', '走跑平稳，攀爬时会在低处停一停。'],
        selfcare: ['80', '能自己拿杯子喝水，开始试着自己穿鞋。']
      },
      quote: '乐乐，你刚来时总是先看一会儿。老师很高兴看到你慢慢愿意伸出手，加入大家。',
      sign: '王老师 · 2026 年 1 月 16 日',
      highlightNote: '从 2025 秋季学期的成长记录中选出的 3 张照片'
    }
  };

  var tabs = Array.prototype.slice.call(document.querySelectorAll('[data-semester]'));
  var overline = document.getElementById('report-overline');
  var title = document.getElementById('report-title');
  var introTitle = document.getElementById('intro-title');
  var introCopy = document.getElementById('intro-copy');
  var highlightNote = document.getElementById('highlight-note');
  var quote = document.getElementById('teacher-quote');
  var sign = document.getElementById('teacher-sign');
  var statNodes = {
    attendance: document.getElementById('stat-attendance'),
    attendanceDenom: document.getElementById('stat-attendance-denom'),
    points: document.getElementById('stat-points'),
    tasks: document.getElementById('stat-tasks'),
    tasksDenom: document.getElementById('stat-tasks-denom'),
    records: document.getElementById('stat-records')
  };

  function renderReport(key) {
    var report = reports[key];
    if (!report || !overline || !title) return;
    overline.textContent = report.overline;
    title.textContent = report.title;
    document.title = report.title + ' · 成长时光';
    if (introTitle) introTitle.textContent = report.introTitle;
    if (introCopy) introCopy.innerHTML = report.intro;
    if (highlightNote) highlightNote.textContent = report.highlightNote;
    if (quote) quote.innerHTML = report.quote;
    if (sign) sign.textContent = report.sign;
    Object.keys(statNodes).forEach(function (name) {
      if (statNodes[name]) statNodes[name].textContent = report.stats[name];
    });
    Object.keys(report.dims).forEach(function (name) {
      var article = document.querySelector('[data-dimension="' + name + '"]');
      if (!article) return;
      var score = article.querySelector('.dimension__score');
      var fill = article.querySelector('.bar__fill');
      var desc = article.querySelector('.dimension__desc');
      var bar = article.querySelector('.bar');
      var data = report.dims[name];
      if (score) score.textContent = data[0];
      if (fill) fill.style.width = data[0] + '%';
      if (desc) desc.textContent = data[1];
      if (bar) bar.setAttribute('aria-valuenow', data[0]);
    });
    tabs.forEach(function (tab) {
      var active = tab.getAttribute('data-semester') === key;
      tab.classList.toggle('is-active', active);
      tab.setAttribute('aria-selected', String(active));
    });
  }

  tabs.forEach(function (tab) {
    tab.addEventListener('click', function () {
      renderReport(tab.getAttribute('data-semester'));
    });
  });

  var requested = new URLSearchParams(window.location.search).get('semester');
  if (requested && reports[requested]) renderReport(requested);
})();
