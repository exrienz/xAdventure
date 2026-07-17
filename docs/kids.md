# Kids Story Module

The `/kids` module is a TV-friendly interactive storybook for children.

## Interactive Storybook v2

Enable with `kids_storybook_v2` in `FEATURE_FLAGS`. Roll back by removing the flag.

When enabled:

- A main character profile is created when the story starts.
- Side characters introduced by the LLM are stored in `state.entities`.
- Profiles include name, relationship, gender, role, status, and traits. Appearance notes are kept in traits when provided.
- Existing character profiles are sent back to the model every turn to keep names, traits, roles, and relationships consistent.
- API responses include `character_profiles`, `image_prompt`, and `image_url`.
- `image_url` uses local `/api/kids/image?prompt=...`, which POSTs to the configured OpenAI-compatible image generation endpoint and returns PNG bytes.
- The browser displays only the latest story segment after each action.

## Age limits

Server-side word caps are enforced after generation:

- Age 4-5: max 20 words, very simple vocabulary
- Age 6-7: max 40 words, simple sentences
- Age 8+: max 80 words, slightly richer vocabulary

## Image prompt logic

Image prompts are built from stable character profiles plus a short scene-action summary. They do **not** dump the full story text into the image URL.

- Character profiles include `appearance` for stable visual continuity: hair length/style, clothing colors, shoes, and accessories.
- Side characters use the same `appearance` field so recurring animals/friends stay visually consistent.
- Scene action is reduced to the first short visual beat only.
- HTML tags, script/style blocks, and unsafe punctuation are removed before URL generation.
- Prompts are child-safe, do not include user secrets, and are URL-query escaped before the backend calls the image provider. The browser never receives the provider API key.
