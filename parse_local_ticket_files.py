"""
Parse local create_*.py scripts to extract canonical ticket candidates.
Outputs: canonical_candidates.json in the workspace.

Behavior:
- Walk workspace for files matching patterns
- For each file, parse AST and extract top-level list/dict literals assigned to names
- Collect list/dict-of-dicts that look like tickets
- Normalize fields: canonical_id, title, body, labels, parent (canonical), points, priority, requires, provides
- Write canonical_candidates.json

This script is conservative and will log files it could not parse or that didn't contain ticket lists.
"""
import ast
import json
import os
import glob
from typing import Any

WORKSPACE = os.path.abspath(os.path.dirname(__file__))
OUTPATH = os.path.join(WORKSPACE, 'canonical_candidates.json')
PATTERNS = [
    'create_*.py', 'create_all_*.py', 'create_remaining*.py', 'create_go*.py', '*create*.py'
]

candidate_list = []
errors = []

def normalize_ticket(d: dict) -> dict:
    # Map common keys to canonical fields
    canonical = {
        'canonical_id': d.get('canonical_id') or d.get('id') or d.get('key') or d.get('ticket_id') or None,
        'title': d.get('title') or d.get('name') or d.get('summary') or '',
        'body': d.get('body') or d.get('description') or d.get('desc') or '',
        'labels': d.get('labels') or d.get('tags') or [],
        'parent_canonical': d.get('parent') or d.get('parent_id') or d.get('parent_key') or None,
        'points': d.get('points') or d.get('estimate') or None,
        'priority': d.get('priority') or None,
        'requires': d.get('requires') or d.get('REQUIRES') or '',
        'provides': d.get('provides') or d.get('PROVIDES') or '',
    }
    # Normalize labels to list of strings
    if isinstance(canonical['labels'], str):
        canonical['labels'] = [s.strip() for s in canonical['labels'].split(',') if s.strip()]
    if canonical['points'] is not None:
        try:
            canonical['points'] = int(canonical['points'])
        except Exception:
            try:
                canonical['points'] = float(canonical['points'])
            except Exception:
                canonical['points'] = None
    return canonical


def extract_literals_from_file(path: str) -> Any:
    with open(path, 'r', encoding='utf-8') as f:
        src = f.read()
    try:
        tree = ast.parse(src, mode='exec')
    except Exception as e:
        raise
    literals = []
    for node in tree.body:
        # look for assignments
        if isinstance(node, ast.Assign):
            try:
                val = node.value
                # Only consider list/tuple/dict literals
                if isinstance(val, (ast.List, ast.Tuple)):
                    obj = ast.literal_eval(val)
                    literals.append(obj)
                elif isinstance(val, ast.Dict):
                    obj = ast.literal_eval(val)
                    literals.append(obj)
                # Also consider NameConstant like None, skip
            except Exception:
                # skip nodes that ast.literal_eval can't handle
                continue
        # also consider simple expression containing a literal (e.g., a top-level list)
        if isinstance(node, ast.Expr):
            try:
                val = node.value
                if isinstance(val, (ast.List, ast.Tuple)):
                    obj = ast.literal_eval(val)
                    literals.append(obj)
                elif isinstance(val, ast.Dict):
                    obj = ast.literal_eval(val)
                    literals.append(obj)
            except Exception:
                continue
    return literals


def main():
    seen_titles = set()
    for pattern in PATTERNS:
        globpath = os.path.join(WORKSPACE, pattern)
        for path in glob.glob(globpath):
            try:
                literals = extract_literals_from_file(path)
            except Exception as e:
                errors.append({'file': path, 'error': str(e)})
                continue
            for lit in literals:
                # if lit is a list of dicts
                if isinstance(lit, list) and all(isinstance(i, dict) for i in lit):
                    for item in lit:
                        cand = normalize_ticket(item)
                        if not cand['title']:
                            continue
                        if cand['title'] in seen_titles:
                            continue
                        seen_titles.add(cand['title'])
                        candidate_list.append(cand)
                elif isinstance(lit, dict):
                    # dict could be a single ticket or a mapping
                    # If mapping of canonical_id -> ticket
                    if all(isinstance(v, dict) for v in lit.values()):
                        for k,v in lit.items():
                            v['canonical_id'] = v.get('canonical_id') or k
                            cand = normalize_ticket(v)
                            if not cand['title']:
                                continue
                            if cand['title'] in seen_titles:
                                continue
                            seen_titles.add(cand['title'])
                            candidate_list.append(cand)
                    else:
                        cand = normalize_ticket(lit)
                        if not cand['title']:
                            continue
                        if cand['title'] in seen_titles:
                            continue
                        seen_titles.add(cand['title'])
                        candidate_list.append(cand)
                else:
                    # skip other literal shapes
                    continue
    out = {
        'workspace': WORKSPACE,
        'candidates_count': len(candidate_list),
        'candidates': candidate_list,
        'errors': errors
    }
    with open(OUTPATH, 'w', encoding='utf-8') as f:
        json.dump(out, f, indent=2, ensure_ascii=False)
    print(f'Wrote {OUTPATH} with {len(candidate_list)} candidates. Errors: {len(errors)}')

if __name__ == '__main__':
    main()
