const binding = require('./binding')

exports.hello = binding.hello
console.log('binding.hello()', binding.hello())

exports.createDummyTx = binding.createDummyTx
console.log(
  'binding.createDummyTx()',
  binding.createDummyTx('bcrt1qnuyejmm2l4kavspq0jqaw0fv07lg6zv3z9z3te', 50000)
)
