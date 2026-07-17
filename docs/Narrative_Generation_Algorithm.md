# Narrative Generation Algorithm

## Overview

The narrative generation engine is a Go service that drives an interactive serialized light novel. It combines a deterministic game-state tracker, a stochastic name generator, genre/archetype prompts, an LLM client, and a safety filter. The engine is designed to run in a self-hosted container with minimal dependencies.

## Components

### 1. Name Generation

The `NameGenerator` (`internal/service/namegen.go`) produces supporting-character names:

- Maintains a rolling history of the last **50** generated names.
- Generates names from genre-curated pools (first names, surnames, epithets).
- Avoids exact repetition within the history and within the current session run.
- Applies the `SafetyFilter` to reject profanity/slurs.
- Records a seed with every generated name for debugging.

A sample genre-appropriate name for **Cyberpunk** might be `Nova Vance the Glitch`, while a **Steampunk** name might be `Emmeline Brasswell the Inventive`.

### 2. Randomness Seed Logic

Each `NameGenerator.Generate` call derives a seed from the current Unix nanosecond plus a small crypto-random jitter. The seed is logged as `name_seed` in the generation metrics so repetition issues can be traced.

### 3. Story Prompt Construction

The engine builds a system prompt plus a rolling context window of recent turns. The prompt contains:

1. **Base identity and JSON schema**: what the LLM is and the exact response shape.
2. **Style rules**: simple-English light-novel style.
3. **Genre/archetype context**: descriptions from `twist.go`.
4. **Entity consistency block**: known characters, their relationships, and traits.
5. **Enhanced narrative logic block** (when `enhanced_narrative_logic` is enabled):
   - Cause-and-effect chain-of-thought constraint.
   - Dramatic arc guidance: Introduction → Rising Action → Climax → Resolution.
   - Consistency rules for returning characters.
6. **Per-turn arc reminder**: brief reminder of the current narrative phase based on turn number.

### 4. Feature Flags and Rollout

Flags are loaded from `FEATURE_FLAGS` and `FEATURE_ROLLOUT` env vars.

- `FEATURE_FLAGS=enhanced_narrative_logic` enables the flag for 100% of traffic.
- `FEATURE_ROLLOUT=enhanced_narrative_logic=10` enables it for ~10% of traffic based on the current second.

Use `FEATURE_ROLLOUT` for safe, gradual releases. Set it to `0` to disable or `100` to fully enable.

### 5. Observability

Each generation logs:

- `turn`, `genre`
- `drift_keywords_found` and `entity_count`
- `v2_enabled`, `enhanced_narrative_logic`
- `name`, `name_seed`, `name_unique`
- `unique_name_rate`

### 6. Safety and Input Validation

- `IsValidNameRequest` rejects empty names, overly long names, and prompt-injection patterns (`ignore`, `disregard`, `system prompt`, markdown fences, URLs, etc.). Invalid names fall back to a procedurally generated safe name.
- `SafetyFilter` masks NSFW story content and rejects profane/slur names before they are recorded.

## API

`POST /api/start` accepts:

```json
{
  "name": "NamaWatak",
  "age": 4,
  "gender": "Lelaki",
  "genre": "Misteri Dongeng",
  "archetype": "Hero",
  "seed": "optional-safe-seed"
}
```

- `name`, `gender`, `genre`, and `archetype` remain required.
- `age` is optional. Missing ages use the default tier for the selected genre.
- Kids genres accept `age` as an integer from `3` to `12`.
- Invalid kids ages are clamped to the lowest tier (`3`) so the request does not fail.
- Standard genres accept `age` as an integer from `10` to `150`.
- Supported kids genres: `Misteri Dongeng`, `Sains Kanak-kanak`, `Pengembaraan Haiwan`, `Dongeng Ajaib`, `Persahabatan`, `Sukan Ceria`, `Kehidupan Sekolah`, and legacy `Kids`.

Response now includes:

```json
{
  "session_id": "...",
  "story": "...",
  "choices": ["..."],
  "status": "active",
  "chapter_title": "Bab 1",
  "age": 4,
  "age_group": "4-5",
  "word_count": 72
}
```

`POST /api/turn` still accepts:

```json
{
  "session_id": "...",
  "choice_index": 0
}
```

and returns the same response fields plus `age`, `age_group`, and `word_count`.

## Rollback

To roll back the enhanced logic, unset or disable the flag:

```bash
FEATURE_FLAGS=enhanced_narrative_logic=false
```

or set rollout to 0:

```bash
FEATURE_ROLLOUT=enhanced_narrative_logic=0
```

The API response structure remains unchanged, so frontend clients are unaffected.
