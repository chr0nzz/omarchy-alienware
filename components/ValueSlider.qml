import QtQuick
import qs.Commons
import qs.Ui

Item {
  id: slider

  property string label: ""
  property string valueText: ""
  property string note: ""
  property real value: 0
  property real from: 0
  property real to: 100
  property real stepSize: 1
  property bool editable: true
  property bool focused: false
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property color accentColor: Color.accent
  property color fillColor: Qt.darker(Color.accent, 1.3)
  property string fontFamily: Style.font.family

  readonly property real span: Math.max(0.0001, to - from)
  readonly property real fraction: Math.max(0, Math.min(1, (value - from) / span))

  signal moved(real next)

  implicitHeight: head.implicitHeight + track.height + Style.space(6) + (noteText.text !== "" ? noteText.implicitHeight + Style.space(3) : 0)

  function snap(raw) {
    var step = stepSize > 0 ? stepSize : 1
    var stepped = from + Math.round((raw - from) / step) * step
    return Math.max(from, Math.min(to, stepped))
  }

  function nudge(direction) {
    if (!editable) return
    moved(snap(value + direction * (stepSize > 0 ? stepSize : 1)))
  }

  Item {
    id: head
    anchors.left: parent.left
    anchors.right: parent.right
    anchors.top: parent.top
    implicitHeight: Math.max(labelText.implicitHeight, valueLabel.implicitHeight)

    Text {
      id: labelText
      textFormat: Text.PlainText
      anchors.left: parent.left
      anchors.verticalCenter: parent.verticalCenter
      text: slider.label
      color: slider.focused ? slider.fg : slider.dim
      font.family: slider.fontFamily
      font.pixelSize: Style.font.bodySmall
      font.bold: slider.focused
    }

    Text {
      id: valueLabel
      textFormat: Text.PlainText
      anchors.right: parent.right
      anchors.verticalCenter: parent.verticalCenter
      text: slider.valueText
      color: slider.editable ? slider.fg : slider.dim
      font.family: slider.fontFamily
      font.pixelSize: Style.font.bodySmall
    }
  }

  Rectangle {
    id: track
    anchors.left: parent.left
    anchors.right: parent.right
    anchors.top: head.bottom
    anchors.topMargin: Style.space(6)
    height: Style.space(6)
    radius: height / 2
    color: Util.alpha(slider.fg, 0.15)

    Rectangle {
      width: slider.fraction * parent.width
      height: parent.height
      radius: parent.radius
      color: slider.editable ? (slider.focused ? slider.accentColor : slider.fillColor) : Util.alpha(slider.fg, 0.35)
      Behavior on width { NumberAnimation { duration: 120 } }
    }

    Rectangle {
      visible: slider.editable
      width: Style.space(12)
      height: Style.space(12)
      radius: width / 2
      x: Math.max(0, Math.min(parent.width - width, slider.fraction * parent.width - width / 2))
      anchors.verticalCenter: parent.verticalCenter
      color: slider.focused ? slider.accentColor : slider.fg
    }

    MouseArea {
      anchors.fill: parent
      anchors.margins: -Style.space(6)
      enabled: slider.editable
      cursorShape: slider.editable ? Qt.PointingHandCursor : Qt.ArrowCursor
      onPressed: function(mouse) { slider.moved(slider.snap(slider.from + mouse.x / Math.max(1, width) * slider.span)) }
      onPositionChanged: function(mouse) {
        if (!pressed) return
        slider.moved(slider.snap(slider.from + mouse.x / Math.max(1, width) * slider.span))
      }
    }
  }

  Text {
    id: noteText
    textFormat: Text.PlainText
    anchors.left: parent.left
    anchors.right: parent.right
    anchors.top: track.bottom
    anchors.topMargin: Style.space(3)
    visible: text !== ""
    text: slider.note
    color: slider.dim
    font.family: slider.fontFamily
    font.pixelSize: Style.font.caption
    wrapMode: Text.Wrap
  }
}
