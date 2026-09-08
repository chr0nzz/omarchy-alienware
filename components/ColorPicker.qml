import QtQuick
import QtQuick.Shapes
import qs.Commons
import qs.Ui
import "../Model.js" as Model

Column {
  id: root

  property string hex: ""
  property var swatches: []
  property bool themeEnabled: true
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property color accent: Color.accent
  property string fontFamily: Style.font.family

  signal picked(string hex)
  signal themeColorRequested()

  spacing: Style.space(8)

  property real hue: 0
  property real sat: 0
  property real val: 0
  property bool syncingFromHex: false

  readonly property bool showSwatches: root.swatches.length > 1 && !Model.paletteIsFlat(root.swatches)
  readonly property color previewColor: Model.hexPreview(Model.hsvToHex(root.hue, root.sat, root.val))

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
    picked(Model.hsvToHex(hue, sat, val))
  }

  onHexChanged: syncFromHex(hex)
  onHueChanged: commitFromHsv()
  onSatChanged: commitFromHsv()
  onValChanged: commitFromHsv()
  Component.onCompleted: syncFromHex(hex)

  Flow {
    width: parent.width
    spacing: Style.space(6)

    Button {
      text: "Theme"
      iconText: "󰏘"
      bordered: true
      enabled: root.themeEnabled
      foreground: root.fg
      accent: root.accent
      fontFamily: root.fontFamily
      fontSize: Style.font.caption
      horizontalPadding: Style.space(8)
      verticalPadding: Style.space(4)
      tooltipText: "Paint every region with the theme colour now"
      onClicked: root.themeColorRequested()
    }

    Repeater {
      model: root.showSwatches ? root.swatches : []

      Button {
        required property var modelData
        background: "#" + modelData.hex
        bordered: true
        foreground: root.fg
        accent: root.accent
        fontFamily: root.fontFamily
        tooltipText: modelData.label
        onClicked: root.picked(modelData.hex)
      }
    }
  }

  Row {
    id: pickerRow
    width: parent.width
    spacing: Style.space(10)

    Item {
      id: wheel
      width: Style.space(150)
      height: width

      readonly property real radius: width / 2
      readonly property real centerX: width / 2
      readonly property real centerY: height / 2

      function applyPoint(px, py) {
        var hs = Model.pointToHueSat((px - centerX) / radius, (py - centerY) / radius)
        root.hue = hs.h
        root.sat = hs.s
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
        color: "transparent"
        border.color: Util.alpha(root.fg, 0.35)
        border.width: Style.normalBorderWidth
      }

      Rectangle {
        id: wheelHandle
        readonly property var point: Model.hueSatToPoint(root.hue, root.sat)
        width: Style.space(14)
        height: width
        radius: width / 2
        color: "transparent"
        border.color: root.fg
        border.width: Style.space(2)
        x: wheel.centerX + point.x * wheel.radius - width / 2
        y: wheel.centerY + point.y * wheel.radius - height / 2
      }

      MouseArea {
        anchors.fill: parent
        preventStealing: true
        cursorShape: Qt.PointingHandCursor
        onPressed: function(mouse) { wheel.applyPoint(mouse.x, mouse.y) }
        onPositionChanged: function(mouse) { if (pressed) wheel.applyPoint(mouse.x, mouse.y) }
      }
    }

    Column {
      id: sideColumn
      spacing: Style.space(8)

      Column {
        id: previewBlock
        spacing: Style.space(4)

        Rectangle {
          id: preview
          width: Style.space(28)
          height: Style.space(28)
          radius: Style.cornerRadius
          color: root.previewColor
          border.color: Util.alpha(root.fg, 0.35)
          border.width: Style.normalBorderWidth
        }

        Text {
          width: preview.width
          horizontalAlignment: Text.AlignHCenter
          textFormat: Text.PlainText
          text: "#" + Model.hsvToHex(root.hue, root.sat, root.val)
          color: root.dim
          font.family: root.fontFamily
          font.pixelSize: Style.font.caption
        }
      }

      Item {
        id: valueBar
        width: Style.space(20)
        height: wheel.height - previewBlock.height - sideColumn.spacing

        readonly property real trackHeight: height

        function applyPoint(py) {
          var frac = 1 - Math.max(0, Math.min(1, py / Math.max(1, trackHeight)))
          root.val = frac * 100
        }

        Rectangle {
          anchors.fill: parent
          border.color: Util.alpha(root.fg, 0.35)
          border.width: Style.normalBorderWidth
          gradient: Gradient {
            orientation: Gradient.Vertical
            GradientStop { position: 0.0; color: Model.hexPreview(Model.hsvToHex(root.hue, root.sat, 100)) }
            GradientStop { position: 1.0; color: "#000000" }
          }
        }

        Rectangle {
          id: valueHandle
          width: parent.width + Style.space(6)
          height: Style.space(4)
          x: -Style.space(3)
          y: Math.max(0, Math.min(valueBar.trackHeight - height, (1 - root.val / 100) * valueBar.trackHeight - height / 2))
          color: root.fg
          border.color: Color.background
          border.width: Style.normalBorderWidth
        }

        MouseArea {
          anchors.fill: parent
          preventStealing: true
          cursorShape: Qt.SizeVerCursor
          onPressed: function(mouse) { valueBar.applyPoint(mouse.y) }
          onPositionChanged: function(mouse) { if (pressed) valueBar.applyPoint(mouse.y) }
        }
      }
    }
  }

  Text {
    width: parent.width
    wrapMode: Text.Wrap
    textFormat: Text.PlainText
    text: "Drag the wheel and bar to choose a colour. Set applies it to the selected regions."
    color: root.dim
    font.family: root.fontFamily
    font.pixelSize: Style.font.caption
  }
}
