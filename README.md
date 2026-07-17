

## Kids interactive storybook

Enable the enhanced `/kids` storybook with:

```env
FEATURE_FLAGS=novel_gen_v2_prompt_fix,kids_mode,kids_storybook_v2
```

The API returns `character_profiles`, `image_prompt`, and `image_url` for each kids segment. The image URL points to local `/api/kids/image`, which proxies `OPENAI_API_BASE/images/generations?response_format=binary` using `OPENAI_IMAGE_MODEL`. Disable `kids_storybook_v2` to roll back to the previous kids flow.

Image prompts use stable `character_profiles` appearance details plus a short scene action, not the full story text, so generated characters remain visually consistent across turns.
