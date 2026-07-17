# Configuration

This package manages the environment variable configuration for the application.

## Model Selection Logic

The LLM Service supports conditional model selection based on the route context.

*   `OPENAI_API_BASE`, `OPENAI_API_KEY`, `OPENAI_MODEL`: The default openai-compatible provider used for all standard inference requests.
*   `KIDS_LLM_API_BASE`, `KIDS_LLM_API_KEY`, `KIDS_LLM_MODEL`: An optional secondary openai-compatible provider used exclusively for requests originating from the `/kids` section (identified by the Kids genre).
*   `IMAGEROUTER_API_BASE`, `IMAGEROUTER_API_KEY`, `IMAGEROUTER_MODEL`: A dedicated openai-compatible image provider used only by `/api/kids/image`.

**Selection Rules:**
1.  All non-kids text requests use the default `OPENAI_*` provider.
2.  Kids text requests use the dedicated kids provider only when `KIDS_LLM_API_BASE`, `KIDS_LLM_API_KEY`, and `KIDS_LLM_MODEL` are all defined and non-empty.
3.  If the kids provider is incomplete, kids text requests fall back to the default `OPENAI_*` provider.
4.  Kids image requests use the dedicated ImageRouter provider only when `IMAGEROUTER_API_BASE`, `IMAGEROUTER_API_KEY`, and `IMAGEROUTER_MODEL` are all defined and non-empty. Otherwise `/api/kids/image` returns `503`.

This split allows default text, kids text, and kids image generation to use three separate provider lanes while keeping every integration openai-compatible.
