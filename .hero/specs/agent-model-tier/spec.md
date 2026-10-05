---
title: Agent model tiers
slug: agent-model-tier
type: feature
status: completed
priority: P2
tags: [install, agents, cost]
horizon: now
smoke: false
delivery_method: supervised
created: 2026-10-05
completed_at: 2026-10-05T09:07:27Z
---

# Agent model tiers

## Context

No Hero agent template declares a model, so every installed Claude Code subagent inherits the session model. In a session on the most capable (and most expensive) model, routine agents such as `issue-tracker`, `session-primer` and the scrubbers run on it too.

Claude Code reads `model:` from agent frontmatter (`haiku`, `sonnet`, `opus`, `inherit` or a full model id). A raw `model:` key cannot go in the canonical templates: `installFlat` (`internal/install/content.go`) copies agent files verbatim for the Claude, OpenCode, Cursor, Grok and generic targets, and OpenCode reads `model` as `provider/model-id` (it already reads these templates' `mode:` and `temperature:` keys). Codex and Copilot rebuild frontmatter from `name`/`description` only, so they are unaffected either way.

## Goal

Agent templates declare a harness-neutral `model_tier:` and the installer maps it to Claude Code's `model:` while stripping it for every other verbatim target, so routine agents run on cheaper models under Claude Code without leaking a Claude-specific key into other harnesses.

## Kickoff

Add a `model_tier:` frontmatter key to canonical agent templates with values `light`, `standard` and `deep`. In `installFlat`, when installing `agents`, rewrite that line: for `TargetClaude` replace it with `model: haiku` (light) or `model: sonnet` (standard) and drop it for `deep` (inherit the session model); for every other target drop the line. An unknown value fails the install with the file and value named. Tag `issue-tracker` and `session-primer` light; the seven scrubbers, `documentation-engineer`, `functional-qa-engineer` and `dependency-analyst` standard. Leave every other agent untagged (inherits). Tests cover the rewrite per target, the unknown-value error, and that every shipped template's tier is valid.

## Approach

`installFlat` already receives `opts.Target`. For `kind == "agents"` it passes the source bytes through `renderAgentModelTier(data, opts.Target == TargetClaude)` before the write. `copyFileFromFS` is split so the idempotency, trusted-checksum and dry-run logic runs on the rewritten bytes. Trusted checksums are recorded from installed files, not sources, so upgrades of rewritten files keep working.

Only the leading frontmatter block is scanned; a `model_tier:` line in a body is untouched. Files without the key are written unchanged, byte for byte.

Tier names, not model names, live in templates so a model rename is a one-line installer change and other harnesses can map tiers later.

## Changes

- `internal/install/content.go` — `installFlat` applies `renderAgentModelTier` to agents; add `renderAgentModelTier` and the tier→Claude model table.
- `internal/install/files.go` — split `copyFileFromFS` into a read step and `writeInstalledFile(opts, result, srcPath, data, dst)`.
- `internal/install/content_test.go` — rewrite per target, unknown value, body occurrence untouched, untagged file byte-identical.
- A content test that every shipped agent template (core and every domain) has a valid `model_tier` or none.
- Templates: `core/agents/session-primer.md`, `domains/engineering/agents/issue-tracker.md` (light); `core/agents/documentation-engineer.md`, `domains/engineering/agents/{comment,deadcode,dedup,defensive,dependency,legacy,type}-scrubber.md`, `domains/engineering/agents/functional-qa-engineer.md`, `domains/engineering/agents/dependency-analyst.md` (standard).

## Boundaries

- Does NOT map tiers for OpenCode, Cursor, Grok, Codex, Copilot or generic; they get no model line.
- Does NOT tag architects, reviewers, delivery leads, engineers, investigators or non-engineering domain agents.
- Does NOT add a per-project override for tiers.
- Does NOT change commands or skills.

## Risks

1. **A light-tier agent underperforms.** Haiku on `session-primer` may load thinner context. Mitigation: only two agents are light; moving one to standard is a one-line template change.
2. **Users who edited an installed agent.** Rewritten bytes differ from the previous install, so a user-edited file still refuses overwrite without `--force`, the same as any template change today.

## Validation

- `go test ./...` and `go build ./...` pass.
- `hero install --target claude` into a scratch project writes `model: haiku` to `issue-tracker.md` and `model: sonnet` to `deadcode-scrubber.md`, and no `model_tier` anywhere.
- `hero install --target opencode` into a scratch project writes no `model_tier` and no `model` line.

## Acceptance Criteria

- WHEN installing agents for the Claude target and a template has `model_tier: light` THEN THE SYSTEM SHALL write `model: haiku` in its place.
- WHEN installing agents for the Claude target and a template has `model_tier: standard` THEN THE SYSTEM SHALL write `model: sonnet` in its place.
- WHEN installing agents for the Claude target and a template has `model_tier: deep` THEN THE SYSTEM SHALL drop the line and write no `model:`.
- WHEN installing agents for any verbatim target other than Claude THEN THE SYSTEM SHALL drop the `model_tier:` line and write no `model:`.
- WHEN a template's `model_tier` value is not light, standard or deep THEN THE SYSTEM SHALL fail the install naming the file and the value.
- WHEN a template has no `model_tier:` in its frontmatter THEN THE SYSTEM SHALL install it byte-identical to the source.
- THE SYSTEM SHALL ship `issue-tracker` and `session-primer` as light, and the seven scrubbers, `documentation-engineer`, `functional-qa-engineer` and `dependency-analyst` as standard.

## Completion Ledger

Delivered in a hero worktree on branch `feat/agent-model-tier`. Stack: Go (`internal/install`). Validation: `go build ./...`, `go vet ./internal/install/`, `go test ./...` (all packages pass), plus scratch installs with a dev binary for the claude and opencode targets.

### Acceptance Criteria

| # | Criterion (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | Claude + light → `model: haiku` | DONE | `internal/install/content.go:91` `renderAgentModelTier`; `TestRenderAgentModelTier`, `TestInstallFlat_ModelTierPerTarget`; scratch claude install shows 2 × `model: haiku` |
| 2 | Claude + standard → `model: sonnet` | DONE | `claudeModelForTier` `content.go:80`; `TestRenderAgentModelTier`; scratch claude install shows 10 × `model: sonnet` |
| 3 | Claude + deep → line dropped, no `model:` | DONE | `TestRenderAgentModelTier` deep case |
| 4 | Other verbatim targets → line dropped, no `model:` | DONE | `TestInstallFlat_ModelTierPerTarget` covers opencode, cursor, grok, generic; scratch opencode install has no `model`/`model_tier` |
| 5 | Unknown value fails install naming file and value | DONE | `renderAgentModelTier` error + `installFlat` wraps with source path; `TestRenderAgentModelTier_Unknown`, `TestInstallFlat_UnknownTierNamesFile` (asserts path and value) |
| 6 | No `model_tier` → byte-identical install | DONE | `TestRenderAgentModelTier_Unchanged`, untagged file check in `TestInstallFlat_ModelTierPerTarget` |
| 7 | Ships the specified light/standard tags | DONE | 12 templates tagged; `TestShippedAgents_ModelTiers` installs every domain and asserts the engineering mapping and that no untagged engineering agent gets a `model:` line |

### Changes

| # | Changes item (abbreviated) | Status | Note |
|---|---|---|---|
| 1 | `content.go` — installFlat applies tier rewrite; table + function | DONE | `installFlat` agents branch, `claudeModelForTier`, `renderAgentModelTier` |
| 2 | `files.go` — split copyFileFromFS / writeInstalledFile | DONE | `files.go:31`; dry-run, idempotency and trusted-checksum logic unchanged, now on rendered bytes |
| 3 | `content_test.go` — rewrite/unknown/body/untagged tests | DONE | Placed in new `internal/install/model_tier_test.go` rather than `content_test.go` to keep the tier tests together |
| 4 | Content test: every shipped template tier valid | DONE | `TestShippedAgents_ModelTiers` (all domains via the real installer) |
| 5 | Template tags | DONE | `core/agents/{session-primer,documentation-engineer}.md`, 10 engineering agents |

### Exercise-the-feature check

- [x] User-visible behavior was exercised end-to-end: built `./cmd/hero` to a scratch binary, ran `hero install project . --target claude` and `--target opencode` in fresh git repos; claude agents carry 2 × `model: haiku` and 10 × `model: sonnet`, opencode agents carry neither `model` nor `model_tier`. Output saved in `exercise-evidence.md` beside this spec.

### Excellence Bar self-check

Yes — small, target-aware rewrite at the one choke point all verbatim targets share, no behavior change for untagged agents, and tests pin both the per-target mapping and the shipped tags.
