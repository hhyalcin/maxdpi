import './style.css';

import {Connect, Disconnect, OpenDiscord, Status} from '../wailsjs/go/main/App';

const $ = <T extends HTMLElement>(id: string) => document.getElementById(id) as T;
const dot = $('dot'), state = $('state'), detail = $('detail');
const toggle = $<HTMLButtonElement>('toggle'), open = $<HTMLButtonElement>('open');

let connected = false;

// Dinamik metin yalnizca textContent ile yazilir (innerHTML yok).
function render(kind: 'off' | 'on' | 'busy' | 'err', title: string, text = '') {
    dot.className = 'dot' + (kind === 'off' ? '' : ' ' + kind);
    state.textContent = title;
    detail.textContent = text;
    connected = kind === 'on';
    toggle.textContent = connected ? 'Kes' : 'Bağlan';
    toggle.disabled = kind === 'busy';
    open.hidden = !connected;
}

toggle.addEventListener('click', async () => {
    if (connected) {
        await Disconnect();
        render('off', 'Bağlı değil', 'Tek tıkla Discord erişimini aç.');
        return;
    }
    render('busy', 'Bağlanıyor…', 'Uygun yöntem aranıyor.');
    try {
        await Connect();
        render('on', 'Bağlı', 'Discord\'u buradan aç.');
    } catch (e) {
        render('err', 'Bağlanamadı', String(e));
    }
});

open.addEventListener('click', async () => {
    try {
        await OpenDiscord();
    } catch (e) {
        detail.textContent = String(e);
    }
});

Status().then((s) => {
    if (s.connected) render('on', 'Bağlı', 'Discord\'u buradan aç.');
});
