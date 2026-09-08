import QtQuick
import qs.Commons
import qs.Ui
import "../Model.js" as Model

Item {
  id: root

  property bool present: true
  property var rows: Model.KEYBOARD_ROWS
  property var mediaColumn: Model.KEYBOARD_MEDIA_COLUMN
  property var keyColors: ({})
  property var keyOn: ({})
  property var selection: []
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property color accent: Color.accent
  property string fontFamily: Style.font.family

  signal toggleRequested(string keyId)
  signal dragSelectRequested(var keyIds)

  readonly property real keyHeight: Style.space(28)
  readonly property real rowGap: Style.space(4)
  readonly property real keyGap: Style.space(3)
  readonly property real mediaWidth: Style.space(46)
  readonly property int rowCount: Model.toList(root.rows).length

  visible: root.present
  implicitWidth: parent ? parent.width : 0
  implicitHeight: root.present ? root.rowCount * root.keyHeight + Math.max(0, root.rowCount - 1) * root.rowGap : 0

  function isSelected(id) {
    return Model.toList(root.selection).indexOf(id) !== -1
  }

  function swatchFor(key) {
    if (!Model.isKeyPaintable(key)) return ""
    return Model.hexPreview(root.keyColors[key.id])
  }

  function isKeyOn(key) {
    return !Model.isKeyPaintable(key) || root.keyOn[key.id] !== false
  }

  property bool dragActive: false
  property var dragRunIds: []

  function beginDrag(id) {
    root.dragActive = true
    root.dragRunIds = id ? [id] : []
  }

  function extendDrag(id) {
    if (!root.dragActive || !id) return
    var next = root.dragRunIds.slice()
    if (next.indexOf(id) === -1) next.push(id)
    root.dragRunIds = next
  }

  function endDrag() {
    if (!root.dragActive) return
    root.dragActive = false
    var run = root.dragRunIds.slice()
    root.dragRunIds = []
    if (run.length > 1) root.dragSelectRequested(run)
    else if (run.length === 1) root.toggleRequested(run[0])
  }

  Column {
    id: rowsColumn
    anchors.left: parent.left
    anchors.top: parent.top
    width: parent.width
    spacing: root.rowGap

    Repeater {
      model: root.rows

      Row {
        id: krow
        required property var modelData
        required property int index
        width: krow.spansFullWidth ? rowsColumn.width : rowsColumn.width - root.mediaWidth - root.rowGap
        height: root.keyHeight
        readonly property bool spansFullWidth: krow.index === 0 || krow.index === root.rows.length - 1
        spacing: root.keyGap
        readonly property real weight: Model.keyboardRowWeight(krow.modelData)

        Repeater {
          model: krow.modelData

          Rectangle {
            id: keyTile
            required property var modelData
            readonly property var keyDef: keyTile.modelData
            readonly property bool paintable: Model.isKeyPaintable(keyTile.keyDef)
            readonly property bool selected: root.isSelected(keyTile.keyDef.id)
            readonly property string swatch: root.swatchFor(keyTile.keyDef)
            readonly property bool keyIsOn: root.isKeyOn(keyTile.keyDef)

            width: Math.max(Style.space(14), (krow.width - krow.spacing * (krow.modelData.length - 1)) * keyTile.keyDef.w / krow.weight)
            height: krow.height
            radius: Style.cornerRadius
            color: {
              if (!keyTile.paintable) return Util.alpha(root.fg, 0.05)
              if (keyTile.swatch !== "") return Util.alpha(keyTile.swatch, keyTile.keyIsOn ? 1.0 : 0.35)
              return Util.alpha(root.fg, keyTile.keyIsOn ? 0.12 : 0.05)
            }
            border.width: keyTile.selected ? Style.normalBorderWidth * 2 : Style.normalBorderWidth
            border.color: keyTile.selected ? root.accent : Util.alpha(root.fg, keyTile.paintable ? 0.35 : 0.15)
            opacity: keyTile.paintable ? 1.0 : 0.45

            Text {
              anchors.centerIn: parent
              textFormat: Text.PlainText
              text: keyTile.keyDef.label
              color: keyTile.paintable && keyTile.keyIsOn ? root.fg : root.dim
              font.family: root.fontFamily
              font.pixelSize: Style.font.caption
              font.bold: keyTile.selected
              fontSizeMode: Text.HorizontalFit
              minimumPixelSize: Math.max(6, Style.font.caption - 4)
              elide: Text.ElideRight
              width: parent.width - Style.space(2)
              horizontalAlignment: Text.AlignHCenter
            }

            MouseArea {
              anchors.fill: parent
              hoverEnabled: true
              enabled: keyTile.paintable
              cursorShape: keyTile.paintable ? Qt.PointingHandCursor : Qt.ArrowCursor
              onPressed: root.beginDrag(keyTile.keyDef.id)
              onEntered: if (pressed) root.extendDrag(keyTile.keyDef.id)
              onReleased: root.endDrag()
            }
          }
        }
      }
    }
  }

  Column {
    id: mediaColumnItem
    x: rowsColumn.width - root.mediaWidth
    y: root.keyHeight + root.rowGap
    width: root.mediaWidth
    spacing: root.rowGap

    Repeater {
      model: root.mediaColumn

      Rectangle {
        id: mediaTile
        required property var modelData
        readonly property var keyDef: mediaTile.modelData
        readonly property bool paintable: Model.isKeyPaintable(mediaTile.keyDef)
        readonly property bool selected: root.isSelected(mediaTile.keyDef.id)
        readonly property string swatch: root.swatchFor(mediaTile.keyDef)
        readonly property bool keyIsOn: root.isKeyOn(mediaTile.keyDef)

        width: mediaColumnItem.width
        height: root.keyHeight
        radius: Style.cornerRadius
        color: {
          if (!mediaTile.paintable) return Util.alpha(root.fg, 0.05)
          if (mediaTile.swatch !== "") return Util.alpha(mediaTile.swatch, mediaTile.keyIsOn ? 1.0 : 0.35)
          return Util.alpha(root.fg, mediaTile.keyIsOn ? 0.12 : 0.05)
        }
        border.width: mediaTile.selected ? Style.normalBorderWidth * 2 : Style.normalBorderWidth
        border.color: mediaTile.selected ? root.accent : Util.alpha(root.fg, mediaTile.paintable ? 0.35 : 0.15)
        opacity: mediaTile.paintable ? 1.0 : 0.45

        Text {
          anchors.centerIn: parent
          textFormat: Text.PlainText
          text: mediaTile.keyDef.label
          color: mediaTile.paintable && mediaTile.keyIsOn ? root.fg : root.dim
          font.family: root.fontFamily
          font.pixelSize: Style.font.caption
          font.bold: mediaTile.selected
          fontSizeMode: Text.HorizontalFit
          minimumPixelSize: Math.max(6, Style.font.caption - 4)
          elide: Text.ElideRight
          width: parent.width - Style.space(2)
          horizontalAlignment: Text.AlignHCenter
        }

        MouseArea {
          anchors.fill: parent
          hoverEnabled: true
          enabled: mediaTile.paintable
          cursorShape: mediaTile.paintable ? Qt.PointingHandCursor : Qt.ArrowCursor
          onPressed: root.beginDrag(mediaTile.keyDef.id)
          onEntered: if (pressed) root.extendDrag(mediaTile.keyDef.id)
          onReleased: root.endDrag()
        }
      }
    }
  }
}
