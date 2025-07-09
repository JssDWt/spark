use bare_rust::{
    ffi::{js_env_t, js_value_t},
    Env, Function, Object, String, Uint8Array,
};
use spark_frost::tx::{create_dummy_tx};

fn js_error(env: &Env, msg: &str) -> js_value_t {
    env.throw_error(msg).unwrap_or(std::ptr::null_mut())
}

#[unsafe(no_mangle)]
pub extern "C" fn bare_addon_exports(
    env: *mut js_env_t,
    exports: *mut js_value_t,
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
    let create_fn = Function::new(&env, |env, info| {
        let argv = info.argv();
        if argv.len() < 2 {
            return js_error(env, "Expected 2 arguments (address, amountSats)");
        }

        let address = match argv.get_string(0) {
            Ok(s) => s.to_string(),
            Err(_) => return js_error(env, "address must be a string"),
        };

        // amount can be bigint or number → use get_u64 fallback
        let amount = match argv.get_bigint(1).or_else(|_| argv.get_u64(1)) {
            Ok(v) => v as u64,
            Err(_) => return js_error(env, "amountSats must be a number/bigint"),
        };

        match create_dummy_tx(&address, amount) {
            Ok(dummy) => {
                let obj = Object::new(env).unwrap();
                let tx_arr = Uint8Array::from(env, &dummy.tx).unwrap();
                obj.set_named_property("tx", tx_arr).unwrap();
                obj.set_named_property("txid", String::new(env, &dummy.txid).unwrap()).unwrap();
                obj.into()
            }
            Err(e) => js_error(env, &e),
        }
    }).unwrap();

    exports
        .set_named_property("createDummyTx", create_fn)
        .unwrap();

    exports.into()
}
