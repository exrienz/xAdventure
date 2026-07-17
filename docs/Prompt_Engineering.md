# Prompt Engineering Guide

This document describes how the Infinite Narrative Engine constructs prompts for the underlying LLM.

## Overview

The engine builds a system prompt plus a rolling context window of recent story turns. The LLM returns structured JSON containing the next story segment, four choices, and state updates.

## System Prompt Structure

The system prompt consists of:

1. **Base identity**: The LLM acts as a serialized light-novel author and interactive fiction Game Master.
2. **Output schema**: A strict JSON schema is provided so the LLM returns parseable responses.
3. **Style rules**: Simple English, third-person limited POV, dialogue, hooks, and internal monologue formatting.
4. **Genre context**: A short description of the selected genre and protagonist archetype.
5. **Strict enforcement block (v2)**: When the feature flag `novel_gen_v2_prompt_fix` is enabled, the prompt explicitly forbids genre drift and lists forbidden tropes for each genre.
6. **Entity consistency block (v2)**: The prompt includes a "Known Characters" summary and instructs the LLM to keep relationships, gender pronouns, and roles consistent.

## Feature Flag: `novel_gen_v2_prompt_fix`

Enable by setting the environment variable:

```
FEATURE_FLAGS=novel_gen_v2_prompt_fix
```

When enabled:
- Genre enforcement rules are injected into the system prompt.
- Entity consistency rules are injected.
- The LLM is instructed to add/update characters via `add_entities` and `update_entities` in `state_update`.

When disabled:
- Only the original generic genre/archetype descriptions are used.
- The API response structure remains unchanged.

## Feature Flag: `enhanced_narrative_logic`

Enable for a percentage of traffic with the rollout variable:

```
FEATURE_ROLLOUT=enhanced_narrative_logic=10
```

Or fully enable with:

```
FEATURE_FLAGS=enhanced_narrative_logic
```

When enabled:
- The system prompt receives an extra block requiring cause-and-effect chain-of-thought before writing.
- The dramatic arc (Introduction, Rising Action, Climax, Resolution) is enforced via per-turn reminders.
- Character behavior must remain consistent with established traits and relationships.

When disabled:
- The original prompt behavior is preserved.
- API response structure remains unchanged.

## Feature Flag: `enable_dynamic_kids_stories`

Enable by setting the environment variable:

```
FEATURE_FLAGS=enable_dynamic_kids_stories
```

When enabled for kids genres (`Kids`, `Misteri Dongeng`, `Sains Kanak-kanak`, `Pengembaraan Haiwan`, `Dongeng Ajaib`, `Persahabatan`, `Sukan Ceria`, `Kehidupan Sekolah`):
- The `age` input is validated as an integer in the range `3-12`.
- Invalid or missing ages are clamped to the lowest safe tier for kids genres.
- Age `4-5` maps to the `4-5` tier with a hard cap of `20` words and very simple vocabulary.
- Age `6-7` maps to the `6-7` tier with a hard cap of `40` words and simple sentences.
- Age `8+` maps to the `8+` tier with a hard cap of `80` words and slightly richer vocabulary.
- The prompt enforces strict standard Bahasa Malaysia only and rejects Indonesian vocabulary, slang, and syntax.
- Logs include `age`, `age_group`, `word_count`, and an alert event when Indonesian markers are detected.

When disabled:
- Kids genres still use the strict Bahasa Malaysia prompt.
- The dynamic age-based word bounds are injected into the prompt and enforced server-side after generation.
- The API response structure remains unchanged.

## Bahasa Malaysia Strict Prompt for Kids

Kids mode uses this localization block:

```text
Style Rules (Strict Bahasa Malaysia for Kids):
- Write the ENTIRE story_text and choices in standard Bahasa Malaysia only.
- Use standard Malaysian Malay vocabulary and spelling. Do NOT use Indonesian dialect, slang, or syntax.
- Forbidden Indonesian markers include: gak, nggak, banget, gue, lu, ngapain, mau, uang, sepeda, apa kabar, rumah sakit, bego, jelek, dong, sih.
- Use Malaysian equivalents: tidak/tak, sangat, saya/aku/kami, mahu, wang, basikal, apa khabar, hospital, bodoh, buruk.
- Keep the story child-safe, warm, playful, and educational.
- Do NOT include horror, explicit violence, romance, adult themes, or scary imagery.
- End every turn with a simple, age-appropriate hook or question.
```

Dynamic age instruction when `enable_dynamic_kids_stories` is enabled:

```text
Use the <age_tier> age tier: keep story_text between <min_words> and <max_words> words.
```

## Genre Enforcement

Each genre has a dedicated description in `internal/service/twist.go`. Under v2, the prompt forbids the LLM from mixing in tropes from unrelated genres. Example:

- **Romance**: must focus on relationships, feelings, attraction, heart-conflict. No shadow monsters, dungeon crawling, or epic quests unless directly tied to the romance plot.
- **Cyberpunk**: must focus on megacorps, hackers, neon cities, implants. No magic, dragons, or medieval settings.
- **Horror**: must focus on dread, survival, and the unknown. No heroic adventure party tropes.

## Entity Consistency

The engine extracts and tracks entities in the `GameState.Entities` map. Each entity has:

- `name`: Character name.
- `relation_to_pc`: Relationship to the protagonist (e.g., "my brother", "rival").
- `gender`: Gender for pronoun consistency.
- `role`: Narrative role (e.g., "mentor", "villain").
- `status`: Current state (e.g., "alive", "missing").
- `traits`: List of personality tags.

The engine builds a "Known Characters" summary and injects it into every turn prompt. The LLM is instructed not to contradict this summary.

## Temperature and Top-P

Randomness is controlled by environment variables:

- `LLM_TEMPERATURE`: default `0.85` (must be > 0.7 for creativity).
- `LLM_TOP_P`: default `0.95`.

These values are passed to the LLM API on every chat completion call.

## State Update Schema

The LLM returns a `state_update` object with these fields:

```json
{
  "health_delta": 0,
  "inventory_add": [],
  "inventory_remove": [],
  "stats_delta": {},
  "bonds_delta": {},
  "karma_delta": 0,
  "fate_points_delta": 0,
  "reputation_delta": {},
  "add_flags": [],
  "remove_flags": [],
  "add_entities": [],
  "update_entities": []
}
```

## Name Generation

The engine now uses a dedicated `NameGenerator` to seed the LLM with a genre-appropriate supporting-character name. Names are chosen from curated pools per genre, checked against the last 50 generated names, filtered for profanity/slurs, and logged with a seed. The LLM may use the suggested name or invent a new one in the same style.

## Safety

A lightweight `SafetyFilter` checks generated text against a blocked-word list. If a match is found, the word is masked with asterisks. This is a fallback; provider-side moderation remains the primary defense.
