import QtQuick
import qs.Commons
import qs.Ui

Item {
  id: row

  property string label: ""
  property string value: ""
  property string note: ""
  property bool muted: false
  property bool alert: false
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property color alertColor: Color.urgent
  property string fontFamily: Style.font.family

  implicitHeight: Math.max(labelText.implicitHeight, valueText.implicitHeight) + (noteText.text !== "" ? noteText.implicitHeight + Style.space(2) : 0)

  Text {
    id: labelText
    textFormat: Text.PlainText
    anchors.left: parent.left
    anchors.top: parent.top
    anchors.right: valueText.left
    anchors.rightMargin: Style.space(8)
    text: row.label
    color: row.dim
    font.family: row.fontFamily
    font.pixelSize: Style.font.bodySmall
    elide: Text.ElideRight
  }

  Text {
    id: valueText
    textFormat: Text.PlainText
    anchors.right: parent.right
    anchors.top: parent.top
    text: row.value
    color: row.alert ? row.alertColor : (row.muted ? row.dim : row.fg)
    font.family: row.fontFamily
    font.pixelSize: Style.font.bodySmall
    font.bold: row.alert
  }

  Text {
    id: noteText
    textFormat: Text.PlainText
    anchors.left: parent.left
    anchors.right: parent.right
    anchors.top: labelText.bottom
    anchors.topMargin: Style.space(2)
    visible: text !== ""
    text: row.note
    color: row.dim
    font.family: row.fontFamily
    font.pixelSize: Style.font.caption
    wrapMode: Text.Wrap
  }
}
