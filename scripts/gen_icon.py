#!/usr/bin/env python3
"""Generate OpenBox application icons (PNG + ICO + SVG).

The icon design is intentionally simple: a stylised open cardboard box
with an upward arrow, signifying "extract / open". Brand colour is a
friendly indigo (#4F46E5) so the app stands out in the taskbar without
clashing with system blues.

Outputs:
  internal/assets/icon.png   (512x512 PNG, embedded into the Go binary)
  internal/assets/icon.svg   (master SVG, kept for future tweaks)
  build/icons/icon.ico       (multi-size ICO: 16/32/48/64/128/256)
"""
from __future__ import annotations

import struct
from io import BytesIO
from pathlib import Path

import cairosvg
from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
SVG_PATH = ROOT / "internal" / "assets" / "icon.svg"
PNG_PATH = ROOT / "internal" / "assets" / "icon.png"
ICO_PATH = ROOT / "build" / "icons" / "icon.ico"

BRAND = "#4F46E5"
BRAND_DARK = "#3730A3"
BRAND_LIGHT = "#A5B4FC"
ACCENT = "#FBBF24"

SVG = f"""<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512">
  <defs>
    <linearGradient id="boxFront" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stop-color="{BRAND}"/>
      <stop offset="100%" stop-color="{BRAND_DARK}"/>
    </linearGradient>
    <linearGradient id="boxFlap" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stop-color="{BRAND_LIGHT}"/>
      <stop offset="100%" stop-color="{BRAND}"/>
    </linearGradient>
  </defs>
  <rect x="32" y="32" width="448" height="448" rx="96" ry="96" fill="#FFFFFF"/>
  <polygon points="128,168 384,168 416,232 96,232" fill="url(#boxFlap)"/>
  <polygon points="112,232 400,232 376,408 136,408" fill="url(#boxFront)"/>
  <polygon points="96,232 200,200 200,232" fill="{BRAND_DARK}" opacity="0.85"/>
  <polygon points="416,232 312,200 312,232" fill="{BRAND_DARK}" opacity="0.85"/>
  <polygon points="200,232 312,232 312,200 256,176 200,200" fill="{BRAND_LIGHT}"/>
  <polygon points="256,96 304,160 272,160 272,220 240,220 240,160 208,160"
           fill="{ACCENT}" stroke="#92400E" stroke-width="4" stroke-linejoin="round"/>
</svg>
"""


def write_png_from_svg() -> None:
    SVG_PATH.parent.mkdir(parents=True, exist_ok=True)
    SVG_PATH.write_text(SVG, encoding="utf-8")
    PNG_PATH.parent.mkdir(parents=True, exist_ok=True)
    cairosvg.svg2png(
        bytestring=SVG.encode("utf-8"),
        write_to=str(PNG_PATH),
        output_width=1024,
        output_height=1024,
    )
    print(f"wrote {SVG_PATH}")
    print(f"wrote {PNG_PATH}  (1024x1024)")


def write_ico() -> None:
    sizes = [16, 32, 48, 64, 128, 256]
    frames: list[bytes] = []
    dir_bytes = bytearray()
    dir_bytes += struct.pack("<HHH", 0, 1, len(sizes))

    base = Image.open(PNG_PATH).convert("RGBA")
    for s in sizes:
        img = base.resize((s, s), Image.LANCZOS)
        buf = BytesIO()
        img.save(buf, format="PNG")
        png_data = buf.getvalue()
        frames.append(png_data)
        w = s if s < 256 else 0
        h = s if s < 256 else 0
        offset = 6 + 16 * len(sizes) + sum(len(f) for f in frames[:-1])
        dir_bytes += struct.pack(
            "<BBBBHHII", w, h, 0, 0, 1, 32, len(png_data), offset,
        )

    ICO_PATH.parent.mkdir(parents=True, exist_ok=True)
    with open(ICO_PATH, "wb") as f:
        f.write(dir_bytes)
        for fr in frames:
            f.write(fr)
    print(f"wrote {ICO_PATH}  (sizes={sizes})")


def main() -> None:
    write_png_from_svg()
    write_ico()


if __name__ == "__main__":
    main()
