# -*- coding: utf-8 -*-
"""一个最小的 OpenAI 图像接口兼容桩，用来端到端验证整条云端底图链路。

刻意在出图时加了 0.28 的径向暗角——那正是真实 gpt-image 系列的表现，
也是这条链路里最该被验证的一环（不消掉的话平铺会出网格）。
"""
import base64, io, json, math, random, sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from PIL import Image

PORT = 8899
MODELS = ["gpt-image-2", "gpt-image-1.5", "gpt-image-1-mini",
          "gpt-4o-mini", "text-embedding-3-large"]  # 后两个用来验证前缀过滤


def make_texture(w, h, seed=0):
    """造一张有细节的假纹理，再乘上径向暗角。"""
    rnd = random.Random(seed)
    img = Image.new("RGB", (w, h))
    px = img.load()
    cx, cy = (w - 1) / 2, (h - 1) / 2
    maxr = math.hypot(cx, cy)
    # 高频细节：多个小周期的正弦 + 颗粒，周期都远小于画幅
    for y in range(h):
        for x in range(w):
            u, v = x / w, y / h
            base = (128
                    + 34 * math.sin(2 * math.pi * u * 17)
                    + 26 * math.cos(2 * math.pi * v * 13)
                    + 18 * math.sin(2 * math.pi * (u + v) * 29))
            base += (rnd.random() - 0.5) * 36
            r = math.hypot(x - cx, y - cy) / maxr
            gain = 1 - 0.28 * r * r          # ← 这就是要被消掉的暗角
            c = max(0, min(255, int(base * gain)))
            px[x, y] = (c, int(c * 0.93), int(c * 0.80))
    return img


def png_b64(img):
    buf = io.BytesIO()
    img.save(buf, format="PNG")
    return base64.b64encode(buf.getvalue()).decode()


class H(BaseHTTPRequestHandler):
    def log_message(self, fmt, *a):
        sys.stderr.write("  [stub] %s\n" % (fmt % a))

    def _json(self, code, obj):
        b = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def _auth_ok(self):
        a = self.headers.get("Authorization", "")
        if not a.startswith("Bearer ") or len(a) < 12:
            self._json(401, {"error": {"message": "missing or malformed api key",
                                       "type": "invalid_request_error"}})
            return False
        return True

    def do_GET(self):
        if not self.path.startswith("/v1/models"):
            return self._json(404, {"error": {"message": "no such endpoint " + self.path}})
        if not self._auth_ok():
            return
        self._json(200, {"object": "list",
                         "data": [{"id": m, "object": "model"} for m in MODELS]})

    def do_POST(self):
        if not self._auth_ok():
            return
        n = int(self.headers.get("Content-Length", 0))
        raw = self.rfile.read(n) if n else b""
        size = "1024x1024"
        prompt = ""
        if self.path.endswith("/images/generations"):
            body = json.loads(raw or b"{}")
            size = body.get("size", "1024x1024")
            prompt = body.get("prompt", "")
            sys.stderr.write("  [stub] generations model=%s size=%s quality=%s "
                             "moderation=%s fmt=%s\n" % (
                                 body.get("model"), size, body.get("quality"),
                                 body.get("moderation"), body.get("output_format")))
            sys.stderr.write("  [stub] prompt=%s\n" % prompt[:150])
            for forbidden in ("input_fidelity", "response_format"):
                if forbidden in body:
                    return self._json(400, {"error": {
                        "message": "unsupported parameter: " + forbidden,
                        "type": "invalid_request_error"}})
        elif self.path.endswith("/images/edits"):
            sys.stderr.write("  [stub] edits, multipart %d bytes\n" % len(raw))
            if b'name="image"' not in raw:
                return self._json(400, {"error": {"message": "missing image field"}})
        else:
            return self._json(404, {"error": {"message": "no such endpoint " + self.path}})

        w, h = (int(x) for x in size.split("x")) if "x" in size else (1024, 1024)
        img = make_texture(w, h, seed=abs(hash(prompt)) % 10000)
        self._json(200, {
            "created": 1700000000,
            "size": size,
            "output_format": "png",
            "data": [{"b64_json": png_b64(img), "url": None, "revised_prompt": None}],
            "usage": {
                "input_tokens": 47, "output_tokens": 1568, "total_tokens": 1615,
                "input_tokens_details": {"text_tokens": 47, "image_tokens": 0},
            },
        })


if __name__ == "__main__":
    print("桩服务监听 http://127.0.0.1:%d/v1" % PORT, flush=True)
    ThreadingHTTPServer(("127.0.0.1", PORT), H).serve_forever()
