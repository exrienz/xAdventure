// ═══════════════════════════════════════════════════════
// xAdventure Kids Module — TV-Optimized JavaScript
// Features:
// - Genre grid with number-key selection (1-9, 0, then A-Z)
// - Choice selection via number keys (1-4)
// - No archetype (Peranan Kamu) selection
// - Full keyboard navigation for TV remotes
// ═══════════════════════════════════════════════════════

const KIDS_GENRES = [
    { value: "Pengembaraan",           label: "Pengembaraan" },
    { value: "Fantasi",                label: "Fantasi" },
    { value: "Dongeng Klasik",          label: "Dongeng Klasik" },
    { value: "Fabel",                   label: "Fabel" },
    { value: "Cerita Haiwan",           label: "Cerita Haiwan" },
    { value: "Cerita Sebelum Tidur",    label: "Cerita Sebelum Tidur" },
    { value: "Edukasi",                 label: "Edukasi" },
    { value: "Persahabatan",            label: "Persahabatan" },
    { value: "Keluarga",                label: "Keluarga" },
    { value: "Humor",                   label: "Humor" },
    { value: "Misteri Kanak-kanak",     label: "Misteri Kanak-kanak" },
    { value: "Fiksyen Sains Kanak-kanak", label: "Fiksyen Sains" },
    { value: "Fiksyen Sejarah",         label: "Fiksyen Sejarah" },
    { value: "Alam dan Persekitaran",   label: "Alam & Persekitaran" },
    { value: "Membesar",                label: "Membesar" },
    { value: "Budaya dan Folklor",      label: "Budaya & Folklor" },
    { value: "Mistik",                  label: "Mistik" },
    { value: "Cerita Interaktif",       label: "Cerita Interaktif" },
    { value: "Sains dan Teknologi",     label: "Sains & Teknologi" },
    { value: "Inspirasi",              label: "Inspirasi" },
];

const DOM = {
    initScreen: document.getElementById('init-screen'),
    gameScreen: document.getElementById('game-screen'),
    startForm: document.getElementById('start-form'),
    startBtn: document.getElementById('start-btn'),
    loading: document.getElementById('loading'),
    errorMsg: document.getElementById('init-error'),
    storyLog: document.getElementById('story-log'),
    storyImageFrame: document.getElementById('story-image-frame'),
    storyImage: document.getElementById('story-image'),
    imageLoading: document.getElementById('image-loading'),
    characterPanel: document.getElementById('character-panel'),
    storyError: document.getElementById('story-error'),
    storyErrorText: document.getElementById('story-error-text'),
    retryBtn: document.getElementById('retry-btn'),
    choicesContainer: document.getElementById('choices-container'),
    chapterTitle: document.getElementById('chapter-title'),
    turnIndicator: document.getElementById('turn-indicator'),
    genreGrid: document.getElementById('genre-grid'),
};

let currentSessionId = null;
let lastStartPayload = null;
let lastChoiceIndex = null;
let lastChoiceText = null;
let retryAction = 'start';
let retryCooldownUntil = 0;
let selectedGenre = null;
let currentChoices = [];

// ═══ INITIALIZATION ═══

function initGenreGrid() {
    DOM.genreGrid.innerHTML = '';
    KIDS_GENRES.forEach((g, i) => {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'genre-btn';
        btn.dataset.genre = g.value;
        btn.dataset.index = i;

        const num = document.createElement('span');
        num.className = 'genre-num';
        num.textContent = genreKeyLabel(i);

        const label = document.createElement('span');
        label.className = 'genre-label';
        label.textContent = g.label;

        btn.appendChild(num);
        btn.appendChild(label);
        btn.addEventListener('click', () => selectGenre(i));
        DOM.genreGrid.appendChild(btn);
    });
    // Auto-select first genre
    selectGenre(0);
}

function genreKeyLabel(index) {
    if (index < 9) return String(index + 1);
    if (index === 9) return '0';
    // For indices 10+, use letters
    return String.fromCharCode(65 + (index - 10)); // A, B, C...
}

function genreKeyToIndex(key) {
    if (key >= '1' && key <= '9') return parseInt(key) - 1;
    if (key === '0') return 9;
    if (key.length === 1) {
        const upper = key.toUpperCase();
        const code = upper.charCodeAt(0);
        if (code >= 65 && code <= 90) return code - 65 + 10;
    }
    return -1;
}

function selectGenre(index) {
    if (index < 0 || index >= KIDS_GENRES.length) return;
    selectedGenre = KIDS_GENRES[index].value;
    DOM.genreGrid.querySelectorAll('.genre-btn').forEach((btn, i) => {
        btn.classList.toggle('selected', i === index);
    });
}

// ═══ FORM SUBMISSION ═══

DOM.startForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    await startStory();
});

DOM.retryBtn.addEventListener('click', () => {
    retryFailedRequest();
});

// ═══ KEYBOARD NAVIGATION ═══

document.addEventListener('keydown', (e) => {
    // Only handle number keys when not typing in an input
    if (e.target.tagName === 'INPUT' || e.target.tagName === 'SELECT') return;

    // Init screen: number keys select genre
    if (DOM.initScreen.style.display !== 'none') {
        const idx = genreKeyToIndex(e.key);
        if (idx >= 0 && idx < KIDS_GENRES.length) {
            e.preventDefault();
            selectGenre(idx);
            return;
        }
        // Enter starts the story
        if (e.key === 'Enter' && selectedGenre) {
            e.preventDefault();
            DOM.startForm.requestSubmit();
            return;
        }
    }

    // Game screen: number keys select choices
    if (DOM.gameScreen.style.display !== 'none') {
        const idx = parseInt(e.key) - 1;
        if (idx >= 0 && idx < currentChoices.length) {
            e.preventDefault();
            const btns = DOM.choicesContainer.querySelectorAll('button');
            if (btns[idx] && !btns[idx].disabled) {
                btns[idx].click();
            }
            return;
        }
        // Enter selects first available choice
        if (e.key === 'Enter') {
            e.preventDefault();
            const btn = DOM.choicesContainer.querySelector('button');
            if (btn && !btn.disabled) btn.click();
        }
    }
});

// ═══ STORY FLOW ═══

async function startStory() {
    const payload = readStartPayload();
    lastStartPayload = payload;
    retryAction = 'start';
    lastChoiceIndex = null;
    lastChoiceText = null;

    setLoading(true);
    clearErrors();

    try {
        const data = await apiStart(payload);
        currentSessionId = data.session_id;
        showGameScreen();
        renderTurn(data);
    } catch (err) {
        currentSessionId = null;
        showGameScreen();
        renderError(friendlyError(err));
        logRenderError(err);
    } finally {
        setLoading(false);
    }
}

function readStartPayload() {
    return {
        name: document.getElementById('name').value,
        age: Number(document.getElementById('age').value),
        gender: document.getElementById('gender').value,
        genre: selectedGenre || KIDS_GENRES[0].value,
        archetype: ""  // No archetype for kids
    };
}

async function apiStart(payload) {
    const res = await fetch('/api/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
    });

    if (!res.ok) {
        throw new Error(await res.text());
    }

    const data = await res.json();
    if (!data.session_id || !data.story || !Array.isArray(data.choices)) {
        throw new Error('Data cerita tidak lengkap.');
    }
    return data;
}

function retryFailedRequest() {
    const now = Date.now();
    if (now < retryCooldownUntil) return;

    retryCooldownUntil = now + 2000;
    DOM.retryBtn.disabled = true;

    setTimeout(() => {
        DOM.retryBtn.disabled = false;
    }, 2000);

    if (retryAction === 'start' && lastStartPayload) {
        startStory();
        return;
    }

    if (retryAction === 'turn' && lastChoiceIndex !== null && lastChoiceText !== null) {
        makeChoice(lastChoiceIndex, lastChoiceText, true);
    }
}

function showGameScreen() {
    DOM.initScreen.style.display = 'none';
    DOM.gameScreen.style.display = 'flex';
    DOM.storyLog.innerHTML = '';
    DOM.choicesContainer.innerHTML = '';
    DOM.chapterTitle.textContent = 'Bab 1';
}

function renderTurn(data) {
    hideError();
    currentChoices = data.choices || [];

    if (data.chapter_title) {
        DOM.chapterTitle.textContent = data.chapter_title;
    }

    // Turn indicator for kids (Turn X of 5)
    if (data.status === 'game_over') {
        DOM.turnIndicator.textContent = 'Tamat';
    } else if (data.word_count !== undefined) {
        DOM.turnIndicator.textContent = '';
    }

    if (data.syllable_split) {
        renderStory(data.syllable_split, data.image_url, data.character_profiles);
    } else {
        renderStory(data.story, data.image_url, data.character_profiles);
    }

    renderChoices(data.choices, data.status);
}

function renderStory(text, imageUrl, characterProfiles) {
    DOM.storyLog.innerHTML = '';
    renderStoryImage(imageUrl);
    renderCharacterPanel(characterProfiles);
    const p = document.createElement('p');
    p.innerHTML = text;
    DOM.storyLog.appendChild(p);
    DOM.storyLog.scrollTop = 0;
}


function renderStoryImage(imageUrl) {
    if (!DOM.storyImage || !DOM.storyImageFrame || !DOM.imageLoading) return;

    DOM.storyImage.onload = null;
    DOM.storyImage.onerror = null;

    if (!imageUrl) {
        DOM.storyImage.removeAttribute('src');
        DOM.storyImage.style.display = 'none';
        DOM.imageLoading.style.display = 'none';
        DOM.storyImageFrame.style.display = 'none';
        DOM.storyImageFrame.classList.remove('image-loaded', 'image-error');
        return;
    }

    DOM.storyImageFrame.style.display = 'flex';
    DOM.storyImageFrame.classList.remove('image-loaded', 'image-error');
    DOM.imageLoading.innerHTML = '<div class="spinner image-spinner"></div><p>Menyiapkan gambar...</p>';
    DOM.imageLoading.style.display = 'flex';
    DOM.storyImage.style.display = 'none';
    DOM.storyImage.alt = 'Gambar cerita semasa sedang dimuatkan';

    DOM.storyImage.onload = () => {
        DOM.imageLoading.style.display = 'none';
        DOM.storyImage.alt = 'Gambar cerita semasa';
        DOM.storyImage.style.display = 'block';
        DOM.storyImageFrame.classList.add('image-loaded');
    };

    DOM.storyImage.onerror = () => {
        DOM.storyImage.removeAttribute('src');
        DOM.storyImage.style.display = 'none';
        DOM.storyImageFrame.classList.add('image-error');
        DOM.imageLoading.innerHTML = '<p>Gambar belum berjaya dimuat. Cerita masih boleh dibaca.</p>';
        DOM.imageLoading.style.display = 'flex';
    };

    DOM.storyImage.src = imageUrl;
}

function renderCharacterPanel(characterProfiles) {
    if (!DOM.characterPanel) return;
    const profiles = characterProfiles || {};
    const chars = Object.values(profiles).slice(0, 3);
    DOM.characterPanel.innerHTML = '';
    if (!chars.length) {
        DOM.characterPanel.style.display = 'none';
        return;
    }
    DOM.characterPanel.style.display = 'flex';
    chars.forEach((c) => {
        const item = document.createElement('div');
        item.className = 'character-chip';
        const traits = Array.isArray(c.traits) ? c.traits.slice(0, 2).join(', ') : '';
        item.textContent = `${c.name || 'Watak'} — ${c.role || c.relation_to_pc || 'kawan'}${traits ? ' · ' + traits : ''}`;
        DOM.characterPanel.appendChild(item);
    });
}

function renderChoices(choices, status) {
    DOM.choicesContainer.innerHTML = '';

    if (status === 'game_over') {
        const restartBtn = document.createElement('button');
        restartBtn.className = 'restart-btn';
        restartBtn.innerHTML = '<span class="choice-text">Mula Cerita Baru</span>';
        restartBtn.onclick = () => window.location.reload();
        DOM.choicesContainer.appendChild(restartBtn);
        return;
    }

    (choices || []).forEach((choice, index) => {
        const btn = document.createElement('button');
        const num = document.createElement('span');
        num.className = 'choice-num';
        num.textContent = String(index + 1);
        const text = document.createElement('span');
        text.className = 'choice-text';
        text.innerHTML = choice;
        btn.appendChild(num);
        btn.appendChild(text);
        btn.onclick = () => makeChoice(index, choice, false);
        DOM.choicesContainer.appendChild(btn);
    });
}

async function makeChoice(index, choiceText, isRetry) {
    if (!isRetry) {
        lastChoiceIndex = index;
        lastChoiceText = choiceText;
        retryAction = 'turn';
    }

    const btns = DOM.choicesContainer.querySelectorAll('button');
    btns.forEach(b => b.disabled = true);
    DOM.choicesContainer.innerHTML = '<div class="spinner"></div><p style="text-align:center;">Tunggu sekejap...</p>';

    try {
        const data = await apiTurn(currentSessionId, index);
        renderTurn(data);
    } catch (err) {
        renderError(friendlyError(err));
        logRenderError(err);
    }
}

async function apiTurn(sessionId, choiceIndex) {
    const res = await fetch('/api/turn', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionId, choice_index: choiceIndex })
    });

    if (!res.ok) {
        throw new Error(await res.text());
    }

    const data = await res.json();
    if (!data.story || !Array.isArray(data.choices)) {
        throw new Error('Data cerita tidak lengkap.');
    }
    return data;
}

// ═══ ERROR HANDLING ═══

function renderError(message) {
    DOM.storyErrorText.textContent = message;
    DOM.storyError.style.display = 'block';
    DOM.retryBtn.disabled = false;
    DOM.choicesContainer.innerHTML = '';
}

function hideError() {
    DOM.storyError.style.display = 'none';
    DOM.storyErrorText.textContent = '';
    DOM.retryBtn.disabled = false;
}

function clearErrors() {
    DOM.errorMsg.textContent = '';
    hideError();
}

function setLoading(isLoading) {
    DOM.startBtn.disabled = isLoading;
    DOM.loading.style.display = isLoading ? 'block' : 'none';
}

function friendlyError(err) {
    const message = err && err.message ? err.message : 'Cerita gagal dimuat.';
    const clean = message.replace(/<[^>]*>/g, '').trim();
    return `Ralat: ${clean || 'Cerita gagal dimuat.'}`;
}

function trackEvent(name, detail) {
    window.dispatchEvent(new CustomEvent(name, { detail }));
}

function logRenderError(err) {
    window.dispatchEvent(new CustomEvent('kids_story_render_error', {
        detail: {
            message: err && err.message ? err.message : 'Unknown render error',
            action: retryAction
        }
    }));
}

// ═══ BOOT ═══

initGenreGrid();