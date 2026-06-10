export function createRealtimeWebSocketClient(options = {}) {
  let socket;
  let reconnectTimer;
  let status = 'Disconnected';

  function emitStatus(nextStatus) {
    status = nextStatus;
    options.onStatusChange?.(nextStatus);
  }

  function clearReconnectTimer() {
    if (reconnectTimer) {
      const clearTimer = options.clearTimeoutFn || globalThis.clearTimeout;
      clearTimer(reconnectTimer);
      reconnectTimer = undefined;
    }
  }

  function closeSocket() {
    if (!socket) return;
    const currentSocket = socket;
    socket = undefined;
    currentSocket.onopen = null;
    currentSocket.onmessage = null;
    currentSocket.onerror = null;
    currentSocket.onclose = null;
    currentSocket.close();
  }

  function socketURL() {
    return typeof options.url === 'function' ? options.url() : options.url;
  }

  function shouldReconnect() {
    return Boolean(options.shouldReconnect?.());
  }

  function scheduleReconnect() {
    if (!shouldReconnect()) return;
    const setTimer = options.setTimeoutFn || globalThis.setTimeout;
    reconnectTimer = setTimer(() => {
      reconnectTimer = undefined;
      if (shouldReconnect()) {
        connect();
      }
    }, options.reconnectDelayMs ?? 5000);
  }

  function connect() {
    clearReconnectTimer();
    closeSocket();

    const WebSocketCtor = options.WebSocketCtor || globalThis.WebSocket;
    if (!WebSocketCtor) {
      emitStatus('Error');
      options.onError?.(new Error('WebSocket is unavailable'));
      return;
    }

    emitStatus('Connecting');
    const nextSocket = new WebSocketCtor(socketURL());
    socket = nextSocket;

    nextSocket.onopen = (event) => {
      if (socket !== nextSocket) return;
      emitStatus('Connected');
      options.onOpen?.(event);
    };

    nextSocket.onmessage = (event) => {
      if (socket !== nextSocket) return;
      try {
        const message = JSON.parse(event.data);
        options.onMessage?.(message, event);
      } catch (err) {
        options.onMalformedMessage?.(event, err);
      }
    };

    nextSocket.onerror = (event) => {
      if (socket !== nextSocket) return;
      emitStatus('Error');
      options.onError?.(event);
      nextSocket.close();
    };

    nextSocket.onclose = (event) => {
      if (socket !== nextSocket) return;
      socket = undefined;
      if (status !== 'Error') {
        emitStatus('Disconnected');
      }
      options.onClose?.(event);
      scheduleReconnect();
    };
  }

  function disconnect(nextStatus = 'Disconnected') {
    clearReconnectTimer();
    closeSocket();
    emitStatus(nextStatus);
  }

  return {
    connect,
    disconnect,
    getSocket: () => socket,
    getStatus: () => status
  };
}
