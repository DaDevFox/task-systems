"""
Compare canonical_candidates.json with GitHub issues and produce exported_issues.json
Requires: issue_map.json in workspace and canonical_candidates.json produced by parse_local_ticket_files.py

Behavior:
- Load canonical candidates and issue_map.json
- For each candidate, check if an issue exists on GitHub with exact title match
- If found, check parent via gh sub-issue list
- Also enforce presence of REQUIRES, PROVIDES, PRIORITY in the issue body (POINTS optional)
- If candidate is missing or incomplete, include it in exported_issues.json with schema specified

This script shells out to gh and jq. Ensure gh is authenticated and jq is available in PATH.
"""
import json
import os
import subprocess
import sys
from typing import Optional

WORKSPACE = os.path.abspath(os.path.dirname(__file__))
CANDPATH = os.path.join(WORKSPACE, 'canonical_candidates.json')
ISSUEMAP = os.path.join(WORKSPACE, 'issue_map.json')
OUTPATH = os.path.join(WORKSPACE, 'exported_issues.json')
REPO = 'DaDevFox/task-systems'

# Enforce fields per user instruction
REQUIRE_FIELDS = ['REQUIRES', 'PROVIDES', 'PRIORITY']


def run(cmd: str, capture_output=True) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, shell=True, capture_output=capture_output, text=True)


def gh_issue_list_by_title(title: str) -> list:
    # Use gh issue list --json number,title,body --repo REPO --limit 500
    cmd = f'gh issue list --repo {REPO} --state all --json number,title,body --limit 500'
    p = run(cmd)
    if p.returncode != 0:
        raise RuntimeError(f'gh issue list failed: {p.stderr}')
    arr = json.loads(p.stdout)
    matches = [item for item in arr if item.get('title') == title]
    return matches


def gh_sub_issue_parent(child_number: int) -> Optional[int]:
    # gh sub-issue list <child_issue> --relation parent -R DaDevFox/task-systems --json parent.number
    cmd = f'gh sub-issue list {child_number} --relation parent -R {REPO} --json parent'
    p = run(cmd)
    if p.returncode != 0:
        # sub-issue extension may return non-zero if no parent; try to parse stderr
        # But we'll treat as no parent
        return None
    try:
        obj = json.loads(p.stdout)
    except Exception:
        return None
    if not obj:
        return None
    parent = obj[0].get('parent')
    if not parent:
        return None
    return parent.get('number')


def body_has_required_fields(body: str) -> bool:
    ub = body.upper() if body else ''
    for f in REQUIRE_FIELDS:
        if f not in ub:
            return False
    return True


def map_parent_canonical_to_number(parent_canonical: Optional[str], issue_map: dict) -> Optional[int]:
    if not parent_canonical:
        return None
    # parent_canonical might be numeric string
    if isinstance(parent_canonical, int):
        return parent_canonical
    if parent_canonical in issue_map:
        return issue_map[parent_canonical]
    # maybe canonical like 'SEC-001' maps to 'SEC-001': 123
    return None


def main():
    if not os.path.exists(CANDPATH):
        print(f'Canonical candidates file not found at {CANDPATH}. Run parse_local_ticket_files.py first.')
        sys.exit(1)
    with open(CANDPATH, 'r', encoding='utf-8') as f:
        data = json.load(f)
    candidates = data.get('candidates', [])
    issues_map = {}
    if os.path.exists(ISSUEMAP):
        with open(ISSUEMAP, 'r', encoding='utf-8') as f:
            issues_map = json.load(f)
    exported = []
    scanned = 0
    fully_correct = 0
    ambiguous = []
    for c in candidates:
        scanned += 1
        title = c.get('title')
        parent_can = c.get('parent_canonical')
        expected_parent_num = map_parent_canonical_to_number(parent_can, issues_map)
        matches = []
        try:
            matches = gh_issue_list_by_title(title)
        except Exception as e:
            print('Error querying gh:', e)
            sys.exit(1)
        if not matches:
            # missing issue -> export
            exported.append({
                'issue_number': None,
                'canonical_id': c.get('canonical_id'),
                'title': title,
                'body': c.get('body') or '',
                'labels': c.get('labels') or [],
                'state': 'planned',
                'parent_issue_number': expected_parent_num,
                'requires': c.get('requires') or '',
                'provides': c.get('provides') or '',
                'points': c.get('points') or None,
                'priority': c.get('priority') or None,
                'created_at': None,
                'url': None
            })
            continue
        # If multiple matches, flag ambiguous
        if len(matches) > 1:
            ambiguous.append({'title': title, 'matches': [m['number'] for m in matches]})
            # we'll treat as not fully correct to be safe
            exported.append({
                'issue_number': None,
                'canonical_id': c.get('canonical_id'),
                'title': title,
                'body': c.get('body') or '',
                'labels': c.get('labels') or [],
                'state': 'planned',
                'parent_issue_number': expected_parent_num,
                'requires': c.get('requires') or '',
                'provides': c.get('provides') or '',
                'points': c.get('points') or None,
                'priority': c.get('priority') or None,
                'created_at': None,
                'url': None
            })
            continue
        # single match
        remote = matches[0]
        remote_num = remote.get('number')
        remote_body = remote.get('body') or ''
        # check parent
        parent_num = None
        try:
            parent_num = gh_sub_issue_parent(remote_num)
        except Exception:
            parent_num = None
        parent_ok = (expected_parent_num is None and parent_num is None) or (expected_parent_num == parent_num)
        # check required fields in body
        body_ok = body_has_required_fields(remote_body)
        if parent_ok and body_ok:
            fully_correct += 1
            # skip export
            continue
        else:
            # incomplete -> export
            exported.append({
                'issue_number': None,
                'canonical_id': c.get('canonical_id'),
                'title': title,
                'body': c.get('body') or '',
                'labels': c.get('labels') or [],
                'state': 'planned',
                'parent_issue_number': expected_parent_num,
                'requires': c.get('requires') or '',
                'provides': c.get('provides') or '',
                'points': c.get('points') or None,
                'priority': c.get('priority') or None,
                'created_at': None,
                'url': None,
                'notes': {
                    'remote_issue_number': remote_num,
                    'remote_parent_number': parent_num,
                    'body_has_required_fields': body_ok
                }
            })
    # write exported_issues.json
    with open(OUTPATH, 'w', encoding='utf-8') as f:
        json.dump(exported, f, indent=2, ensure_ascii=False)
    print(f'Processed {scanned} candidates. Fully-correct on GitHub: {fully_correct}. Exported: {len(exported)}. Ambiguous: {len(ambiguous)}')
    if ambiguous:
        print('Ambiguous titles (multiple matches):')
        for a in ambiguous:
            print(a)

if __name__ == '__main__':
    main()
