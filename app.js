(function() {
  var ta = document.getElementById('input');
  var logEl = document.getElementById('log');
  var statusEl = document.getElementById('status');
  var statusText = statusEl.querySelector('.status-text');
  var eventCountEl = document.getElementById('eventCount');
  var filtersEl = document.getElementById('filters');
  var clearBtn = document.getElementById('clearBtn');
  var clearLogBtn = document.getElementById('clearLogBtn');

  var ws = null;
  var reconnectDelay = 800;
  var maxReconnectDelay = 10000;
  var autoScroll = true;
  var entryCount = 0;
  var MAX_ENTRIES = 500;
  // Delta tracking
  var lastSentLength = 0;
  var isComposing = false;

  // Event logging
  var eventGroups = {
    'Key': ['keydown', 'keyup', 'keypress'],
    'Input': ['input', 'beforeinput'],
    'Compose': ['compositionstart', 'compositionupdate', 'compositionend'],
    'Focus': ['focus', 'blur'],
    'Select': ['select', 'selectionchange'],
    'Other': ['change', 'paste', 'cut', 'copy']
  };

  var enabledGroups = {};
  var groupForEvent = {};

  Object.keys(eventGroups).forEach(function(group) {
    enabledGroups[group] = true;
    eventGroups[group].forEach(function(evt) {
      groupForEvent[evt] = group;
    });
  });

  // Build filter checkboxes
  Object.keys(eventGroups).forEach(function(group) {
    var label = document.createElement('label');
    var cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.checked = true;
    cb.addEventListener('change', function() {
      enabledGroups[group] = cb.checked;
    });
    label.appendChild(cb);
    label.appendChild(document.createTextNode(group));
    filtersEl.appendChild(label);
  });

  // --- WebSocket ---

  function connect() {
    var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(proto + '//' + location.host + '/ws');

    ws.onopen = function() {
      reconnectDelay = 800;
      setStatus('connected');
      // On reconnect, treat current textarea content as already sent
      lastSentLength = ta.value.length;
    };

    ws.onclose = function() {
      setStatus('disconnected');
      scheduleReconnect();
    };

    ws.onerror = function() {
      if (ws) ws.close();
    };

    ws.onmessage = function(e) {
      try {
        var msg = JSON.parse(e.data);
        if (msg.type === 'status') {
          logEvent('ws:status', msg.message, true);
        }
      } catch(err) {}
    };
  }

  function scheduleReconnect() {
    setTimeout(function() {
      connect();
    }, reconnectDelay);
    reconnectDelay = Math.min(reconnectDelay * 1.5, maxReconnectDelay);
  }

  function wsSend(msg) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(msg));
    }
  }

  function setStatus(state) {
    statusEl.setAttribute('data-state', state);
    statusText.textContent = state;
  }

  // --- Delta tracking ---

  // Sends committed (non-composing) text changes
  function syncDelta() {
    var val = ta.value;
    if (val.length > lastSentLength) {
      var newText = val.substring(lastSentLength);
      wsSend({type: 'text', data: newText});
      logEvent('ws:send', 'text: "' + truncate(newText, 60) + '"', true);
      lastSentLength = val.length;
    } else if (val.length < lastSentLength) {
      var deleteCount = lastSentLength - val.length;
      for (var i = 0; i < deleteCount; i++) {
        wsSend({type: 'key', key: 'Backspace'});
      }
      logEvent('ws:send', 'backspace x' + deleteCount, true);
      lastSentLength = val.length;
    }
  }

  function truncate(s, max) {
    if (s.length <= max) return s;
    return s.substring(0, max) + '...';
  }

  // --- Event listeners ---

  ta.addEventListener('compositionstart', function(e) {
    isComposing = true;
    logEvent('compositionstart', 'data="' + (e.data || '') + '"');
  });

  ta.addEventListener('compositionupdate', function(e) {
    var data = e.data || '';
    logEvent('compositionupdate', 'data="' + data + '"');
    wsSend({type: 'compositionupdate', data: data});
    logEvent('ws:send', 'comp: "' + truncate(data, 60) + '"', true);
  });

  ta.addEventListener('compositionend', function(e) {
    isComposing = false;
    var data = e.data || '';
    logEvent('compositionend', 'data="' + data + '"');
    wsSend({type: 'compositionend', data: data});
    // Update lastSentLength to account for the committed composition text
    lastSentLength = ta.value.length;
    logEvent('ws:send', 'compend: "' + truncate(data, 60) + '"', true);
  });

  ta.addEventListener('beforeinput', function(e) {
    var parts = [];
    if (e.inputType) parts.push('inputType=' + e.inputType);
    if (e.data !== undefined && e.data !== null) parts.push('data="' + e.data + '"');
    if (e.isComposing) parts.push('composing');
    logEvent('beforeinput', parts.join(' | '));
  });

  ta.addEventListener('input', function(e) {
    var parts = [];
    if (e.inputType) parts.push('inputType=' + e.inputType);
    if (e.data !== undefined && e.data !== null) parts.push('data="' + e.data + '"');
    if (e.isComposing) parts.push('composing');
    logEvent('input', parts.join(' | '));

    if (!isComposing) {
      syncDelta();
    }
  });

  ['keydown', 'keyup'].forEach(function(evt) {
    ta.addEventListener(evt, function(e) {
      var parts = [];
      parts.push('key="' + e.key + '"');
      parts.push('code=' + e.code);
      if (e.isComposing) parts.push('composing');
      var mods = [];
      if (e.ctrlKey) mods.push('ctrl');
      if (e.altKey) mods.push('alt');
      if (e.shiftKey) mods.push('shift');
      if (e.metaKey) mods.push('meta');
      if (mods.length) parts.push('mods=[' + mods.join('+') + ']');
      logEvent(evt, parts.join(' | '));
    });
  });

  ['focus', 'blur'].forEach(function(evt) {
    ta.addEventListener(evt, function() {
      logEvent(evt, '');
    });
  });

  ta.addEventListener('select', function() {
    logEvent('select', 'sel=' + ta.selectionStart + '-' + ta.selectionEnd);
  });

  document.addEventListener('selectionchange', function() {
    if (document.activeElement !== ta) return;
    logEvent('selectionchange', 'sel=' + ta.selectionStart + '-' + ta.selectionEnd);
  });

  ['paste', 'cut', 'copy', 'change'].forEach(function(evt) {
    ta.addEventListener(evt, function() {
      logEvent(evt, '');
    });
  });

  // Clear button — resets without sending backspaces
  clearBtn.addEventListener('click', function() {
    ta.value = '';
    lastSentLength = 0;
    wsSend({type: 'clear'});
    logEvent('ws:send', 'clear', true);
    ta.focus();
  });

  clearLogBtn.addEventListener('click', function() {
    logEl.innerHTML = '';
    entryCount = 0;
    eventCountEl.textContent = '0';
  });

  // --- Event log ---

  function esc(s) {
    if (s === undefined || s === null) return '';
    return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  }

  function logEvent(type, detail, isSent) {
    var group = groupForEvent[type] || 'Other';
    if (!isSent && !enabledGroups[group]) return;

    entryCount++;
    eventCountEl.textContent = String(entryCount);

    if (entryCount > MAX_ENTRIES) {
      var first = logEl.firstChild;
      if (first) logEl.removeChild(first);
    }

    var div = document.createElement('div');
    div.className = 'log-entry';
    var cls = isSent ? 'evt-sent' : ('evt-' + group.toLowerCase());
    div.innerHTML = '<span class="evt-name ' + cls + '">' + esc(type) + '</span> <span class="evt-detail">' + esc(detail) + '</span>';
    logEl.appendChild(div);

    if (autoScroll) {
      logEl.scrollTop = logEl.scrollHeight;
    }
  }

  logEl.addEventListener('scroll', function() {
    var atBottom = logEl.scrollHeight - logEl.scrollTop - logEl.clientHeight < 30;
    autoScroll = atBottom;
  });

  // Start
  connect();
})();
