import QtQuick
import Quickshell
import Quickshell.Io
import qs.Commons
import "Model.js" as Model

Item {
  id: root
  visible: false

  property var shell: null
  property var manifest: null

  readonly property string pluginId: "xyzlab.alienware"
  readonly property string home: Quickshell.env("HOME") || ""
  readonly property string stateDir: (Quickshell.env("XDG_STATE_HOME") || home + "/.local/state") + "/omarchy/alienware"
  readonly property string statePath: stateDir + "/state.json"

  readonly property var defaults: manifest && manifest.barWidget && manifest.barWidget.defaults ? manifest.barWidget.defaults : ({})
  readonly property var settings: resolveSettings(shell ? shell.shellConfig : null)

  function resolveSettings(config) {
    var merged = {}
    for (var d in defaults) merged[d] = defaults[d]
    var entry = findEntry(config)
    if (entry) for (var k in entry) if (k !== "id") merged[k] = entry[k]
    return merged
  }

  function findEntry(config) {
    if (!config || typeof config !== "object") return null
    var key = Util.canonicalWidgetId(pluginId)
    var bar = config.bar && typeof config.bar === "object" ? config.bar : null
    var layout = bar && bar.layout && typeof bar.layout === "object" ? bar.layout : null
    var sections = ["left", "center", "right"]
    if (layout) {
      for (var s = 0; s < sections.length; s++) {
        var arr = Model.toList(layout[sections[s]])
        for (var i = 0; i < arr.length; i++) {
          var e = Util.normalizeLayoutEntry(arr[i])
          if (e && Util.canonicalWidgetId(e.id) === key) return e
        }
      }
    }
    var plugins = Model.toList(config.plugins)
    for (var p = 0; p < plugins.length; p++) {
      var pe = plugins[p]
      if (pe && Util.canonicalWidgetId(pe.id) === key) return pe
    }
    return null
  }

  function setting(name, fallback) {
    var v = settings ? settings[name] : undefined
    return v === undefined || v === null ? fallback : v
  }

  readonly property int pollInterval: Model.clampInterval(setting("pollInterval", 2))
  readonly property int hotTemp: Model.clampHot(setting("hotTemp", 90))
  readonly property string display: Model.normalizeDisplay(setting("display", "temp"))
  readonly property bool themeRgb: Model.boolOr(setting("themeRgb", false), false)

  property var status: Model.emptyStatus()
  property bool hasStatus: false
  property var lastResult: null
  property double lastGoodAt: 0
  property double nowMs: Date.now()
  property bool pollBusy: false
  property double pollStartedAt: 0
  property int missedPolls: 0

  readonly property bool stale: !hasStatus ? true : Model.isStale(status.ts, nowMs, pollInterval) || missedPolls >= 3
  readonly property string health: Model.statusHealth(hasStatus, lastResult, stale)
  readonly property string statusLine: Model.healthLine(health, lastResult, hasStatus ? status : null)
  readonly property bool available: health === "ok" || health === "stale"

  readonly property string profileCurrent: status && status.profile ? status.profile.current : ""
  readonly property var profileChoices: Model.profileChoices(status)
  readonly property string profileWarning: hasStatus ? Model.profileWarning(status) : ""
  readonly property bool turboAvailable: status && status.turbo ? status.turbo.available : false
  readonly property bool turboEnabled: status && status.turbo ? status.turbo.enabled : false
  readonly property bool curveActive: status && status.curve ? status.curve.active : false

  property var rgb: Model.normalizeRgbStatus(null)
  property bool rgbLoaded: false
  property string rgbError: ""
  readonly property bool rgbConnected: rgb.device.ready && rgbError === ""
  readonly property var regions: rgb.regions

  property var regionColors: ({})
  property var regionOn: ({})

  property string color: ""
  property int brightness: 100
  property bool lightsOn: true
  property bool themeSync: false

  property int curveInterval: 2
  property int curveHysteresis: 3
  property var curveCpu: Model.defaultCurve()
  property var curveGpu: Model.defaultCurve()

  property bool stateLoaded: false
  property bool lightsRestored: false
  property int rgbRestoreTries: 0
  property bool dirReady: false
  property bool busy: false
  property string lastAction: ""
  property string actionStatus: ""
  property bool actionError: false

  readonly property string themeHex: Model.colorToHex(Color.accent)
  property string themePaletteText: ""
  readonly property var themeSwatches: Model.themeSwatches(themePaletteText)

  signal statusUpdated()
  signal actionFinished(bool ok, string text)

  Timer {
    interval: 5000
    running: true
    repeat: true
    onTriggered: root.nowMs = Date.now()
  }

  Process {
    id: mkdirProc
    command: ["mkdir", "-p", root.stateDir]
    onExited: root.dirReady = true
  }

  FileView {
    id: stateFile
    path: root.dirReady ? root.statePath : ""
    atomicWrites: true
    printErrors: false
    onLoaded: root.applyState(text())
    onLoadFailed: root.applyState("")
  }

  function applyState(raw) {
    var parsed = Model.parseState(raw)
    regionColors = parsed.regions
    regionOn = parsed.regionsOn
    color = parsed.color
    brightness = parsed.brightness
    lightsOn = parsed.lightsOn
    themeSync = parsed.themeSync === undefined ? themeRgb === true : parsed.themeSync === true
    curveInterval = parsed.curve.interval
    curveHysteresis = parsed.curve.hysteresis
    curveCpu = parsed.curve.cpu
    curveGpu = parsed.curve.gpu
    if (!stateLoaded) {
      stateLoaded = true
      poll()
      refreshRgb()
      refreshThemePalette()
    }
  }

  Timer {
    id: saveTimer
    interval: 400
    onTriggered: root.saveNow()
  }

  function scheduleSave() { saveTimer.restart() }

  function reportAction(ok, text) {
    actionError = ok !== true
    actionStatus = String(text || "")
    actionFinished(ok === true, actionStatus)
    actionReset.restart()
    return ok === true
  }

  function saveNow() {
    if (!dirReady) { saveTimer.restart(); return }
    stateFile.setText(JSON.stringify(Model.buildStatePayload({
      savedAt: Date.now(),
      regions: regionColors,
      regionsOn: regionOn,
      color: color,
      brightness: brightness,
      lightsOn: lightsOn,
      themeSync: themeSync,
      curve: { interval: curveInterval, hysteresis: curveHysteresis, cpu: curveCpu, gpu: curveGpu }
    })))
  }

  function shellArgv(argv) {
    return ["bash", "-c", '"$@"', "alienwarectl-run"].concat(Model.toList(argv))
  }

  Timer {
    id: pollTimer
    interval: root.pollInterval * 1000
    running: root.stateLoaded
    repeat: true
    triggeredOnStart: false
    onTriggered: root.poll()
  }

  function poll() {
    if (!stateLoaded) return
    if (statusProc.running) {
      missedPolls++
      var waited = Date.now() - pollStartedAt
      if (waited > Math.max(10000, pollInterval * 4000)) statusProc.running = false
      return
    }
    missedPolls = 0
    pollStartedAt = Date.now()
    pollBusy = true
    statusProc.command = shellArgv(Model.cmdStatus())
    statusProc.running = true
  }

  Process {
    id: statusProc
    stdout: StdioCollector { id: statusOut; waitForEnd: true }
    stderr: StdioCollector { id: statusErr; waitForEnd: true }
    onExited: function(exitCode) {
      root.pollBusy = false
      var result = Model.parseResult(statusOut.text, exitCode)
      if (!result.ok && !result.error) result.error = String(statusErr.text || "").trim()
      root.lastResult = result
      if (result.ok) {
        root.status = Model.normalizeStatus(result.data)
        root.hasStatus = true
        root.lastGoodAt = Date.now()
        root.nowMs = root.lastGoodAt
        root.adoptCurveFromStatus()
      }
      root.statusUpdated()
    }
  }

  property bool curveAdopted: false

  function adoptCurveFromStatus() {
    if (curveAdopted) return
    if (!status.curve.active) { curveAdopted = true; return }
    curveInterval = status.curve.interval
    curveHysteresis = status.curve.hysteresis
    curveCpu = status.curve.cpu
    curveGpu = status.curve.gpu
    curveAdopted = true
    scheduleSave()
  }

  function refreshRgb() {
    if (rgbProc.running) return
    rgbProc.command = shellArgv(Model.cmdRgbStatus())
    rgbProc.running = true
  }

  Process {
    id: rgbProc
    stdout: StdioCollector { id: rgbOut; waitForEnd: true }
    onExited: function(exitCode) {
      var result = Model.parseResult(rgbOut.text, exitCode)
      if (result.ok) {
        root.rgb = Model.normalizeRgbStatus(result.data)
        root.rgbError = ""
        root.rgbLoaded = true
        if (!root.lightsRestored && root.stateLoaded) {
          root.lightsRestored = true
          if (root.rgb.device.ready) root.restoreSavedLights()
        }
      } else {
        root.rgbError = result.error
        root.rgbLoaded = true
      }
    }
  }

  function refreshThemePalette() {
    if (themePaletteProc.running) return
    themePaletteProc.running = true
  }

  Process {
    id: themePaletteProc
    command: ["omarchy", "theme", "color", "--all"]
    stdout: StdioCollector { id: themePaletteOut; waitForEnd: true }
    onExited: function(exitCode) {
      if (exitCode === 0) root.themePaletteText = themePaletteOut.text
    }
  }

  Timer {
    id: rgbRestoreRetry
    interval: root.rgbRestoreTries < 5 ? 3000 : 30000
    repeat: true
    running: root.stateLoaded && !root.lightsRestored && root.rgbRestoreTries < 30
    onTriggered: {
      root.rgbRestoreTries++
      root.refreshRgb()
    }
  }

  property var queue: []

  function enqueue(argv, label, stdinText) {
    var job = { argv: Model.toList(argv), label: String(label || ""), stdin: String(stdinText || ""), key: Model.queueKey(argv) }
    var next = []
    for (var i = 0; i < queue.length; i++) {
      if (queue[i] && queue[i].key === job.key) continue
      next.push(queue[i])
    }
    next.push(job)
    queue = next
    pump()
  }

  function pump() {
    if (writeProc.running) return
    if (!queue.length) {
      busy = false
      return
    }
    var job = queue[0]
    queue = queue.slice(1)
    busy = true
    lastAction = job.label
    writeProc.payload = job.stdin
    if (job.stdin) {
      writeProc.command = ["bash", "-c", 'printf %s "$ALIENWARE_STDIN" | "$@"', "alienwarectl-run"].concat(job.argv)
    } else {
      writeProc.command = shellArgv(job.argv)
    }
    writeProc.running = true
    writeWatchdog.restart()
  }

  Timer {
    id: writeWatchdog
    interval: 20000
    onTriggered: {
      if (!writeProc.running) return
      writeProc.running = false
      root.actionError = true
      root.actionStatus = (root.lastAction ? root.lastAction : "Command") + " timed out"
      root.actionFinished(false, root.actionStatus)
    }
  }

  Process {
    id: writeProc
    property string payload: ""
    environment: ({ ALIENWARE_STDIN: payload })
    stdout: StdioCollector { id: writeOut; waitForEnd: true }
    stderr: StdioCollector { id: writeErr; waitForEnd: true }
    onExited: function(exitCode) {
      var result = Model.parseResult(writeOut.text, exitCode)
      if (!result.ok && !result.error) result.error = String(writeErr.text || "").trim()
      root.actionError = !result.ok
      root.actionStatus = result.ok ? (root.lastAction ? root.lastAction + " applied" : "Applied") : result.error
      root.actionFinished(result.ok, root.actionStatus)
      actionReset.restart()
      refreshSoon.restart()
      root.pump()
    }
  }

  Timer {
    id: actionReset
    interval: 6000
    onTriggered: if (!root.busy) root.actionStatus = ""
  }

  Timer {
    id: refreshSoon
    interval: 400
    onTriggered: root.poll()
  }

  function setProfile(name) {
    var wanted = String(name || "")
    if (!wanted) return false
    enqueue(Model.cmdProfile(wanted), "Profile " + Model.profileLabel(wanted, status.profile.gmodeForced))
    return true
  }

  function cycleProfile(direction) {
    if (!available) return false
    return setProfile(Model.nextProfile(profileChoices, profileCurrent, direction))
  }

  function setBoost(fan, value) {
    enqueue(Model.cmdBoost(fan, value), "Boost")
    return true
  }

  function nudgeBoost(fan, delta) {
    var current = Model.fanById(status, fan)
    return setBoost(fan, (current ? current.boost : 0) + Model.toInt(delta, 0))
  }

  function applyCurve() {
    var payload = Model.buildCurvePayload(curveInterval, curveHysteresis, curveCpu, curveGpu)
    var check = Model.validateCurve(payload)
    if (!check.ok) {
      actionError = true
      actionStatus = check.error
      actionFinished(false, check.error)
      return false
    }
    enqueue(Model.cmdCurveApply(), "Curve", JSON.stringify(payload))
    scheduleSave()
    return true
  }

  function stopCurve() {
    enqueue(Model.cmdCurveStop(), "Curve stop")
    return true
  }

  function resetCurve() {
    curveCpu = Model.defaultCurve()
    curveGpu = Model.defaultCurve()
    curveInterval = 2
    curveHysteresis = 3
    scheduleSave()
  }

  function applyPreset(name) {
    curveCpu = Model.curvePreset(name)
    curveGpu = Model.curvePreset(name)
    scheduleSave()
  }

  function setCurvePoints(fan, points) {
    if (String(fan) === "gpu") curveGpu = Model.normalizeCurve(points)
    else curveCpu = Model.normalizeCurve(points)
    scheduleSave()
  }

  function curveFor(fan) {
    return String(fan) === "gpu" ? curveGpu : curveCpu
  }

  function setCurveInterval(value) {
    curveInterval = Model.clampInterval(value)
    scheduleSave()
  }

  function setCurveHysteresis(value) {
    curveHysteresis = Model.clampHysteresis(value)
    scheduleSave()
  }

  function setTurbo(on) {
    if (!turboAvailable) {
      actionError = true
      actionStatus = "Turbo control is not available"
      return false
    }
    enqueue(Model.cmdTurbo(on === true), on ? "Turbo on" : "Turbo off")
    return true
  }

  function toggleTurbo() { return setTurbo(!turboEnabled) }

  function setPowerLimit(index, watts) {
    enqueue(Model.cmdPl(index, watts), "Power limit")
    return true
  }

  function constraintWritable(index) {
    var list = Model.toList(status.power.constraints)
    for (var i = 0; i < list.length; i++) if (list[i] && list[i].index === Model.toInt(index, -1)) return list[i].writable
    return false
  }

  function allRegionsColorMap(hex) {
    var map = {}
    var ids = Model.regionIds()
    for (var i = 0; i < ids.length; i++) map[ids[i]] = hex
    return map
  }

  function setColor(hex) {
    var clean = Model.normalizeHex(hex)
    if (!clean) {
      actionError = true
      actionStatus = "That is not a hex colour"
      return false
    }
    regionColors = allRegionsColorMap(clean)
    color = clean
    lightsOn = true
    enqueue(Model.cmdRgbSetMap(Model.effectiveRegionColors(regionColors, regionOn)), "Colour")
    scheduleSave()
    return true
  }

  function setRegionColors(ids, hex) {
    var clean = Model.normalizeHex(hex)
    if (!clean) {
      actionError = true
      actionStatus = "That is not a hex colour"
      return false
    }
    var list = Model.toList(ids)
    if (!list.length) {
      actionError = true
      actionStatus = "Select at least one region first"
      return false
    }
    var nextColors = {}
    for (var k in regionColors) nextColors[k] = regionColors[k]
    var nextOn = {}
    for (var k2 in regionOn) nextOn[k2] = regionOn[k2]
    var map = {}
    for (var i = 0; i < list.length; i++) {
      nextColors[list[i]] = clean
      nextOn[list[i]] = true
      map[list[i]] = clean
    }
    regionColors = nextColors
    regionOn = nextOn
    color = clean
    lightsOn = true
    enqueue(Model.cmdRgbSetMap(map), "Colour")
    scheduleSave()
    return true
  }

  function setRegionOn(ids, on) {
    var list = Model.toList(ids)
    if (!list.length) {
      actionError = true
      actionStatus = "Select at least one region first"
      return false
    }
    var nextOn = {}
    for (var k in regionOn) nextOn[k] = regionOn[k]
    var map = {}
    for (var i = 0; i < list.length; i++) {
      var id = list[i]
      nextOn[id] = on === true
      map[id] = on === true ? (Model.normalizeHex(regionColors[id]) || Model.normalizeHex(color) || "FFFFFF") : "000000"
    }
    regionOn = nextOn
    if (on === true) lightsOn = true
    enqueue(Model.cmdRgbSetMap(map), on === true ? "Region on" : "Region off")
    scheduleSave()
    return true
  }

  function toggleRegionOn(ids) {
    var list = Model.toList(ids)
    if (!list.length) return false
    var allOn = true
    for (var i = 0; i < list.length; i++) {
      if (regionOn[list[i]] === false) { allOn = false; break }
    }
    return setRegionOn(list, !allOn)
  }

  function setBrightness(value) {
    brightness = Model.clampBrightness(value)
    if (brightness > 0) lightsOn = true
    enqueue(Model.cmdRgbBrightness(brightness), "Brightness")
    scheduleSave()
    return true
  }

  function nudgeBrightness(delta) {
    return setBrightness(Model.brightnessStep(brightness, delta))
  }

  function lightsOff() {
    lightsOn = false
    enqueue(Model.cmdRgbOff(), "Lights off")
    scheduleSave()
    return true
  }

  function restoreLights() {
    lightsOn = true
    enqueue(Model.cmdRgbBrightness(brightness), "Brightness")
    var map = Model.effectiveRegionColors(regionColors, regionOn)
    if (Model.hasAnyKey(map)) enqueue(Model.cmdRgbSetMap(map), "Colour")
    scheduleSave()
    return true
  }

  function toggleLights() {
    return lightsOn ? lightsOff() : restoreLights()
  }

  function restoreSavedLights() {
    var steps = Model.restoreSequence({
      lightsOn: lightsOn,
      regions: regionColors,
      regionsOn: regionOn,
      brightness: brightness
    })
    if (!steps.length) return false
    for (var i = 0; i < steps.length; i++) enqueue(steps[i].argv, steps[i].label)
    return true
  }

  function identifyRegion(id) {
    enqueue(Model.cmdRgbIdentify(id), "Identify")
    return true
  }

  function setThemeSync(on) {
    themeSync = on === true
    scheduleSave()
    if (themeSync) applyThemeColor()
    return true
  }

  function toggleThemeSync() { return setThemeSync(!themeSync) }

  function applyThemeColor() {
    var hex = themeHex
    if (!hex) return false
    regionColors = allRegionsColorMap(hex)
    color = hex
    lightsOn = true
    enqueue(Model.cmdRgbSetMap(Model.effectiveRegionColors(regionColors, regionOn)), "Theme colour")
    scheduleSave()
    return true
  }

  onThemeHexChanged: {
    if (stateLoaded) refreshThemePalette()
    if (stateLoaded && themeSync) themeApply.restart()
  }

  Timer {
    id: themeApply
    interval: 600
    onTriggered: root.applyThemeColor()
  }

  function reload() {
    lastResult = null
    poll()
    refreshRgb()
    refreshThemePalette()
  }

  function summon() {
    Quickshell.execDetached(["omarchy-shell", "shell", "summon", pluginId, "{}"])
  }

  IpcHandler {
    target: "alienware"

    function status(): string {
      return JSON.stringify({
        health: root.health,
        statusLine: root.statusLine,
        profile: root.profileCurrent,
        fans: root.status.fans,
        temps: root.status.temps,
        turbo: root.status.turbo,
        curveActive: root.curveActive,
        rgb: {
          connected: root.rgbConnected,
          lightsOn: root.lightsOn,
          brightness: root.brightness,
          color: root.color,
          regions: Model.regionColorPayload(root.regionColors)
        },
        warnings: root.status.warnings,
        error: root.lastResult && root.lastResult.ok === false ? root.lastResult.error : ""
      })
    }

    function profile(name: string): string {
      if (!name) return root.profileCurrent
      return root.setProfile(name) ? "ok" : root.actionStatus
    }

    function boost(fan: string, value: string): string {
      return root.setBoost(fan, parseInt(value, 10)) ? "ok" : root.actionStatus
    }

    function curve(action: string): string {
      var a = String(action || "").toLowerCase()
      if (a === "stop") return root.stopCurve() ? "ok" : root.actionStatus
      if (a === "reset") { root.resetCurve(); return "ok" }
      if (a === "apply" || a === "") return root.applyCurve() ? "ok" : root.actionStatus
      if (!Model.isKnownPreset(a)) return "unknown curve action: " + a
      root.applyPreset(a)
      return root.applyCurve() ? "ok" : root.actionStatus
    }

    function turbo(state: string): string {
      var s = String(state || "").toLowerCase()
      if (s === "on") return root.setTurbo(true) ? "ok" : root.actionStatus
      if (s === "off") return root.setTurbo(false) ? "ok" : root.actionStatus
      return root.toggleTurbo() ? "ok" : root.actionStatus
    }

    function rgb(action: string, value: string): string {
      var a = String(action || "").toLowerCase()
      if (a === "off") return root.lightsOff() ? "ok" : root.actionStatus
      if (a === "on") return root.restoreLights() ? "ok" : root.actionStatus
      if (a === "toggle") return root.toggleLights() ? "ok" : root.actionStatus
      if (a === "color" || a === "colour") return root.setColor(value) ? "ok" : root.actionStatus
      if (a === "brightness") return root.setBrightness(parseInt(value, 10)) ? "ok" : root.actionStatus
      if (a === "theme") return root.applyThemeColor() ? "ok" : root.actionStatus
      if (a === "identify") return root.identifyRegion(value) ? "ok" : root.actionStatus
      if (a === "reload" || a === "") { root.refreshRgb(); return "ok" }
      return "unknown rgb action"
    }

    function open(): void { root.summon() }
    function reload(): void { root.reload() }
  }

  Component.onCompleted: mkdirProc.running = true

  Component.onDestruction: {
    pollTimer.stop()
    refreshSoon.stop()
    themeApply.stop()
    saveTimer.stop()
    actionReset.stop()
    writeWatchdog.stop()
    queue = []
    statusProc.running = false
    rgbProc.running = false
    writeProc.running = false
    themePaletteProc.running = false
  }
}
