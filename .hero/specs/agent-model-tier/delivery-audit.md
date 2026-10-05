# Delivery audit — agent-model-tier

**Audited:** `git diff --cached origin/main` (staged, uncommitted) in worktree `agent-model-tier`
**Verdict:** SHIP
**Surface:** noteworthy

## Acceptance criteria
- [✓] Claude + `model_tier: light` → `model: haiku` — `internal/install/content.go` `claudeModelForTier` / `renderAgentModelTier`; `TestRenderAgentModelTier` (light/claude case asserts exact bytes), `TestInstallFlat_ModelTierPerTarget` (claude asserts `model: haiku` present); engineer-reported scratch install shows 2 × `model: haiku`
- [✓] Claude + `standard` → `model: sonnet` — same function; `TestRenderAgentModelTier` standard/claude case; `TestShippedAgents_ModelTiers` asserts `\nmodel: sonnet\n` in the 10 standard agents
- [✓] Claude + `deep` → line dropped, no `model:` — `renderAgentModelTier` writes no replacement when the mapped model is `""`; `TestRenderAgentModelTier` deep/claude case asserts exact output
- [✓] Other verbatim targets → line dropped, no `model:` — `installFlat` passes `opts.Target == TargetClaude`; `TestInstallFlat_ModelTierPerTarget` runs opencode, cursor, grok, generic and asserts no `model_tier` and no `model: haiku`. Codex/Copilot rebuild frontmatter (`render.go:230` `renderCopilotPromptFile` keeps only `description`), so no leak there
- [✓] Unknown value fails naming file and value — `renderAgentModelTier` returns `unknown model_tier "<v>"`; `installFlat` wraps with `srcPath`. `TestRenderAgentModelTier_Unknown` asserts the value only; the file-name half is code-only evidence (see notes)
- [✓] No `model_tier` → byte-identical — function returns `data` unchanged when no key in frontmatter; `TestRenderAgentModelTier_Unchanged` (body occurrence, no frontmatter, untagged) and the untagged-file check in `TestInstallFlat_ModelTierPerTarget` for all five targets
- [✓] Ships specified light/standard tags — diff adds `model_tier: light` to `core/agents/session-primer.md`, `domains/engineering/agents/issue-tracker.md`; `model_tier: standard` to `core/agents/documentation-engineer.md`, the seven `*-scrubber.md`, `functional-qa-engineer.md`, `dependency-analyst.md`. `TestShippedAgents_ModelTiers` installs every domain via the real FS and asserts the mapping for engineering

## Changes
- [✓] `internal/install/content.go` — `installFlat` agents branch reads bytes, applies `renderAgentModelTier`, writes via `writeInstalledFile`; `claudeModelForTier` table and `renderAgentModelTier` added
- [✓] `internal/install/files.go` — `copyFileFromFS` now delegates to new `writeInstalledFile(opts, result, srcPath, data, dst)`; dry-run, byte-equality no-op and trusted-checksum paths moved intact. Trusted checksums are recorded from `action.Dest` (`internal/install/state.go:367`), i.e. installed bytes, as the Approach requires
- [~] `internal/install/content_test.go` tests — tests exist and cover all four listed cases, but live in new `internal/install/model_tier_test.go` instead of `content_test.go`. Deviation acknowledged in the ledger; no functional impact
- [✓] Content test for every shipped template tier — `TestShippedAgents_ModelTiers` iterates `hero.AvailableDomains()` with core overlay; any unknown tier fails the install
- [✓] Template tags — all 12 files touched with the specified tier

## Open items (if any)
- None. No PARTIAL / SKIPPED / BLOCKED rows in the ledger.

## Audit notes
- Changes item 3 placed in `model_tier_test.go`, not `content_test.go` as the spec named. Reasonable; recorded so the spec text and code agree on review.
- AC5 "naming the file": no test asserts the source path appears in the error. The `fmt.Errorf("%s: %w", srcPath, err)` wrap in `installFlat` provides it; only the value is test-pinned.
- `TestShippedAgents_ModelTiers` asserts the tagged agents carry the right model but does not assert that untagged engineering agents (architects, reviewers, leads) carry no `model:` line. The Boundaries section is upheld by the diff (only 12 templates touched), not by a test.
- Scratch-install evidence (2 × haiku, 10 × sonnet for claude; none for opencode) is engineer-reported; no captured output on disk. Counts are consistent with the diff (2 light + 10 standard templates). Auditor re-ran `go test ./internal/install/` — all pass, including the five tier tests.
- Diff is well scoped: only the spec, the 12 templates, `content.go`, `files.go` and the new test file.
