use bitcoin::{Amount, OutPoint, Txid, TxIn, TxOut, Transaction, ScriptBuf, Witness, Sequence, Address};
use bitcoin::absolute::LockTime;
use bitcoin::transaction::Version;
use bitcoin::hashes::Hash;
use std::str::FromStr;

#[derive(Clone, Debug)]
pub struct DummyTx {
    pub tx: Vec<u8>,
    pub txid: String,
}

pub fn create_dummy_tx(address: &str, amount_sats: u64) -> Result<DummyTx, String> {
    // Create a fake input that spends the all-zero txid.
    let input = TxIn {
        previous_output: OutPoint {
            txid: Txid::from_slice(&[0; 32]).map_err(|e| format!("txid err: {e}"))?,
            vout: 0,
        },
        script_sig: ScriptBuf::new(),
        sequence: Sequence::from_height(0),
        witness: Witness::new(),
    };

    let dest_address = Address::from_str(address)
        .map_err(|e| format!("invalid address: {e}"))?
        .assume_checked();

    let output = TxOut {
        value: Amount::from_sat(amount_sats),
        script_pubkey: dest_address.script_pubkey(),
    };

    let tx = Transaction {
        version: Version::TWO,
        lock_time: LockTime::ZERO,
        input: vec![input],
        output: vec![output],
    };

    Ok(DummyTx {
        tx: bitcoin::consensus::serialize(&tx),
        txid: tx.compute_txid().to_string(),
    })
} 