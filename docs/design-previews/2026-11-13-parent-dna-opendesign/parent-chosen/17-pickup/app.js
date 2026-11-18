(function () {
  'use strict';

  var params = new URLSearchParams(window.location.search);
  var requested = (params.get('state') || '').toLowerCase();

  /* 到园状态与「今天谁来接」是两件事：状态只认 in/away/done，
   * 其余 state 值留给评审用的指定抽屉分支。 */
  document.body.dataset.state = requested === 'away' || requested === 'done' ? requested : 'in';

  var designations = {
    picker: { designated: true, openSheet: true, emptySheet: false },
    undesignated: { designated: false, openSheet: false, emptySheet: false },
    'no-auth': { designated: false, openSheet: true, emptySheet: true }
  };
  var plan = designations[requested] || { designated: true, openSheet: false, emptySheet: false };

  var main = document.querySelector('.mp-main');
  var nav = document.querySelector('.mp-nav');
  var layer = document.getElementById('pickup-sheet-layer');
  var options = document.getElementById('pickup-person-options');
  var emptyState = document.getElementById('picker-empty');
  var nameEl = document.getElementById('designate-person');
  var relEl = document.getElementById('designate-rel');
  var nameBlock = document.getElementById('designate-name');
  var changeButton = document.getElementById('designate-change');
  var emptyBlock = document.getElementById('designate-empty');
  var designateCta = document.getElementById('designate-cta');
  var triggers = [changeButton, designateCta];

  if (!main || !layer || !options || !emptyState || !nameEl || !relEl || !nameBlock || !changeButton || !emptyBlock) {
    return;
  }

  if (!plan.designated) {
    nameBlock.hidden = true;
    changeButton.hidden = true;
    emptyBlock.hidden = false;
  }

  if (plan.emptySheet) {
    options.hidden = true;
    emptyState.hidden = false;
  }

  function setExpanded(value) {
    triggers.forEach(function (trigger) {
      if (trigger) trigger.setAttribute('aria-expanded', String(value));
    });
  }

  function openSheet() {
    layer.classList.add('is-open');
    main.setAttribute('inert', '');
    if (nav) nav.setAttribute('inert', '');
    setExpanded(true);
  }

  function closeSheet() {
    layer.classList.remove('is-open');
    main.removeAttribute('inert');
    if (nav) nav.removeAttribute('inert');
    setExpanded(false);
  }

  triggers.forEach(function (trigger) {
    if (!trigger) return;
    trigger.addEventListener('click', openSheet);
  });

  layer.addEventListener('click', function (event) {
    if (event.target.closest('[data-close-sheet]')) closeSheet();
  });

  options.addEventListener('click', function (event) {
    var option = event.target.closest('.person-option');
    if (!option || option.disabled) return;

    var buttons = options.querySelectorAll('.person-option');
    for (var index = 0; index < buttons.length; index += 1) {
      buttons[index].setAttribute('aria-pressed', String(buttons[index] === option));
    }

    nameEl.textContent = option.getAttribute('data-person') || '';
    relEl.textContent = (option.getAttribute('data-relation') || '') + ' · 已通过';
    closeSheet();
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && layer.classList.contains('is-open')) {
      event.preventDefault();
      closeSheet();
    }
  });

  if (plan.openSheet) {
    openSheet();
  }
})();
