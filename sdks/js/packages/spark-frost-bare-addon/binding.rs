use bare_rust::{
    ffi::{js_env_t, js_value_t},
    Env, Function, Object, String, Uint8Array, TypedArray
};
use spark_frost::tx::{create_dummy_tx};
use std::ptr;

fn js_error(_env: &Env, _msg: &str) -> *mut js_value_t {
    ptr::null_mut()
}

#[unsafe(no_mangle)]
pub extern "C" fn bare_addon_exports(
    env: *mut js_env_t,
    _exports: *mut js_value_t,
) -> *mut js_value_t {
    let env = Env::from(env);

    let mut exports = Object::new(&env).unwrap();

    let function = Function::new(&env, |env, _info| {
        String::new(&env, "Hello from Rust").unwrap()
    });

    exports
        .set_named_property("hello", function.unwrap())
        .unwrap();

    // createDummyTx(address: string, amountSats: bigint | number)
    let create_dummy_tx_fn = Function::new(&env, |env, info| {
        let js_addr: bare_rust::String = match info.arg(0) {
            Some(v) => v,
            None => return js_error(env, "address argument missing or not a string"),
        };

        let mut address_bytes = match js_addr.to_bytes() {
            Ok(b) => b,
            Err(_) => return js_error(env, "failed to read string bytes"),
        };
        // Remove the trailing null byte if it exists
        if address_bytes.last() == Some(&0) {
            address_bytes.pop();
        }

        let address = match std::str::from_utf8(&address_bytes) {
            Ok(s) => s,
            Err(_) => return js_error(env, "address not valid UTF-8"),
        };

        println!("got address: {:?}", address);

        // let num: Number = match info.arg(1) {
        //     Some(v) => v,
        //     None => return js_error(env, "amountSats argument missing or not a number"),
        // };

        // let amount = f64::from(num) as u64;
        let amount = 50000;

        match create_dummy_tx(&address, amount) {
            Ok(dummy) => {
                let mut obj = Object::new(env).unwrap();
                let tx_arr = Uint8Array::new(env, dummy.tx.len()).unwrap();
                tx_arr.as_mut_slice().copy_from_slice(&dummy.tx);
                obj.set_named_property("tx", tx_arr).unwrap();
                obj.set_named_property("txid", String::new(env, &dummy.txid).unwrap()).unwrap();
                obj.into()
            }
            Err(e) => {
                println!("Error creating dummy tx: {}", e);
                js_error(env, &format!("failed to create dummy tx: {}", e))
            },
        }
    }).unwrap();

    exports
        .set_named_property("createDummyTx", create_dummy_tx_fn)
        .unwrap();

    exports.into()
}
