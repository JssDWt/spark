const binding = require('./binding')
const secp256k1 = require('@noble/secp256k1')

exports.hello = binding.hello
exports.createDummyTx = binding.createDummyTx
exports.encryptEcies = binding.encryptEcies

function log(msg, ...args) {
  console.log(`index.js: ${msg}`, ...args)
}

log('binding.hello()', binding.hello())

console.log('\n')

log(
  `binding.createDummyTx("bcrt1qnuyejmm2l4kavspq0jqaw0fv07lg6zv3z9z3te", 10000n):`
)
const successResult = binding.createDummyTx(
  'bcrt1qnuyejmm2l4kavspq0jqaw0fv07lg6zv3z9z3te',
  10000n
)
log('dummyTx:', successResult)

console.log('\n')

try {
  log(`binding.createDummyTx("this_address_will_error", 10000n):`)
  const errResult = binding.createDummyTx('this_address_will_error', 10000n)
} catch (e) {
  log(e)
}

console.log('\n')

try {
  log(`binding.createDummyTx("bcrt1qnuyejmm2l4kavspq0jqaw0fv07lg6zv3z9z3te"):`)
  const errResult = binding.createDummyTx(
    'bcrt1qnuyejmm2l4kavspq0jqaw0fv07lg6zv3z9z3te'
  )
} catch (e) {
  log(e)
}

console.log('\n')

const plaintext = new Uint8Array([1, 2, 3, 4])
// Dummy public key (65-byte uncompressed secp256k1 full of 0x02 values just for example)
const pk = new Uint8Array(33).fill(2)
try {
  log('binding.encryptEcies(plaintext, pk):')
  const cipher = binding.encryptEcies(plaintext, pk)
  log('ciphertext length', cipher.length)
} catch (e) {
  log(e)
}

console.log('\n')

const privKey = new Uint8Array(32).fill(1)
const pubKey = secp256k1.getPublicKey(privKey, true)
const pt2 = new Uint8Array([10, 11, 12])
try {
  log('binding.encryptEcies(pt2, pubKey):')
  const ct2 = binding.encryptEcies(pt2, pubKey)
  log('binding.decryptEcies(ct2, privKey):')
  const dec2 = binding.decryptEcies(ct2, privKey)
  log('decrypt success:', Buffer.from(dec2).equals(Buffer.from(pt2)))
} catch (e) {
  log(e)
}

console.log('\n')

log('signFrost smoke test (expect error)')
try {
  binding.signFrost(
    new Uint8Array([1, 2, 3]),
    {
      secretKey: new Uint8Array([0]),
      publicKey: new Uint8Array([0]),
      verifyingKey: new Uint8Array([0])
    },
    { hiding: new Uint8Array([0]), binding: new Uint8Array([0]) },
    { hiding: new Uint8Array([0]), binding: new Uint8Array([0]) },
    {},
    null
  )
} catch (e) {
  log('signFrost returned error as expected', e.message || e)
}
