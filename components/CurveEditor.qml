import QtQuick
import qs.Commons
import qs.Ui
import "../Model.js" as Model

Item {
  id: editor

  property var points: []
  property int floorPercent: 0
  property var currentTemp: null
  property int selectedIndex: 0
  property bool editable: true
  property color fg: Color.foreground
  property color dim: Qt.darker(fg, 1.5)
  property color accent: Color.accent
  property color alertColor: Color.urgent
  property string fontFamily: Style.font.family

  readonly property var list: Model.toList(points)
  readonly property int plotHeight: Style.space(140)
  readonly property int handleSize: Style.space(10)

  signal edited(var next)
  signal pointPicked(int index)

  implicitHeight: column.implicitHeight

  function plotX(temp) {
    return plot.width * Math.max(0, Math.min(1, temp / 110))
  }

  function plotY(percent) {
    return plot.height - plot.height * Math.max(0, Math.min(1, percent / 100))
  }

  function tempAt(x) {
    return Math.round(Math.max(0, Math.min(1, x / Math.max(1, plot.width))) * 110)
  }

  function boostAt(y) {
    var pct = 1 - Math.max(0, Math.min(1, y / Math.max(1, plot.height)))
    return Model.percentToBoost(Math.round(pct * 100) - floorPercent)
  }

  function nearestIndex(x, y) {
    var best = -1
    var bestDist = Number.MAX_VALUE
    for (var i = 0; i < list.length; i++) {
      var dx = plotX(list[i].temp) - x
      var dy = plotY(Model.boostPercent(list[i].boost) + floorPercent) - y
      var d = dx * dx + dy * dy
      if (d < bestDist) { bestDist = d; best = i }
    }
    return { index: best, distance: Math.sqrt(bestDist) }
  }

  function selectPoint(index) {
    if (index < 0 || index >= list.length) return
    selectedIndex = index
    pointPicked(index)
  }

  function applyDrag(index, x, y) {
    if (!editable) return
    edited(Model.setPoint(list, index, tempAt(x), boostAt(y)))
  }

  function addAt(x, y) {
    if (!editable) return
    var next = Model.insertPoint(list, tempAt(x), boostAt(y))
    edited(next)
  }

  function removeAt(index) {
    if (!editable) return
    edited(Model.removePoint(list, index))
  }

  onListChanged: canvas.requestPaint()
  onFloorPercentChanged: canvas.requestPaint()
  onCurrentTempChanged: canvas.requestPaint()
  onSelectedIndexChanged: canvas.requestPaint()

  Column {
    id: column
    anchors.left: parent.left
    anchors.right: parent.right
    spacing: Style.space(6)

    Item {
      id: plot
      width: parent.width
      height: editor.plotHeight

      Canvas {
        id: canvas
        anchors.fill: parent
        renderStrategy: Canvas.Immediate

        onPaint: {
          var ctx = getContext("2d")
          ctx.reset()
          ctx.clearRect(0, 0, width, height)

          ctx.strokeStyle = Util.alpha(editor.fg, 0.12)
          ctx.lineWidth = 1
          for (var g = 1; g < 4; g++) {
            var gy = height * g / 4
            ctx.beginPath()
            ctx.moveTo(0, gy)
            ctx.lineTo(width, gy)
            ctx.stroke()
          }
          for (var t = 20; t < 110; t += 20) {
            var gx = editor.plotX(t)
            ctx.beginPath()
            ctx.moveTo(gx, 0)
            ctx.lineTo(gx, height)
            ctx.stroke()
          }

          var floorY = editor.plotY(editor.floorPercent)
          ctx.fillStyle = Util.alpha(editor.fg, 0.14)
          ctx.fillRect(0, floorY, width, height - floorY)
          ctx.strokeStyle = Util.alpha(editor.fg, 0.45)
          ctx.setLineDash([4, 4])
          ctx.beginPath()
          ctx.moveTo(0, floorY)
          ctx.lineTo(width, floorY)
          ctx.stroke()
          ctx.setLineDash([])

          var overlay = Model.curveOverlay(editor.list, editor.floorPercent)
          if (overlay.length) {
            ctx.beginPath()
            for (var i = 0; i < overlay.length; i++) {
              var px = editor.plotX(overlay[i].temp)
              var py = editor.plotY(overlay[i].total)
              if (i === 0) ctx.moveTo(px, py)
              else ctx.lineTo(px, py)
            }
            ctx.strokeStyle = editor.accent
            ctx.lineWidth = 2
            ctx.stroke()
          }

          if (typeof editor.currentTemp === "number") {
            var cx = editor.plotX(editor.currentTemp)
            ctx.strokeStyle = Util.alpha(editor.alertColor, 0.7)
            ctx.lineWidth = 1
            ctx.beginPath()
            ctx.moveTo(cx, 0)
            ctx.lineTo(cx, height)
            ctx.stroke()
          }
        }
      }

      Repeater {
        model: editor.list

        Rectangle {
          id: handle
          required property var modelData
          required property int index
          readonly property bool current: editor.selectedIndex === index
          width: editor.handleSize
          height: editor.handleSize
          radius: width / 2
          x: editor.plotX(handle.modelData.temp) - width / 2
          y: editor.plotY(Model.boostPercent(handle.modelData.boost) + editor.floorPercent) - height / 2
          color: handle.current ? editor.accent : editor.fg
          border.width: handle.current ? 2 : 0
          border.color: editor.fg
          opacity: editor.editable ? 1 : 0.5
        }
      }

      MouseArea {
        id: drag
        anchors.fill: parent
        hoverEnabled: true
        acceptedButtons: Qt.LeftButton | Qt.RightButton
        cursorShape: Qt.CrossCursor
        property int grabbed: -1

        onPressed: function(mouse) {
          var hit = editor.nearestIndex(mouse.x, mouse.y)
          if (mouse.button === Qt.RightButton) {
            if (hit.index >= 0 && hit.distance <= editor.handleSize * 2) editor.removeAt(hit.index)
            return
          }
          if (hit.index >= 0 && hit.distance <= editor.handleSize * 2) {
            grabbed = hit.index
            editor.selectPoint(hit.index)
          } else {
            grabbed = -1
          }
        }

        onPositionChanged: function(mouse) {
          if (grabbed < 0 || !pressed) return
          editor.applyDrag(grabbed, mouse.x, mouse.y)
        }

        onReleased: grabbed = -1

        onDoubleClicked: function(mouse) {
          var hit = editor.nearestIndex(mouse.x, mouse.y)
          if (hit.index >= 0 && hit.distance <= editor.handleSize * 2) return
          editor.addAt(mouse.x, mouse.y)
        }
      }
    }

    Item {
      width: parent.width
      implicitHeight: axisLeft.implicitHeight

      Text {
        id: axisLeft
        textFormat: Text.PlainText
        anchors.left: parent.left
        text: "0°C"
        color: editor.dim
        font.family: editor.fontFamily
        font.pixelSize: Style.font.caption
      }

      Text {
        textFormat: Text.PlainText
        anchors.horizontalCenter: parent.horizontalCenter
        text: "55°C"
        color: editor.dim
        font.family: editor.fontFamily
        font.pixelSize: Style.font.caption
      }

      Text {
        textFormat: Text.PlainText
        anchors.right: parent.right
        text: "110°C"
        color: editor.dim
        font.family: editor.fontFamily
        font.pixelSize: Style.font.caption
      }
    }

    Column {
      width: parent.width
      spacing: Style.space(2)

      Repeater {
        model: editor.list

        Item {
          id: pointRow
          required property var modelData
          required property int index
          readonly property bool current: editor.selectedIndex === index
          width: parent.width
          implicitHeight: pointText.implicitHeight + Style.space(4)

          Rectangle {
            anchors.fill: parent
            radius: Style.space(4)
            visible: pointRow.current
            color: Style.selectedFillFor(editor.fg, editor.accent)
          }

          Text {
            id: pointText
            textFormat: Text.PlainText
            anchors.left: parent.left
            anchors.leftMargin: Style.space(6)
            anchors.verticalCenter: parent.verticalCenter
            text: (pointRow.index + 1) + ".  " + pointRow.modelData.temp + "°C"
            color: pointRow.current ? editor.accent : editor.fg
            font.family: editor.fontFamily
            font.pixelSize: Style.font.caption
          }

          Text {
            textFormat: Text.PlainText
            anchors.right: parent.right
            anchors.rightMargin: Style.space(6)
            anchors.verticalCenter: parent.verticalCenter
            text: "+" + Model.boostPercent(pointRow.modelData.boost) + "%  (" + pointRow.modelData.boost + ")"
            color: pointRow.current ? editor.accent : editor.dim
            font.family: editor.fontFamily
            font.pixelSize: Style.font.caption
          }

          MouseArea {
            anchors.fill: parent
            cursorShape: Qt.PointingHandCursor
            onClicked: editor.selectPoint(pointRow.index)
          }
        }
      }
    }

    Text {
      textFormat: Text.PlainText
      width: parent.width
      wrapMode: Text.Wrap
      text: Model.CURVE_NOTE + " The shaded band is what firmware is running right now."
      color: editor.dim
      font.family: editor.fontFamily
      font.pixelSize: Style.font.caption
    }
  }
}
