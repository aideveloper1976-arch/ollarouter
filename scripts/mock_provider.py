"""Minimal OpenAI-compatible mock provider for the OllaRouter smoke test.

Serves GET /v1/models and POST /v1/chat/completions (JSON or SSE stream) and
echoes the last user message back. Standard library only.

Usage: python3 scripts/mock_provider.py <port>
"""
import json, sys
from http.server import BaseHTTPRequestHandler, HTTPServer
class H(BaseHTTPRequestHandler):
    def log_message(self,*a): pass
    def _j(self,code,obj):
        b=json.dumps(obj).encode(); self.send_response(code); self.send_header('Content-Type','application/json'); self.send_header('Content-Length',str(len(b))); self.end_headers(); self.wfile.write(b)
    def do_GET(self):
        if self.path.endswith('/models'): self._j(200,{"object":"list","data":[{"id":"mock-small","object":"model"}]})
        else: self._j(404,{"error":"nf"})
    def do_POST(self):
        n=int(self.headers.get('Content-Length',0)); req=json.loads(self.rfile.read(n) or b'{}')
        last=req["messages"][-1]["content"]
        if req.get("stream"):
            self.send_response(200); self.send_header('Content-Type','text/event-stream'); self.end_headers()
            for part in ["echo: ", str(last)]:
                self.wfile.write(("data: "+json.dumps({"id":"x","object":"chat.completion.chunk","model":req["model"],"choices":[{"index":0,"delta":{"content":part},"finish_reason":None}]})+"\n\n").encode())
            self.wfile.write(("data: "+json.dumps({"id":"x","object":"chat.completion.chunk","model":req["model"],"choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}})+"\n\n").encode())
            self.wfile.write(b"data: [DONE]\n\n"); return
        self._j(200,{"id":"x","object":"chat.completion","model":req["model"],"choices":[{"index":0,"message":{"role":"assistant","content":"echo: "+str(last)},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}})
HTTPServer(('127.0.0.1',int(sys.argv[1])),H).serve_forever()
