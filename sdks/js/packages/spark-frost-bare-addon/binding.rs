use bare_rust::{
    ffi::{js_env_t, js_value_t},
    Env, Function, Object, String, Uint8Array, TypedArray, Value, BigInt
};
use spark_frost::bridge::{create_dummy_tx};

use frost_secp256k1_tr::Identifier;
use hex;
use std::collections::HashMap;

macro_rules! log_binding {
    ($($arg:tt)*) => {
        println!("binding.rs: {}", format!($($arg)*));
    };
}

fn js_error(env: &Env, msg: &str) -> Value {
    String::new(env, msg).unwrap().into()
}

fn js_err(env: &Env, msg: &str) -> Value { js_error(env, msg) }

// Helper to convert a JS Uint8Array property to Vec<u8>
fn get_uint8_vec(env: &Env, obj: &Object, name: &str) -> Result<Vec<u8>, Value> {
    let arr: Uint8Array = obj
        .get_named_property(name)
        .map_err(|_| js_err(env, &format!("missing field {name}")))?;
    Ok(arr.as_slice().to_vec())
}

fn js_commitment_to_proto(env: &Env, obj: &Object) -> Result<spark_frost::proto::common::SigningCommitment, Value> {
    let hiding = get_uint8_vec(env, obj, "hiding")?;
    let binding = get_uint8_vec(env, obj, "binding")?;
    Ok(spark_frost::proto::common::SigningCommitment { hiding, binding })
}

#[unsafe(no_mangle)]
pub extern "C" fn bare_addon_exports(
    env: *mut js_env_t,
    _exports: *mut js_value_t,
) -> *mut js_value_t {
    let env = Env::from(env);

    let mut exports = Object::new(&env).unwrap();

    let function = Function::new(&env, |env, _| {
        Ok(String::new(env, "Hello from Rust")?.into())
    }).unwrap();

    exports
        .set_named_property("hello", function)
        .unwrap();

    // createDummyTx(address: string, amountSats: bigint | number)
    let create_dummy_tx_fn = Function::new(&env, |env, info| -> Result<Value, Value> {
        let js_addr: String = info.arg(0).ok_or(js_err(env, "address argument missing or not a string"))?;

        let address: std::string::String = js_addr.into();
        log_binding!("got address: {:?}", address);

        let bigint: BigInt = info.arg(1).ok_or(js_err(env, "amountSats argument missing or not a bigint"))?;
        let amount = u64::from(bigint);
        log_binding!("got amount sats: {}", amount);

        match create_dummy_tx(&address, amount) {
            Ok(dummy) => {
                let mut obj = Object::new(env)?;
                let tx_arr = Uint8Array::new(env, dummy.tx.len())?;
                tx_arr.as_mut_slice().copy_from_slice(&dummy.tx);
                obj.set_named_property("tx", tx_arr)?;
                obj.set_named_property("txid", String::new(env, &dummy.txid)? )?;
                Ok(obj.into())
            }
            Err(e) => {
                log_binding!("error creating dummy tx: {}", e);
                Err(js_err(env, &format!("failed to create dummy tx: {}", e)))
            },
        }
    }).unwrap();

    exports
        .set_named_property("createDummyTx", create_dummy_tx_fn)
        .unwrap();

    // encryptEcies({ msg: Uint8Array, publicKey: Uint8Array }) -> Uint8Array
    let encrypt_ecies_fn = Function::new(&env, |env, info| -> Result<Value, Value> {
        let msg_arr: Uint8Array = info.arg(0).ok_or(js_err(env, "msg argument missing or not a Uint8Array"))?;
        let pk_arr: Uint8Array = info.arg(1).ok_or(js_err(env, "publicKey argument missing or not a Uint8Array"))?;

        let ciphertext = match spark_frost::bridge::encrypt_ecies(msg_arr.as_slice(), pk_arr.as_slice()) {
            Ok(c) => c,
            Err(e) => return Err(js_err(env, &format!("encrypt error: {}", e))),
        };

        let js_cipher = Uint8Array::new(env, ciphertext.len())?;
        js_cipher.as_mut_slice().copy_from_slice(&ciphertext);
        Ok(js_cipher.into())
    }).unwrap();

    exports.set_named_property("encryptEcies", encrypt_ecies_fn).unwrap();

    // decryptEcies({ ciphertext: Uint8Array, secretKey: Uint8Array }) -> Uint8Array
    let decrypt_ecies_fn = Function::new(&env, |env, info| -> Result<Value, Value> {
        let ct_arr: Uint8Array = info.arg(0).ok_or(js_err(env, "ciphertext argument missing or not a Uint8Array"))?;
        let sk_arr: Uint8Array = info.arg(1).ok_or(js_err(env, "secretKey argument missing or not a Uint8Array"))?;

        let plaintext = match spark_frost::bridge::decrypt_ecies(ct_arr.as_slice().to_vec(), sk_arr.as_slice().to_vec()) {
            Ok(p) => p,
            Err(e) => return Err(js_err(env, &format!("decrypt error: {}", e))),
        };

        let js_plaintext = Uint8Array::new(env, plaintext.len())?;
        js_plaintext.as_mut_slice().copy_from_slice(&plaintext);
        Ok(js_plaintext.into())
    }).unwrap();

    exports.set_named_property("decryptEcies", decrypt_ecies_fn).unwrap();

    // signFrost(msg, keyPackage, nonce, selfCommitment, commitments, adaptorPubKey?)
    let sign_frost_fn = Function::new(&env, |env, info| -> Result<Value, Value> {
        // msg
        let msg_arr: Uint8Array = info.arg(0).ok_or(js_err(env, "msg argument missing"))?;

        // keyPackage
        let kp_obj: Object = info.arg(1).ok_or(js_err(env, "keyPackage argument missing"))?;
        let secret_key = get_uint8_vec(env, &kp_obj, "secretKey")?;
        let public_key = get_uint8_vec(env, &kp_obj, "publicKey")?;
        let verifying_key = get_uint8_vec(env, &kp_obj, "verifyingKey")?;

        // Build proto KeyPackage
        let identifier = Identifier::derive(b"user").map_err(|e| js_err(env, &e.to_string()))?;
        let identifier_string = hex::encode(identifier.to_scalar().to_bytes());
        let kp_proto = spark_frost::proto::frost::KeyPackage {
            identifier: identifier_string.clone(),
            secret_share: secret_key.clone(),
            public_shares: HashMap::from([(identifier_string.clone(), public_key.clone())]),
            public_key: verifying_key.clone(),
            min_signers: 1,
        };

        // nonce
        let nonce_obj: Object = info.arg(2).ok_or(js_err(env, "nonce argument missing"))?;
        let nonce_hiding = get_uint8_vec(env, &nonce_obj, "hiding")?;
        let nonce_binding = get_uint8_vec(env, &nonce_obj, "binding")?;
        let nonce_proto = spark_frost::proto::frost::SigningNonce { hiding: nonce_hiding, binding: nonce_binding };

        // self commitment
        let self_commit_obj: Object = info.arg(3).ok_or(js_err(env, "selfCommitment argument missing"))?;
        let self_commit_proto = js_commitment_to_proto(env, &self_commit_obj)?;

        // commitments map arg4 (currently expect empty or ignored)
        let commitments_proto: HashMap<std::string::String, spark_frost::proto::common::SigningCommitment> = HashMap::new();

        // adaptor public key (optional)
        let adaptor_pk: Option<Vec<u8>> = info.arg(5).map(|a: Uint8Array| a.as_slice().to_vec());

        // Call bridge
        match spark_frost::bridge::sign_frost(
            msg_arr.as_slice().to_vec(),
            kp_proto,
            nonce_proto,
            self_commit_proto,
            commitments_proto,
            adaptor_pk,
        ) {
            Ok(sig) => {
                let js_sig = Uint8Array::new(env, sig.len())?;
                js_sig.as_mut_slice().copy_from_slice(&sig);
                Ok(js_sig.into())
            }
            Err(e) => Err(js_err(env, &e)),
        }
    }).unwrap();

    exports.set_named_property("signFrost", sign_frost_fn).unwrap();

    exports.into()
}
