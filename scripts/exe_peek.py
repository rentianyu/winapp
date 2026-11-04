#!/usr/bin/env python3
"""exe_peek.py — Windows PE 产物体检（对应 Android 的 apk_peek）

用法:
    python3 exe_peek.py app.exe
    python3 exe_peek.py app.exe --expect-version 0.1.0 --expect-arch amd64
    python3 exe_peek.py app.exe --json

检查项:
  1. PE 魔数（MZ + PE\0\0）与机器字长（PE32 / PE32+）
  2. 子系统必须是 GUI（-H windowsgui），否则双击会弹黑窗
  3. VS_VERSIONINFO：FileVersion / ProductVersion / ProductName / FileDescription
  4. Authenticode 签名是否存在（不代表可信，只代表有签名）
  5. SHA256 与体积

退出码: 0 通过 / 1 有检查项失败
"""
import argparse
import hashlib
import json
import os
import re
import struct
import sys

# Windows runner / 中文 Windows 下 stdout 默认是 cp1252 或 gbk，
# 直接 print 中文会抛 UnicodeEncodeError，这里强制切 UTF-8。
for _stream in (sys.stdout, sys.stderr):
    try:
        enc = (_stream.encoding or "").lower().replace("-", "")
        if enc not in ("utf8", "utf8mb4"):
            _stream.reconfigure(encoding="utf-8", errors="replace")
    except Exception:  # noqa: BLE001
        pass

try:
    import pefile
except ImportError:
    pefile = None

MACHINE = {0x8664: "amd64", 0x14C: "386", 0xAA64: "arm64", 0x1C4: "arm"}
SUBSYS = {2: "WINDOWS_GUI", 3: "WINDOWS_CUI"}


def parse_version_info(pe):
    """从 PE 资源里取 VS_VERSIONINFO 字符串表"""
    out = {}
    try:
        if not hasattr(pe, "FileInfo") or not pe.FileInfo:
            return out
        for fileinfo_list in pe.FileInfo:
            for fi in fileinfo_list:
                if fi.Key != b"StringFileInfo":
                    continue
                for st in fi.StringTable:
                    for k, v in st.entries.items():
                        key = k.decode("utf-8", "ignore")
                        out[key] = v.decode("utf-8", "ignore").strip("\x00")
    except Exception as e:  # 资源被裁剪时容错
        out["_parse_error"] = str(e)
    return out


def peek(path):
    data = open(path, "rb").read()
    info = {
        "path": os.path.abspath(path),
        "size": len(data),
        "size_mb": round(len(data) / 1048576, 2),
        "sha256": hashlib.sha256(data).hexdigest(),
        "mz": data[:2] == b"MZ",
    }
    if not info["mz"]:
        info["error"] = "不是 PE 文件（缺 MZ 头）"
        return info

    pe_off = struct.unpack_from("<I", data, 0x3C)[0]
    info["pe_offset"] = pe_off
    if data[pe_off:pe_off + 4] != b"PE\x00\x00":
        info["error"] = "PE 签名缺失"
        return info

    machine, nsect = struct.unpack_from("<HH", data, pe_off + 4)
    info["machine"] = MACHINE.get(machine, hex(machine))
    info["sections"] = nsect
    magic = struct.unpack_from("<H", data, pe_off + 24)[0]
    info["pe_type"] = {0x10B: "PE32", 0x20B: "PE32+"}.get(magic, hex(magic))
    subsys = struct.unpack_from("<H", data, pe_off + 24 + 68)[0]
    info["subsystem"] = SUBSYS.get(subsys, str(subsys))

    if pefile:
        try:
            # 注意：不能用 fast_load=True + 手动 parse_data_directories(...)，
            # 那种写法在某些 PE 结构下不会填充 FileInfo，导致版本信息读成空。
            pe = pefile.PE(data=data)
            vi = parse_version_info(pe)
            info["version_info"] = vi
            info["sections_list"] = [
                s.Name.rstrip(b"\x00").decode("ascii", "ignore") for s in pe.sections
            ]
            info["has_rsrc"] = ".rsrc" in info["sections_list"]
            info["signed"] = bool(
                pe.OPTIONAL_HEADER.DATA_DIRECTORY[
                    pefile.DIRECTORY_ENTRY["IMAGE_DIRECTORY_ENTRY_SECURITY"]
                ].Size > 0
            )
            pe.close()
        except Exception as e:
            info["pefile_error"] = str(e)
    return info


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("exe")
    ap.add_argument("--expect-version", default=None, help="期望的 FileVersion（如 0.1.0）")
    ap.add_argument("--expect-arch", default=None, choices=["amd64", "arm64", "386"])
    ap.add_argument("--json", action="store_true")
    args = ap.parse_args()

    if not os.path.exists(args.exe):
        print(f"FAIL: 文件不存在 {args.exe}")
        return 1

    info = peek(args.exe)
    fails = []
    if info.get("error"):
        fails.append(info["error"])
    if pefile is None:
        fails.append("缺少 pefile 模块，无法读取版本信息（pip install pefile）")
    if info.get("subsystem") not in (None, "WINDOWS_GUI"):
        fails.append(f"子系统不是 GUI（{info.get('subsystem')}），双击会弹黑窗")
    if args.expect_arch and info.get("machine") != args.expect_arch:
        fails.append(f"架构不符: {info.get('machine')} != {args.expect_arch}")

    vi = info.get("version_info") or {}
    fv = vi.get("FileVersion") or ""
    if args.expect_version:
        # FileVersion 可能是 "0.1.0" 或 "0.1.0.0" 或 "0,1,0,0"
        norm = re.sub(r"[^0-9.]", ".", fv).strip(".")
        want = args.expect_version.strip()
        if not (norm == want or norm.startswith(want + ".") or norm.replace(".", "") == want.replace(".", "")):
            fails.append(f"版本不符: FileVersion={fv or '(空)'} 期望 {want}")

    if args.json:
        info["ok"] = not fails
        info["fails"] = fails
        print(json.dumps(info, ensure_ascii=False, indent=2))
    else:
        print(f"文件      : {info['path']}")
        print(f"体积      : {info['size_mb']} MB ({info['size']} B)")
        print(f"PE        : {info.get('pe_type')} / {info.get('machine')} / {info.get('subsystem')} / {info.get('sections')} 节")
        if info.get("has_rsrc") is not None:
            print(f"资源节    : {'.rsrc 存在' if info['has_rsrc'] else '缺 .rsrc（图标/版本信息未嵌入）'}")
        print(f"FileVer   : {fv or '(无)'}")
        print(f"ProductVer: {vi.get('ProductVersion', '(无)')}")
        print(f"产品名    : {vi.get('ProductName', '(无)')}")
        print(f"描述      : {vi.get('FileDescription', '(无)')}")
        print(f"签名      : {'有 Authenticode' if info.get('signed') else '无（公开分发会被 SmartScreen 警告）'}")
        print(f"SHA256    : {info['sha256']}")
        print()
        if fails:
            print("❌ FAIL")
            for f in fails:
                print("   -", f)
        else:
            print("✅ PASS")
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
