import QtQuick
import qs.Commons
import qs.Ui
import "../Model.js" as Model

Rectangle {
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

  property color powerColor: "#8a8a96"
  property bool powerLit: false
  property bool powerSelected: false
  property bool powerLocked: false

  signal toggleRequested(string keyId)
  signal dragSelectRequested(var keyIds)
  signal powerClicked()

  readonly property color deckColor: "#0d0d11"
  readonly property color capColor: "#1a1a20"
  readonly property color capEdge: "#26262e"
  readonly property color unlitLegend: "#5a5a66"
  readonly property color offLegend: "#34343c"

  readonly property real pad: Style.space(8)
  readonly property real keyHeight: Style.space(30)
  readonly property real rowGap: Style.space(4)
  readonly property real keyGap: Style.space(4)
  readonly property real mediaWidth: Style.space(46)
  readonly property int rowCount: Model.toList(root.rows).length
  readonly property real innerWidth: root.width - root.pad * 2
  readonly property real topRowHeight: root.keyHeight + root.rowGap
  readonly property int topRowKeys: 16
  readonly property real topKeyWidth: (root.innerWidth - root.keyGap * (root.topRowKeys - 1)) / root.topRowKeys
  readonly property real endKeyX: (root.topKeyWidth + root.keyGap) * 14

  visible: root.present
  implicitHeight: root.present
    ? root.pad * 2 + root.topRowHeight + root.rowCount * root.keyHeight + Math.max(0, root.rowCount - 1) * root.rowGap
    : 0
  radius: Math.max(Style.cornerRadius, Style.space(4))
  color: root.deckColor
  border.color: Util.alpha(root.fg, 0.15)
  border.width: Style.normalBorderWidth

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

  component KeyCap: Rectangle {
    id: cap

    property var keyDef: null
    property string legend: ""
    property string swatch: ""
    property bool paintable: true
    property bool lit: true
    property bool picked: false
    property bool hot: capMouse.containsMouse
    property bool clickable: paintable

    signal pressedKey()
    signal enteredKey()
    signal releasedKey()

    readonly property bool glowing: paintable && lit && swatch !== ""
    readonly property color legendColor: {
      if (!paintable) return root.offLegend
      if (!lit) return root.offLegend
      return swatch !== "" ? swatch : root.unlitLegend
    }

    radius: Math.max(2, Style.cornerRadius / 2)
    color: {
      var base = cap.hot && cap.clickable ? Qt.lighter(root.capColor, 1.35) : root.capColor
      return cap.glowing ? Qt.tint(base, Util.alpha(cap.swatch, 0.16)) : base
    }
    border.width: cap.picked ? Math.max(2, Style.normalBorderWidth * 2) : Style.normalBorderWidth
    border.color: cap.picked ? root.accent : root.capEdge
    opacity: cap.paintable ? 1.0 : 0.55

    Text {
      anchors.centerIn: parent
      textFormat: Text.PlainText
      text: cap.legend
      color: cap.legendColor
      font.family: root.fontFamily
      font.pixelSize: Style.font.caption
      font.bold: cap.glowing || cap.picked
      fontSizeMode: Text.HorizontalFit
      minimumPixelSize: Math.max(6, Style.font.caption - 4)
      elide: Text.ElideRight
      width: parent.width - Style.space(4)
      horizontalAlignment: Text.AlignHCenter
      Behavior on color { ColorAnimation { duration: 140 } }
    }

    MouseArea {
      id: capMouse
      anchors.fill: parent
      hoverEnabled: true
      enabled: cap.clickable
      cursorShape: cap.clickable ? Qt.PointingHandCursor : Qt.ArrowCursor
      onPressed: cap.pressedKey()
      onEntered: { if (pressed) cap.enteredKey() }
      onReleased: cap.releasedKey()
    }
  }

  Item {
    id: inner
    x: root.pad
    y: root.pad
    width: root.innerWidth
    height: root.height - root.pad * 2

    KeyCap {
      id: powerCap
      x: root.endKeyX
      y: 0
      width: root.topKeyWidth
      height: root.keyHeight
      legend: "⏻"
      swatch: root.powerLit ? root.powerColor : ""
      lit: root.powerLit
      picked: root.powerSelected
      clickable: true
      onReleasedKey: root.powerClicked()
    }

    Text {
      x: 0
      y: 0
      height: root.keyHeight
      verticalAlignment: Text.AlignVCenter
      textFormat: Text.PlainText
      text: "ALIENWARE"
      color: root.unlitLegend
      font.family: root.fontFamily
      font.pixelSize: Style.font.caption
      font.letterSpacing: 3
      font.bold: true
      opacity: 0.6
    }

    Column {
      id: rowsColumn
      anchors.left: parent.left
      anchors.top: parent.top
      anchors.topMargin: root.topRowHeight
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

            KeyCap {
              required property var modelData
              keyDef: modelData
              width: Math.max(Style.space(14), (krow.width - krow.spacing * (krow.modelData.length - 1)) * modelData.w / krow.weight)
              height: krow.height
              legend: modelData.label
              paintable: Model.isKeyPaintable(modelData)
              swatch: root.swatchFor(modelData)
              lit: root.isKeyOn(modelData)
              picked: root.isSelected(modelData.id)
              onPressedKey: root.beginDrag(modelData.id)
              onEnteredKey: root.extendDrag(modelData.id)
              onReleasedKey: root.endDrag()
            }
          }
        }
      }
    }

    Column {
      id: mediaColumnItem
      x: rowsColumn.width - root.mediaWidth
      y: root.topRowHeight + root.keyHeight + root.rowGap
      width: root.mediaWidth
      spacing: root.rowGap

      Repeater {
        model: root.mediaColumn

        KeyCap {
          required property var modelData
          keyDef: modelData
          width: mediaColumnItem.width
          height: root.keyHeight
          legend: modelData.label
          paintable: Model.isKeyPaintable(modelData)
          swatch: root.swatchFor(modelData)
          lit: root.isKeyOn(modelData)
          picked: root.isSelected(modelData.id)
          onPressedKey: root.beginDrag(modelData.id)
          onEnteredKey: root.extendDrag(modelData.id)
          onReleasedKey: root.endDrag()
        }
      }
    }
  }
}
