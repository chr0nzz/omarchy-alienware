const fs = require("node:fs")
const path = require("node:path")

const rootDir = path.join(__dirname, "..")

function source(name) {
  return fs.readFileSync(path.join(rootDir, name), "utf8")
    .split("\n")
    .filter(function(line) { return !/^\.(pragma|import)\b/.test(line) })
    .join("\n")
}

function load(name, globals) {
  const src = source(name)
  const names = Array.from(src.matchAll(/^(?:function|var) (\w+)/gm), function(m) { return m[1] })
  const keys = Object.keys(globals)
  const factory = new Function(...keys, src + "\nreturn { " + names.join(", ") + " }")
  return factory(...keys.map(function(k) { return globals[k] }))
}

const Model = load("Model.js", {})

module.exports = { Model }
