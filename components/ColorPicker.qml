import QtQuick
import qs.Commons
import qs.Ui
import "../Model.js" as Model

Column {
  id: root

  property string hex: ""
  property var swatches: []
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property color accent: Color.accent
  property string fontFamily: Style.font.family

  signal picked(string hex)

  spacing: Style.space(8)

  property real hue: 0
  property real sat: 0
  property real val: 0
  property bool syncingFromHex: false

  readonly property color previewColor: Model.validHex(hex) ? Model.hexPreview(hex) : Util.alpha(root.fg, 0.15)

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

  Text {
    textFormat: Text.PlainText
    visible: root.swatches.length > 0
    text: "Theme swatches"
    color: root.dim
    font.family: root.fontFamily
    font.pixelSize: Style.font.caption
  }

  Flow {
    width: parent.width
    spacing: Style.space(6)
    visible: root.swatches.length > 0

    Repeater {
      model: root.swatches

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

  Row {
    width: parent.width
    spacing: Style.space(10)

    BorderSurface {
      id: preview
      width: Style.space(36)
      height: Style.space(36)
      radius: Style.cornerRadius
      color: root.previewColor
      borderSpec: Border.flat(Util.alpha(root.fg, 0.35), Style.normalBorderWidth)
    }

    Column {
      width: parent.width - preview.width - parent.spacing
      spacing: Style.space(6)

      Item {
        width: parent.width
        implicitHeight: Math.max(hueLabel.implicitHeight, hueValue.implicitHeight)

        Text {
          id: hueLabel
          textFormat: Text.PlainText
          anchors.left: parent.left
          text: "Hue"
          color: root.dim
          font.family: root.fontFamily
          font.pixelSize: Style.font.bodySmall
        }

        Text {
          id: hueValue
          textFormat: Text.PlainText
          anchors.right: parent.right
          text: Math.round(root.hue) + "°"
          color: root.fg
          font.family: root.fontFamily
          font.pixelSize: Style.font.bodySmall
        }
      }

      PanelSlider {
        width: parent.width
        minimum: 0
        maximum: 360
        step: 1
        integer: true
        value: root.hue
        trackColor: Util.alpha(root.fg, 0.15)
        fillColor: root.accent
        knobColor: root.fg
        onMoved: function(next) { root.hue = next }
      }

      Item {
        width: parent.width
        implicitHeight: Math.max(satLabel.implicitHeight, satValue.implicitHeight)

        Text {
          id: satLabel
          textFormat: Text.PlainText
          anchors.left: parent.left
          text: "Saturation"
          color: root.dim
          font.family: root.fontFamily
          font.pixelSize: Style.font.bodySmall
        }

        Text {
          id: satValue
          textFormat: Text.PlainText
          anchors.right: parent.right
          text: Model.formatPercent(root.sat)
          color: root.fg
          font.family: root.fontFamily
          font.pixelSize: Style.font.bodySmall
        }
      }

      PanelSlider {
        width: parent.width
        minimum: 0
        maximum: 100
        step: 1
        integer: true
        value: root.sat
        trackColor: Util.alpha(root.fg, 0.15)
        fillColor: root.accent
        knobColor: root.fg
        onMoved: function(next) { root.sat = next }
      }

      Item {
        width: parent.width
        implicitHeight: Math.max(valLabel.implicitHeight, valValue.implicitHeight)

        Text {
          id: valLabel
          textFormat: Text.PlainText
          anchors.left: parent.left
          text: "Value"
          color: root.dim
          font.family: root.fontFamily
          font.pixelSize: Style.font.bodySmall
        }

        Text {
          id: valValue
          textFormat: Text.PlainText
          anchors.right: parent.right
          text: Model.formatPercent(root.val)
          color: root.fg
          font.family: root.fontFamily
          font.pixelSize: Style.font.bodySmall
        }
      }

      PanelSlider {
        width: parent.width
        minimum: 0
        maximum: 100
        step: 1
        integer: true
        value: root.val
        trackColor: Util.alpha(root.fg, 0.15)
        fillColor: root.accent
        knobColor: root.fg
        onMoved: function(next) { root.val = next }
      }
    }
  }

  Text {
    width: parent.width
    textFormat: Text.PlainText
    wrapMode: Text.Wrap
    text: "This picks a colour. Press Set above to apply it to the selected regions."
    color: root.dim
    font.family: root.fontFamily
    font.pixelSize: Style.font.caption
    font.italic: true
  }
}
