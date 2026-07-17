package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/muz/xadventure/internal/config"
	"github.com/muz/xadventure/internal/domain"
	"github.com/muz/xadventure/internal/llm"
	"github.com/muz/xadventure/internal/repository"
	"github.com/muz/xadventure/internal/service/syllable"
)

type Engine struct {
	repo          repository.Repository
	llmClient     *llm.Client
	kidsLLMClient *llm.Client
	twistEngine   *TwistEngine
	safety        *SafetyFilter
	nameGenerator *NameGenerator
	cfg           *config.Config
	mu            sync.RWMutex
	sessionMu     map[string]*sync.Mutex
}

func NewEngine(repo repository.Repository, llmClient *llm.Client, kidsLLMClient *llm.Client, cfg *config.Config) *Engine {
	safety := NewSafetyFilter()
	return &Engine{
		repo:          repo,
		llmClient:     llmClient,
		kidsLLMClient: kidsLLMClient,
		twistEngine:   NewTwistEngine(60),
		safety:        safety,
		nameGenerator: NewNameGenerator(50, safety),
		cfg:           cfg,
		sessionMu:     make(map[string]*sync.Mutex),
	}
}

func (e *Engine) getSessionLock(id string) *sync.Mutex {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.sessionMu[id] == nil {
		e.sessionMu[id] = &sync.Mutex{}
	}
	return e.sessionMu[id]
}

func (e *Engine) GetSession(ctx context.Context, id string) (*domain.Session, error) {
	return e.repo.GetSession(ctx, id)
}

func (e *Engine) textClientForGenre(genre string) *llm.Client {
	if domain.IsKidsGenre(genre) && e.kidsLLMClient != nil {
		return e.kidsLLMClient
	}
	return e.llmClient
}

func (e *Engine) GenerateStoryText(ctx context.Context, sessionID string) (string, error) {
	session, err := e.repo.GetSession(ctx, sessionID)
	if err != nil || session == nil {
		return "", fmt.Errorf("session not found")
	}

	logs, err := e.repo.GetStoryLogs(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch logs: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: A Tale of %s\n", session.Genre, session.UserName))
	sb.WriteString(fmt.Sprintf("Archetype: %s | Seed: %s\n\n", session.Archetype, session.Seed))

	for i, log := range logs {
		ch := i/6 + 1
		if i%6 == 0 {
			sb.WriteString(fmt.Sprintf("\nChapter %d\n", ch))
		}
		if i > 0 && i%6 == 0 {
			sb.WriteString("***\n\n")
		}
		sb.WriteString(log.Content + "\n\n")
	}

	if session.Status == "game_over" {
		if domain.IsKidsGenre(session.Genre) {
			sb.WriteString("\n\n--- TAMAT ---")
		} else {
			sb.WriteString("\n\n--- THE END ---")
		}
	}

	return sb.String(), nil
}

func (e *Engine) StartSession(ctx context.Context, req *domain.StartRequest) (*domain.Session, *domain.StoryLog, []string, error) {
	seed := sanitize(req.Seed)
	if seed == "" {
		seed = GenerateSeed()
	}

	// Validate and sanitize user-supplied name to prevent prompt injection.
	userName := sanitize(req.Name)
	if !IsValidNameRequest(userName) {
		userName = FallbackName(req.Genre)
	}

	// Archetype is optional for kids genres; default to Hero.
	archetype := sanitize(req.Archetype)
	if archetype == "" {
		archetype = "Hero"
	}

	// Reset per-run dedup and generate a genre-appropriate supporting name.
	e.nameGenerator.Reset()
	generated := e.nameGenerator.Generate(req.Genre)

	sessionID := uuid.New().String()
	age := req.AgeOrDefault(domain.KidsDefaultAge)
	if !domain.IsKidsGenre(req.Genre) {
		age = req.AgeOrDefault(21)
	}
	session := &domain.Session{
		ID:             sessionID,
		UserName:       userName,
		Genre:          sanitize(req.Genre),
		Age:            age,
		Gender:         sanitize(req.Gender),
		Archetype:      archetype,
		Seed:           seed,
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		CurrentChoices: []string{},
		State: domain.GameState{
			Health:        100,
			MaxHealth:     100,
			Inventory:     []string{},
			Stats:         map[string]int{},
			Bonds:         map[string]int{},
			Karma:         0,
			FatePoints:    1,
			Reputation:    map[string]int{},
			Flags:         []string{},
			Archetype:     archetype,
			PlotTwists:    0,
			ChapterNumber: 1,
			Entities: map[string]domain.Entity{
				strings.ToLower(userName): {
					Name:         userName,
					RelationToPC: "main character",
					Gender:       sanitize(req.Gender),
					Role:         "protagonist",
					Status:       "active",
					Appearance:   defaultKidsAppearance(sanitize(req.Gender)),
					Traits:       []string{"curious", "kind", "brave"},
				},
			},
		},
	}

	if err := e.repo.CreateSession(ctx, session); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	enhanced := e.cfg.FeatureEnabled("enhanced_narrative_logic")
	systemPrompt := e.getSystemPrompt(req.Genre, archetype, age, enhanced)
	openingScene := OpeningScene()

	nameHint := ""
	if generated.Name != "" {
		nameHint = fmt.Sprintf("Suggested NPC name style for this genre: %s. You may use it for an early supporting character or invent a new one in the same style.", generated.Name)
	}

	languageInstruction := ""
	pageInstruction := ""
	if domain.IsKidsGenre(req.Genre) {
		languageInstruction = "Language instruction: write every word in standard Bahasa Malaysia only. Do not use Indonesian vocabulary, slang, or syntax. Create a consistent main character profile and 1-2 supporting side character profiles using state_update.add_entities. For every character, set appearance with stable visual details: hair length/style, shirt/top color, trousers/skirt, shoes, and any distinctive accessory. IMPORTANT: write all appearance values in ENGLISH (visual description only, no story text). Reuse the same appearance exactly when characters reappear. Also set state_update.visual_setting to a concise ENGLISH reusable storybook background profile for this story world, covering the place, lighting, important props, and atmosphere. Keep that visual setting stable across later pages unless the story truly moves to a new place. Also provide image_scene: a concise ENGLISH visual description of what is happening in this scene right now (characters present, their poses/expressions, setting, mood). Example: \"A young girl with shoulder-length dark hair, yellow shirt, and blue skirt kneels beside a small brown bird on a green school field. Morning sunlight. Cheerful mood.\""
		pageInstruction = KidsPageInstruction(1, session.Age)
	}

	userMsg := fmt.Sprintf(`Begin a new serialized light novel.
Protagonist: %s, %d years old, %s.
Archetype: %s (%s).
Genre: %s (%s).
Story seed: %s.
Opening style: %s
%s
%s
%s

Introduce the protagonist, set the scene, include one line of dialogue if another being is present, and provide 4 compelling choices. End the scene on a hook.
Current State: %s`,
		session.UserName, session.Age, session.Gender,
		session.Archetype, ArchetypeDescription(session.Archetype),
		session.Genre, GenreDescription(session.Genre),
		session.Seed,
		BuildOpening(openingScene),
		nameHint,
		languageInstruction,
		pageInstruction,
		stateJSON(session.State),
	)

	textClient := e.textClientForGenre(session.Genre)
	if domain.IsKidsGenre(session.Genre) && textClient == e.kidsLLMClient {
		slog.Info("kids_model_selected", "session_id", session.ID, "model", textClient.Model, "provider_base", textClient.BaseURL)
	}

	llmResult, err := textClient.GenerateTurnWithModel(ctx, []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMsg},
	}, "")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("LLM error on start: %w", err)
	}

	llmResult.Response.StoryText = e.safety.Sanitize(llmResult.Response.StoryText)

	if domain.IsKidsGenre(session.Genre) {
		// Post-generation Bahasa Malaysia enforcement for kids stories via LLM self-review.
		llmResult.Response.StoryText, llmResult.Response.Choices = e.reviewBahasaMalaysia(ctx, textClient, llmResult.Response.StoryText, llmResult.Response.Choices)
		if e.cfg.FeatureEnabled("kids_storybook_v2") {
			llmResult.Response.StoryText = EnforceKidsWordLimit(llmResult.Response.StoryText, session.Age)
		}
	}

	slog.Info("debug_coloring", "genre", session.Genre, "isKids", domain.IsKidsGenre(session.Genre), "age", session.Age, "feature", e.cfg.FeatureEnabled("syllable_coloring"))
	var colorCodedStory string
	if domain.IsKidsGenre(session.Genre) && session.Age >= 4 && session.Age <= 5 && e.cfg.FeatureEnabled("syllable_coloring") {
		slog.Info("color_mode_activated", "session_id", sessionID, "age", session.Age)
		colorCodedStory = syllable.FormatSentenceWithColors(llmResult.Response.StoryText, "#FF0000", "#000000")
		for i, choice := range llmResult.Response.Choices {
			llmResult.Response.Choices[i] = syllable.FormatSentenceWithColors(choice, "#FF0000", "#000000")
		}
	} else if domain.IsKidsGenre(session.Genre) && e.cfg.FeatureEnabled("kids_mode") {
	}

	session.CurrentChoices = llmResult.Response.Choices
	e.applyStateDelta(&session.State, llmResult.Response.StateUpdate)
	e.updateEntities(&session.State, llmResult.Response.StateUpdate)

	if err := e.repo.UpdateSession(ctx, session); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to update session state: %w", err)
	}

	log := &domain.StoryLog{
		SessionID:         sessionID,
		TurnNumber:        1,
		Content:           llmResult.Response.StoryText,
		ColorCodedContent: colorCodedStory,
		ChapterTitle:      "",
		ChoiceMade:        "",
		ImageScene:        llmResult.Response.ImageScene,
		Timestamp:         time.Now(),
	}

	if err := e.repo.AppendStoryLog(ctx, log); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to save story log: %w", err)
	}

	e.logMetrics(session.Genre, llmResult.Response.StoryText, session.State.Entities, 1, generated, session.Age, session.Status, textClient.Model)

	return session, log, llmResult.Response.Choices, nil
}

func (e *Engine) ProcessTurn(ctx context.Context, sessionID string, choiceIndex int) (*domain.Session, *domain.StoryLog, []string, error) {
	lock := e.getSessionLock(sessionID)
	lock.Lock()
	defer lock.Unlock()

	session, err := e.repo.GetSession(ctx, sessionID)
	if err != nil || session == nil {
		return nil, nil, nil, fmt.Errorf("session not found")
	}

	if session.Status != "active" {
		return nil, nil, nil, fmt.Errorf("session is no longer active")
	}

	if choiceIndex < 0 || choiceIndex >= len(session.CurrentChoices) {
		return nil, nil, nil, domain.ErrInvalidChoice
	}
	selectedChoiceText := session.CurrentChoices[choiceIndex]

	logs, err := e.repo.GetStoryLogs(ctx, sessionID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load logs: %w", err)
	}

	turnNumber := len(logs) + 1

	twistLevel, twistInstruction := e.twistEngine.RollTwist(turnNumber, session.State.PlotTwists, session.Genre)
	twistPrefix := ""
	if twistLevel != domain.TwistNone {
		twistPrefix = fmt.Sprintf("Plot twist directive: %s\n\n", twistInstruction)
		session.State.PlotTwists++
	}

	spontaneousTwist := "You are also free to introduce your own surprising beat, reversal, or revelation whenever it serves the drama."

	enhanced := e.cfg.FeatureEnabled("enhanced_narrative_logic")
	systemPrompt := e.getSystemPrompt(session.Genre, session.Archetype, session.Age, enhanced)

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
	}

	startIdx := 0
	if len(logs) > 5 {
		startIdx = len(logs) - 5
	}

	for i := startIdx; i < len(logs); i++ {
		messages = append(messages, llm.Message{
			Role:    "assistant",
			Content: logs[i].Content,
		})
		if logs[i].ChoiceMade != "" {
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: fmt.Sprintf("I chose: %s", logs[i].ChoiceMade),
			})
		}
	}

	chapterInstruction := ""
	if domain.IsKidsGenre(session.Genre) {
		chapterInstruction = KidsPageInstruction(turnNumber, session.Age)
	} else if turnNumber%6 == 0 {
		chapterInstruction = "This scene ENDS the current chapter. Wrap up the chapter with a strong cliffhanger. Do NOT add any chapter title text in the story_text. The system will add the chapter heading."
	}

	entitySummary := e.entitySummary(session.State.Entities)

	// Build a concise narrative arc reminder for enhanced mode.
	// Kids stories have their own arc built into chapterInstruction, so skip.
	arcInstruction := ""
	if enhanced && !domain.IsKidsGenre(session.Genre) {
		arcInstruction = e.arcInstruction(turnNumber)
	}

	languageInstruction := ""
	if domain.IsKidsGenre(session.Genre) {
		languageInstruction = "Language instruction: continue in standard Bahasa Malaysia only. Do NOT use Indonesian vocabulary, slang, or syntax. If you accidentally use an Indonesian word, replace it with the Malaysian equivalent immediately. Reuse Known Characters exactly: keep each character name, appearance, role, and traits consistent. If a new side character appears, add a complete reusable profile in state_update.add_entities. IMPORTANT: write all appearance values in ENGLISH (visual description only). Reuse the Known Visual Setting exactly when the story is still in the same place. Only change state_update.visual_setting when the story genuinely moves to a new location, and then describe the new stable setting in concise ENGLISH. Also provide image_scene: a concise ENGLISH visual description of what is happening in this scene right now (characters present, their poses/expressions, setting, mood)."
	} else {
		languageInstruction = "Use simple, clear English with short sentences."
	}

	userMsg := fmt.Sprintf(`%sI choose: %s

Current State: %s
Active Flags: %s
Known Characters: %s
Known Visual Setting: %s

Continue the story from this choice. %s Advance the plot, reveal character, escalate tension appropriately for the selected genre and age. %s
%s
%s
Generate the next scene, provide 4 fresh choices, and return state updates.
Do NOT include any chapter number or title in the story_text. The system handles that.
Do NOT contradict the known characters or their relationships listed above.`,
		twistPrefix,
		selectedChoiceText,
		stateJSON(session.State),
		strings.Join(session.State.Flags, ", "),
		entitySummary,
		e.visualSettingSummary(session.State.VisualSetting),
		languageInstruction,
		spontaneousTwist,
		chapterInstruction,
		arcInstruction,
	)
	messages = append(messages, llm.Message{Role: "user", Content: userMsg})

	textClient := e.textClientForGenre(session.Genre)
	if domain.IsKidsGenre(session.Genre) && textClient == e.kidsLLMClient {
		slog.Info("kids_model_selected", "session_id", session.ID, "model", textClient.Model, "provider_base", textClient.BaseURL)
	}

	llmResult, err := textClient.GenerateTurnWithModel(ctx, messages, "")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("LLM error on turn: %w", err)
	}

	llmResult.Response.StoryText = e.safety.Sanitize(llmResult.Response.StoryText)

	if domain.IsKidsGenre(session.Genre) {
		if turnNumber >= KidsStoryMinPages && llmResult.Response.StoryComplete {
			llmResult.Response.StoryText = e.polishKidsEnding(ctx, textClient, llmResult.Response.StoryText, session.Age, turnNumber)
		} else if turnNumber >= KidsStoryPageCount {
			llmResult.Response.StoryText = e.polishKidsEnding(ctx, textClient, llmResult.Response.StoryText, session.Age, turnNumber)
			llmResult.Response.StoryComplete = true
		}

		// Post-generation Bahasa Malaysia enforcement for kids stories via LLM self-review.
		llmResult.Response.StoryText, llmResult.Response.Choices = e.reviewBahasaMalaysia(ctx, textClient, llmResult.Response.StoryText, llmResult.Response.Choices)
		if e.cfg.FeatureEnabled("kids_storybook_v2") {
			llmResult.Response.StoryText = EnforceKidsWordLimit(llmResult.Response.StoryText, session.Age)
		}
	}

	slog.Info("debug_coloring", "genre", session.Genre, "isKids", domain.IsKidsGenre(session.Genre), "age", session.Age, "feature", e.cfg.FeatureEnabled("syllable_coloring"))
	var colorCodedStory string
	if domain.IsKidsGenre(session.Genre) && session.Age >= 4 && session.Age <= 5 && e.cfg.FeatureEnabled("syllable_coloring") {
		slog.Info("color_mode_activated", "session_id", sessionID, "age", session.Age)
		colorCodedStory = syllable.FormatSentenceWithColors(llmResult.Response.StoryText, "#FF0000", "#000000")
		for i, choice := range llmResult.Response.Choices {
			llmResult.Response.Choices[i] = syllable.FormatSentenceWithColors(choice, "#FF0000", "#000000")
		}
	} else if domain.IsKidsGenre(session.Genre) && e.cfg.FeatureEnabled("kids_mode") {
	}

	session.CurrentChoices = llmResult.Response.Choices
	e.applyStateDelta(&session.State, llmResult.Response.StateUpdate)
	e.updateEntities(&session.State, llmResult.Response.StateUpdate)
	if llmResult.Response.TwistAdded {
		session.State.PlotTwists++
	}

	if session.State.Health <= 0 {
		session.State.Health = 0
		session.Status = "game_over"
	} else if domain.IsKidsGenre(session.Genre) && turnNumber >= KidsStoryMinPages && llmResult.Response.StoryComplete {
		session.Status = "game_over"
	} else if domain.IsKidsGenre(session.Genre) && turnNumber >= KidsStoryPageCount {
		session.Status = "game_over"
	}

	if turnNumber%6 == 1 {
		session.State.ChapterNumber++
	}

	chapterTitle := e.chapterTitleForTurn(turnNumber, session.State.ChapterNumber)

	if err := e.repo.UpdateSession(ctx, session); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to update session state: %w", err)
	}

	log := &domain.StoryLog{
		SessionID:         sessionID,
		TurnNumber:        turnNumber,
		Content:           llmResult.Response.StoryText,
		ColorCodedContent: colorCodedStory,
		ChapterTitle:      chapterTitle,
		ChoiceMade:        selectedChoiceText,
		ImageScene:        llmResult.Response.ImageScene,
		Timestamp:         time.Now(),
	}

	if err := e.repo.AppendStoryLog(ctx, log); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to save story log: %w", err)
	}

	e.logMetrics(session.Genre, llmResult.Response.StoryText, session.State.Entities, turnNumber, GeneratedName{}, session.Age, session.Status, textClient.Model)

	return session, log, llmResult.Response.Choices, nil
}

func (e *Engine) applyStateDelta(state *domain.GameState, delta llm.StateDelta) {
	state.Health += delta.HealthChange
	if state.Health > state.MaxHealth {
		state.Health = state.MaxHealth
	}
	if state.Health < 0 {
		state.Health = 0
	}

	for _, item := range delta.AddItems {
		state.Inventory = append(state.Inventory, item)
	}

	for _, toRemove := range delta.RemoveItems {
		newInv := []string{}
		for _, item := range state.Inventory {
			if item != toRemove {
				newInv = append(newInv, item)
			}
		}
		state.Inventory = newInv
	}

	for k, v := range delta.Stats {
		state.Stats[k] += v
	}
	for k, v := range delta.Bonds {
		state.Bonds[k] += v
	}
	state.Karma += delta.Karma
	state.FatePoints += delta.FatePoints
	if state.FatePoints < 0 {
		state.FatePoints = 0
	}
	for k, v := range delta.Reputation {
		state.Reputation[k] += v
	}

	state.Flags = mergeFlags(state.Flags, delta.AddFlags, delta.RemoveFlags)
	if delta.VisualSetting != "" {
		state.VisualSetting = delta.VisualSetting
	}
}

func (e *Engine) updateEntities(state *domain.GameState, delta llm.StateDelta) {
	if state.Entities == nil {
		state.Entities = map[string]domain.Entity{}
	}
	for _, ent := range delta.AddEntities {
		state.Entities[strings.ToLower(ent.Name)] = domain.Entity{
			Name:         ent.Name,
			RelationToPC: ent.RelationToPC,
			Gender:       ent.Gender,
			Role:         ent.Role,
			Status:       ent.Status,
			Appearance:   ent.Appearance,
			Traits:       ent.Traits,
		}
	}
	for _, ent := range delta.UpdateEntities {
		key := strings.ToLower(ent.Name)
		existing, ok := state.Entities[key]
		if !ok {
			existing = domain.Entity{Name: ent.Name}
		}
		if ent.RelationToPC != "" {
			existing.RelationToPC = ent.RelationToPC
		}
		if ent.Gender != "" {
			existing.Gender = ent.Gender
		}
		if ent.Role != "" {
			existing.Role = ent.Role
		}
		if ent.Status != "" {
			existing.Status = ent.Status
		}
		if ent.Appearance != "" {
			existing.Appearance = ent.Appearance
		}
		if len(ent.Traits) > 0 {
			existing.Traits = ent.Traits
		}
		state.Entities[key] = existing
	}
}

func (e *Engine) entitySummary(entities map[string]domain.Entity) string {
	if len(entities) == 0 {
		return "None yet."
	}
	var parts []string
	for _, ent := range entities {
		desc := ent.Name
		if ent.RelationToPC != "" {
			desc += fmt.Sprintf(" (%s)", ent.RelationToPC)
		}
		if ent.Gender != "" {
			desc += fmt.Sprintf(", %s", ent.Gender)
		}
		if ent.Role != "" {
			desc += fmt.Sprintf(", %s", ent.Role)
		}
		if ent.Appearance != "" {
			desc += fmt.Sprintf(", appearance: %s", ent.Appearance)
		}
		parts = append(parts, desc)
	}
	return strings.Join(parts, "; ")
}

func (e *Engine) visualSettingSummary(setting string) string {
	setting = strings.TrimSpace(setting)
	if setting == "" {
		return "None yet."
	}
	return setting
}

func (e *Engine) chapterTitleForTurn(turnNumber, chapterNumber int) string {
	if turnNumber%6 == 1 {
		return fmt.Sprintf("Chapter %d", chapterNumber)
	}
	return ""
}

func (e *Engine) getSystemPrompt(genre, archetype string, age int, enhanced bool) string {
	v2Enabled := e.cfg.FeatureEnabled("novel_gen_v2_prompt_fix")

	base := `You are a serialized light-novel author and interactive fiction Game Master.
You must respond ONLY in valid JSON matching the requested schema exactly.
Schema fields:
"story_text": string
"choices": array of exactly 4 strings
"state_update": object with health_delta, inventory_add, inventory_remove, stats_delta, bonds_delta, karma_delta, fate_points_delta, reputation_delta, add_flags, remove_flags, visual_setting (string for a reusable visual world/background profile in ENGLISH for kids stories), add_entities (array), update_entities (array or empty object if no updates)
"format_hints": object with chapter_title (string), scene_break (boolean), monologue (boolean or count), dialogue_lines (boolean or count)
"twist_added": boolean
"story_complete": boolean
"image_scene": string — for kids stories, a concise ENGLISH visual description of the current scene for image generation. Describe characters present, their poses/expressions, setting, and mood. Example: "A young girl with shoulder-length dark hair, yellow shirt, and blue skirt kneels beside a small brown bird on a green school field. Morning sunlight. Cheerful mood."`

	style := FormatAsLightNovel(genre)
	if domain.IsKidsGenre(genre) {
		style = KidsStyleRules(genre, age, e.cfg.FeatureEnabled("enable_dynamic_kids_stories"))
	}
	genreDesc := GenreDescription(genre)
	archDesc := ArchetypeDescription(archetype)

	var enforceRules string
	if v2Enabled {
		enforceRules = fmt.Sprintf(`
STRICT GENRE ENFORCEMENT:
- The story MUST stay inside the %s genre at all times.
- Do NOT include themes, tropes, locations, creatures, or technology from other genres unless they are core to %s.
- If the genre is Romance, focus on relationships, feelings, attraction, conflict between hearts. No dungeon crawling, shadow monsters, or epic adventure quests unless directly justified by the romance plot.
- If the genre is Cyberpunk, focus on megacorps, hackers, neon cities, implants, and digital souls. No magic, dragons, or medieval settings.
- If the genre is Horror, focus on dread, survival, and the unknown. No heroic adventure party tropes.

ENTITY CONSISTENCY:
- Use the "Known Characters" list in every turn.
- Never change a character's relationship to the protagonist unless the state_update explicitly updates it.
- Keep pronouns and gender consistent for every named character.
- When a new named character appears, add them to add_entities with name, relation_to_pc, gender, role, status, appearance, and traits.
- For kids stories, treat Entities as character profiles. Keep name, appearance, personality traits, role, and relationship reusable and consistent across every turn.
- Kids appearance must be concrete and visual: hair length/style, shirt/top color, trousers/skirt, shoes, and distinctive accessory where useful.
- For kids stories, keep the storybook background and world details visually consistent across pages. If the location changes, update state_update.visual_setting to the new stable place description.`, genre, genre)
	} else {
		enforceRules = "Follow the genre and keep characters consistent when possible."
	}

	var narrativeLogic string
	if enhanced {
		narrativeLogic = `
NARRATIVE LOGIC AND DEPTH (Enhanced Mode):
- Before writing, briefly outline the scene's cause-and-effect chain: what caused the protagonist's current situation, what they do now, and what consequences logically follow.
- Maintain a coherent dramatic arc across the story: Introduction, Rising Action, Climax, and Resolution.
- Every character action must flow from established motivation, prior events, and the protagonist's archetype.
- Do NOT introduce non-sequitur events; every beat must connect to a previous beat or choice.
- If a character reappears, their behavior must remain consistent with their listed traits, relationship, and status.
- Show internal consequences: emotions, shifting bonds, growing tension, or foreshadowing.
- End each scene on a hook that arises naturally from the events just shown.`
	}

	return fmt.Sprintf(`%s

%s

%s

%s

Genre Context: %s
Protagonist Archetype: %s

Remember: the world must react to the protagonist. End every turn on a hook.`, base, style, enforceRules, narrativeLogic, genreDesc, archDesc)
}

func (e *Engine) arcInstruction(turnNumber int) string {
	// Kids stories use their own 10-page arc handled by KidsPageInstruction.
	// This method is for non-kids stories only.
	phase := turnNumber / 6
	switch {
	case phase == 0:
		return "ARC REMINDER: This is the Introduction. Establish the protagonist, their want, and the central conflict clearly."
	case phase < 3:
		return "ARC REMINDER: This is Rising Action. Escalate stakes through cause-and-effect complications, reveal character, and deepen relationships."
	case phase == 3:
		return "ARC REMINDER: Approach the Climax. Converge plot threads, force a hard choice, and make the next consequence unavoidable."
	default:
		return "ARC REMINDER: This is Resolution/Falling Action. Deliver satisfying consequences, resolve the immediate conflict, and plant seeds for future arcs."
	}
}

func (e *Engine) logMetrics(genre, text string, entities map[string]domain.Entity, turnNumber int, name GeneratedName, age int, status string, model string) {
	driftKeywords := map[string][]string{
		"Romance":   {"shadow", "dragon", "dungeon", "sword", "quest", "kingdom", "orc", "elf", "magic spell"},
		"Cyberpunk": {"dragon", "magic", "kingdom", "sword", "orc", "elf", "wizard"},
		"Horror":    {"romance", "love letter", "date", "kiss", "crush", "marriage proposal"},
		"Sci-Fi":    {"magic", "dragon", "wizard", "curse", "haunted"},
		"Adventure": {"spaceship", "hacker", "cyberware", "megacorp", "AI core"},
		"Mystery":   {"dragon", "spaceship", "magic", "kingdom"},
		"Xianxia":   {"spaceship", "cyberpunk", "megacorp", "pistol"},
	}

	lower := strings.ToLower(text)
	driftCount := 0
	if keywords, ok := driftKeywords[genre]; ok {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				driftCount++
			}
		}
	}

	entityScore := len(entities)
	wordCount := CountWords(text)
	ageGroup := ""
	completionSegment := ""
	if domain.IsKidsGenre(genre) {
		ageGroup = domain.KidsAgeTier(age)
		completionSegment = ageGroup
	}
	indonesianMarkers := IndonesianMarkers(text)
	if len(indonesianMarkers) > 0 {
		slog.Error("kids_story_indonesian_language_alert",
			"genre", genre,
			"age", age,
			"age_group", ageGroup,
			"markers", indonesianMarkers,
			"word_count", wordCount,
		)
	}

	actualModel := model
	if actualModel == "" {
		actualModel = e.llmClient.Model // default model
	}

	slog.Info("generation metrics",
		"model", actualModel,
		"turn", turnNumber,
		"genre", genre,
		"age", age,
		"age_group", ageGroup,
		"word_count", wordCount,
		"story_completion_rate_segment", completionSegment,
		"story_completed", strings.EqualFold(status, "game_over"),
		"drift_keywords_found", driftCount,
		"entity_count", entityScore,
		"v2_enabled", e.cfg.FeatureEnabled("novel_gen_v2_prompt_fix"),
		"enhanced_narrative_logic", e.cfg.FeatureEnabled("enhanced_narrative_logic"),
		"name", name.Name,
		"name_seed", name.Seed,
		"name_unique", name.IsUnique,
		"unique_name_rate", e.nameGenerator.UniqueRate([]string{}),
	)
}

func defaultKidsAppearance(gender string) string {
	if strings.EqualFold(gender, "Perempuan") {
		return "young Malaysian girl with shoulder-length dark hair, yellow short-sleeve shirt, blue skirt, white socks, red shoes"
	}
	return "young Malaysian boy with short dark hair, blue short-sleeve shirt, brown shorts, white socks, green shoes"
}

func stateJSON(state domain.GameState) string {
	b, _ := json.Marshal(state)
	return string(b)
}

func mergeFlags(existing, add, remove []string) []string {
	flagSet := make(map[string]bool)
	for _, f := range existing {
		flagSet[f] = true
	}
	for _, f := range add {
		flagSet[f] = true
	}
	for _, f := range remove {
		delete(flagSet, f)
	}
	result := []string{}
	for f := range flagSet {
		result = append(result, f)
	}
	return result
}

func sanitize(s string) string {
	if len(s) > 100 {
		s = s[:100]
	}
	return strings.TrimSpace(s)
}

// reviewBahasaMalaysia sends the story text and choices to the LLM for a
// self-review pass. The LLM checks for Indonesian language usage (vocabulary,
// slang, grammar, expressions) and rewrites the text in standard Bahasa
// Malaysia for Malaysian children. Returns the corrected text and choices.
//
// If the LLM call fails, it returns the original text unchanged (fail-open)
// so a transient API error doesn't block the story.
func (e *Engine) reviewBahasaMalaysia(ctx context.Context, client *llm.Client, storyText string, choices []string) (string, []string) {
	reviewPrompt := `Anda adalah penyunting bahasa yang teliti. Tugas anda ialah menyemak teks cerita kanak-kanak dan memastikan ia ditulis sepenuhnya dalam Bahasa Malaysia standard (Malaysia), BUKAN Bahasa Indonesia.

Peraturan ketat:
1. Ganti sebarang perkataan, frasa, slang, atau istilah Bahasa Indonesia dengan setara Bahasa Malaysia yang standard.
2. Betulkan tatabahasa atau gaya bahasa yang mengikut konvensi Indonesia kepada gaya Malaysia.
3. Pastikan kosa kata sesuai untuk kanak-kanak Malaysia (contoh: cikgu, murid, tandas, basikal, kereta api, kemeja, setem, tiket, kerusi, meja, almari, katil, tingkap, peti sejuk, mesin basuh, lampu, suis, motosikal, telefon, televisyen, kasut, stoking, seluar, songkok, tudung).
4. Jangan ubah makna, jalan cerita, atau struktur HTML.
5. Jangan tambah atau buang kandungan. Hanya betulkan bahasa.
6. Kekalkan nada yang mesra, gembira, dan sesuai untuk kanak-kanak.
7. Output MESTI 100% Bahasa Malaysia standard. Bercampur dengan Bahasa Indonesia TIDAK diterima sama sekali.

Pulangkan HANYA teks yang telah dibetulkan. Jangan penjelasan, jangan ulasan, jakan pembungkusan.`

	// Build the user message with story text and choices
	var sb strings.Builder
	sb.WriteString("Semakan teks cerita:\n\n=== TEKS CERITA ===\n")
	sb.WriteString(storyText)
	sb.WriteString("\n\n=== PILIHAN ===\n")
	for i, choice := range choices {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, choice))
	}
	sb.WriteString("\n\nPulangkan teks cerita yang telah dibetulkan diikuti dengan baris baru \"===PILIHAN===\" dan kemudian setiap pilihan yang dibetulkan pada baris berasingan dengan format [n] teks pilihan.")

	messages := []llm.Message{
		{Role: "system", Content: reviewPrompt},
		{Role: "user", Content: sb.String()},
	}

	corrected, err := client.GenerateText(ctx, messages, "")
	if err != nil {
		slog.Warn("bahasa_malaysia_review_failed", "error", err)
		return storyText, choices
	}

	// Parse the response: story text before "===PILIHAN===" and choices after.
	parts := strings.SplitN(corrected, "===PILIHAN===", 2)
	if len(parts) != 2 {
		// If the LLM didn't follow the format, use the entire response as story text.
		slog.Warn("bahasa_malaysia_review_format_unexpected", "response_preview", corrected[:min(200, len(corrected))])
		return strings.TrimSpace(corrected), choices
	}

	cleanedStory := strings.TrimSpace(parts[0])
	if cleanedStory == "" {
		cleanedStory = storyText
	}

	// Parse choices
	correctedChoices := choices
	choiceLines := strings.Split(strings.TrimSpace(parts[1]), "\n")
	parsedCount := 0
	for _, line := range choiceLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Extract text after [n] prefix
		if idx := strings.Index(line, "] "); idx >= 0 && idx < 5 {
			line = strings.TrimSpace(line[idx+2:])
		} else if idx := strings.Index(line, "]"); idx >= 0 && idx < 5 {
			line = strings.TrimSpace(line[idx+1:])
		}
		if parsedCount < len(correctedChoices) {
			correctedChoices[parsedCount] = line
		}
		parsedCount++
	}

	slog.Info("bahasa_malaysia_review_done", "story_changed", cleanedStory != storyText, "choices_changed", !equalSlices(correctedChoices, choices))
	return cleanedStory, correctedChoices
}

func (e *Engine) polishKidsEnding(ctx context.Context, client *llm.Client, storyText string, age, turnNumber int) string {
	minWords, maxWords := kidsWordBounds(age)
	targetWords := kidsTargetWords(age, turnNumber)
	prompt := fmt.Sprintf(`Anda ialah editor penutup cerita kanak-kanak.

Tugas anda:
1. Tulis SEMULA petikan ini menjadi penutup cerita yang lengkap, hangat, dan memuaskan.
2. Masalah utama mesti benar-benar selesai DALAM petikan ini.
3. Jangan tamat dengan hanya petunjuk, sedar sesuatu, rancangan masa depan, atau "mereka akan..." tanpa hasil sebenar.
4. Tunjukkan penyelesaian berlaku sekarang, kemudian beri pendaratan yang tenang dan satu pengajaran lembut.
5. Kekalkan Bahasa Malaysia standard, mesra kanak-kanak, dan sesuai umur.
6. Kekalkan watak, suasana, dan idea utama yang sudah ada. Jangan buka konflik baru.
7. Panjang sasaran %d patah perkataan, antara %d hingga %d patah perkataan.
8. Pulangkan HANYA teks cerita akhir. Jangan beri penjelasan.`, targetWords, minWords, maxWords)

	messages := []llm.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: storyText},
	}

	rewritten, err := client.GenerateText(ctx, messages, "")
	if err != nil {
		slog.Warn("kids_ending_polish_failed", "turn", turnNumber, "error", err)
		return storyText
	}
	rewritten = strings.TrimSpace(rewritten)
	if rewritten == "" {
		return storyText
	}
	slog.Info("kids_ending_polished", "turn", turnNumber, "changed", rewritten != storyText)
	return rewritten
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
