import QtQuick
import qs.Commons
import qs.Ui
import "../Model.js" as Model

CursorSurface {
  id: row

  required property var region
  required property int rowIndex
  property string swatch: ""
  property bool selected: false
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property string fontFamily: Style.font.family

  readonly property string title: region ? region.name : ""
  readonly property string hint: {
    if (!region) return ""
    var count = region.ledCount
    return count + (count === 1 ? " led" : " leds")
  }

  signal picked(int rowIndex)
  signal toggleRequested(string regionId)
  signal identifyRequested(string regionId)

  foreground: fg
  fill: Style.hoverFillFor(fg, accent)
  currentFill: Style.selectedFillFor(fg, accent)
  current: row.selected
  implicitHeight: content.implicitHeight + Style.space(12)

  MouseArea {
    anchors.fill: parent
    hoverEnabled: true
    cursorShape: Qt.PointingHandCursor
    onClicked: {
      row.picked(row.rowIndex)
      row.toggleRequested(row.region.id)
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

    Item {
      id: checkSlot
      anchors.left: parent.left
      anchors.verticalCenter: parent.verticalCenter
      width: Style.space(18)
      height: Style.space(18)

      Text {
        anchors.centerIn: parent
        textFormat: Text.PlainText
        visible: row.selected
        text: "󰄬"
        color: row.accent
        font.family: row.fontFamily
        font.pixelSize: Style.font.body
      }
    }

    Rectangle {
      id: dot
      anchors.left: checkSlot.right
      anchors.leftMargin: Style.space(8)
      anchors.verticalCenter: parent.verticalCenter
      width: Style.space(14)
      height: Style.space(14)
      radius: width / 2
      color: row.swatch !== "" ? row.swatch : Util.alpha(row.fg, 0.2)
      border.width: 1
      border.color: Util.alpha(row.fg, 0.35)
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
        color: row.fg
        font.family: row.fontFamily
        font.pixelSize: Style.font.bodySmall
        font.bold: row.selected
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
        tooltipText: "Light this region"
        foreground: row.fg
        fontFamily: row.fontFamily
        onClicked: row.identifyRequested(row.region.id)
      }
    }
  }
}
