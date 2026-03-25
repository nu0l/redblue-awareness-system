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

  const connect = () => {
    opts.onStatus?.("connecting");
    ws = new WebSocket(toWsUrl(opts.apiBaseHttp, opts.matchId, opts.token));
    ws.onopen = () => {
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
      opts.onStatus?.("reconnecting");
      opts.onClose?.();
      timer = window.setTimeout(connect, 3000);
    };
    ws.onerror = () => {
      // let onclose trigger reconnection
    };
  };

  connect();

  return {
    close: () => {
      if (timer) window.clearTimeout(timer);
      ws?.close();
      opts.onStatus?.("closed");
    }
  };
}
