import QtQuick
import QtQuick.Shapes
import qs.Commons
import qs.Ui
import "../Model.js" as Model

Rectangle {
  id: root

  property string hex: ""
  property var swatches: []
  property bool themeEnabled: true
  property bool canApply: false
  property string targetText: ""
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property color accent: Color.accent
  property string fontFamily: Style.font.family
  readonly property alias field: hexField

  signal picked(string hex)
  signal applyRequested()
  signal themeColorRequested()
  signal fieldEscaped()

  property real hue: 0
  property real sat: 0
  property real val: 100
  property bool syncingFromHex: false
  property string lastPicked: ""

  readonly property bool valid: Model.validHex(root.hex)
  readonly property color previewColor: root.valid ? Model.hexPreview(root.hex) : Util.alpha(root.fg, 0.08)
  readonly property var themeChips: {
    var list = Model.toList(root.swatches)
    return list.length > 1 && !Model.paletteIsFlat(list) ? list : []
  }

  radius: Math.max(Style.cornerRadius, Style.space(4))
  color: Style.normalFillFor(root.fg, root.accent)
  border.color: Util.alpha(root.fg, 0.15)
  border.width: Style.normalBorderWidth
  implicitHeight: content.implicitHeight + Style.space(24)

  function syncFromHex(value) {
    var hsv = Model.hexToHsv(value)
    if (!hsv) return
    syncingFromHex = true
    hue = hsv.h
    sat = hsv.s
    val = hsv.v
    syncingFromHex = false
  }

  function commitFromHsv() {
    if (syncingFromHex) return
    lastPicked = Model.hsvToHex(hue, sat, val)
    picked(lastPicked)
  }

  onHexChanged: {
    if (hexField.text.toUpperCase() !== String(root.hex).toUpperCase()) hexField.text = root.hex
    if (Model.normalizeHex(hex) !== root.lastPicked) syncFromHex(hex)
  }
  onHueChanged: commitFromHsv()
  onSatChanged: commitFromHsv()
  onValChanged: commitFromHsv()
  Component.onCompleted: { hexField.text = root.hex; syncFromHex(hex) }

  component Chip: Rectangle {
    id: chip
    property string chipHex: ""
    property string label: ""
    readonly property bool current: Model.normalizeHex(chip.chipHex) === Model.normalizeHex(root.hex)
    width: Style.space(22)
    height: Style.space(22)
    radius: Math.max(2, Style.cornerRadius / 2)
    color: Model.hexPreview(chip.chipHex)
    border.width: chip.current ? Math.max(2, Style.normalBorderWidth * 2) : Style.normalBorderWidth
    border.color: chip.current ? root.fg : (chipMouse.containsMouse ? Util.alpha(root.fg, 0.7) : Util.alpha(root.fg, 0.25))

    MouseArea {
      id: chipMouse
      anchors.fill: parent
      hoverEnabled: true
      cursorShape: Qt.PointingHandCursor
      onClicked: root.picked(Model.normalizeHex(chip.chipHex))
    }
  }

  Row {
    id: content
    x: Style.space(12)
    y: Style.space(12)
    width: root.width - Style.space(24)
    spacing: Style.space(16)

    Item {
      id: wheel
      width: Style.space(132)
      height: width

      readonly property real radius: width / 2
      readonly property real centerX: width / 2
      readonly property real centerY: height / 2

      function applyPoint(px, py) {
        var hs = Model.pointToHueSat((px - centerX) / radius, (centerY - py) / radius)
        root.hue = hs.h
        root.sat = hs.s
        if (root.val < 15) root.val = 100
      }

      Shape {
        anchors.fill: parent
        preferredRendererType: Shape.CurveRenderer

        ShapePath {
          fillRule: ShapePath.WindingFill
          strokeWidth: 0
          fillGradient: ConicalGradient {
            centerX: wheel.centerX
            centerY: wheel.centerY
            angle: 0
            GradientStop { position: 0 / 360; color: Model.hexPreview(Model.hsvToHex(0, 100, 100)) }
            GradientStop { position: 60 / 360; color: Model.hexPreview(Model.hsvToHex(60, 100, 100)) }
            GradientStop { position: 120 / 360; color: Model.hexPreview(Model.hsvToHex(120, 100, 100)) }
            GradientStop { position: 180 / 360; color: Model.hexPreview(Model.hsvToHex(180, 100, 100)) }
            GradientStop { position: 240 / 360; color: Model.hexPreview(Model.hsvToHex(240, 100, 100)) }
            GradientStop { position: 300 / 360; color: Model.hexPreview(Model.hsvToHex(300, 100, 100)) }
            GradientStop { position: 360 / 360; color: Model.hexPreview(Model.hsvToHex(360, 100, 100)) }
          }
          startX: wheel.centerX + wheel.radius
          startY: wheel.centerY
          PathArc { x: wheel.centerX - wheel.radius; y: wheel.centerY; radiusX: wheel.radius; radiusY: wheel.radius }
          PathArc { x: wheel.centerX + wheel.radius; y: wheel.centerY; radiusX: wheel.radius; radiusY: wheel.radius }
        }

        ShapePath {
          fillRule: ShapePath.WindingFill
          strokeWidth: 0
          fillGradient: RadialGradient {
            centerX: wheel.centerX
            centerY: wheel.centerY
            centerRadius: wheel.radius
            focalX: wheel.centerX
            focalY: wheel.centerY
            focalRadius: 0
            GradientStop { position: 0.0; color: Qt.rgba(1, 1, 1, 1) }
            GradientStop { position: 1.0; color: Qt.rgba(1, 1, 1, 0) }
          }
          startX: wheel.centerX + wheel.radius
          startY: wheel.centerY
          PathArc { x: wheel.centerX - wheel.radius; y: wheel.centerY; radiusX: wheel.radius; radiusY: wheel.radius }
          PathArc { x: wheel.centerX + wheel.radius; y: wheel.centerY; radiusX: wheel.radius; radiusY: wheel.radius }
        }
      }

      Rectangle {
        anchors.fill: parent
        radius: wheel.radius
        color: "black"
        opacity: 1 - root.val / 100
      }

      Rectangle {
        anchors.fill: parent
        radius: wheel.radius
        color: "transparent"
        border.color: Util.alpha(root.fg, 0.25)
        border.width: Style.normalBorderWidth
      }

      Rectangle {
        readonly property var point: Model.hueSatToPoint(root.hue, root.sat)
        width: Style.space(16)
        height: width
        radius: width / 2
        color: root.previewColor
        border.color: "white"
        border.width: Math.max(2, Style.space(2))
        x: wheel.centerX + point.x * wheel.radius - width / 2
        y: wheel.centerY - point.y * wheel.radius - height / 2

        Rectangle {
          anchors.fill: parent
          anchors.margins: -1
          radius: width / 2
          color: "transparent"
          border.color: "black"
          border.width: 1
          opacity: 0.5
        }
      }

      MouseArea {
        anchors.fill: parent
        preventStealing: true
        cursorShape: Qt.CrossCursor
        onPressed: function(mouse) {
          var dx = (mouse.x - wheel.centerX) / wheel.radius
          var dy = (mouse.y - wheel.centerY) / wheel.radius
          if (dx * dx + dy * dy > 1.05) { mouse.accepted = false; return }
          wheel.applyPoint(mouse.x, mouse.y)
        }
        onPositionChanged: function(mouse) { if (pressed) wheel.applyPoint(mouse.x, mouse.y) }
      }
    }

    Item {
      id: valueBar
      width: Style.space(14)
      height: wheel.height

      function applyPoint(py) {
        root.val = Math.round((1 - Math.max(0, Math.min(1, py / Math.max(1, height)))) * 100)
      }

      Rectangle {
        anchors.fill: parent
        radius: width / 2
        border.color: Util.alpha(root.fg, 0.25)
        border.width: Style.normalBorderWidth
        gradient: Gradient {
          orientation: Gradient.Vertical
          GradientStop { position: 0.0; color: Model.hexPreview(Model.hsvToHex(root.hue, root.sat, 100)) }
          GradientStop { position: 1.0; color: "#000000" }
        }
      }

      Rectangle {
        width: parent.width + Style.space(6)
        height: Style.space(6)
        radius: height / 2
        x: -Style.space(3)
        y: Math.max(0, Math.min(valueBar.height - height, (1 - root.val / 100) * valueBar.height - height / 2))
        color: "white"
        border.color: "black"
        border.width: 1
      }

      MouseArea {
        anchors.fill: parent
        anchors.margins: -Style.space(4)
        preventStealing: true
        cursorShape: Qt.SizeVerCursor
        onPressed: function(mouse) { valueBar.applyPoint(mouse.y - Style.space(4)) }
        onPositionChanged: function(mouse) { if (pressed) valueBar.applyPoint(mouse.y - Style.space(4)) }
      }
    }

    Column {
      id: side
      width: content.width - wheel.width - valueBar.width - content.spacing * 2
      spacing: Style.space(12)

      Row {
        width: parent.width
        spacing: Style.space(12)

        Rectangle {
          id: preview
          width: Style.space(52)
          height: Style.space(52)
          radius: Math.max(Style.cornerRadius, Style.space(4))
          color: root.previewColor
          border.color: Util.alpha(root.fg, 0.3)
          border.width: Style.normalBorderWidth
          Behavior on color { ColorAnimation { duration: 120 } }
        }

        Column {
          width: parent.width - preview.width - parent.spacing
          anchors.verticalCenter: parent.verticalCenter
          spacing: Style.space(6)

          Row {
            spacing: Style.space(6)

            Text {
              anchors.verticalCenter: parent.verticalCenter
              text: "#"
              color: root.dim
              font.family: root.fontFamily
              font.pixelSize: Style.font.body
            }

            TextField {
              id: hexField
              width: Style.space(100)
              placeholderText: "RRGGBB"
              foreground: root.fg
              font.family: root.fontFamily
              onTextEdited: {
                var clean = Model.normalizeHex(text)
                if (clean) root.picked(clean)
              }
              Keys.onPressed: function(event) {
                if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
                  var clean = Model.normalizeHex(hexField.text)
                  if (clean) { root.picked(clean); root.applyRequested() }
                  event.accepted = true
                } else if (event.key === Qt.Key_Escape) {
                  root.fieldEscaped()
                  event.accepted = true
                }
              }
            }
          }

          Button {
            iconText: "󰸱"
            text: root.canApply ? "Apply to " + root.targetText : "Select keys or regions"
            bordered: true
            selected: root.canApply && root.valid
            enabled: root.canApply && root.valid
            foreground: root.fg
            accent: root.accent
            fontFamily: root.fontFamily
            fontSize: Style.font.caption
            tooltipText: "Enter in the hex field does the same"
            onClicked: root.applyRequested()
          }
        }
      }

      Column {
        width: parent.width
        spacing: Style.space(6)

        Text {
          text: "PRESETS"
          color: root.dim
          font.family: root.fontFamily
          font.pixelSize: Style.font.caption
          font.bold: true
          font.letterSpacing: 1.2
        }

        Flow {
          width: parent.width
          spacing: Style.space(6)

          Repeater {
            model: Model.COLOR_PRESETS
            Chip { required property var modelData; chipHex: modelData.hex; label: modelData.label }
          }
        }
      }

      Column {
        width: parent.width
        spacing: Style.space(6)

        Item {
          width: parent.width
          height: themeLabel.implicitHeight

          Text {
            id: themeLabel
            text: "THEME"
            color: root.dim
            font.family: root.fontFamily
            font.pixelSize: Style.font.caption
            font.bold: true
            font.letterSpacing: 1.2
          }
        }

        Flow {
          width: parent.width
          spacing: Style.space(6)

          Button {
            iconText: "󰏘"
            text: "Paint all with theme"
            bordered: true
            enabled: root.themeEnabled
            foreground: root.fg
            accent: root.accent
            fontFamily: root.fontFamily
            fontSize: Style.font.caption
            horizontalPadding: Style.space(8)
            verticalPadding: Style.space(2)
            tooltipText: "Paint every key and region with the theme accent now"
            onClicked: root.themeColorRequested()
          }

          Repeater {
            model: root.themeChips
            Chip { required property var modelData; chipHex: modelData.hex; label: modelData.label }
          }
        }
      }
    }
  }
}
