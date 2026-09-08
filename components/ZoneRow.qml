import QtQuick
import qs.Commons
import qs.Ui
import "../Model.js" as Model

CursorSurface {
  id: row

  required property var zone
  required property int rowIndex
  property string swatch: ""
  property bool selected: false
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property string fontFamily: Style.font.family

  readonly property bool known: zone ? zone.known === true : false
  readonly property bool on: zone ? zone.enabled === true : false
  readonly property string title: zone ? zone.label : ""
  readonly property string hint: {
    if (!zone) return ""
    if (!known) return "Not named yet"
    var bits = ["Slot " + (zone.index + 1)]
    if (zone.ledCount > 0) bits.push(zone.ledCount + (zone.ledCount === 1 ? " led" : " leds"))
    if (!on) bits.push("hidden")
    return bits.join(" · ")
  }

  signal picked(int rowIndex)
  signal identifyRequested(int zoneIndex)
  signal toggleRequested(int zoneIndex)
  signal renameRequested(int zoneIndex)

  foreground: fg
  fill: Style.hoverFillFor(fg, accent)
  currentFill: Style.selectedFillFor(fg, accent)
  implicitHeight: content.implicitHeight + Style.space(12)

  MouseArea {
    id: mouse
    anchors.fill: parent
    hoverEnabled: true
    acceptedButtons: Qt.LeftButton | Qt.MiddleButton
    cursorShape: Qt.PointingHandCursor
    onClicked: function(m) {
      row.picked(row.rowIndex)
      if (m.button === Qt.MiddleButton) row.toggleRequested(row.zone.index)
      else row.identifyRequested(row.zone.index)
    }
  }

  Item {
    id: content
    anchors.left: parent.left
    anchors.right: parent.right
    anchors.verticalCenter: parent.verticalCenter
    anchors.leftMargin: Style.space(10)
    anchors.rightMargin: Style.space(10)
    implicitHeight: Math.max(labels.implicitHeight, dot.height, actions.implicitHeight)

    Rectangle {
      id: dot
      anchors.left: parent.left
      anchors.verticalCenter: parent.verticalCenter
      width: Style.space(14)
      height: Style.space(14)
      radius: width / 2
      color: row.swatch !== "" ? row.swatch : Util.alpha(row.fg, 0.2)
      border.width: 1
      border.color: Util.alpha(row.fg, row.on ? 0.6 : 0.25)
      opacity: row.on ? 1 : 0.45
    }

    Column {
      id: labels
      anchors.left: dot.right
      anchors.leftMargin: Style.space(10)
      anchors.right: actions.left
      anchors.rightMargin: Style.space(8)
      anchors.verticalCenter: parent.verticalCenter
      spacing: Style.space(2)

      Text {
        textFormat: Text.PlainText
        width: parent.width
        text: row.title
        color: row.known ? row.fg : row.dim
        font.family: row.fontFamily
        font.pixelSize: Style.font.bodySmall
        font.italic: !row.known
        elide: Text.ElideRight
      }

      Text {
        textFormat: Text.PlainText
        width: parent.width
        text: row.hint
        color: row.dim
        font.family: row.fontFamily
        font.pixelSize: Style.font.caption
        elide: Text.ElideRight
      }
    }

    Row {
      id: actions
      anchors.right: parent.right
      anchors.verticalCenter: parent.verticalCenter
      spacing: Style.space(2)

      PanelActionButton {
        iconText: "󱄄"
        tooltipText: "Light this slot"
        foreground: row.fg
        fontFamily: row.fontFamily
        onClicked: row.identifyRequested(row.zone.index)
      }

      PanelActionButton {
        iconText: "󰏫"
        tooltipText: row.known ? "Rename" : "Name this slot"
        foreground: row.fg
        fontFamily: row.fontFamily
        onClicked: row.renameRequested(row.zone.index)
      }

      PanelActionButton {
        visible: row.known
        iconText: row.on ? "󰈈" : "󰈉"
        tooltipText: row.on ? "Hide from the zone list" : "Show in the zone list"
        foreground: row.fg
        fontFamily: row.fontFamily
        hasCursor: row.on
        onClicked: row.toggleRequested(row.zone.index)
      }
    }
  }
}
