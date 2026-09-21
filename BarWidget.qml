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

  readonly property int hotTemp: Model.clampHot(setting("hotTemp", service ? service.hotTemp : 90))
  readonly property var items: Model.parseBarItems(setting("display", service ? service.barItems : "cpu"))
  readonly property var view: Model.barView(status, health, items, hotTemp)

  readonly property string viewState: view.state
  readonly property bool warning: view.tone === "warn"
  readonly property bool muted: view.tone === "muted"
  readonly property bool unavailable: viewState === "missing" || viewState === "error"
  readonly property bool hasText: {
    var segs = view.segments
    for (var i = 0; i < segs.length; i++) if (segs[i].text !== "") return true
    return false
  }
  readonly property bool wide: view.segments.length > 1 || hasText

  function segmentColor(seg) {
    if (seg.alert) return root.bar ? root.bar.urgent : Color.urgent
    if (root.unavailable || root.muted) return Qt.darker(button.foreground, 1.6)
    if (seg.id === "logo" && service && service.curveActive) return Color.accent
    return button.foreground
  }

  readonly property string tooltip: {
    var parts = []
    parts.push(Model.barTooltip(status, health, service ? service.lastResult : null))
    if (service && service.rgbConnected && !service.lightsOn) parts.push("Lights off")
    parts.push("Left click panel · right click mode · middle click lights · scroll brightness")
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
    horizontalMargin: root.wide && !root.vertical ? 6 : 0
    fixedWidth: root.vertical ? -1 : (root.wide ? content.implicitWidth + Style.spaceReal(12) : Style.bar.iconSlot)
    fixedHeight: root.vertical ? (root.wide ? content.implicitHeight + Style.spaceReal(10) : Style.bar.iconSlot) : -1

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
      columns: root.vertical ? 1 : root.view.segments.length
      columnSpacing: Style.space(10)
      rowSpacing: Style.space(6)
      horizontalItemAlignment: Grid.AlignHCenter
      verticalItemAlignment: Grid.AlignVCenter

      Repeater {
        model: root.view.segments

        Grid {
          required property var modelData
          columns: root.vertical ? 1 : 2
          columnSpacing: Style.space(4)
          rowSpacing: Style.space(1)
          horizontalItemAlignment: Grid.AlignHCenter
          verticalItemAlignment: Grid.AlignVCenter

          Text {
            textFormat: Text.PlainText
            text: modelData.glyph
            color: root.segmentColor(modelData)
            font.family: button.fontFamily
            font.pixelSize: modelData.id === "logo" ? Style.bar.iconFont : Math.round(Style.bar.iconFont * 0.8)
            renderType: Text.NativeRendering
            Behavior on color { ColorAnimation { duration: 160 } }
          }

          Text {
            visible: modelData.text !== ""
            textFormat: Text.PlainText
            text: root.vertical ? modelData.text.replace("°", "") : modelData.text
            color: root.segmentColor(modelData)
            font.family: button.fontFamily
            font.pixelSize: Style.font.caption
            font.bold: modelData.alert
            renderType: Text.NativeRendering
            Behavior on color { ColorAnimation { duration: 160 } }
          }
        }
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
