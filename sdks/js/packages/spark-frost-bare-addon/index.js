const binding = require('./binding')

exports.hello = binding.hello
console.log('binding.hello()', binding.hello())

exports.createDummyTx = binding.createDummyTx
console.log('binding.createDummyTx()', binding.createDummyTx())
