const binding = require('./binding')

exports.hello = binding.hello
console.log('binding.hello', binding.hello())
