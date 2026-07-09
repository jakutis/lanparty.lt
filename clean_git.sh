#!/bin/bash

echo "⚠️  WARNING: This script will perform the following actions:"
echo "  1. Forcefully remove all git worktrees (except the main working tree)."
echo "  2. Prune any stale worktree references."
echo "  3. Checkout the 'main' branch."
echo "  4. Forcefully delete ALL local branches except 'main'."
echo ""
read -p "Are you sure you want to proceed? [y/N]: " confirm

if [[ "$confirm" != [yY] && "$confirm" != [yY][eE][sS] ]]; then
    echo "Operation aborted."
    exit 0
fi

echo ""
echo "🧹 Cleaning up..."

# Ensure we're in a git repo
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Error: Not inside a git repository."
    exit 1
fi

echo "Removing worktrees..."
# The first line of `git worktree list` is always the main repository.
# We skip it and process the rest.
git worktree list | awk 'NR>1 {print $1}' | while read -r wt_path; do
    echo "Removing worktree: $wt_path"
    git worktree remove --force "$wt_path" || echo "Failed to remove worktree: $wt_path"
done

# Prune any stale worktree metadata
git worktree prune

echo "Switching to main branch..."
git checkout main || { echo "Failed to checkout main branch. It might not exist."; exit 1; }

echo "Removing non-main branches..."
branches_to_remove=$(git branch --format='%(refname:short)' | grep -v '^main$')

if [ -n "$branches_to_remove" ]; then
    echo "$branches_to_remove" | xargs git branch -D
    echo "Removed branches."
else
    echo "No other branches to remove."
fi

echo "✨ Done."
