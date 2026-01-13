#!/usr/bin/env python3
"""
Fix all parent/child relationships using gh sub-issue extension.
"""

import subprocess
import time

def add_sub_issue(parent_num, child_num, repo="DaDevFox/task-systems"):
    """Add child as sub-issue to parent using gh sub-extension"""
    cmd = [
        "gh", "sub-issue", "add",
        str(parent_num), str(child_num),
        "-R", repo
    ]

    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode != 0:
        print(f"ERROR linking #{child_num} to #{parent_num}")
        print(result.stderr)
        return False

    print(f"  ✓ Linked #{child_num} → #{parent_num}")
    return True

# Parent-child mappings
parent_children = {
    162: [226, 227, 228, 229, 230, 231],  # SEC-001
    163: [232, 233],  # SEC-004
    164: [234, 235],  # SEC-005
    165: [236, 237],  # SEC-006
    166: [238, 239],  # SEC-007
    168: [240, 241, 242],  # ARCH-002
    171: [243, 244, 245],  # EVENT-001
    172: [246, 247, 248],  # OBS-001
}

# GO parent tickets and their children (need to check which tickets exist)
go_parents = {
    180: [249, 250, 251, 252, 253, 254, 255, 264, 265, 266],  # GO-004 (auth server)
    177: [256, 257, 267, 279, 280, 281],  # GO-002 (domain)
    176: [258, 259, 260, 261, 268, 269, 270, 271, 272, 273, 274, 275, 276, 277, 278, 282, 283, 284, 285, 286, 287, 288, 289, 290, 291],  # GO-001 (foundation)
    179: [262, 263, 274, 275, 276, 287, 288, 289, 290, 291],  # GO-003 (repository)
}

def main():
    print("Fixing parent-child relationships using gh sub-issue...\n")

    total_linked = 0
    total_failed = 0

    # Link security children
    print("=== SECURITY ===")
    for parent_num, children in parent_children.items():
        if parent_num not in [162, 163, 164, 165, 166]:
            continue
        print(f"Parent #{parent_num}:")
        for child_num in children:
            if add_sub_issue(parent_num, child_num):
                total_linked += 1
            else:
                total_failed += 1
            time.sleep(0.3)  # Small delay to avoid rate limits

    # Link GO children
    print("\n=== GOLANG ===")
    for parent_num, children in go_parents.items():
        print(f"Parent #{parent_num}:")
        for child_num in children:
            if add_sub_issue(parent_num, child_num):
                total_linked += 1
            else:
                total_failed += 1
            time.sleep(0.3)  # Small delay to avoid rate limits

    print(f"\n=== SUMMARY ===")
    print(f"Total linked: {total_linked}")
    print(f"Total failed: {total_failed}")

if __name__ == "__main__":
    main()
