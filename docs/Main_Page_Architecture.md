# Main Page Architecture

## Navigation

The main page includes a direct navigation link labeled **Cerita Kanak-kanak** in the hero/header section.

```text
Header / Hero
- Title: Infinite Narrative Engine
- Subtitle: Forge your own serialized light novel.
- Link: Cerita Kanak-kanak -> /kids
```

## API Response Additions

The `POST /api/turn` response includes a `syllable_split` parameter containing the HTML representation of the story with syllable color-coding for early readers (4-5yo).

## Routing

- Main page: `/`
- Kids Story Mode: `/kids`
- Kids API start endpoint: `POST /api/start`
- Kids API turn endpoint: `POST /api/turn`

## Error Recovery

Kids Story Mode uses client-side retry behavior:

1. If `/api/start` or `/api/turn` fails, the UI shows an error panel with **Cuba Lagi**.
2. Clicking **Cuba Lagi** reuses the previous request payload or selected choice.
3. The retry button is debounced for 2 seconds.
4. Error text is inserted with `textContent`, not `innerHTML`, to avoid XSS from raw API payloads.

## Analytics Hooks

Frontend dispatches local browser events for monitoring integration:

- `kids_story_retry_clicked`
- `kids_story_render_error`

These events can be forwarded to Sentry, DataDog, or another monitoring dashboard by the hosting environment.
