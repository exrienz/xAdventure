
## 2.7.0

- Enhanced `/kids` as an interactive storybook behind `kids_storybook_v2`.
- Added reusable character profiles via `state.entities`.
- Added sanitized image prompt and local image URL fields.
- Enforced age-specific kids story word caps and latest-segment-only rendering.

# Changelog

All notable changes to the Infinite Narrative Engine project will be documented in this file.

## [2.7.0] - 2026-06-26

### Added
- Added feature to configure a dedicated LLM model for the /kids section via the `KIDS_LLM_MODEL` environment variable.

## [2.6.0] - 2026-06-20

### Added
- Added syllable color-coding feature for early readers (4-5yo) in Bahasa Malaysia stories. Implemented text length constraints for age-appropriate reading.

## [2.5.1] - 2026-06-19

### Added
- Updated Kids Story Mode to support age-based story length, new genres, and strict Bahasa Malaysia localization.
- Added error retry button to Kids Story Mode and added main page navigation link.
- Added `age`, `age_group`, and `word_count` to Kids Story API responses.
- Added frontend analytics hooks for `kids_story_retry_clicked` and `kids_story_render_error`.

### Changed
- Kids age validation now accepts integer ages from 3 to 12.
- Kids genres now include `Misteri Dongeng`, `Sains Kanak-kanak`, `Pengembaraan Haiwan`, `Dongeng Ajaib`, `Persahabatan`, `Sukan Ceria`, and `Kehidupan Sekolah`.
- Kids Story prompts now enforce standard Bahasa Malaysia and reject Indonesian vocabulary, slang, and syntax.
- Generation metrics now log age, age group, word count, and Indonesian-language alert markers.

### Security
- Kids Story Mode now clamps invalid ages to the lowest tier and sanitizes error text in the browser.
- Retry action is client-side debounced for 2 seconds.

## [2.4.0] - 2026-06-13

### Added
- Added Kids Interactive Story Reader with Bahasa Malaysia syllable highlighting and branching narrative support.
- New route `/kids` protected by `kids_mode` feature flag.
- Age restriction (4-8 years) for Kids mode enforced at API layer.
- `syllable` module to correctly parse and color-code (alternating black/red) Bahasa Malaysia syllables.

## [2.3.0] - 2026-06-13

### Fixed
- Fixed genre consistency issue where stories would default to dark themes regardless of user selection. Improved archetype filtering.

### Added
- Genre-Archetype Compatibility Matrix in `internal/domain/game.go`.
- `strict_genre_enforcement` feature flag for strict checking of archetypes based on genre compatibility.
- Twist logic adjusted to prevent explicit dark and horrific twists in non-dark genres like Romance or Comedy.

## [2.2.0] - 2026-06-13

### Added
- `NameGenerator` producing genre-appropriate character names with a 50-name history buffer and per-request deduplication.
- `enhanced_narrative_logic` feature flag with percentage-based rollout support (`FEATURE_ROLLOUT=enhanced_narrative_logic=10`).
- Enhanced system prompt enforcing narrative arc (Introduction, Rising Action, Climax, Resolution), cause-and-effect, and character consistency.
- Chain-of-thought constraints requiring the LLM to outline cause-and-effect before writing each scene.
- Name generation seed logging and `unique_name_rate` metric for observability.
- Expanded `SafetyFilter` with common profanity and slur list for generated names.
- `IsValidNameRequest` input validation to block prompt-injection patterns in user names.

### Changed
- `getSystemPrompt` now accepts an `enhanced` flag to inject narrative logic rules.
- `FeatureEnabled` supports both boolean flags and percentage rollouts.

### Security
- Sanitized user-supplied protagonist names; invalid names fall back to "Traveler".
- Generated names pass the safety filter before being recorded or suggested.

## [1.3.0] - 2026-06-13

### Fixed
* Fixed novel generation to strictly adhere to selected genres, maintain character entity consistency, and improve narrative randomness.

### Added
- Genre allowlist validation and strict genre enforcement in system prompts.
- Character/entity tracking with consistency injection into every LLM turn.
- Configurable LLM temperature and top_p parameters (defaults: 0.85 / 0.95).
- Feature flag `novel_gen_v2_prompt_fix` for A/B testing the new prompt logic.
- Lightweight output safety filter for NSFW content.
- Structured logging of genre drift keywords and entity consistency metrics.

### Changed
- System prompt now explicitly forbids genre drift and entity contradictions when feature flag is enabled.
- API response structure remains unchanged; only internal prompt logic modified.

## [1.2.0] - 2026-06-13

### Added
- Light novel style generation with simple English output.
- Deterministic chapter numbering tracked server-side.
- Expanded genre and archetype options.
- Plot twist engine for unpredictable narrative beats.
- Story seed input for replayability.

### Fixed
- Cache staleness issues on mobile browsers by adding no-cache headers and self-healing reload logic.
- Removed user choice mentions from downloaded tale transcript for seamless reading.

## [1.1.0] - 2026-06-13

### Added
- Repository interface for testability.
- Session-level mutex to prevent race conditions.
- Graceful shutdown and structured logging.
- Health check endpoint and CORS middleware.

### Changed
- Server-side storage of current choices instead of client-controlled choices.

## [1.0.0] - 2026-06-13

### Added
- Initial release of the Infinite Narrative Engine.
- Character creation, LLM-driven game master loop, SQLite persistence, and web UI.
