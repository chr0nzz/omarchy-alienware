const test = require("node:test")
const assert = require("node:assert/strict")
const { RgbSubTab } = require("./load.js")

test("normalizeRgbSubTab defaults to keyboard when nothing was saved", () => {
  assert.equal(RgbSubTab.normalizeRgbSubTab(undefined, true), "keyboard")
  assert.equal(RgbSubTab.normalizeRgbSubTab("", true), "keyboard")
  assert.equal(RgbSubTab.normalizeRgbSubTab(null, true), "keyboard")
})

test("normalizeRgbSubTab keeps a saved chassis choice", () => {
  assert.equal(RgbSubTab.normalizeRgbSubTab("chassis", true), "chassis")
  assert.equal(RgbSubTab.normalizeRgbSubTab("  chassis  ", true), "chassis")
  assert.equal(RgbSubTab.normalizeRgbSubTab("CHASSIS", true), "chassis")
})

test("normalizeRgbSubTab keeps a saved keyboard choice", () => {
  assert.equal(RgbSubTab.normalizeRgbSubTab("keyboard", true), "keyboard")
})

test("normalizeRgbSubTab falls back to keyboard for unrecognised saved text", () => {
  assert.equal(RgbSubTab.normalizeRgbSubTab("garbage", true), "keyboard")
  assert.equal(RgbSubTab.normalizeRgbSubTab("0", true), "keyboard")
})

test("normalizeRgbSubTab forces chassis when the keyboard is not present, regardless of the saved value", () => {
  assert.equal(RgbSubTab.normalizeRgbSubTab("keyboard", false), "chassis")
  assert.equal(RgbSubTab.normalizeRgbSubTab("chassis", false), "chassis")
  assert.equal(RgbSubTab.normalizeRgbSubTab(undefined, false), "chassis")
})

test("SUB_TABS lists both surfaces with keyboard first", () => {
  assert.deepEqual(RgbSubTab.SUB_TABS, ["keyboard", "chassis"])
})

test("DEFAULT_SUB_TAB is keyboard", () => {
  assert.equal(RgbSubTab.DEFAULT_SUB_TAB, "keyboard")
})
