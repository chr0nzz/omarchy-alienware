import QtQuick
import qs.Commons
import qs.Ui
import "../Model.js" as Model

Column {
  id: root

  property string hex: ""
  property var swatches: []
  property bool themeEnabled: true
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property color accent: Color.accent
  property string fontFamily: Style.font.family

  signal picked(string hex)
  signal themeColorRequested()

  spacing: Style.space(8)

  property real hue: 0
  property real sat: 0
  property real val: 0
  property bool syncingFromHex: false

  readonly property bool showSwatches: root.swatches.length > 1 && !Model.paletteIsFlat(root.swatches)

  function syncFromHex(value) {
    var hsv = Model.hexToHsv(value)
    if (!hsv) return
    syncingFromHex = true
    hue = hsv.h
    sat = hsv.s
    val = hsv.v
    syncingFromHex = false
  }

  function commitFromHsv() {
    if (syncingFromHex) return
    picked(Model.hsvToHex(hue, sat, val))
  }

  onHexChanged: syncFromHex(hex)
  onHueChanged: commitFromHsv()
  onSatChanged: commitFromHsv()
  onValChanged: commitFromHsv()
  Component.onCompleted: syncFromHex(hex)

  Flow {
    width: parent.width
    spacing: Style.space(6)

    Button {
      text: "Theme"
      iconText: "󰏘"
      bordered: true
      enabled: root.themeEnabled
      foreground: root.fg
      accent: root.accent
      fontFamily: root.fontFamily
      fontSize: Style.font.caption
      horizontalPadding: Style.space(8)
      verticalPadding: Style.space(4)
      tooltipText: "Paint every region with the theme colour now"
      onClicked: root.themeColorRequested()
    }

    Repeater {
      model: root.showSwatches ? root.swatches : []

      Button {
        required property var modelData
        background: "#" + modelData.hex
        bordered: true
        foreground: root.fg
        accent: root.accent
        fontFamily: root.fontFamily
        tooltipText: modelData.label
        onClicked: root.picked(modelData.hex)
      }
    }
  }

  ValueSlider {
    width: parent.width
    label: "Hue"
    valueText: Math.round(root.hue) + "°"
    value: root.hue
    from: 0
    to: 360
    stepSize: 1
    fg: root.fg
    dim: root.dim
    accentColor: root.accent
    fillColor: root.accent
    fontFamily: root.fontFamily
    onMoved: function(next) { root.hue = next }
  }

  ValueSlider {
    width: parent.width
    label: "Saturation"
    valueText: Model.formatPercent(root.sat)
    value: root.sat
    from: 0
    to: 100
    stepSize: 1
    fg: root.fg
    dim: root.dim
    accentColor: root.accent
    fillColor: root.accent
    fontFamily: root.fontFamily
    onMoved: function(next) { root.sat = next }
  }

  ValueSlider {
    width: parent.width
    label: "Value"
    valueText: Model.formatPercent(root.val)
    value: root.val
    from: 0
    to: 100
    stepSize: 1
    note: "Sliders choose a colour. Set applies it to the selected regions."
    fg: root.fg
    dim: root.dim
    accentColor: root.accent
    fillColor: root.accent
    fontFamily: root.fontFamily
    onMoved: function(next) { root.val = next }
  }
}
