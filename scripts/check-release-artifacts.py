#!/usr/bin/env python3
"""Verify the six release binaries and SHA256SUMS without executing foreign code."""
import argparse
import hashlib
from pathlib import Path
import re
import struct

EXPECTED = {
    'MCPBox-linux-amd64': ('elf', 62),
    'MCPBox-linux-arm64': ('elf', 183),
    'MCPBox-macos-amd64': ('macho', 0x01000007),
    'MCPBox-macos-arm64': ('macho', 0x0100000C),
    'MCPBox-windows-amd64.exe': ('pe', 0x8664),
    'MCPBox-windows-arm64.exe': ('pe', 0xAA64),
}


def verify(directory):
    actual = {p.name for p in directory.iterdir()}
    if actual != set(EXPECTED) | {'SHA256SUMS'}:
        raise ValueError(f'unexpected/missing release files: {actual ^ (set(EXPECTED) | {"SHA256SUMS"})}')
    entries = {}
    manifest = directory / 'SHA256SUMS'
    if manifest.is_symlink() or not manifest.is_file():
        raise ValueError('SHA256SUMS must be a regular file')
    for line in manifest.read_text().splitlines():
        match = re.fullmatch(r'([a-fA-F0-9]{64})  (\S+)', line)
        if not match or match[2] not in EXPECTED or match[2] in entries:
            raise ValueError('invalid or duplicate checksum entry')
        entries[match[2]] = match[1].lower()
    if set(entries) != set(EXPECTED):
        raise ValueError('manifest does not cover all six binaries')
    for name, (kind, machine) in EXPECTED.items():
        path = directory / name
        if path.is_symlink() or not path.is_file():
            raise ValueError(f'{name}: expected a regular binary')
        digest = hashlib.sha256()
        with path.open('rb') as stream:
            header = stream.read(4096)
            digest.update(header)
            for chunk in iter(lambda: stream.read(1024 * 1024), b''):
                digest.update(chunk)
        if digest.hexdigest() != entries[name]:
            raise ValueError(f'{name}: SHA-256 mismatch')
        if kind == 'elf':
            valid = len(header) >= 20 and header[:6] == b'\x7fELF\x02\x01' and struct.unpack_from('<H', header, 18)[0] == machine
        elif kind == 'macho':
            valid = len(header) >= 8 and header[:4] == b'\xcf\xfa\xed\xfe' and struct.unpack_from('<I', header, 4)[0] == machine
        else:
            offset = struct.unpack_from('<I', header, 60)[0] if len(header) >= 64 else len(header)
            valid = header[:2] == b'MZ' and offset + 6 <= len(header) and header[offset:offset + 4] == b'PE\0\0' and struct.unpack_from('<H', header, offset + 4)[0] == machine
        if not valid:
            raise ValueError(f'{name}: wrong executable format or architecture')
        print(f'OK: {name}')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory', type=Path)
    args = parser.parse_args()
    try:
        verify(args.directory)
    except (OSError, ValueError) as error:
        parser.exit(1, f'FAIL: {error}\n')
