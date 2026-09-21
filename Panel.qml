import QtQuick
import Quickshell
import Quickshell.Io
import qs.Commons
import qs.Ui
import "Model.js" as Model
import "components/RgbSubTab.js" as RgbSubTab
import "components"

Panel {
  id: root
  moduleName: "xyzlab.alienware"
  ipcTarget: ""
  manageIpc: false

  property var anchorItem: null
  property var hostWidget: null
  property var service: null
  property bool openedFromHotkey: false
  readonly property var barIdentity: hostWidget || root

  function open() {
    openedFromHotkey = false
    root.controller.show()
    setCenterHoverRevealSuppressed(false)
    onOpened()
  }

  function openFromHotkey() {
    openedFromHotkey = true
    root.controller.show()
    onOpened()
    Qt.callLater(function() {
      if (root.opened) setCenterHoverRevealSuppressed(true)
    })
  }

  function onOpened() {
    if (!Model.validHex(pickedHex) && service) pickedHex = Model.normalizeHex(service.color) || service.themeHex || "FF0000"
    statusIndex = 0
    settingsOpen = false
    settingsStatus = ""
    if (service) {
      service.reload()
    }
  }

  function close() {
    root.controller.hide()
    settingsOpen = false
    setCenterHoverRevealSuppressed(false)
  }

  function toggle() {
    if (root.opened) root.close()
    else root.openFromHotkey()
  }

  function switchPanel(direction) {
    if (root.bar && typeof root.bar.switchPanelFrom === "function")
      return root.bar.switchPanelFrom(root.barIdentity, direction)
    return false
  }

  function setCenterHoverRevealSuppressed(value) {
    if (!root.bar) return
    try {
      if (typeof root.bar.setCenterHoverRevealSuppressed === "function")
        root.bar.setCenterHoverRevealSuppressed(value)
      else if ("centerHoverRevealSuppressed" in root.bar)
        root.bar.centerHoverRevealSuppressed = value
    } catch (e) {
    }
  }

  readonly property color fg: bar ? bar.foreground : Color.foreground
  readonly property color dim: Qt.darker(fg, 1.5)
  readonly property color urgentColor: bar ? bar.urgent : Color.urgent
  readonly property color accent: Color.accent
  readonly property string fontFamily: bar ? bar.fontFamily : Style.font.family

  readonly property var tabs: [
    { key: "fans", label: "Fans", icon: "󰈐" },
    { key: "rgb", label: "Lighting", icon: "󰌌" },
    { key: "power", label: "Power", icon: "󱐋" }
  ]
  property string tab: "fans"

  readonly property var hw: service ? service.status : Model.emptyStatus()
  readonly property string health: service ? service.health : "missing"
  readonly property bool available: service ? service.available : false
  readonly property bool stale: health === "stale"
  readonly property int hotTemp: service ? service.hotTemp : 90
  readonly property var cpuFan: Model.fanById(hw, "cpu")
  readonly property var gpuFan: Model.fanById(hw, "gpu")
  readonly property var cpuTemp: typeof hw.temps.cpu === "number" ? hw.temps.cpu : null
  readonly property var gpuTemp: typeof hw.temps.gpu === "number" ? hw.temps.gpu : null

  property string curveFan: "cpu"
  property int pointCursor: 0
  readonly property var curvePoints: service ? service.curveFor(curveFan) : Model.defaultCurve()
  readonly property int curveFloor: {
    var fan = curveFan === "gpu" ? gpuFan : cpuFan
    return fan ? fan.percent : 0
  }
  readonly property var curveTemp: curveFan === "gpu" ? gpuTemp : cpuTemp

  property int regionCursor: 0
  property var regionSelection: []
  readonly property var rgb: service ? service.rgb : Model.normalizeRgbStatus(null)
  readonly property var regions: service ? service.regions : Model.normalizeRegions(null)
  readonly property bool rgbConnected: service ? service.rgbConnected : false
  readonly property string rgbError: service ? service.rgbError : ""

  readonly property var regionSwatchMap: {
    var ids = Model.regionIds()
    var out = {}
    var colors = root.service ? root.service.regionColors : {}
    for (var i = 0; i < ids.length; i++) {
      var id = ids[i]
      out[id] = colors[id] ? Model.hexPreview(colors[id]) : ""
    }
    return out
  }
  readonly property var regionPoweredMap: Model.normalizeRegionOnMap(root.service ? root.service.regionOn : null)

  readonly property var rgbSubTabs: [
    { key: "keyboard", label: "KEYBOARD" },
    { key: "chassis", label: "CHASSIS" }
  ]
  property string rgbSubTab: RgbSubTab.DEFAULT_SUB_TAB
  property bool rgbSubTabLoaded: false
  readonly property bool rgbSubTabReady: root.service && root.service.dirReady
  property bool pickerOpen: false

  readonly property var batteryState: root.service ? root.service.battery : ({ ok: false })
  readonly property bool batteryOnPower: root.service ? root.service.batterySync : true
  readonly property string powerButtonHex: {
    if (root.batteryOnPower) return Model.batteryColor(root.batteryState)
    if (root.service && root.service.regionOn["power"] === false) return ""
    return root.service ? Model.normalizeHex(root.service.regionColors["power"]) : ""
  }
  readonly property bool powerButtonLit: root.powerButtonHex !== ""
  readonly property color powerButtonColor: root.powerButtonLit ? Model.hexPreview(root.powerButtonHex) : root.dim
  readonly property string batteryLine: {
    var b = root.batteryState
    if (!b || b.ok !== true || b.percent < 0) return "battery state unavailable"
    var pct = Math.round(b.percent) + "%"
    if (b.percent < 10) return pct + " critical, red"
    return pct + (b.charging === true ? " charging" : " on battery")
  }

  readonly property string rgbSubTabEffective: RgbSubTab.normalizeRgbSubTab(root.rgbSubTab, root.kbdPresent)

  function applyRgbSubTabText(raw) {
    root.rgbSubTab = RgbSubTab.normalizeRgbSubTab(raw, true)
    root.rgbSubTabLoaded = true
  }

  function setRgbSubTab(key) {
    var next = RgbSubTab.normalizeRgbSubTab(key, root.kbdPresent)
    if (next === root.rgbSubTabEffective) return
    root.rgbSubTab = next
    if (next === "keyboard") root.clearRegionSelection()
    else root.clearKeySelection()
  }

  function moveRgbSubTab(direction) {
    var keys = RgbSubTab.SUB_TABS
    var idx = keys.indexOf(root.rgbSubTabEffective)
    if (idx === -1) idx = 0
    var next = (idx + direction + keys.length) % keys.length
    root.setRgbSubTab(keys[next])
  }

  onRgbSubTabChanged: {
    if (!root.rgbSubTabLoaded) return
    rgbSubTabSaveTimer.restart()
  }

  FileView {
    id: rgbSubTabFile
    path: root.rgbSubTabReady ? root.service.stateDir + "/rgb-subtab" : ""
    atomicWrites: true
    printErrors: false
    onLoaded: root.applyRgbSubTabText(text())
    onLoadFailed: root.applyRgbSubTabText("")
  }

  Timer {
    id: rgbSubTabSaveTimer
    interval: 250
    onTriggered: if (rgbSubTabFile.path) rgbSubTabFile.setText(root.rgbSubTab)
  }

  property var keySelection: []
  readonly property var kbd: service ? service.kbd : Model.normalizeKbdStatus(null)
  readonly property bool kbdPresent: service ? service.kbdPresent : false

  property int powerCursor: 0
  readonly property var constraints: Model.toList(hw.power.constraints)
  readonly property int powerFieldCount: constraints.length + 1

  property bool settingsOpen: false
  property string settingsStatus: ""
  property bool settingsError: false
  property string dPoll: "2"
  property string dHot: "90"
  property string dDisplay: "cpu"

  readonly property string heroGlyph: "󰢚"

  readonly property var statusParts: {
    var parts = []
    if (!service) return ["Loading"]
    parts.push(service.statusLine)
    if (available) {
      if (hw.profile.current) parts.push(Model.profileLabel(hw.profile.current, hw.profile.gmodeForced))
      if (cpuTemp !== null) parts.push("CPU " + Model.formatTemp(cpuTemp, true))
      if (hw.curve.active) parts.push("Curve running")
    }
    if (service.actionStatus) parts.push(service.actionStatus)
    return parts
  }
  property int statusIndex: 0
  onStatusPartsChanged: if (statusIndex >= statusParts.length) statusIndex = 0
  readonly property string statusCaps: statusParts.length ? String(statusParts[Math.min(statusIndex, statusParts.length - 1)]).toUpperCase() : ""

  Timer {
    interval: 3000
    running: root.opened && root.statusParts.length > 1
    repeat: true
    onTriggered: root.statusIndex = (root.statusIndex + 1) % root.statusParts.length
  }

  function moveTab(dx) {
    var idx = 0
    for (var i = 0; i < tabs.length; i++) if (tabs[i].key === tab) idx = i
    idx = (idx + dx + tabs.length) % tabs.length
    setTab(tabs[idx].key)
  }

  function setTab(key) {
    tab = key
  }

  function moveCursor(dy) {
    if (tab === "fans") {
      var points = Model.toList(curvePoints)
      if (!points.length) return
      pointCursor = Math.max(0, Math.min(points.length - 1, pointCursor + dy))
    } else if (tab === "rgb") {
      if (rgbSubTabEffective !== "chassis") return
      var list = Model.toList(regions)
      if (!list.length) return
      regionCursor = Math.max(0, Math.min(list.length - 1, regionCursor + dy))
    } else {
      powerCursor = Math.max(0, Math.min(Math.max(0, powerFieldCount - 1), powerCursor + dy))
    }
  }

  function nudgePoint(dTemp, dBoost) {
    if (!service) return
    service.setCurvePoints(curveFan, Model.movePoint(curvePoints, pointCursor, dTemp, dBoost))
  }

  function setCurvePoints(next) {
    if (!service) return
    service.setCurvePoints(curveFan, next)
  }

  function adjustPowerField(direction) {
    if (!service) return
    if (powerCursor >= constraints.length) {
      service.toggleTurbo()
      return
    }
    var c = constraints[powerCursor]
    if (!Model.powerLimitWritable(c)) return
    service.setPowerLimit(c.index + 1, c.watts + direction * 5)
  }

  function applyPreset(name) {
    if (!service) return
    service.applyPreset(name)
    pointCursor = 0
  }

  function toggleRegionSelection(id) {
    var list = regionSelection.slice()
    var idx = list.indexOf(id)
    if (idx === -1) list.push(id)
    else list.splice(idx, 1)
    regionSelection = list
  }

  function selectAllRegions() {
    regionSelection = Model.regionIds()
  }

  function clearRegionSelection() {
    regionSelection = []
  }

  function isRegionOn(id) {
    return service ? service.regionOn[id] !== false : true
  }

  function setRegionPower(ids, on) {
    if (!service) return
    service.setRegionOn(ids, on)
  }

  function activeRegionIds() {
    if (regionSelection.length) return regionSelection.slice()
    var list = Model.toList(regions)
    var current = list[regionCursor]
    return current ? [current.id] : []
  }

  function toggleRegionPower() {
    if (!service) return
    var ids = activeRegionIds()
    if (!ids.length) return
    service.toggleRegionOn(ids)
  }

  function toggleKeySelection(id) {
    var list = keySelection.slice()
    var idx = list.indexOf(id)
    if (idx === -1) list.push(id)
    else list.splice(idx, 1)
    keySelection = list
  }

  function addKeysToSelection(ids) {
    var list = keySelection.slice()
    var incoming = Model.toList(ids)
    for (var i = 0; i < incoming.length; i++) {
      if (list.indexOf(incoming[i]) === -1) list.push(incoming[i])
    }
    keySelection = list
  }

  function selectAllKeys() {
    keySelection = Model.keyboardPaintableIds()
  }

  function clearKeySelection() {
    keySelection = []
  }

  function toggleKeySelectionPower() {
    if (!service) return
    var ids = keySelection.slice()
    if (!ids.length) return
    service.toggleKeyOn(ids)
  }

  property string pickedHex: ""
  readonly property var paintRegions: {
    if (rgbSubTabEffective === "keyboard") return regionSelection.indexOf("power") !== -1 ? ["power"] : []
    return regionSelection
  }
  readonly property var paintKeys: rgbSubTabEffective === "keyboard" ? keySelection : []
  readonly property string paintTarget: Model.selectionSummary(paintKeys.length, paintRegions)

  function applyColorField() {
    if (!service) return
    if (!Model.validHex(pickedHex)) {
      service.reportAction(false, "Enter a colour as RRGGBB")
      return
    }
    if (!paintKeys.length && !paintRegions.length) {
      service.reportAction(false, rgbSubTabEffective === "keyboard" ? "Select some keys first" : "Select a region first")
      return
    }
    if (paintKeys.length) service.setKeyColors(paintKeys, pickedHex)
    if (paintRegions.length) service.setRegionColors(paintRegions, pickedHex)
  }

  function pickPowerKey() {
    if (!service) return
    if (service.batterySync) {
      service.reportAction(false, "The power button shows battery status. Turn off Battery on power button to colour it.")
      return
    }
    toggleRegionSelection("power")
  }

  function refocusPanel() {
    Qt.callLater(function() { if (keyCatcher) keyCatcher.forceActiveFocus() })
  }

  function currentSetting(name, fallback) {
    var v = settings ? settings[name] : undefined
    if (v === undefined || v === null) {
      var d = service && service.defaults ? service.defaults[name] : undefined
      return d === undefined || d === null ? fallback : d
    }
    return v
  }

  function loadDraft() {
    dPoll = String(Model.clampInterval(currentSetting("pollInterval", 2)))
    dHot = String(Model.clampHot(currentSetting("hotTemp", 90)))
    dDisplay = Model.serializeBarItems(currentSetting("display", "cpu"))
    sPoll.text = dPoll
    sHot.text = dHot
  }

  function openSettings() {
    settingsStatus = ""
    settingsError = false
    loadDraft()
    settingsOpen = true
  }

  function closeSettings() {
    settingsOpen = false
    settingsStatus = ""
    refocusPanel()
  }

  function toggleSettings() {
    if (settingsOpen) closeSettings()
    else openSettings()
  }

  function failSettings(text) {
    settingsError = true
    settingsStatus = text
    return false
  }

  function saveSettings() {
    var poll = parseInt(dPoll, 10)
    if (!isFinite(poll) || poll < 1 || poll > 30) return failSettings("Poll interval must be 1 to 30 seconds")
    var hot = parseInt(dHot, 10)
    if (!isFinite(hot) || hot < 50 || hot > 110) return failSettings("Hot temperature must be 50 to 110 C")
    if (!root.bar || !root.bar.shell || typeof root.bar.shell.updateEntryInline !== "function")
      return failSettings("Shell refused the update")
    var entry = {}
    for (var k in settings) if (k !== "id") entry[k] = settings[k]
    entry.pollInterval = poll
    entry.hotTemp = hot
    entry.display = Model.serializeBarItems(dDisplay)
    root.bar.shell.updateEntryInline(root.moduleName, entry)
    settingsError = false
    settingsStatus = "Saved"
    settingsOpen = false
    refocusPanel()
    return true
  }

  function handleSettingsKey(event) {
    if (event.key === Qt.Key_Escape) {
      closeSettings()
      event.accepted = true
    } else if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
      saveSettings()
      event.accepted = true
    }
  }

  readonly property string footerText: {
    if (settingsOpen) return "⏎ save · esc cancel"
    if (tab === "fans") return "h/l tab · p mode · j/k point · +/- boost · 1-4 preset · f fan · a apply · x stop · esc"
    if (tab === "rgb") {
      var rgbParts = ["h/l tab"]
      if (kbdPresent) rgbParts.push("H/L surface")
      if (rgbSubTabEffective === "chassis") { rgbParts.push("j/k region", "space select", "a/x regions", "i identify") }
      else { rgbParts.push("click/drag keys", "K/X keys") }
      rgbParts.push("p on/off", "c colour", "t theme", "b battery", "esc")
      return rgbParts.join(" · ")
    }
    return "h/l tab · p mode · 1-4 mode · j/k field · +/- adjust · t turbo · esc"
  }

  readonly property string editorFocusBlock: colorPicker.field.activeFocus || sPoll.activeFocus || sHot.activeFocus ? "yes" : ""

  function handleTextKey(t) {
    if (!service) return
    if (settingsOpen) {
      if (t === "s") closeSettings()
      return
    }
    if (t === "s") { openSettings(); return }
    if (t === "r") { service.reload(); return }
    if (tab === "fans") {
      switch (t) {
      case "+": nudgePoint(0, 8); break
      case "=": nudgePoint(0, 8); break
      case "-": nudgePoint(0, -8); break
      case "_": nudgePoint(0, -8); break
      case "a": service.applyCurve(); break
      case "x": service.stopCurve(); break
      case "0": service.resetCurve(); break
      case "1": applyPreset("silent"); break
      case "2": applyPreset("balanced"); break
      case "3": applyPreset("cool"); break
      case "4": applyPreset("max"); break
      case "f": curveFan = curveFan === "cpu" ? "gpu" : "cpu"; pointCursor = 0; break
      case "d": setCurvePoints(Model.removePoint(curvePoints, pointCursor)); break
      case "p": service.cycleProfile(1); break
      default: break
      }
      return
    }
    if (tab === "rgb") {
      switch (t) {
      case "c": Qt.callLater(function() { colorPicker.field.forceActiveFocus(); colorPicker.field.selectAll() }); break
      case "b": service.toggleBatterySync(); break
      case "t": service.toggleThemeSync(); break
      case "o": service.toggleLights(); break
      case "H": moveRgbSubTab(-1); break
      case "L": moveRgbSubTab(1); break
      case "i": if (rgbSubTabEffective === "chassis") { var current = regions[regionCursor]; if (current) service.identifyRegion(current.id) } break
      case "a": if (rgbSubTabEffective === "chassis") selectAllRegions(); break
      case "x": if (rgbSubTabEffective === "chassis") clearRegionSelection(); break
      case "K": if (rgbSubTabEffective === "keyboard") selectAllKeys(); break
      case "X": if (rgbSubTabEffective === "keyboard") clearKeySelection(); break
      case "p": if (rgbSubTabEffective === "keyboard") { if (keySelection.length) toggleKeySelectionPower() } else toggleRegionPower(); break
      default: break
      }
      return
    }
    switch (t) {
    case "+": adjustPowerField(1); break
    case "=": adjustPowerField(1); break
    case "-": adjustPowerField(-1); break
    case "_": adjustPowerField(-1); break
    case "1": service.setProfile(Model.toList(service.profileChoices)[0] || ""); break
    case "2": service.setProfile(Model.toList(service.profileChoices)[1] || ""); break
    case "3": service.setProfile(Model.toList(service.profileChoices)[2] || ""); break
    case "4": service.setProfile(Model.toList(service.profileChoices)[3] || ""); break
    case "p": service.cycleProfile(1); break
    case "t": service.toggleTurbo(); break
    case "w": service.applyCurve(); break
    default: break
    }
  }

  KeyboardPanel {
    id: panel
    anchorItem: root.anchorItem
    owner: root.barIdentity
    bar: root.bar
    open: root.opened
    focusTarget: keyCatcher
    contentWidth: panel.fittedContentWidth(Style.space(680))
    contentHeight: panel.fittedContentHeight(column.implicitHeight)

    PanelKeyCatcher {
      id: keyCatcher
      anchors.fill: parent
      blocked: root.editorFocusBlock !== ""

      onMoveRequested: function(dx, dy) {
        if (root.settingsOpen) return
        if (dy !== 0) root.moveCursor(dy)
        else if (dx !== 0) root.moveTab(dx)
      }

      onActivateRequested: {
        if (root.settingsOpen) { root.saveSettings(); return }
        if (root.tab === "fans" && root.service) root.service.applyCurve()
        else if (root.tab === "rgb") {
          if (root.rgbSubTabEffective === "chassis") {
            var current = root.regions[root.regionCursor]
            if (current) root.toggleRegionSelection(current.id)
          }
        }
        else root.adjustPowerField(1)
      }

      onCloseRequested: {
        if (root.settingsOpen) { root.closeSettings(); return }
        root.close()
      }

      onTabRequested: function(direction) { root.switchPanel(direction) }
      onTextKey: function(t) { root.handleTextKey(t) }

      Column {
        id: column
        anchors.fill: parent
        spacing: Style.space(12)

        Item {
          width: parent.width
          implicitHeight: Math.max(heroIcon.implicitHeight, heroLabels.implicitHeight, headerActions.implicitHeight)

          Text {
            id: heroIcon
            textFormat: Text.PlainText
            anchors.left: parent.left
            anchors.verticalCenter: parent.verticalCenter
            text: root.heroGlyph
            color: root.available ? root.accent : root.dim
            font.family: root.fontFamily
            font.pixelSize: Style.font.display
            opacity: root.available ? 1.0 : 0.5
          }

          Column {
            id: heroLabels
            anchors.left: heroIcon.right
            anchors.leftMargin: Style.space(14)
            anchors.right: headerActions.left
            anchors.rightMargin: Style.space(10)
            anchors.verticalCenter: parent.verticalCenter
            spacing: Style.space(2)

            Text {
              text: root.hw.model ? String(root.hw.model) : "Alienware"
              color: root.fg
              font.family: root.fontFamily
              font.pixelSize: Style.font.title
              font.bold: true
              elide: Text.ElideRight
              width: parent.width
            }

            Text {
              textFormat: Text.PlainText
              text: root.statusCaps
              color: root.health === "error" || root.health === "missing" ? root.urgentColor : root.dim
              font.family: root.fontFamily
              font.pixelSize: Style.font.caption
              font.bold: true
              font.letterSpacing: 1.2
              elide: Text.ElideRight
              width: parent.width
            }
          }

          Row {
            id: headerActions
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            spacing: Style.space(2)

            PanelActionButton {
              iconText: root.service && root.service.lightsOn ? "󱄄" : "󰛨"
              tooltipText: root.service && root.service.lightsOn ? "Turn the lights off (o)" : "Turn the lights on (o)"
              enabled: root.rgbConnected
              foreground: root.fg
              fontFamily: root.fontFamily
              hasCursor: root.service ? root.service.lightsOn : false
              onClicked: if (root.service) root.service.toggleLights()
            }
            PanelActionButton {
              iconText: "󰑓"
              tooltipText: "Re-read the hardware (r)"
              foreground: root.fg
              fontFamily: root.fontFamily
              onClicked: if (root.service) root.service.reload()
            }
            PanelActionButton {
              iconText: "󰒓"
              tooltipText: root.settingsOpen ? "Back (s)" : "Settings (s)"
              foreground: root.fg
              fontFamily: root.fontFamily
              hasCursor: root.settingsOpen
              onClicked: root.toggleSettings()
            }
          }
        }

        Card {
          visible: root.health === "missing" && !root.settingsOpen
          edge: root.urgentColor

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            text: root.service ? root.service.statusLine : "Loading"
            color: root.urgentColor
            font.family: root.fontFamily
            font.pixelSize: Style.font.bodySmall
            font.bold: true
          }

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            text: "The alienwarectl helper is not installed. The button opens a terminal so you can see every command before it runs. By hand: cd packaging/bin && makepkg -si"
            color: root.dim
            font.family: root.fontFamily
            font.pixelSize: Style.font.caption
          }

          Button {
            iconText: "󰏗"
            text: "Install the helper"
            bordered: true
            focusable: true
            foreground: root.fg
            fontFamily: root.fontFamily
            fontSize: Style.font.caption
            onClicked: if (root.service) root.service.installHelper()
          }
        }

        Card {
          id: modeCard
          visible: !root.settingsOpen && root.available

          CardTitle {
            title: "THERMAL MODE"
            detail: root.hw.profile.current ? Model.profileLabel(root.hw.profile.current, root.hw.profile.gmodeForced) : ""
          }

          Row {
            id: modeRow
            width: parent.width
            spacing: Style.space(4)
            readonly property var choices: root.service ? Model.toList(root.service.profileChoices) : []
            readonly property real cell: choices.length ? (width - spacing * (choices.length - 1)) / choices.length : width

            Repeater {
              model: modeRow.choices

              Rectangle {
                id: modeTile
                required property var modelData
                required property int index
                readonly property bool current: root.hw.profile.current === modelData
                readonly property bool usable: root.available && root.hw.profile.writable
                width: modeRow.cell
                height: modeCol.implicitHeight + Style.space(14)
                radius: Math.max(Style.cornerRadius, Style.space(3))
                color: current ? Util.alpha(root.accent, 0.22) : (modeMouse.containsMouse && usable ? Style.hoverFillFor(root.fg, root.accent) : "transparent")
                border.width: current ? Math.max(2, Style.normalBorderWidth * 2) : Style.normalBorderWidth
                border.color: current ? root.accent : Util.alpha(root.fg, 0.18)
                opacity: usable ? 1.0 : 0.5
                Behavior on color { ColorAnimation { duration: 140 } }

                Column {
                  id: modeCol
                  anchors.centerIn: parent
                  width: parent.width - Style.space(8)
                  spacing: Style.space(3)

                  Text {
                    width: parent.width
                    horizontalAlignment: Text.AlignHCenter
                    textFormat: Text.PlainText
                    text: Model.profileGlyph(modeTile.modelData)
                    color: modeTile.current ? root.accent : root.fg
                    font.family: root.fontFamily
                    font.pixelSize: Style.font.title
                  }

                  Text {
                    width: parent.width
                    horizontalAlignment: Text.AlignHCenter
                    textFormat: Text.PlainText
                    text: Model.profileLabel(modeTile.modelData, root.hw.profile.gmodeForced)
                    color: modeTile.current ? root.fg : root.dim
                    font.family: root.fontFamily
                    font.pixelSize: Style.font.caption
                    font.bold: modeTile.current
                    elide: Text.ElideRight
                  }
                }

                MouseArea {
                  id: modeMouse
                  anchors.fill: parent
                  hoverEnabled: true
                  enabled: modeTile.usable
                  cursorShape: Qt.PointingHandCursor
                  onClicked: if (root.service) root.service.setProfile(modeTile.modelData)
                }
              }
            }
          }

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            visible: root.service ? root.service.profileWarning !== "" : false
            text: "󰀦  " + (root.service ? root.service.profileWarning : "")
            color: root.urgentColor
            font.family: root.fontFamily
            font.pixelSize: Style.font.caption
          }
        }

        Rectangle {
          id: tabStrip
          width: parent.width
          height: tabRow.implicitHeight + Style.space(6)
          visible: !root.settingsOpen
          radius: Math.max(Style.cornerRadius, Style.space(4))
          color: Style.normalFillFor(root.fg, root.accent)
          border.color: Util.alpha(root.fg, 0.15)
          border.width: Style.normalBorderWidth

          Row {
            id: tabRow
            anchors.centerIn: parent
            width: parent.width - Style.space(6)
            spacing: Style.space(3)

            Repeater {
              model: root.tabs

              Rectangle {
                id: tabTile
                required property var modelData
                readonly property bool current: root.tab === modelData.key
                width: (tabRow.width - tabRow.spacing * (root.tabs.length - 1)) / root.tabs.length
                height: tabLabel.implicitHeight + Style.space(12)
                radius: Math.max(Style.cornerRadius - 2, Style.space(3))
                color: current ? Util.alpha(root.fg, 0.14) : (tabMouse.containsMouse ? Util.alpha(root.fg, 0.06) : "transparent")
                Behavior on color { ColorAnimation { duration: 120 } }

                Text {
                  id: tabLabel
                  anchors.centerIn: parent
                  textFormat: Text.PlainText
                  text: tabTile.modelData.icon + "  " + tabTile.modelData.label
                  color: tabTile.current ? root.fg : root.dim
                  font.family: root.fontFamily
                  font.pixelSize: Style.font.bodySmall
                  font.bold: tabTile.current
                }

                Rectangle {
                  visible: tabTile.current
                  anchors.bottom: parent.bottom
                  anchors.horizontalCenter: parent.horizontalCenter
                  width: Style.space(24)
                  height: Math.max(2, Style.space(2))
                  radius: height / 2
                  color: root.accent
                }

                MouseArea {
                  id: tabMouse
                  anchors.fill: parent
                  hoverEnabled: true
                  cursorShape: Qt.PointingHandCursor
                  onClicked: root.setTab(tabTile.modelData.key)
                }
              }
            }
          }
        }

        Column {
          id: fansTab
          width: parent.width
          spacing: Style.space(12)
          visible: root.tab === "fans" && !root.settingsOpen

          Row {
            id: fanCards
            width: parent.width
            spacing: Style.space(8)
            readonly property real cardWidth: (width - spacing) / 2

            FanCard {
              width: fanCards.cardWidth
              fan: root.cpuFan
              temp: root.cpuTemp
              stale: root.stale
              hotTemp: root.hotTemp
              fg: root.fg
              dim: root.dim
              accent: root.accent
              alertColor: root.urgentColor
              fontFamily: root.fontFamily
            }

            FanCard {
              width: fanCards.cardWidth
              fan: root.gpuFan
              temp: root.gpuTemp
              stale: root.stale
              hotTemp: root.hotTemp
              fg: root.fg
              dim: root.dim
              accent: root.accent
              alertColor: root.urgentColor
              fontFamily: root.fontFamily
            }
          }

          Card {
            CardTitle {
              title: "BOOST CURVE"
              detail: root.hw.curve.active ? "Running" : "Idle"
              detailColor: root.hw.curve.active ? root.accent : root.dim

              ButtonGroup {
                options: [{ value: "cpu", label: "CPU fan" }, { value: "gpu", label: "GPU fan" }]
                value: root.curveFan
                foreground: root.fg
                accent: root.accent
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                focusable: false
                onChanged: function(v) { root.curveFan = v; root.pointCursor = 0 }
              }
            }

            CurveEditor {
              width: parent.width
              points: root.curvePoints
              floorPercent: root.curveFloor
              currentTemp: root.curveTemp
              selectedIndex: root.pointCursor
              editable: root.available
              fg: root.fg
              dim: root.dim
              accent: root.accent
              alertColor: root.urgentColor
              fontFamily: root.fontFamily
              onEdited: function(next) { root.setCurvePoints(next) }
              onPointPicked: function(index) { root.pointCursor = index }
            }

            Item {
              width: parent.width
              implicitHeight: Math.max(presetFlow.implicitHeight, curveActions.implicitHeight)

              Flow {
                id: presetFlow
                anchors.left: parent.left
                anchors.verticalCenter: parent.verticalCenter
                width: parent.width - curveActions.implicitWidth - Style.space(12)
                spacing: Style.space(4)

                Repeater {
                  model: Model.presetNames()

                  Button {
                    required property var modelData
                    required property int index
                    text: modelData.charAt(0).toUpperCase() + modelData.slice(1)
                    tooltipText: "Preset " + (index + 1)
                    bordered: true
                    foreground: root.fg
                    fontFamily: root.fontFamily
                    fontSize: Style.font.caption
                    onClicked: root.applyPreset(modelData)
                  }
                }
              }

              Row {
                id: curveActions
                anchors.right: parent.right
                anchors.verticalCenter: parent.verticalCenter
                spacing: Style.space(4)

                Button {
                  iconText: "󰑓"
                  bordered: true
                  foreground: root.fg
                  fontFamily: root.fontFamily
                  fontSize: Style.font.caption
                  tooltipText: "Back to the default points (0)"
                  onClicked: if (root.service) root.service.resetCurve()
                }

                Button {
                  iconText: "󰓛"
                  text: "Stop"
                  bordered: true
                  enabled: root.available && root.hw.curve.active
                  foreground: root.fg
                  fontFamily: root.fontFamily
                  fontSize: Style.font.caption
                  tooltipText: "Hand the fans back to firmware (x)"
                  onClicked: if (root.service) root.service.stopCurve()
                }

                Button {
                  iconText: "󰐊"
                  text: root.hw.curve.active ? "Update" : "Apply"
                  bordered: true
                  selected: true
                  enabled: root.available
                  foreground: root.fg
                  accent: root.accent
                  fontFamily: root.fontFamily
                  fontSize: Style.font.caption
                  tooltipText: "Send the curve to the daemon (a)"
                  onClicked: if (root.service) root.service.applyCurve()
                }
              }
            }

            Row {
              width: parent.width
              spacing: Style.space(16)

              ValueSlider {
                width: (parent.width - parent.spacing) / 2
                label: "Hysteresis"
                valueText: (root.service ? root.service.curveHysteresis : 3) + " °C"
                value: root.service ? root.service.curveHysteresis : 3
                from: 0
                to: 15
                stepSize: 1
                note: "Drop needed before the boost steps down"
                fg: root.fg
                dim: root.dim
                fontFamily: root.fontFamily
                onMoved: function(next) { if (root.service) root.service.setCurveHysteresis(next) }
              }

              ValueSlider {
                width: (parent.width - parent.spacing) / 2
                label: "Interval"
                valueText: (root.service ? root.service.curveInterval : 2) + " s"
                value: root.service ? root.service.curveInterval : 2
                from: 1
                to: 30
                stepSize: 1
                note: "How often the daemon re-reads temperatures"
                fg: root.fg
                dim: root.dim
                fontFamily: root.fontFamily
                onMoved: function(next) { if (root.service) root.service.setCurveInterval(next) }
              }
            }
          }
        }

        Column {
          id: rgbTab
          width: parent.width
          spacing: Style.space(12)
          visible: root.tab === "rgb" && !root.settingsOpen

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            visible: !root.rgbConnected
            text: "󰀦  " + (root.rgbError !== "" ? root.rgbError : "Waiting for the lighting controller")
            color: root.urgentColor
            font.family: root.fontFamily
            font.pixelSize: Style.font.bodySmall
          }

          Card {
            Row {
              id: switchRow
              width: parent.width
              spacing: Style.space(10)
              readonly property real cell: (width - spacing * 2) / 3

              SwitchTile {
                width: switchRow.cell
                icon: "󱄄"
                title: "Lighting"
                detail: root.rgb.device.ready ? (root.service && root.service.lightsOn ? "On" : "Off") : "No controller"
                checked: root.service ? root.service.lightsOn : false
                enabled: root.rgbConnected
                onToggled: if (root.service) root.service.toggleLights()
              }

              SwitchTile {
                width: switchRow.cell
                icon: "󰏘"
                title: "Follow theme"
                detail: "Uses the accent"
                checked: root.service ? root.service.themeSync : false
                enabled: root.rgbConnected
                onToggled: if (root.service) root.service.toggleThemeSync()
              }

              SwitchTile {
                width: switchRow.cell
                icon: "󰂄"
                title: "Battery on ⏻"
                detail: root.service && root.service.batterySync ? root.batteryLine : "Power button is yours"
                checked: root.service ? root.service.batterySync : true
                enabled: root.rgbConnected
                onToggled: if (root.service) root.service.toggleBatterySync()
              }
            }

            ValueSlider {
              width: parent.width
              label: "Brightness"
              valueText: (root.service ? root.service.brightness : 100) + "%"
              value: root.service ? root.service.brightness : 100
              from: 0
              to: 100
              stepSize: 5
              editable: root.rgbConnected
              fg: root.fg
              dim: root.dim
              fontFamily: root.fontFamily
              onMoved: function(next) { if (root.service) root.service.setBrightness(next) }
            }
          }

          Item {
            width: parent.width
            implicitHeight: Math.max(surfaceGroup.implicitHeight, surfaceActions.implicitHeight)

            ButtonGroup {
              id: surfaceGroup
              anchors.left: parent.left
              anchors.verticalCenter: parent.verticalCenter
              visible: root.kbdPresent
              options: [{ value: "keyboard", label: "󰌌  Keyboard" }, { value: "chassis", label: "󰢚  Chassis" }]
              value: root.rgbSubTabEffective
              foreground: root.fg
              accent: root.accent
              fontFamily: root.fontFamily
              fontSize: Style.font.caption
              focusable: false
              onChanged: function(v) { root.setRgbSubTab(v) }
            }

            Row {
              id: surfaceActions
              anchors.right: parent.right
              anchors.verticalCenter: parent.verticalCenter
              spacing: Style.space(4)

              Text {
                anchors.verticalCenter: parent.verticalCenter
                rightPadding: Style.space(6)
                textFormat: Text.PlainText
                text: root.paintTarget ? root.paintTarget + " selected" : "Nothing selected"
                color: root.paintTarget ? root.fg : root.dim
                font.family: root.fontFamily
                font.pixelSize: Style.font.caption
              }

              Button {
                text: "All"
                bordered: true
                foreground: root.fg
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                tooltipText: root.rgbSubTabEffective === "keyboard" ? "Select every mapped key (K)" : "Select every region (a)"
                onClicked: root.rgbSubTabEffective === "keyboard" ? root.selectAllKeys() : root.selectAllRegions()
              }

              Button {
                text: "None"
                bordered: true
                foreground: root.fg
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                tooltipText: root.rgbSubTabEffective === "keyboard" ? "Clear the key selection (X)" : "Clear the selection (x)"
                onClicked: {
                  if (root.rgbSubTabEffective === "keyboard") {
                    root.clearKeySelection()
                    if (root.regionSelection.indexOf("power") !== -1) root.toggleRegionSelection("power")
                  } else root.clearRegionSelection()
                }
              }

              Button {
                visible: root.rgbSubTabEffective === "keyboard"
                iconText: "󰐥"
                bordered: true
                enabled: root.keySelection.length > 0
                foreground: root.fg
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                tooltipText: "Turn the selected keys on or off (p)"
                onClicked: root.toggleKeySelectionPower()
              }
            }
          }

          KeyboardMap {
            width: parent.width
            visible: root.kbdPresent && root.rgbSubTabEffective === "keyboard"
            present: root.kbdPresent
            keyColors: root.service ? root.service.keyColors : ({})
            keyOn: root.service ? root.service.keyOn : ({})
            selection: root.keySelection
            fg: root.fg
            dim: root.dim
            accent: root.accent
            fontFamily: root.fontFamily
            powerColor: root.powerButtonColor
            powerLit: root.powerButtonLit
            powerSelected: root.regionSelection.indexOf("power") !== -1
            onToggleRequested: function(keyId) { root.toggleKeySelection(keyId) }
            onDragSelectRequested: function(keyIds) { root.addKeysToSelection(keyIds) }
            onPowerClicked: root.pickPowerKey()
          }

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            visible: root.kbdPresent && root.rgbSubTabEffective === "keyboard"
            text: "Click or drag across keys to select them. Click ⏻ for the power button" + (root.service && root.service.batterySync ? " once Battery on ⏻ is off." : ".") + " Faded keys have no confirmed LED yet."
            color: root.dim
            font.family: root.fontFamily
            font.pixelSize: Style.font.caption
          }

          RegionMap {
            width: parent.width
            visible: root.rgbSubTabEffective === "chassis"
            regions: root.regions
            swatches: root.regionSwatchMap
            poweredMap: root.regionPoweredMap
            selection: root.regionSelection
            cursorIndex: root.regionCursor
            fg: root.fg
            dim: root.dim
            accent: root.accent
            fontFamily: root.fontFamily
            onPicked: function(rowIndex) { root.regionCursor = rowIndex }
            onToggleRequested: function(regionId) {
              if (regionId === "power" && root.service && root.service.batterySync) { root.pickPowerKey(); return }
              root.toggleRegionSelection(regionId)
            }
            onIdentifyRequested: function(regionId) { if (root.service) root.service.identifyRegion(regionId) }
            onPowerRequested: function(regionId) { root.setRegionPower([regionId], !root.isRegionOn(regionId)) }
          }

          ColorPicker {
            id: colorPicker
            width: parent.width
            hex: root.pickedHex
            swatches: root.service ? root.service.themeSwatches : []
            themeEnabled: root.rgbConnected
            canApply: root.rgbConnected && root.paintTarget !== ""
            targetText: root.paintTarget
            fg: root.fg
            dim: root.dim
            accent: root.accent
            fontFamily: root.fontFamily
            onPicked: function(nextHex) { root.pickedHex = nextHex }
            onApplyRequested: root.applyColorField()
            onThemeColorRequested: if (root.service) root.service.applyThemeColor()
            onFieldEscaped: root.refocusPanel()
          }
        }

        Column {
          id: powerTab
          width: parent.width
          spacing: Style.space(12)
          visible: root.tab === "power" && !root.settingsOpen

          Grid {
            id: statGrid
            width: parent.width
            columns: 4
            columnSpacing: Style.space(8)
            rowSpacing: Style.space(8)
            readonly property real cell: (width - columnSpacing * (columns - 1)) / columns

            StatTile {
              width: statGrid.cell
              icon: "󰻠"
              label: "CPU"
              value: Model.formatTemp(root.cpuTemp, true)
              sub: Model.formatClock(Model.cpuClock(root.hw))
              alert: root.cpuTemp !== null && root.cpuTemp >= root.hotTemp
              muted: root.stale
            }
            StatTile {
              width: statGrid.cell
              icon: "󰢮"
              label: "GPU"
              value: Model.formatTemp(root.gpuTemp, true)
              sub: root.hw.gpu.available ? Model.formatWatts(root.hw.gpu.draw, 1) + " draw" : (root.hw.gpu.asleep ? "asleep, saving power" : "no reading")
              alert: root.gpuTemp !== null && root.gpuTemp >= root.hotTemp
              muted: root.stale
            }
            StatTile {
              width: statGrid.cell
              icon: "󱐋"
              label: "GPU limit"
              value: Model.formatWatts(root.hw.gpu.limit)
              sub: root.hw.gpu.available ? "default " + Model.formatWatts(root.hw.gpu.defaultLimit) : Model.PL_LOCKED_NOTE
              muted: true
            }
            StatTile {
              width: statGrid.cell
              icon: "󰈐"
              label: "Fans"
              value: Model.maxRpm(root.hw) > 0 ? Model.maxRpm(root.hw) + " rpm" : "Stopped"
              sub: "CPU " + (root.cpuFan ? root.cpuFan.percent : 0) + "% · GPU " + (root.gpuFan ? root.gpuFan.percent : 0) + "%"
              muted: root.stale
            }
          }

          Card {
            CardTitle {
              title: "CPU POWER LIMITS"
              detail: root.hw.power.available ? "RAPL" : "Unavailable"
            }

            Text {
              width: parent.width
              wrapMode: Text.Wrap
              visible: !root.hw.power.available
              text: "The RAPL interface is not readable on this machine"
              color: root.dim
              font.family: root.fontFamily
              font.pixelSize: Style.font.caption
            }

            Repeater {
              model: root.hw.power.available ? root.constraints : []

              ValueSlider {
                required property var modelData
                required property int index
                width: parent.width
                label: modelData.label
                valueText: Model.formatWatts(modelData.watts)
                value: modelData.watts
                from: 5
                to: 250
                stepSize: 5
                editable: Model.powerLimitWritable(modelData)
                focused: root.powerCursor === index
                note: Model.powerLimitNote(modelData)
                fg: root.fg
                dim: root.dim
                fontFamily: root.fontFamily
                onMoved: function(next) {
                  if (root.service && Model.powerLimitWritable(modelData)) root.service.setPowerLimit(modelData.index + 1, next)
                }
              }
            }
          }

          Card {
            edge: root.powerCursor >= root.constraints.length ? root.accent : Util.alpha(root.fg, 0.15)

            SwitchTile {
              width: parent.width
              icon: "󰓅"
              title: "Intel turbo"
              detail: root.hw.turbo.available ? "Boost above base clock. Off runs cooler and quieter." : "no_turbo is not readable on this machine"
              checked: root.hw.turbo.enabled
              enabled: root.hw.turbo.available && root.available
              onToggled: if (root.service) root.service.toggleTurbo()
            }
          }
        }

        Column {
          id: settingsColumn
          width: parent.width
          spacing: Style.space(12)
          visible: root.settingsOpen

          Card {
            CardTitle { title: "BAR WIDGET" }

            Text {
              width: parent.width
              wrapMode: Text.Wrap
              textFormat: Text.PlainText
              text: "What the bar shows. Pick any, in this order. Each reading gets its own icon."
              color: root.dim
              font.family: root.fontFamily
              font.pixelSize: Style.font.caption
            }

            Flow {
              width: parent.width
              spacing: Style.space(4)

              Repeater {
                model: Model.BAR_ITEMS

                Button {
                  required property var modelData
                  readonly property bool on: Model.parseBarItems(root.dDisplay).indexOf(modelData.id) !== -1
                  iconText: modelData.glyph
                  text: modelData.label
                  selected: on
                  bordered: true
                  focusable: true
                  foreground: root.fg
                  accent: root.accent
                  fontFamily: root.fontFamily
                  fontSize: Style.font.caption
                  tooltipText: on ? "Shown in the bar" : "Hidden"
                  onClicked: root.dDisplay = Model.serializeBarItems(Model.toggleBarItem(root.dDisplay, modelData.id))
                }
              }
            }

            Rectangle {
              width: previewRow.implicitWidth + Style.space(20)
              height: previewRow.implicitHeight + Style.space(10)
              radius: Math.max(Style.cornerRadius, Style.space(3))
              color: root.bar ? root.bar.background : Color.background
              border.color: Util.alpha(root.fg, 0.2)
              border.width: Style.normalBorderWidth

              Row {
                id: previewRow
                anchors.centerIn: parent
                spacing: Style.space(10)

                Repeater {
                  model: Model.barView(root.hw, root.health, root.dDisplay, root.hotTemp).segments

                  Row {
                    required property var modelData
                    spacing: Style.space(4)

                    Text {
                      anchors.verticalCenter: parent.verticalCenter
                      textFormat: Text.PlainText
                      text: modelData.glyph
                      color: modelData.alert ? root.urgentColor : (root.bar ? root.bar.foreground : root.fg)
                      font.family: root.fontFamily
                      font.pixelSize: modelData.id === "logo" ? Style.bar.iconFont : Math.round(Style.bar.iconFont * 0.8)
                    }

                    Text {
                      visible: modelData.text !== ""
                      anchors.verticalCenter: parent.verticalCenter
                      textFormat: Text.PlainText
                      text: modelData.text
                      color: modelData.alert ? root.urgentColor : (root.bar ? root.bar.foreground : root.fg)
                      font.family: root.fontFamily
                      font.pixelSize: Style.font.caption
                    }
                  }
                }
              }
            }

            Row {
              width: parent.width
              spacing: Style.space(16)

              Column {
                width: (parent.width - parent.spacing) / 2
                spacing: Style.space(6)

                Text {
                  textFormat: Text.PlainText
                  text: "Hot at (°C, 50 to 110)"
                  color: root.dim
                  font.family: root.fontFamily
                  font.pixelSize: Style.font.caption
                }

                TextField {
                  id: sHot
                  width: parent.width
                  placeholderText: "90"
                  foreground: root.fg
                  font.family: root.fontFamily
                  onTextChanged: root.dHot = text
                  Keys.onPressed: function(event) { root.handleSettingsKey(event) }
                }
              }

              Column {
                width: (parent.width - parent.spacing) / 2
                spacing: Style.space(6)

                Text {
                  textFormat: Text.PlainText
                  text: "Poll every (seconds, 1 to 30)"
                  color: root.dim
                  font.family: root.fontFamily
                  font.pixelSize: Style.font.caption
                }

                TextField {
                  id: sPoll
                  width: parent.width
                  placeholderText: "2"
                  foreground: root.fg
                  font.family: root.fontFamily
                  onTextChanged: root.dPoll = text
                  Keys.onPressed: function(event) { root.handleSettingsKey(event) }
                }
              }
            }

            Text {
              width: parent.width
              wrapMode: Text.Wrap
              text: "Lighting switches (follow theme, battery on the power button) live on the Lighting tab."
              color: root.dim
              font.family: root.fontFamily
              font.pixelSize: Style.font.caption
            }
          }

          Item {
            width: parent.width
            implicitHeight: Math.max(settingsButtons.implicitHeight, settingsStatusText.implicitHeight)

            Row {
              id: settingsButtons
              spacing: Style.space(6)
              anchors.left: parent.left
              anchors.verticalCenter: parent.verticalCenter

              Button {
                iconText: "󰆓"
                text: "Save"
                bordered: true
                selected: true
                focusable: true
                foreground: root.fg
                accent: root.accent
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                tooltipText: "Enter"
                onClicked: root.saveSettings()
              }

              Button {
                text: "Cancel"
                bordered: true
                focusable: true
                foreground: root.fg
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                tooltipText: "Esc"
                onClicked: root.closeSettings()
              }
            }

            Text {
              id: settingsStatusText
              textFormat: Text.PlainText
              anchors.right: parent.right
              anchors.verticalCenter: parent.verticalCenter
              width: Math.max(0, parent.width - settingsButtons.implicitWidth - Style.space(10))
              horizontalAlignment: Text.AlignRight
              wrapMode: Text.Wrap
              text: root.settingsStatus || "Saved to shell.json"
              color: root.settingsError ? root.urgentColor : root.dim
              font.family: root.fontFamily
              font.pixelSize: Style.font.caption
            }
          }
        }

        Text {
          width: parent.width
          textFormat: Text.PlainText
          visible: root.service ? root.service.actionStatus !== "" : false
          text: root.service ? root.service.actionStatus : ""
          color: root.service && root.service.actionError ? root.urgentColor : root.fg
          font.family: root.fontFamily
          font.pixelSize: Style.font.caption
          wrapMode: Text.Wrap
        }

        Text {
          width: parent.width
          textFormat: Text.PlainText
          text: root.footerText
          color: Qt.darker(root.fg, 1.8)
          font.family: root.fontFamily
          font.pixelSize: Style.font.caption
          elide: Text.ElideRight
        }
      }
    }
  }

  component Card: Rectangle {
    id: card
    default property alias content: cardBody.data
    property color edge: Util.alpha(root.fg, 0.15)
    width: parent ? parent.width : 0
    implicitHeight: cardBody.implicitHeight + Style.space(24)
    radius: Math.max(Style.cornerRadius, Style.space(4))
    color: Style.normalFillFor(root.fg, root.accent)
    border.color: card.edge
    border.width: Style.normalBorderWidth

    Column {
      id: cardBody
      x: Style.space(12)
      y: Style.space(12)
      width: card.width - Style.space(24)
      spacing: Style.space(10)
    }
  }

  component CardTitle: Item {
    id: ct
    property string title: ""
    property string detail: ""
    property color detailColor: root.dim
    default property alias trailing: ctTrailing.data
    width: parent ? parent.width : 0
    implicitHeight: Math.max(ctTitle.implicitHeight, ctTrailing.implicitHeight)

    Row {
      anchors.left: parent.left
      anchors.verticalCenter: parent.verticalCenter
      spacing: Style.space(8)

      Text {
        id: ctTitle
        anchors.verticalCenter: parent.verticalCenter
        textFormat: Text.PlainText
        text: ct.title
        color: root.fg
        font.family: root.fontFamily
        font.pixelSize: Style.font.caption
        font.bold: true
        font.letterSpacing: 1.4
      }

      Text {
        visible: ct.detail !== ""
        anchors.verticalCenter: parent.verticalCenter
        textFormat: Text.PlainText
        text: "· " + ct.detail
        color: ct.detailColor
        font.family: root.fontFamily
        font.pixelSize: Style.font.caption
      }
    }

    Row {
      id: ctTrailing
      anchors.right: parent.right
      anchors.verticalCenter: parent.verticalCenter
      spacing: Style.space(4)
    }
  }

  component SwitchTile: Item {
    id: st
    property string icon: ""
    property string title: ""
    property string detail: ""
    property bool checked: false
    signal toggled()
    implicitHeight: Math.max(stText.implicitHeight, stSwitch.implicitHeight)
    opacity: enabled ? 1.0 : 0.5

    Text {
      id: stIcon
      anchors.left: parent.left
      anchors.verticalCenter: parent.verticalCenter
      textFormat: Text.PlainText
      text: st.icon
      color: st.checked ? root.accent : root.dim
      font.family: root.fontFamily
      font.pixelSize: Style.font.title
    }

    Column {
      id: stText
      anchors.left: stIcon.right
      anchors.leftMargin: Style.space(8)
      anchors.right: stSwitch.left
      anchors.rightMargin: Style.space(6)
      anchors.verticalCenter: parent.verticalCenter
      spacing: Style.space(1)

      Text {
        width: parent.width
        textFormat: Text.PlainText
        text: st.title
        color: root.fg
        font.family: root.fontFamily
        font.pixelSize: Style.font.bodySmall
        font.bold: true
        elide: Text.ElideRight
      }

      Text {
        width: parent.width
        textFormat: Text.PlainText
        text: st.detail
        color: root.dim
        font.family: root.fontFamily
        font.pixelSize: Style.font.caption
        elide: Text.ElideRight
      }
    }

    ToggleSwitch {
      id: stSwitch
      anchors.right: parent.right
      anchors.verticalCenter: parent.verticalCenter
      checked: st.checked
      interactive: st.enabled
      foreground: root.fg
      accent: root.accent
      onToggled: st.toggled()
    }
  }

  component StatTile: Rectangle {
    id: tile
    property string icon: ""
    property string label: ""
    property string value: ""
    property string sub: ""
    property bool alert: false
    property bool muted: false
    implicitHeight: tileCol.implicitHeight + Style.space(20)
    radius: Math.max(Style.cornerRadius, Style.space(4))
    color: tile.alert ? Util.alpha(root.urgentColor, 0.12) : Style.normalFillFor(root.fg, root.accent)
    border.color: tile.alert ? root.urgentColor : Util.alpha(root.fg, 0.15)
    border.width: Style.normalBorderWidth

    Column {
      id: tileCol
      x: Style.space(10)
      y: Style.space(10)
      width: tile.width - Style.space(20)
      spacing: Style.space(4)

      Text {
        width: parent.width
        textFormat: Text.PlainText
        text: tile.icon + "  " + tile.label.toUpperCase()
        color: root.dim
        font.family: root.fontFamily
        font.pixelSize: Style.font.caption
        font.bold: true
        font.letterSpacing: 1.1
        elide: Text.ElideRight
      }

      Text {
        width: parent.width
        textFormat: Text.PlainText
        text: tile.value || "-"
        color: tile.alert ? root.urgentColor : root.fg
        opacity: tile.muted && !tile.alert ? 0.75 : 1.0
        font.family: root.fontFamily
        font.pixelSize: Style.font.title
        font.bold: true
        elide: Text.ElideRight
      }

      Text {
        width: parent.width
        textFormat: Text.PlainText
        text: tile.sub
        color: root.dim
        font.family: root.fontFamily
        font.pixelSize: Style.font.caption
        elide: Text.ElideRight
      }
    }
  }
}
