const test = require("node:test")
const assert = require("node:assert/strict")
const { Model } = require("./load")

const EM_DASH = String.fromCharCode(0x2014)

const GOOD_STATUS = {
  ok: true,
  ts: 1757260000,
  model: "Alienware x15 R2",
  hwmon: "/sys/class/hwmon/hwmon4",
  fans: [
    { id: "cpu", index: 1, label: "CPU Fan", rpm: 2100, max: 5700, boost: 0, percent: 37 },
    { id: "gpu", index: 2, label: "GPU Fan", rpm: 1900, max: 5300, boost: 0, percent: 36 }
  ],
  temps: { cpu: 47, gpu: 43, sodimm: 44, other: 45 },
  profile: {
    current: "balanced",
    choices: ["low-power", "quiet", "balanced", "balanced-performance", "performance", "custom"],
    ppdRunning: true,
    gmodeForced: false,
    writable: true
  },
  turbo: { available: true, enabled: true },
  power: {
    available: true,
    constraints: [
      { index: 0, name: "long_term", watts: 65, writable: false },
      { index: 1, name: "short_term", watts: 140, writable: false },
      { index: 2, name: "peak_power", watts: 215, writable: false }
    ]
  },
  gpu: { available: true, draw: 22.4, limit: 55, defaultLimit: 90, maxLimit: 140, limitWritable: false },
  curve: { active: false, interval: 2, hysteresis: 3, cpu: [{ temp: 40, boost: 0 }], gpu: [{ temp: 40, boost: 0 }] },
  warnings: ["dell_smm hwmon not present"]
}

function status(overrides) {
  return Model.normalizeStatus(Object.assign({}, GOOD_STATUS, overrides || {}))
}

test("toList accepts array-like values and rejects everything else", () => {
  assert.deepEqual(Model.toList([1, 2]), [1, 2])
  assert.deepEqual(Model.toList({ length: 2, 0: "a", 1: "b" }), ["a", "b"])
  assert.deepEqual(Model.toList(null), [])
  assert.deepEqual(Model.toList("abc"), [])
  assert.deepEqual(Model.toList({ a: 1 }), [])
})

test("clampInt falls back for junk and clamps to range", () => {
  assert.equal(Model.clampInt("7", 0, 10, 3), 7)
  assert.equal(Model.clampInt("junk", 0, 10, 3), 3)
  assert.equal(Model.clampInt(99, 0, 10, 3), 10)
  assert.equal(Model.clampInt(-4, 0, 10, 3), 0)
  assert.equal(Model.clampInt(null, 0, 10, 3), 3)
})

test("setting clamps follow the contract ranges", () => {
  assert.equal(Model.clampInterval(0), 1)
  assert.equal(Model.clampInterval(99), 30)
  assert.equal(Model.clampInterval("x"), 2)
  assert.equal(Model.clampHysteresis(99), 15)
  assert.equal(Model.clampHysteresis(-1), 0)
  assert.equal(Model.clampHot("x"), 90)
  assert.equal(Model.clampHot(200), 110)
  assert.equal(Model.clampHot(10), 50)
  assert.equal(Model.clampBoost(999), 255)
  assert.equal(Model.clampBrightness(-5), 0)
})

test("normalizeDisplay maps the three bar modes", () => {
  assert.equal(Model.normalizeDisplay("temp"), "temp")
  assert.equal(Model.normalizeDisplay("Full"), "full")
  assert.equal(Model.normalizeDisplay("expanded"), "full")
  assert.equal(Model.normalizeDisplay("nonsense"), "icon")
  assert.equal(Model.normalizeDisplay(null), "icon")
})

test("parseJson tolerates malformed and empty payloads", () => {
  assert.equal(Model.parseJson(""), null)
  assert.equal(Model.parseJson("{not json"), null)
  assert.equal(Model.parseJson("null"), null)
  assert.equal(Model.parseJson("[1]").length, 1)
  assert.deepEqual(Model.parseJson('{"a":1}'), { a: 1 })
})

test("parseResult reports a missing binary from the shell exit code", () => {
  const r = Model.parseResult("", 127)
  assert.equal(r.ok, false)
  assert.equal(r.code, "no-binary")
  assert.match(r.error, /not installed/)
})

test("parseResult surfaces helper error JSON", () => {
  const r = Model.parseResult('{"ok":false,"error":"daemon not on the bus","code":"no-daemon"}', 1)
  assert.equal(r.ok, false)
  assert.equal(r.code, "no-daemon")
  assert.match(r.error, /daemon is not running/)
  assert.match(r.error, /daemon not on the bus/)
})

test("parseResult treats unparsable output with exit 0 as an internal failure", () => {
  const r = Model.parseResult("segfault", 0)
  assert.equal(r.ok, false)
  assert.equal(r.code, "internal")
})

test("parseResult rejects good JSON with a bad exit code", () => {
  const r = Model.parseResult('{"ok":true}', 3)
  assert.equal(r.ok, false)
  assert.equal(r.code, "internal")
})

test("parseResult passes a healthy payload through", () => {
  const r = Model.parseResult(JSON.stringify(GOOD_STATUS), 0)
  assert.equal(r.ok, true)
  assert.equal(r.data.model, "Alienware x15 R2")
})

test("normalizeStatus keeps the two fans and derives percent and stopped", () => {
  const s = status()
  assert.equal(s.fans.length, 2)
  assert.equal(s.fans[0].id, "cpu")
  assert.equal(s.fans[0].percent, 37)
  assert.equal(s.fans[0].stopped, false)
  assert.equal(s.fansAvailable, true)
})

test("normalizeStatus drops extra fans reported by the helper", () => {
  const s = status({ fans: GOOD_STATUS.fans.concat([{ id: "fan3", rpm: 10, max: 100 }]) })
  assert.equal(s.fans.length, 2)
})

test("normalizeStatus computes percent when the helper omits it", () => {
  const s = status({ fans: [{ id: "cpu", rpm: 2850, max: 5700 }] })
  assert.equal(s.fans[0].percent, 50)
})

test("a fan at zero rpm is idle, not an error", () => {
  const s = status({ fans: [{ id: "cpu", rpm: 0, max: 5700 }, { id: "gpu", rpm: 0, max: 5300 }] })
  assert.equal(s.fans[0].stopped, true)
  assert.equal(s.fans[0].percent, 0)
  assert.equal(Model.fansStopped(s), true)
  assert.equal(Model.formatRpm(0), "stopped")
})

test("fansStopped needs every fan at zero", () => {
  assert.equal(Model.fansStopped(status({ fans: [{ id: "cpu", rpm: 0, max: 5700 }, { id: "gpu", rpm: 900, max: 5300 }] })), false)
  assert.equal(Model.fansStopped(status({ fans: [] })), false)
})

test("normalizeStatus survives complete garbage", () => {
  const s = Model.normalizeStatus(null)
  assert.equal(s.fans.length, 0)
  assert.deepEqual(s.temps, {})
  assert.equal(s.profile.available, false)
  assert.equal(s.power.available, false)
  assert.equal(s.gpu.available, false)
  assert.equal(s.turbo.available, false)
  assert.equal(s.curve.active, false)
  assert.equal(s.curve.cpu.length >= 2, true)
})

test("normalizeStatus keeps missing subsystems marked unavailable", () => {
  const s = Model.normalizeStatus({
    ok: true,
    fans: [],
    temps: { cpu: 50 },
    power: { available: false, constraints: [] },
    gpu: { available: false, draw: null },
    turbo: { available: false },
    warnings: ["no rapl", ""]
  })
  assert.equal(s.power.available, false)
  assert.equal(s.gpu.available, false)
  assert.equal(Number.isFinite(s.gpu.draw), false)
  assert.deepEqual(s.warnings, ["no rapl"])
})

test("temps drop unreadable keys instead of reporting zero", () => {
  const s = status({ temps: { cpu: 47, gpu: null, sodimm: "warm" } })
  assert.equal(s.temps.cpu, 47)
  assert.equal("gpu" in s.temps, false)
  assert.equal("sodimm" in s.temps, false)
})

test("fanById and maxRpm read the normalized fan list", () => {
  const s = status()
  assert.equal(Model.fanById(s, "gpu").rpm, 1900)
  assert.equal(Model.fanById(s, "nope"), null)
  assert.equal(Model.maxRpm(s), 2100)
})

test("hottest takes the higher of cpu and gpu", () => {
  assert.equal(Model.hottest(status()), 47)
  assert.equal(Model.hottest(status({ temps: { gpu: 91 } })), 91)
  assert.equal(Number.isFinite(Model.hottest(status({ temps: {} }))), false)
})

test("fanPercent clamps and survives a zero maximum", () => {
  assert.equal(Model.fanPercent(2850, 5700), 50)
  assert.equal(Model.fanPercent(9000, 5700), 100)
  assert.equal(Model.fanPercent(100, 0), 0)
  assert.equal(Model.fanPercent(null, 5700), 0)
})

test("boost and percent convert both ways", () => {
  assert.equal(Model.boostPercent(255), 100)
  assert.equal(Model.boostPercent(0), 0)
  assert.equal(Model.boostPercent(128), 50)
  assert.equal(Model.percentToBoost(100), 255)
  assert.equal(Model.percentToBoost(0), 0)
  assert.equal(Model.percentToBoost(200), 255)
})

test("formatters degrade to n/a rather than lying", () => {
  assert.equal(Model.formatTemp(47), "47°")
  assert.equal(Model.formatTemp(47, true), "47°C")
  assert.equal(Model.formatTemp(null), "n/a")
  assert.equal(Model.formatRpm(2100), "2100 rpm")
  assert.equal(Model.formatRpm("x"), "n/a")
  assert.equal(Model.formatWatts(22.4, 1), "22.4 W")
  assert.equal(Model.formatWatts(65), "65 W")
  assert.equal(Model.formatWatts(null), "n/a")
  assert.equal(Model.formatPercent(37), "37%")
  assert.equal(Model.formatPercent(null), "n/a")
})

test("cpu clock is optional and never invented", () => {
  assert.equal(Number.isFinite(Model.cpuClock(status())), false)
  assert.equal(Model.formatClock(Model.cpuClock(status())), "not reported")
  assert.equal(Model.formatClock(3800), "3.80 GHz")
  assert.equal(Model.formatClock(800), "800 MHz")
})

test("profileLabel maps every contract choice", () => {
  assert.equal(Model.profileLabel("low-power"), "Low power")
  assert.equal(Model.profileLabel("quiet"), "Quiet")
  assert.equal(Model.profileLabel("balanced"), "Balanced")
  assert.equal(Model.profileLabel("balanced-performance"), "Balanced perf")
  assert.equal(Model.profileLabel("performance"), "Performance")
  assert.equal(Model.profileLabel("custom"), "Custom")
  assert.equal(Model.profileLabel(""), "Unknown")
  assert.equal(Model.profileLabel("brand-new-thing"), "Brand new thing")
})

test("performance is labelled G-Mode when the module forces it", () => {
  assert.equal(Model.profileLabel("performance", true), "G-Mode")
  assert.equal(Model.profileLabel("balanced", true), "Balanced")
})

test("nextProfile cycles both directions and tolerates an unknown current", () => {
  const choices = GOOD_STATUS.profile.choices
  assert.equal(Model.nextProfile(choices, "balanced"), "balanced-performance")
  assert.equal(Model.nextProfile(choices, "custom"), "low-power")
  assert.equal(Model.nextProfile(choices, "low-power", -1), "custom")
  assert.equal(Model.nextProfile(choices, "ghost"), "low-power")
  assert.equal(Model.nextProfile([], "balanced"), "balanced")
})

test("profileChoices falls back to the documented list", () => {
  assert.equal(Model.profileChoices(status()).length, 6)
  assert.equal(Model.profileChoices(Model.normalizeStatus(null)).length, 6)
})

test("profileWarning flags ppd and read only profiles", () => {
  assert.match(Model.profileWarning(status()), /power-profiles-daemon/)
  assert.match(Model.profileWarning(status({ profile: { current: "balanced", choices: ["balanced"], writable: false } })), /read only/)
  assert.equal(Model.profileWarning(status({ profile: { current: "balanced", choices: ["balanced"], writable: true, ppdRunning: false } })), "")
})

test("normalizeCurve sorts, clamps and dedupes points", () => {
  const out = Model.normalizeCurve([
    { temp: 90, boost: 300 },
    { temp: 40, boost: -5 },
    { temp: 40, boost: 20 },
    { temp: 200, boost: 100 }
  ])
  assert.deepEqual(out, [{ temp: 40, boost: 20 }, { temp: 90, boost: 255 }, { temp: 110, boost: 100 }])
})

test("normalizeCurve drops junk entries", () => {
  const out = Model.normalizeCurve([null, "x", { boost: 10 }, { temp: 50, boost: 10 }, { temp: 80, boost: 200 }])
  assert.deepEqual(out, [{ temp: 50, boost: 10 }, { temp: 80, boost: 200 }])
})

test("normalizeCurve returns a usable default for empty input", () => {
  const out = Model.normalizeCurve([])
  assert.equal(out.length >= 2, true)
  assert.deepEqual(out, Model.defaultCurve())
  assert.deepEqual(Model.normalizeCurve(null), Model.defaultCurve())
})

test("normalizeCurve promotes a single point to a real curve", () => {
  const out = Model.normalizeCurve([{ temp: 60, boost: 10 }])
  assert.equal(out.length, 2)
  assert.equal(out[0].temp, 60)
  assert.equal(out[1].temp, 110)
  const top = Model.normalizeCurve([{ temp: 110, boost: 255 }])
  assert.equal(top.length, 2)
  assert.equal(top[0].temp, 0)
})

test("normalizeCurve thins down to the eight point ceiling and keeps the ends", () => {
  const many = []
  for (let t = 0; t <= 110; t += 5) many.push({ temp: t, boost: t * 2 })
  const out = Model.normalizeCurve(many)
  assert.equal(out.length, 8)
  assert.equal(out[0].temp, 0)
  assert.equal(out[out.length - 1].temp, 110)
  for (let i = 1; i < out.length; i++) assert.equal(out[i].temp > out[i - 1].temp, true)
})

test("insertPoint refuses duplicates and respects the ceiling", () => {
  const base = [{ temp: 40, boost: 0 }, { temp: 90, boost: 255 }]
  assert.equal(Model.insertPoint(base, 60, 100).length, 3)
  assert.equal(Model.insertPoint(base, 40, 100).length, 2)
  const full = []
  for (let i = 0; i < 8; i++) full.push({ temp: i * 10, boost: i })
  assert.equal(Model.canInsertPoint(full), false)
  assert.equal(Model.insertPoint(full, 95, 200).length, 8)
})

test("removePoint keeps at least two points", () => {
  const three = [{ temp: 40, boost: 0 }, { temp: 60, boost: 90 }, { temp: 90, boost: 255 }]
  assert.deepEqual(Model.removePoint(three, 1), [{ temp: 40, boost: 0 }, { temp: 90, boost: 255 }])
  const two = [{ temp: 40, boost: 0 }, { temp: 90, boost: 255 }]
  assert.deepEqual(Model.removePoint(two, 0), two)
  assert.deepEqual(Model.removePoint(three, 9), three)
})

test("setPoint cannot drag a point past its neighbours", () => {
  const three = [{ temp: 40, boost: 0 }, { temp: 60, boost: 90 }, { temp: 90, boost: 255 }]
  assert.equal(Model.setPoint(three, 1, 10, 90)[1].temp, 41)
  assert.equal(Model.setPoint(three, 1, 200, 90)[1].temp, 89)
  assert.equal(Model.setPoint(three, 0, -50, 0)[0].temp, 0)
  assert.equal(Model.setPoint(three, 2, 999, 999)[2].temp, 110)
  assert.equal(Model.setPoint(three, 2, 999, 999)[2].boost, 255)
  assert.deepEqual(Model.setPoint(three, 7, 50, 50), three)
})

test("movePoint applies relative steps", () => {
  const three = [{ temp: 40, boost: 0 }, { temp: 60, boost: 90 }, { temp: 90, boost: 255 }]
  assert.equal(Model.movePoint(three, 1, 5, 0)[1].temp, 65)
  assert.equal(Model.movePoint(three, 1, 0, -100)[1].boost, 0)
  assert.equal(Model.movePoint(three, 1, 0, 1000)[1].boost, 255)
})

test("interpolateBoost is linear between points and flat outside them", () => {
  const curve = [{ temp: 40, boost: 0 }, { temp: 80, boost: 200 }]
  assert.equal(Model.interpolateBoost(curve, 20), 0)
  assert.equal(Model.interpolateBoost(curve, 40), 0)
  assert.equal(Model.interpolateBoost(curve, 60), 100)
  assert.equal(Model.interpolateBoost(curve, 80), 200)
  assert.equal(Model.interpolateBoost(curve, 105), 200)
  assert.equal(Model.interpolateBoost(curve, "hot"), 0)
})

test("curveSeries walks the full range and ends on the last temperature", () => {
  const series = Model.curveSeries([{ temp: 40, boost: 0 }, { temp: 80, boost: 200 }], 0, 110, 10)
  assert.equal(series[0].temp, 0)
  assert.equal(series[series.length - 1].temp, 110)
  assert.equal(series[series.length - 1].boost, 200)
})

test("curveOverlay draws the firmware floor under the added boost", () => {
  const overlay = Model.curveOverlay([{ temp: 40, boost: 0 }, { temp: 80, boost: 255 }], 30)
  const hot = overlay[overlay.length - 1]
  assert.equal(hot.floor, 30)
  assert.equal(hot.added, 100)
  assert.equal(hot.total, 100)
  const cold = overlay[0]
  assert.equal(cold.added, 0)
  assert.equal(cold.total, 30)
})

test("the curve note says the boost is additive", () => {
  assert.match(Model.CURVE_NOTE, /additive/i)
  assert.equal(Model.CURVE_NOTE.indexOf(EM_DASH), -1)
})

test("validateCurve accepts a normalized pair", () => {
  const v = Model.validateCurve({ cpu: [{ temp: 40, boost: 0 }, { temp: 90, boost: 255 }], gpu: [] })
  assert.equal(v.ok, true)
})

test("buildCurvePayload matches the contract shape", () => {
  const payload = Model.buildCurvePayload(99, 99, [{ temp: 40, boost: 0 }, { temp: 90, boost: 999 }], null)
  assert.equal(payload.interval, 30)
  assert.equal(payload.hysteresis, 15)
  assert.deepEqual(payload.cpu, [{ temp: 40, boost: 0 }, { temp: 90, boost: 255 }])
  assert.deepEqual(payload.gpu, Model.defaultCurve())
  assert.deepEqual(Object.keys(payload).sort(), ["cpu", "gpu", "hysteresis", "interval"])
  assert.deepEqual(JSON.parse(Model.curveJson(99, 99, payload.cpu, payload.gpu)), payload)
})

test("presets are ordered and each one is a valid curve", () => {
  const names = Model.presetNames()
  assert.equal(names.length, 4)
  for (const name of names) {
    const points = Model.curvePreset(name)
    assert.equal(points.length >= 2, true)
    assert.deepEqual(Model.normalizeCurve(points), points)
  }
  assert.deepEqual(Model.curvePreset("2"), Model.curvePreset("balanced"))
  assert.deepEqual(Model.curvePreset("nothing"), Model.defaultCurve())
})

test("validHex and normalizeHex reject bad colours", () => {
  assert.equal(Model.validHex("#ff0000"), true)
  assert.equal(Model.validHex("ff0000"), true)
  assert.equal(Model.validHex("#f00"), true)
  assert.equal(Model.validHex("#ff00"), false)
  assert.equal(Model.validHex("gg0000"), false)
  assert.equal(Model.validHex(""), false)
  assert.equal(Model.validHex(null), false)
  assert.equal(Model.validHex("#ff0000 "), true)
  assert.equal(Model.normalizeHex(" #f0a "), "FF00AA")
  assert.equal(Model.normalizeHex("abcdef"), "ABCDEF")
  assert.equal(Model.normalizeHex("nope"), "")
  assert.equal(Model.hexPreview("f00"), "#FF0000")
  assert.equal(Model.hexPreview("zz"), "")
})

test("colorToHex strips the alpha channel QML hands back", () => {
  assert.equal(Model.colorToHex("#ff3366cc"), "3366CC")
  assert.equal(Model.colorToHex("#3366cc"), "3366CC")
  assert.equal(Model.colorToHex("transparent"), "")
})

test("hexToRgb returns channel values or null", () => {
  assert.deepEqual(Model.hexToRgb("#00ff80"), { r: 0, g: 255, b: 128 })
  assert.equal(Model.hexToRgb("zzz"), null)
  assert.equal(Model.hexToRgb(""), null)
})

test("the four regions are fixed, named and ordered", () => {
  assert.deepEqual(Model.regionIds(), ["power", "logo", "ring-top", "ring-bottom"])
  assert.equal(Model.REGIONS.length, 4)
  assert.equal(Model.regionById("logo").name, "Lid logo")
  assert.equal(Model.regionById("ring-top").ledCount, 8)
  assert.equal(Model.regionById("nope"), null)
})

test("normalizeRgbStatus reads the four fixed regions from the helper", () => {
  const rgb = Model.normalizeRgbStatus({
    ok: true, backend: "alienfx",
    device: { path: "/dev/hidraw0", vendorId: "0x187c", productId: "0x0550", ready: true },
    regions: [
      { id: "power", name: "Power button", ledCount: 1 },
      { id: "logo", name: "Lid logo", ledCount: 1 },
      { id: "ring-top", name: "Ring top", ledCount: 8 },
      { id: "ring-bottom", name: "Ring bottom", ledCount: 8 }
    ]
  })
  assert.equal(rgb.backend, "alienfx")
  assert.equal(rgb.device.ready, true)
  assert.equal(rgb.device.vendorId, "0x187c")
  assert.equal(rgb.regions.length, 4)
  assert.deepEqual(rgb.regions.map((r) => r.id), ["power", "logo", "ring-top", "ring-bottom"])
})

test("normalizeRgbStatus survives an empty payload with the fixed region table", () => {
  const rgb = Model.normalizeRgbStatus(null)
  assert.equal(rgb.device.ready, false)
  assert.equal(rgb.regions.length, 4)
  assert.deepEqual(rgb.regions, Model.REGIONS)
})

test("normalizeRgbStatus ignores a region id the fixed table does not know about", () => {
  const rgb = Model.normalizeRgbStatus({
    ok: true,
    device: { ready: true },
    regions: [{ id: "power", name: "Power button", ledCount: 1 }, { id: "keyboard", name: "Keyboard", ledCount: 90 }]
  })
  assert.equal(rgb.regions.length, 4)
  assert.equal(rgb.regions.some((r) => r.id === "keyboard"), false)
})

test("regionColorPayload keeps only known region ids with a valid hex", () => {
  const payload = Model.regionColorPayload({
    power: "#ff0000",
    logo: "",
    "ring-top": "not a colour",
    keyboard: "00ff00"
  })
  assert.deepEqual(payload, { power: "FF0000" })
})

test("regionColorPayload survives junk input", () => {
  assert.deepEqual(Model.regionColorPayload(null), {})
  assert.deepEqual(Model.regionColorPayload("nope"), {})
})

test("hasAnyKey tells an empty map from a populated one", () => {
  assert.equal(Model.hasAnyKey({}), false)
  assert.equal(Model.hasAnyKey({ power: "FF0000" }), true)
  assert.equal(Model.hasAnyKey(null), false)
})

test("brightnessStep moves in fives and clamps", () => {
  assert.equal(Model.brightnessStep(50, 1), 55)
  assert.equal(Model.brightnessStep(50, -1), 45)
  assert.equal(Model.brightnessStep(98, 1), 100)
  assert.equal(Model.brightnessStep(2, -1), 0)
  assert.equal(Model.brightnessStep(50, 1, 10), 60)
})

test("statusAge and isStale use the poll interval", () => {
  const now = 1757260060000
  assert.equal(Model.statusAge(1757260000, now), 60)
  assert.equal(Number.isFinite(Model.statusAge(0, now)), false)
  assert.equal(Model.isStale(1757260000, now, 2), true)
  assert.equal(Model.isStale(1757260058, now, 2), false)
  assert.equal(Model.isStale(0, now, 2), true)
})

test("statusHealth keeps good state when a poll fails", () => {
  assert.equal(Model.statusHealth(true, { ok: true }, false), "ok")
  assert.equal(Model.statusHealth(true, { ok: true }, true), "stale")
  assert.equal(Model.statusHealth(true, { ok: false, code: "no-daemon" }, false), "error")
  assert.equal(Model.statusHealth(false, null, false), "missing")
  assert.equal(Model.statusHealth(false, { ok: false, code: "internal" }, false), "error")
  assert.equal(Model.statusHealth(true, { ok: false, code: "no-binary" }, false), "missing")
})

test("healthLine explains each health state", () => {
  assert.match(Model.healthLine("missing", null, null), /not installed/)
  assert.match(Model.healthLine("error", { error: "boom" }, null), /boom/)
  assert.match(Model.healthLine("stale", null, null), /stale/i)
  assert.equal(Model.healthLine("ok", null, status()), "dell_smm hwmon not present")
  assert.equal(Model.healthLine("ok", null, status({ warnings: [] })), "Ready")
})

test("barState hides everything when the binary is missing", () => {
  const s = Model.barState(status(), "missing", "full", 90)
  assert.equal(s.state, "missing")
  assert.equal(s.view, "icon")
  assert.equal(s.text, "")
  assert.equal(s.tone, "muted")
})

test("barState marks a helper error without pretending to have numbers", () => {
  const s = Model.barState(status(), "error", "temp", 90)
  assert.equal(s.state, "error")
  assert.equal(s.text, "")
  assert.equal(s.tone, "warn")
})

test("barState icon mode shows no text", () => {
  const s = Model.barState(status(), "ok", "icon", 90)
  assert.equal(s.state, "ok")
  assert.equal(s.view, "icon")
  assert.equal(s.text, "")
})

test("barState compact mode shows the cpu temperature", () => {
  const s = Model.barState(status(), "ok", "temp", 90)
  assert.equal(s.view, "compact")
  assert.equal(s.text, "47°")
})

test("barState expanded mode shows profile, rpm and gpu temperature", () => {
  const s = Model.barState(status(), "ok", "full", 90)
  assert.equal(s.view, "expanded")
  assert.equal(s.text, "Balanced · 2100 rpm · 43°")
})

test("barState goes to warning at or above the hot threshold", () => {
  const hot = status({ temps: { cpu: 90, gpu: 70 } })
  const s = Model.barState(hot, "ok", "icon", 90)
  assert.equal(s.state, "warning")
  assert.equal(s.tone, "warn")
  assert.equal(s.view, "compact")
  assert.equal(s.text, "90°")
  assert.equal(Model.barState(status({ temps: { cpu: 89, gpu: 70 } }), "ok", "icon", 90).state, "ok")
})

test("barState shows stopped fans as a muted idle state", () => {
  const idle = status({ fans: [{ id: "cpu", rpm: 0, max: 5700 }, { id: "gpu", rpm: 0, max: 5300 }], temps: { cpu: 41, gpu: 38 } })
  const s = Model.barState(idle, "ok", "full", 90)
  assert.equal(s.state, "stopped")
  assert.equal(s.tone, "muted")
  assert.equal(s.text, "Balanced · 0 rpm · 38°")
})

test("barState prefers a hot warning over stopped fans", () => {
  const bad = status({ fans: [{ id: "cpu", rpm: 0, max: 5700 }, { id: "gpu", rpm: 0, max: 5300 }], temps: { cpu: 95, gpu: 60 } })
  assert.equal(Model.barState(bad, "ok", "temp", 90).state, "warning")
})

test("barState dims stale readings but keeps the last numbers", () => {
  const s = Model.barState(status(), "stale", "temp", 90)
  assert.equal(s.state, "stale")
  assert.equal(s.tone, "muted")
  assert.equal(s.text, "47°")
})

test("barTooltip lists the fans and temperatures", () => {
  const t = Model.barTooltip(status(), "ok", null)
  assert.match(t, /Balanced/)
  assert.match(t, /CPU Fan 2100 rpm/)
  assert.match(t, /CPU 47°C/)
  assert.match(t, /GPU 43°C/)
})

test("barTooltip falls back to the health line when there is nothing to show", () => {
  assert.match(Model.barTooltip(null, "missing", null), /not installed/)
  assert.match(Model.barTooltip(status(), "error", { error: "no bus" }), /no bus/)
})

test("parseState reads saved region colours", () => {
  const state = Model.parseState(JSON.stringify({
    version: 2,
    regions: { power: "#f00", logo: "not a colour", keyboard: "00ff00" },
    color: "#f00",
    brightness: 250,
    themeSync: true
  }))
  assert.deepEqual(state.regions, { power: "FF0000" })
  assert.equal(state.color, "FF0000")
  assert.equal(state.brightness, 100)
  assert.equal(state.themeSync, true)
  assert.equal(state.lightsOn, true)
})

test("parseState drops an old zones array instead of crashing", () => {
  const state = Model.parseState(JSON.stringify({
    version: 1,
    zones: [{ index: 0, name: "Keyboard left", enabled: true }],
    color: "#0f0",
    brightness: 60,
    lightsOn: false
  }))
  assert.deepEqual(state.regions, {})
  assert.equal(state.color, "00FF00")
  assert.equal(state.brightness, 60)
  assert.equal(state.lightsOn, false)
})

test("parseState returns safe defaults for a missing or broken file", () => {
  const state = Model.parseState("{broken")
  assert.deepEqual(state.regions, {})
  assert.equal(state.color, "")
  assert.equal(state.brightness, 100)
  assert.equal(state.themeSync, false)
  assert.deepEqual(state.curve.cpu, Model.defaultCurve())
  assert.equal(state.curve.interval, 2)
})

test("buildStatePayload round trips through parseState at version two", () => {
  const payload = Model.buildStatePayload({
    regions: { power: "#f00", logo: "#00ff00" },
    color: "#00ff00",
    brightness: 80,
    lightsOn: false,
    themeSync: true,
    curve: { interval: 4, hysteresis: 5, cpu: Model.curvePreset("silent"), gpu: null }
  })
  assert.equal(payload.version, 2)
  const back = Model.parseState(JSON.stringify(payload))
  assert.deepEqual(back.regions, { power: "FF0000", logo: "00FF00" })
  assert.equal(back.color, "00FF00")
  assert.equal(back.brightness, 80)
  assert.equal(back.lightsOn, false)
  assert.equal(back.curve.interval, 4)
  assert.deepEqual(back.curve.cpu, Model.curvePreset("silent"))
})

test("power constraints get readable labels and a writability flag", () => {
  const s = status()
  assert.equal(s.power.available, true)
  assert.equal(s.power.constraints[0].label, "PL1 long term")
  assert.equal(s.power.constraints[1].label, "PL2 short term")
  assert.equal(s.power.constraints[2].label, "Peak power")
  assert.equal(s.power.anyWritable, false)
  assert.match(Model.PL_LOCKED_NOTE, /firmware/i)
})

test("gpu limits stay read only and unreadable draw becomes n/a", () => {
  const s = status({ gpu: { available: true, draw: null, limit: "[N/A]", defaultLimit: 90, maxLimit: 140, limitWritable: false } })
  assert.equal(s.gpu.limitWritable, false)
  assert.equal(Model.formatWatts(s.gpu.draw, 1), "n/a")
  assert.equal(Model.formatWatts(s.gpu.limit), "n/a")
  assert.equal(Model.formatWatts(s.gpu.defaultLimit), "90 W")
})

test("command builders match the contract verbs", () => {
  assert.deepEqual(Model.cmdStatus(), ["alienwarectl", "status"])
  assert.deepEqual(Model.cmdProfile("quiet"), ["alienwarectl", "profile", "quiet"])
  assert.deepEqual(Model.cmdBoost("gpu", 999), ["alienwarectl", "boost", "gpu", "255"])
  assert.deepEqual(Model.cmdCurveApply(), ["alienwarectl", "curve", "apply", "-"])
  assert.deepEqual(Model.cmdCurveStop(), ["alienwarectl", "curve", "stop"])
  assert.deepEqual(Model.cmdTurbo(true), ["alienwarectl", "turbo", "on"])
  assert.deepEqual(Model.cmdTurbo(false), ["alienwarectl", "turbo", "off"])
  assert.deepEqual(Model.cmdPl(2, 140), ["alienwarectl", "pl", "2", "140"])
  assert.deepEqual(Model.cmdGpu(), ["alienwarectl", "gpu"])
  assert.deepEqual(Model.cmdVersion(), ["alienwarectl", "version"])
})

test("rgb command builders normalize their arguments", () => {
  assert.deepEqual(Model.cmdRgbStatus(), ["alienwarectl", "rgb", "status"])
  assert.deepEqual(Model.cmdRgbSet("logo", "#f00"), ["alienwarectl", "rgb", "set", "logo", "FF0000"])
  assert.deepEqual(Model.cmdRgbSetAll("00ff00"), ["alienwarectl", "rgb", "set-all", "00FF00"])
  assert.deepEqual(Model.cmdRgbBrightness(500), ["alienwarectl", "rgb", "brightness", "100"])
  assert.deepEqual(Model.cmdRgbIdentify("ring-top"), ["alienwarectl", "rgb", "identify", "ring-top"])
  assert.deepEqual(Model.cmdRgbOff(), ["alienwarectl", "rgb", "off"])
})

test("cmdRgbSetMap orders regions by the fixed table and drops invalid colours", () => {
  const argv = Model.cmdRgbSetMap({ "ring-bottom": "#00f", power: "#f00", keyboard: "#0f0" })
  assert.deepEqual(argv, ["alienwarectl", "rgb", "set-map", "power=FF0000,ring-bottom=0000FF"])
})

test("cmdRgbSetMap survives an empty map", () => {
  assert.deepEqual(Model.cmdRgbSetMap({}), ["alienwarectl", "rgb", "set-map", ""])
})

test("queueKey keeps a colour write to one region from evicting a write to another", () => {
  const power = Model.queueKey(Model.cmdRgbSet("power", "FF0000"))
  const logo = Model.queueKey(Model.cmdRgbSet("logo", "00FF00"))
  assert.notEqual(power, logo)
})

test("queueKey still coalesces repeated writes to the same region", () => {
  const first = Model.queueKey(Model.cmdRgbSet("power", "FF0000"))
  const second = Model.queueKey(Model.cmdRgbSet("power", "0000FF"))
  assert.equal(first, second)
})

test("queueKey coalesces set-map spam regardless of which regions it touches", () => {
  const first = Model.queueKey(Model.cmdRgbSetMap({ power: "FF0000" }))
  const second = Model.queueKey(Model.cmdRgbSetMap({ logo: "00FF00", "ring-top": "0000FF" }))
  assert.equal(first, second)
})

test("queueKey still coalesces brightness and set-all spam regardless of value", () => {
  assert.equal(Model.queueKey(Model.cmdRgbBrightness(10)), Model.queueKey(Model.cmdRgbBrightness(90)))
  assert.equal(Model.queueKey(Model.cmdRgbSetAll("FF0000")), Model.queueKey(Model.cmdRgbSetAll("00FF00")))
})

test("no model string uses an em dash", () => {
  const strings = [Model.CURVE_NOTE, Model.PL_LOCKED_NOTE, Model.EFFECTS_NOTE]
  for (const key of Object.keys(Model.ERROR_MESSAGES)) strings.push(Model.ERROR_MESSAGES[key])
  for (const key of Object.keys(Model.PROFILE_LABELS)) strings.push(Model.PROFILE_LABELS[key])
  for (const s of strings) assert.equal(String(s).indexOf(EM_DASH), -1)
})

test("normalizeStatus carries the cpu object through", function() {
  var s = Model.normalizeStatus({ cpu: { available: true, mhz: 4700 } })
  assert.equal(s.cpu.available, true)
  assert.equal(s.cpu.mhz, 4700)
  assert.equal(Model.cpuClock(s), 4700)
})

test("cpuClock is NaN when the helper reports cpu unavailable", function() {
  var s = Model.normalizeStatus({ cpu: { available: false, mhz: 0 } })
  assert.ok(Number.isNaN(Model.cpuClock(s)))
  assert.equal(Model.formatClock(Model.cpuClock(s)), "not reported")
})

test("cpuClock is NaN when the helper omits cpu entirely", function() {
  var s = Model.normalizeStatus({})
  assert.ok(Number.isNaN(Model.cpuClock(s)))
})

test("a single point curve keeps the user's boost instead of jumping to maximum", function() {
  var c = Model.normalizeCurve([{ temp: 40, boost: 10 }])
  assert.equal(c.length, 2)
  assert.equal(c[0].boost, 10)
  assert.equal(c[1].boost, 10)
})

test("cmdPl never asks the helper for zero watts", function() {
  assert.deepEqual(Model.cmdPl(1, 0), ["alienwarectl", "pl", "1", "1"])
  assert.deepEqual(Model.cmdPl(2, -40), ["alienwarectl", "pl", "2", "1"])
})

test("every constraint the pl verb can address is writable", function() {
  assert.equal(Model.powerLimitWritable({ index: 0, writable: true }), true)
  assert.equal(Model.powerLimitWritable({ index: 1, writable: true }), true)
  assert.equal(Model.powerLimitWritable({ index: 2, writable: true }), true)
  assert.equal(Model.powerLimitWritable({ index: 3, writable: true }), false)
  assert.equal(Model.powerLimitWritable({ index: 0, writable: false }), false)
})

test("cmdPl addresses the peak power constraint", function() {
  assert.deepEqual(Model.cmdPl(3, 215), ["alienwarectl", "pl", "3", "215"])
})

test("a firmware locked constraint says so", function() {
  assert.equal(Model.powerLimitNote({ index: 0, writable: false }), Model.PL_LOCKED_NOTE)
  assert.equal(Model.powerLimitNote({ index: 2, writable: true }), "")
  assert.equal(Model.powerLimitNote({ index: 1, writable: true }), "")
})

test("isKnownPreset rejects anything that is not a shipped preset", function() {
  var names = Model.presetNames()
  assert.ok(names.length > 0)
  assert.equal(Model.isKnownPreset(names[0]), true)
  assert.equal(Model.isKnownPreset("appply"), false)
  assert.equal(Model.isKnownPreset(""), false)
  assert.equal(Model.isKnownPreset(null), false)
})

test("restore applies brightness before the region colours so the colours are not discarded", function() {
  var steps = Model.restoreSequence({
    lightsOn: true, brightness: 60, regions: { power: "FF0044", logo: "00FF00" }
  })
  var verbs = steps.map(function(s) { return s.argv.slice(1).join(" ") })
  assert.deepEqual(verbs, ["rgb brightness 60", "rgb set-map power=FF0044,logo=00FF00"])
})

test("restore skips the colour step when no region has a saved colour", function() {
  var steps = Model.restoreSequence({ lightsOn: true, brightness: 60, regions: {} })
  var verbs = steps.map(function(s) { return s.argv.slice(1).join(" ") })
  assert.deepEqual(verbs, ["rgb brightness 60"])
})

test("restore does nothing when the lights are off", function() {
  assert.deepEqual(Model.restoreSequence({
    lightsOn: false, brightness: 60, regions: { power: "FF0044" }
  }), [])
})

test("restore tolerates a missing state object", function() {
  assert.deepEqual(Model.restoreSequence(null), [])
  assert.deepEqual(Model.restoreSequence({}), [])
})
