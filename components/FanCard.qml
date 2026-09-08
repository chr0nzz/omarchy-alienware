import QtQuick
import qs.Commons
import qs.Ui
import "../Model.js" as Model

Rectangle {
  id: card

  property var fan: null
  property var temp: null
  property bool stale: false
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property color accent: Color.accent
  property color alertColor: Color.urgent
  property int hotTemp: 90
  property string fontFamily: Style.font.family

  readonly property bool hasFan: !!fan
  readonly property int rpm: hasFan ? fan.rpm : 0
  readonly property int percent: hasFan ? fan.percent : 0
  readonly property int boost: hasFan ? fan.boost : 0
  readonly property bool stopped: hasFan && fan.rpm === 0
  readonly property bool hot: typeof temp === "number" && temp >= hotTemp
  readonly property string title: hasFan ? fan.label : "Fan"

  radius: Style.space(6)
  color: Style.hoverFillFor(fg, accent)
  implicitHeight: body.implicitHeight + Style.space(20)

  Column {
    id: body
    anchors.left: parent.left
    anchors.right: parent.right
    anchors.verticalCenter: parent.verticalCenter
    anchors.leftMargin: Style.space(12)
    anchors.rightMargin: Style.space(12)
    spacing: Style.space(6)

    Item {
      width: parent.width
      implicitHeight: Math.max(titleText.implicitHeight, tempText.implicitHeight)

      Text {
        id: titleText
        textFormat: Text.PlainText
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        text: card.title
        color: card.fg
        font.family: card.fontFamily
        font.pixelSize: Style.font.body
        font.bold: true
      }

      Text {
        id: tempText
        textFormat: Text.PlainText
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        text: Model.formatTemp(card.temp, true)
        color: card.hot ? card.alertColor : (card.stale ? card.dim : card.fg)
        font.family: card.fontFamily
        font.pixelSize: Style.font.body
        font.bold: card.hot
      }
    }

    Item {
      width: parent.width
      implicitHeight: Math.max(rpmText.implicitHeight, percentText.implicitHeight)

      Text {
        id: rpmText
        textFormat: Text.PlainText
        anchors.left: parent.left
        anchors.verticalCenter: parent.verticalCenter
        text: card.stopped ? "Idle, fan stopped" : Model.formatRpm(card.rpm)
        color: card.stopped || card.stale ? card.dim : card.fg
        font.family: card.fontFamily
        font.pixelSize: Style.font.bodySmall
      }

      Text {
        id: percentText
        textFormat: Text.PlainText
        anchors.right: parent.right
        anchors.verticalCenter: parent.verticalCenter
        text: Model.formatPercent(card.percent) + " of max"
        color: card.dim
        font.family: card.fontFamily
        font.pixelSize: Style.font.caption
      }
    }

    Rectangle {
      width: parent.width
      height: Style.space(5)
      radius: height / 2
      color: Util.alpha(card.fg, 0.15)

      Rectangle {
        width: Math.max(0, Math.min(1, card.percent / 100)) * parent.width
        height: parent.height
        radius: parent.radius
        color: card.hot ? card.alertColor : (card.stopped ? card.dim : card.accent)
        Behavior on width { NumberAnimation { duration: 200 } }
      }
    }

    Text {
      textFormat: Text.PlainText
      width: parent.width
      text: card.boost > 0
        ? "Boost +" + Model.boostPercent(card.boost) + "% on top of the firmware curve"
        : "No boost, firmware is in control"
      color: card.dim
      font.family: card.fontFamily
      font.pixelSize: Style.font.caption
      elide: Text.ElideRight
    }
  }
}
