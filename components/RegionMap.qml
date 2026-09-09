import QtQuick
import QtQuick.Shapes
import qs.Commons
import qs.Ui
import "../Model.js" as Model

Item {
  id: root

  property var regions: []
  property var swatches: ({})
  property var poweredMap: ({})
  property var selection: []
  property int cursorIndex: -1
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property color accent: Color.accent
  property string fontFamily: Style.font.family

  signal picked(int rowIndex)
  signal toggleRequested(string regionId)
  signal identifyRequested(string regionId)
  signal powerRequested(string regionId)

  readonly property string alienGlyph: "\uDB82\uDC9A"
  readonly property real badgeSize: Style.space(44)
  readonly property real deckWidth: Math.min(root.width > 0 ? root.width * 0.86 : Style.space(300), Style.space(360))
  readonly property real deckHeight: Style.space(54)
  readonly property real lidHeight: Style.space(96)
  readonly property real ringStroke: Style.space(9)
  readonly property real ringAccentStroke: Style.space(3)
  readonly property var chipOrder: ["logo", "ring-top", "ring-bottom"]

  function regionList() { return Model.toList(root.regions) }

  function indexOfId(id) {
    var list = regionList()
    for (var i = 0; i < list.length; i++) if (list[i] && list[i].id === id) return i
    return -1
  }

  function regionById(id) {
    var list = regionList()
    for (var i = 0; i < list.length; i++) if (list[i] && list[i].id === id) return list[i]
    return null
  }

  function nameFor(id) {
    var r = regionById(id)
    return r ? r.name : ""
  }

  function isSelected(id) { return Model.toList(root.selection).indexOf(id) !== -1 }
  function isCursor(id) { return root.cursorIndex === indexOfId(id) }
  function isPowered(id) { return root.poweredMap ? root.poweredMap[id] !== false : true }
  function swatchHex(id) { return root.swatches ? String(root.swatches[id] || "") : "" }

  function glyphColor(id) {
    var hex = swatchHex(id)
    var on = isPowered(id)
    if (hex !== "") return Util.alpha(hex, on ? 1.0 : 0.4)
    return on ? root.fg : root.dim
  }

  function arcStrokeColor(id) {
    var hex = swatchHex(id)
    var on = isPowered(id)
    if (hex !== "") return Util.alpha(hex, on ? 0.95 : 0.35)
    return Util.alpha(root.fg, on ? 0.45 : 0.16)
  }

  function accentStrokeColor(id) {
    return isSelected(id) ? root.accent : "transparent"
  }

  function stadiumPath(w, h, inset, top) {
    var x0 = inset
    var x1 = w - inset
    var y0 = inset
    var y1 = h - inset
    var r = (y1 - y0) / 2
    var cy = (y0 + y1) / 2
    if (top) {
      return "M" + x0 + "," + cy + " A" + r + "," + r + " 0 0 1 " + (x0 + r) + "," + y0 +
        " L" + (x1 - r) + "," + y0 + " A" + r + "," + r + " 0 0 1 " + x1 + "," + cy
    }
    return "M" + x0 + "," + cy + " A" + r + "," + r + " 0 0 0 " + (x0 + r) + "," + y1 +
      " L" + (x1 - r) + "," + y1 + " A" + r + "," + r + " 0 0 0 " + x1 + "," + cy
  }

  function arcPath(cx, cy, r, top) {
    var x1 = cx - r
    var y1 = cy
    var x2 = cx + r
    var y2 = cy
    var sweep = top ? 1 : 0
    return "M" + x1 + "," + y1 + " A" + r + "," + r + " 0 0 " + sweep + " " + x2 + "," + y2
  }

  function activate(id) {
    root.picked(indexOfId(id))
    root.toggleRequested(id)
  }

  implicitWidth: parent ? parent.width : 0
  implicitHeight: layout.implicitHeight

  Column {
    id: layout
    width: parent.width
    spacing: Style.space(4)

    Item {
      id: lid
      anchors.horizontalCenter: parent.horizontalCenter
      width: root.deckWidth
      height: root.lidHeight

      Rectangle {
        anchors.fill: parent
        radius: Style.space(6)
        color: "transparent"
        border.color: root.isSelected("logo") ? root.accent : Util.alpha(root.fg, 0.28)
        border.width: root.isSelected("logo") ? Style.space(2) : Style.normalBorderWidth
      }

      Text {
        anchors.centerIn: parent
        textFormat: Text.PlainText
        text: root.alienGlyph
        color: root.glyphColor("logo")
        font.family: root.fontFamily
        font.pixelSize: Style.font.display
      }

      Text {
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.bottom
        anchors.bottomMargin: Style.space(4)
        textFormat: Text.PlainText
        text: root.nameFor("logo").toUpperCase()
        color: root.isPowered("logo") ? (root.isSelected("logo") ? root.accent : root.dim) : root.dim
        font.family: root.fontFamily
        font.pixelSize: Style.font.caption
      }

      MouseArea {
        anchors.fill: parent
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: root.activate("logo")
      }
    }

    Item {
      id: ringBox
      anchors.horizontalCenter: parent.horizontalCenter
      width: root.deckWidth
      height: root.deckHeight

      Row {
        anchors.centerIn: parent
        spacing: Style.space(7)

        Repeater {
          model: 6
          Rectangle {
            width: index === 0 || index === 5 ? Style.space(13) : Style.space(9)
            height: Style.space(7)
            radius: Style.space(2)
            color: Util.alpha(root.fg, 0.20)
          }
        }
      }

      Shape {
        anchors.fill: parent
        preferredRendererType: Shape.CurveRenderer

        ShapePath {
          strokeWidth: root.ringStroke
          strokeColor: root.arcStrokeColor("ring-top")
          fillColor: "transparent"
          capStyle: ShapePath.RoundCap
          PathSvg { path: root.stadiumPath(root.deckWidth, root.deckHeight, root.ringStroke / 2, true) }
        }

        ShapePath {
          strokeWidth: root.ringStroke
          strokeColor: root.arcStrokeColor("ring-bottom")
          fillColor: "transparent"
          capStyle: ShapePath.RoundCap
          PathSvg { path: root.stadiumPath(root.deckWidth, root.deckHeight, root.ringStroke / 2, false) }
        }

        ShapePath {
          strokeWidth: root.ringAccentStroke
          strokeColor: root.accentStrokeColor("ring-top")
          fillColor: "transparent"
          capStyle: ShapePath.RoundCap
          PathSvg { path: root.stadiumPath(root.deckWidth, root.deckHeight, root.ringStroke + root.ringAccentStroke, true) }
        }

        ShapePath {
          strokeWidth: root.ringAccentStroke
          strokeColor: root.accentStrokeColor("ring-bottom")
          fillColor: "transparent"
          capStyle: ShapePath.RoundCap
          PathSvg { path: root.stadiumPath(root.deckWidth, root.deckHeight, root.ringStroke + root.ringAccentStroke, false) }
        }
      }

      MouseArea {
        x: 0
        y: 0
        width: parent.width
        height: parent.height / 2
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: root.activate("ring-top")
      }

      MouseArea {
        x: 0
        y: parent.height / 2
        width: parent.width
        height: parent.height / 2
        hoverEnabled: true
        cursorShape: Qt.PointingHandCursor
        onClicked: root.activate("ring-bottom")
      }
    }

    Row {
      anchors.horizontalCenter: parent.horizontalCenter
      spacing: Style.space(10)

      Text {
        textFormat: Text.PlainText
        text: root.nameFor("ring-top").toUpperCase()
        color: root.isPowered("ring-top") ? (root.isSelected("ring-top") ? root.accent : root.fg) : root.dim
        font.family: root.fontFamily
        font.pixelSize: Style.font.caption
        font.bold: root.isSelected("ring-top")
      }

      Text {
        textFormat: Text.PlainText
        text: "/"
        color: root.dim
        font.family: root.fontFamily
        font.pixelSize: Style.font.caption
      }

      Text {
        textFormat: Text.PlainText
        text: root.nameFor("ring-bottom").toUpperCase()
        color: root.isPowered("ring-bottom") ? (root.isSelected("ring-bottom") ? root.accent : root.fg) : root.dim
        font.family: root.fontFamily
        font.pixelSize: Style.font.caption
        font.bold: root.isSelected("ring-bottom")
      }
    }

    Grid {
      id: controlGrid
      width: parent.width
      columns: 2
      columnSpacing: Style.space(6)
      rowSpacing: Style.space(6)
      topPadding: Style.space(6)

      Repeater {
        model: root.chipOrder

        CursorSurface {
          id: chip
          required property string modelData
          width: (controlGrid.width - controlGrid.columnSpacing) / 2
          implicitHeight: chipRow.implicitHeight + Style.space(8)
          radius: Style.cornerRadius
          foreground: root.fg
          accent: root.accent
          hasCursor: root.isCursor(chip.modelData)
          current: root.isSelected(chip.modelData)

          MouseArea {
            anchors.fill: parent
            hoverEnabled: true
            cursorShape: Qt.PointingHandCursor
            onClicked: root.activate(chip.modelData)
          }

          Row {
            id: chipRow
            anchors.left: parent.left
            anchors.right: chipActions.left
            anchors.verticalCenter: parent.verticalCenter
            anchors.leftMargin: Style.space(8)
            anchors.rightMargin: Style.space(6)
            spacing: Style.space(6)

            Text {
              anchors.verticalCenter: parent.verticalCenter
              textFormat: Text.PlainText
              visible: root.isSelected(chip.modelData)
              text: "󰄬"
              color: root.accent
              font.family: root.fontFamily
              font.pixelSize: Style.font.caption
            }

            Text {
              anchors.verticalCenter: parent.verticalCenter
              textFormat: Text.PlainText
              text: root.nameFor(chip.modelData)
              color: root.isPowered(chip.modelData) ? root.fg : root.dim
              font.family: root.fontFamily
              font.pixelSize: Style.font.bodySmall
              font.bold: root.isSelected(chip.modelData)
              elide: Text.ElideRight
            }
          }

          Row {
            id: chipActions
            anchors.right: parent.right
            anchors.verticalCenter: parent.verticalCenter
            anchors.rightMargin: Style.space(6)
            spacing: Style.space(4)

            PanelActionButton {
              iconText: "󱄄"
              tooltipText: "Light this region"
              foreground: root.fg
              fontFamily: root.fontFamily
              onClicked: root.identifyRequested(chip.modelData)
            }

            Button {
              text: root.isPowered(chip.modelData) ? "On" : "Off"
              selected: root.isPowered(chip.modelData)
              bordered: true
              foreground: root.fg
              fontFamily: root.fontFamily
              fontSize: Style.font.caption
              horizontalPadding: Style.space(6)
              verticalPadding: Style.space(2)
              onClicked: root.powerRequested(chip.modelData)
            }
          }
        }
      }
    }
  }
}
