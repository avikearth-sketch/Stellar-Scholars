
(() => {
"use strict";

const GIBS_BASE = "https://gibs.earthdata.nasa.gov/wmts/epsg3857/best";
const HOME = { center: [23.685, 90.3563], zoom: 3 };
const MAX_ZOOM = 12;                 // visual zoom limit (enlarges pixels past native level)
const TIMELINE_MIN = "2000-01-01";   // slider start; each layer reports its own real dates
const TODAY = new Date().toISOString().slice(0, 10);
const NEARBY_KM = 500;
const BLANK_TILE = "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7";


const LAYERS = {
    imagery:     { id: "MODIS_Terra_CorrectedReflectance_TrueColor", name: "Satellite imagery", pane: "imagery",
                   box: "layer-satellite", temporal: true, fallback: { tms: "GoogleMapsCompatible_Level9", ext: "jpg" } },
    vegetation:  { id: "MODIS_Terra_NDVI_8Day", name: "Vegetation (MODIS Terra NDVI)", pane: "data",
                   box: "layer-vegetation", temporal: true, opacity: 0.7 },
    temperature: { id: "MODIS_Terra_Land_Surface_Temp_Day", name: "Land surface temperature (MODIS Terra, day)", pane: "data",
                   box: "layer-temperature", temporal: true, opacity: 0.7 },
    fires:       { id: "VIIRS_SNPP_Thermal_Anomalies_375m_All", name: "Fires / thermal anomalies (VIIRS 375 m)", pane: "data",
                   box: "layer-fires", temporal: true },
    boundaries:  { id: "Reference_Features", name: "Country boundaries", pane: "reference",
                   box: "ref-boundaries", temporal: false, fallback: { tms: "GoogleMapsCompatible_Level9", ext: "png" } },
    labels:      { id: "Reference_Labels", name: "Place labels", pane: "reference",
                   box: "ref-labels", temporal: false, fallback: { tms: "GoogleMapsCompatible_Level9", ext: "png" } }
};
const IMAGERY_IDS = [
    "MODIS_Terra_CorrectedReflectance_TrueColor",
    "MODIS_Aqua_CorrectedReflectance_TrueColor",
    "VIIRS_SNPP_CorrectedReflectance_TrueColor"
];

const state = {
    date: TODAY,
    sel: null,            
    caps: undefined,      
    events: [],           
    eventsLoaded: false,
    history: [],          
    busy: false
};
const layerState = {};   
let map, selectedMarker = null, eventGroup = null, eventsPromise = null;

const $ = (id) => document.getElementById(id);
const DAY = 86400000;
const toMs = (s) => Date.parse(s + "T00:00:00Z");
const fmtDate = (ms) => new Date(ms).toISOString().slice(0, 10);

function escapeHTML(value) {
    const div = document.createElement("div");
    div.textContent = String(value ?? "");
    return div.innerHTML;
}

function toast(message, type = "info", ms) {
    const box = $("toasts");
    if (!box) { console.warn(message); return; }
    const el = document.createElement("div");
    el.className = `toast ${type}`;
    el.textContent = message;
    box.appendChild(el);
    setTimeout(() => el.remove(), ms || (type === "error" ? 7000 : 4000));
}

function fmtLat(lat) { return `${Math.abs(lat).toFixed(4)}° ${lat >= 0 ? "N" : "S"}`; }
function fmtLng(lng) { return `${Math.abs(lng).toFixed(4)}° ${lng >= 0 ? "E" : "W"}`; }

function distanceKm(lat1, lng1, lat2, lng2) {
    const r = Math.PI / 180, dLat = (lat2 - lat1) * r, dLng = (lng2 - lng1) * r;
    const a = Math.sin(dLat / 2) ** 2 + Math.cos(lat1 * r) * Math.cos(lat2 * r) * Math.sin(dLng / 2) ** 2;
    return 6371 * 2 * Math.asin(Math.sqrt(a));
}

function debounce(fn, wait) {
    let t;
    return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), wait); };
}


function parseRange(text) {
    const [a, b, p] = (text || "").trim().split("/");
    const m = /^P(\d+)([DMY])$/.exec(p || "P1D");
    if (!a || !b || !m) return null;
    let n = +m[1], u = m[2];
    if (u === "Y") { n *= 12; u = "M"; }
    const start = toMs(a.slice(0, 10)), end = toMs(b.slice(0, 10));
    return Number.isNaN(start) || Number.isNaN(end) ? null : { start, end, n, u };
}


function snap(ranges, ms) {
    let best = null;
    for (const r of ranges) {
        if (ms < r.start) continue;
        const eff = Math.min(ms, r.end);
        let c;
        if (r.u === "D") {
            c = r.start + Math.floor((eff - r.start) / (r.n * DAY)) * r.n * DAY;
        } else {
            const a = new Date(r.start), b = new Date(eff);
            let months = (b.getUTCFullYear() - a.getUTCFullYear()) * 12 + b.getUTCMonth() - a.getUTCMonth();
            if (b.getUTCDate() < a.getUTCDate()) months--;
            const d = new Date(r.start);
            d.setUTCMonth(d.getUTCMonth() + Math.floor(months / r.n) * r.n);
            c = d.getTime();
        }
        if (best === null || c > best) best = c;
    }
    return best;
}

function periodLabel(ranges) {
    const r = ranges && ranges[ranges.length - 1];
    if (!r) return null;
    if (r.u === "D") return r.n === 1 ? "daily" : `${r.n}-day composite`;
    return r.n === 1 ? "monthly" : r.n === 12 ? "annual" : `${r.n}-month`;
}

function wantedIds() {
    const ids = new Set(IMAGERY_IDS);
    Object.values(LAYERS).forEach((d) => ids.add(d.id));
    return ids;
}

function extractCaps(doc, wanted) {
    const out = {};
    for (const el of doc.getElementsByTagName("Layer")) {
        const idEl = [...el.children].find((c) => c.localName === "Identifier");
        const id = idEl && idEl.textContent.trim();
        if (!id || !wanted.has(id)) continue;
        const tms = (el.getElementsByTagName("TileMatrixSet")[0] || {}).textContent;
        const format = (el.getElementsByTagName("Format")[0] || {}).textContent || "";
        const ranges = [...el.getElementsByTagName("Dimension")]
            .flatMap((d) => [...d.getElementsByTagName("Value")])
            .map((v) => parseRange(v.textContent)).filter(Boolean);
        const level = +((/Level(\d+)/.exec(tms || "") || [])[1]);
        if (!tms || Number.isNaN(level)) continue;
        out[id] = { tms: tms.trim(), level, ext: format.includes("png") ? "png" : "jpg", ranges: ranges.length ? ranges : null };
    }
    return out;
}

async function loadCapabilities() {
    const cacheKey = "esd.caps.v1." + TODAY;
    try {
        const cached = sessionStorage.getItem(cacheKey);
        if (cached) return JSON.parse(cached);
    } catch (e) {  }

   
    const urls = ["/api/gibs/capabilities?projection=epsg3857", GIBS_BASE + "/1.0.0/WMTSCapabilities.xml"];
    let lastError;
    for (const url of urls) {
        try {
            const res = await fetch(url);
            if (!res.ok) throw new Error(`HTTP ${res.status}`);
            const doc = new DOMParser().parseFromString(await res.text(), "application/xml");
            if (doc.querySelector("parsererror")) throw new Error("Capabilities XML could not be parsed");
            const meta = extractCaps(doc, wantedIds());
            try { sessionStorage.setItem(cacheKey, JSON.stringify(meta)); } catch (e) { /* too big: ignore */ }
            return meta;
        } catch (err) {
            lastError = err;
            console.error("GIBS capabilities failed from", url, err);
        }
    }
    throw lastError;
}

function getMeta(key) {
    const def = LAYERS[key];
    const fromCaps = state.caps && state.caps[def.id];
    if (fromCaps) return fromCaps;
    if (def.fallback) {
        return { tms: def.fallback.tms, ext: def.fallback.ext, ranges: null, fallback: true,
                 level: +(/Level(\d+)/.exec(def.fallback.tms)[1]) };
    }
    return null;
}


function resolve(key) {
    const def = LAYERS[key], meta = getMeta(key);
    if (!meta) return { state: state.caps === undefined ? "pending" : state.caps === null ? "unverified" : "missing" };
    if (!def.temporal) return { state: "ok", meta };
    if (!meta.ranges) return { state: "ok", meta, date: state.date, unverifiedDates: true };
    const ms = toMs(state.date), s = snap(meta.ranges, ms);
    if (s === null) return { state: "outside", meta, first: fmtDate(Math.min(...meta.ranges.map((r) => r.start))) };
    return { state: "ok", meta, date: fmtDate(s), latest: ms > Math.max(...meta.ranges.map((r) => r.end)) };
}


function initMap() {
    map = L.map("map", { center: HOME.center, zoom: HOME.zoom, minZoom: 2, maxZoom: MAX_ZOOM, worldCopyJump: true, zoomControl: true });


    [["imagery", 210], ["data", 300], ["reference", 350]].forEach(([name, z]) => {
        const pane = map.createPane(name);
        pane.style.zIndex = z;
        pane.style.pointerEvents = "none";
    });

    for (const [key, def] of Object.entries(LAYERS)) {
        const layer = L.tileLayer("", {
            pane: def.pane, opacity: def.opacity ?? 1, minZoom: 0, maxZoom: MAX_ZOOM,
            maxNativeZoom: 9, errorTileUrl: BLANK_TILE, time: "",
            attribution: def.pane === "imagery" ? "NASA GIBS / NASA Earthdata" : ""
        });
        layerState[key] = { layer, sig: null, res: null, errors: 0, text: "", cls: "" };
        layer.on("loading", () => { layerState[key].errors = 0; showStatus(key, "loading…", ""); });
        layer.on("tileerror", () => { layerState[key].errors++; });
        layer.on("load", () => {
            const n = layerState[key].errors;
            n ? showStatus(key, `${n} tile(s) missing (no data in view)`, "warn") : showStatus(key, "", "");
        });
    }

    eventGroup = L.layerGroup();
    layerState.events = { text: "", cls: "" };   

    map.on("mousemove", (e) => updateCoordinates(e.latlng.lat, e.latlng.lng));
    map.on("click", (e) => selectLocation(e.latlng.lat, e.latlng.lng));
    map.on("zoomend", updateZoomNote);
    map.on("moveend", () => loadChartSoon());
}

function updateCoordinates(lat, lng) {
    const la = $("latitude"), lo = $("longitude");
    if (la) la.textContent = fmtLat(lat);
    if (lo) lo.textContent = fmtLng(lng);
}

function setStatusBase(key, text, cls) {
    layerState[key].text = text;
    layerState[key].cls = cls;
    showStatus(key, "", "");
}

function showStatus(key, extra, extraCls) {
    const el = document.querySelector(`[data-status="${key}"]`);
    if (!el) return;
    const st = layerState[key] || { text: "", cls: "" };
    el.textContent = [st.text, extra].filter(Boolean).join(" · ");
    el.className = "layer-status " + (extraCls || st.cls || "");
}

function applyLayer(key) {
    const def = LAYERS[key], st = layerState[key], box = $(def.box);
    const on = !!(box && box.checked);
    const res = resolve(key);
    st.res = res;

    const hide = () => { if (map.hasLayer(st.layer)) map.removeLayer(st.layer); };
    if (!on) { hide(); setStatusBase(key, "", ""); return; }

    if (res.state !== "ok") {
        hide();
        const msg = {
            pending: ["Checking GIBS availability…", ""],
            unverified: ["Could not verify this layer with GIBS", "err"],
            missing: ["Not offered by GIBS", "err"],
            outside: [`No data for ${state.date} (starts ${res.first})`, "warn"]
        }[res.state];
        st.sig = null;
        setStatusBase(key, msg[0], msg[1]);
        return;
    }

    const m = res.meta;
    const url = `${GIBS_BASE}/${def.id}/default/${def.temporal ? "{time}/" : ""}${m.tms}/{z}/{y}/{x}.${m.ext}`;
    const sig = url + "|" + (res.date || "");
    st.layer.options.maxNativeZoom = m.level;
    st.layer.options.time = res.date || "";
    if (st.sig !== sig) {
        st.sig = sig;

        st.layer.setUrl(url, true);
        if (map.hasLayer(st.layer)) st.layer.redraw();
    }
    if (!map.hasLayer(st.layer)) st.layer.addTo(map);

    let text = "";
    if (def.temporal) {
        const label = m.ranges ? periodLabel(m.ranges) : "dates unverified";
        text = `${res.date} · ${label}${res.latest ? " · latest available" : ""}`;
    }
    setStatusBase(key, text, res.unverifiedDates ? "warn" : "ok");
}

function applyAllLayers() {
    Object.keys(LAYERS).forEach(applyLayer);
    updateLayerDates();
    updateZoomNote();
    renderSelected();
    updateAIContext();
    syncChart();
}

function updateZoomNote() {
    const el = $("zoom-note");
    if (!el || !map) return;
    const z = map.getZoom();
    const over = Object.keys(LAYERS).filter((k) => {
        const r = layerState[k].res;
        return LAYERS[k].pane !== "reference" && r && r.state === "ok" && $(LAYERS[k].box)?.checked && z > r.meta.level;
    });
    el.hidden = !over.length;
    if (over.length) {
        el.textContent = "Zoomed past native NASA resolution for: " +
            over.map((k) => `${LAYERS[k].name} (max z${layerState[k].res.meta.level})`).join(", ") +
            ". Pixels are enlarged, not more detailed.";
    }
}

function activeLayers() {
    const list = [];
    for (const [key, def] of Object.entries(LAYERS)) {
        const r = layerState[key].res;
        if (!$(def.box)?.checked || !r || r.state !== "ok") continue;
        list.push({
            id: def.id, name: def.name, type: def.pane === "reference" ? "reference" : "gibs",
            datasetDate: def.temporal ? r.date : null,
            temporalResolution: def.temporal ? (r.meta.ranges ? periodLabel(r.meta.ranges) : "unverified") : "static",
            nativeMaxZoom: r.meta.level
        });
    }
    if ($("layer-events")?.checked && state.eventsLoaded) {
        list.push({ id: "NASA_EONET_open_events", name: "Natural events (EONET, currently open)", type: "events",
                    datasetDate: null, temporalResolution: "current snapshot" });
    }
    return list;
}

function setupLayerControls() {
    Object.entries(LAYERS).forEach(([key, def]) => {
        $(def.box)?.addEventListener("change", (ev) => {
            if (key === "temperature" && ev.target.checked) chart.closed = false;
            applyLayer(key); updateLayerDates(); updateZoomNote(); renderSelected(); updateAIContext();
            if (key === "temperature") syncChart();
        });
    });
    $("imagery-source")?.addEventListener("change", (e) => {
        LAYERS.imagery.id = e.target.value;
        layerState.imagery.sig = null;
        applyAllLayers();
    });
    $("layer-events")?.addEventListener("change", (e) => {
        if (e.target.checked) {
            ensureEvents().then(() => showEventMarkers(true));
        } else {
            showEventMarkers(false);
            setStatusBase("events", "", "");
        }
        renderSelected(); updateAIContext();
    });
}


const applyLayersSoon = debounce(applyAllLayers, 200);

function setDate(dateStr, source) {
    let ms = toMs(dateStr);
    if (Number.isNaN(ms)) return;
    ms = Math.min(Math.max(ms, toMs(TIMELINE_MIN)), toMs(TODAY));
    state.date = fmtDate(ms);
    if ($("date-display")) $("date-display").textContent = state.date === TODAY ? `Today (${state.date})` : state.date;
    if (source !== "input" && $("date-input")) $("date-input").value = state.date;
    if (source !== "slider" && $("time-slider")) $("time-slider").value = Math.round((ms - toMs(TIMELINE_MIN)) / DAY);
    applyLayersSoon();
    renderSelected();
    updateAIContext();
}

function updateLayerDates() {
    const el = $("timeline-layer-dates");
    if (!el) return;
    const parts = Object.keys(LAYERS)
        .filter((k) => LAYERS[k].temporal && layerState[k].res && layerState[k].res.state === "ok" && $(LAYERS[k].box)?.checked)
        .map((k) => `${LAYERS[k].name.split(" (")[0]}: ${layerState[k].res.date}`);
    el.textContent = parts.length ? "Showing dataset dates → " + parts.join(" · ") : "";
}

function setupTimeline() {
    const slider = $("time-slider"), input = $("date-input");
    const days = Math.round((toMs(TODAY) - toMs(TIMELINE_MIN)) / DAY);
    if (slider) { slider.min = 0; slider.max = days; slider.step = 1; slider.value = days; }
    if (input) { input.min = TIMELINE_MIN; input.max = TODAY; input.value = TODAY; }

    slider?.addEventListener("input", () => setDate(fmtDate(toMs(TIMELINE_MIN) + slider.value * DAY), "slider"));
    input?.addEventListener("change", () => input.value && setDate(input.value, "input"));
    $("date-prev")?.addEventListener("click", () => setDate(fmtDate(toMs(state.date) - DAY)));
    $("date-next")?.addEventListener("click", () => setDate(fmtDate(toMs(state.date) + DAY)));
    $("date-latest")?.addEventListener("click", () => setDate(TODAY));


    const ticks = $("timeline-labels");
    if (ticks) {
        const y0 = +TIMELINE_MIN.slice(0, 4), y1 = +TODAY.slice(0, 4), total = toMs(TODAY) - toMs(TIMELINE_MIN);
        for (let y = y0; y <= y1; y += 6) {
            const pos = (toMs(`${y}-01-01`) - toMs(TIMELINE_MIN)) / total * 100;
            if (pos > 94) continue;
            const s = document.createElement("span");
            s.style.left = pos + "%"; s.textContent = y;
            ticks.appendChild(s);
        }
        const t = document.createElement("span");
        t.style.left = "100%"; t.textContent = "Today";
        ticks.appendChild(t);
    }
    setDate(TODAY);
}


const EVENT_COLORS = { wildfires: "#ff8a3d", severeStorms: "#b48cff", volcanoes: "#ff5c5c", floods: "#4da3ff",
                       seaLakeIce: "#7fe3ff", drought: "#d9b45a", dustHaze: "#c9a27a", landslides: "#a0785a" };


function firstPoint(c) {
    if (!Array.isArray(c)) return null;
    if (c.length >= 2 && typeof c[0] === "number" && typeof c[1] === "number") return c;
    for (const item of c) { const p = firstPoint(item); if (p) return p; }
    return null;
}

function parseEvent(ev) {
    try {
        const geoms = Array.isArray(ev.geometry) ? ev.geometry : [];
        for (let i = geoms.length - 1; i >= 0; i--) {         
            const p = firstPoint(geoms[i] && geoms[i].coordinates);
            if (!p) continue;
            const [lng, lat] = p;
            if (!Number.isFinite(lat) || !Number.isFinite(lng) || Math.abs(lat) > 90 || Math.abs(lng) > 180) continue;
            const cat = (ev.categories && ev.categories[0]) || {};
            const src = (ev.sources || []).find((s) => /^https?:\/\//.test(s.url || ""));
            return { id: ev.id, title: ev.title || "NASA natural event", categoryId: cat.id || "", category: cat.title || "Natural event",
                     date: geoms[i].date || null, lat, lng,
                     link: src ? src.url : (/^https?:\/\//.test(ev.link || "") ? ev.link : null) };
        }
    } catch (err) { console.error("Skipping malformed EONET event", ev, err); }
    return null;
}

function ensureEvents() {
    if (state.eventsLoaded) return Promise.resolve();
    if (eventsPromise) return eventsPromise;
    showStatus("events", "", "");
    setStatusBase("events", "loading NASA events…", "");
    eventsPromise = fetch("/api/eonet/events?status=open&limit=200")
        .then(async (res) => {
            if (!res.ok) throw new Error(`EONET returned ${res.status}`);
            const json = await res.json();
            const raw = (json && json.data && json.data.events) || (json && json.events) || [];
            state.events = raw.map(parseEvent).filter(Boolean);
            state.eventsLoaded = true;
            setStatusBase("events", state.events.length ? `${state.events.length} open events (current snapshot)` : "No events returned",
                          state.events.length ? "ok" : "warn");
            buildEventMarkers();
        })
        .catch((err) => {
            console.error("Could not load NASA EONET events:", err);
            setStatusBase("events", "Could not load events", "err");
            toast("Could not load NASA natural events. Try again in a moment.", "error");
            const box = $("layer-events"); if (box) box.checked = false;
        })
        .finally(() => { eventsPromise = null; });
    return eventsPromise;
}

function buildEventMarkers() {
    eventGroup.clearLayers();
    state.events.forEach((e) => {
        const m = L.circleMarker([e.lat, e.lng], {
            radius: 6, weight: 2, fillOpacity: 0.75, bubblingMouseEvents: false,
            color: EVENT_COLORS[e.categoryId] || "#62d9c7", fillColor: EVENT_COLORS[e.categoryId] || "#62d9c7"
        });
        m.bindPopup(`<strong>${escapeHTML(e.title)}</strong><br>${escapeHTML(e.category)}` +
            (e.date ? `<br><small>Latest report: ${escapeHTML(String(e.date).slice(0, 10))}</small>` : "") +
            `<br><small>${e.lat.toFixed(3)}°, ${e.lng.toFixed(3)}°</small>` +
            (e.link ? `<br><a href="${escapeHTML(e.link)}" target="_blank" rel="noopener">Source</a>` : ""));
        eventGroup.addLayer(m);
    });
    showEventMarkers(!!$("layer-events")?.checked);
}

function showEventMarkers(show) {
    if (show && !map.hasLayer(eventGroup)) eventGroup.addTo(map);
    if (!show && map.hasLayer(eventGroup)) map.removeLayer(eventGroup);
}

function nearbyEvents() {
    if (!state.sel) return [];
    return state.events
        .map((e) => ({ e, d: distanceKm(state.sel.lat, state.sel.lng, e.lat, e.lng) }))
        .filter((x) => x.d <= NEARBY_KM)
        .sort((a, b) => a.d - b.d)
        .map(({ e, d }) => ({ id: e.id, title: e.title, category: e.category, date: e.date, distanceKm: Math.round(d),
                              latitude: e.lat, longitude: e.lng, link: e.link }));
}


function selectLocation(lat, lng) {
    state.sel = { lat, lng };
    if (selectedMarker) map.removeLayer(selectedMarker);
    selectedMarker = L.marker([lat, lng]).addTo(map).bindPopup(
        `<div style="min-width:180px"><strong>Selected location</strong><br>Latitude: ${lat.toFixed(4)}°<br>Longitude: ${lng.toFixed(4)}°</div>`
    ).openPopup();
    updateCoordinates(lat, lng);
    renderSelected();
    updateAIContext();
}

function renderSelected() {
    const el = $("selected-area");
    if (!el) return;
    if (!state.sel) { el.innerHTML = "<span>No area selected</span>"; return; }

    const layers = activeLayers();
    const layerHtml = layers.length
        ? "<ul>" + layers.map((l) => `<li>${escapeHTML(l.name)}${l.datasetDate ? ` <span class="sa-muted">(${escapeHTML(l.datasetDate)})</span>` : ""}</li>`).join("") + "</ul>"
        : '<span class="sa-muted">none</span>';

    let eventsHtml;
    if (!$("layer-events")?.checked && !state.eventsLoaded) {
        eventsHtml = '<span class="sa-muted">Turn on “Natural events” to list nearby events.</span>';
    } else if (!state.eventsLoaded) {
        eventsHtml = '<span class="sa-muted">Loading…</span>';
    } else {
        const near = nearbyEvents();
        eventsHtml = near.length
            ? "<ul>" + near.slice(0, 5).map((n) => `<li>${escapeHTML(n.title)} <span class="sa-muted">(${n.distanceKm} km)</span></li>`).join("") + "</ul>"
              + (near.length > 5 ? `<span class="sa-muted">+${near.length - 5} more</span>` : "")
            : `<span class="sa-muted">None within ${NEARBY_KM} km</span>`;
    }

    el.innerHTML = `
        <div><strong>Latitude:</strong> ${fmtLat(state.sel.lat)}</div>
        <div><strong>Longitude:</strong> ${fmtLng(state.sel.lng)}</div>
        <div class="sa-row"><strong>Date:</strong> ${escapeHTML(state.date)}</div>
        <div class="sa-row"><strong>Active layers:</strong>${layerHtml}</div>
        <div class="sa-row"><strong>Nearby events:</strong> ${eventsHtml}</div>`;
}

function resetMap() {
    map.setView(HOME.center, HOME.zoom);
    if (selectedMarker) { map.removeLayer(selectedMarker); selectedMarker = null; }
    state.sel = null;
    updateCoordinates(HOME.center[0], HOME.center[1]);
    renderSelected();
    updateAIContext();
    toast("Map reset.", "success", 2000);
}

function findMyLocation() {
    const btn = $("locate-btn");
    if (!navigator.geolocation) { toast("Your browser does not support location access.", "error"); return; }
    const original = btn ? btn.textContent : "";
    if (btn) { btn.disabled = true; btn.textContent = "📍 Locating…"; }
    const done = () => { if (btn) { btn.disabled = false; btn.textContent = original; } };

    navigator.geolocation.getCurrentPosition(
        (pos) => {
            map.setView([pos.coords.latitude, pos.coords.longitude], 7);
            selectLocation(pos.coords.latitude, pos.coords.longitude);
            toast("Location found.", "success", 2000);
            done();
        },
        (err) => {
            console.error("Geolocation error:", err);
            toast("Couldn't access your location. You can click anywhere on the map instead.", "warn");
            done();
        },
        { timeout: 10000 }
    );
}


const chart = { cache: new Map(), ctrl: null, data: null, closed: false };
const loadChartSoon = debounce(() => loadChart(), 800);

function chartMessage(html) {
    const body = $("tc-body");
    if (body) body.innerHTML = html;
    if ($("tc-readout")) $("tc-readout").textContent = "";
}

function syncChart() {
    const box = $("temp-chart");
    if (!box || !map) return;
    const on = !!$("layer-temperature")?.checked && !chart.closed;
    box.hidden = !on;
    if (!on) return;
    if (chart.data) renderChart(); else loadChartSoon();
}

async function loadChart() {
    if (!$("layer-temperature")?.checked || chart.closed || !map) return;
    const b = map.getBounds();
    const south = Math.max(-90, b.getSouth()), north = Math.min(90, b.getNorth());
    const half = (b.getEast() - b.getWest()) / 2;
    const lng = ((b.getCenter().lng + 540) % 360) - 180;
    if (north - south > 30 || half * 2 > 30) {
        chart.data = null;
        if ($("tc-sub")) $("tc-sub").textContent = "";
        chartMessage('<div class="tc-msg">Zoom in to see a regional average (the view must be under 30° wide).</div>');
        return;
    }
    const q = new URLSearchParams({ south: south.toFixed(3), north: north.toFixed(3),
                                    west: (lng - half).toFixed(3), east: (lng + half).toFixed(3) });
    const key = [south, north, lng - half, lng + half].map((v) => v.toFixed(1)).join("|");
    if (chart.cache.has(key)) { chart.data = chart.cache.get(key); renderChart(); return; }

    if (chart.ctrl) chart.ctrl.abort();
    chart.ctrl = new AbortController();
    chartMessage('<div class="tc-msg">Loading NASA POWER data…</div>');
    try {
        const res = await fetch("/api/climate/temperature?" + q, { signal: chart.ctrl.signal });
        const json = await res.json().catch(() => null);
        if (!res.ok || !json || json.success === false) {
            throw new Error((json && (json.details || json.error)) || `Server returned ${res.status}`);
        }
        chart.cache.set(key, json.data);
        chart.data = json.data;
        renderChart();
    } catch (err) {
        if (err.name === "AbortError") return;
        console.error("Temperature history failed:", err);
        chart.data = null;
        chartMessage(`<div class="tc-msg err">Could not load temperature history: ${escapeHTML(err.message)}</div>`);
    }
}

function renderChart() {
    const d = chart.data;
    if (!d || !d.series || !d.series.length) { chartMessage('<div class="tc-msg">No temperature data for this area.</div>'); return; }

    const W = 320, H = 130, pl = 34, pr = 6, pt = 8, pb = 18, n = d.series.length;
    const lines = [["t2m", "Air (2 m)", "#62d9c7"]];
    if (d.hasTS) lines.push(["ts", "Surface skin", "#ff9a5c"]);
    const raw = lines.map((l) => d.series.map((s) => s[l[0]]));
    // 12-month rolling mean hides the yearly cycle so the long trend is visible.
    const roll = (vals) => vals.map((_, i) => {
        const w = vals.slice(Math.max(0, i - 11), i + 1).filter((v) => v != null);
        return w.length >= 6 ? w.reduce((a, c) => a + c, 0) / w.length : null;
    });
    const rolled = raw.map(roll);
    const all = raw.flat().concat(rolled.flat()).filter((v) => v != null);
    if (!all.length) { chartMessage('<div class="tc-msg">No temperature data for this area.</div>'); return; }
    const lo = Math.floor(Math.min(...all)), hi = Math.ceil(Math.max(...all));
    const x = (i) => pl + (W - pl - pr) * i / Math.max(1, n - 1);
    const y = (v) => pt + (H - pt - pb) * (1 - (v - lo) / Math.max(1e-6, hi - lo));
    const path = (vals) => {
        let out = "", pen = false;
        vals.forEach((v, i) => {
            if (v == null) { pen = false; return; }
            out += `${pen ? "L" : "M"}${x(i).toFixed(1)},${y(v).toFixed(1)}`;
            pen = true;
        });
        return out;
    };

    const ym = state.date.slice(0, 4) + state.date.slice(5, 7);
    const mi = d.series.findIndex((s) => s.p === ym);
    const first = d.series[0].p, last = d.series[n - 1].p;
    const svg = `<svg id="tc-svg" viewBox="0 0 ${W} ${H}" width="100%" role="img" aria-label="Regional temperature history">
        <line x1="${pl}" x2="${W - pr}" y1="${y(lo)}" y2="${y(lo)}" stroke="#2a3d49"/>
        <line x1="${pl}" x2="${W - pr}" y1="${y(hi)}" y2="${y(hi)}" stroke="#2a3d49" stroke-dasharray="2 3"/>
        <text x="${pl - 4}" y="${y(hi) + 3}" text-anchor="end" class="tc-ax">${hi}°</text>
        <text x="${pl - 4}" y="${y(lo) + 3}" text-anchor="end" class="tc-ax">${lo}°</text>
        <text x="${pl}" y="${H - 4}" class="tc-ax">${first.slice(0, 4)}</text>
        <text x="${W - pr}" y="${H - 4}" text-anchor="end" class="tc-ax">${last.slice(0, 4)}</text>
        ${raw.map((v, k) => `<path d="${path(v)}" fill="none" stroke="${lines[k][2]}" stroke-opacity=".28" stroke-width="1"/>`).join("")}
        ${rolled.map((v, k) => `<path d="${path(v)}" fill="none" stroke="${lines[k][2]}" stroke-width="2"/>`).join("")}
        ${mi >= 0 ? `<line x1="${x(mi)}" x2="${x(mi)}" y1="${pt}" y2="${H - pb}" stroke="#f2c35b" stroke-dasharray="3 2"/>` : ""}
        <line id="tc-hover" x1="0" x2="0" y1="${pt}" y2="${H - pb}" stroke="#fff" stroke-opacity=".5" visibility="hidden"/>
    </svg>`;

    const body = $("tc-body");
    body.innerHTML = svg;
    const legend = lines.map((l) => `<span style="color:${l[2]}">● ${l[1]}</span>`).join(" ") +
        (mi >= 0 ? ' <span style="color:#f2c35b">┆ selected date</span>' : "");
    const readout = $("tc-readout");
    readout.innerHTML = legend;
    if ($("tc-sub")) {
        const [s0, s1, w0, e0] = d.bbox;
        $("tc-sub").textContent = `${d.points} sample point${d.points === 1 ? "" : "s"} in view (${s0.toFixed(1)}…${s1.toFixed(1)}°N, ${w0.toFixed(1)}…${e0.toFixed(1)}°E) · °C · ${first.slice(0, 4)}–${last.slice(0, 4)}`;
    }

    const el = $("tc-svg"), hov = $("tc-hover");
    el.addEventListener("mousemove", (e) => {
        const r = el.getBoundingClientRect();
        const px = (e.clientX - r.left) / r.width * W;
        const i = Math.min(n - 1, Math.max(0, Math.round((px - pl) / (W - pl - pr) * (n - 1))));
        hov.setAttribute("x1", x(i)); hov.setAttribute("x2", x(i)); hov.setAttribute("visibility", "visible");
        const s = d.series[i];
        readout.textContent = `${s.p.slice(0, 4)}-${s.p.slice(4)}` +
            (s.t2m != null ? ` · air ${s.t2m.toFixed(1)}°C` : "") + (s.ts != null ? ` · skin ${s.ts.toFixed(1)}°C` : "");
    });
    el.addEventListener("mouseleave", () => { hov.setAttribute("visibility", "hidden"); readout.innerHTML = legend; });
}


function buildContext() {
    return {
        location: state.sel ? { latitude: +state.sel.lat.toFixed(5), longitude: +state.sel.lng.toFixed(5) } : null,
        selectedDate: state.date,
        activeLayers: activeLayers(),
        nearbyEvents: nearbyEvents().slice(0, 15)
    };
}

function updateAIContext() {
    const el = $("ai-context");
    if (!el) return;
    const n = activeLayers().length;
    el.textContent = (state.sel ? `📍 ${fmtLat(state.sel.lat)}, ${fmtLng(state.sel.lng)}` : "📍 No location selected (click the map)") +
        ` · ${state.date} · ${n} layer${n === 1 ? "" : "s"}`;
}

function addMessage(kind, text, opts = {}) {
    const box = $("chat-messages");
    const el = document.createElement("div");
    el.className = "message live " + ({ user: "user-message", assistant: "ai-message", error: "ai-message error" }[kind] || "ai-message");
    if (opts.notice) el.classList.add("notice");
    el.textContent = text;
    if (opts.retry) {
        const b = document.createElement("button");
        b.className = "retry"; b.textContent = "Retry";
        b.addEventListener("click", () => { el.remove(); sendMessage(opts.retry, false); });
        el.appendChild(document.createElement("br"));
        el.appendChild(b);
    }
    box.appendChild(el);
    box.scrollTop = box.scrollHeight;
    return el;
}

function setBusy(busy) {
    state.busy = busy;
    const send = $("send-question"), input = $("question-input");
    if (send) send.disabled = busy;
    if (input) input.disabled = busy;
    if (!busy && input) input.focus();
}

async function sendMessage(text, echo = true) {
    text = (text || "").trim();
    if (!text || state.busy) return;
    if (echo) addMessage("user", text);
    const input = $("question-input");
    if (input) { input.value = ""; input.style.height = "auto"; }
    setBusy(true);

    const typing = document.createElement("div");
    typing.className = "message ai-message typing";
    typing.innerHTML = "<span></span><span></span><span></span>";
    $("chat-messages").appendChild(typing);
    $("chat-messages").scrollTop = $("chat-messages").scrollHeight;

    try {
        if ($("layer-events")?.checked) await ensureEvents();
        const res = await fetch("/api/ai/chat", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ message: text, history: state.history.slice(-20), context: buildContext() })
        });
        let data = null;
        try { data = await res.json(); } catch (e) { /* non-JSON error body */ }
        if (!res.ok || !data || data.success === false) {
            throw new Error((data && (data.details || data.error)) || `Server returned ${res.status}`);
        }
        typing.remove();
        addMessage("assistant", data.reply || "(empty reply)", { notice: data.configured === false });
        if (data.configured !== false) {
            state.history.push({ role: "user", content: text }, { role: "assistant", content: data.reply });
        }
    } catch (err) {
        console.error("AI chat failed:", err);
        typing.remove();
        addMessage("error", "Sorry, the AI request failed: " + err.message, { retry: text });
    } finally {
        setBusy(false);
    }
}

function openAIPanel() {
    const panel = $("ai-panel");
    if (!panel) return;
    panel.classList.add("open");
    panel.setAttribute("aria-hidden", "false");
    updateAIContext();
    setTimeout(() => $("question-input")?.focus(), 250);
}

function closeAIPanel() {
    const panel = $("ai-panel");
    if (!panel) return;
    panel.classList.remove("open");
    panel.setAttribute("aria-hidden", "true");
}

function setupChat() {
    const input = $("question-input");
    $("ai-button")?.addEventListener("click", openAIPanel);
    $("close-ai")?.addEventListener("click", closeAIPanel);
    $("send-question")?.addEventListener("click", () => sendMessage(input.value));

    input?.addEventListener("keydown", (e) => {
       
        if (e.key === "Enter" && !e.shiftKey && !e.isComposing) {
            e.preventDefault();
            sendMessage(input.value);
        }
    });
    input?.addEventListener("input", () => {
        input.style.height = "auto";
        input.style.height = Math.min(input.scrollHeight, 120) + "px";
    });
    document.addEventListener("keydown", (e) => { if (e.key === "Escape") closeAIPanel(); });
}


document.addEventListener("DOMContentLoaded", () => {
    initMap();
    setupLayerControls();
    $("locate-btn")?.addEventListener("click", findMyLocation);
    $("reset-btn")?.addEventListener("click", resetMap);
    $("tc-close")?.addEventListener("click", () => { chart.closed = true; syncChart(); });
    setupTimeline();     // also draws the first layers (imagery/reference use verified fallbacks)
    setupChat();
    renderSelected();
    updateAIContext();
    if ($("layer-events")?.checked) ensureEvents().then(() => showEventMarkers(true));

    loadCapabilities()
        .then((meta) => { state.caps = meta; })
        .catch((err) => {
            console.error("GIBS capabilities unavailable:", err);
            state.caps = null;
            toast("Couldn't read NASA GIBS layer details. Vegetation, temperature and fire layers are unavailable until this works.", "warn", 8000);
        })
        .finally(applyAllLayers);

    console.log("Earth System Detective ready.");
});

})();