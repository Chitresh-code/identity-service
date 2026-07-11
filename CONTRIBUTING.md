# Contributing

Coding conventions live in `AGENTS.md` — follow them at all times, for every change.
The workflow below applies at all times too, for every task, no exceptions.

## Workflow

1. **Work item first.** Every task starts as a GitLab work item (issue) before any code
   is written. No untracked changes land on `main`.
2. **Branch per work item.** Branch name format: `type/short-title`
   (e.g. `feat/jwks-endpoint`, `fix/token-refresh-race`).
   Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`.
3. **Commit format:** `type(scope): one line summary`
   (e.g. `feat(auth): add JWKS endpoint`).
   Keep the summary in the imperative mood, under ~72 characters.
4. **Narrate decisions on the work item as you go.** When you make a non-obvious call
   (library choice, schema shape, a tradeoff) or finish a meaningful chunk of work,
   add a comment to the work item explaining it. The work item should read as a log of
   what happened and why, not just an open/closed ticket.
5. **New or changed API endpoint?** Add or update the matching request in `bruno/`
   before opening the PR. The Bruno collection should always reflect the real API
   surface.
6. **Before opening the PR:** update the project wiki with anything a future
   contributor would need (new service capability, new endpoint, new architectural
   decision). Do this before creating the PR, not after.
   (Group-level wikis are a GitLab Premium feature and unavailable on this plan — only
   the per-project wiki is used. Revisit if the group ever upgrades.)
7. **Open the PR**, link it to the work item, and let it close the work item on merge
   (or close it manually right after merging).

## Summary

```text
work item → branch (type/short-title) → commits (type(scope): summary)
   → comment decisions/progress on the work item as you go
   → update bruno/ for API changes → update project wiki
   → open PR → merge → close work item
```
