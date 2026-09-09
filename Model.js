.pragma library

var TEMP_MIN = 0
var TEMP_MAX = 110
var BOOST_MIN = 0
var BOOST_MAX = 255
var MIN_POINTS = 2
var MAX_POINTS = 8
var INTERVAL_MIN = 1
var INTERVAL_MAX = 30
var HYSTERESIS_MIN = 0
var HYSTERESIS_MAX = 15
var HOT_MIN = 50
var HOT_MAX = 110
var HOT_DEFAULT = 90

var PROFILE_ORDER = ["low-power", "quiet", "balanced", "balanced-performance", "performance", "custom"]

var PROFILE_LABELS = {
  "low-power": "Low power",
  "quiet": "Quiet",
  "balanced": "Balanced",
  "balanced-performance": "Balanced perf",
  "performance": "Performance",
  "custom": "Custom"
}

var PROFILE_GLYPHS = {
  "low-power": "󰌪",
  "quiet": "󰤄",
  "balanced": "󰾅",
  "balanced-performance": "󰓅",
  "performance": "󰈸",
  "custom": "󰢻"
}

var CONSTRAINT_LABELS = {
  "long_term": "PL1 long term",
  "short_term": "PL2 short term",
  "peak_power": "Peak power"
}

var GLYPHS = {
  alien: "󰢚",
  fan: "󰈐",
  hot: "󰈸",
  idle: "󰤁",
  missing: "󰀦",
  error: "󰀦",
  light: "󱄄"
}

var ERROR_MESSAGES = {
  "no-binary": "alienwarectl is not installed",
  "no-daemon": "alienwarectl daemon is not running",
  "denied": "Permission denied",
  "not-supported": "Not supported on this machine",
  "bad-request": "The helper rejected that request",
  "hw-missing": "Hardware interface not present",
  "device-busy": "The lighting device is busy, try again",
  "aw-elc-busy": "The lighting controller is busy, try again",
  "kbd-busy": "The keyboard controller is busy, try again",
  "no-openrgb": "OpenRGB server is not reachable",
  "internal": "The helper hit an internal error"
}

var CURVE_NOTE = "Boost is additive. It can only push the fans above the firmware curve, never below it."
var PL_LOCKED_NOTE = "Locked by firmware"
var EFFECTS_NOTE = "Effects are not implemented yet. Set a colour for the selected regions instead."
var KBD_UNMAPPED_NOTE = "Greyed out keys have not had their wire index confirmed yet and cannot take a colour."

var REGIONS = [
  { id: "power", name: "Power button", ledCount: 1 },
  { id: "logo", name: "Lid logo", ledCount: 1 },
  { id: "ring-top", name: "Ring top", ledCount: 8 },
  { id: "ring-bottom", name: "Ring bottom", ledCount: 8 }
]

function kbKey(id, label, w, index, second) {
  var primary = index === undefined ? null : index
  var extras = []
  if (primary !== null) {
    extras.push(primary)
    if (typeof second === "number") extras.push(second)
  }
  return { id: id, label: label, w: w, index: primary, indices: extras }
}

function keyIndices(key) {
  if (!isKeyPaintable(key)) return []
  if (key.indices && typeof key.indices.length === "number" && key.indices.length) return key.indices
  return [key.index]
}

var KEYBOARD_ROWS = [
  [
    kbKey("esc", "󱊷", 1, 0),
    kbKey("f1", "F1", 1, 1), kbKey("f2", "F2", 1, 2), kbKey("f3", "F3", 1, 3), kbKey("f4", "F4", 1, 4),
    kbKey("f5", "F5", 1, 5), kbKey("f6", "F6", 1, 6), kbKey("f7", "F7", 1, 7), kbKey("f8", "F8", 1, 8),
    kbKey("f9", "F9", 1, 9), kbKey("f10", "F10", 1, 10), kbKey("f11", "F11", 1, 11), kbKey("f12", "F12", 1, 12),
    kbKey("home", "home", 1, 13), kbKey("end", "end", 1, 14), kbKey("del", "⌦", 1, 15)
  ],
  [
    kbKey("grave", "`", 1, 20),
    kbKey("1", "1", 1, 21), kbKey("2", "2", 1, 22), kbKey("3", "3", 1, 23), kbKey("4", "4", 1, 24),
    kbKey("5", "5", 1, 25), kbKey("6", "6", 1, 26), kbKey("7", "7", 1, 27), kbKey("8", "8", 1, 28),
    kbKey("9", "9", 1, 29), kbKey("0", "0", 1, 30), kbKey("minus", "-", 1, 31), kbKey("equals", "=", 1, 32),
    kbKey("backspace", "⌫", 2, 34, 35)
  ],
  [
    kbKey("tab", "⇥", 1.5, 40),
    kbKey("q", "Q", 1, 42), kbKey("w", "W", 1, 43), kbKey("e", "E", 1, 44), kbKey("r", "R", 1, 45),
    kbKey("t", "T", 1, 46), kbKey("y", "Y", 1, 47), kbKey("u", "U", 1, 48), kbKey("i", "I", 1, 49),
    kbKey("o", "O", 1, 50), kbKey("p", "P", 1, 51),
    kbKey("lbracket", "[", 1, 52), kbKey("rbracket", "]", 1, 53), kbKey("backslash", "\\", 1.5, 55)
  ],
  [
    kbKey("caps", "⇪", 1.75, 60, 61),
    kbKey("a", "A", 1, 62), kbKey("s", "S", 1, 63), kbKey("d", "D", 1, 64), kbKey("f", "F", 1, 65), kbKey("g", "G", 1, 66),
    kbKey("h", "H", 1, 67), kbKey("j", "J", 1, 68), kbKey("k", "K", 1, 69), kbKey("l", "L", 1, 70),
    kbKey("semicolon", ";", 1, 71), kbKey("quote", "'", 1, 72), kbKey("enter", "󰌑", 2.25, 74)
  ],
  [
    kbKey("lshift", "⇧", 2.25, 80, 81),
    kbKey("z", "Z", 1, 83), kbKey("x", "X", 1, 84), kbKey("c", "C", 1, 85), kbKey("v", "V", 1, 86), kbKey("b", "B", 1, 87),
    kbKey("n", "N", 1, 88), kbKey("m", "M", 1, 89), kbKey("comma", ",", 1, 90), kbKey("period", ".", 1, 91), kbKey("slash", "/", 1, 92),
    kbKey("rshift", "⇧", 1.75, 94), kbKey("pageup", "↑", 1, 114)
  ],
  [
    kbKey("lctrl", "⌃", 1.25, 100), kbKey("fn", "fn", 1.25, 101), kbKey("lsuper", "󰖳", 1.25, 102, 103), kbKey("lalt", "⌥", 1.25, 104),
    kbKey("space", "␣", 6),
    kbKey("ralt", "⌥", 1.25, 111), kbKey("rsuper", "󰖳", 1.25, 109), kbKey("rctrl", "⌃", 1.25, 112),
    kbKey("left", "←", 1, 133), kbKey("pagedown", "↓", 1, 134), kbKey("right", "→", 1, 135)
  ]
]

var KEYBOARD_MEDIA_COLUMN = [
  kbKey("micmute", "󰍭", 1, 19),
  kbKey("volmute", "󰖁", 1, 16),
  kbKey("volup", "󰕾", 1, 18),
  kbKey("voldown", "󰖀", 1, 17)
]

function toList(value) {
  if (!value || typeof value !== "object" || typeof value.length !== "number") return []
  var out = []
  for (var i = 0; i < value.length; i++) out.push(value[i])
  return out
}

function isObject(value) {
  return !!value && typeof value === "object" && typeof value.length !== "number"
}

function toNumber(value, fallback) {
  var n = Number(value)
  if (typeof value === "boolean" || value === null || value === undefined || value === "" || !isFinite(n)) return fallback
  return n
}

function toInt(value, fallback) {
  var n = toNumber(value, NaN)
  if (!isFinite(n)) return fallback
  return Math.round(n)
}

function clamp(value, lo, hi) {
  if (!isFinite(value)) return lo
  return Math.max(lo, Math.min(hi, value))
}

function clampInt(value, lo, hi, fallback) {
  var n = toInt(value, NaN)
  if (!isFinite(n)) n = fallback
  return clamp(n, lo, hi)
}

function boolOr(value, fallback) {
  if (typeof value === "boolean") return value
  if (value === undefined || value === null) return fallback
  var s = String(value).toLowerCase()
  if (s === "true" || s === "1" || s === "yes" || s === "on") return true
  if (s === "false" || s === "0" || s === "no" || s === "off") return false
  return fallback
}

function clampInterval(value) { return clampInt(value, INTERVAL_MIN, INTERVAL_MAX, 2) }
function clampHysteresis(value) { return clampInt(value, HYSTERESIS_MIN, HYSTERESIS_MAX, 3) }
function clampHot(value) { return clampInt(value, HOT_MIN, HOT_MAX, HOT_DEFAULT) }
function clampBoost(value) { return clampInt(value, BOOST_MIN, BOOST_MAX, 0) }
function clampBrightness(value) { return clampInt(value, 0, 100, 100) }

function normalizeDisplay(value) {
  var s = String(value === undefined || value === null ? "" : value).toLowerCase().trim()
  if (s === "temp" || s === "compact") return "temp"
  if (s === "full" || s === "expanded") return "full"
  return "icon"
}

function parseJson(text) {
  var raw = String(text === undefined || text === null ? "" : text).trim()
  if (!raw) return null
  try {
    var parsed = JSON.parse(raw)
    return parsed && typeof parsed === "object" ? parsed : null
  } catch (e) {
    return null
  }
}

function describeError(code, message) {
  var msg = String(message || "").trim()
  var known = ERROR_MESSAGES[String(code || "")]
  if (msg && known && msg.toLowerCase() !== known.toLowerCase()) return known + ": " + msg
  if (known) return known
  return msg || "Unknown error"
}

function parseResult(text, exitCode) {
  var code = toInt(exitCode, 0)
  var parsed = parseJson(text)
  if (!parsed) {
    if (code === 127 || code === 126) return { ok: false, code: "no-binary", error: describeError("no-binary", ""), data: null }
    return { ok: false, code: "internal", error: code === 0 ? "The helper printed nothing" : "The helper exited with code " + code, data: null }
  }
  if (parsed.ok === false) {
    var slug = String(parsed.code || "internal")
    return { ok: false, code: slug, error: describeError(slug, parsed.error), data: parsed }
  }
  if (code !== 0) {
    return { ok: false, code: "internal", error: "The helper exited with code " + code, data: parsed }
  }
  return { ok: true, code: "", error: "", data: parsed }
}

function fanPercent(rpm, max) {
  var r = toNumber(rpm, 0)
  var m = toNumber(max, 0)
  if (!(m > 0)) return 0
  return Math.round(clamp(r / m * 100, 0, 100))
}

function boostPercent(boost) {
  return Math.round(clampBoost(boost) / BOOST_MAX * 100)
}

function percentToBoost(percent) {
  return Math.round(clampInt(percent, 0, 100, 0) / 100 * BOOST_MAX)
}

function normalizeFan(raw, index) {
  var src = isObject(raw) ? raw : {}
  var id = String(src.id || (index === 1 ? "gpu" : "cpu"))
  var rpm = toInt(src.rpm, 0)
  var max = toInt(src.max, 0)
  var percent = src.percent === undefined || src.percent === null ? fanPercent(rpm, max) : clampInt(src.percent, 0, 100, fanPercent(rpm, max))
  return {
    id: id,
    index: toInt(src.index, index + 1),
    label: String(src.label || (id === "gpu" ? "GPU Fan" : "CPU Fan")),
    rpm: Math.max(0, rpm),
    max: Math.max(0, max),
    boost: clampBoost(src.boost),
    percent: percent,
    stopped: rpm === 0
  }
}

function normalizeTemps(raw) {
  var out = {}
  if (!isObject(raw)) return out
  var keys = ["cpu", "gpu", "sodimm", "other"]
  for (var i = 0; i < keys.length; i++) {
    var v = raw[keys[i]]
    if (v === undefined || v === null) continue
    var n = toNumber(v, NaN)
    if (!isFinite(n)) continue
    out[keys[i]] = Math.round(n)
  }
  return out
}

function normalizeProfile(raw) {
  var src = isObject(raw) ? raw : {}
  var choices = []
  var list = toList(src.choices)
  for (var i = 0; i < list.length; i++) {
    var name = String(list[i] || "").trim()
    if (name && choices.indexOf(name) === -1) choices.push(name)
  }
  var current = String(src.current || "").trim()
  if (current && choices.indexOf(current) === -1) choices.push(current)
  return {
    available: choices.length > 0,
    current: current,
    choices: choices,
    ppdRunning: src.ppdRunning === true,
    gmodeForced: src.gmodeForced === true,
    writable: src.writable !== false
  }
}

function normalizeConstraint(raw, index) {
  var src = isObject(raw) ? raw : {}
  var name = String(src.name || "")
  return {
    index: toInt(src.index, index),
    name: name,
    label: CONSTRAINT_LABELS[name] || (name ? name : "Constraint " + index),
    watts: toInt(src.watts, 0),
    writable: src.writable === true
  }
}

function normalizePower(raw) {
  var src = isObject(raw) ? raw : {}
  var list = toList(src.constraints)
  var out = []
  for (var i = 0; i < list.length; i++) out.push(normalizeConstraint(list[i], i))
  return {
    available: src.available === true && out.length > 0,
    constraints: out,
    anyWritable: out.some ? out.some(function(c) { return c.writable }) : false
  }
}

function normalizeGpu(raw) {
  var src = isObject(raw) ? raw : {}
  return {
    available: src.available === true,
    draw: toNumber(src.draw, NaN),
    limit: toNumber(src.limit, NaN),
    defaultLimit: toNumber(src.defaultLimit, NaN),
    maxLimit: toNumber(src.maxLimit, NaN),
    limitWritable: src.limitWritable === true
  }
}

function normalizeCpu(raw) {
  var src = isObject(raw) ? raw : {}
  return {
    available: src.available === true,
    mhz: toNumber(src.mhz, NaN)
  }
}

function normalizeCurveState(raw) {
  var src = isObject(raw) ? raw : {}
  return {
    active: src.active === true,
    interval: clampInterval(src.interval),
    hysteresis: clampHysteresis(src.hysteresis),
    cpu: normalizeCurve(src.cpu),
    gpu: normalizeCurve(src.gpu)
  }
}

function normalizeStatus(raw) {
  var src = isObject(raw) ? raw : {}
  var fanList = toList(src.fans)
  var fans = []
  for (var i = 0; i < fanList.length && fans.length < 2; i++) fans.push(normalizeFan(fanList[i], fans.length))
  var warnings = []
  var warnList = toList(src.warnings)
  for (var w = 0; w < warnList.length; w++) {
    var line = String(warnList[w] || "").trim()
    if (line) warnings.push(line)
  }
  return {
    ok: src.ok !== false,
    ts: toInt(src.ts, 0),
    model: String(src.model || ""),
    hwmon: String(src.hwmon || ""),
    fans: fans,
    temps: normalizeTemps(src.temps),
    profile: normalizeProfile(src.profile),
    turbo: {
      available: isObject(src.turbo) ? src.turbo.available === true : false,
      enabled: isObject(src.turbo) ? src.turbo.enabled === true : false
    },
    power: normalizePower(src.power),
    cpu: normalizeCpu(src.cpu),
    gpu: normalizeGpu(src.gpu),
    curve: normalizeCurveState(src.curve),
    warnings: warnings,
    fansAvailable: fans.length > 0
  }
}

function emptyStatus() {
  return normalizeStatus(null)
}

function fanById(status, id) {
  if (!status) return null
  var fans = toList(status.fans)
  for (var i = 0; i < fans.length; i++) if (fans[i] && fans[i].id === id) return fans[i]
  return null
}

function fansStopped(status) {
  if (!status) return false
  var fans = toList(status.fans)
  if (!fans.length) return false
  for (var i = 0; i < fans.length; i++) if (!fans[i] || fans[i].rpm !== 0) return false
  return true
}

function maxRpm(status) {
  var fans = toList(status ? status.fans : null)
  var best = 0
  for (var i = 0; i < fans.length; i++) if (fans[i] && fans[i].rpm > best) best = fans[i].rpm
  return best
}

function hottest(status) {
  if (!status || !isObject(status.temps)) return NaN
  var best = NaN
  var keys = ["cpu", "gpu"]
  for (var i = 0; i < keys.length; i++) {
    var v = status.temps[keys[i]]
    if (typeof v !== "number") continue
    if (!isFinite(best) || v > best) best = v
  }
  return best
}

function formatTemp(value, withUnit) {
  var n = toNumber(value, NaN)
  if (!isFinite(n)) return "n/a"
  return Math.round(n) + (withUnit ? "°C" : "°")
}

function formatRpm(value) {
  var n = toInt(value, NaN)
  if (!isFinite(n)) return "n/a"
  if (n === 0) return "stopped"
  return n + " rpm"
}

function formatWatts(value, digits) {
  var n = toNumber(value, NaN)
  if (!isFinite(n)) return "n/a"
  var d = toInt(digits, 0)
  return (d > 0 ? n.toFixed(d) : String(Math.round(n))) + " W"
}

function formatPercent(value) {
  var n = toNumber(value, NaN)
  if (!isFinite(n)) return "n/a"
  return Math.round(clamp(n, 0, 100)) + "%"
}

function formatClock(value) {
  var n = toNumber(value, NaN)
  if (!isFinite(n)) return "not reported"
  if (n >= 1000) return (n / 1000).toFixed(2) + " GHz"
  return Math.round(n) + " MHz"
}

function cpuClock(status) {
  if (!status || !isObject(status.cpu)) return NaN
  if (status.cpu.available !== true) return NaN
  return toNumber(status.cpu.mhz, NaN)
}

function profileLabel(name, gmodeForced) {
  var key = String(name || "").trim()
  if (!key) return "Unknown"
  if (key === "performance" && gmodeForced === true) return "G-Mode"
  return PROFILE_LABELS[key] || key.charAt(0).toUpperCase() + key.slice(1).replace(/-/g, " ")
}

function profileGlyph(name) {
  return PROFILE_GLYPHS[String(name || "")] || GLYPHS.fan
}

function profileChoices(status) {
  var list = status && status.profile ? toList(status.profile.choices) : []
  if (list.length) return list
  return PROFILE_ORDER.slice()
}

function nextProfile(choices, current, direction) {
  var list = toList(choices)
  if (!list.length) return String(current || "")
  var step = toInt(direction, 1)
  if (step === 0) step = 1
  var idx = list.indexOf(String(current || ""))
  if (idx === -1) return String(list[0])
  var next = (idx + step) % list.length
  if (next < 0) next += list.length
  return String(list[next])
}

function shouldRestoreProfile(wanted, current, lastGoodAt, resumeAt) {
  var want = String(wanted || "")
  var have = String(current || "")
  if (!want || !have) return false
  if (toNumber(lastGoodAt, 0) <= toNumber(resumeAt, 0)) return false
  return want !== have
}

function parseMuteState(text) {
  var s = String(text === undefined || text === null ? "" : text)
  var m = s.match(/Volume:\s*([0-9]*\.?[0-9]+)/)
  if (!m) return null
  return { volume: toNumber(m[1], 0), muted: s.indexOf("[MUTED]") >= 0 }
}

function isAudioEvent(line) {
  var s = String(line === undefined || line === null ? "" : line)
  if (s.indexOf("Event ") < 0) return false
  return s.indexOf(" on sink") >= 0 || s.indexOf(" on source") >= 0
}

function parseMutePair(text) {
  var lines = String(text === undefined || text === null ? "" : text).split("\n")
  var out = { sinkMuted: false, sourceMuted: false, ok: false }
  for (var i = 0; i < lines.length; i++) {
    var line = lines[i]
    if (line.indexOf("SINK ") === 0) {
      out.sinkMuted = line.indexOf("[MUTED]") >= 0
      out.ok = true
    } else if (line.indexOf("SOURCE ") === 0) {
      out.sourceMuted = line.indexOf("[MUTED]") >= 0
      out.ok = true
    }
  }
  return out
}

function parseBatteryState(text) {
  var lines = String(text === undefined || text === null ? "" : text).split("\n")
  var out = { percent: -1, charging: false, ok: false }
  for (var i = 0; i < lines.length; i++) {
    var line = lines[i]
    if (line.indexOf("CAP ") === 0) {
      var n = toNumber(line.substring(4).replace(/[^0-9.]/g, ""), -1)
      if (n >= 0) { out.percent = n; out.ok = true }
    } else if (line.indexOf("ST ") === 0) {
      var st = line.substring(3).trim().toLowerCase()
      if (st !== "") {
        out.charging = st === "charging" || st === "full"
        out.ok = true
      }
    }
  }
  return out
}

function batteryColor(state, colors) {
  var st = isObject(state) ? state : {}
  var c = isObject(colors) ? colors : {}
  if (st.ok !== true || toNumber(st.percent, -1) < 0) return ""
  if (toNumber(st.percent, 100) < 10) return normalizeHex(c.low) || "FF0000"
  if (st.charging === true) return normalizeHex(c.charging) || "00FF66"
  return normalizeHex(c.discharging) || "FF8800"
}

function applyBatteryOverlay(map, opts) {
  var o = isObject(opts) ? opts : {}
  var base = isObject(map) ? map : {}
  if (o.enabled !== true || o.lightsOn === false) return base
  var hex = batteryColor(o.battery, o.colors)
  if (!hex) return base
  var out = {}
  for (var k in base) out[k] = base[k]
  out["power"] = hex
  return out
}

function applyMuteOverlay(map, opts) {
  var o = isObject(opts) ? opts : {}
  var base = isObject(map) ? map : {}
  if (o.enabled !== true || o.lightsOn === false) return base
  var hex = normalizeHex(o.color) || "FF0000"
  var out = {}
  for (var k in base) out[k] = base[k]
  if (o.sinkMuted === true) out["volmute"] = hex
  if (o.sourceMuted === true) out["micmute"] = hex
  return out
}

function profileWarning(status) {
  if (!status || !status.profile) return ""
  if (!status.profile.writable) return "The platform profile is read only right now"
  if (status.profile.ppdRunning) return "power-profiles-daemon is running and may override this"
  return ""
}

function clampPoint(raw) {
  var src = isObject(raw) ? raw : {}
  return {
    temp: clampInt(src.temp, TEMP_MIN, TEMP_MAX, TEMP_MIN),
    boost: clampBoost(src.boost)
  }
}

function sortPoints(points) {
  var list = toList(points).slice()
  list.sort(function(a, b) {
    var at = a ? toNumber(a.temp, 0) : 0
    var bt = b ? toNumber(b.temp, 0) : 0
    if (at === bt) return toNumber(a ? a.boost : 0, 0) - toNumber(b ? b.boost : 0, 0)
    return at - bt
  })
  return list
}

function thinPoints(points, limit) {
  var list = points.slice()
  if (list.length <= limit) return list
  var out = [list[0]]
  var span = list.length - 1
  var slots = limit - 2
  for (var i = 1; i <= slots; i++) {
    var idx = Math.round(i * span / (slots + 1))
    if (idx <= 0) idx = 1
    if (idx >= span) idx = span - 1
    if (out.indexOf(list[idx]) === -1) out.push(list[idx])
  }
  out.push(list[span])
  return out
}

function normalizeCurve(points) {
  var sorted = sortPoints(points)
  var cleaned = []
  for (var i = 0; i < sorted.length; i++) {
    var raw = sorted[i]
    if (!isObject(raw)) continue
    if (!isFinite(toNumber(raw.temp, NaN))) continue
    var p = clampPoint(raw)
    var last = cleaned.length ? cleaned[cleaned.length - 1] : null
    if (last && last.temp === p.temp) {
      if (p.boost > last.boost) cleaned[cleaned.length - 1] = p
      continue
    }
    cleaned.push(p)
  }
  if (!cleaned.length) return defaultCurve()
  if (cleaned.length === 1) {
    var only = cleaned[0]
    if (only.temp < TEMP_MAX) cleaned.push({ temp: TEMP_MAX, boost: clampBoost(only.boost) })
    else cleaned.unshift({ temp: TEMP_MIN, boost: 0 })
    return cleaned
  }
  if (cleaned.length > MAX_POINTS) cleaned = thinPoints(cleaned, MAX_POINTS)
  return cleaned
}

function defaultCurve() {
  return [{ temp: 45, boost: 0 }, { temp: 65, boost: 64 }, { temp: 80, boost: 160 }, { temp: 90, boost: 255 }]
}

function curvePreset(name) {
  var key = String(name || "").toLowerCase()
  if (key === "silent" || key === "1") return [{ temp: 50, boost: 0 }, { temp: 75, boost: 32 }, { temp: 90, boost: 96 }]
  if (key === "balanced" || key === "2") return defaultCurve()
  if (key === "cool" || key === "3") return [{ temp: 40, boost: 32 }, { temp: 60, boost: 96 }, { temp: 75, boost: 176 }, { temp: 85, boost: 255 }]
  if (key === "max" || key === "4") return [{ temp: 0, boost: 255 }, { temp: 110, boost: 255 }]
  return defaultCurve()
}

function presetNames() {
  return ["silent", "balanced", "cool", "max"]
}

function canInsertPoint(points) {
  return normalizeCurve(points).length < MAX_POINTS
}

function insertPoint(points, temp, boost) {
  var list = normalizeCurve(points)
  if (list.length >= MAX_POINTS) return list
  var t = clampInt(temp, TEMP_MIN, TEMP_MAX, TEMP_MIN)
  for (var i = 0; i < list.length; i++) if (list[i].temp === t) return list
  return normalizeCurve(list.concat([{ temp: t, boost: clampBoost(boost) }]))
}

function removePoint(points, index) {
  var list = normalizeCurve(points)
  var i = toInt(index, -1)
  if (list.length <= MIN_POINTS) return list
  if (i < 0 || i >= list.length) return list
  return list.slice(0, i).concat(list.slice(i + 1))
}

function setPoint(points, index, temp, boost) {
  var list = normalizeCurve(points)
  var i = toInt(index, -1)
  if (i < 0 || i >= list.length) return list
  var lowerBound = i > 0 ? list[i - 1].temp + 1 : TEMP_MIN
  var upperBound = i < list.length - 1 ? list[i + 1].temp - 1 : TEMP_MAX
  if (upperBound < lowerBound) upperBound = lowerBound
  var out = list.slice()
  out[i] = {
    temp: clampInt(temp, lowerBound, upperBound, list[i].temp),
    boost: clampBoost(boost)
  }
  return out
}

function movePoint(points, index, deltaTemp, deltaBoost) {
  var list = normalizeCurve(points)
  var i = toInt(index, -1)
  if (i < 0 || i >= list.length) return list
  return setPoint(list, i, list[i].temp + toInt(deltaTemp, 0), list[i].boost + toInt(deltaBoost, 0))
}

function interpolateBoost(points, temp) {
  var list = normalizeCurve(points)
  var t = toNumber(temp, NaN)
  if (!isFinite(t)) return 0
  if (t <= list[0].temp) return list[0].boost
  var last = list[list.length - 1]
  if (t >= last.temp) return last.boost
  for (var i = 1; i < list.length; i++) {
    var a = list[i - 1]
    var b = list[i]
    if (t > b.temp) continue
    var span = b.temp - a.temp
    if (span <= 0) return b.boost
    return Math.round(a.boost + (b.boost - a.boost) * (t - a.temp) / span)
  }
  return last.boost
}

function curveSeries(points, from, to, step) {
  var start = clampInt(from, TEMP_MIN, TEMP_MAX, TEMP_MIN)
  var end = clampInt(to, TEMP_MIN, TEMP_MAX, TEMP_MAX)
  var inc = Math.max(1, toInt(step, 5))
  if (end < start) { var swap = start; start = end; end = swap }
  var out = []
  for (var t = start; t <= end; t += inc) out.push({ temp: t, boost: interpolateBoost(points, t) })
  if (out.length === 0 || out[out.length - 1].temp !== end) out.push({ temp: end, boost: interpolateBoost(points, end) })
  return out
}

function curveOverlay(points, floorPercent) {
  var floor = clampInt(floorPercent, 0, 100, 0)
  var series = curveSeries(points, TEMP_MIN, TEMP_MAX, 5)
  var out = []
  for (var i = 0; i < series.length; i++) {
    var added = boostPercent(series[i].boost)
    out.push({
      temp: series[i].temp,
      boost: series[i].boost,
      floor: floor,
      added: added,
      total: clamp(floor + added, 0, 100)
    })
  }
  return out
}

function validateCurve(curve) {
  var src = isObject(curve) ? curve : {}
  var cpu = normalizeCurve(src.cpu)
  var gpu = normalizeCurve(src.gpu)
  if (cpu.length < MIN_POINTS) return { ok: false, error: "The CPU curve needs at least two points" }
  if (gpu.length < MIN_POINTS) return { ok: false, error: "The GPU curve needs at least two points" }
  if (cpu.length > MAX_POINTS || gpu.length > MAX_POINTS) return { ok: false, error: "A curve may hold at most " + MAX_POINTS + " points" }
  return { ok: true, error: "" }
}

function buildCurvePayload(interval, hysteresis, cpu, gpu) {
  return {
    interval: clampInterval(interval),
    hysteresis: clampHysteresis(hysteresis),
    cpu: normalizeCurve(cpu),
    gpu: normalizeCurve(gpu)
  }
}

function curveJson(interval, hysteresis, cpu, gpu) {
  return JSON.stringify(buildCurvePayload(interval, hysteresis, cpu, gpu))
}

function validHex(value) {
  return /^#?([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(String(value === undefined || value === null ? "" : value).trim())
}

function normalizeHex(value) {
  var s = String(value === undefined || value === null ? "" : value).trim()
  if (!validHex(s)) return ""
  s = s.replace(/^#/, "")
  if (s.length === 3) s = s.charAt(0) + s.charAt(0) + s.charAt(1) + s.charAt(1) + s.charAt(2) + s.charAt(2)
  return s.toUpperCase()
}

function colorToHex(value) {
  var s = String(value === undefined || value === null ? "" : value).trim().replace(/^#/, "")
  if (/^[0-9a-fA-F]{8}$/.test(s)) return s.slice(2).toUpperCase()
  return normalizeHex(s)
}

function hexToRgb(value) {
  var hex = normalizeHex(value)
  if (!hex) return null
  return {
    r: parseInt(hex.slice(0, 2), 16),
    g: parseInt(hex.slice(2, 4), 16),
    b: parseInt(hex.slice(4, 6), 16)
  }
}

function hexPreview(value) {
  var hex = normalizeHex(value)
  return hex ? "#" + hex : ""
}

var THEME_SWATCH_NAMES = ["accent", "red", "orange", "yellow", "green", "cyan", "blue", "purple"]

function capitalizeLabel(name) {
  var s = String(name || "")
  return s ? s.charAt(0).toUpperCase() + s.slice(1) : ""
}

function hexToHsv(value) {
  var rgb = hexToRgb(value)
  if (!rgb) return null
  var r = rgb.r / 255
  var g = rgb.g / 255
  var b = rgb.b / 255
  var max = Math.max(r, g, b)
  var min = Math.min(r, g, b)
  var delta = max - min
  var h = 0
  if (delta > 0) {
    if (max === r) h = 60 * (((g - b) / delta) % 6)
    else if (max === g) h = 60 * ((b - r) / delta + 2)
    else h = 60 * ((r - g) / delta + 4)
    if (h < 0) h += 360
  }
  var s = max > 0 ? delta / max * 100 : 0
  var v = max * 100
  return { h: h, s: s, v: v }
}

function hsvToHex(h, s, v) {
  var hh = ((toNumber(h, 0) % 360) + 360) % 360
  var ss = clamp(toNumber(s, 0), 0, 100) / 100
  var vv = clamp(toNumber(v, 0), 0, 100) / 100
  var c = vv * ss
  var x = c * (1 - Math.abs((hh / 60) % 2 - 1))
  var m = vv - c
  var r = 0
  var g = 0
  var b = 0
  if (hh < 60) { r = c; g = x; b = 0 }
  else if (hh < 120) { r = x; g = c; b = 0 }
  else if (hh < 180) { r = 0; g = c; b = x }
  else if (hh < 240) { r = 0; g = x; b = c }
  else if (hh < 300) { r = x; g = 0; b = c }
  else { r = c; g = 0; b = x }
  var toByte = function(channel) { return clampInt(Math.round((channel + m) * 255), 0, 255, 0) }
  var toHexPart = function(n) { return ("0" + n.toString(16)).slice(-2) }
  return (toHexPart(toByte(r)) + toHexPart(toByte(g)) + toHexPart(toByte(b))).toUpperCase()
}

function pointToHueSat(x, y) {
  var xx = toNumber(x, 0)
  var yy = toNumber(y, 0)
  var r = Math.sqrt(xx * xx + yy * yy)
  var s = clamp(r, 0, 1) * 100
  var h = 0
  if (r > 0) {
    h = Math.atan2(yy, xx) * 180 / Math.PI
    if (h < 0) h += 360
  }
  return { h: h, s: s }
}

function hueSatToPoint(h, s) {
  var hh = ((toNumber(h, 0) % 360) + 360) % 360
  var ss = clamp(toNumber(s, 0), 0, 100) / 100
  var rad = hh * Math.PI / 180
  return { x: ss * Math.cos(rad), y: ss * Math.sin(rad) }
}

function parseThemePalette(text) {
  var lines = String(text === undefined || text === null ? "" : text).split("\n")
  var map = {}
  var order = []
  for (var i = 0; i < lines.length; i++) {
    var line = lines[i].replace(/\r$/, "")
    if (!line.trim()) continue
    var parts = line.split("\t")
    if (parts.length !== 2) continue
    var name = parts[0].trim()
    var hexValue = parts[1].trim()
    if (!name || !/^#[0-9a-fA-F]{6}$/.test(hexValue)) continue
    if (!(name in map)) order.push(name)
    map[name] = normalizeHex(hexValue)
  }
  return { map: map, order: order }
}

function themeSwatches(text) {
  var parsed = parseThemePalette(text)
  var out = []
  var seenHex = {}
  for (var i = 0; i < THEME_SWATCH_NAMES.length; i++) {
    var name = THEME_SWATCH_NAMES[i]
    var hex = parsed.map[name]
    if (!hex || seenHex[hex]) continue
    seenHex[hex] = true
    out.push({ id: name, label: capitalizeLabel(name), hex: hex })
  }
  return out
}

function regionIds() {
  var out = []
  for (var i = 0; i < REGIONS.length; i++) out.push(REGIONS[i].id)
  return out
}

function regionById(id) {
  var want = String(id || "")
  for (var i = 0; i < REGIONS.length; i++) if (REGIONS[i].id === want) return REGIONS[i]
  return null
}

function normalizeRegion(raw, fallback) {
  var src = isObject(raw) ? raw : {}
  return {
    id: String(src.id || (fallback ? fallback.id : "")),
    name: String(src.name || (fallback ? fallback.name : "")),
    ledCount: toInt(src.ledCount, fallback ? fallback.ledCount : 0)
  }
}

function normalizeRegions(raw) {
  var list = toList(raw)
  var byId = {}
  for (var i = 0; i < list.length; i++) {
    var r = isObject(list[i]) ? list[i] : {}
    var id = String(r.id || "")
    if (id) byId[id] = r
  }
  var out = []
  for (var i = 0; i < REGIONS.length; i++) {
    var fallback = REGIONS[i]
    out.push(normalizeRegion(byId[fallback.id], fallback))
  }
  return out
}

function normalizeRegionOnMap(raw) {
  var src = isObject(raw) ? raw : {}
  var ids = regionIds()
  var out = {}
  for (var i = 0; i < ids.length; i++) {
    out[ids[i]] = src[ids[i]] === false ? false : true
  }
  return out
}

function normalizeRgbStatus(raw) {
  var src = isObject(raw) ? raw : {}
  var dev = isObject(src.device) ? src.device : {}
  return {
    ok: src.ok !== false,
    backend: String(src.backend || ""),
    device: {
      path: String(dev.path || ""),
      vendorId: String(dev.vendorId || ""),
      productId: String(dev.productId || ""),
      ready: dev.ready === true
    },
    regions: normalizeRegions(src.regions)
  }
}

function regionColorPayload(regions) {
  var src = isObject(regions) ? regions : {}
  var ids = regionIds()
  var out = {}
  for (var i = 0; i < ids.length; i++) {
    var hex = normalizeHex(src[ids[i]])
    if (hex) out[ids[i]] = hex
  }
  return out
}

function effectiveRegionColors(regions, regionsOn) {
  var src = isObject(regions) ? regions : {}
  var onMap = normalizeRegionOnMap(regionsOn)
  var ids = regionIds()
  var out = {}
  for (var i = 0; i < ids.length; i++) {
    var id = ids[i]
    if (onMap[id] === false) {
      out[id] = "000000"
      continue
    }
    var hex = normalizeHex(src[id])
    if (hex) out[id] = hex
  }
  return out
}

function blankRegionMap(ids) {
  var list = toList(ids)
  var valid = regionIds()
  var out = {}
  for (var i = 0; i < list.length; i++) {
    if (valid.indexOf(String(list[i])) < 0) continue
    out[String(list[i])] = "000000"
  }
  return out
}

function regionPowerPayload(regions, regionsOn, ids, lightsOn) {
  if (lightsOn === true) return effectiveRegionColors(regions, regionsOn)
  return blankRegionMap(ids)
}

function hasAnyKey(obj) {
  if (!isObject(obj)) return false
  for (var k in obj) return true
  return false
}

function restoreSequence(state) {
  var out = []
  if (!isObject(state) || state.lightsOn !== true) return out
  out.push({ argv: cmdRgbBrightness(state.brightness), label: "Brightness" })
  var map = effectiveRegionColors(state.regions, state.regionsOn)
  if (hasAnyKey(map)) out.push({ argv: cmdRgbSetMap(map), label: "Colour" })
  return out
}

function keyboardAllKeys() {
  var out = []
  for (var r = 0; r < KEYBOARD_ROWS.length; r++) out = out.concat(KEYBOARD_ROWS[r])
  return out.concat(KEYBOARD_MEDIA_COLUMN)
}

function keyboardRowWeight(row) {
  var list = toList(row)
  var sum = 0
  for (var i = 0; i < list.length; i++) sum += toNumber(list[i] ? list[i].w : 0, 0)
  return sum || 1
}

function isKeyPaintable(key) {
  return !!key && typeof key.index === "number" && key.index >= 0
}

function keyboardKeyById(id) {
  var want = String(id || "")
  var all = keyboardAllKeys()
  for (var i = 0; i < all.length; i++) if (all[i].id === want) return all[i]
  return null
}

function keyboardPaintableIds() {
  var all = keyboardAllKeys()
  var out = []
  for (var i = 0; i < all.length; i++) if (isKeyPaintable(all[i])) out.push(all[i].id)
  return out
}

function themeKeyColorMap(hex) {
  var clean = normalizeHex(hex)
  var map = {}
  if (!clean) return map
  var ids = keyboardPaintableIds()
  for (var i = 0; i < ids.length; i++) map[ids[i]] = clean
  return map
}

function normalizeKeyColorMap(raw) {
  var src = isObject(raw) ? raw : {}
  var ids = keyboardPaintableIds()
  var out = {}
  for (var i = 0; i < ids.length; i++) {
    var hex = normalizeHex(src[ids[i]])
    if (hex) out[ids[i]] = hex
  }
  return out
}

function normalizeKeyOnMap(raw) {
  var src = isObject(raw) ? raw : {}
  var ids = keyboardPaintableIds()
  var out = {}
  for (var i = 0; i < ids.length; i++) out[ids[i]] = src[ids[i]] === false ? false : true
  return out
}

function effectiveKeyColors(keys, keysOn) {
  var src = isObject(keys) ? keys : {}
  var onMap = normalizeKeyOnMap(keysOn)
  var ids = keyboardPaintableIds()
  var out = {}
  for (var i = 0; i < ids.length; i++) {
    var id = ids[i]
    if (onMap[id] === false) { out[id] = "000000"; continue }
    var hex = normalizeHex(src[id])
    if (hex) out[id] = hex
  }
  return out
}

function blankKeyMap(ids) {
  var list = toList(ids)
  var out = {}
  for (var i = 0; i < list.length; i++) {
    if (!isKeyPaintable(keyboardKeyById(list[i]))) continue
    out[String(list[i])] = "000000"
  }
  return out
}

function keyPowerPayload(keys, keysOn, ids, lightsOn) {
  if (lightsOn === true) return effectiveKeyColors(keys, keysOn)
  return blankKeyMap(ids)
}

function keyboardRestoreSequence(state) {
  var out = []
  if (!isObject(state) || state.lightsOn !== true) return out
  var map = effectiveKeyColors(state.keys, state.keysOn)
  if (hasAnyKey(map)) out.push({ argv: cmdKbdSetMap(map), label: "Keyboard colour" })
  return out
}

function normalizeKbdStatus(raw) {
  var src = isObject(raw) ? raw : {}
  return {
    ok: src.ok !== false,
    present: src.present === true,
    keyCount: toInt(src.keyCount, 0)
  }
}

function paletteIsFlat(swatches) {
  var list = toList(swatches)
  if (list.length < 2) return false
  var minR = 255, minG = 255, minB = 255
  var maxR = 0, maxG = 0, maxB = 0
  var seen = 0
  for (var i = 0; i < list.length; i++) {
    var rgb = hexToRgb(list[i] && list[i].hex)
    if (!rgb) continue
    seen++
    minR = Math.min(minR, rgb.r); maxR = Math.max(maxR, rgb.r)
    minG = Math.min(minG, rgb.g); maxG = Math.max(maxG, rgb.g)
    minB = Math.min(minB, rgb.b); maxB = Math.max(maxB, rgb.b)
  }
  if (seen < 2) return false
  var spread = Math.max(maxR - minR, maxG - minG, maxB - minB)
  return spread < 24
}

function powerLimitWritable(constraint) {
  if (!isObject(constraint)) return false
  if (constraint.writable !== true) return false
  var idx = toInt(constraint.index, -1)
  return idx >= 0 && idx <= 2
}

function powerLimitNote(constraint) {
  if (!isObject(constraint)) return ""
  if (constraint.writable !== true) return PL_LOCKED_NOTE
  return ""
}

function isKnownPreset(name) {
  var list = presetNames()
  for (var i = 0; i < list.length; i++) if (list[i] === String(name || "")) return true
  return false
}

function brightnessStep(current, delta, step) {
  var s = Math.max(1, toInt(step, 5))
  return clampBrightness(clampBrightness(current) + toInt(delta, 0) * s)
}

function statusAge(ts, nowMs) {
  var t = toNumber(ts, 0)
  if (!(t > 0)) return NaN
  var now = toNumber(nowMs, Date.now())
  return Math.max(0, Math.round(now / 1000 - t))
}

function isStale(ts, nowMs, interval) {
  var age = statusAge(ts, nowMs)
  if (!isFinite(age)) return true
  return age > Math.max(6, clampInterval(interval) * 3)
}

function statusHealth(hasStatus, lastResult, stale) {
  if (lastResult && lastResult.ok === false && lastResult.code === "no-binary") return "missing"
  if (!hasStatus) {
    if (lastResult && lastResult.ok === false) return "error"
    return "missing"
  }
  if (lastResult && lastResult.ok === false) return "error"
  if (stale === true) return "stale"
  return "ok"
}

function healthLine(health, lastResult, status) {
  if (health === "missing") {
    if (lastResult && lastResult.error) return lastResult.error
    return "alienwarectl is not installed"
  }
  if (health === "error") return lastResult && lastResult.error ? lastResult.error : "Could not read the hardware"
  if (health === "stale") return "Readings are stale"
  if (status && toList(status.warnings).length) return String(status.warnings[0])
  return "Ready"
}

function barState(status, health, display, hot) {
  var mode = normalizeDisplay(display)
  var threshold = clampHot(hot)
  var view = mode === "full" ? "expanded" : (mode === "temp" ? "compact" : "icon")
  var h = String(health || "missing")
  if (h === "missing" || h === "error") {
    return {
      state: h,
      view: "icon",
      text: "",
      tone: h === "missing" ? "muted" : "warn",
      glyph: GLYPHS.missing
    }
  }
  var temp = hottest(status)
  var cpuTemp = status && typeof status.temps.cpu === "number" ? status.temps.cpu : NaN
  var gpuTemp = status && typeof status.temps.gpu === "number" ? status.temps.gpu : NaN
  var warning = isFinite(temp) && temp >= threshold
  var stopped = fansStopped(status)
  var state = warning ? "warning" : (stopped ? "stopped" : (h === "stale" ? "stale" : "ok"))
  if (warning && view === "icon") view = "compact"
  var text = ""
  if (view === "compact") {
    text = formatTemp(isFinite(cpuTemp) ? cpuTemp : temp, false)
  } else if (view === "expanded") {
    var parts = []
    if (status && status.profile.current) parts.push(profileLabel(status.profile.current, status.profile.gmodeForced))
    parts.push(stopped ? "0 rpm" : maxRpm(status) + " rpm")
    if (isFinite(gpuTemp)) parts.push(formatTemp(gpuTemp, false))
    else if (isFinite(cpuTemp)) parts.push(formatTemp(cpuTemp, false))
    text = parts.join(" · ")
  }
  var tone = "normal"
  if (state === "warning") tone = "warn"
  else if (state === "stale") tone = "muted"
  return {
    state: state,
    view: view,
    text: text,
    tone: tone,
    glyph: warning ? GLYPHS.hot : GLYPHS.alien
  }
}

function barTooltip(status, health, lastResult) {
  var parts = []
  if (health === "missing" || health === "error" || !status) {
    parts.push(healthLine(health, lastResult, status))
    return parts.join(" · ")
  }
  if (status.profile.current) parts.push(profileLabel(status.profile.current, status.profile.gmodeForced))
  var fans = toList(status.fans)
  for (var i = 0; i < fans.length; i++) {
    var f = fans[i]
    if (!f) continue
    parts.push(f.label + " " + formatRpm(f.rpm))
  }
  if (typeof status.temps.cpu === "number") parts.push("CPU " + formatTemp(status.temps.cpu, true))
  if (typeof status.temps.gpu === "number") parts.push("GPU " + formatTemp(status.temps.gpu, true))
  if (status.curve.active) parts.push("Curve active")
  if (health === "stale") parts.push("Stale")
  return parts.join(" · ")
}

function parseState(raw) {
  var parsed = parseJson(raw)
  var src = isObject(parsed) ? parsed : {}
  return {
    version: toInt(src.version, 1),
    regions: regionColorPayload(src.regions),
    regionsOn: normalizeRegionOnMap(src.regionsOn),
    keys: normalizeKeyColorMap(src.keys),
    keysOn: normalizeKeyOnMap(src.keysOn),
    color: normalizeHex(src.color),
    brightness: src.brightness === undefined || src.brightness === null ? 100 : clampBrightness(src.brightness),
    lightsOn: src.lightsOn !== false,
    themeSync: src.themeSync === true,
    curve: {
      interval: clampInterval(src.curve && src.curve.interval),
      hysteresis: clampHysteresis(src.curve && src.curve.hysteresis),
      cpu: src.curve && src.curve.cpu ? normalizeCurve(src.curve.cpu) : defaultCurve(),
      gpu: src.curve && src.curve.gpu ? normalizeCurve(src.curve.gpu) : defaultCurve()
    }
  }
}

function buildStatePayload(state) {
  var src = isObject(state) ? state : {}
  return {
    version: 2,
    savedAt: toInt(src.savedAt, 0),
    regions: regionColorPayload(src.regions),
    regionsOn: normalizeRegionOnMap(src.regionsOn),
    keys: normalizeKeyColorMap(src.keys),
    keysOn: normalizeKeyOnMap(src.keysOn),
    color: normalizeHex(src.color),
    brightness: clampBrightness(src.brightness),
    lightsOn: src.lightsOn !== false,
    themeSync: src.themeSync === true,
    curve: buildCurvePayload(
      src.curve ? src.curve.interval : 2,
      src.curve ? src.curve.hysteresis : 3,
      src.curve ? src.curve.cpu : null,
      src.curve ? src.curve.gpu : null)
  }
}

function cmdStatus() { return ["alienwarectl", "status"] }
function cmdProfile(name) { return ["alienwarectl", "profile", String(name || "")] }
function cmdBoost(fan, value) { return ["alienwarectl", "boost", String(fan || "cpu"), String(clampBoost(value))] }
function cmdCurveApply() { return ["alienwarectl", "curve", "apply", "-"] }
function cmdCurveStop() { return ["alienwarectl", "curve", "stop"] }
function cmdTurbo(on) { return ["alienwarectl", "turbo", on ? "on" : "off"] }
function cmdPl(index, watts) { return ["alienwarectl", "pl", String(clampInt(index, 1, 3, 1)), String(Math.max(1, toInt(watts, 1)))] }
function cmdGpu() { return ["alienwarectl", "gpu"] }
function cmdRgbStatus() { return ["alienwarectl", "rgb", "status"] }
function cmdRgbSet(regionId, hex) { return ["alienwarectl", "rgb", "set", String(regionId || ""), normalizeHex(hex)] }
function cmdRgbSetAll(hex) { return ["alienwarectl", "rgb", "set-all", normalizeHex(hex)] }
function cmdRgbSetMap(map) {
  var ids = regionIds()
  var parts = []
  var src = isObject(map) ? map : {}
  for (var i = 0; i < ids.length; i++) {
    var hex = normalizeHex(src[ids[i]])
    if (!hex) continue
    parts.push(ids[i] + "=" + hex)
  }
  return ["alienwarectl", "rgb", "set-map", parts.join(",")]
}
function cmdRgbBrightness(value) { return ["alienwarectl", "rgb", "brightness", String(clampBrightness(value))] }
function cmdRgbIdentify(regionId) { return ["alienwarectl", "rgb", "identify", String(regionId || "")] }
function cmdRgbOff() { return ["alienwarectl", "rgb", "off"] }
function cmdKbdStatus() { return ["alienwarectl", "kbd", "status"] }
function cmdKbdSetMap(map) {
  var src = isObject(map) ? map : {}
  var pairs = []
  for (var id in src) {
    var key = keyboardKeyById(id)
    if (!isKeyPaintable(key)) continue
    var hex = normalizeHex(src[id])
    if (!hex) continue
    var idxs = keyIndices(key)
    for (var n = 0; n < idxs.length; n++) pairs.push({ index: idxs[n], hex: hex })
  }
  pairs.sort(function(a, b) { return a.index - b.index })
  var parts = []
  for (var i = 0; i < pairs.length; i++) parts.push(pairs[i].index + "=" + pairs[i].hex)
  return ["alienwarectl", "kbd", "set-map", parts.join(",")]
}
function cmdKbdSetAll(hex) { return ["alienwarectl", "kbd", "set-all", normalizeHex(hex)] }
function cmdKbdOff() { return ["alienwarectl", "kbd", "off"] }
function cmdVersion() { return ["alienwarectl", "version"] }

function queueKey(argv) {
  var list = toList(argv)
  if (list[0] === "alienwarectl" && list[1] === "rgb" && list[2] === "set" && list.length >= 4) {
    return list.slice(0, 4).join(" ")
  }
  return list.slice(0, 3).join(" ")
}

function elcReadAction(state) {
  var s = isObject(state) ? state : {}
  if (s.readRunning === true) return "running"
  if (s.writeRunning === true || s.queued === true) return "defer"
  return "start"
}

function elcWriteAction(state) {
  var s = isObject(state) ? state : {}
  if (s.writeRunning === true) return "running"
  if (s.queued !== true) return "idle"
  if (s.readRunning === true) return "defer"
  return "start"
}
