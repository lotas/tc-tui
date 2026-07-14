#!/usr/bin/env python3
"""Drives tc-tui inside a pty and renders chosen frames to PNG via pyte.

Not part of the build; a one-off tool used to generate docs/screenshots/*.png.
Usage: python3 scripts/screenshot.py
"""
import os
import pty
import pyte
import select
import sys
import time
from PIL import Image, ImageDraw, ImageFont

COLS, ROWS = 130, 40
CELL_W, CELL_H = 9, 18
FONT_PATH_CANDIDATES = [
    "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
    "/usr/share/fonts/dejavu/DejaVuSansMono.ttf",
]

PALETTE = {
    "default": (216, 216, 216),
    "black": (20, 20, 20),
    "red": (224, 90, 90),
    "green": (100, 200, 120),
    "yellow": (220, 200, 90),
    "blue": (100, 150, 230),
    "magenta": (200, 110, 220),
    "cyan": (90, 200, 210),
    "white": (230, 230, 230),
    "brightblack": (110, 110, 110),
    "brightred": (255, 120, 120),
    "brightgreen": (130, 230, 150),
    "brightyellow": (240, 220, 120),
    "brightblue": (130, 170, 250),
    "brightmagenta": (230, 140, 240),
    "brightcyan": (130, 220, 230),
    "brightwhite": (255, 255, 255),
}
BG = (18, 18, 22)


def find_font(size):
    for p in FONT_PATH_CANDIDATES:
        if os.path.exists(p):
            return ImageFont.truetype(p, size)
    return ImageFont.load_default()


FONT = find_font(15)
FONT_BOLD = find_font(15)


def color_for(name, default):
    if not name or name == "default":
        return default
    if name in PALETTE:
        return PALETTE[name]
    return default


def render(screen, out_path):
    img = Image.new("RGB", (COLS * CELL_W, ROWS * CELL_H), BG)
    draw = ImageDraw.Draw(img)
    for y in range(ROWS):
        line = screen.buffer[y]
        for x in range(COLS):
            ch = line[x]
            fg = color_for(ch.fg, PALETTE["default"])
            bg = color_for(ch.bg, None)
            if ch.reverse:
                fg, bg = (bg or BG), fg
            if bg:
                draw.rectangle(
                    [x * CELL_W, y * CELL_H, (x + 1) * CELL_W, (y + 1) * CELL_H],
                    fill=bg,
                )
            if ch.data and ch.data != " ":
                draw.text((x * CELL_W, y * CELL_H), ch.data, font=FONT, fill=fg)
    img.save(out_path)
    print(f"wrote {out_path}")


def main():
    root_url = os.environ.get("TASKCLUSTER_ROOT_URL")
    if not root_url:
        print("TASKCLUSTER_ROOT_URL must be set", file=sys.stderr)
        sys.exit(1)

    binary = os.path.abspath("./tc-tui")
    out_dir = os.path.abspath("docs/screenshots")
    os.makedirs(out_dir, exist_ok=True)

    pid, master_fd = pty.fork()
    if pid == 0:
        os.environ["TERM"] = "xterm-256color"
        os.environ["COLUMNS"] = str(COLS)
        os.environ["LINES"] = str(ROWS)
        os.execv(binary, [binary])
        os._exit(1)

    import fcntl
    import struct
    import termios

    fcntl.ioctl(master_fd, termios.TIOCSWINSZ, struct.pack("HHHH", ROWS, COLS, 0, 0))

    stream = pyte.Stream()
    screen = pyte.Screen(COLS, ROWS)
    stream.attach(screen)

    def pump(duration):
        end = time.time() + duration
        while time.time() < end:
            r, _, _ = select.select([master_fd], [], [], 0.1)
            if master_fd in r:
                try:
                    data = os.read(master_fd, 65536)
                except OSError:
                    break
                if not data:
                    break
                stream.feed(data.decode("utf-8", "ignore"))

    def send(s):
        os.write(master_fd, s.encode())

    def snap(name, settle=1.0):
        pump(settle)
        render(screen, os.path.join(out_dir, name))

    # 1. initial screen: worker pools list (default root resource)
    pump(3.0)
    snap("worker-pools.png", settle=0.5)

    # 2. select first row -> worker pool detail
    send("\r")
    snap("worker-pool-detail.png", settle=1.5)

    send("\x1b")
    pump(0.5)

    # 3. jump straight to a busy pool's workers (real running workers, not
    # the empty pool that happened to sort first)
    send(":workers proj-fuzzing/grizzly-reduce-worker\r")
    snap("workers.png", settle=1.5)

    send("\r")  # open first worker
    snap("worker-detail.png", settle=1.0)

    send("\x1b")
    pump(0.3)
    send("\x1b")
    pump(0.3)

    # 4. jump to a plain global list via the command bar
    send(":roles\r")
    snap("roles.png", settle=1.5)

    # 5. help screen
    send("?")
    snap("help.png", settle=0.5)
    send("\x1b")
    pump(0.3)

    send("q")
    time.sleep(0.3)
    try:
        os.kill(pid, 15)
    except ProcessLookupError:
        pass


if __name__ == "__main__":
    main()
