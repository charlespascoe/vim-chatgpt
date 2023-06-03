import vim
import re

bullet_re = re.compile(r"^(\s{4})*[-*] ")
repeated_ws_re = re.compile(r"(?<=[^ \t])\s{2,}(?=[^ \t])")


def join(lines=None, start=None, end=None):
    if lines is None:
        lines = vim.current.buffer

    if start is not None or end is not None:
        start = start or 1
        end = end or len(lines)
        lines = lines[start - 1 : end]

    return "\n".join(
        repeated_ws_re.sub(" ", line.rstrip()) for line in merge(iter(lines))
    )


def merge(buf_iter):
    for line in buf_iter:
        if line.startswith("```"):
            yield line
            yield from read_code(buf_iter)
        else:
            yield from read_block(line, buf_iter)


def read_code(buf_iter):
    for line in buf_iter:
        yield line.replace("\t", "    ")

        if line == "```":
            return


def read_block(first, buf_iter):
    if first == "":
        yield ""
        return

    block = [first]

    for line in buf_iter:
        if line == "" or bullet_re.match(line):
            yield " ".join(block)
            yield from read_block(line, buf_iter)
            return

        block.append(line)

    yield " ".join(block)
