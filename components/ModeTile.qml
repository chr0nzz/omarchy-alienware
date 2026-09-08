import QtQuick
import qs.Commons
import qs.Ui

CursorSurface {
  id: tile

  required property string name
  required property string label
  property string glyph: ""
  property string caption: ""
  property bool selected: false
  property bool locked: false
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property string fontFamily: Style.font.family

  signal activated(string name)

  foreground: fg
  fill: Style.hoverFillFor(fg, accent)
  currentFill: Style.selectedFillFor(fg, accent)
  implicitHeight: body.implicitHeight + Style.space(14)

  MouseArea {
    anchors.fill: parent
    hoverEnabled: true
    cursorShape: tile.locked ? Qt.ForbiddenCursor : Qt.PointingHandCursor
    onClicked: if (!tile.locked) tile.activated(tile.name)
  }

  Rectangle {
    anchors.fill: parent
    radius: Style.space(6)
    color: tile.selected ? Style.selectedFillFor(tile.fg, tile.accent) : "transparent"
    border.width: tile.selected ? 1 : 0
    border.color: Util.alpha(tile.accent, 0.6)
  }

  Column {
    id: body
    anchors.left: parent.left
    anchors.right: parent.right
    anchors.verticalCenter: parent.verticalCenter
    anchors.leftMargin: Style.space(10)
    anchors.rightMargin: Style.space(10)
    spacing: Style.space(2)

    Text {
      textFormat: Text.PlainText
      visible: tile.glyph !== ""
      text: tile.glyph
      color: tile.selected ? tile.accent : (tile.locked ? tile.dim : tile.fg)
      font.family: tile.fontFamily
      font.pixelSize: Style.font.heading
    }

    Text {
      textFormat: Text.PlainText
      width: parent.width
      text: tile.label
      color: tile.selected ? tile.accent : (tile.locked ? tile.dim : tile.fg)
      font.family: tile.fontFamily
      font.pixelSize: Style.font.bodySmall
      font.bold: tile.selected
      elide: Text.ElideRight
    }

    Text {
      textFormat: Text.PlainText
      visible: tile.caption !== ""
      width: parent.width
      text: tile.caption
      color: tile.dim
      font.family: tile.fontFamily
      font.pixelSize: Style.font.caption
      wrapMode: Text.Wrap
    }
  }
}
