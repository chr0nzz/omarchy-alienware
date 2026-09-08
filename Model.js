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

var RGB_MODES = ["Static", "Flashing", "Morph", "Spectrum Cycle", "Rainbow Wave", "Breathing"]

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
  "no-openrgb": "OpenRGB server is not reachable",
  "internal": "The helper hit an internal error"
}

var CURVE_NOTE = "Boost is additive. It can only push the fans above the firmware curve, never below it."
var PL_LOCKED_NOTE = "Locked by firmware"
var ZONE_WIZARD_NOTE = "This laptop reports generic slots. Light each one and name the ones you can see."

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

function normalizeMode(name, available) {
  var list = toList(available)
  if (!list.length) list = RGB_MODES.slice()
  var want = String(name || "").trim().toLowerCase()
  for (var i = 0; i < list.length; i++) {
    if (String(list[i] || "").trim().toLowerCase() === want) return String(list[i])
  }
  return String(list[0])
}

function modeList(rgb) {
  var list = toList(rgb && rgb.device ? rgb.device.modes : null)
  var out = []
  for (var i = 0; i < list.length; i++) {
    var m = String(list[i] || "").trim()
    if (m && out.indexOf(m) === -1) out.push(m)
  }
  return out.length ? out : RGB_MODES.slice()
}

function nextMode(available, current, direction) {
  var list = modeList({ device: { modes: available } })
  var step = toInt(direction, 1)
  if (step === 0) step = 1
  var idx = list.indexOf(normalizeMode(current, list))
  if (idx === -1) idx = 0
  var next = (idx + step) % list.length
  if (next < 0) next += list.length
  return list[next]
}

function normalizeRgbStatus(raw) {
  var src = isObject(raw) ? raw : {}
  var dev = isObject(src.device) ? src.device : {}
  var zoneList = toList(src.zones)
  var zones = []
  for (var i = 0; i < zoneList.length; i++) {
    var z = isObject(zoneList[i]) ? zoneList[i] : {}
    zones.push({
      index: toInt(z.index, i),
      name: String(z.name || ""),
      ledCount: toInt(z.ledCount, 0)
    })
  }
  var count = toInt(dev.zoneCount, zones.length)
  return {
    ok: src.ok !== false,
    connected: src.connected === true,
    server: String(src.server || ""),
    device: {
      index: toInt(dev.index, 0),
      name: String(dev.name || ""),
      zoneCount: Math.max(0, count),
      ledCount: toInt(dev.ledCount, 0),
      activeMode: String(dev.activeMode || ""),
      modes: modeList({ device: dev }),
      colorModes: stringList(dev.colorModes)
    },
    zones: zones
  }
}

function stringList(value) {
  var list = toList(value)
  var out = []
  for (var i = 0; i < list.length; i++) {
    var name = String(list[i] === undefined || list[i] === null ? "" : list[i]).trim()
    if (name) out.push(name)
  }
  return out
}

function modeTakesColor(mode, colorModes) {
  var want = String(mode || "").trim()
  if (!want) return false
  var list = stringList(colorModes)
  for (var i = 0; i < list.length; i++) if (list[i] === want) return true
  return false
}

function restoreSequence(state) {
  var out = []
  if (!isObject(state) || state.lightsOn !== true) return out
  var mode = String(state.mode || "").trim()
  var hex = normalizeHex(state.color)
  if (mode) out.push({ argv: cmdRgbMode(mode), label: "Effect " + mode })
  out.push({ argv: cmdRgbBrightness(state.brightness), label: "Brightness" })
  if (hex && modeTakesColor(mode, state.colorModes)) {
    out.push({ argv: cmdRgbSetAll(hex), label: "Colour" })
  }
  return out
}

function slotLabel(index) {
  return "Slot " + (toInt(index, 0) + 1)
}

function mergeZones(deviceZones, savedZones, zoneCount) {
  var device = toList(deviceZones)
  var saved = toList(savedZones)
  var count = toInt(zoneCount, NaN)
  if (!isFinite(count) || count < 0) count = device.length
  var byIndex = {}
  for (var s = 0; s < saved.length; s++) {
    var sv = isObject(saved[s]) ? saved[s] : null
    if (!sv) continue
    var si = toInt(sv.index, NaN)
    if (!isFinite(si) || si < 0) continue
    byIndex[si] = sv
  }
  var out = []
  for (var i = 0; i < count; i++) {
    var dev = isObject(device[i]) ? device[i] : {}
    var devName = String(dev.name || "")
    var savedEntry = byIndex[i]
    var name = savedEntry ? String(savedEntry.name || "").trim() : ""
    var known = name !== ""
    out.push({
      index: i,
      name: name,
      deviceName: devName,
      ledCount: toInt(dev.ledCount, 0),
      known: known,
      enabled: known && (savedEntry.enabled !== false),
      label: known ? name : (devName && devName.toLowerCase() !== "unknown" ? devName : slotLabel(i))
    })
  }
  return out
}

function zoneConflicts(deviceZones, savedZones, zoneCount) {
  var device = toList(deviceZones)
  var count = toInt(zoneCount, NaN)
  if (!isFinite(count) || count < 0) count = device.length
  var dropped = []
  var saved = toList(savedZones)
  for (var i = 0; i < saved.length; i++) {
    var sv = isObject(saved[i]) ? saved[i] : null
    if (!sv) continue
    var idx = toInt(sv.index, NaN)
    if (!isFinite(idx) || idx < 0 || idx >= count) dropped.push(sv)
  }
  return { dropped: dropped, deviceCount: count, savedCount: saved.length }
}

function namedZones(zones) {
  var list = toList(zones)
  var out = []
  for (var i = 0; i < list.length; i++) if (list[i] && list[i].enabled) out.push(list[i])
  return out
}

function unnamedZones(zones) {
  var list = toList(zones)
  var out = []
  for (var i = 0; i < list.length; i++) if (list[i] && !list[i].known) out.push(list[i])
  return out
}

function zonesNeedWizard(zones) {
  return namedZones(zones).length === 0 && toList(zones).length > 0
}

function zoneMapPayload(zones) {
  var list = toList(zones)
  var out = []
  for (var i = 0; i < list.length; i++) {
    var z = list[i]
    if (!z || !z.known) continue
    out.push({ index: toInt(z.index, i), name: String(z.name || ""), enabled: z.enabled !== false })
  }
  return out
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

function unionZoneMaps(primary, extra) {
  var out = []
  var seen = {}
  var a = toList(primary)
  for (var i = 0; i < a.length; i++) {
    var p = a[i]
    if (!p) continue
    var pi = toInt(p.index, -1)
    if (pi < 0 || seen[pi]) continue
    seen[pi] = true
    out.push({ index: pi, name: String(p.name || ""), enabled: p.enabled !== false })
  }
  var b = toList(extra)
  for (var k = 0; k < b.length; k++) {
    var q = b[k]
    if (!q) continue
    var qi = toInt(q.index, -1)
    if (qi < 0 || seen[qi]) continue
    if (!String(q.name || "").trim()) continue
    seen[qi] = true
    out.push({ index: qi, name: String(q.name || ""), enabled: q.enabled !== false })
  }
  out.sort(function(x, y) { return x.index - y.index })
  return out
}

function isKnownPreset(name) {
  var list = presetNames()
  for (var i = 0; i < list.length; i++) if (list[i] === String(name || "")) return true
  return false
}

function setZoneName(zones, index, name) {
  var list = toList(zones)
  var i = toInt(index, -1)
  var clean = String(name === undefined || name === null ? "" : name).trim()
  var out = []
  for (var k = 0; k < list.length; k++) {
    var z = list[k]
    if (!z || z.index !== i) { out.push(z); continue }
    out.push({
      index: z.index,
      name: clean,
      deviceName: z.deviceName,
      ledCount: z.ledCount,
      known: clean !== "",
      enabled: clean !== "",
      label: clean !== "" ? clean : (z.deviceName && z.deviceName.toLowerCase() !== "unknown" ? z.deviceName : slotLabel(z.index))
    })
  }
  return out
}

function toggleZoneEnabled(zones, index) {
  var list = toList(zones)
  var i = toInt(index, -1)
  var out = []
  for (var k = 0; k < list.length; k++) {
    var z = list[k]
    if (!z || z.index !== i || !z.known) { out.push(z); continue }
    out.push({
      index: z.index,
      name: z.name,
      deviceName: z.deviceName,
      ledCount: z.ledCount,
      known: z.known,
      enabled: !z.enabled,
      label: z.label
    })
  }
  return out
}

function nextSlot(zones, current, direction) {
  var list = toList(zones)
  if (!list.length) return -1
  var step = toInt(direction, 1)
  if (step === 0) step = 1
  var idx = -1
  for (var i = 0; i < list.length; i++) if (list[i] && list[i].index === toInt(current, -1)) idx = i
  if (idx === -1) return list[0].index
  var next = (idx + step) % list.length
  if (next < 0) next += list.length
  return list[next].index
}

function wizardProgress(zones, cursor) {
  var list = toList(zones)
  var total = list.length
  var pos = 0
  for (var i = 0; i < list.length; i++) if (list[i] && list[i].index === toInt(cursor, -1)) pos = i
  return {
    total: total,
    position: total ? pos + 1 : 0,
    named: namedZones(list).length,
    done: total > 0 && pos + 1 >= total
  }
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
  else if (state === "stopped" || state === "stale") tone = "muted"
  return {
    state: state,
    view: view,
    text: text,
    tone: tone,
    glyph: warning ? GLYPHS.hot : (stopped ? GLYPHS.idle : GLYPHS.fan)
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
  var zones = []
  var list = toList(src.zones)
  for (var i = 0; i < list.length; i++) {
    var z = isObject(list[i]) ? list[i] : null
    if (!z) continue
    var idx = toInt(z.index, NaN)
    if (!isFinite(idx) || idx < 0) continue
    var name = String(z.name || "").trim()
    if (!name) continue
    zones.push({ index: idx, name: name, enabled: z.enabled !== false })
  }
  return {
    version: toInt(src.version, 1),
    zones: zones,
    color: normalizeHex(src.color),
    mode: String(src.mode || ""),
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
    version: 1,
    savedAt: toInt(src.savedAt, 0),
    zones: zoneMapPayload(src.zones),
    color: normalizeHex(src.color),
    mode: String(src.mode || ""),
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
function cmdRgbSet(zone, hex) { return ["alienwarectl", "rgb", "set", String(toInt(zone, 0)), normalizeHex(hex)] }
function cmdRgbSetAll(hex) { return ["alienwarectl", "rgb", "set-all", normalizeHex(hex)] }
function cmdRgbMode(name) { return ["alienwarectl", "rgb", "mode", String(name || "Static")] }
function cmdRgbBrightness(value) { return ["alienwarectl", "rgb", "brightness", String(clampBrightness(value))] }
function cmdRgbIdentify(zone) { return ["alienwarectl", "rgb", "identify", String(toInt(zone, 0))] }
function cmdRgbOff() { return ["alienwarectl", "rgb", "off"] }
function cmdVersion() { return ["alienwarectl", "version"] }
