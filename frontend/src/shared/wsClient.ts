import type { WSMessage } from "./types";

type Options = {
  matchId: string;
  apiBaseHttp: string; // http://127.0.0.1:8081
  token?: string;
  onMessage: (msg: WSMessage) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onStatus?: (status: "connecting" | "connected" | "reconnecting" | "closed") => void;
};

function toWsUrl(httpBase: string, matchId: string, token?: string) {
  const url = new URL(httpBase, window.location.origin);
  const scheme = url.protocol === "https:" ? "wss:" : "ws:";
  const u = new URL(`${scheme}//${url.host}/ws`);
  u.searchParams.set("match_id", matchId);
  if (token) u.searchParams.set("token", token);
  return u.toString();
}

export function connectMatchWS(opts: Options) {
  let ws: WebSocket | null = null;
  let timer: number | undefined;
  let retryCount = 0;
  let closedByUser = false;

  const connect = () => {
    if (closedByUser) return;
    opts.onStatus?.("connecting");
    ws = new WebSocket(toWsUrl(opts.apiBaseHttp, opts.matchId, opts.token));
    ws.onopen = () => {
      retryCount = 0;
      opts.onStatus?.("connected");
      opts.onOpen?.();
    };
    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data) as WSMessage;
        opts.onMessage(msg);
      } catch {
        // ignore
      }
    };
    ws.onclose = () => {
      if (closedByUser) return;
      opts.onStatus?.("reconnecting");
      opts.onClose?.();
      const backoff = Math.min(15000, 1000 * Math.pow(1.8, retryCount));
      const jitter = Math.floor(Math.random() * 350);
      retryCount += 1;
      timer = window.setTimeout(connect, backoff + jitter);
    };
    ws.onerror = () => {
      // let onclose trigger reconnection
    };
  };

  connect();

  return {
    close: () => {
      closedByUser = true;
      if (timer) window.clearTimeout(timer);
      ws?.close();
      opts.onStatus?.("closed");
    }
  };
}
