const test = require("node:test")
const assert = require("node:assert")
const fs = require("node:fs")
const path = require("node:path")
const { Model } = require("./load.js")

function parseKeymap() {
  const text = fs.readFileSync(path.join(__dirname, "..", "docs", "KEYMAP.txt"), "utf8")
  const out = {}
  for (const line of text.split("\n")) {
    const m = line.match(/^\s*(\d+)\s\s+([a-z0-9]+)(,\s*with\s+(\d+)\s+as its second led)?\s*$/)
    if (!m) continue
    out[m[2]] = m[4] === undefined ? [Number(m[1])] : [Number(m[1]), Number(m[4])]
  }
  return out
}

test("KEYMAP.txt lists every paintable key exactly once", () => {
  const map = parseKeymap()
  const ids = Model.keyboardPaintableIds()
  assert.equal(Object.keys(map).length, ids.length)
  for (const id of ids) assert.ok(map[id], id + " is missing from KEYMAP.txt")
})

test("KEYMAP.txt and Model.js agree on every index", () => {
  const map = parseKeymap()
  for (const id of Model.keyboardPaintableIds()) {
    assert.deepEqual(Model.keyboardKeyById(id).indices, map[id], id)
  }
})

test("no two keys claim the same led", () => {
  const seen = new Map()
  for (const id of Model.keyboardPaintableIds()) {
    for (const idx of Model.keyboardKeyById(id).indices) {
      assert.ok(!seen.has(idx), "index " + idx + " claimed by both " + seen.get(idx) + " and " + id)
      seen.set(idx, id)
    }
  }
  assert.equal(seen.size, 88)
})
