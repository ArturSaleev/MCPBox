"""Regression tests for the release artifact gate; no binary execution required."""
import contextlib
import hashlib
import importlib.util
import io
from pathlib import Path
import struct
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('release_check', Path(__file__).with_name('check-release-artifacts.py'))
checker = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checker)


class ReleaseArtifactTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for name, (kind, machine) in checker.EXPECTED.items():
            data = bytearray(256)
            if kind == 'elf':
                data[:6] = b'\x7fELF\x02\x01'
                struct.pack_into('<H', data, 18, machine)
            elif kind == 'macho':
                data[:4] = b'\xcf\xfa\xed\xfe'
                struct.pack_into('<I', data, 4, machine)
            else:
                data[:2] = b'MZ'
                struct.pack_into('<I', data, 60, 128)
                data[128:132] = b'PE\0\0'
                struct.pack_into('<H', data, 132, machine)
            (self.root / name).write_bytes(data)
        self.write_manifest()

    def write_manifest(self):
        lines = [f'{hashlib.sha256((self.root / name).read_bytes()).hexdigest()}  {name}' for name in checker.EXPECTED]
        (self.root / 'SHA256SUMS').write_text('\n'.join(lines) + '\n')

    def verify(self):
        with contextlib.redirect_stdout(io.StringIO()):
            checker.verify(self.root)

    def test_expected_set_passes(self):
        self.verify()

    def test_tampering_fails(self):
        with (self.root / 'MCPBox-linux-amd64').open('ab') as stream:
            stream.write(b'modified')
        with self.assertRaisesRegex(ValueError, 'SHA-256 mismatch'):
            self.verify()

    def test_wrong_architecture_fails_even_with_matching_checksum(self):
        (self.root / 'MCPBox-linux-amd64').write_bytes((self.root / 'MCPBox-linux-arm64').read_bytes())
        self.write_manifest()
        with self.assertRaisesRegex(ValueError, 'architecture'):
            self.verify()

    def test_missing_target_fails(self):
        (self.root / 'MCPBox-macos-arm64').unlink()
        with self.assertRaisesRegex(ValueError, 'missing'):
            self.verify()

    def test_duplicate_checksum_fails(self):
        p = self.root / 'SHA256SUMS'
        text = p.read_text()
        p.write_text(text + text.splitlines()[0] + '\n')
        with self.assertRaisesRegex(ValueError, 'duplicate'):
            self.verify()
