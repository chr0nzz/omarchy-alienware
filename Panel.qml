import QtQuick
import Quickshell
import qs.Commons
import qs.Ui
import "Model.js" as Model
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
    setCenterHoverRevealSuppressed(false)
    root.controller.show()
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
    statusIndex = 0
    settingsOpen = false
    settingsStatus = ""
    if (service) {
      service.reload()
    }
  }

  function close() {
    setCenterHoverRevealSuppressed(false)
    settingsOpen = false
    root.controller.hide()
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
    if (root.bar && "centerHoverRevealSuppressed" in root.bar)
      root.bar.centerHoverRevealSuppressed = value
  }

  readonly property color fg: bar ? bar.foreground : Color.foreground
  readonly property color dim: Qt.darker(fg, 1.5)
  readonly property color urgentColor: bar ? bar.urgent : Color.urgent
  readonly property color accent: Color.accent
  readonly property string fontFamily: bar ? bar.fontFamily : Style.font.family

  readonly property var tabs: [
    { key: "fans", label: "Fans" },
    { key: "rgb", label: "RGB" },
    { key: "power", label: "Power" }
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

  property int powerCursor: 0
  readonly property var constraints: Model.toList(hw.power.constraints)
  readonly property int powerFieldCount: constraints.length + 1

  property bool settingsOpen: false
  property string settingsStatus: ""
  property bool settingsError: false
  property string dPoll: "2"
  property string dHot: "90"
  property string dDisplay: "temp"
  property bool dThemeRgb: false

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

  function applyColorField() {
    if (!service) return
    var text = colorField.text
    if (!Model.validHex(text)) {
      service.reportAction(false, "Enter a colour as RRGGBB")
      return
    }
    service.setRegionColors(regionSelection, text)
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
    dDisplay = Model.normalizeDisplay(currentSetting("display", "temp"))
    dThemeRgb = Model.boolOr(currentSetting("themeRgb", false), false)
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
    entry.display = Model.normalizeDisplay(dDisplay)
    entry.themeRgb = dThemeRgb
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
    if (tab === "fans") return "h/l tab · j/k point · +/- boost · 1-4 preset · a apply · s settings · esc"
    if (tab === "rgb") return "h/l tab · j/k region · space select · a all · x clear · p power · c colour · i identify · t theme sync · esc"
    return "h/l tab · j/k field · +/- adjust · 1-4 mode · p profiles · w save · esc"
  }

  readonly property string editorFocusBlock: colorField.activeFocus || sPoll.activeFocus || sHot.activeFocus ? "yes" : ""

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
      case "c": Qt.callLater(function() { colorField.forceActiveFocus(); colorField.selectAll() }); break
      case "t": service.toggleThemeSync(); break
      case "o": service.toggleLights(); break
      case "i": var current = regions[regionCursor]; if (current) service.identifyRegion(current.id); break
      case "a": selectAllRegions(); break
      case "x": clearRegionSelection(); break
      case "p": toggleRegionPower(); break
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
    contentWidth: panel.fittedContentWidth(Style.space(470))
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
          var current = root.regions[root.regionCursor]
          if (current) root.toggleRegionSelection(current.id)
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
            color: root.available ? root.fg : root.dim
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
              text: "Alienware"
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
              color: root.health === "error" || root.health === "missing" ? root.urgentColor : Qt.darker(root.fg, 1.4)
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
              iconText: Model.profileGlyph(root.hw.profile.current)
              tooltipText: "Cycle the thermal profile (p)"
              enabled: root.available
              foreground: root.fg
              fontFamily: root.fontFamily
              onClicked: if (root.service) root.service.cycleProfile(1)
            }
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

        PanelSeparator { foreground: root.fg }

        Column {
          width: parent.width
          spacing: Style.space(6)
          visible: root.health === "missing" && !root.settingsOpen

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            text: root.service ? root.service.statusLine : "Loading"
            color: root.urgentColor
            font.family: root.fontFamily
            font.pixelSize: Style.font.bodySmall
          }

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            text: "Install the helper and start the daemon:\nsudo systemctl enable --now alienwarectl.service\nalienwarectl status"
            color: root.dim
            font.family: root.fontFamily
            font.pixelSize: Style.font.caption
          }
        }

        Row {
          id: tabStrip
          width: parent.width
          spacing: Style.space(4)
          visible: !root.settingsOpen

          Repeater {
            model: root.tabs

            Button {
              required property var modelData
              text: modelData.label
              selected: root.tab === modelData.key
              bordered: true
              foreground: root.fg
              fontFamily: root.fontFamily
              fontSize: Style.font.caption
              horizontalPadding: Style.space(10)
              verticalPadding: Style.space(3)
              onClicked: root.setTab(modelData.key)
            }
          }
        }

        Column {
          id: fansTab
          width: parent.width
          spacing: Style.space(10)
          visible: root.tab === "fans" && !root.settingsOpen

          Flow {
            width: parent.width
            spacing: Style.space(4)

            Repeater {
              model: root.service ? root.service.profileChoices : []

              Button {
                required property var modelData
                text: Model.profileLabel(modelData, root.hw.profile.gmodeForced)
                iconText: Model.profileGlyph(modelData)
                selected: root.hw.profile.current === modelData
                enabled: root.available && root.hw.profile.writable
                bordered: true
                foreground: root.fg
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                onClicked: if (root.service) root.service.setProfile(modelData)
              }
            }

          }

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            visible: root.service ? root.service.profileWarning !== "" : false
            text: root.service ? root.service.profileWarning : ""
            color: root.dim
            font.family: root.fontFamily
            font.pixelSize: Style.font.caption
          }

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

          PanelSectionHeader { text: "BOOST CURVE"; foreground: root.fg; fontFamily: root.fontFamily }

          Row {
            width: parent.width
            spacing: Style.space(4)

            Button {
              text: "CPU fan"
              selected: root.curveFan === "cpu"
              bordered: true
              foreground: root.fg
              fontFamily: root.fontFamily
              fontSize: Style.font.caption
              onClicked: { root.curveFan = "cpu"; root.pointCursor = 0 }
            }

            Button {
              text: "GPU fan"
              selected: root.curveFan === "gpu"
              bordered: true
              foreground: root.fg
              fontFamily: root.fontFamily
              fontSize: Style.font.caption
              onClicked: { root.curveFan = "gpu"; root.pointCursor = 0 }
            }

            Button {
              text: root.hw.curve.active ? "Curve running" : "Curve idle"
              iconText: root.hw.curve.active ? "󰐊" : "󰏤"
              selected: root.hw.curve.active
              bordered: true
              foreground: root.fg
              fontFamily: root.fontFamily
              fontSize: Style.font.caption
              tooltipText: root.hw.curve.active ? "Stop the curve daemon (x)" : "Apply the curve to start it (a)"
              onClicked: {
                if (!root.service) return
                if (root.hw.curve.active) root.service.stopCurve()
                else root.service.applyCurve()
              }
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

          Flow {
            width: parent.width
            spacing: Style.space(4)

            Repeater {
              model: Model.presetNames()

              Button {
                required property var modelData
                required property int index
                text: (index + 1) + " " + modelData.charAt(0).toUpperCase() + modelData.slice(1)
                bordered: true
                foreground: root.fg
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                onClicked: root.applyPreset(modelData)
              }
            }
          }

          ValueSlider {
            width: parent.width
            label: "Hysteresis"
            valueText: (root.service ? root.service.curveHysteresis : 3) + " °C"
            value: root.service ? root.service.curveHysteresis : 3
            from: 0
            to: 15
            stepSize: 1
            note: "How far the temperature must fall before the boost steps back down"
            fg: root.fg
            dim: root.dim
            fontFamily: root.fontFamily
            onMoved: function(next) { if (root.service) root.service.setCurveHysteresis(next) }
          }

          ValueSlider {
            width: parent.width
            label: "Curve interval"
            valueText: (root.service ? root.service.curveInterval : 2) + " s"
            value: root.service ? root.service.curveInterval : 2
            from: 1
            to: 30
            stepSize: 1
            note: "How often the daemon re-evaluates the curve"
            fg: root.fg
            dim: root.dim
            fontFamily: root.fontFamily
            onMoved: function(next) { if (root.service) root.service.setCurveInterval(next) }
          }

          Row {
            width: parent.width
            spacing: Style.space(6)

            Button {
              iconText: "󰄬"
              text: "Apply"
              bordered: true
              enabled: root.available
              foreground: root.fg
              fontFamily: root.fontFamily
              fontSize: Style.font.caption
              tooltipText: "Send the curve to the daemon (a)"
              onClicked: if (root.service) root.service.applyCurve()
            }

            Button {
              iconText: "󰜺"
              text: "Stop"
              bordered: true
              enabled: root.available
              foreground: root.fg
              fontFamily: root.fontFamily
              fontSize: Style.font.caption
              tooltipText: "Hand the fans back to firmware (x)"
              onClicked: if (root.service) root.service.stopCurve()
            }

            Button {
              iconText: "󰑓"
              text: "Reset"
              bordered: true
              foreground: root.fg
              fontFamily: root.fontFamily
              fontSize: Style.font.caption
              tooltipText: "Back to the default points (0)"
              onClicked: if (root.service) root.service.resetCurve()
            }
          }
        }

        Column {
          id: rgbTab
          width: parent.width
          spacing: Style.space(10)
          visible: root.tab === "rgb" && !root.settingsOpen

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            visible: !root.rgbConnected
            text: root.rgbError !== "" ? root.rgbError : "Waiting for the lighting controller"
            color: root.urgentColor
            font.family: root.fontFamily
            font.pixelSize: Style.font.bodySmall
          }

          Row {
            width: parent.width
            spacing: Style.space(6)

            Toggle {
              width: (parent.width - parent.spacing) / 2
              label: "Lighting"
              description: root.rgb.device.ready ? "" : "No controller found"
              checked: root.service ? root.service.lightsOn : false
              enabled: root.rgbConnected
              foreground: root.fg
              fontFamily: root.fontFamily
              titleSize: Style.font.bodySmall
              descriptionSize: Style.font.caption
              onClicked: if (root.service) root.service.toggleLights()
            }

            Toggle {
              width: (parent.width - parent.spacing) / 2
              label: "Follow theme"
              checked: root.service ? root.service.themeSync : false
              enabled: root.rgbConnected
              foreground: root.fg
              fontFamily: root.fontFamily
              titleSize: Style.font.bodySmall
              descriptionSize: Style.font.caption
              onClicked: if (root.service) root.service.toggleThemeSync()
            }
          }

          Item {
            width: parent.width
            implicitHeight: Math.max(regionsHeader.implicitHeight, regionsActions.implicitHeight)

            PanelSectionHeader {
              id: regionsHeader
              anchors.left: parent.left
              anchors.verticalCenter: parent.verticalCenter
              text: "REGIONS"
              foreground: root.fg
              fontFamily: root.fontFamily
            }

            Row {
              id: regionsActions
              anchors.right: parent.right
              anchors.verticalCenter: parent.verticalCenter
              spacing: Style.space(6)

              Button {
                text: "Select all"
                bordered: true
                foreground: root.fg
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                tooltipText: "Select every region (a)"
                onClicked: root.selectAllRegions()
              }

              Button {
                text: "Clear"
                bordered: true
                foreground: root.fg
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                tooltipText: "Clear the selection (x)"
                onClicked: root.clearRegionSelection()
              }
            }
          }

          Column {
            width: parent.width
            spacing: Style.space(2)

            Repeater {
              model: root.regions

              RegionRow {
                required property var modelData
                required property int index
                width: parent.width
                region: modelData
                rowIndex: index
                swatch: root.service && root.service.regionColors[modelData.id] ? Model.hexPreview(root.service.regionColors[modelData.id]) : ""
                selected: root.regionSelection.indexOf(modelData.id) !== -1
                powered: root.service ? root.service.regionOn[modelData.id] !== false : true
                hasCursor: root.regionCursor === index
                fg: root.fg
                dim: root.dim
                fontFamily: root.fontFamily
                onPicked: function(rowIndex) { root.regionCursor = rowIndex }
                onToggleRequested: function(regionId) { root.toggleRegionSelection(regionId) }
                onIdentifyRequested: function(regionId) { if (root.service) root.service.identifyRegion(regionId) }
                onPowerRequested: function(regionId) { root.setRegionPower([regionId], !root.isRegionOn(regionId)) }
              }
            }
          }

          PanelSectionHeader { text: "COLOUR"; foreground: root.fg; fontFamily: root.fontFamily; topPadding: Style.space(4) }

          Row {
            id: colorRow
            width: parent.width
            spacing: Style.space(6)

            BorderSurface {
              width: Style.space(28)
              height: Style.space(28)
              radius: Style.cornerRadius
              anchors.verticalCenter: parent.verticalCenter
              color: Model.validHex(colorField.text) ? Model.hexPreview(colorField.text) : Util.alpha(root.fg, 0.15)
              borderSpec: Border.flat(Util.alpha(root.fg, 0.35), Style.normalBorderWidth)
            }

            TextField {
              id: colorField
              width: parent.width - Style.space(28) - applyColor.width - Style.space(12)
              anchors.verticalCenter: parent.verticalCenter
              placeholderText: "RRGGBB"
              foreground: root.fg
              font.family: root.fontFamily
              Keys.onPressed: function(event) {
                if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
                  root.applyColorField()
                  event.accepted = true
                } else if (event.key === Qt.Key_Escape) {
                  root.refocusPanel()
                  event.accepted = true
                }
              }
            }

            Button {
              id: applyColor
              iconText: "󰸌"
              text: "Set"
              bordered: true
              enabled: root.rgbConnected
              anchors.verticalCenter: parent.verticalCenter
              foreground: root.fg
              fontFamily: root.fontFamily
              fontSize: Style.font.caption
              tooltipText: "Apply to the selected regions"
              onClicked: root.applyColorField()
            }
          }

          ColorPicker {
            width: parent.width
            hex: colorField.text
            swatches: root.service ? root.service.themeSwatches : []
            themeEnabled: root.rgbConnected
            fg: root.fg
            accent: root.accent
            fontFamily: root.fontFamily
            onPicked: function(nextHex) { colorField.text = nextHex }
            onThemeColorRequested: if (root.service) root.service.applyThemeColor()
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

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            text: Model.EFFECTS_NOTE
            color: root.dim
            font.family: root.fontFamily
            font.pixelSize: Style.font.caption
          }
        }

        Column {
          id: powerTab
          width: parent.width
          spacing: Style.space(10)
          visible: root.tab === "power" && !root.settingsOpen

          PanelSectionHeader { text: "THERMAL MODE"; foreground: root.fg; fontFamily: root.fontFamily }

          Flow {
            width: parent.width
            spacing: Style.space(4)

            Repeater {
              model: root.service ? root.service.profileChoices : []

              Button {
                required property var modelData
                required property int index
                text: Model.profileLabel(modelData, root.hw.profile.gmodeForced)
                iconText: Model.profileGlyph(modelData)
                tooltipText: index < 4 ? "Press " + (index + 1) : ""
                selected: root.hw.profile.current === modelData
                enabled: root.available && root.hw.profile.writable
                bordered: true
                foreground: root.fg
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                onClicked: if (root.service) root.service.setProfile(modelData)
              }
            }
          }

          PanelSectionHeader { text: "READOUTS"; foreground: root.fg; fontFamily: root.fontFamily; topPadding: Style.space(4) }

          Column {
            width: parent.width
            spacing: Style.space(6)

            ReadoutRow {
              width: parent.width
              label: "CPU temperature"
              value: Model.formatTemp(root.cpuTemp, true)
              alert: root.cpuTemp !== null && root.cpuTemp >= root.hotTemp
              muted: root.stale
              fg: root.fg
              dim: root.dim
              alertColor: root.urgentColor
              fontFamily: root.fontFamily
            }

            ReadoutRow {
              width: parent.width
              label: "CPU clock"
              value: Model.formatClock(Model.cpuClock(root.hw))
              muted: true
              fg: root.fg
              dim: root.dim
              fontFamily: root.fontFamily
            }

            ReadoutRow {
              width: parent.width
              label: "GPU temperature"
              value: Model.formatTemp(root.gpuTemp, true)
              alert: root.gpuTemp !== null && root.gpuTemp >= root.hotTemp
              muted: root.stale
              fg: root.fg
              dim: root.dim
              alertColor: root.urgentColor
              fontFamily: root.fontFamily
            }

            ReadoutRow {
              width: parent.width
              label: "GPU power draw"
              value: Model.formatWatts(root.hw.gpu.draw, 1)
              muted: !root.hw.gpu.available
              fg: root.fg
              dim: root.dim
              fontFamily: root.fontFamily
            }

            ReadoutRow {
              width: parent.width
              label: "GPU power limit"
              value: Model.formatWatts(root.hw.gpu.limit)
              note: root.hw.gpu.available ? Model.PL_LOCKED_NOTE + ", default " + Model.formatWatts(root.hw.gpu.defaultLimit) : ""
              muted: true
              fg: root.fg
              dim: root.dim
              fontFamily: root.fontFamily
            }
          }

          PanelSectionHeader { text: "POWER LIMITS"; foreground: root.fg; fontFamily: root.fontFamily; topPadding: Style.space(4) }

          Text {
            width: parent.width
            wrapMode: Text.Wrap
            visible: !root.hw.power.available
            text: "The RAPL interface is not readable on this machine"
            color: root.dim
            font.family: root.fontFamily
            font.pixelSize: Style.font.caption
          }

          Column {
            width: parent.width
            spacing: Style.space(8)
            visible: root.hw.power.available

            Repeater {
              model: root.constraints

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

          Toggle {
            width: parent.width
            label: "Intel turbo"
            description: root.hw.turbo.available ? "Lets the cores boost above base clock" : "no_turbo is not readable on this machine"
            checked: root.hw.turbo.enabled
            enabled: root.hw.turbo.available && root.available
            foreground: root.fg
            fontFamily: root.fontFamily
            titleSize: Style.font.body
            onClicked: if (root.service) root.service.toggleTurbo()
          }
        }

        Column {
          id: settingsColumn
          width: parent.width
          spacing: Style.space(8)
          visible: root.settingsOpen

          PanelSectionHeader { text: "PLUGIN SETTINGS"; foreground: root.fg; fontFamily: root.fontFamily }

          Text {
            textFormat: Text.PlainText
            text: "Seconds between hardware polls"
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

          Text {
            textFormat: Text.PlainText
            text: "Temperature that turns the bar widget hot (C)"
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

          Text {
            textFormat: Text.PlainText
            text: "What the bar widget shows"
            color: root.dim
            font.family: root.fontFamily
            font.pixelSize: Style.font.caption
          }

          Row {
            spacing: Style.space(4)

            Repeater {
              model: [{ v: "icon", l: "Icon only" }, { v: "temp", l: "Icon and temperature" }, { v: "full", l: "Profile, rpm and temperature" }]

              Button {
                required property var modelData
                text: modelData.l
                selected: root.dDisplay === modelData.v
                bordered: true
                focusable: true
                foreground: root.fg
                fontFamily: root.fontFamily
                fontSize: Style.font.caption
                onClicked: root.dDisplay = modelData.v
              }
            }
          }

          Toggle {
            width: parent.width
            label: "Follow the Omarchy theme for RGB"
            description: "Repaint the keyboard when the theme colour changes"
            checked: root.dThemeRgb
            foreground: root.fg
            fontFamily: root.fontFamily
            titleSize: Style.font.body
            onClicked: root.dThemeRgb = !root.dThemeRgb
          }

          PanelSeparator { foreground: root.fg }

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
                focusable: true
                foreground: root.fg
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
          color: root.service && root.service.actionError ? root.urgentColor : root.dim
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
}
