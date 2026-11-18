(function () {
  'use strict';

  var QR_SIZE = 29;
  var QR_QUIET = 4;
  var QR_CELL = 8;
  var QR_OFFSET = QR_QUIET * QR_CELL;
  /* 静态评审要稳定的图案；点「立即刷新」才换种子。 */
  var DEFAULT_SEED = 'pickup-2026-11-12';

  var params = new URLSearchParams(window.location.search);
  var requested = (params.get('state') || '').toLowerCase();

  var passBody = document.getElementById('pass-body');
  var panels = document.querySelectorAll('[data-panel]');
  var qrPathEl = document.getElementById('qr-modules');
  var refreshButton = document.getElementById('refresh-button');
  var backButton = document.getElementById('back-button');
  var refreshStatus = document.getElementById('refresh-status');
  var retryButtons = [document.getElementById('retry-expired'), document.getElementById('retry-offline')];

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

  function paintQr(seed) {
    if (qrPathEl) {
      qrPathEl.setAttribute('d', buildQrPath(seed));
    }
  }

  var activePanel = null;
  for (var index = 0; index < panels.length; index += 1) {
    if (panels[index].getAttribute('data-panel') === requested) {
      activePanel = panels[index];
    }
  }

  if (activePanel) {
    panels.forEach(function (panel) {
      panel.hidden = panel !== activePanel;
    });
    if (passBody) passBody.hidden = true;
  }

  function restorePass() {
    if (passBody) passBody.hidden = false;
    panels.forEach(function (panel) {
      panel.hidden = true;
    });
    paintQr('pickup-' + Date.now());
    if (refreshStatus) refreshStatus.textContent = '已重新获取接送码';
  }

  paintQr(DEFAULT_SEED);

  if (refreshButton) {
    refreshButton.addEventListener('click', function () {
      paintQr('pickup-refresh-' + Date.now());
      if (refreshStatus) refreshStatus.textContent = '接送码已刷新';
    });
  }

  retryButtons.forEach(function (button) {
    if (button) button.addEventListener('click', restorePass);
  });

  if (backButton) {
    backButton.addEventListener('click', function () {
      if (window.history.length > 1) {
        window.history.back();
      }
    });
  }
}());
