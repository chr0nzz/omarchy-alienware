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
  property bool powered: true
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
  signal powerRequested(string regionId)

  foreground: fg
  fill: Style.hoverFillFor(fg, accent)
  currentFill: Style.selectedFillFor(fg, accent)
  current: row.selected
  implicitHeight: content.implicitHeight + Style.space(10)

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
    implicitHeight: Math.max(chip.height, labels.implicitHeight, actions.implicitHeight)

    Item {
      id: checkSlot
      anchors.left: parent.left
      anchors.verticalCenter: parent.verticalCenter
      width: Style.space(16)
      height: Style.space(16)

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

    BorderSurface {
      id: chip
      anchors.left: checkSlot.right
      anchors.leftMargin: Style.space(8)
      anchors.verticalCenter: parent.verticalCenter
      width: Style.space(34)
      height: Style.space(34)
      radius: Style.cornerRadius
      color: row.swatch !== ""
        ? Util.alpha(row.swatch, row.powered ? 1.0 : 0.35)
        : Util.alpha(row.fg, row.powered ? 0.15 : 0.06)
      borderSpec: Border.flat(Util.alpha(row.fg, 0.35), Style.normalBorderWidth)

      Text {
        anchors.centerIn: parent
        visible: !row.powered
        textFormat: Text.PlainText
        text: "OFF"
        color: row.fg
        font.family: row.fontFamily
        font.pixelSize: Style.font.caption
        font.bold: true
      }
    }

    Column {
      id: labels
      anchors.left: chip.right
      anchors.leftMargin: Style.space(10)
      anchors.right: actions.left
      anchors.rightMargin: Style.space(8)
      anchors.verticalCenter: parent.verticalCenter
      spacing: Style.space(2)

      Text {
        textFormat: Text.PlainText
        width: parent.width
        text: row.title
        color: row.powered ? row.fg : row.dim
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
      spacing: Style.space(4)

      PanelActionButton {
        iconText: "󱄄"
        tooltipText: "Light this region (i)"
        foreground: row.fg
        fontFamily: row.fontFamily
        onClicked: row.identifyRequested(row.region.id)
      }

      Button {
        text: row.powered ? "On" : "Off"
        selected: row.powered
        bordered: true
        foreground: row.fg
        fontFamily: row.fontFamily
        fontSize: Style.font.caption
        horizontalPadding: Style.space(8)
        verticalPadding: Style.space(3)
        tooltipText: row.powered ? "Turn this region off (p)" : "Turn this region on (p)"
        onClicked: row.powerRequested(row.region.id)
      }
    }
  }
}
