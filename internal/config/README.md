# Configuration

This package manages the environment variable configuration for the application.

## Model Selection Logic

The LLM Service supports conditional model selection based on the route context.

*   `OPENAI_MODEL` (mapped to `LLM_MODEL` in documentation context): The default model used for all standard inference requests.
*   `KIDS_LLM_MODEL`: An optional secondary model identifier used exclusively for requests originating from the `/kids` section (identified by the Kids genre).

**Selection Rules:**
1.  If a request is for the `/kids` section (Kids genre is selected) and `KIDS_LLM_MODEL` is defined and non-empty in the `.env` file, the LLM client will route the request to the `KIDS_LLM_MODEL`.
2.  If `KIDS_LLM_MODEL` is not defined, or if the request is for any other section/genre, the system defaults to using `OPENAI_MODEL`.

This mechanism allows for cost optimization and tailored safety tuning for younger demographics while utilizing the same underlying API provider configuration (Base URL and API Key).
