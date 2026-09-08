import QtQuick
import Quickshell
import qs.Commons
import qs.Ui
import "Model.js" as Model

BarWidget {
  id: root
  moduleName: "xyzlab.alienware"

  readonly property var service: bar && bar.shell ? bar.shell.serviceFor("xyzlab.alienware") : null
  readonly property var status: service ? service.status : null
  readonly property string health: service ? service.health : "missing"
  readonly property int hotTemp: service ? service.hotTemp : 90
  readonly property string display: service ? service.display : "temp"
  readonly property var barState: Model.barState(status, health, display, hotTemp)

  readonly property string viewState: barState.state
  readonly property string glyph: barState.glyph
  readonly property string valueText: barState.text
  readonly property bool warning: viewState === "warning"
  readonly property bool stopped: viewState === "stopped"
  readonly property bool muted: barState.tone === "muted"
  readonly property bool unavailable: viewState === "missing" || viewState === "error"
  readonly property bool textVisible: !root.vertical ? valueText !== "" : valueText !== "" && barState.view !== "expanded"
  readonly property string verticalText: {
    if (!root.vertical || !textVisible) return ""
    return valueText.replace("°", "")
  }

  readonly property color glyphColor: {
    if (warning) return root.bar ? root.bar.urgent : Color.urgent
    if (viewState === "error") return root.bar ? root.bar.urgent : Color.urgent
    if (muted || unavailable) return Qt.darker(button.foreground, 1.6)
    if (service && service.curveActive) return Color.accent
    return button.foreground
  }

  readonly property string tooltip: {
    var parts = []
    parts.push(Model.barTooltip(status, health, service ? service.lastResult : null))
    if (service && service.rgbConnected && !service.lightsOn) parts.push("Lights off")
    parts.push("Left click panel · right click profile · middle click lights · scroll brightness")
    return parts.join("\n")
  }

  function injectPanel() {
    var target = panelLoader.item
    if (!target) return
    if ("bar" in target) target.bar = root.bar
    if ("settings" in target) target.settings = root.settings
    if ("anchorItem" in target) target.anchorItem = button
    if ("hostWidget" in target) target.hostWidget = root
    if ("service" in target) target.service = root.service
  }

  function togglePanel() {
    if (panelLoader.item && panelLoader.item.toggle) panelLoader.item.toggle()
  }

  readonly property bool opened: panelLoader.item ? panelLoader.item.opened === true : false

  function open() {
    if (panelLoader.item && panelLoader.item.openFromHotkey) panelLoader.item.openFromHotkey()
  }

  function close() {
    if (panelLoader.item && panelLoader.item.close) panelLoader.item.close()
  }

  readonly property bool popoutSwitchClosing: panelLoader.item ? panelLoader.item.popoutSwitchClosing === true : false

  function closeForPopoutSwitch() {
    if (panelLoader.item) panelLoader.item.closeForPopoutSwitch()
  }

  implicitWidth: button.implicitWidth
  implicitHeight: button.implicitHeight

  onBarChanged: injectPanel()
  onSettingsChanged: injectPanel()
  onServiceChanged: injectPanel()

  Loader {
    id: panelLoader
    active: true
    source: Qt.resolvedUrl("Panel.qml")
    visible: false
    onLoaded: {
      root.injectPanel()
      Qt.callLater(root.injectPanel)
    }
  }

  WidgetButton {
    id: button
    anchors.fill: parent
    bar: root.bar
    labelVisible: false
    hasVisualContent: true
    dimmed: root.muted || root.unavailable
    tooltipText: root.tooltip
    horizontalMargin: root.textVisible && !root.vertical ? 6 : 0
    fixedWidth: root.vertical ? -1 : (root.textVisible ? content.implicitWidth + Style.spaceReal(12) : Style.bar.iconSlot)
    fixedHeight: root.vertical ? (root.textVisible ? content.implicitHeight + Style.spaceReal(10) : Style.bar.iconSlot) : -1

    onPressed: function(b) {
      if (b === Qt.RightButton) {
        if (root.service) root.service.cycleProfile(1)
      } else if (b === Qt.MiddleButton) {
        if (root.service) root.service.toggleLights()
      } else {
        root.togglePanel()
      }
    }

    Grid {
      id: content
      anchors.centerIn: parent
      columns: root.vertical ? 1 : 2
      rows: root.vertical ? 2 : 1
      columnSpacing: Style.space(4)
      rowSpacing: Style.space(1)
      horizontalItemAlignment: Grid.AlignHCenter
      verticalItemAlignment: Grid.AlignVCenter

      Text {
        id: glyphText
        textFormat: Text.PlainText
        text: root.glyph
        color: root.glyphColor
        font.family: button.fontFamily
        font.pixelSize: Style.bar.iconFont
        renderType: Text.NativeRendering
        Behavior on color { ColorAnimation { duration: 160 } }
      }

      Text {
        id: valueLabel
        visible: root.textVisible
        textFormat: Text.PlainText
        text: root.vertical ? root.verticalText : root.valueText
        color: root.glyphColor
        font.family: button.fontFamily
        font.pixelSize: Style.font.caption
        font.bold: root.warning
        renderType: Text.NativeRendering
        Behavior on color { ColorAnimation { duration: 160 } }
      }
    }

    MouseArea {
      anchors.fill: parent
      acceptedButtons: Qt.NoButton
      onWheel: function(wheel) {
        if (!root.service) return
        var step = wheel.angleDelta.y > 0 ? 1 : (wheel.angleDelta.y < 0 ? -1 : 0)
        if (step !== 0) root.service.nudgeBrightness(step)
      }
    }
  }
}
