/* 任务详情 · 原型交互（状态演示条由 _shared/shell.js 负责） */
(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

  /* 轻提示 */
  const toastEl = $('#toast');
  let toastTimer = null;
  function toast(message, duration = 1800) {
    if (!toastEl) return;
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toastEl.classList.remove('is-visible'), duration);
  }

  /* 切状态：点一下 shell 的 chip，保持 aria-pressed 与 data-state 同步 */
  function setState(name) {
    const chip = $(`.mp-demo-chip[data-mp-state="${name}"]`);
    if (chip) chip.click();
    else app.dataset.state = name;
  }

  const back = $('.mp-back');
  if (back) back.addEventListener('click', () => toast('返回亲子任务列表'));

  /* 打卡表单 */
  const uploadTile = $('#uploadTile');
  const uploadPreview = $('#uploadPreview');
  const removePhotoBtn = $('#removePhoto');
  const input = $('#checkinInput');
  const submitBtn = $('#submitCheckin');
  const note = $('#checkinNote');
  const quickBtn = $('#quickDone');
  let hasPhoto = false;

  function syncSubmit() {
    const ready = hasPhoto || !!(input && input.value.trim().length > 0);
    if (submitBtn) submitBtn.disabled = !ready;
    if (note) note.textContent = ready ? '可以提交了' : '填写内容或上传照片后可提交';
  }

  if (uploadTile) {
    uploadTile.addEventListener('click', () => {
      hasPhoto = true;
      uploadTile.hidden = true;
      if (uploadPreview) uploadPreview.hidden = false;
      syncSubmit();
      toast('照片已添加（原型模拟）');
    });
  }

  if (removePhotoBtn) {
    removePhotoBtn.addEventListener('click', () => {
      hasPhoto = false;
      if (uploadPreview) uploadPreview.hidden = true;
      if (uploadTile) uploadTile.hidden = false;
      syncSubmit();
    });
  }

  if (input) input.addEventListener('input', syncSubmit);

  function resetDraft() {
    hasPhoto = false;
    if (input) input.value = '';
    if (uploadPreview) uploadPreview.hidden = true;
    if (uploadTile) uploadTile.hidden = false;
    syncSubmit();
  }

  if (submitBtn) {
    submitBtn.addEventListener('click', () => {
      if (submitBtn.disabled) return;
      resetDraft();
      toast('已提交，等待老师回应');
      setState('submitted');
    });
  }

  if (quickBtn) {
    quickBtn.addEventListener('click', () => {
      resetDraft();
      toast('已提交，等待老师回应');
      setState('submitted');
    });
  }

  /* 老师回应的反馈 chip：同一组内单选 */
  $$('.reactions').forEach((group) => {
    $$('.reaction', group).forEach((btn) => {
      btn.addEventListener('click', () => {
        $$('.reaction', group).forEach((other) => {
          other.setAttribute('aria-pressed', String(other === btn));
        });
        toast('已反馈给老师');
      });
    });
  });

  /* 语音：同一时间只有一条在播 */
  const VOICE_IDLE = '▶ 播放老师的语音';
  const VOICE_PLAYING = '正在播放…';
  let voiceTimer = null;

  $$('[data-voice]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const wasPlaying = btn.dataset.playing === 'true';
      $$('[data-voice]').forEach((other) => {
        other.dataset.playing = 'false';
        other.textContent = VOICE_IDLE;
      });
      clearTimeout(voiceTimer);
      if (wasPlaying) return;
      btn.dataset.playing = 'true';
      btn.textContent = VOICE_PLAYING;
      voiceTimer = setTimeout(() => {
        btn.dataset.playing = 'false';
        btn.textContent = VOICE_IDLE;
      }, 2600);
    });
  });

  /* 给孩子看 */
  const kidMask = $('#kidMask');
  const kidGlyph = $('#kidGlyph');
  const kidText = $('#kidText');
  const kidPlay = $('#kidPlay');

  function openKid(glyph, text) {
    if (!kidMask) return;
    if (kidGlyph) kidGlyph.textContent = glyph || '🌸';
    if (kidText) kidText.textContent = text || '';
    kidMask.hidden = false;
  }

  function closeKid() {
    if (!kidMask) return;
    kidMask.hidden = true;
    if (kidPlay) kidPlay.textContent = '▶ 听老师说';
  }

  if (kidMask) {
    kidMask.addEventListener('click', closeKid);
    const body = $('.kid-body', kidMask);
    if (body) body.addEventListener('click', (event) => event.stopPropagation());
  }

  if (kidPlay) {
    kidPlay.addEventListener('click', () => {
      kidPlay.textContent = kidPlay.textContent === '正在播放…' ? '▶ 听老师说' : '正在播放…';
    });
  }

  $$('[data-kid]').forEach((btn) => {
    btn.addEventListener('click', () => openKid(btn.dataset.kidGlyph, btn.dataset.kidText));
  });

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && kidMask && !kidMask.hidden) closeKid();
  });

  /* 公开状态：none / pending / approved / revoked（与源码 publicState 同构）。
   * 申请与取消是独立请求；「一键完成」（无文字无媒体）不给申请入口。
   * 当前周期的提交同样带公开入口，不再只能去历史行里误操作旧记录。 */
  function setPublic(scope, state, options = {}) {
    const badge = scope.querySelector('[data-public-badge]');
    const label = scope.querySelector('[data-public-label]');
    const btn = scope.querySelector('[data-public]');

    const writeBadge = (text, modifier) => {
      if (!badge) return;
      badge.hidden = !text;
      if (text) {
        badge.textContent = text;
        badge.className = `badge ${modifier}`;
      }
    };
    const writeLabel = (text) => {
      if (!label) return;
      label.hidden = !text;
      if (text) label.textContent = text;
    };

    if (state === 'pending') {
      writeBadge('待审核', 'badge--pending');
      writeLabel('已申请，等待老师审核');
      if (btn) { btn.hidden = false; btn.textContent = '取消公开'; btn.dataset.kind = 'pending'; btn.dataset.step = 'idle'; }
    } else if (state === 'approved') {
      writeBadge('老师已通过', 'badge--ok');
      writeLabel(options.visible === false ? '暂不对本班展示' : '本班家长可见');
      if (btn) { btn.hidden = false; btn.textContent = '取消公开'; btn.dataset.kind = 'approved'; btn.dataset.step = 'idle'; }
    } else if (state === 'revoked') {
      writeBadge('未公开', 'badge--muted');
      writeLabel('');
      if (btn) { btn.hidden = false; btn.textContent = '重新申请'; btn.dataset.kind = 'revoked'; btn.dataset.step = 'idle'; }
    } else {
      writeBadge('');
      writeLabel('');
      if (btn) { btn.hidden = false; btn.textContent = '申请公开'; btn.dataset.kind = 'none'; btn.dataset.step = 'idle'; }
    }
  }

  document.addEventListener('click', (event) => {
    const btn = event.target.closest('[data-public]');
    if (!btn) return;
    const scope = btn.closest('[data-public-scope]');
    if (!scope) return;

    // 源码的确认态是单值（pendingCancelId）：换一条操作时上一条退回普通态。
    $$('[data-public][data-step="confirm"]').forEach((other) => {
      if (other === btn) return;
      other.dataset.step = 'idle';
      if (other.dataset.kind === 'pending' || other.dataset.kind === 'approved') {
        other.textContent = '取消公开';
      }
    });

    if (btn.dataset.kind === 'none' || btn.dataset.kind === 'revoked') {
      setPublic(scope, 'pending');
      toast('已申请，等待老师审核');
      return;
    }
    if (btn.dataset.step === 'confirm') {
      setPublic(scope, 'revoked');
      toast('已撤下');
      return;
    }
    btn.dataset.step = 'confirm';
    btn.textContent = '确认撤下？';
    toast('再点一次确认撤下');
  });

  /* 我的提交卡：公开状态跟着演示态走（已提交 = 未申请，已通过 = 老师已通过） */
  const submitPublic = $('.submit-card [data-public-scope]');
  function syncSubmitPublic(state) {
    if (!submitPublic) return;
    if (state === 'approved') setPublic(submitPublic, 'approved');
    else setPublic(submitPublic, 'none');
  }
  document.addEventListener('mp:state', (event) => syncSubmitPublic(event.detail && event.detail.state));
  syncSubmitPublic(app.dataset.state || 'normal');

  /* 空态行动 */
  const emptyCta = $('#emptyCta');
  if (emptyCta) {
    emptyCta.addEventListener('click', () => {
      setState('normal');
      if (input) input.focus();
    });
  }

  /* 失败态重试 */
  const retryBtn = $('#retryBtn');
  if (retryBtn) {
    retryBtn.addEventListener('click', () => {
      toast('正在重试…');
      setState('loading');
      setTimeout(() => setState('normal'), 900);
    });
  }

  const switchChild = $('#switchChild');
  if (switchChild) {
    switchChild.addEventListener('click', () => toast('已切换到消息里的孩子（原型模拟）'));
  }

  syncSubmit();
})();
