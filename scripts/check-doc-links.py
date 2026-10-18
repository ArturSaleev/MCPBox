#!/usr/bin/env python3
"""Check Markdown/HTML links in the README entry points (Python 3 + optional curl).

python3 scripts/check-doc-links.py
python3 scripts/check-doc-links.py --external --published-ref main

Localhost links are examples for a running local installation and are not probed.
Published checks resolve repository-relative targets on GitHub, including images.
HTTP checks use HEAD, so release binaries are never downloaded.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
from html.parser import HTMLParser
import ipaddress
from pathlib import Path
import re
import subprocess
import sys
from urllib.parse import quote, unquote, urlsplit

ROOT = Path(__file__).resolve().parents[1]
REPOSITORY = 'https://github.com/ArturSaleev/MCPBox'


class HTMLLinks(HTMLParser):
    def __init__(self):
        super().__init__()
        self.links = []

    def handle_starttag(self, tag, attrs):
        for key, value in attrs:
            if key in ('href', 'src') and value:
                self.links.append(value)


def links_in(text):
    # Ignore fenced code and inline code examples, but retain Markdown destinations.
    text = re.sub(r'^\s*(```|~~~).*?^\s*\1\s*$', '', text, flags=re.M | re.S)
    text = re.sub(r'`[^`\n]*`', '', text)
    parser = HTMLLinks()
    parser.feed(text)
    links = re.findall(r'!?\[[^\]\n]*\]\(\s*(<[^>]+>|[^\s)]+)(?:\s+[^)]*)?\)', text)
    references = dict(re.findall(r'^\s*\[([^\]]+)\]:\s*(\S+)', text, flags=re.M))
    for label, ref in re.findall(r'!?\[([^\]]+)\]\[([^\]]*)\]', text):
        links.append(references.get(ref or label, 'missing-reference:' + (ref or label)))
    return set(link.strip('<>') for link in links + parser.links)


def local_host(host):
    if host in ('localhost',) or host.endswith('.localhost'):
        return True
    try:
        return ipaddress.ip_address(host).is_loopback
    except ValueError:
        return False


def check_url(url):
    result = subprocess.run(
        ['curl', '--silent', '--show-error', '--location', '--head',
         '--connect-timeout', '10', '--max-time', '30', '--retry', '1',
         '--output', '/dev/null', '--write-out', '%{http_code}', url],
        capture_output=True, text=True,
    )
    status = result.stdout.strip()
    if result.returncode or not status.isdigit() or not 200 <= int(status) < 400:
        return f'{url}: HTTP {status or "unknown"}' + (f' ({result.stderr.strip()})' if result.stderr.strip() else '')
    return None


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('files', nargs='*', default=['README.md', 'README-ru.md'])
    ap.add_argument('--external', action='store_true', help='Probe external HTTP(S) links using curl HEAD')
    ap.add_argument('--published-ref', help='Also probe relative targets on GitHub at this branch/tag/commit')
    args = ap.parse_args()
    errors, online, skipped = [], set(), set()
    total = 0
    for filename in args.files:
        source = (ROOT / filename).resolve()
        if not source.is_file():
            errors.append(f'{filename}: source file not found')
            continue
        for target in sorted(links_in(source.read_text())):
            total += 1
            parsed = urlsplit(target)
            if parsed.scheme in ('http', 'https'):
                if local_host(parsed.hostname or ''):
                    skipped.add(target)
                elif args.external:
                    online.add(target.split('#')[0])
                continue
            if parsed.scheme in ('mailto', 'tel'):
                continue
            if parsed.scheme:
                errors.append(f'{filename}: unsupported link {target}')
                continue
            path = (source.parent / unquote(parsed.path)).resolve() if parsed.path else source
            if not path.exists():
                errors.append(f'{filename}: missing local target {target}')
                continue
            try:
                relative = path.relative_to(ROOT).as_posix()
            except ValueError:
                errors.append(f'{filename}: link escapes repository: {target}')
                continue
            # A local ignored/untracked file would disappear when the repo is published.
            tracked = subprocess.run(['git', 'ls-files', '--error-unmatch', '--', relative],
                                     cwd=ROOT, capture_output=True).returncode == 0
            if not tracked:
                errors.append(f'{filename}: target is not tracked by Git: {target}')
            if args.published_ref:
                online.add(f'{REPOSITORY}/blob/{quote(args.published_ref, safe="")}/{quote(relative, safe="/")}')
    if online:
        with ThreadPoolExecutor(max_workers=6) as pool:
            errors.extend(error for error in pool.map(check_url, sorted(online)) if error)
    for error in errors:
        print('FAIL:', error, file=sys.stderr)
    print(f'{total} unique-per-file links; {len(online)} HTTP targets; {len(skipped)} localhost examples skipped; {len(errors)} failures.')
    return bool(errors)


if __name__ == '__main__':
    sys.exit(main())
