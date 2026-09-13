# Agentic SDLC and Spec-Driven Development

Kiro-style Spec-Driven Development on an agentic SDLC

## Project Context

### Paths
- Steering: `.kiro/steering/`
- Specs: `.kiro/specs/`

### Steering vs Specification

**Steering** (`.kiro/steering/`) - Guide AI with project-wide rules and context
**Specs** (`.kiro/specs/`) - Formalize development process for individual features

### Active Specifications
- Check `.kiro/specs/` for active specifications
- Use `/kiro-spec-status [feature-name]` to check progress

## Development Guidelines
- Think in English, generate responses in Japanese. All Markdown content written to project files (e.g., requirements.md, design.md, tasks.md, research.md, validation reports) MUST be written in the target language configured for this specification (see spec.json.language).

## Minimal Workflow
- Phase 0 (optional): `/kiro-steering`, `/kiro-steering-custom`
- Discovery: `/kiro-discovery "idea"` — determines action path, writes brief.md + roadmap.md for multi-spec projects
- Phase 1 (Specification):
  - Single spec: `/kiro-spec-quick {feature} [--auto]` or step by step:
    - `/kiro-spec-init "description"`
    - `/kiro-spec-requirements {feature}`
    - `/kiro-validate-gap {feature}` (optional: for existing codebase)
    - `/kiro-spec-design {feature} [-y]`
    - `/kiro-validate-design {feature}` (optional: design review)
    - `/kiro-spec-tasks {feature} [-y]`
  - Multi-spec: `/kiro-spec-batch` — creates all specs from roadmap.md in parallel by dependency wave
- Phase 2 (Implementation): `/kiro-impl {feature} [tasks]`
  - Without task numbers: autonomous mode (subagent per task + independent review + final validation)
  - With task numbers: manual mode (selected tasks in main context, still reviewer-gated before completion)
  - `/kiro-validate-impl {feature}` (standalone re-validation)
- Progress check: `/kiro-spec-status {feature}` (use anytime)

## Linear synchronization

- Team: `Rn404` / Project: `nippo-cli` (https://linear.app/rn404/project/nippo-cli-468e5971d8fb)
- Granularity: one Linear issue per spec (`.kiro/specs/{feature}/`). Never mirror individual
  `tasks.md` sub-tasks (1.1, 2.1, ...) as separate issues or sub-issues.

1. **Create, at `/kiro-spec-requirements` approval** (i.e. as soon as
   `approvals.requirements.approved` becomes `true` in `spec.json`) — this is the earliest point
   where the spec has a stable, behavior-only description (problem + acceptance criteria) worth
   showing to the team, well before design/task-level detail exists.
   - Title: feature name
   - Description: requirements.md's Introduction/Project Description, summarized in natural
     language (no implementation detail)
   - Team: `Rn404`, Project: `nippo-cli`
   - Save the created issue's `id` and `url` back into `spec.json` as `linear_issue_id` /
     `linear_issue_url`, so later steps can find it without searching by title.
2. **Update, don't duplicate**: if `spec.json` already has `linear_issue_id`, update that issue
   instead of creating a new one. Refresh the description when requirements are revised, or when
   design/tasks are approved, summarizing what changed rather than dumping raw design/task content.
3. **Start work**: when `/kiro-impl` begins implementing the spec, set the issue to "In Progress".
4. **Track progress**: during `/kiro-impl`, add one short comment to the issue each time a
   `tasks.md` sub-task is marked `[x]` (e.g., "Task 1 完了 — commit `<hash>`"). One comment per
   completed sub-task only — never per implementer/reviewer dispatch, retry round, or debug
   cycle. Do not change status for this (status changes are reserved for steps 3/5).
5. **Finish work**: when `/kiro-validate-impl` returns `GO`:
   - If the implementation was committed directly to the base branch (no open PR), set the issue
     to "Done".
   - If the implementation is sitting in one or more open, unmerged PRs (e.g. a stacked-PR
     series), set the issue to "In Review" instead — validated is not the same as shipped. Only
     set it to "Done" after confirming the full PR stack has actually been merged into the base
     branch (e.g. via `gh pr view --json state,mergedAt`), not just that CI is green.
6. **Surface friction**: if a task is marked `_Blocked:_`, or a requirement/design change is
   material enough to affect the issue's original scope, add a comment explaining it. Do not
   change status for this alone (status changes are reserved for steps 3/5).
7. **Best-effort**: Linear API/MCP failures must never block the local kiro-* workflow. On
   failure, warn once and continue; do not retry-loop or halt implementation.

## Skills Structure
Skills are located in `.claude/skills/kiro-*/SKILL.md`
- Each skill is a directory with a `SKILL.md` file
- Skills run inline with access to conversation context
- Skills may delegate parallel research to subagents for efficiency
- Additional files (templates, examples) can be added to skill directories
- `kiro-review` — task-local adversarial review protocol used by reviewer subagents
- `kiro-debug` — root-cause-first debug protocol used by debugger subagents
- `kiro-verify-completion` — fresh-evidence gate before success or completion claims
- **If there is even a 1% chance a skill applies to the current task, invoke it.** Do not skip skills because the task seems simple.

## Development Rules
- 3-phase approval workflow: Requirements → Design → Tasks → Implementation
- Human review required each phase; use `-y` only for intentional fast-track
- Keep steering current and verify alignment with `/kiro-spec-status`
- Follow the user's instructions precisely, and within that scope act autonomously: gather the necessary context and complete the requested work end-to-end in this run, asking questions only when essential information is missing or the instructions are critically ambiguous.

## Git Push Policy
- `main` is branch-protected in this repo (PR required, `guard` CI check required). An admin
  account can bypass this, but bypassing is still a deliberate exception, not a default.
- Always ask for explicit confirmation before pushing directly to `main` — every time, not just
  once per session, and regardless of whether the change is application code, config, or docs.
  Prefer a PR unless the user explicitly says to push directly.

## Steering Configuration
- Load entire `.kiro/steering/` as project memory
- Default files: `product.md`, `tech.md`, `structure.md`
- Custom files are supported (managed via `/kiro-steering-custom`)
