(function () {
  'use strict';

  var INITIAL_CODE = 'A7K2-9F3D';
  var endOfDay = new Date();
  endOfDay.setHours(23, 59, 59, 999);
  var REFRESH_SECONDS = Math.max(1, Math.round((endOfDay.getTime() - Date.now()) / 1000));
  var QR_SIZE = 29;
  var QR_QUIET = 4;
  var QR_CELL = 8;
  var QR_OFFSET = QR_QUIET * QR_CELL;
  var CODE_ALPHABET = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';

  var countdownEl = document.getElementById('countdown');
  var codeEl = document.getElementById('verify-code');
  var qrPathEl = document.getElementById('qr-modules');
  var refreshButton = document.getElementById('refresh-button');
  var backButton = document.getElementById('back-button');
  var refreshStatus = document.getElementById('refresh-status');

  function hashString(text) {
    var hash = 2166136261;
    for (var index = 0; index < text.length; index += 1) {
      hash ^= text.charCodeAt(index);
      hash = Math.imul(hash, 16777619);
    }
    return hash >>> 0;
  }

  function createRandom(seed) {
    var state = seed >>> 0;
    return function () {
      state = (state + 0x6D2B79F5) >>> 0;
      var value = state;
      value = Math.imul(value ^ (value >>> 15), value | 1);
      value ^= value + Math.imul(value ^ (value >>> 7), value | 61);
      return ((value ^ (value >>> 14)) >>> 0) / 4294967296;
    };
  }

  function buildMatrix(code) {
    var cells = [];
    var reserved = [];
    var row;
    var column;
    for (row = 0; row < QR_SIZE; row += 1) {
      cells.push(new Array(QR_SIZE).fill(false));
      reserved.push(new Array(QR_SIZE).fill(false));
    }

    function placeFinder(originX, originY) {
      for (var y = -1; y <= 7; y += 1) {
        for (var x = -1; x <= 7; x += 1) {
          var gridX = originX + x;
          var gridY = originY + y;
          if (gridX < 0 || gridY < 0 || gridX >= QR_SIZE || gridY >= QR_SIZE) {
            continue;
          }
          reserved[gridY][gridX] = true;
          var inside = x >= 0 && x <= 6 && y >= 0 && y <= 6;
          cells[gridY][gridX] = inside && (
            x === 0 || x === 6 || y === 0 || y === 6 ||
            (x >= 2 && x <= 4 && y >= 2 && y <= 4)
          );
        }
      }
    }

    placeFinder(0, 0);
    placeFinder(QR_SIZE - 7, 0);
    placeFinder(0, QR_SIZE - 7);

    var centerX = QR_SIZE - 9;
    var centerY = QR_SIZE - 9;
    for (var offsetY = -2; offsetY <= 2; offsetY += 1) {
      for (var offsetX = -2; offsetX <= 2; offsetX += 1) {
        var alignX = centerX + offsetX;
        var alignY = centerY + offsetY;
        reserved[alignY][alignX] = true;
        cells[alignY][alignX] = Math.max(Math.abs(offsetX), Math.abs(offsetY)) !== 1;
      }
    }

    for (var timing = 8; timing <= QR_SIZE - 9; timing += 1) {
      reserved[6][timing] = true;
      reserved[timing][6] = true;
      cells[6][timing] = timing % 2 === 0;
      cells[timing][6] = timing % 2 === 0;
    }

    var random = createRandom(hashString(code));
    for (row = 0; row < QR_SIZE; row += 1) {
      for (column = 0; column < QR_SIZE; column += 1) {
        if (!reserved[row][column]) {
          cells[row][column] = random() < 0.47;
        }
      }
    }

    return cells;
  }

  function buildQrPath(code) {
    var cells = buildMatrix(code);
    var commands = [];
    for (var row = 0; row < QR_SIZE; row += 1) {
      var column = 0;
      while (column < QR_SIZE) {
        if (!cells[row][column]) {
          column += 1;
          continue;
        }
        var end = column;
        while (end + 1 < QR_SIZE && cells[row][end + 1]) {
          end += 1;
        }
        var x = QR_OFFSET + column * QR_CELL;
        var y = QR_OFFSET + row * QR_CELL;
        var width = (end - column + 1) * QR_CELL;
        commands.push('M' + x + ' ' + y + 'h' + width + 'v' + QR_CELL + 'h-' + width + 'z');
        column = end + 1;
      }
    }
    return commands.join('');
  }

  function newVerificationCode(previous) {
    var next = previous;
    var attempt = 0;
    while (next === previous && attempt < 12) {
      var raw = '';
      if (window.crypto && typeof window.crypto.getRandomValues === 'function') {
        var values = new Uint32Array(8);
        window.crypto.getRandomValues(values);
        for (var index = 0; index < values.length; index += 1) {
          raw += CODE_ALPHABET[values[index] % CODE_ALPHABET.length];
        }
      } else {
        for (var fallback = 0; fallback < 8; fallback += 1) {
          raw += CODE_ALPHABET[Math.floor(Math.random() * CODE_ALPHABET.length)];
        }
      }
      next = raw.slice(0, 4) + '-' + raw.slice(4, 8);
      attempt += 1;
    }
    return next;
  }

  function paintCode(code) {
    if (codeEl) {
      codeEl.textContent = code;
    }
    if (qrPathEl) {
      qrPathEl.setAttribute('d', buildQrPath(code));
    }
  }

  var currentCode = INITIAL_CODE;
  var deadline = Date.now() + REFRESH_SECONDS * 1000;
  var timer = null;
  var lastShown = null;

  function renderCountdown() {
    if (!countdownEl) {
      return;
    }
    var remaining = Math.max(0, Math.ceil((deadline - Date.now()) / 1000));
    var next = String(remaining);
    if (next !== lastShown) {
      countdownEl.textContent = next;
      lastShown = next;
    }
  }

  function renewCode(announce) {
    currentCode = newVerificationCode(currentCode);
    paintCode(currentCode);
    deadline = Date.now() + REFRESH_SECONDS * 1000;
    lastShown = null;
    renderCountdown();
    if (announce && refreshStatus) {
      refreshStatus.textContent = '接送码已刷新';
    }
  }

  function tick() {
    if (Date.now() >= deadline) {
      renewCode(false);
      return;
    }
    renderCountdown();
  }

  paintCode(currentCode);
  renderCountdown();

  if (refreshButton) {
    refreshButton.addEventListener('click', function () {
      renewCode(true);
    });
  }

  if (backButton) {
    backButton.addEventListener('click', function () {
      if (window.history.length > 1) {
        window.history.back();
      }
    });
  }

  document.addEventListener('visibilitychange', function () {
    if (!document.hidden) {
      tick();
    }
  });

  timer = window.setInterval(tick, 500);
  window.addEventListener('pagehide', function () {
    window.clearInterval(timer);
  });
}());
