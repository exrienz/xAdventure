

## Kids interactive storybook

## Hot reload

`docker compose up --build` now runs the app through `air` inside the container.
Any changes to Go files, templates, or static assets under `web/static` will trigger
an automatic rebuild/restart, so you can refresh the browser without rebuilding the
image manually.

Enable the enhanced `/kids` storybook with:

```env
FEATURE_FLAGS=novel_gen_v2_prompt_fix,kids_mode,kids_storybook_v2
```

The API returns `character_profiles`, `image_prompt`, and `image_url` for each kids segment. Disable `kids_storybook_v2` to roll back to the previous kids flow.

Provider split:
- Default text LLM uses `OPENAI_API_BASE`, `OPENAI_API_KEY`, and `OPENAI_MODEL`.
- Kids text LLM can use a separate openai-compatible provider via `KIDS_LLM_API_BASE`, `KIDS_LLM_API_KEY`, and `KIDS_LLM_MODEL`. If any of those are missing, kids text falls back to the default provider.
- Kids image generation uses a separate openai-compatible image provider via `IMAGEROUTER_API_BASE`, `IMAGEROUTER_API_KEY`, and `IMAGEROUTER_MODEL`.

The kids image URL points to local `/api/kids/image`, which proxies `POST {IMAGEROUTER_API_BASE}/images/generations` using the dedicated ImageRouter credentials. The proxy sends the prompt with `quality=low`, `size=1024x1024`, and `output_format=jpeg`.

Image prompts use stable `character_profiles` appearance details plus a short scene action, not the full story text, so generated characters remain visually consistent across turns.
