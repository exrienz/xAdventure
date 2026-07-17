const DOM = {
    initScreen: document.getElementById('init-screen'),
    gameScreen: document.getElementById('game-screen'),
    startForm: document.getElementById('start-form'),
    btnStart: document.getElementById('btn-start'),
    initLoading: document.getElementById('init-loading'),

    charNameDisplay: document.getElementById('char-name-display'),
    archetypeDisplay: document.getElementById('archetype-display'),
    hpText: document.getElementById('hp-text'),
    hpBar: document.getElementById('hp-bar'),
    karmaText: document.getElementById('karma-text'),
    fateText: document.getElementById('fate-text'),
    reputationList: document.getElementById('reputation-list'),
    bondsList: document.getElementById('bonds-list'),
    inventoryList: document.getElementById('inventory-list'),
    statsList: document.getElementById('stats-list'),

    storyLog: document.getElementById('story-log'),
    choiceArea: document.getElementById('choice-area'),
    turnLoading: document.getElementById('turn-loading'),
    choiceBtns: document.querySelectorAll('.choice-btn'),

    btnExport: document.getElementById('btn-export'),
    btnRestart: document.getElementById('btn-restart')
};

const SESSION_KEY = 'ine_session_id';
let currentSessionId = null;
let currentChoices = [];
let isProcessing = false;

async function apiStart(data) {
    const res = await fetch('/api/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

async function apiTurn(sessionID, choiceIndex) {
    const res = await fetch('/api/turn', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionID, choice_index: choiceIndex })
    });
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

async function apiGetSession(sessionID) {
    const res = await fetch(`/api/session/${sessionID}`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

DOM.startForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    await startGame();
});

DOM.choiceBtns.forEach((btn) => {
    btn.addEventListener('click', () => {
        const idx = parseInt(btn.getAttribute('data-index'));
        handleChoice(idx);
    });
});

DOM.btnExport.addEventListener('click', () => {
    if (!currentSessionId) return;
    window.location.href = `/api/download/${currentSessionId}`;
});

DOM.btnRestart.addEventListener('click', () => {
    if (confirm('Are you sure you want to end this journey?')) {
        localStorage.removeItem(SESSION_KEY);
        window.location.reload();
    }
});

document.addEventListener('keydown', (e) => {
    if (DOM.choiceArea.classList.contains('hidden') || isProcessing) return;
    const key = e.key.toUpperCase();
    const map = { 'A': 0, 'B': 1, 'C': 2, 'D': 3 };
    if (map[key] !== undefined) {
        handleChoice(map[key]);
    }
});

async function startGame() {
    const formData = new FormData(DOM.startForm);
    const data = {
        name: formData.get('name'),
        age: parseInt(formData.get('age')),
        gender: formData.get('gender'),
        genre: formData.get('genre'),
        archetype: formData.get('archetype'),
        seed: formData.get('seed')
    };

    DOM.btnStart.disabled = true;
    DOM.initLoading.classList.remove('hidden');

    try {
        const resp = await apiStart(data);
        setupGame(resp);
    } catch (err) {
        alert('Failed to start journey: ' + err.message);
        DOM.btnStart.disabled = false;
    } finally {
        DOM.initLoading.classList.add('hidden');
    }
}

function setupGame(resp) {
    currentSessionId = resp.session_id;
    localStorage.setItem(SESSION_KEY, currentSessionId);

    DOM.initScreen.classList.add('hidden');
    DOM.gameScreen.classList.remove('hidden');

    DOM.charNameDisplay.textContent = resp.user_name || document.getElementById('name').value;
    DOM.archetypeDisplay.textContent = document.getElementById('archetype').value;

    updateStateUI(resp.state);
    appendStorySegment({ content: resp.story, chapter_title: resp.chapter_title, format_hints: {} });
    updateChoices(resp.choices);
}

async function handleChoice(index) {
    if (isProcessing) return;
    if (index < 0 || index >= currentChoices.length) return;

    const choiceText = currentChoices[index];
    isProcessing = true;

    DOM.choiceArea.classList.add('hidden');
    DOM.turnLoading.classList.remove('hidden');

    const choiceEl = document.createElement('div');
    choiceEl.className = 'user-choice-log';
    choiceEl.textContent = `You chose: ${choiceText}`;
    DOM.storyLog.appendChild(choiceEl);

    const clearEl = document.createElement('div');
    clearEl.className = 'clear';
    DOM.storyLog.appendChild(clearEl);

    scrollToBottom();

    try {
        const resp = await apiTurn(currentSessionId, index);

        updateStateUI(resp.state);
        appendStorySegment({ content: resp.story, chapter_title: resp.chapter_title, format_hints: resp.format_hints });

        if (resp.status === 'game_over') {
            appendStorySegment({ content: '--- THE END ---', chapter_title: '', format_hints: {} });
            DOM.turnLoading.classList.add('hidden');
            return;
        }

        updateChoices(resp.choices);
    } catch (err) {
        appendStorySegment({ content: 'The narrator stumbles... (System Error: ' + err.message + '). Please try again.', chapter_title: '', format_hints: {} });
        DOM.choiceArea.classList.remove('hidden');
    } finally {
        isProcessing = false;
        DOM.turnLoading.classList.add('hidden');
    }
}

function updateStateUI(state) {
    DOM.hpText.textContent = `${state.health} / ${state.max_health}`;
    let hpPercent = (state.health / state.max_health) * 100;
    DOM.hpBar.style.width = `${hpPercent}%`;
    DOM.hpBar.style.backgroundColor = hpPercent <= 25 ? 'var(--danger-color)' : 'var(--health-color)';

    DOM.karmaText.textContent = state.karma || 0;
    DOM.fateText.textContent = state.fate_points || 0;

    renderMapList(DOM.reputationList, state.reputation);
    renderMapList(DOM.bondsList, state.bonds);
    renderMapList(DOM.statsList, state.stats);

    DOM.inventoryList.innerHTML = '';
    if (!state.inventory || state.inventory.length === 0) {
        DOM.inventoryList.innerHTML = '<div class="empty-state">Your pockets are empty.</div>';
    } else {
        state.inventory.forEach((item) => {
            const el = document.createElement('span');
            el.className = 'inventory-item';
            el.textContent = item;
            DOM.inventoryList.appendChild(el);
        });
    }
}

function renderMapList(container, mapData) {
    container.innerHTML = '';
    const keys = Object.keys(mapData || {});
    if (keys.length === 0) {
        container.innerHTML = '<div class="empty-state">None yet.</div>';
        return;
    }
    keys.forEach((k) => {
        const el = document.createElement('span');
        el.className = 'tag-item';
        el.textContent = `${k}: ${mapData[k]}`;
        container.appendChild(el);
    });
}

function appendStorySegment(segment) {
    return new Promise((resolve) => {
        if (segment.chapter_title) {
            const chapterEl = document.createElement('div');
            chapterEl.className = 'chapter-title';
            chapterEl.textContent = segment.chapter_title;
            chapterEl.setAttribute('aria-live', 'polite');
            DOM.storyLog.appendChild(chapterEl);
        } else if (segment.format_hints && segment.format_hints.chapter_title) {
            const chapterEl = document.createElement('div');
            chapterEl.className = 'chapter-title';
            chapterEl.textContent = segment.format_hints.chapter_title;
            chapterEl.setAttribute('aria-live', 'polite');
            DOM.storyLog.appendChild(chapterEl);
        }

        if (segment.format_hints && segment.format_hints.scene_break) {
            const breakEl = document.createElement('div');
            breakEl.className = 'scene-break';
            breakEl.textContent = '***';
            DOM.storyLog.appendChild(breakEl);
        }

        const el = document.createElement('div');
        el.className = 'story-segment';
        el.setAttribute('aria-live', 'polite');
        DOM.storyLog.appendChild(el);

        const text = segment.content;
        let displayText = text;
        // Apply markdown-style italics for internal monologue by replacing *text* with em elements
        displayText = displayText.replace(/\*([^*]+)\*/g, '<em class="monologue">$1</em>');
        el.innerHTML = displayText;

        // Typewriter animation only on plain text nodes
        const plainNodes = Array.from(el.childNodes).filter(n => n.nodeType === Node.TEXT_NODE);
        let animatedNodes = 0;
        if (plainNodes.length === 0) {
            resolve();
            return;
        }

        plainNodes.forEach((node) => {
            const fullText = node.textContent;
            node.textContent = '';
            let i = 0;
            const speed = 6;

            function typeChar() {
                if (i < fullText.length) {
                    node.textContent += fullText.charAt(i);
                    i++;
                    scrollToBottom();
                    setTimeout(typeChar, speed);
                } else {
                    animatedNodes++;
                    if (animatedNodes === plainNodes.length) resolve();
                }
            }
            typeChar();
        });
    });
}

function updateChoices(choices) {
    currentChoices = choices;
    DOM.choiceBtns.forEach((btn, idx) => {
        if (choices[idx]) {
            btn.querySelector('.text').textContent = choices[idx];
            btn.style.display = 'flex';
            btn.setAttribute('aria-label', `Choice ${String.fromCharCode(65 + idx)}: ${choices[idx]}`);
        } else {
            btn.style.display = 'none';
        }
    });
    DOM.choiceArea.classList.remove('hidden');
    scrollToBottom();
}

function scrollToBottom() {
    DOM.storyLog.scrollTop = DOM.storyLog.scrollHeight;
}

// Session recovery on page load
window.addEventListener('load', async () => {
    const savedSessionId = localStorage.getItem(SESSION_KEY);
    if (savedSessionId) {
        try {
            const resp = await apiGetSession(savedSessionId);
            if (resp.status === 'active') {
                const cont = confirm('You have an active journey. Continue?');
                if (cont) {
                    currentSessionId = resp.session_id;
                    DOM.initScreen.classList.add('hidden');
                    DOM.gameScreen.classList.remove('hidden');
                    DOM.charNameDisplay.textContent = resp.user_name || 'Character';
                    updateStateUI(resp.state);
                    updateChoices(resp.choices);
                } else {
                    localStorage.removeItem(SESSION_KEY);
                }
            }
        } catch (e) {
            localStorage.removeItem(SESSION_KEY);
        }
    }
});
