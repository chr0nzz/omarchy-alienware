.pragma library

var KEYBOARD = "keyboard"
var CHASSIS = "chassis"
var SUB_TABS = [KEYBOARD, CHASSIS]
var DEFAULT_SUB_TAB = KEYBOARD

function normalizeRgbSubTab(raw, kbdPresent) {
  if (kbdPresent === false) return CHASSIS
  var value = String(raw === undefined || raw === null ? "" : raw).trim().toLowerCase()
  return value === CHASSIS ? CHASSIS : DEFAULT_SUB_TAB
}
